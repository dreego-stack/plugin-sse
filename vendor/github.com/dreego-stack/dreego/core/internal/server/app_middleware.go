package server

import (
	"net/http"

	sess "github.com/dreego-stack/dreego/core/internal/session"
)

func (a *App) redirectRewriteMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, rw := range a.rewrites {
			if target, ok := applyRewrite(rw, r.URL.Path); ok {
				r2 := r.Clone(r.Context())
				r2.URL.Path = target
				r2.URL.RawPath = ""
				r = r2
			}
		}

		for _, rd := range a.redirects {
			if target, ok := matchRedirect(rd, r.URL.Path); ok {
				http.Redirect(w, r, target, rd.status)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (a *App) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, sess.WithStore(r, a.sessionStore))
	})
}

func (a *App) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

func (a *App) readyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.ready.Load() {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready"))
	}
}

func (a *App) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		csp := a.cspHeader
		if csp == "" {
			csp = "default-src 'self'"
		}
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}
