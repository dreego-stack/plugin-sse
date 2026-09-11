package core

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	corei18n "github.com/dreego-stack/dreego/core/internal/i18n"
)

type MessageArg = corei18n.Argument
type MessageArgumentFormat = corei18n.ArgumentFormat
type MessageValue = corei18n.Value
type MessageSelector = corei18n.Selector
type LocalizedMessage = corei18n.Message
type LocaleCatalog = corei18n.LocaleCatalog
type I18nConfig = corei18n.Config
type Localizer = corei18n.Localizer
type LocaleResolver = corei18n.Resolver

type MessageNumber interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

func StringMessageArg(name, value string) MessageArg {
	return MessageArg{Name: name, Value: value}
}

func NumberMessageArg[T MessageNumber](name string, value T) MessageArg {
	return MessageArg{Name: name, Value: value}
}

func TimeMessageArg(name string, value time.Time) MessageArg {
	return MessageArg{Name: name, Value: value}
}

func MessageText(text string) MessageValue {
	return corei18n.Text(text)
}

func MessagePlural(argument, kind string, cases map[string]MessageValue) MessageValue {
	return corei18n.Plural(argument, kind, cases)
}

func MessageSelect(argument string, cases map[string]MessageValue) MessageValue {
	return corei18n.Select(argument, cases)
}

func NewLocalizer(config I18nConfig) (Localizer, error) {
	return corei18n.NewCatalogLocalizer(config)
}

func Localize(ctx context.Context, key string, arguments ...MessageArg) (string, error) {
	localizer, locale, ok := corei18n.FromContext(ctx)
	if !ok {
		return key, nil
	}
	return localizer.Localize(ctx, locale, key, arguments)
}

func Message(ctx context.Context, key string, arguments ...MessageArg) string {
	value, err := Localize(ctx, key, arguments...)
	if err != nil {
		slog.Error("dreego: message localization failed", "key", key, "error", err)
		return key
	}
	return value
}

func Locale(ctx context.Context) string {
	return corei18n.Locale(ctx)
}

func WithLocale(ctx context.Context, localizer Localizer, locale string) context.Context {
	return corei18n.WithContext(ctx, localizer, locale)
}

func LocalizedHTML(ctx context.Context, document string) string {
	return corei18n.LocalizeDocument(document, Locale(ctx))
}

func SetLocale(w http.ResponseWriter, r *http.Request, locale string) error {
	return corei18n.SetLocale(w, r, locale)
}
