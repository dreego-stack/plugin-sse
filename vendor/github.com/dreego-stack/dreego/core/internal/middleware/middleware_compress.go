package middleware

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

func Compress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gzipOK := acceptsGzip(r.Header.Get("Accept-Encoding"))
			if !gzipOK {
				defer appendVary(w, "Accept-Encoding")
				next.ServeHTTP(w, r)
				return
			}
			gw := &gzipResponseWriter{ResponseWriter: w, method: r.Method}
			gw.gz = gzip.NewWriter(&gw.buf)
			defer func() {
				if r := recover(); r != nil {
					gw.discard()
					appendVary(w, "Accept-Encoding")
					panic(r)
				}
				appendVary(w, "Accept-Encoding")
				gw.commit()
			}()
			next.ServeHTTP(gw, r)
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	method    string
	mu        sync.Mutex
	gz        *gzip.Writer
	buf       gzipBuffer
	status    int
	written   bool
	bypass    bool
	committed bool
}

type gzipBuffer struct {
	buf []byte
}

func (b *gzipBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (w *gzipResponseWriter) decide() {
	if w.written {
		return
	}
	w.written = true
	if w.method == http.MethodHead {
		w.bypass = true
		return
	}
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	if status == http.StatusNoContent || status == http.StatusNotModified {
		w.bypass = true
		return
	}
	if w.Header().Get("Content-Encoding") != "" {
		w.bypass = true
		return
	}
	ct := w.Header().Get("Content-Type")
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.ToLower(strings.TrimSpace(ct))
	if isAlreadyCompressedContentType(ct) {
		w.bypass = true
		return
	}
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if code >= 100 && code < 200 {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	if w.written {
		return
	}
	w.status = code
	w.decide()
	if w.bypass {
		w.ResponseWriter.WriteHeader(code)
		return
	}
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.written {
		w.decide()
	}
	if w.bypass {
		return w.ResponseWriter.Write(b)
	}
	return w.gz.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.written {
		w.decide()
	}
	if w.bypass {
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}
		return
	}
	if !w.committed {
		w.gz.Close()
		w.commitHeaders()
	}
	w.gz.Flush()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *gzipResponseWriter) commitHeaders() {
	if w.committed {
		return
	}
	w.committed = true
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Del("Content-Length")
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(w.status)
	if n := len(w.buf.buf); n > 0 {
		_, _ = w.ResponseWriter.Write(w.buf.buf)
		w.buf.buf = w.buf.buf[:0]
	}
	w.gz = gzip.NewWriter(w.ResponseWriter)
}

func (w *gzipResponseWriter) discard() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bypass = true
	w.gz.Close()
}

func (w *gzipResponseWriter) commit() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.written {
		w.decide()
	}
	if w.bypass {
		return
	}
	if w.committed {
		w.gz.Close()
		return
	}
	w.gz.Close()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Del("Content-Length")
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(w.buf.buf)
}

func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		w.mu.Lock()
		w.bypass = true
		w.mu.Unlock()
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (w *gzipResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (w *gzipResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{w}, src)
}

func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func isAlreadyCompressedContentType(ct string) bool {
	if strings.HasPrefix(ct, "image/") ||
		strings.HasPrefix(ct, "video/") ||
		strings.HasPrefix(ct, "audio/") {
		return true
	}
	switch ct {
	case "application/zip",
		"application/gzip",
		"application/x-gzip",
		"application/x-bzip2",
		"application/x-7z-compressed":
		return true
	}
	return false
}

func acceptsGzip(header string) bool {
	if header == "" {
		return false
	}
	gzipQ := -1.0
	wildcardQ := -1.0
	for _, part := range strings.Split(header, ",") {
		p := strings.TrimSpace(part)
		name := p
		q := 1.0
		if idx := strings.Index(p, ";"); idx >= 0 {
			name = strings.TrimSpace(p[:idx])
			if rest := strings.TrimSpace(p[idx+1:]); strings.HasPrefix(rest, "q=") {
				if v, err := strconv.ParseFloat(rest[2:], 64); err == nil {
					q = v
				}
			}
		}
		switch strings.ToLower(name) {
		case "gzip":
			gzipQ = q
		case "*":
			wildcardQ = q
		}
	}
	if gzipQ > 0 {
		return true
	}
	return gzipQ == -1 && wildcardQ > 0
}

func appendVary(w http.ResponseWriter, value string) {
	existing := w.Header().Values("Vary")
	for _, v := range existing {
		if strings.EqualFold(v, value) {
			return
		}
	}
	w.Header().Add("Vary", value)
}
