package ir

import (
	"strconv"
	"strings"
)

type ExprKind int

const (
	ExprKindOther ExprKind = iota
	ExprKindStringLiteral
	ExprKindIntLiteral
)

func ClassifyExpression(expr string) ExprKind {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ExprKindOther
	}
	if isGoStringLiteral(expr) {
		return ExprKindStringLiteral
	}
	if isGoIntLiteral(expr) {
		return ExprKindIntLiteral
	}
	return ExprKindOther
}

func isGoStringLiteral(expr string) bool {
	if len(expr) < 2 {
		return false
	}
	quote := expr[0]
	if quote != '"' && quote != '`' && quote != '\'' {
		return false
	}
	if expr[len(expr)-1] != quote {
		return false
	}
	if quote == '`' {
		return !strings.Contains(expr[1:len(expr)-1], "`")
	}
	_, err := strconv.Unquote(expr)
	return err == nil
}

func isGoIntLiteral(expr string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}
	base := 10
	if strings.HasPrefix(expr, "0x") || strings.HasPrefix(expr, "0X") {
		base = 16
		expr = expr[2:]
	} else if strings.HasPrefix(expr, "0b") || strings.HasPrefix(expr, "0B") {
		base = 2
		expr = expr[2:]
	} else if strings.HasPrefix(expr, "0o") || strings.HasPrefix(expr, "0O") {
		base = 8
		expr = expr[2:]
	}
	if expr == "" {
		return false
	}
	if _, err := strconv.ParseInt(expr, base, 64); err == nil {
		return true
	}
	_, err := strconv.ParseUint(expr, base, 64)
	return err == nil
}

func KindName(k ExprKind) string {
	switch k {
	case ExprKindStringLiteral:
		return "string"
	case ExprKindIntLiteral:
		return "int"
	default:
		return "other"
	}
}
