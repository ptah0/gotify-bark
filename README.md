# gotify-bark

Forward Gotify messages to Bark or other notification services using the maintained [Shoutrrr fork](https://github.com/nicholas-fedor/shoutrrr).

See [development and runtime documentation](docs/README.md) and
[repository guidance](AGENTS.md) for contributing.

## Configuration

Set these environment variables (or copy `.env.example` to `.env` when using the Taskfile or Docker):

```dotenv
APP_GOTIFY_URL=wss://gotify.example.com
APP_GOTIFY_KEY=GOTIFY_CLIENT_TOKEN
APP_SHOUTRRR_URLS=bark://:DEVICE_KEY@api.day.app/?badge=1&category=category
```

Use a Gotify **client token** to subscribe to messages. The binary reads environment variables; it does not load `.env` itself.
Set `APP_DEBUG=true` or pass `--debug` to enable debug logging.
The Gotify URL must use `ws://` or `wss://`. A base path such as
`wss://gotify.example.com/gotify` is supported; the subscription uses `/gotify/stream`.

For multiple devices, separate notification URLs with commas:

```dotenv
APP_SHOUTRRR_URLS=bark://:DEVICE_ONE@api.day.app/?badge=1&category=category,bark://:DEVICE_TWO@api.day.app/?badge=1&category=category
```

For a self-hosted Bark server:

```dotenv
APP_SHOUTRRR_URLS=bark://:DEVICE_KEY@bark.example.com/?badge=1&category=category
```

HTTPS is the default. For an HTTP server, add `&scheme=http`. A server path prefix can go before `?`.
Bark options such as `sound`, `group`, and `icon` go in the query string; see the [Bark URL documentation](https://github.com/nicholas-fedor/shoutrrr/blob/v0.21.0/docs/services/push/bark/index.md).
URL-encode reserved characters in keys and option values, including literal commas (`%2C`). Treat notification URLs as secrets.

Alternatively, repeat the CLI flag (quote URLs to protect `&` from the shell):

```sh
go run ./cmd/gotify-bark \
  --gotify-url wss://gotify.example.com \
  --gotify-key "$APP_GOTIFY_KEY" \
  --shoutrrr-url 'bark://:DEVICE_ONE@api.day.app/?badge=1&category=category' \
  --shoutrrr-url 'bark://:DEVICE_TWO@api.day.app/?badge=1&category=category'
```

Gotify's body is sent as the notification message and its title as Shoutrrr's `title` parameter. Title support depends on the destination service. Malformed messages are skipped; delivery failures are logged without credentials and are not retried. `/status` remains available on port 8080.

Messages are forwarded sequentially, with destinations for each message sent concurrently.
A delivery exceeding 10 seconds logs an error and processing continues. The Shoutrrr
`Sender` API used here cannot cancel sends: a timed-out delivery may still complete, and further
messages skip that destination while it remains busy. Other destinations continue.
Failed connections and unexpected Gotify disconnects log sanitized errors and retry
every second. Messages missed while disconnected are not replayed.
`SIGINT` and `SIGTERM` stop the service and close its WebSocket and status listener.
`/status` uses `health-go` with system information and two checks: `gotify` reports
the observed WebSocket connection state; `notifications` reports the latest completed
delivery result across all destinations. Either failure returns HTTP 503. A subsequent
successful delivery to all destinations clears the delivery failure. Before the first
delivery, that check passes; malformed messages do not change it. These checks do not
send test notifications or actively probe idle connections or provider availability.

## Migration

Replace `APP_BARK_URL` and `APP_BARK_DEVICE` with `APP_SHOUTRRR_URLS`, using one URL per device. The old variables and `--bark-url`/`--bark-device` flags are no longer supported. Include `badge=1&category=category` to preserve the previous Bark defaults.

For Docker Compose, place the updated environment file at `deploy/.env` and recreate the container after building or obtaining the updated image. Verify a Gotify message reaches each configured device.

## Checks

```sh
go test ./...
go build ./cmd/gotify-bark
```

## CI and releases

[Build and release](.github/workflows/build.yml) builds pull requests targeting
`main` when opened, updated, or reopened, and on every push to `main`, including
PR merges and direct pushes. Version-tag pushes (`v*`) also trigger the workflow
for releases. It checks Go formatting, runs
`go vet` and race-enabled tests, builds the binary, and builds the Docker image
for Linux amd64 and arm64. Pull-request and `main` builds do not publish anything.
Dependabot checks Actions updates monthly.

Before the first release, configure the repository in GitHub Settings:

- Under **Secrets and variables → Actions**, add `DOCKERHUB_USERNAME` and
  `DOCKERHUB_TOKEN` secrets. Use a Docker Hub access token with write permission
  to the target Docker Hub repository.
- Docker Hub defaults to the lowercase GitHub owner/repository name. To override
  it, set the `DOCKERHUB_IMAGE` repository variable to the full image name, such
  as `your-account/gotify-bark`.
- GHCR and GitHub Releases use the built-in `GITHUB_TOKEN`; no personal access
  token is needed. Organization policies must allow the workflow to write
  repository contents and packages. For an existing GHCR package, grant this
  repository Actions access in the package settings.
- After the first publication, set the GHCR package visibility to **Public**
  if anonymous pulls are wanted; new packages default to private.

Create a release by pushing a version tag from the intended commit:

```sh
git tag v1.2.3
git push origin v1.2.3
```

Supported tags are `vMAJOR.MINOR.PATCH`, optionally followed by `-alpha.N`,
`-beta.N`, or `-rc.N` (for example, `v1.2.3-rc.1`). Other tags starting with `v`
fail validation; tags without that prefix do not trigger the workflow.
After checks pass, the workflow publishes:

- Multi-platform images to `ptah0/gotify-bark` and
  `ghcr.io/ptah0/gotify-bark`, tagged with the version without `v` and a short
  Git SHA. Stable versions also update `latest`; prereleases do not.
- A GitHub Release with generated notes, Linux amd64/arm64 binary archives,
  and `checksums.txt`. Prerelease tags create GitHub prereleases.

For example, pull `ghcr.io/ptah0/gotify-bark:1.2.3` or
`ptah0/gotify-bark:1.2.3`. In forks, both registries default to the lowercase
GitHub owner/repository name; `DOCKERHUB_IMAGE` overrides only Docker Hub.
Publishing a release manually in the GitHub UI does not trigger this workflow;
the tag push is the release trigger. Push stable releases in ascending version
order because each stable tag updates the image's `latest` tag.

The GitHub Release is created after both image registries succeed. Publication
across registries is not atomic: if one fails, fix its credentials or access
and rerun the failed jobs. Rerunning the release job replaces matching assets
on an existing release. Protect release tags with a GitHub tag ruleset and
require the `checks` and `image` jobs in the default branch's protection rules.
