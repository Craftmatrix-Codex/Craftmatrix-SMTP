# Go-Smtp

A Go-based outbound SMTP submission service.

## Status

Initial Docker Compose scaffold. SMTP protocol handling, authentication, TLS, queueing, DKIM signing, and delivery are not implemented yet.

## Local health check

```sh
go run .
curl http://localhost:8080/health
```

## Docker Compose

Copy `.env.example` to `.env`, provide secrets outside Git, and run:

```sh
docker compose up --build
```

Never commit `.env`, private keys, or SMTP passwords.
