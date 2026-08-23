package tests

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dreego "github.com/dreego-stack/dreego/core"
	sse "github.com/dreego-stack/plugin-sse"
)

func TestSSEPluginIntegration(t *testing.T) {
	app := dreego.New()
	if err := sse.Register(app, sse.Options{Path: "/sse", Heartbeat: 50 * time.Millisecond}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	server := httptest.NewServer(app.Handler())
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/sse", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "identity")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	sse.BrokerInstance().Broadcast("test", "hello from integration")

	scanner := bufio.NewScanner(resp.Body)
	gotEvent := false
	gotData := false
	deadline := time.After(2 * time.Second)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "event: test" {
			gotEvent = true
		}
		if line == "data: hello from integration" {
			gotData = true
		}
		if gotEvent && gotData {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for SSE event")
		default:
		}
	}
	if !gotEvent {
		t.Error("did not receive event: test")
	}
	if !gotData {
		t.Error("did not receive data: hello from integration")
	}
}

func TestSSEPluginCustomPath(t *testing.T) {
	app := dreego.New()
	if err := sse.Register(app, sse.Options{Path: "/events"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSSEPluginErrAppBuilt(t *testing.T) {
	app := dreego.New()
	_ = app.Handler()
	err := sse.Register(app, sse.Options{})
	if err == nil {
		t.Fatal("expected ErrAppBuilt after build")
	}
}