package server

import (
	"net/http"
	"strings"
)

type route struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

type redirectRule struct {
	from   string
	to     string
	status int
}

type rewriteRule struct {
	from string
	to   string
}

func matchRewrite(rw rewriteRule, path string) bool {
	return rewriteMatches(rw.from, path)
}

func matchRedirect(rd redirectRule, path string) (string, bool) {
	if !rewriteMatches(rd.from, path) {
		return "", false
	}
	if !strings.HasSuffix(rd.from, "/*") {
		return rd.to, true
	}
	return redirectTarget(rd.from, rd.to, path), true
}

func applyRewrite(rw rewriteRule, path string) (string, bool) {
	if !rewriteMatches(rw.from, path) {
		return "", false
	}
	if !strings.HasSuffix(rw.from, "/*") {
		return rw.to, true
	}
	return redirectTarget(rw.from, rw.to, path), true
}

// rewriteMatches reports whether path is covered by from.
//
// Exact rule ("/api"): matches only "/api".
// Wildcard rule ("/api/*"): matches "/api" and any path whose next
// segment boundary aligns with the prefix, e.g. "/api/x" but not "/apiary".
func rewriteMatches(from, path string) bool {
	if !strings.HasSuffix(from, "/*") {
		return path == from
	}
	prefix := strings.TrimSuffix(from, "/*")
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}

func redirectTarget(from, to, path string) string {
	fromPrefix := strings.TrimSuffix(from, "/*")
	toPrefix := strings.TrimSuffix(to, "/*")
	suffix := strings.TrimPrefix(path, fromPrefix)
	if suffix == "" {
		return toPrefix
	}
	return toPrefix + suffix
}
