# Go-Smtp

A lightweight, self-hosted Go SMTP submission and outbound-delivery server.

It accepts authenticated mail on SMTP submission port `587`, stores messages in a durable filesystem queue, and delivers directly to recipient-domain MX servers on port `25`. An optional authenticated upstream relay is also supported. The project is designed for deployment on a VPS with Docker Compose/Coolify and Cloudflare-managed DNS.

> This is an outbound SMTP server. It is not a mailbox platform: it does not provide IMAP, POP3, webmail, inbound mailbox storage, or user inboxes.

## Architecture

```text
SMTP client
  │ AUTH + optional STARTTLS
  ▼
smtp.forgetools.site:587
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

## DNS setup for `forgetools.site`

Mail records must be DNS-only in Cloudflare. Do not orange-cloud SMTP records.

### Forward A record

Point the SMTP hostname at the VPS public IP:

```text
smtp.forgetools.site.  A  <YOUR_VPS_IP>
```

### SPF

Publish one SPF record at the sending domain. Do not create multiple SPF records:

```text
forgetools.site.  TXT  "v=spf1 ip4:<YOUR_VPS_IP> -all"
```

If other services also send mail for the domain, combine them into this same SPF record instead of adding a second one.

### DKIM

The service uses selector `mail` in the production setup. Publish the public key generated from the private key configured in Coolify:

```text
mail._domainkey.forgetools.site.  TXT  "v=DKIM1; k=rsa; p=<base64-public-key>"
```

Never publish the private key. Never commit it to this repository.

### DMARC

Start with monitoring mode while validating delivery:

```text
_dmarc.forgetools.site.  TXT  "v=DMARC1; p=none; rua=mailto:aspirasrenz@gmail.com"
```

After SPF/DKIM alignment and reporting are confirmed, a stricter policy can be considered, for example `p=quarantine` or `p=reject`.

### MX and reverse DNS

An MX record for `forgetools.site` is required only when this server is expected to receive mail for the domain. It is not required merely to send directly to Gmail or another recipient domain.

Reverse DNS/PTR is controlled by the VPS provider, not Cloudflare. The desired setup is:

```text
<YOUR_VPS_IP> → smtp.forgetools.site
smtp.forgetools.site → <YOUR_VPS_IP>
```

The current VPS PTR may remain provider-generated if the provider does not allow it to be changed. This can reduce sender reputation even when SPF, DKIM, and DMARC pass.

## Environment variables

Required:

```text
SMTP_HOSTNAME=smtp.forgetools.site
SMTP_AUTH_USERNAME=noreply@forgetools.site
SMTP_AUTH_PASSWORD=<secret>
```

Common settings:

```text
SMTP_ADDR=:587
HTTP_ADDR=:8080
SMTP_QUEUE_DIR=/tmp/go-smtp/queue
SMTP_DELIVERY_TIMEOUT=30s
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
SMTP_DKIM_DOMAIN=forgetools.site
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

## Sending a test message

Example using Python's standard library:

```python
import smtplib

message = """From: noreply@forgetools.site
To: recipient@example.com
Subject: SMTP test

Test message.
"""

with smtplib.SMTP("smtp.forgetools.site", 587) as client:
    client.ehlo()
    client.starttls()
    client.ehlo()
    client.login("noreply@forgetools.site", "<password>")
    client.sendmail("noreply@forgetools.site", ["recipient@example.com"], message)
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
