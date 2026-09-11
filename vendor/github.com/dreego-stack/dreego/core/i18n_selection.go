package core

import (
	"net/http"
	"net/url"
	"strings"
)

type LocaleSelectionOptions struct {
	LocaleField  string
	ReturnField  string
	FallbackPath string
}

func LocaleSelectionHandler(options LocaleSelectionOptions) http.HandlerFunc {
	localeField := options.LocaleField
	if localeField == "" {
		localeField = "locale"
	}
	returnField := options.ReturnField
	if returnField == "" {
		returnField = "return"
	}
	fallback := safeReturnPath(options.FallbackPath, "", "/")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := SetLocale(w, r, r.Form.Get(localeField)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		destination := safeReturnPath(r.Form.Get(returnField), r.URL.Path, fallback)
		http.Redirect(w, r, destination, http.StatusSeeOther)
	}
}

func safeReturnPath(candidate, selectionPath, fallback string) string {
	if candidate == "" {
		return fallback
	}
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(parsed.Path, "\\") {
		return fallback
	}
	if parsed.Path == selectionPath {
		return fallback
	}
	return parsed.RequestURI()
}
