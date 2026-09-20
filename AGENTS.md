# Repository guidance

gotify-bark grab Gotify WebSocket message, push through Shoutrrr, send to Bark or other notify service.

- Package manager: Go modules (`go mod`); use Go version writ in `go.mod`.
- Build from repo root: `go build ./cmd/gotify-bark`.

Read guide before do task:

- [Development](docs/agents/development.md): Go change, CLI config.
- [Git workflow](docs/agents/git.md): Conventional Commit words when commit change.
- [Verification](docs/agents/verification.md): check for code or docs change, deploy-task side effect.
- [Credential safety](docs/agents/credentials.md): config, log, test, example with secret.
- [Runtime overview](docs/README.md): message flow, local run, container health.
- [User configuration](README.md): env variable, notify URL.