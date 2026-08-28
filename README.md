# Go-Smtp

A Go-based SMTP submission service with AUTH PLAIN/LOGIN, optional STARTTLS, durable local queuing, and direct MX or SMTP relay delivery.

## Configuration

Required: `SMTP_HOSTNAME`, `SMTP_AUTH_USERNAME`, and `SMTP_AUTH_PASSWORD`. Optional: `SMTP_ADDR` (default `:587`), `HTTP_ADDR` (default `:8080`), and `SMTP_TLS_CERT_FILE`/`SMTP_TLS_KEY_FILE`. When both TLS files are configured, the server advertises STARTTLS; authentication is otherwise allowed in cleartext for local development.

Accepted messages are durably written to `SMTP_QUEUE_DIR` (default `/var/lib/go-smtp/queue`). If `SMTP_RELAY_HOST` is set, messages use that relay and `SMTP_RELAY_PORT` (default 587). Otherwise the worker resolves recipient MX records and delivers directly to port 25. `SMTP_DELIVERY_TIMEOUT` (default `30s`) controls MX connection and SMTP command deadlines. Failed deliveries remain queued with incremented retry attempts. Direct delivery signs messages with DKIM when `SMTP_DKIM_SELECTOR`, `SMTP_DKIM_DOMAIN`, and either `SMTP_DKIM_PRIVATE_KEY_PATH`, `SMTP_DKIM_PRIVATE_KEY`, or `SMTP_DKIM_PRIVATE_KEY_BASE64` are set. `SMTP_DKIM_PRIVATE_KEY` accepts the PEM contents directly; `SMTP_DKIM_PRIVATE_KEY_BASE64` accepts a single-line standard base64-encoded PEM and takes priority when both key environment variables are set, making it suitable for Coolify and other deployment platforms that do not preserve multiline environment values. The path option remains available. Never log or commit private key contents.

## Local

```sh
SMTP_HOSTNAME=localhost SMTP_AUTH_USERNAME=user SMTP_AUTH_PASSWORD=secret go run .
curl http://localhost:8080/health
```

## Docker Compose

Copy `.env.example` to `.env`, provide secrets outside Git, and run `docker compose up --build`. Mount certificate files and set both TLS environment variables to enable STARTTLS.

Never commit `.env`, private keys, or SMTP passwords.
