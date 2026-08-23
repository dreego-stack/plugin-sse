package core

import (
	stdctx "context"
	"net/http"

	"github.com/dreego-stack/dreego/core/internal/context"
	"github.com/dreego-stack/dreego/core/internal/middleware"
	"github.com/dreego-stack/dreego/core/internal/server"
	"github.com/dreego-stack/dreego/core/internal/session"
	"github.com/dreego-stack/dreego/core/internal/validate"
)

var ErrRedirect = context.ErrRedirect
var ErrAppBuilt = server.ErrAppBuilt
var ErrRouteConflict = server.ErrRouteConflict
var ErrServerRunning = server.ErrServerRunning
var ErrSessionTooLarge = session.ErrSessionTooLarge
var ErrCookiePathOverride = session.ErrCookiePathOverride

type Context = context.Context

type SSRContext = context.SSRContext

type Component interface {
	Render(ctx *SSRContext) (string, error)
}

type ComponentFunc func(ctx *SSRContext) (string, error)

func (f ComponentFunc) Render(ctx *SSRContext) (string, error) {
	return f(ctx)
}

type App = server.App

func New() *App {
	return server.New()
}

func NewSSR(w http.ResponseWriter, r *http.Request) *SSRContext {
	return context.NewSSR(w, r)
}

type Store = session.Store
type Options = session.Options
type CookieStore = session.CookieStore
type CookiePolicy = session.CookiePolicy

func NewCookieStore(secret []byte) *CookieStore {
	return session.NewCookieStore(secret)
}

func WithStore(r *http.Request, s Store) *http.Request {
	return session.WithStore(r, s)
}

func StoreFromCtx(ctx stdctx.Context) Store {
	return session.StoreFromCtx(ctx)
}

func BindForm(r *http.Request, target any) error {
	return validate.BindForm(r, target)
}

func ValidateForm(form any) map[string]string {
	return validate.ValidateForm(form)
}

type Setter = validate.Setter

func SaveOld(c *SSRContext, form any) {
	validate.SaveOld(c, form)
}

func SaveErrors(c *SSRContext, errs map[string]string) {
	validate.SaveErrors(c, errs)
}

func RequestLogging() func(http.Handler) http.Handler {
	return middleware.RequestLogging()
}

func MaxBodyReader(max int64) func(http.Handler) http.Handler {
	return middleware.MaxBodyReader(max)
}

func Compress() func(http.Handler) http.Handler {
	return middleware.Compress()
}

func CSRF(store Store) func(http.Handler) http.Handler {
	return middleware.CSRF(store)
}

func Recovery(errorHandler http.HandlerFunc) func(http.Handler) http.Handler {
	return middleware.Recovery(errorHandler)
}

func RequestID() func(http.Handler) http.Handler {
	return middleware.RequestID()
}

func RequestIDFromCtx(ctx stdctx.Context) string {
	return middleware.RequestIDFromCtx(ctx)
}

type ServerConfig = server.ServerConfig

func DefaultServerConfig() ServerConfig {
	return server.DefaultServerConfig()
}
