package ir

import "fmt"

func PosToLineCol(src string, pos int) (line, col int) {
	line = 1
	col = 1
	for i := 0; i < pos && i < len(src); i++ {
		if src[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return
}

func SourceLocation(src string, pos int) string {
	line, col := PosToLineCol(src, pos)
	if line == 0 && col == 0 {
		return "?:?"
	}
	return fmt.Sprintf("%d:%d", line, col)
}

func SetNodeSource(nodes []TemplateNode, src string, posOffset int) {
	for i := range nodes {
		nodes[i].Source = src
		nodes[i].Pos += posOffset
		SetNodeSource(nodes[i].Children, src, posOffset)
		SetNodeSource(nodes[i].ElseChildren, src, posOffset)
	}
}

func SetSourceText(nodes []TemplateNode, src string) {
	for i := range nodes {
		nodes[i].SourceText = src
		SetSourceText(nodes[i].Children, src)
		SetSourceText(nodes[i].ElseChildren, src)
	}
}
