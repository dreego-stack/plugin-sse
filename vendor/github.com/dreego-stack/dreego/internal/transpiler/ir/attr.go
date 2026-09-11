package ir

import "strings"

func AttrNameAt(tag string, i int) string {
	if i < 0 || i >= len(tag) {
		return ""
	}
	pos := 0
	for pos < len(tag) {
		pos = SkipAttrSpace(tag, pos)
		if pos >= len(tag) {
			return ""
		}
		nameStart := pos
		for pos < len(tag) && tag[pos] != '=' && !IsAttrSpace(tag[pos]) {
			pos++
		}
		name := tag[nameStart:pos]
		pos = SkipAttrSpace(tag, pos)
		if pos >= len(tag) || tag[pos] != '=' {
			continue
		}
		pos = SkipAttrSpace(tag, pos+1)
		if pos >= len(tag) {
			return ""
		}
		if tag[pos] == '"' || tag[pos] == '\'' {
			quote := tag[pos]
			pos++
			valStart := pos
			for pos < len(tag) && tag[pos] != quote {
				pos++
			}
			if pos >= len(tag) {
				return ""
			}
			if i >= valStart && i < pos {
				return name
			}
			pos++
			continue
		}
		valStart := pos
		for pos < len(tag) && !IsAttrSpace(tag[pos]) && tag[pos] != '>' {
			pos++
		}
		if i >= valStart && i < pos {
			return name
		}
	}
	return ""
}

func IsURLAttr(name string) bool {
	switch strings.ToLower(name) {
	case "href", "src", "action", "srcset", "poster", "cite", "formaction", "data", "background", "longdesc", "usemap", "xlink:href":
		return true
	}
	return false
}

func IsScriptAttr(name string) bool {
	n := strings.ToLower(name)
	if strings.HasPrefix(n, "on") && len(n) > 2 && n[2] >= 'a' && n[2] <= 'z' {
		return true
	}
	switch {
	case strings.HasPrefix(n, "x-on:"), strings.HasPrefix(n, "x-on."), strings.HasPrefix(n, "@"):
		return true
	case strings.HasPrefix(n, "hx-on:"):
		return true
	}
	switch n {
	case "x-data", "x-init", "x-effect", "x-html", "x-show", "x-model", "x-text", "x-transition":
		return true
	}
	return false
}

func AttrContext(name string) string {
	n := strings.ToLower(name)
	if n == "srcdoc" {
		return "SafeSrcdoc"
	}
	if IsScriptAttr(name) {
		return "SafeScript"
	}
	if IsURLAttr(name) {
		return "SafeURL"
	}
	if n == "style" || n == "x-bind:style" || n == ":style" {
		return "SafeStyle"
	}
	return "SafeAttr"
}

func AttrValue(tag string, attr string) string {
	name := strings.ToLower(attr)
	lower := strings.ToLower(tag)
	i := 0
	for {
		idx := strings.Index(lower[i:], name)
		if idx < 0 {
			return ""
		}
		start := i + idx
		if start > 0 && !IsAttrSpace(lower[start-1]) {
			i = start + len(name)
			continue
		}
		pos := SkipAttrSpace(lower, start+len(name))
		if pos >= len(tag) || tag[pos] != '=' {
			i = pos
			continue
		}
		pos = SkipAttrSpace(tag, pos+1)
		if pos >= len(tag) {
			return ""
		}
		if tag[pos] == '"' || tag[pos] == '\'' {
			quote := tag[pos]
			pos++
			end := strings.IndexByte(tag[pos:], quote)
			if end < 0 {
				return ""
			}
			return tag[pos : pos+end]
		}
		end := pos
		for end < len(tag) && !IsAttrSpace(tag[end]) && tag[end] != '>' {
			end++
		}
		return tag[pos:end]
	}
}

func IsAttrSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func SkipAttrSpace(s string, i int) int {
	for i < len(s) && IsAttrSpace(s[i]) {
		i++
	}
	return i
}

func TagEnd(input string) int {
	var quote byte
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '"', '\'':
			if quote == 0 {
				quote = input[i]
			} else if quote == input[i] {
				quote = 0
			}
		case '>':
			if quote == 0 {
				return i
			}
		}
	}
	return -1
}
