package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func RequestLogging() func(http.Handler) http.Handler {
	logger := slog.New(&jsonlHandler{w: os.Stderr})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)
			logger.Info("request",
				"rid", RequestIDFromCtx(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"ip", r.RemoteAddr,
				"duration", time.Since(start).String(),
			)
		})
	}
}

type jsonlHandler struct {
	w io.Writer
}

func (h *jsonlHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *jsonlHandler) Handle(_ context.Context, r slog.Record) error {
	m := map[string]any{"time": r.Time.UTC().Format("2006-01-02T15:04:05")}
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(h.w, string(b)+"\n")
	return err
}

func (h *jsonlHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *jsonlHandler) WithGroup(_ string) slog.Handler      { return h }
