# gotify-bark

Forward Gotify messages to Bark or other notification services using [Shoutrrr](https://containrrr.dev/shoutrrr/v0.8/).

## Configuration

Set these environment variables (or copy `.env.example` to `.env` when using the Taskfile or Docker):

```dotenv
APP_GOTIFY_URL=wss://gotify.example.com
APP_GOTIFY_KEY=GOTIFY_CLIENT_TOKEN
APP_SHOUTRRR_URLS=bark://:DEVICE_KEY@api.day.app/?badge=1&category=category
```

Use a Gotify **client token** to subscribe to messages. The binary reads environment variables; it does not load `.env` itself.

For multiple devices, separate notification URLs with commas:

```dotenv
APP_SHOUTRRR_URLS=bark://:DEVICE_ONE@api.day.app/?badge=1&category=category,bark://:DEVICE_TWO@api.day.app/?badge=1&category=category
```

For a self-hosted Bark server:

```dotenv
APP_SHOUTRRR_URLS=bark://:DEVICE_KEY@bark.example.com/?badge=1&category=category
```

HTTPS is the default. For an HTTP server, add `&scheme=http`. A server path prefix can go before `?`.
Bark options such as `sound`, `group`, and `icon` go in the query string; see the [Bark URL documentation](https://containrrr.dev/shoutrrr/v0.8/services/bark/).
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

## Migration

Replace `APP_BARK_URL` and `APP_BARK_DEVICE` with `APP_SHOUTRRR_URLS`, using one URL per device. The old variables and `--bark-url`/`--bark-device` flags are no longer supported. Include `badge=1&category=category` to preserve the previous Bark defaults.

For Docker Compose, place the updated environment file at `deploy/.env` and recreate the container after building or obtaining the updated image. Verify a Gotify message reaches each configured device.

## Checks

```sh
go test ./...
go build ./cmd/gotify-bark
```
