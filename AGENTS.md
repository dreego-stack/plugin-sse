# Agent Instructions for plugin-sse

- Don't create binaries here — only in /tmp or ./tmp

## Language Rule

- **Chat with user:** German
- **Everything in this repository:** English (code, comments, docs, commits, tests)

## What This Is

This is the **Server-Sent Events (SSE) plugin** for the Dreego framework.
It registers an SSE endpoint and manages client connections with the Go
standard library only — no external dependencies.

## How It Works

1. `sse.Register(app, sse.Options{Path: "/sse"})` registers the SSE endpoint
2. Clients connect via `new EventSource("/sse")` in the browser
3. `sse.BrokerInstance().Broadcast("event", "data")` sends events to all clients
4. Heartbeat comments (`: ping`) keep connections alive through proxies
5. Client disconnects are cleaned up via context cancellation

## Plugin Contract

- Exports `Register(app *dreego.App, options Options) error`
- Typed Options (Path, Heartbeat)
- No central Plugin interface
- Must be called before `app.Build()` / `app.Listen()`
- Core never imports a plugin; the plugin imports `github.com/dreego-stack/dreego/core`

## Testing

- `go test ./...` — unit tests with real HTTP responses via httptest
- `go test -race ./...` — race detection

## CI

- `.github/workflows/ci.yml` — `go vet`, `go test -race`, and a compatibility
  job that tests against the latest published dreego tag
- `.github/workflows/release.yml` — validates change file, tests, creates tag
- `.github/dependabot.yml` — auto-updates dreego dependency weekly

## Coding Rules

- Max 300 lines per handwritten file
- No code comments (except where needed for clarity)
- Go 1.22+, standard library only (no external deps)
- One Go package per repository

## Commit Convention

Every change lands via a pull request with one `.changes/*.md` file:

```yaml
---
version: patch
---

- Feat: add X
- Bug: fix Y
```