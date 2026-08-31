# 📮 Craftmatrix SMTP

[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![SMTP](https://img.shields.io/badge/SMTP-587%20%7C%2025-6f42c1)](https://www.rfc-editor.org/rfc/rfc5321)
[![Security](https://img.shields.io/badge/TLS-STARTTLS-success?logo=letsencrypt&logoColor=white)](https://datatracker.ietf.org/doc/html/rfc3207)

> 🛠️ **Craftmatrix SMTP** is a lightweight, self-hosted Go SMTP submission and direct-MX delivery server for `example.com`.

| Capability | Status |
|---|---:|
| Authenticated SMTP submission | ✅ Ready |
| Port 587 STARTTLS | ✅ Ready |
| Durable outbound queue | ✅ Ready |
| Direct MX delivery on port 25 | ✅ Ready |
| Outbound opportunistic STARTTLS | ✅ Ready |
| SPF and DKIM | ✅ Configured |
| DMARC | 🟡 Recommended/configure policy |
| Reverse DNS/PTR | ⚠️ VPS-provider controlled |
| IMAP, POP3, webmail, inboxes | ❌ Not included |

## 🧭 Contents

- [✨ Features](#-features)
- [🏗️ Architecture](#️-architecture)
- [🌐 DNS setup](#-dns-setup-for-forgetoolssite)
- [⚙️ Configuration](#️-configuration)
- [🐳 Docker Compose](#-docker-compose)
- [🚀 Coolify deployment](#-coolify-deployment)
- [📨 SMTP clients and testing](#-smtp-client-compatibility-and-testing)
- [📊 Deliverability](#-deliverability-checklist)
- [🔐 Security](#-security-notes)
- [🧪 Development](#-development-verification)

## ✨ Features

- 🔑 `AUTH PLAIN` and `AUTH LOGIN`
- 🔒 STARTTLS for client submission
- 📥 Durable filesystem queue with atomic writes
- 🔁 Retry retention when delivery fails, with exponential backoff
- ⏱️ Evenly paced outbound delivery with a configurable per-minute limit
- 🌍 Recipient MX lookup and direct TCP 25 delivery
- 🛡️ Opportunistic STARTTLS for recipient MX servers
- ✍️ DKIM signing for direct-MX messages
- 🧾 RFC 5322 header normalization (`Date`, `Message-ID`, and more)
- 🩺 HTTP `/health` endpoint and container healthcheck
- 🔧 Environment-variable configuration
- ☁️ Docker Compose and Coolify deployment support

> 💡 This is an outbound SMTP server. It does not provide IMAP, POP3, webmail, inbound mailbox storage, or user inboxes.

## 🏗️ Architecture

| Traffic | Port | Exposure | Purpose |
|---|---:|---|---|
| SMTP submission | `587` | Direct host port | Authenticated client sending |
| Direct MX delivery | `25` | Outbound only | Server-to-server delivery |
| HTTP health/root | `8080` | Internal container port | Coolify proxy and health checks |

```text
📧 SMTP client → smtp.example.com:587 → 📥 queue → 🌍 recipient MX:25 → 📬 mailbox
🌐 HTTP /health:8080 → ☁️ Coolify Traefik
```

The normal Coolify HTTP proxy handles HTTP only. SMTP is not routed through the ordinary HTTP reverse proxy.

## ✨ Features

It accepts authenticated mail on SMTP submission port `587`, stores messages in a durable filesystem queue, and delivers directly to recipient-domain MX servers on port `25`. An optional authenticated upstream relay is also supported. The project is designed for deployment on a VPS with Docker Compose/Coolify and Cloudflare-managed DNS.

> This is an outbound SMTP server. It is not a mailbox platform: it does not provide IMAP, POP3, webmail, inbound mailbox storage, or user inboxes.

## Architecture

```text
SMTP client
  │ AUTH + optional STARTTLS
  ▼
smtp.example.com:587
  │
  ▼
Go SMTP submission server
  │ durable filesystem queue
  ▼
Direct MX delivery on TCP 25
  │ opportunistic STARTTLS when recipient MX advertises it
  ▼
Recipient mail server (for example Gmail)
```

The HTTP server is separate from SMTP:

- `GET /` returns `Hello World`.
- `GET /health` returns a machine-readable health response.
- Coolify's Traefik proxy may route HTTP/HTTPS to the internal HTTP port.
- SMTP port `587` is a direct TCP mapping. A normal HTTP reverse proxy cannot proxy SMTP.

## Features

- SMTP submission on port `587`
- `AUTH PLAIN` and `AUTH LOGIN`
- Optional inbound STARTTLS for authenticated submission
- Durable filesystem queue with atomic writes
- Retry retention when outbound delivery fails
- Direct recipient MX lookup and delivery on port `25`
- Opportunistic STARTTLS for direct MX delivery
- Optional `SMTP_REQUIRE_TLS=true` to reject MX delivery without STARTTLS
- Optional authenticated SMTP relay mode
- RFC 5322 header normalization, including `Date` and `Message-ID`
- DKIM signing for direct-MX messages
- HTTP root and health endpoints
- Container healthcheck
- Environment-variable configuration

## 🌐 DNS setup for `example.com`

Mail records must be DNS-only in Cloudflare. Do not orange-cloud SMTP records.

### Forward A record

Point the SMTP hostname at the VPS public IP:

```text
smtp.example.com.  A  <YOUR_VPS_IP>
```

### SPF

Publish one SPF record at the sending domain. Do not create multiple SPF records:

```text
example.com.  TXT  "v=spf1 ip4:<YOUR_VPS_IP> -all"
```

If other services also send mail for the domain, combine them into this same SPF record instead of adding a second one.

### DKIM

The service uses selector `mail` in the production setup. Publish the public key generated from the private key configured in Coolify:

```text
mail._domainkey.example.com.  TXT  "v=DKIM1; k=rsa; p=<base64-public-key>"
```

Never publish the private key. Never commit it to this repository.

### DMARC

Start with monitoring mode while validating delivery:

```text
_dmarc.example.com.  TXT  "v=DMARC1; p=none; rua=mailto:postmaster@example.com"
```

After SPF/DKIM alignment and reporting are confirmed, a stricter policy can be considered, for example `p=quarantine` or `p=reject`.

### MX and reverse DNS

An MX record for `example.com` is required only when this server is expected to receive mail for the domain. It is not required merely to send directly to Gmail or another recipient domain.

Reverse DNS/PTR is controlled by the VPS provider, not Cloudflare. The desired setup is:

```text
<YOUR_VPS_IP> → smtp.example.com
smtp.example.com → <YOUR_VPS_IP>
```

The current VPS PTR may remain provider-generated if the provider does not allow it to be changed. This can reduce sender reputation even when SPF, DKIM, and DMARC pass.

## Environment variables

Required:

```text
SMTP_HOSTNAME=smtp.example.com
SMTP_AUTH_USERNAME=noreply@example.com
SMTP_AUTH_PASSWORD=<secret>
```

Common settings:

```text
SMTP_ADDR=:587
HTTP_ADDR=:8080
SMTP_QUEUE_DIR=/tmp/go-smtp/queue
SMTP_DELIVERY_TIMEOUT=30s
SMTP_RATE_LIMIT_PER_MINUTE=25
SMTP_REQUIRE_TLS=false
```

Optional submission STARTTLS certificate:

```text
SMTP_TLS_CERT_FILE=/run/secrets/smtp-cert.pem
SMTP_TLS_KEY_FILE=/run/secrets/smtp-key.pem
```

Optional relay mode. If `SMTP_RELAY_HOST` is empty, direct MX delivery is used:

```text
SMTP_RELAY_HOST=
SMTP_RELAY_PORT=587
SMTP_RELAY_USERNAME=
SMTP_RELAY_PASSWORD=
```

DKIM settings:

```text
SMTP_DKIM_SELECTOR=mail
SMTP_DKIM_DOMAIN=example.com
SMTP_DKIM_PRIVATE_KEY_PATH=
SMTP_DKIM_PRIVATE_KEY=
SMTP_DKIM_PRIVATE_KEY_BASE64=<single-line-base64-encoded-PEM>
```

`SMTP_DKIM_PRIVATE_KEY_BASE64` is preferred for Coolify because it avoids multiline environment-value parsing problems. The private key must be supplied through a protected secret and must never be logged or committed.

## Docker Compose

The repository uses `docker-compose.yaml` because some Coolify configurations look specifically for that filename:

```sh
cp .env.example .env
# Edit .env and provide secrets outside Git.
docker compose -f docker-compose.yaml up --build -d
```

The Compose deployment exposes SMTP directly:

```yaml
ports:
  - "${SMTP_PORT:-587}:587"
```

HTTP port `8080` is intentionally not published on the host. It is exposed internally for Coolify's proxy and health routing, avoiding conflicts with other applications already using host port `8080`.

The queue is stored in the named `smtp_queue` volume and survives container restarts.

## Coolify deployment

Use a Git-backed Docker Compose application:

- Repository: `https://github.com/Craftmatrix-Codex/Go-Smtp`
- Branch: `main`
- Compose file: `docker-compose.yaml`
- Internal HTTP port: `8080`
- Public SMTP port: `587`
- HTTP health path: `/health`
- Health method: `GET`
- Expected health status: `200`

Configure the stable HTTP hostname through Coolify's Domains setting. The SMTP hostname must remain DNS-only in Cloudflare and must not be routed through the ordinary HTTP proxy.

Coolify environment values should include the required SMTP credentials, `SMTP_HOSTNAME`, DKIM settings, and the base64 DKIM private key. Redeploy after changing environment variables and verify the application status is `running:healthy`.

## SMTP client compatibility and testing

This server is a standard authenticated SMTP submission server. Use a client that supports custom SMTP host configuration, port `587`, STARTTLS, and normal-password authentication.

### Recommended clients

- **Thunderbird** — recommended for testing and daily use
- **Apple Mail**
- **Microsoft Outlook**
- **Mailspring**
- Python `smtplib` or another standard SMTP library

### Exact client settings

```text
SMTP server:       smtp.example.com
SMTP port:         587
Security:          STARTTLS
Authentication:    Normal password
Username:          noreply@example.com
Password:          the configured Go SMTP password
```

Use the hostname, not the IP address. The TLS certificate is issued to `smtp.example.com`, so connecting to `<YOUR_VPS_IP>` causes hostname verification errors.

Do not use port `25` for client submission. Port `25` is used by the server for direct server-to-server MX delivery. Do not select implicit SSL/TLS on port `587`; select STARTTLS.

The live SMTP listener must advertise:

```text
250-STARTTLS
```

If a client reports an invalid certificate, update its CA bundle and ensure SNI/server name is `smtp.example.com`. Do not disable certificate verification in production.

### GMass and Gmail

GMass is primarily a Gmail/Google Workspace extension and is not a general-purpose SMTP test client. It normally sends through the connected Gmail account or Google sending infrastructure rather than allowing arbitrary SMTP-server testing. Keep GMass connected to Gmail for GMass campaigns, and use Thunderbird or another standard SMTP client to test this server.

Gmail's **Send mail as** feature may support an external SMTP server, but Gmail requires verification of the sender address. Because this project is outbound-only and does not provide an inbound mailbox, Gmail may not be able to receive that verification code through this server. This is separate from SMTP authentication and does not indicate that port `587` is broken.

### Sending a test message

Example using Python's standard library:

```python
import smtplib

message = """From: noreply@example.com
To: recipient@example.com
Subject: SMTP test

Test message.
"""

with smtplib.SMTP("smtp.example.com", 587) as client:
    client.ehlo()
    client.starttls()
    client.ehlo()
    client.login("noreply@example.com", "<password>")
    client.sendmail("noreply@example.com", ["recipient@example.com"], message)
```

Submission acceptance means the message entered the queue. It does not by itself prove final delivery. Check the application logs for direct-MX responses and inspect the recipient mailbox, including spam.

## Deliverability checklist

- [ ] Forward A record points to the VPS
- [ ] SMTP hostname is DNS-only in Cloudflare
- [ ] SPF includes the VPS IP
- [ ] DKIM public TXT matches the deployed private key
- [ ] DKIM signing is enabled in the service
- [ ] DMARC record exists
- [ ] VPS outbound TCP port `25` is open
- [ ] PTR/reverse DNS is set by the VPS provider if possible
- [ ] Forward and reverse DNS are consistent
- [ ] `running:healthy` is shown in Coolify
- [ ] SMTP AUTH and STARTTLS are tested
- [ ] Recipient MX accepts the message
- [ ] Sender reputation is built gradually with legitimate low-volume mail

New self-hosted VPS senders commonly land in spam even when technically valid. Reputation, consistent identity, PTR, SPF, DKIM, DMARC, message quality, and sending history all affect placement.

## Security notes

- Never commit `.env`, SMTP passwords, DKIM private keys, TLS private keys, or provider tokens.
- Use Coolify protected environment variables or external secret mounts.
- Use STARTTLS for SMTP submission.
- Set `SMTP_REQUIRE_TLS=true` if direct delivery must never fall back to plaintext when a recipient MX lacks STARTTLS.
- DKIM signs mail; it does not encrypt message contents.
- STARTTLS encrypts the SMTP transport hop. End-to-end encryption requires PGP or S/MIME and recipient-side key management.

## Development verification

```sh
GOTOOLCHAIN=local gofmt -w main.go smtp/*.go
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go build ./...
GOTOOLCHAIN=local go vet ./...
git diff --check
```
