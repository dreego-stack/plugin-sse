# plugin-sse

Server-Sent Events (SSE) plugin for the
[Dreego](https://github.com/dreego-stack/dreego) framework. Adds real-time
event streaming with the Go standard library only — no external dependencies.

## Quick Start

```go
package main

import (
    "log"
    "github.com/dreego-stack/dreego/core"
    sse "github.com/dreego-stack/plugin-sse"
)

func main() {
    app := dreego.New()
    if err := sse.Register(app, sse.Options{Path: "/sse"}); err != nil {
        log.Fatal(err)
    }
    // Broadcast events from anywhere:
    sse.BrokerInstance().Broadcast("update", "hello world")
    app.Listen(":8080")
}
```

Frontend:

```javascript
const es = new EventSource("/sse");
es.addEventListener("update", (e) => {
    console.log("got:", e.data);
});
```

## Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Path` | `string` | `"/sse"` | URL path for the SSE endpoint |
| `Heartbeat` | `time.Duration` | `15s` | Interval for keep-alive ping comments |

## Broker

`sse.BrokerInstance()` returns a singleton broker. Call `Broadcast(name, data)`
to send named events to all connected clients. The broker is thread-safe.

## Getting Started (Development)

```sh
make init    # download and vendor dependencies
make test    # run tests
make run     # run the demo app (broadcasts time every 2s)
```

## License

MPL-2.0, same as Dreego.