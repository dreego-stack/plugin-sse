package md

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/dreego-stack/dreego/internal/transpiler/ir"
)

type Mode int

const (
	ModeTrusted Mode = iota
	ModeSafe
)

var (
	atxHeading     = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	ulItem         = regexp.MustCompile(`^[-*+]\s+(.*)$`)
	olItem         = regexp.MustCompile(`^(\d+)\.\s+(.*)$`)
	listMarker     = regexp.MustCompile(`^([-*+]|\d+\.)\s*$`)
	codeRe         = regexp.MustCompile("`([^`]*)`")
	linkRe         = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	strongRe       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	emRe           = regexp.MustCompile(`\*([^*]+)\*`)
	htmlBlockStart = regexp.MustCompile(`^</?([a-zA-Z][a-zA-Z0-9-]*)(\s|>|/)`)
	fenceLanguage  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_+.#-]*$`)
)

func safeFenceLanguage(info string) string {
	if fenceLanguage.MatchString(info) {
		return info
	}
	return ""
}

func ToNodes(src string, mode Mode) ([]ir.TemplateNode, error) {
	return parseBlocks(src, newRenderer(mode))
}

func parseBlocks(src string, r *mdRenderer) ([]ir.TemplateNode, error) {
	lines := strings.Split(src, "\n")
	var nodes []ir.TemplateNode
	var para []string

	flushPara := func() {
		if len(para) > 0 {
			nodes = append(nodes, textNode("<p>"+r.renderInline(strings.Join(para, " "))+"</p>"))
			para = nil
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			flushPara()
			lang := safeFenceLanguage(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			var code []string
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				code = append(code, html.EscapeString(lines[i]))
				i++
			}
			open := "<pre><code"
			if lang != "" {
				open += ` class="language-` + html.EscapeString(lang) + `"`
			}
			open += ">"
			nodes = append(nodes, textNode(open+strings.Join(code, "\n")+"</code></pre>"))
			continue
		}

		if htmlBlockStart.MatchString(trimmed) {
			flushPara()
			var raw []string
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				if r.mode == ModeSafe {
					raw = append(raw, html.EscapeString(lines[i]))
				} else {
					raw = append(raw, lines[i])
				}
				i++
			}
			i--
			nodes = append(nodes, textNode(strings.Join(raw, "\n")))
			continue
		}

		if m := atxHeading.FindStringSubmatch(trimmed); m != nil {
			flushPara()
			level := len(m[1])
			nodes = append(nodes, textNode(fmt.Sprintf("<h%d>%s</h%d>", level, r.renderInline(m[2]), level)))
			continue
		}

		if isHR(trimmed) {
			flushPara()
			nodes = append(nodes, textNode("<hr>"))
			continue
		}

		if strings.HasPrefix(trimmed, ">") {
			flushPara()
			var q []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				q = append(q, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), ">")))
				i++
			}
			i--
			nodes = append(nodes, textNode("<blockquote>"+r.renderInline(strings.Join(q, " "))+"</blockquote>"))
			continue
		}

		if listMarker.MatchString(trimmed) {
			return nil, fmt.Errorf("md: unparseable list marker on line %d", i+1)
		}

		if m := ulItem.FindStringSubmatch(trimmed); m != nil {
			flushPara()
			items, next := parseList(lines, i, r)
			i = next
			nodes = append(nodes, textNode("<ul>"+strings.Join(items, "")+"</ul>"))
			continue
		}

		if m := olItem.FindStringSubmatch(trimmed); m != nil {
			flushPara()
			items, next := parseList(lines, i, r)
			i = next
			nodes = append(nodes, textNode("<ol>"+strings.Join(items, "")+"</ol>"))
			continue
		}

		if strings.Contains(trimmed, "|") && i+1 < len(lines) && isTableSeparator(lines[i+1]) {
			flushPara()
			header := trimmed
			sep := strings.TrimSpace(lines[i+1])
			i += 2
			var body []string
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" && strings.Contains(lines[i], "|") {
				body = append(body, strings.TrimSpace(lines[i]))
				i++
			}
			i--
			nodes = append(nodes, textNode(buildTable(header, sep, body, r)))
			continue
		}

		if m := footnoteDefRe.FindStringSubmatch(trimmed); m != nil {
			flushPara()
			r.addDef(m[1], m[2])
			continue
		}

		if trimmed == "" {
			flushPara()
			continue
		}

		para = append(para, trimmed)
	}

	flushPara()
	if r.hasDefs() {
		nodes = append(nodes, textNode(r.footnotesSection()))
	}
	return nodes, nil
}

func textNode(content string) ir.TemplateNode {
	return ir.TemplateNode{Type: ir.NodeText, Content: content}
}

func isHR(s string) bool {
	if !strings.HasPrefix(s, "-") {
		return false
	}
	for _, r := range s {
		if r != '-' {
			return false
		}
	}
	return len(s) >= 3
}

func isHTMLNameStart(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
