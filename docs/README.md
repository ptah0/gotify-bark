# Development and runtime

See the [project README](../README.md) for configuration, notification URL
examples, and migration instructions. Contributor and agent conventions live in
[AGENTS.md](../AGENTS.md).

## Local development

Use a Go toolchain compatible with the version declared in [go.mod](../go.mod).
From the repository root:

```sh
go test ./...
go build ./cmd/gotify-bark
```

The tests use a local HTTP server to exercise Bark delivery, multiple
destinations, malformed input, configuration errors, and credential-safe logging.
They do not require a live Gotify or Bark account.

To run the service, set the environment variables documented in the project
README, then run:

```sh
go run ./cmd/gotify-bark
```

The binary does not load `.env` files. `task dev` loads the root `.env`, downloads
modules, builds the binary, and runs it. Docker Compose uses `deploy/.env`.

## Message flow

1. `cmd/gotify-bark/main.go` reads CLI flags and environment variables into
   `internal.Config`, configures logging, and calls `internal.Run`.
2. `Run` validates notification URLs, creates a reusable Shoutrrr sender, starts
   the status server, and connects to Gotify at `/stream` with a client token.
3. The WebSocket reader passes each message to `sendPush`. It decodes the JSON,
   forwards the body, and supplies the title as Shoutrrr's `title` parameter.
   Gotify priority and date are decoded but not forwarded.
4. Invalid JSON and delivery failures are logged and processing continues.
   Delivery errors are sanitized to avoid exposing credentials.
5. A WebSocket read failure ends the run. An interrupt requests a clean WebSocket
   close and waits up to one second for the reader to finish.

There is no application-level reconnect, delivery retry, or persistent queue.
The Compose configuration sets `restart: unless-stopped` for process restarts.

## Health and containers

`internal/actuator.go` serves `/status` on port 8080 with system information.
It does not register a Gotify or notification-provider check, so a successful
response does not establish end-to-end notification delivery.

The [Dockerfile](../Dockerfile) checks that endpoint inside the container. The
[Compose file](../deploy/compose.yaml) does not publish port 8080 to the host.

The Docker builder currently uses Go 1.25 while `go.mod` declares Go 1.27;
container builds may require an automatic toolchain download. Check both files
when changing the required Go version.
