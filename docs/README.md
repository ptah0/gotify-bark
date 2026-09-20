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

The tests use local HTTP and WebSocket servers to exercise delivery, multiple
destinations, malformed input, configuration errors, cancellation, listener cleanup,
and credential-safe logging. Virtual-time tests check delivery timeouts and late results.
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
   `internal.Config`, configures logging, and calls `internal.Run` with a context
   canceled by `SIGINT` or `SIGTERM`.
2. `Run` validates Gotify and notification configuration, initializes reusable Shoutrrr
   providers, starts the status server, and connects to Gotify with a client token.
   It appends `/stream` to the configured base path and preserves other query parameters.
3. The WebSocket reader passes each message to `sendPush`. It decodes the JSON,
   forwards the body, and supplies the title as Shoutrrr's `title` parameter.
   Unused Gotify fields, including priority and date, are ignored.
4. Invalid JSON and ordinary delivery failures are logged once and processing continues.
   Delivery errors are sanitized to avoid exposing credentials. Destinations are sent
   concurrently; each message completes or times out before the next is forwarded.
5. A delivery exceeding 10 seconds logs a sanitized error and processing continues.
   Delivery uses Shoutrrr's provider API directly, bypassing v0.8.0's leaking router
   timeout wrapper. Buffered results allow late sends to finish. Providers have no
   cancellation API; subsequent messages skip a busy destination until it finishes.
   Failed connections and unexpected disconnects log sanitized errors and reconnect
   every second, keeping the status server running.
6. Cancellation interrupts dialing and forwarding, requests a WebSocket close with a
   one-second write deadline, and closes the connection and status listener. Provider
   calls already in progress can continue until they finish or the process exits.

There is no delivery retry, missed-message replay, or persistent queue.
The Compose configuration sets `restart: unless-stopped` for process restarts.

## Health and containers

`internal/actuator.go` serves `/status` on port 8080 using `health-go` with system
information and a private mux. The `gotify` check fails until a WebSocket connection
is established and after a detected disconnect. The `notifications` check fails when
the latest delivery fails for any destination; it recovers only after a subsequent
delivery succeeds for all destinations. Before the first delivery it passes, and
malformed messages do not change its state. Either failing check returns HTTP 503
with sanitized failure text. Listener errors propagate to the caller.

Checks observe application state; they do not send test notifications, probe providers,
or detect an idle broken connection before the WebSocket reports it. Status reads do
not clear failures. Timeouts and disconnects leave the process running while health checks report failures.

The [Dockerfile](../Dockerfile) checks that endpoint inside the container. The
[Compose file](../deploy/compose.yaml) does not publish port 8080 to the host.
Docker marks failed health checks as unhealthy; Compose's restart policy restarts
exited processes, not containers solely because they are unhealthy.

The Docker builder and `go.mod` both use Go 1.27. Check both files when changing
the required Go version. The image has no application data volume.
