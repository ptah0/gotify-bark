# Repository guidance

gotify-bark forwards Gotify WebSocket messages through Shoutrrr to Bark or other notification services.

- Package manager: Go modules (`go mod`); use Go version declared in `go.mod`.
- Build from repo root: `go build ./cmd/gotify-bark`.

Read relevant guide when working on these tasks:

- [Development](docs/agents/development.md): Go changes, CLI configuration.
- [Git workflow](docs/agents/git.md): Conventional Commit messages when committing changes.
- [Verification](docs/agents/verification.md): checks for code or docs changes, deployment-task side effects.
- [Credential safety](docs/agents/credentials.md): config, logging, tests, examples involving secrets.
- [Runtime overview](docs/README.md): message flow, local execution, container health.
- [User configuration](README.md): environment variables, notification URLs, migration.