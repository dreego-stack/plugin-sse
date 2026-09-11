package i18n

import (
	"context"
	"fmt"
	"golang.org/x/text/language"
)

type CatalogLocalizer struct {
	defaultLocale string
	catalogs      map[string]LocaleCatalog
	tags          map[string]language.Tag
	fallbacks     map[string][]string
}

func NewCatalogLocalizer(config Config) (*CatalogLocalizer, error) {
	config = CloneConfig(config)
	localizer := &CatalogLocalizer{
		defaultLocale: config.DefaultLocale,
		catalogs:      make(map[string]LocaleCatalog, len(config.Locales)),
		tags:          make(map[string]language.Tag, len(config.Locales)),
		fallbacks:     make(map[string][]string, len(config.Fallbacks)),
	}
	for _, catalog := range config.Locales {
		tag, err := language.Parse(catalog.Locale)
		if err != nil {
			return nil, fmt.Errorf("locale %q: %w", catalog.Locale, err)
		}
		locale := tag.String()
		if _, exists := localizer.catalogs[locale]; exists {
			return nil, fmt.Errorf("duplicate locale %q", locale)
		}
		localizer.catalogs[locale] = catalog
		localizer.tags[locale] = tag
	}
	defaultTag, err := language.Parse(config.DefaultLocale)
	if err != nil {
		return nil, fmt.Errorf("default locale: %w", err)
	}
	localizer.defaultLocale = defaultTag.String()
	if _, exists := localizer.catalogs[localizer.defaultLocale]; !exists {
		return nil, fmt.Errorf("default locale %q has no catalog", localizer.defaultLocale)
	}
	for source, targets := range config.Fallbacks {
		sourceTag, err := language.Parse(source)
		if err != nil {
			return nil, fmt.Errorf("fallback source %q: %w", source, err)
		}
		canonicalSource := sourceTag.String()
		if _, exists := localizer.catalogs[canonicalSource]; !exists {
			return nil, fmt.Errorf("fallback source %q has no catalog", canonicalSource)
		}
		for _, target := range targets {
			targetTag, err := language.Parse(target)
			if err != nil {
				return nil, fmt.Errorf("fallback target %q: %w", target, err)
			}
			canonicalTarget := targetTag.String()
			if _, exists := localizer.catalogs[canonicalTarget]; !exists {
				return nil, fmt.Errorf("fallback target %q has no catalog", canonicalTarget)
			}
			localizer.fallbacks[canonicalSource] = append(localizer.fallbacks[canonicalSource], canonicalTarget)
		}
	}
	if err := localizer.validateFallbacks(); err != nil {
		return nil, err
	}
	return localizer, nil
}

func (l *CatalogLocalizer) Localize(_ context.Context, locale, key string, arguments []Argument) (string, error) {
	if tag, err := language.Parse(locale); err == nil {
		locale = tag.String()
	}
	entry, locale, exists := l.findMessage(locale, key)
	if !exists {
		return "", fmt.Errorf("message %q is not defined", key)
	}
	values := make(map[string]any, len(arguments))
	for _, argument := range arguments {
		values[argument.Name] = argument.Value
	}
	tag := l.tags[locale]
	return renderValue(entry.Value, entry.Arguments, values, tag)
}

func (l *CatalogLocalizer) findMessage(locale, key string) (Message, string, bool) {
	visited := map[string]bool{}
	var find func(string) (Message, string, bool)
	find = func(candidate string) (Message, string, bool) {
		if visited[candidate] {
			return Message{}, "", false
		}
		visited[candidate] = true
		if catalog, exists := l.catalogs[candidate]; exists {
			if message, exists := catalog.Messages[key]; exists {
				return message, candidate, true
			}
		}
		for _, fallback := range l.fallbacks[candidate] {
			if message, matched, exists := find(fallback); exists {
				return message, matched, true
			}
		}
		return Message{}, "", false
	}
	if message, matched, exists := find(locale); exists {
		return message, matched, true
	}
	return find(l.defaultLocale)
}

func (l *CatalogLocalizer) validateFallbacks() error {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string) error
	visit = func(locale string) error {
		if visiting[locale] {
			return fmt.Errorf("fallback cycle includes locale %q", locale)
		}
		if visited[locale] {
			return nil
		}
		visiting[locale] = true
		for _, target := range l.fallbacks[locale] {
			if err := visit(target); err != nil {
				return err
			}
		}
		visiting[locale] = false
		visited[locale] = true
		return nil
	}
	for locale := range l.fallbacks {
		if err := visit(locale); err != nil {
			return err
		}
	}
	return nil
}
