package sse

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	dreego "github.com/dreego-stack/dreego/core"
)

func newServer(t *testing.T, opts Options) *httptest.Server {
	t.Helper()
	app := dreego.New()
	if err := Register(app, opts); err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(app.Handler())
}

func connect(t *testing.T, srv *httptest.Server, path string) (*http.Response, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+path, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	return resp, cancel
}

func readLine(t *testing.T, r *bufio.Reader, d time.Duration) string {
	t.Helper()
	type res struct {
		s string
		e error
	}
	ch := make(chan res, 1)
	go func() { s, e := r.ReadString('\n'); ch <- res{s, e} }()
	select {
	case x := <-ch:
		if x.e != nil {
			t.Fatalf("read: %v", x.e)
		}
		return x.s
	case <-time.After(d):
		t.Fatal("read timeout")
		return ""
	}
}

func waitFor(t *testing.T, want int, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if BrokerInstance().clientCount() == want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("want %d clients, got %d", want, BrokerInstance().clientCount())
}

func rawHeaders(t *testing.T, srv *httptest.Server, path string) (int, http.Header) {
	t.Helper()
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: localhost\r\nAccept-Encoding: identity\r\n\r\n", path)
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(strings.TrimSpace(statusLine), " ", 3)
	code, _ := strconv.Atoi(parts[1])
	h := http.Header{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) == 2 {
			h.Add(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
		}
	}
	return code, h
}

func TestRegisterDefaultPath(t *testing.T) {
	srv := newServer(t, Options{})
	defer srv.Close()
	code, h := rawHeaders(t, srv, "/sse")
	if code != 200 {
		t.Fatalf("status %d, want 200", code)
	}
	if !strings.Contains(h.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("Content-Type %q, want text/event-stream", h.Get("Content-Type"))
	}
	if !strings.Contains(h.Get("Connection"), "keep-alive") {
		t.Fatalf("Connection %q, want keep-alive", h.Get("Connection"))
	}
}

func TestRegisterCustomPath(t *testing.T) {
	srv := newServer(t, Options{Path: "/events"})
	defer srv.Close()
	code, _ := rawHeaders(t, srv, "/events")
	if code != 200 {
		t.Fatalf("status %d, want 200", code)
	}
}

func TestBroadcastReachesClient(t *testing.T) {
	srv := newServer(t, Options{Heartbeat: time.Hour})
	defer srv.Close()
	resp, cancel := connect(t, srv, "/sse")
	defer cancel()
	defer resp.Body.Close()
	waitFor(t, 1, 2*time.Second)
	BrokerInstance().Broadcast("update", "hello")
	r := bufio.NewReader(resp.Body)
	var got strings.Builder
	for i := 0; i < 3; i++ {
		got.WriteString(readLine(t, r, time.Second))
	}
	if !strings.Contains(got.String(), "event: update") || !strings.Contains(got.String(), "data: hello") {
		t.Fatalf("got %q, want event: update + data: hello", got.String())
	}
}

func TestHeartbeatSent(t *testing.T) {
	srv := newServer(t, Options{Heartbeat: 50 * time.Millisecond})
	defer srv.Close()
	resp, cancel := connect(t, srv, "/sse")
	defer cancel()
	defer resp.Body.Close()
	r := bufio.NewReader(resp.Body)
	line := readLine(t, r, time.Second)
	if !strings.Contains(line, ": ping") {
		t.Fatalf("got %q, want : ping", line)
	}
}

func TestClientDisconnectCleanup(t *testing.T) {
	srv := newServer(t, Options{})
	defer srv.Close()
	resp, cancel := connect(t, srv, "/sse")
	waitFor(t, 1, 2*time.Second)
	resp.Body.Close()
	cancel()
	waitFor(t, 0, 2*time.Second)
}

func TestRegisterAfterBuild(t *testing.T) {
	app := dreego.New()
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}
	if err := Register(app, Options{}); !errors.Is(err, dreego.ErrAppBuilt) {
		t.Fatalf("err %v, want ErrAppBuilt", err)
	}
}

type noFlush struct {
	h http.Header
	s int
}

func (w *noFlush) Header() http.Header {
	if w.h == nil {
		w.h = http.Header{}
	}
	return w.h
}
func (w *noFlush) Write(b []byte) (int, error) { return len(b), nil }
func (w *noFlush) WriteHeader(s int)            { w.s = s }

func TestSSENotCompressedWhenGzipAccepted(t *testing.T) {
	srv := newServer(t, Options{Heartbeat: 50 * time.Millisecond})
	defer srv.Close()
	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, "GET /sse HTTP/1.1\r\nHost: localhost\r\nAccept-Encoding: gzip\r\n\r\n")
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(strings.TrimSpace(statusLine), " ", 3)
	code, _ := strconv.Atoi(parts[1])
	if code != 200 {
		t.Fatalf("status %d, want 200", code)
	}
	h := http.Header{}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) == 2 {
			h.Add(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
		}
	}
	if got := h.Get("Content-Encoding"); got != "identity" {
		t.Fatalf("Content-Encoding %q, want identity", got)
	}
	if got := h.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type %q, want text/event-stream", got)
	}
	line, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	line = strings.TrimSpace(line)
	if strings.Contains(line, ": ping") {
		return
	}
	if size, err := strconv.Atoi(line); err == nil {
		chunk := make([]byte, size)
		if _, err := io.ReadFull(br, chunk); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(chunk), ": ping") {
			t.Fatalf("chunk %q, want plaintext : ping (stream must not be gzipped)", chunk)
		}
		return
	}
	t.Fatalf("got %q, want plaintext : ping (stream must not be gzipped)", line)
}

func TestNoFlusherReturns500(t *testing.T) {
	h := sseHandler(BrokerInstance(), time.Second)
	w := &noFlush{}
	r := httptest.NewRequest(http.MethodGet, "/sse", nil)
	h.ServeHTTP(w, r)
	if w.s != 500 {
		t.Fatalf("status %d, want 500", w.s)
	}
}