package ir

import "strings"

const mdtohtmlMarker = "dreego.mdtohtml("

// TranslateMdtohtml rewrites the dreego.mdtohtml(...) stdlib syntax in server
// section code into the exported core calls. The trust decision is visible at
// the call site, per use:
//
//	dreego.mdtohtml(x)                 -> dreego.MarkdownToHTML(x)
//	dreego.mdtohtml(x, trusted: true)  -> dreego.MarkdownToHTMLTrusted(x)
//	dreego.mdtohtml(x, trusted: false) -> dreego.MarkdownToHTML(x)
//
// The trusted: true/false argument is stripped and the function name is
// rewritten. If the call cannot be parsed (unbalanced parens), the code is left
// unchanged so the Go compiler reports the error naturally.
func TranslateMdtohtml(code string) string {
	var b strings.Builder
	rest := code
	for {
		idx := strings.Index(rest, mdtohtmlMarker)
		if idx < 0 {
			b.WriteString(rest)
			break
		}
		b.WriteString(rest[:idx])
		open := idx + len(mdtohtmlMarker) - 1
		close, ok := matchParen(rest, open)
		if !ok {
			b.WriteString(rest[idx:])
			break
		}
		inner := rest[open+1 : close]
		content, trusted, recognized := splitTrustedArg(inner)
		if !recognized {
			b.WriteString(rest[idx : close+1])
			rest = rest[close+1:]
			continue
		}
		if trusted {
			b.WriteString("dreego.MarkdownToHTMLTrusted(")
		} else {
			b.WriteString("dreego.MarkdownToHTML(")
		}
		b.WriteString(content)
		b.WriteString(")")
		rest = rest[close+1:]
	}
	return b.String()
}

// matchParen returns the index of the closing paren that matches the opening
// paren at open, tracking nested parens. It returns ok=false when unbalanced.
func matchParen(s string, open int) (int, bool) {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// splitTrustedArg splits the inner argument list on the last top-level comma and
// reports whether the final argument is a trusted: true/false marker. The
// returned content excludes the marker argument. recognized is false when the
// last argument is not a recognized trusted: marker, in which case the call is
// left unchanged.
func splitTrustedArg(inner string) (content string, trusted bool, recognized bool) {
	lastComma := -1
	depth := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				lastComma = i
			}
		}
	}
	if lastComma < 0 {
		return inner, false, true
	}
	last := strings.TrimSpace(inner[lastComma+1:])
	switch last {
	case "trusted: true":
		return strings.TrimSpace(inner[:lastComma]), true, true
	case "trusted: false":
		return strings.TrimSpace(inner[:lastComma]), false, true
	}
	return inner, false, false
}
