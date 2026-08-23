package context

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"
)

const maxBindBodySize = 1 << 20

func (c *SSRContext) JSON(status int, data any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		http.Error(c.W, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.W.WriteHeader(status)
	c.W.Write(buf.Bytes())
}

func (c *SSRContext) XML(status int, data any) {
	var buf bytes.Buffer
	if err := xml.NewEncoder(&buf).Encode(data); err != nil {
		http.Error(c.W, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	c.W.Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.W.WriteHeader(status)
	c.W.Write(buf.Bytes())
}

func (c *SSRContext) Bind(target any) error {
	body := http.MaxBytesReader(c.W, c.R.Body, maxBindBodySize)
	return json.NewDecoder(body).Decode(target)
}

func (c *SSRContext) Write(status int, contentType string, body []byte) {
	c.W.Header().Set("Content-Type", contentType+"; charset=utf-8")
	c.W.WriteHeader(status)
	c.W.Write(body)
}

func (c *SSRContext) Wants(mime string) bool {
	accept := c.R.Header.Get("Accept")
	if accept == "" {
		return false
	}
	return stringsContainsMime(accept, mime)
}

func stringsContainsMime(accept, mime string) bool {
	for _, part := range strings.Split(accept, ",") {
		t := strings.TrimSpace(part)
		if idx := strings.IndexByte(t, ';'); idx >= 0 {
			t = t[:idx]
		}
		if t == mime {
			return true
		}
	}
	return false
}
