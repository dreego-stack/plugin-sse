package sse

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	dreego "github.com/dreego-stack/dreego/core"
)

type Options struct {
	Path      string
	Heartbeat time.Duration
}

type Broker struct {
	mu      sync.RWMutex
	clients map[chan event]struct{}
}

type event struct {
	name string
	data string
}

func newBroker() *Broker {
	return &Broker{clients: map[chan event]struct{}{}}
}

func (b *Broker) subscribe() chan event {
	ch := make(chan event, 16)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broker) unsubscribe(ch chan event) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *Broker) Broadcast(name, data string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- event{name: name, data: data}:
		default:
		}
	}
}

func (b *Broker) clientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

var (
	globalBroker *Broker
	once         sync.Once
)

func BrokerInstance() *Broker {
	once.Do(func() {
		globalBroker = newBroker()
	})
	return globalBroker
}

func Register(app *dreego.App, options Options) error {
	if options.Path == "" {
		options.Path = "/sse"
	}
	if options.Heartbeat == 0 {
		options.Heartbeat = 15 * time.Second
	}
	broker := BrokerInstance()
	handler := sseHandler(broker, options.Heartbeat)
	return app.Register(http.MethodGet, options.Path, handler)
}

func sseHandler(broker *Broker, heartbeat time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.Header().Set("Content-Encoding", "identity")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		ch := broker.subscribe()
		defer broker.unsubscribe(ch)

		ctx := r.Context()
		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprintf(w, ": ping\n\n")
				flusher.Flush()
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if ev.name != "" {
					fmt.Fprintf(w, "event: %s\n", ev.name)
				}
				fmt.Fprintf(w, "data: %s\n\n", ev.data)
				flusher.Flush()
			}
		}
	}
}