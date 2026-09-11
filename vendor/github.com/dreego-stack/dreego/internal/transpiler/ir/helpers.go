package ir

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

func GoLiteral(s string) string {
	if strings.Contains(s, "`") {
		return strconv.Quote(s)
	}
	return "`" + s + "`"
}

func ToPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	var result strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		result.WriteByte(byte(unicode.ToUpper(rune(p[0]))))
		if len(p) > 1 {
			result.WriteString(strings.ToLower(p[1:]))
		}
	}
	return result.String()
}

func SourceRef(src string, pos int) string {
	loc := SourceLocation(src, pos)
	if src == "" {
		return "?:" + loc
	}
	return fmt.Sprintf("%s:%s", src, loc)
}

func FindExprEnd(s string) int {
	var quote byte
	escaped := false
	for i := 0; i+1 < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if quote == '`' {
				if c == '`' {
					quote = 0
				}
				continue
			}
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			quote = c
		case '}':
			if s[i+1] == '}' {
				return i
			}
		}
	}
	return -1
}

func FindMessageEnd(source string) int {
	depth := 0
	var quote byte
	escaped := false
	for index := 0; index < len(source); index++ {
		char := source[index]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"', '`':
			quote = char
		case '(', '[', '{':
			depth++
		case ')', '}':
			if depth > 0 {
				depth--
			}
		case ']':
			if depth > 0 {
				depth--
			} else if index+1 < len(source) && source[index+1] == ']' {
				return index
			}
		}
	}
	return -1
}

func ParseExpression(raw string) (expr string, filters []string) {
	if !strings.Contains(raw, "|") {
		return raw, nil
	}
	var parts []string
	start := 0
	var quote byte
	escaped := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if quote != 0 {
			if quote == '`' {
				if c == '`' {
					quote = 0
				}
				continue
			}
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'', '`':
			quote = c
		case '|':
			if (i+1 >= len(raw) || raw[i+1] != '|') && (i == 0 || raw[i-1] != '|') {
				parts = append(parts, raw[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, raw[start:])
	expr = strings.TrimSpace(parts[0])
	for _, f := range parts[1:] {
		filters = append(filters, strings.TrimSpace(f))
	}
	return
}
