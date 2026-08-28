# Go-Smtp

A Go-based SMTP submission service with AUTH PLAIN/LOGIN, optional STARTTLS, and in-memory message acceptance.

## Configuration

Required: `SMTP_HOSTNAME`, `SMTP_AUTH_USERNAME`, and `SMTP_AUTH_PASSWORD`. Optional: `SMTP_ADDR` (default `:587`), `HTTP_ADDR` (default `:8080`), and `SMTP_TLS_CERT_FILE`/`SMTP_TLS_KEY_FILE`. When both TLS files are configured, the server advertises STARTTLS; authentication is otherwise allowed in cleartext for local development.

Messages are accepted in memory only (no delivery or durable queue yet).

## Local

```sh
SMTP_HOSTNAME=localhost SMTP_AUTH_USERNAME=user SMTP_AUTH_PASSWORD=secret go run .
curl http://localhost:8080/health
```

## Docker Compose

Copy `.env.example` to `.env`, provide secrets outside Git, and run `docker compose up --build`. Mount certificate files and set both TLS environment variables to enable STARTTLS.

Never commit `.env`, private keys, or SMTP passwords.
