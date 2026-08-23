package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
)

func Recovery(errorHandler http.HandlerFunc) func(http.Handler) http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
						"stack", string(debug.Stack()),
					)
					if errorHandler != nil {
						w.Header().Set("Content-Type", "text/html; charset=utf-8")
						w.WriteHeader(http.StatusInternalServerError)
						errorHandler(w, r)
					} else {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
