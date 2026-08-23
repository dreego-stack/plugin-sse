package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ridCtxKey string

const RequestIDKey ridCtxKey = "request_id"

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = newID()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), RequestIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic("requestid: failed to read random bytes: " + err.Error())
	}
	dst := make([]byte, 16)
	hex.Encode(dst, b)
	return string(dst)
}

func RequestIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDKey).(string); ok {
		return v
	}
	return ""
}
