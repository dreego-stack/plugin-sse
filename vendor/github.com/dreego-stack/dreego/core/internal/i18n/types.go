package i18n

import (
	"context"
	"maps"
	"net/http"
)

type Argument struct {
	Name  string
	Value any
}

type ArgumentFormat struct {
	Format           string
	CurrencyArgument string
	TimeZoneArgument string
	Layout           string
}

type Value struct {
	Text     *string
	Selector *Selector
}

type Selector struct {
	Argument string
	Kind     string
	Cases    map[string]Value
}

type Message struct {
	Value     Value
	Arguments map[string]ArgumentFormat
}

type LocaleCatalog struct {
	Locale   string
	Messages map[string]Message
}

type Config struct {
	DefaultLocale string
	Locales       []LocaleCatalog
	Detection     []string
	CookieName    string
	Account       Resolver
	Resolvers     []Resolver
	URLStrategy   string
	Domains       map[string]string
	Fallbacks     map[string][]string
}

type Resolver func(*http.Request) string

type Localizer interface {
	Localize(ctx context.Context, locale, key string, arguments []Argument) (string, error)
}

func Text(text string) Value {
	return Value{Text: &text}
}

func Plural(argument, kind string, cases map[string]Value) Value {
	return Value{Selector: &Selector{Argument: argument, Kind: kind, Cases: cases}}
}

func Select(argument string, cases map[string]Value) Value {
	return Value{Selector: &Selector{Argument: argument, Kind: "select", Cases: cases}}
}

func CloneConfig(source Config) Config {
	clone := source
	clone.Detection = append([]string(nil), source.Detection...)
	clone.Resolvers = append([]Resolver(nil), source.Resolvers...)
	clone.Domains = make(map[string]string, len(source.Domains))
	maps.Copy(clone.Domains, source.Domains)
	clone.Fallbacks = make(map[string][]string, len(source.Fallbacks))
	for locale, fallbacks := range source.Fallbacks {
		clone.Fallbacks[locale] = append([]string(nil), fallbacks...)
	}
	clone.Locales = make([]LocaleCatalog, len(source.Locales))
	for index, catalog := range source.Locales {
		messages := make(map[string]Message, len(catalog.Messages))
		for key, message := range catalog.Messages {
			formats := make(map[string]ArgumentFormat, len(message.Arguments))
			maps.Copy(formats, message.Arguments)
			messages[key] = Message{Value: cloneValue(message.Value), Arguments: formats}
		}
		clone.Locales[index] = LocaleCatalog{Locale: catalog.Locale, Messages: messages}
	}
	return clone
}

func cloneValue(value Value) Value {
	clone := Value{Text: value.Text}
	if value.Text != nil {
		text := *value.Text
		clone.Text = &text
	}
	if value.Selector != nil {
		selector := &Selector{Argument: value.Selector.Argument, Kind: value.Selector.Kind, Cases: make(map[string]Value, len(value.Selector.Cases))}
		for name, child := range value.Selector.Cases {
			selector.Cases[name] = cloneValue(child)
		}
		clone.Selector = selector
	}
	return clone
}
