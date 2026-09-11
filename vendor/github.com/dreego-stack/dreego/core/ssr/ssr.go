// Package ssr owns HTTP listener lifecycle, server limits, signals, and
// graceful shutdown for a Dreego App.
package ssr

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	dreego "github.com/dreego-stack/dreego/core"
	"github.com/dreego-stack/dreego/core/internal/middleware"
	"github.com/dreego-stack/dreego/core/internal/session"
	"github.com/dreego-stack/dreego/core/internal/validate"
)

var ErrServerRunning = errors.New("dreego: server already running")

type ServerConfig struct {
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	MaxHeaderBytes      int
	MaxHeaderValueCount int
	ShutdownTimeout     time.Duration
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadHeaderTimeout:   10 * time.Second,
		ReadTimeout:         30 * time.Second,
		WriteTimeout:        30 * time.Second,
		IdleTimeout:         120 * time.Second,
		MaxHeaderBytes:      1 << 20,
		MaxHeaderValueCount: http.DefaultMaxHeaderValueCount,
		ShutdownTimeout:     10 * time.Second,
	}
}

func (c ServerConfig) withDefaults() ServerConfig {
	defaults := DefaultServerConfig()
	if c.ReadHeaderTimeout <= 0 {
		c.ReadHeaderTimeout = defaults.ReadHeaderTimeout
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = defaults.ReadTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = defaults.WriteTimeout
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = defaults.IdleTimeout
	}
	if c.MaxHeaderBytes <= 0 {
		c.MaxHeaderBytes = defaults.MaxHeaderBytes
	}
	if c.MaxHeaderValueCount <= 0 {
		c.MaxHeaderValueCount = defaults.MaxHeaderValueCount
	}
	if c.ShutdownTimeout <= 0 {
		c.ShutdownTimeout = defaults.ShutdownTimeout
	}
	return c
}

type Host struct {
	mu           sync.Mutex
	app          *dreego.App
	config       ServerConfig
	server       *http.Server
	shutdownDone chan error
	lifecycle    *hostLifecycle
}

type hostLifecycle struct {
	done chan struct{}
	err  error
}

func New(app *dreego.App, config ...ServerConfig) *Host {
	cfg := DefaultServerConfig()
	if len(config) > 0 {
		cfg = config[0].withDefaults()
	}
	return &Host{app: app, config: cfg}
}

func Listen(app *dreego.App, addr string) error {
	return New(app).Listen(addr)
}

func (h *Host) Listen(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return h.Serve(listener)
}

func (h *Host) Serve(listener net.Listener) error {
	lifecycle, err := h.start(listener)
	if err != nil {
		return err
	}
	return h.wait(lifecycle)
}

func (h *Host) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	_, err = h.start(listener)
	return err
}

func (h *Host) start(listener net.Listener) (*hostLifecycle, error) {
	if err := h.app.Build(); err != nil {
		listener.Close()
		return nil, err
	}
	cfg := h.config.withDefaults()
	server := &http.Server{
		Handler:             h.app.Handler(),
		ReadHeaderTimeout:   cfg.ReadHeaderTimeout,
		ReadTimeout:         cfg.ReadTimeout,
		WriteTimeout:        cfg.WriteTimeout,
		IdleTimeout:         cfg.IdleTimeout,
		MaxHeaderBytes:      cfg.MaxHeaderBytes,
		MaxHeaderValueCount: cfg.MaxHeaderValueCount,
	}

	h.mu.Lock()
	if h.server != nil {
		h.mu.Unlock()
		listener.Close()
		return nil, ErrServerRunning
	}
	lifecycle := &hostLifecycle{done: make(chan struct{})}
	h.server = server
	h.shutdownDone = make(chan error, 1)
	h.lifecycle = lifecycle
	signalContext, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	shutdownDone := h.shutdownDone
	h.mu.Unlock()
	go h.run(listener, server, shutdownDone, lifecycle, cfg, signalContext, stopSignals)
	return lifecycle, nil
}

func (h *Host) run(listener net.Listener, server *http.Server, shutdownDone chan error, lifecycle *hostLifecycle, cfg ServerConfig, signalContext context.Context, stopSignals context.CancelFunc) {
	defer stopSignals()
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Serve(listener) }()

	var result error
	select {
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		result = server.Shutdown(shutdownContext)
		cancel()
		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			result = err
		}
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			result = <-shutdownDone
		} else {
			result = err
		}
	}

	h.mu.Lock()
	h.server = nil
	h.shutdownDone = nil
	lifecycle.err = result
	close(lifecycle.done)
	h.mu.Unlock()
}

func (h *Host) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	server := h.server
	shutdownDone := h.shutdownDone
	lifecycle := h.lifecycle
	h.mu.Unlock()
	if server == nil {
		return nil
	}
	err := server.Shutdown(ctx)
	select {
	case shutdownDone <- err:
	default:
	}
	return h.wait(lifecycle)
}

func (h *Host) Wait() error {
	h.mu.Lock()
	lifecycle := h.lifecycle
	h.mu.Unlock()
	if lifecycle == nil {
		return nil
	}
	return h.wait(lifecycle)
}

func (h *Host) wait(lifecycle *hostLifecycle) error {
	<-lifecycle.done
	h.mu.Lock()
	err := lifecycle.err
	h.mu.Unlock()
	return err
}

func DefaultAddr() string { return ":8080" }

func RequestLogging() func(http.Handler) http.Handler { return middleware.RequestLogging() }

func MaxBodyReader(max int64) func(http.Handler) http.Handler { return middleware.MaxBodyReader(max) }

func Compress() func(http.Handler) http.Handler { return middleware.Compress() }

func CSRF(store dreego.Store) func(http.Handler) http.Handler { return middleware.CSRF(store) }

func Recovery(errorHandler http.HandlerFunc) func(http.Handler) http.Handler {
	return middleware.Recovery(errorHandler)
}

func RequestID() func(http.Handler) http.Handler { return middleware.RequestID() }

func RequestIDFromCtx(ctx context.Context) string { return middleware.RequestIDFromCtx(ctx) }

func WithStore(r *http.Request, store dreego.Store) *http.Request { return session.WithStore(r, store) }

func BindForm(r *http.Request, target any) error { return validate.BindForm(r, target) }
