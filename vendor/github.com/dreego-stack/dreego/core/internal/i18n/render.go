package i18n

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/currency"
	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func renderValue(value Value, formats map[string]ArgumentFormat, arguments map[string]any, tag language.Tag) (string, error) {
	if value.Text != nil {
		return interpolate(*value.Text, formats, arguments, tag)
	}
	if value.Selector == nil {
		return "", fmt.Errorf("message value is empty")
	}
	raw, exists := arguments[value.Selector.Argument]
	if !exists {
		return "", fmt.Errorf("missing argument %q", value.Selector.Argument)
	}
	caseName, err := selectCase(value.Selector, raw, tag)
	if err != nil {
		return "", err
	}
	selected, exists := value.Selector.Cases[caseName]
	if !exists {
		selected = value.Selector.Cases["other"]
	}
	return renderValue(selected, formats, arguments, tag)
}

func selectCase(selector *Selector, value any, tag language.Tag) (string, error) {
	if selector.Kind == "select" {
		return fmt.Sprint(value), nil
	}
	i, v, w, f, t, exact, ok := pluralOperands(value)
	if !ok {
		return "", fmt.Errorf("plural argument %q must be numeric", selector.Argument)
	}
	if _, exists := selector.Cases["="+exact]; exists {
		return "=" + exact, nil
	}
	rules := plural.Cardinal
	if selector.Kind == "ordinal" {
		rules = plural.Ordinal
	}
	form := rules.MatchPlural(tag, i, v, w, f, t)
	switch form {
	case plural.Zero:
		return "zero", nil
	case plural.One:
		return "one", nil
	case plural.Two:
		return "two", nil
	case plural.Few:
		return "few", nil
	case plural.Many:
		return "many", nil
	default:
		return "other", nil
	}
}

func interpolate(text string, formats map[string]ArgumentFormat, arguments map[string]any, tag language.Tag) (string, error) {
	var output strings.Builder
	for len(text) > 0 {
		if strings.HasPrefix(text, "{{") {
			output.WriteByte('{')
			text = text[2:]
			continue
		}
		if strings.HasPrefix(text, "}}") {
			output.WriteByte('}')
			text = text[2:]
			continue
		}
		if text[0] == '}' {
			return "", fmt.Errorf("unexpected closing placeholder brace")
		}
		if text[0] != '{' {
			end := strings.IndexAny(text, "{}")
			if end < 0 {
				output.WriteString(text)
				break
			}
			output.WriteString(text[:end])
			text = text[end:]
			continue
		}
		end := strings.IndexByte(text[1:], '}')
		if end < 0 {
			return "", fmt.Errorf("unclosed placeholder")
		}
		name := text[1 : 1+end]
		value, exists := arguments[name]
		if !exists {
			return "", fmt.Errorf("missing argument %q", name)
		}
		formatted, err := formatValue(value, formats[name], arguments, tag)
		if err != nil {
			return "", fmt.Errorf("argument %q: %w", name, err)
		}
		output.WriteString(formatted)
		text = text[end+2:]
	}
	return output.String(), nil
}

func formatValue(value any, format ArgumentFormat, arguments map[string]any, tag language.Tag) (string, error) {
	printer := message.NewPrinter(tag)
	switch format.Format {
	case "number", "integer", "":
		return printer.Sprint(value), nil
	case "percent":
		number, ok := decimal(value)
		if !ok {
			return "", fmt.Errorf("percent must be numeric")
		}
		return printer.Sprintf("%.2f%%", number*100), nil
	case "currency":
		code, ok := arguments[format.CurrencyArgument].(string)
		if !ok {
			return "", fmt.Errorf("currency argument %q must be an ISO 4217 string", format.CurrencyArgument)
		}
		unit, err := currency.ParseISO(code)
		if err != nil {
			return "", err
		}
		return printer.Sprint(currency.Symbol(unit.Amount(value))), nil
	case "date", "time", "datetime":
		return formatTime(value, format, arguments)
	default:
		return fmt.Sprint(value), nil
	}
}

func formatTime(value any, format ArgumentFormat, arguments map[string]any) (string, error) {
	instant, ok := value.(time.Time)
	if !ok {
		return "", fmt.Errorf("%s must be time.Time", format.Format)
	}
	if format.TimeZoneArgument != "" {
		zone, ok := arguments[format.TimeZoneArgument].(string)
		if !ok {
			return "", fmt.Errorf("time zone argument %q must be an IANA time zone string", format.TimeZoneArgument)
		}
		location, err := time.LoadLocation(zone)
		if err != nil {
			return "", err
		}
		instant = instant.In(location)
	}
	layout := format.Layout
	if layout == "" {
		layout = map[string]string{"date": "2006-01-02", "time": "15:04:05", "datetime": "2006-01-02 15:04:05 MST"}[format.Format]
	}
	return instant.Format(layout), nil
}

func integer(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case int8:
		return int(number), true
	case int16:
		return int(number), true
	case int64:
		return int(number), true
	case int32:
		return int(number), true
	case uint:
		return int(number), true
	case uint8:
		return int(number), true
	case uint16:
		return int(number), true
	case uint32:
		return int(number), true
	case uint64:
		return int(number), true
	case uintptr:
		return int(number), true
	default:
		return 0, false
	}
}

func decimal(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	default:
		integer, ok := integer(value)
		return float64(integer), ok
	}
}

func pluralOperands(value any) (int, int, int, int, int, string, bool) {
	number, ok := decimal(value)
	if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, 0, 0, 0, 0, "", false
	}
	exact := strconv.FormatFloat(number, 'f', -1, 64)
	number = math.Abs(number)
	operand := strconv.FormatFloat(number, 'f', -1, 64)
	integerPart, fraction, _ := strings.Cut(operand, ".")
	integerValue, err := strconv.ParseUint(integerPart, 10, 64)
	if err != nil {
		return 0, 0, 0, 0, 0, "", false
	}
	visible := len(fraction)
	trimmed := strings.TrimRight(fraction, "0")
	withoutZeros := len(trimmed)
	fractionValue, _ := strconv.Atoi(fraction)
	trimmedValue, _ := strconv.Atoi(trimmed)
	return int(integerValue % 10_000_000), visible, withoutZeros, fractionValue % 10_000_000, trimmedValue % 10_000_000, exact, true
}

func MatchSupported(matcher language.Matcher, preferred []string) string {
	tag, _ := language.MatchStrings(matcher, preferred...)
	base, _ := tag.Base()
	return base.String()
}
