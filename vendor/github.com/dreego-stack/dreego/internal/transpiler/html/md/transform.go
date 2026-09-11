package md

import (
	"fmt"
	"html"
	"strings"

	"github.com/dreego-stack/dreego/internal/transpiler/ir"
)

type mdSegment struct {
	isExpr    bool
	text      string
	node      ir.TemplateNode
	protected ir.TemplateNode
}

func TransformNodes(nodes []ir.TemplateNode) ([]ir.TemplateNode, error) {
	lines := buildLines(nodes)
	r := newRenderer(ModeTrusted)
	var out []ir.TemplateNode
	var inline []ir.TemplateNode
	var blockOpen, blockClose string
	var inFence bool
	var fenceLang string
	var fenceContent [][]ir.TemplateNode

	flushBlock := func() {
		if len(inline) == 0 {
			return
		}
		if blockOpen != "" {
			if inline[0].Type == ir.NodeText {
				inline[0].Content = blockOpen + inline[0].Content
			} else {
				inline = append([]ir.TemplateNode{textNode(blockOpen)}, inline...)
			}
		}
		if blockClose != "" {
			last := len(inline) - 1
			if inline[last].Type == ir.NodeText {
				inline[last].Content += blockClose
			} else {
				inline = append(inline, textNode(blockClose))
			}
		}
		out = append(out, mergeText(inline)...)
		inline = nil
		blockOpen, blockClose = "", ""
	}

	emitFence := func() {
		var content []ir.TemplateNode
		for i, fl := range fenceContent {
			if i > 0 {
				content = append(content, textNode("\n"))
			}
			for _, n := range fl {
				if n.Type == ir.NodeText {
					n.Content = html.EscapeString(n.Content)
				}
				content = append(content, n)
			}
		}
		open := "<pre><code"
		if fenceLang != "" {
			open += ` class="language-` + html.EscapeString(fenceLang) + `"`
		}
		open += ">"
		nodes := []ir.TemplateNode{textNode(open)}
		nodes = append(nodes, content...)
		nodes = append(nodes, textNode("</code></pre>"))
		out = append(out, mergeText(nodes)...)
	}

	for idx := 0; idx < len(lines); idx++ {
		line := lines[idx]
		if len(line) == 1 && line[0].protected.Type != 0 {
			flushBlock()
			out = append(out, line[0].protected)
			continue
		}
		raw := lineRaw(line)
		trimmed := strings.TrimSpace(raw)

		if inFence {
			if strings.HasPrefix(trimmed, "```") {
				emitFence()
				inFence = false
				fenceContent = nil
			} else {
				fenceContent = append(fenceContent, lineSegments(line, true, r))
			}
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			flushBlock()
			fenceLang = safeFenceLanguage(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
			inFence = true
			fenceContent = nil
			continue
		}

		if !hasText(line) && blockOpen == "" {
			for _, s := range line {
				out = append(out, s.node)
			}
			continue
		}

		switch {
		case trimmed == "":
			flushBlock()
		case isATX(raw):
			flushBlock()
			level := headingLevel(raw)
			content := trimTrailingSpace(stripPrefix(line, markerLen(raw, atxHeading)))
			if before, expr, after, found := splitAtExpr(content); found {
				inline = append(inline, lineSegments(trimTrailingSpace(before), false, r)...)
				blockOpen, blockClose = fmt.Sprintf("<h%d>", level), fmt.Sprintf("</h%d>", level)
				flushBlock()
				out = append(out, lineSegments(expr, false, r)...)
				inline = append(inline, lineSegments(after, false, r)...)
				blockOpen, blockClose = "<p>", "</p>"
			} else {
				inline = append(inline, lineSegments(content, false, r)...)
				blockOpen, blockClose = fmt.Sprintf("<h%d>", level), fmt.Sprintf("</h%d>", level)
				flushBlock()
			}
		case isBlockquote(raw):
			flushBlock()
			content := trimTrailingSpace(trimLeadingSpace(stripPrefix(line, 1)))
			if before, expr, after, found := splitAtExpr(content); found {
				inline = append(inline, lineSegments(trimTrailingSpace(before), false, r)...)
				blockOpen, blockClose = "<blockquote>", "</blockquote>"
				flushBlock()
				out = append(out, lineSegments(expr, false, r)...)
				inline = append(inline, lineSegments(after, false, r)...)
				blockOpen, blockClose = "<p>", "</p>"
			} else {
				inline = append(inline, lineSegments(content, false, r)...)
				blockOpen, blockClose = "<blockquote>", "</blockquote>"
				flushBlock()
			}
		case isHR(trimmed):
			flushBlock()
			out = append(out, textNode("<hr>"))
		case htmlBlockStart.MatchString(trimmed):
			flushBlock()
			var rawNodes []ir.TemplateNode
			for idx < len(lines) && strings.TrimSpace(lineRaw(lines[idx])) != "" {
				if len(rawNodes) > 0 {
					rawNodes = append(rawNodes, textNode("\n"))
				}
				rawNodes = append(rawNodes, lineSegments(lines[idx], true, r)...)
				idx++
			}
			idx--
			out = append(out, mergeText(rawNodes)...)
		case strings.Contains(trimmed, "|") && idx+1 < len(lines) && isTableSeparator(lineRaw(lines[idx+1])):
			flushBlock()
			var consumed int
			out = append(out, emitTableLines(lines, idx, &consumed, r)...)
			idx += consumed
		case footnoteDefRe.MatchString(trimmed):
			flushBlock()
			m := footnoteDefRe.FindStringSubmatch(trimmed)
			r.addDef(m[1], m[2])
		case isListItem(raw):
			flushBlock()
			var consumed int
			out = append(out, emitListLines(lines, idx, &consumed, r)...)
			idx += consumed
		default:
			if blockOpen != "" {
				inline = append(inline, textNode(" "))
			} else {
				flushBlock()
				blockOpen, blockClose = "<p>", "</p>"
			}
			inline = append(inline, lineSegments(trimTrailingSpace(line), false, r)...)
		}
	}
	if inFence {
		emitFence()
	}
	flushBlock()
	if r.hasDefs() {
		out = append(out, textNode(r.footnotesSection()))
	}
	return out, nil
}

func buildLines(nodes []ir.TemplateNode) [][]mdSegment {
	var lines [][]mdSegment
	var cur []mdSegment
	flush := func() {
		if len(cur) > 0 {
			lines = append(lines, cur)
			cur = nil
		}
	}
	for _, n := range nodes {
		switch n.Type {
		case ir.NodeExpression, ir.NodeMessage:
			cur = append(cur, mdSegment{isExpr: true, node: n})
		case ir.NodeText:
			parts := strings.Split(n.Content, "\n")
			for j, part := range parts {
				if j > 0 {
					flush()
				}
				if part == "" {
					if j == 0 {
						flush()
					} else {
						lines = append(lines, []mdSegment{{isExpr: false, text: ""}})
					}
				} else {
					cur = append(cur, mdSegment{isExpr: false, text: part})
				}
			}
		default:
			flush()
			lines = append(lines, []mdSegment{{protected: n}})
		}
	}
	flush()
	return lines
}

func lineRaw(line []mdSegment) string {
	var b strings.Builder
	for _, s := range line {
		if s.isExpr {
			b.WriteString("{{expr}}")
		} else {
			b.WriteString(s.text)
		}
	}
	return b.String()
}

func hasText(line []mdSegment) bool {
	for _, s := range line {
		if !s.isExpr {
			return true
		}
	}
	return false
}

func lineSegments(line []mdSegment, raw bool, r *mdRenderer) []ir.TemplateNode {
	var out []ir.TemplateNode
	for _, s := range line {
		if s.isExpr {
			out = append(out, s.node)
		} else if raw {
			out = append(out, textNode(s.text))
		} else {
			out = append(out, textNode(r.renderInline(s.text)))
		}
	}
	return out
}
