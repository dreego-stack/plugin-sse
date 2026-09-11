package md

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	imageRe       = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)
	footnoteRefRe = regexp.MustCompile(`\[\^(\d+)\]`)
	footnoteDefRe = regexp.MustCompile(`^\[\^(\d+)\]:\s*(.*)$`)
)

type mdRenderer struct {
	mode      Mode
	defs      map[string]string
	refCounts map[string]int
	refOrder  []string
}

func newRenderer(mode Mode) *mdRenderer {
	return &mdRenderer{mode: mode, defs: map[string]string{}, refCounts: map[string]int{}}
}

func (r *mdRenderer) addDef(num, content string) {
	if _, ok := r.defs[num]; !ok {
		r.defs[num] = r.renderInline(content)
	}
}

func (r *mdRenderer) hasDefs() bool {
	return len(r.defs) > 0
}

func (r *mdRenderer) renderInline(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		i := inlineHTMLStart(s)
		if i < 0 {
			b.WriteString(r.renderInlineText(s))
			break
		}
		b.WriteString(r.renderInlineText(s[:i]))
		end := strings.IndexByte(s[i:], '>')
		if end < 0 {
			if r.mode == ModeSafe {
				b.WriteString(r.renderInlineText(s[i:]))
			} else {
				b.WriteString(s[i:])
			}
			break
		}
		if r.mode == ModeSafe {
			b.WriteString(r.renderInlineText(s[i : i+end+1]))
		} else {
			b.WriteString(s[i : i+end+1])
		}
		s = s[i+end+1:]
	}
	return b.String()
}

func inlineHTMLStart(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '`' {
			if end := strings.IndexByte(s[i+1:], '`'); end >= 0 {
				i += end + 1
			}
			continue
		}
		if s[i] == '<' && i+1 < len(s) && (isHTMLNameStart(s[i+1]) || s[i+1] == '/') {
			return i
		}
	}
	return -1
}

func (r *mdRenderer) renderInlineText(s string) string {
	s = html.EscapeString(s)
	s = codeRe.ReplaceAllString(s, "<code>$1</code>")
	s = imageRe.ReplaceAllStringFunc(s, func(m string) string {
		p := imageRe.FindStringSubmatch(m)
		u := safeURL(p[2], true)
		if u == "" {
			return m
		}
		return `<img src="` + html.EscapeString(u) + `" alt="` + p[1] + `">`
	})
	s = linkRe.ReplaceAllStringFunc(s, func(m string) string {
		p := linkRe.FindStringSubmatch(m)
		u := safeURL(p[2], false)
		if u == "" {
			return m
		}
		return `<a href="` + html.EscapeString(u) + `">` + p[1] + `</a>`
	})
	s = strongRe.ReplaceAllString(s, "<strong>$1</strong>")
	s = emRe.ReplaceAllString(s, "<em>$1</em>")
	s = r.replaceFootnoteRefs(s)
	return s
}

func (r *mdRenderer) replaceFootnoteRefs(s string) string {
	return footnoteRefRe.ReplaceAllStringFunc(s, func(m string) string {
		num := footnoteRefRe.FindStringSubmatch(m)[1]
		r.refCounts[num]++
		count := r.refCounts[num]
		if count == 1 {
			r.refOrder = append(r.refOrder, num)
		}
		return `<sup class="footnote-ref"><a href="#fn-` + num + `" id="fnref-` + num + `">` + strconv.Itoa(count) + `</a></sup>`
	})
}

func (r *mdRenderer) footnotesSection() string {
	if len(r.refOrder) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="footnotes"><ol>`)
	for _, num := range r.refOrder {
		content := r.defs[num]
		b.WriteString(`<li id="fn-` + num + `">` + content + ` <a href="#fnref-` + num + `" class="footnote-backref">↩</a></li>`)
	}
	b.WriteString(`</ol></section>`)
	return b.String()
}

func safeURL(raw string, isImage bool) string {
	u := html.UnescapeString(raw)
	// html.UnescapeString does a single pass: a decoded &amp; can reveal a
	// further entity (e.g. &amp;#x09; -> &#x09;). Decode until stable so
	// entity-encoded control characters are visible to the scheme check and
	// cannot smuggle a javascript: scheme past hasScheme.
	for {
		next := html.UnescapeString(u)
		if next == u {
			break
		}
		u = next
	}
	u = strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, r == 0x7f, r == ' ', r == '\t', r == '\n', r == '\r':
			return -1
		}
		return r
	}, u)
	if u == "" {
		return ""
	}
	if !hasScheme(u) {
		return u
	}
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") {
		return u
	}
	if isImage && strings.HasPrefix(lower, "data:image/") {
		rest := lower[len("data:image/"):]
		baseIdx := strings.Index(rest, ";base64,")
		if baseIdx > 0 {
			switch rest[:baseIdx] {
			case "png", "jpeg", "gif", "webp":
				return u
			}
		}
	}
	return ""
}

func hasScheme(u string) bool {
	for i := 0; i < len(u); i++ {
		switch u[i] {
		case ':':
			return i > 0
		case '/', '?', '#':
			return false
		}
	}
	return false
}
