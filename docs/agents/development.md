# Development

Read this guide when changing Go code or CLI configuration.

- Format changed Go files with `gofmt`.
- Keep configuration examples in the [project README](../../README.md) and
  `.env.example` aligned with CLI changes.
- For the existing message flow and source locations, see the
  [runtime overview](../README.md#message-flow).

Entry points:

- `cmd/gotify-bark/main.go`: CLI flags, environment variables, and logging setup.
- `internal/core.go`: configuration validation, Gotify connection, and forwarding.
- `internal/actuator.go`: HTTP `/status` endpoint on port 8080.
- `internal/core_test.go`: forwarding and configuration regression tests.
