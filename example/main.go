package main

import (
	"fmt"
	"log"
	"os"
	"time"

	dreego "github.com/dreego-stack/dreego/core"
	"github.com/dreego-stack/dreego/core/ssr"
	sse "github.com/dreego-stack/plugin-sse"
)

func main() {
	app := dreego.New()
	if err := sse.Register(app, sse.Options{Path: "/sse"}); err != nil {
		log.Fatal(err)
	}
	go broadcastLoop()
	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	if err := ssr.Listen(app, addr); err != nil {
		log.Fatal(err)
	}
}

func broadcastLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		sse.BrokerInstance().Broadcast("time", fmt.Sprintf("server time: %s", time.Now().Format(time.RFC3339)))
	}
}
