package smtp

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfigRequiresCredentials(t *testing.T) {
	_, err := LoadConfig(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing configuration error")
	}
}

func TestLoadConfigReadsSubmissionCredentials(t *testing.T) {
	values := map[string]string{
		"SMTP_HOSTNAME":        "smtp.example.com",
		"SMTP_AUTH_USERNAME":   "noreply@example.com",
		"SMTP_AUTH_PASSWORD":   "configured",
		"SMTP_DKIM_SELECTOR":   "mail",
		"SMTP_DKIM_DOMAIN":     "example.com",
		"SMTP_DKIM_PRIVATE_KEY_PATH": "/run/secrets/dkim.pem",
		"SMTP_QUEUE_DIR":        "/var/lib/smtp/queue",
		"SMTP_RELAY_HOST":       "relay.example.com",
		"SMTP_RELAY_PORT":       "2525",
		"SMTP_DELIVERY_TIMEOUT": "45s",
		"SMTP_REQUIRE_TLS":      "true",
	}
	c, err := LoadConfig(func(k string) string { return values[k] })
	if err != nil {
		t.Fatal(err)
	}
	if c.Hostname != values["SMTP_HOSTNAME"] || c.Username != values["SMTP_AUTH_USERNAME"] || c.Password != values["SMTP_AUTH_PASSWORD"] {
		t.Fatalf("unexpected submission config: %+v", c)
	}
	if c.DKIMSelector != values["SMTP_DKIM_SELECTOR"] || c.DKIMDomain != values["SMTP_DKIM_DOMAIN"] || c.DKIMPrivateKeyPath != values["SMTP_DKIM_PRIVATE_KEY_PATH"] {
		t.Fatalf("unexpected DKIM config: %+v", c)
	}
	if c.QueueDir != values["SMTP_QUEUE_DIR"] || c.RelayHost != values["SMTP_RELAY_HOST"] || c.RelayPort != 2525 || c.DeliveryTimeout.String() != "45s" || !c.RequireTLS {
		t.Fatalf("unexpected delivery config: %+v", c)
	}
}

func TestComposeWiresConsumedSMTPConfigurationNames(t *testing.T) {
	compose, err := os.ReadFile("../docker-compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(compose)
	for _, name := range []string{
		"SMTP_TLS_CERT_FILE", "SMTP_TLS_KEY_FILE", "SMTP_TLS_CERT_BASE64", "SMTP_TLS_KEY_BASE64",
		"SMTP_AUTH_USERNAME", "SMTP_AUTH_PASSWORD", "SMTP_RELAY_HOST", "SMTP_RELAY_PORT",
		"SMTP_RELAY_USERNAME", "SMTP_RELAY_PASSWORD", "SMTP_DKIM_SELECTOR",
	} {
		if !strings.Contains(text, name+": ${"+name+":-}") && name != "SMTP_AUTH_USERNAME" && name != "SMTP_AUTH_PASSWORD" {
			t.Errorf("compose does not wire %s", name)
		}
	}
	if !strings.Contains(text, "SMTP_DKIM_SELECTOR: ${SMTP_DKIM_SELECTOR:-mail}") {
		t.Error("compose does not provide the default SMTP DKIM selector")
	}
}
