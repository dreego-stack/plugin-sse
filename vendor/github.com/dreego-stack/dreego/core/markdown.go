package core

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/dreego-stack/dreego/internal/transpiler/html/md"
	"github.com/dreego-stack/dreego/internal/transpiler/ir"
)

var markdownLogger = slog.New(slog.NewJSONHandler(os.Stderr, nil))

var trustedWarnOnce sync.Once

func MarkdownToHTML(src string) (string, error) {
	return markdownToHTML(src, md.ModeSafe)
}

func MarkdownToHTMLTrusted(src string) (string, error) {
	trustedWarnOnce.Do(func() {
		markdownLogger.Warn("MarkdownToHTMLTrusted: raw HTML passthrough enabled — only use with trusted content")
	})
	return markdownToHTML(src, md.ModeTrusted)
}

func markdownToHTML(src string, mode md.Mode) (string, error) {
	nodes, err := md.ToNodes(src, mode)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, n := range nodes {
		if n.Type != ir.NodeText {
			return "", fmt.Errorf("markdown: control flow not supported in runtime markdown")
		}
		b.WriteString(n.Content)
	}
	return b.String(), nil
}
