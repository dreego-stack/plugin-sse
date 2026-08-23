package session

import (
	"context"
	"net/http"
)

type ctxKey struct{}

type storeCtxKey struct{}

func WithStore(r *http.Request, s Store) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), storeCtxKey{}, s))
}

func StoreFromCtx(ctx context.Context) Store {
	s, _ := ctx.Value(storeCtxKey{}).(Store)
	return s
}
