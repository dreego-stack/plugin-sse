package context

import (
	gcontext "context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/dreego-stack/dreego/core/internal/middleware"
	"github.com/dreego-stack/dreego/core/internal/session"
)

var sessionLogger = slog.New(slog.NewJSONHandler(os.Stderr, nil))

type Context interface {
	gcontext.Context
	Param(name string) string
	Data(key string) any
	Errors(field string) string
	Old(field string) string
	Redirect(url string, code int) error
	SessionVal(key string) string
	SetSessionVal(key, value string)
	DelSessionVal(key string)
	CSRFToken() string
	RequestID() string
	FormError() error
	SessionError() error
}

var ErrRedirect = errors.New("redirect")

type SSRContext struct {
	gcontext.Context
	W          http.ResponseWriter
	R          *http.Request
	data       map[string]any
	formErr    error
	sessionErr error
}

// NewSSR builds an SSRContext. A nil r is tolerated: the context falls back to
// context.Background() and Data/Set/Get work on the in-memory data map. The
// request-bound methods (Param, Query, FormValue, Redirect, SessionVal,
// SetSessionVal, DelSessionVal, DestroySession) dereference c.R and panic on a
// nil request — they are only safe when r is non-nil.
func NewSSR(w http.ResponseWriter, r *http.Request) *SSRContext {
	var ctx gcontext.Context = gcontext.Background()
	if r != nil {
		ctx = r.Context()
	}
	return &SSRContext{
		Context: ctx,
		W:       w,
		R:       r,
		data:    make(map[string]any),
	}
}

func (c *SSRContext) Param(name string) string {
	return c.R.PathValue(name)
}

func (c *SSRContext) Data(key string) any {
	if c.data == nil {
		return nil
	}
	return c.data[key]
}

func (c *SSRContext) Set(key string, value any) {
	if c.data == nil {
		c.data = make(map[string]any)
	}
	c.data[key] = value
}

func (c *SSRContext) Delete(key string) {
	if c.data == nil {
		return
	}
	delete(c.data, key)
}

func (c *SSRContext) Get(key string) string {
	if c.data == nil {
		return ""
	}
	s, _ := c.data[key].(string)
	return s
}

func (c *SSRContext) Query(key string) string {
	return c.R.URL.Query().Get(key)
}

func (c *SSRContext) FormValue(key string) string {
	if c.R == nil {
		return ""
	}
	if err := c.R.ParseForm(); err != nil {
		c.formErr = err
		return ""
	}
	return c.R.FormValue(key)
}

func (c *SSRContext) FormError() error {
	return c.formErr
}

func (c *SSRContext) SessionVal(key string) string {
	s := session.StoreFromCtx(c.Context)
	if s == nil {
		return ""
	}
	v, err := s.Get(c.R, key)
	if err != nil {
		c.sessionErr = err
		return ""
	}
	return v
}

func (c *SSRContext) SetSessionVal(key, value string) {
	s := session.StoreFromCtx(c.Context)
	if s == nil {
		return
	}
	if err := s.Set(c.W, c.R, key, value, nil); err != nil {
		c.sessionErr = err
		sessionWriteError(c.W, err)
	}
}

func (c *SSRContext) DelSessionVal(key string) {
	s := session.StoreFromCtx(c.Context)
	if s == nil {
		return
	}
	if err := s.Delete(c.W, c.R, key); err != nil {
		c.sessionErr = err
		sessionWriteError(c.W, err)
	}
}

func (c *SSRContext) DestroySession() {
	s := session.StoreFromCtx(c.Context)
	if s == nil {
		return
	}
	if err := s.Destroy(c.W, c.R); err != nil {
		c.sessionErr = err
		sessionWriteError(c.W, err)
	}
}

func sessionWriteError(w http.ResponseWriter, err error) {
	sessionLogger.Error("session write failed", "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (c *SSRContext) SessionError() error {
	return c.sessionErr
}

func (c *SSRContext) CSRFToken() string {
	return c.SessionVal("csrf_token")
}

func (c *SSRContext) RequestID() string {
	return middleware.RequestIDFromCtx(c.Context)
}

func (c *SSRContext) Errors(field string) string {
	return c.Get("error_" + field)
}

func (c *SSRContext) Old(field string) string {
	return c.Get("old_" + field)
}

func (c *SSRContext) Redirect(url string, code int) error {
	http.Redirect(c.W, c.R, url, code)
	return ErrRedirect
}
