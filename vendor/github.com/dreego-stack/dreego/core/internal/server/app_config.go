package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dreego-stack/dreego/core/internal/validate"
)

var ErrAppBuilt = errors.New("dreego: app configuration is frozen")
var ErrRouteConflict = errors.New("dreego: route conflict")

func (a *App) handleMux(fn func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("dreego: conflicting route patterns: %v", recovered)
		}
	}()
	fn()
	return nil
}

func (a *App) warnMissingSessionStore() {
	slog.Warn("dreego: CSRF is enabled but no session store is configured; CSRF protection will not be active")
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.Handler().ServeHTTP(w, r)
}

func (a *App) mutable() error {
	if a.built {
		return ErrAppBuilt
	}
	return nil
}

func (a *App) Register(method, pattern string, handler http.HandlerFunc) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	if err := validateRoutePattern(method, pattern, handler); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	if method == http.MethodGet && (pattern == "/health" || pattern == "/ready") {
		return fmt.Errorf("%w: %s %s is reserved", ErrRouteConflict, method, pattern)
	}
	for _, existing := range a.routes {
		if existing.method == method && existing.pattern == pattern {
			return fmt.Errorf("%w: %s %s", ErrRouteConflict, method, pattern)
		}
	}
	a.routes = append(a.routes, route{method, pattern, handler})
	return nil
}

func validateRoutePattern(method, pattern string, handler http.HandlerFunc) (err error) {
	if handler == nil {
		return errors.New("dreego: route handler is nil")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("dreego: invalid route %s %s: %v", method, pattern, recovered)
		}
	}()
	registeredPattern := pattern
	if method != "" {
		registeredPattern = method + " " + pattern
	}
	http.NewServeMux().HandleFunc(registeredPattern, handler)
	return nil
}

func (a *App) RegisterRedirect(from, to string, status int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	if err := validateRedirect(from, to, status); err != nil {
		return err
	}
	a.redirects = append(a.redirects, redirectRule{from: from, to: to, status: status})
	return nil
}

func (a *App) RegisterRewrite(from, to string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	if err := validateRewrite(from, to); err != nil {
		return err
	}
	a.rewrites = append(a.rewrites, rewriteRule{from: from, to: to})
	return nil
}

func (a *App) RegisterStatic(path, mime string, content []byte) error {
	data := append([]byte(nil), content...)
	return a.Register(http.MethodGet, path, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", mime)
		_, _ = w.Write(data)
	})
}

func (a *App) Use(middlewares ...func(http.Handler) http.Handler) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	for _, middleware := range middlewares {
		if middleware != nil {
			a.middlewares = append(a.middlewares, middleware)
		}
	}
	return nil
}

func (a *App) SetLogging(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.loggingEnabled = enabled
	return nil
}

func (a *App) SetCSRF(enabled bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.csrfEnabled = enabled
	return nil
}

func (a *App) SetErrorHandler(code int, handler http.HandlerFunc) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.errorHandlers[code] = handler
	return nil
}

func (a *App) SetSessionStore(store Store) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.sessionStore = store
	return nil
}

func (a *App) SessionStore() Store {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sessionStore
}

func (a *App) SetCSP(value string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.cspHeader = value
	return nil
}

func (a *App) RegisterRule(name string, rule func(string) string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.mutable(); err != nil {
		return err
	}
	a.customRules[name] = rule
	return nil
}

func (a *App) ValidateForm(form any) map[string]string {
	a.mu.RLock()
	rules := make(map[string]validate.ValidatorFunc, len(a.customRules))
	for name, rule := range a.customRules {
		rules[name] = rule
	}
	a.mu.RUnlock()
	return validate.Validate(form, rules)
}
