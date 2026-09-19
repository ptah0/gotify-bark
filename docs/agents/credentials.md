# Credential safety

Read this guide when changing configuration validation, logging, notification
handling, or examples containing credentials.

- Preserve configuration validation and credential-safe errors.
- Never log Gotify tokens, notification URLs, device keys, or raw provider errors
  that may contain secrets. Shoutrrr configuration and delivery errors can expose
  credentials; retain the sanitized errors in `internal/core.go`.
- Use placeholder credentials in documentation and tests; do not commit `.env`.
