# Repository guidance

gotify-bark forwards Gotify WebSocket messages through Shoutrrr to Bark or other notification services.

- Package manager: Go modules (`go mod`); use the Go version declared in `go.mod`.
- Build from the repository root: `go build ./cmd/gotify-bark`.

Read the relevant guide when working on these tasks:

- [Development](docs/agents/development.md): Go changes and CLI configuration.
- [Git workflow](docs/agents/git.md): Conventional Commit messages when committing changes.
- [Verification](docs/agents/verification.md): checks for code or documentation changes and deployment-task side effects.
- [Credential safety](docs/agents/credentials.md): configuration, logging, tests, and examples involving secrets.
- [Runtime overview](docs/README.md): message flow, local execution, and container health.
- [User configuration](README.md): environment variables, notification URLs, and migration.
