package i18n

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"golang.org/x/text/language"
)

type Negotiator struct {
	config     Config
	localizer  Localizer
	locales    []string
	matcher    language.Matcher
	cookieName string
	byTag      map[string]string
	byDomain   map[string]string
}

func NewNegotiator(config Config, localizer Localizer) (*Negotiator, error) {
	if len(config.Locales) == 0 {
		return nil, fmt.Errorf("at least one locale is required")
	}
	tags := make([]language.Tag, 0, len(config.Locales))
	locales := make([]string, 0, len(config.Locales))
	byTag := make(map[string]string, len(config.Locales))
	for _, catalog := range config.Locales {
		tag, err := language.Parse(catalog.Locale)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
		locales = append(locales, tag.String())
		byTag[strings.ToLower(tag.String())] = tag.String()
	}
	defaultTag, err := language.Parse(config.DefaultLocale)
	if err != nil || byTag[strings.ToLower(defaultTag.String())] == "" {
		return nil, fmt.Errorf("default locale %q is not supported", config.DefaultLocale)
	}
	config.DefaultLocale = defaultTag.String()
	cookieName := config.CookieName
	if cookieName == "" {
		cookieName = "dreego_locale"
	}
	if len(config.Detection) == 0 {
		config.Detection = []string{"cookie", "browser", "custom", "default"}
	}
	byDomain := make(map[string]string, len(config.Domains))
	for locale, domain := range config.Domains {
		byDomain[strings.ToLower(domain)] = locale
	}
	return &Negotiator{config: config, localizer: localizer, locales: locales, matcher: language.NewMatcher(tags), cookieName: cookieName, byTag: byTag, byDomain: byDomain}, nil
}

func (n *Negotiator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale, prefix := n.explicitLocale(r)
		if locale == "" {
			locale = n.Resolve(r)
		}
		if prefix != "" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			r.URL.RawPath = ""
		}
		w.Header().Set("Content-Language", locale)
		for _, detector := range n.config.Detection {
			switch detector {
			case "browser":
				addVary(w.Header(), "Accept-Language")
			case "cookie":
				addVary(w.Header(), "Cookie")
			case "account":
				if n.config.Account != nil {
					w.Header().Set("Vary", "*")
				}
			case "custom":
				if len(n.config.Resolvers) > 0 {
					w.Header().Set("Vary", "*")
				}
			}
		}
		ctx := withNegotiation(r.Context(), n.localizer, locale, n.cookieName, n.byTag)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func addVary(header http.Header, value string) {
	values := header.Values("Vary")
	if len(values) == 1 && values[0] == "*" {
		return
	}
	for _, existing := range values {
		for part := range strings.SplitSeq(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	values = append(values, value)
	header.Set("Vary", strings.Join(values, ", "))
}

func (n *Negotiator) Resolve(r *http.Request) string {
	if locale, _ := n.explicitLocale(r); locale != "" {
		return locale
	}
	for _, detector := range n.config.Detection {
		switch detector {
		case "account":
			if locale := n.resolveCandidate(n.config.Account, r); locale != "" {
				return locale
			}
		case "cookie":
			if cookie, err := r.Cookie(n.cookieName); err == nil {
				if locale := n.match([]string{cookie.Value}); locale != "" {
					return locale
				}
			}
		case "browser":
			tags, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
			if err == nil {
				preferred := make([]string, len(tags))
				for index, tag := range tags {
					preferred[index] = tag.String()
				}
				if locale := n.match(preferred); locale != "" {
					return locale
				}
			}
		case "custom":
			for _, resolver := range n.config.Resolvers {
				if locale := n.resolveCandidate(resolver, r); locale != "" {
					return locale
				}
			}
		case "default":
			return n.config.DefaultLocale
		}
	}
	return n.config.DefaultLocale
}

func (n *Negotiator) explicitLocale(request *http.Request) (string, string) {
	switch n.config.URLStrategy {
	case "prefix":
		path := strings.TrimPrefix(request.URL.Path, "/")
		segment, _, _ := strings.Cut(path, "/")
		if locale := n.byTag[strings.ToLower(segment)]; locale != "" {
			return locale, "/" + segment
		}
	case "domain":
		host := request.Host
		if parsed, _, err := net.SplitHostPort(host); err == nil {
			host = parsed
		}
		return n.byDomain[strings.ToLower(host)], ""
	}
	return "", ""
}

func (n *Negotiator) resolveCandidate(resolver Resolver, request *http.Request) string {
	if resolver == nil {
		return ""
	}
	return n.match([]string{resolver(request)})
}

func (n *Negotiator) match(preferred []string) string {
	if len(preferred) == 0 || preferred[0] == "" {
		return ""
	}
	tags := parseTags(preferred)
	if len(tags) == 0 {
		return ""
	}
	_, index, confidence := n.matcher.Match(tags...)
	if confidence == language.No || index < 0 || index >= len(n.locales) {
		return ""
	}
	return n.locales[index]
}

func parseTags(values []string) []language.Tag {
	tags := make([]language.Tag, 0, len(values))
	for _, value := range values {
		if tag, err := language.Parse(value); err == nil {
			tags = append(tags, tag)
		}
	}
	return tags
}
