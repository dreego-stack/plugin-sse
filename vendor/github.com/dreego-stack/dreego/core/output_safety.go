package core

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// SafeText renders v for an HTML text context. Markup and quotes are escaped so
// the value cannot introduce elements or attributes.
func SafeText(v any) string {
	return html.EscapeString(fmt.Sprintf("%v", v))
}

// SafeAttr renders v for a quoted HTML attribute value. Markup and quotes are
// escaped so the value cannot break out of the attribute.
func SafeAttr(v any) string {
	return html.EscapeString(fmt.Sprintf("%v", v))
}

func SafeSrcdoc(v any) string {
	return html.EscapeString(html.EscapeString(fmt.Sprintf("%v", v)))
}

// SafeURL renders v for a URL attribute such as href, src, or action. Values
// with an unsafe or unknown scheme (javascript:, data:, vbscript:, file:, …)
// are replaced with "#". Relative URLs and http, https, mailto, and tel are
// allowed. The result is HTML-escaped.
func SafeURL(v any) string {
	s := fmt.Sprintf("%v", v)
	if !safeURLScheme(s) {
		return "#"
	}
	return html.EscapeString(s)
}

// SafeScript renders v for a script context such as an event handler attribute.
// The value is JSON-encoded, so it becomes a JS string literal and can never be
// executed as code.
func SafeScript(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return html.EscapeString(string(b))
}

// SafeStyle renders v for a style context. Markup is escaped, so </style> and
// <!-- sequences become inert text even when the input mixes case.
func SafeStyle(v any) string {
	return html.EscapeString(fmt.Sprintf("%v", v))
}

// SafeRefresh renders v for a <meta http-equiv="refresh"> content attribute.
// The URL portion after "url=" is extracted with the same tolerance for
// whitespace and case that browsers use, scheme-validated like SafeURL, and an
// unsafe URL is replaced with "#". The result is HTML-escaped.
func SafeRefresh(v any) string {
	s := fmt.Sprintf("%v", v)
	m := refreshURLRE.FindStringSubmatchIndex(s)
	if m == nil {
		return html.EscapeString(s)
	}
	if !safeURLScheme(s[m[2]:m[3]]) {
		return html.EscapeString(s[:m[2]] + "#")
	}
	return html.EscapeString(s)
}

// refreshURLRE matches the url= portion of a refresh content value with the
// tolerant whitespace and case handling browsers apply. The capture group
// holds the URL after "url=". DOTALL mode makes the capture span newlines so
// obfuscated schemes such as "java\nscript:" reach the scheme validator
// instead of being truncated before the colon.
var refreshURLRE = regexp.MustCompile(`(?is)url\s*=\s*(.*)`)

// SafeRaw renders v without any escaping. It is the explicit opt-in for
// trusted HTML, URLs, or scripts; using it with untrusted input reintroduces
// the XSS risk that the context rules prevent.
func SafeRaw(v any) string {
	return fmt.Sprintf("%v", v)
}

func safeURLScheme(s string) bool {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return true
	}
	if strings.Contains(trimmed, ",") {
		for _, part := range strings.Split(trimmed, ",") {
			if !safeURLScheme(part) {
				return false
			}
		}
		return true
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "//") {
		return true
	}
	colon := strings.IndexByte(lower, ':')
	if colon < 0 {
		return true
	}
	scheme := lower[:colon]
	for i := 0; i < len(scheme); i++ {
		c := scheme[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '+' && c != '.' && c != '-' {
			return false
		}
	}
	switch scheme {
	case "http", "https", "mailto", "tel":
		return true
	}
	return false
}
