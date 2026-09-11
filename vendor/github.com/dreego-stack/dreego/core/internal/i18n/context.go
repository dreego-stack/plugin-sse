package i18n

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type contextState struct {
	localizer  Localizer
	locale     string
	cookieName string
	supported  map[string]string
}

type contextKey struct{}

func WithContext(ctx context.Context, localizer Localizer, locale string) context.Context {
	return context.WithValue(ctx, contextKey{}, contextState{localizer: localizer, locale: locale})
}

func withNegotiation(ctx context.Context, localizer Localizer, locale, cookieName string, supported map[string]string) context.Context {
	return context.WithValue(ctx, contextKey{}, contextState{localizer: localizer, locale: locale, cookieName: cookieName, supported: supported})
}

func FromContext(ctx context.Context) (Localizer, string, bool) {
	state, ok := ctx.Value(contextKey{}).(contextState)
	return state.localizer, state.locale, ok && state.localizer != nil
}

func Locale(ctx context.Context) string {
	state, _ := ctx.Value(contextKey{}).(contextState)
	return state.locale
}

func SetLocale(w http.ResponseWriter, r *http.Request, locale string) error {
	state, ok := r.Context().Value(contextKey{}).(contextState)
	if !ok || len(state.supported) == 0 {
		return fmt.Errorf("i18n is not configured for this request")
	}
	canonical := state.supported[strings.ToLower(locale)]
	if canonical == "" {
		return fmt.Errorf("locale %q is not supported", locale)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     state.cookieName,
		Value:    canonical,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		MaxAge:   365 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}
