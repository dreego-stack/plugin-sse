package md

import (
	"regexp"
	"strings"

	"github.com/dreego-stack/dreego/internal/transpiler/ir"
)

func mergeText(nodes []ir.TemplateNode) []ir.TemplateNode {
	var out []ir.TemplateNode
	for _, n := range nodes {
		if len(out) > 0 && out[len(out)-1].Type == ir.NodeText && n.Type == ir.NodeText {
			out[len(out)-1].Content += n.Content
		} else {
			out = append(out, n)
		}
	}
	return out
}

func splitAtExpr(line []mdSegment) (before, expr, after []mdSegment, found bool) {
	for i, s := range line {
		if !s.isExpr {
			continue
		}
		if hasSignificantText(line[i+1:]) {
			return line[:i], line[i : i+1], line[i+1:], true
		}
	}
	return nil, nil, nil, false
}

func hasSignificantText(segs []mdSegment) bool {
	for _, s := range segs {
		if s.isExpr {
			return true
		}
		if strings.TrimSpace(s.text) != "" {
			return true
		}
	}
	return false
}

func stripPrefix(line []mdSegment, n int) []mdSegment {
	var out []mdSegment
	remaining := n
	for _, s := range line {
		if remaining <= 0 {
			out = append(out, s)
			continue
		}
		if s.isExpr {
			out = append(out, s)
			continue
		}
		if len(s.text) <= remaining {
			remaining -= len(s.text)
		} else {
			out = append(out, mdSegment{isExpr: false, text: s.text[remaining:]})
			remaining = 0
		}
	}
	return out
}

func trimLeadingSpace(line []mdSegment) []mdSegment {
	if len(line) == 0 || line[0].isExpr {
		return line
	}
	t := strings.TrimLeft(line[0].text, " ")
	if t == "" {
		return line[1:]
	}
	line[0].text = t
	return line
}

func trimTrailingSpace(line []mdSegment) []mdSegment {
	if len(line) == 0 || line[len(line)-1].isExpr {
		return line
	}
	last := len(line) - 1
	t := strings.TrimRight(line[last].text, " \t")
	if t == "" {
		return line[:last]
	}
	line[last].text = t
	return line
}

func markerLen(raw string, re *regexp.Regexp) int {
	m := re.FindStringSubmatch(raw)
	if m == nil {
		return 0
	}
	return len(raw) - len(m[len(m)-1])
}

func isATX(raw string) bool { return atxHeading.MatchString(raw) }

func headingLevel(raw string) int {
	m := atxHeading.FindStringSubmatch(raw)
	return len(m[1])
}

func isBlockquote(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), ">")
}

func isListItem(raw string) bool {
	return ulItem.MatchString(raw) || olItem.MatchString(raw)
}

func isOL(raw string) bool { return olItem.MatchString(raw) }

func listItemRe(raw string) *regexp.Regexp {
	if olItem.MatchString(raw) {
		return olItem
	}
	return ulItem
}
