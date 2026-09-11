package i18n

import (
	"strings"

	"golang.org/x/text/language"
)

func LocalizeDocument(document, locale string) string {
	if locale == "" {
		return document
	}
	tag, err := language.Parse(locale)
	if err != nil {
		return document
	}
	locale = tag.String()
	document = setHTMLAttribute(document, "lang", locale)
	if IsRTL(locale) {
		document = setHTMLAttribute(document, "dir", "rtl")
	}
	return document
}

func IsRTL(locale string) bool {
	tag, err := language.Parse(locale)
	if err != nil {
		return false
	}
	script, _ := tag.Script()
	switch script.String() {
	case "Adlm", "Arab", "Armi", "Avst", "Chrs", "Cprt", "Diak", "Elym",
		"Hebr", "Khar", "Lydi", "Mand", "Mani", "Mend", "Merc", "Mero",
		"Narb", "Nbat", "Nkoo", "Palm", "Phli", "Phlp", "Phnx", "Prti",
		"Rohg", "Samr", "Sogd", "Sogo", "Syrc", "Thaa", "Yezi":
		return true
	default:
		return false
	}
}

func setHTMLAttribute(document, name, value string) string {
	lower := strings.ToLower(document)
	start := strings.Index(lower, "<html")
	if start < 0 {
		return document
	}
	end := strings.IndexByte(document[start:], '>')
	if end < 0 {
		return document
	}
	end += start
	for index := start + len("<html"); index < end; {
		for index < end && isSpace(document[index]) {
			index++
		}
		nameStart := index
		for index < end && !isSpace(document[index]) && document[index] != '=' && document[index] != '>' {
			index++
		}
		attribute := document[nameStart:index]
		for index < end && isSpace(document[index]) {
			index++
		}
		if index >= end || document[index] != '=' {
			continue
		}
		index++
		for index < end && isSpace(document[index]) {
			index++
		}
		valueStart := index
		valueEnd := index
		if index < end && (document[index] == '\'' || document[index] == '"') {
			quote := document[index]
			valueStart++
			valueEnd = valueStart
			for valueEnd < end && document[valueEnd] != quote {
				valueEnd++
			}
			index = valueEnd + 1
		} else {
			for valueEnd < end && !isSpace(document[valueEnd]) && document[valueEnd] != '>' {
				valueEnd++
			}
			index = valueEnd
		}
		if strings.EqualFold(attribute, name) {
			return document[:valueStart] + value + document[valueEnd:]
		}
	}
	return document[:end] + " " + name + "=\"" + value + "\"" + document[end:]
}

func isSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}
