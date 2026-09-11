package md

import (
	"regexp"
	"strings"

	"github.com/dreego-stack/dreego/internal/transpiler/ir"
)

var tableSepRe = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)

func isTableSeparator(s string) bool {
	return tableSepRe.MatchString(s)
}

func buildTable(header, sep string, body []string, r *mdRenderer) string {
	headers := splitCells(header)
	sepCells := splitCells(sep)
	n := len(headers)
	var b strings.Builder
	b.WriteString("<table><thead><tr>")
	for i, h := range headers {
		b.WriteString(cellTag("th", h, alignAt(sepCells, i), r))
	}
	b.WriteString("</tr></thead><tbody>")
	for _, row := range body {
		cells := splitCells(row)
		b.WriteString("<tr>")
		for i := range n {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			b.WriteString(cellTag("td", c, alignAt(sepCells, i), r))
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

func buildTableNodes(header, sep []mdSegment, body [][]mdSegment, r *mdRenderer) []ir.TemplateNode {
	headers := splitCellSegments(header)
	sepCells := splitCellSegments(sep)
	n := len(headers)
	var out []ir.TemplateNode
	out = append(out, textNode("<table><thead><tr>"))
	for i, h := range headers {
		out = append(out, cellTagNodes("th", h, alignAtSegments(sepCells, i), r)...)
	}
	out = append(out, textNode("</tr></thead><tbody>"))
	for _, row := range body {
		cells := splitCellSegments(row)
		out = append(out, textNode("<tr>"))
		for i := range n {
			var c []mdSegment
			if i < len(cells) {
				c = cells[i]
			}
			out = append(out, cellTagNodes("td", c, alignAtSegments(sepCells, i), r)...)
		}
		out = append(out, textNode("</tr>"))
	}
	out = append(out, textNode("</tbody></table>"))
	return mergeText(out)
}

func emitTableLines(lines [][]mdSegment, start int, consumed *int, r *mdRenderer) []ir.TemplateNode {
	header := lines[start]
	sep := lines[start+1]
	idx := start + 2
	var body [][]mdSegment
	for idx < len(lines) {
		bl := lineRaw(lines[idx])
		if strings.TrimSpace(bl) == "" || !strings.Contains(bl, "|") {
			break
		}
		body = append(body, lines[idx])
		idx++
	}
	*consumed = idx - start - 1
	return buildTableNodes(header, sep, body, r)
}

func splitCells(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func splitCellSegments(line []mdSegment) [][]mdSegment {
	var cells [][]mdSegment
	var cur []mdSegment
	flush := func() {
		cells = append(cells, trimCellSegments(cur))
		cur = nil
	}
	for _, s := range line {
		if s.isExpr {
			cur = append(cur, s)
			continue
		}
		parts := strings.Split(s.text, "|")
		for j, p := range parts {
			if j > 0 {
				flush()
			}
			if p != "" {
				cur = append(cur, mdSegment{isExpr: false, text: p})
			}
		}
	}
	flush()
	raw := lineRaw(line)
	if strings.HasPrefix(strings.TrimSpace(raw), "|") && len(cells) > 0 && cellEmpty(cells[0]) {
		cells = cells[1:]
	}
	if strings.HasSuffix(strings.TrimSpace(raw), "|") && len(cells) > 0 && cellEmpty(cells[len(cells)-1]) {
		cells = cells[:len(cells)-1]
	}
	return cells
}

func trimCellSegments(c []mdSegment) []mdSegment {
	for len(c) > 0 && !c[0].isExpr {
		t := strings.TrimLeft(c[0].text, " \t")
		if t == "" {
			c = c[1:]
			continue
		}
		c[0].text = t
		break
	}
	for len(c) > 0 && !c[len(c)-1].isExpr {
		last := len(c) - 1
		t := strings.TrimRight(c[last].text, " \t")
		if t == "" {
			c = c[:last]
			continue
		}
		c[last].text = t
		break
	}
	return c
}

func cellEmpty(c []mdSegment) bool {
	for _, s := range c {
		if s.isExpr {
			return false
		}
		if strings.TrimSpace(s.text) != "" {
			return false
		}
	}
	return true
}

func cellAlign(s string) string {
	s = strings.TrimSpace(s)
	left := strings.HasPrefix(s, ":")
	right := strings.HasSuffix(s, ":")
	switch {
	case left && right:
		return "center"
	case right:
		return "right"
	case left:
		return "left"
	}
	return ""
}

func alignAt(sepCells []string, i int) string {
	if i < len(sepCells) {
		return cellAlign(sepCells[i])
	}
	return ""
}

func alignAtSegments(sepCells [][]mdSegment, i int) string {
	if i < len(sepCells) {
		return cellAlign(lineRaw(sepCells[i]))
	}
	return ""
}

func cellTag(tag, content, align string, r *mdRenderer) string {
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(tag)
	if align != "" {
		b.WriteString(` style="text-align: `)
		b.WriteString(align)
		b.WriteString(`"`)
	}
	b.WriteString(">")
	b.WriteString(r.renderInline(content))
	b.WriteString("</")
	b.WriteString(tag)
	b.WriteString(">")
	return b.String()
}

func cellTagNodes(tag string, cell []mdSegment, align string, r *mdRenderer) []ir.TemplateNode {
	var out []ir.TemplateNode
	open := "<" + tag
	if align != "" {
		open += ` style="text-align: ` + align + `"`
	}
	open += ">"
	out = append(out, textNode(open))
	out = append(out, lineSegments(cell, false, r)...)
	out = append(out, textNode("</"+tag+">"))
	return out
}
