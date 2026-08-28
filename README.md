# Go-Smtp

A Go-based SMTP submission service with AUTH PLAIN/LOGIN, optional STARTTLS, durable local queuing, and SMTP relay delivery.

## Configuration

Required: `SMTP_HOSTNAME`, `SMTP_AUTH_USERNAME`, and `SMTP_AUTH_PASSWORD`. Optional: `SMTP_ADDR` (default `:587`), `HTTP_ADDR` (default `:8080`), and `SMTP_TLS_CERT_FILE`/`SMTP_TLS_KEY_FILE`. When both TLS files are configured, the server advertises STARTTLS; authentication is otherwise allowed in cleartext for local development.

Accepted messages are durably written to `SMTP_QUEUE_DIR` (default `/var/lib/go-smtp/queue`). Delivery is intentionally disabled unless `SMTP_RELAY_HOST` and `SMTP_RELAY_PORT` are explicitly configured. Set `SMTP_RELAY_USERNAME` and `SMTP_RELAY_PASSWORD` when the upstream relay requires AUTH. Direct MX delivery is not enabled in this slice; use a trusted relay.

## Local

```sh
SMTP_HOSTNAME=localhost SMTP_AUTH_USERNAME=user SMTP_AUTH_PASSWORD=secret go run .
curl http://localhost:8080/health
```

## Docker Compose

Copy `.env.example` to `.env`, provide secrets outside Git, and run `docker compose up --build`. Mount certificate files and set both TLS environment variables to enable STARTTLS.

Never commit `.env`, private keys, or SMTP passwords.
