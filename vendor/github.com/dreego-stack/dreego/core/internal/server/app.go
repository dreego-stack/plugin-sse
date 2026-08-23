package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	mw "github.com/dreego-stack/dreego/core/internal/middleware"
	sess "github.com/dreego-stack/dreego/core/internal/session"
)

var ErrServerRunning = errors.New("dreego: server already running")

type Store = sess.Store

type storeValidator interface {
	Validate() error
}

type App struct {
	mu             sync.RWMutex
	routes         []route
	redirects      []redirectRule
	rewrites       []rewriteRule
	loggingEnabled bool
	csrfEnabled    bool
	errorHandlers  map[int]http.HandlerFunc
	sessionStore   Store
	builtHandler   http.Handler
	middlewares    []func(http.Handler) http.Handler
	customRules    map[string]func(string) string
	ready          atomic.Bool
	cspHeader      string
	built          bool
	buildDone      chan struct{}
	server         *http.Server
	serverConfig   ServerConfig
	shutdownDone   chan error
}

func New() *App {
	a := &App{
		loggingEnabled: true,
		csrfEnabled:    true,
		errorHandlers:  map[int]http.HandlerFunc{},
		customRules:    map[string]func(string) string{},
		cspHeader:      mw.DefaultCSP,
		buildDone:      make(chan struct{}),
	}
	a.ready.Store(true)
	return a
}

func (a *App) SetReady(r bool) {
	a.ready.Store(r)
}

func (a *App) Build() error {
	a.mu.Lock()
	if a.built {
		a.mu.Unlock()
		return nil
	}
	a.built = true
	buildDone := a.buildDone
	redirects := append([]redirectRule(nil), a.redirects...)
	rewrites := append([]rewriteRule(nil), a.rewrites...)
	a.mu.Unlock()

	completed := false
	defer func() {
		if completed {
			return
		}
		a.mu.Lock()
		a.built = false
		close(buildDone)
		a.buildDone = make(chan struct{})
		a.mu.Unlock()
	}()

	if err := validateRedirectCycles(redirects, rewrites); err != nil {
		return err
	}

	if a.csrfEnabled && a.sessionStore == nil {
		a.warnMissingSessionStore()
	}

	if v, ok := a.sessionStore.(storeValidator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("dreego: session store validation failed: %w", err)
		}
	}

	mux := http.NewServeMux()

	if err := a.handleMux(func() {
		mux.HandleFunc("GET /health", a.healthHandler())
		mux.HandleFunc("GET /ready", a.readyHandler())
		for _, r := range a.routes {
			if r.method != "" {
				mux.HandleFunc(r.method+" "+r.pattern, r.handler)
			} else {
				mux.HandleFunc(r.pattern, r.handler)
			}
		}
	}); err != nil {
		return err
	}

	var h http.Handler = mux
	h = a.redirectRewriteMiddleware(h)
	if a.sessionStore != nil && a.csrfEnabled {
		h = mw.CSRF(a.sessionStore)(h)
	}
	if a.sessionStore != nil {
		h = a.sessionMiddleware(h)
	}
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		if a.middlewares[i] == nil {
			continue
		}
		h = a.middlewares[i](h)
	}
	if a.loggingEnabled {
		h = mw.RequestLogging()(h)
	}
	h = mw.RequestID()(h)
	h = mw.Compress()(h)
	h = a.securityHeadersMiddleware(h)
	h = mw.Recovery(a.errorHandlers[500])(h)
	a.mu.Lock()
	a.builtHandler = h
	close(buildDone)
	a.mu.Unlock()
	completed = true
	return nil
}

func (a *App) Handler() http.Handler {
	for {
		if err := a.Build(); err != nil {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			})
		}
		a.mu.RLock()
		h := a.builtHandler
		built := a.built
		buildDone := a.buildDone
		a.mu.RUnlock()
		if h != nil {
			return h
		}
		if !built {
			continue
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-buildDone
			a.Handler().ServeHTTP(w, r)
		})
	}
}

func (a *App) Listen(addr string) error {
	if err := a.Build(); err != nil {
		return err
	}
	cfg := a.serverConfig.withDefaults()
	srv := &http.Server{
		Addr:              addr,
		Handler:           a.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	a.mu.Lock()
	if a.server != nil {
		a.mu.Unlock()
		return ErrServerRunning
	}
	a.server = srv
	shutdownDone := make(chan error, 1)
	a.shutdownDone = shutdownDone
	a.mu.Unlock()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.ListenAndServe() }()

	clearState := func() {
		a.mu.Lock()
		a.server = nil
		a.shutdownDone = nil
		a.mu.Unlock()
	}

	select {
	case <-sigCtx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		err := srv.Shutdown(shutCtx)
		if serr := <-serverErr; serr != nil && serr != http.ErrServerClosed {
			clearState()
			return serr
		}
		clearState()
		return err
	case err := <-serverErr:
		if err == http.ErrServerClosed {
			rerr := <-shutdownDone
			clearState()
			return rerr
		}
		clearState()
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	srv := a.server
	done := a.shutdownDone
	a.mu.Unlock()
	if srv == nil {
		return nil
	}
	err := srv.Shutdown(ctx)
	if done != nil {
		select {
		case done <- err:
		default:
		}
	}
	return err
}
