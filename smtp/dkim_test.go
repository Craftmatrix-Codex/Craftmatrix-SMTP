package smtp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"
)

func TestLoadConfigReadsDKIMSettings(t *testing.T) {
	values := map[string]string{
		"SMTP_HOSTNAME": "smtp.example.com", "SMTP_AUTH_USERNAME": "u", "SMTP_AUTH_PASSWORD": "p",
		"SMTP_DKIM_SELECTOR": "mail", "SMTP_DKIM_DOMAIN": "example.com", "SMTP_DKIM_PRIVATE_KEY_PATH": "/run/secrets/dkim.key",
	}
	cfg, err := LoadConfig(func(k string) string { return values[k] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DKIMSelector != "mail" || cfg.DKIMDomain != "example.com" || cfg.DKIMPrivateKeyPath != "/run/secrets/dkim.key" {
		t.Fatalf("DKIM config = %#v", cfg)
	}
}

func TestLoadConfigReadsDKIMPrivateKey(t *testing.T) {
	values := map[string]string{
		"SMTP_HOSTNAME": "smtp.example.com", "SMTP_AUTH_USERNAME": "u", "SMTP_AUTH_PASSWORD": "p",
		"SMTP_DKIM_SELECTOR": "mail", "SMTP_DKIM_DOMAIN": "example.com", "SMTP_DKIM_PRIVATE_KEY": "PEM contents",
	}
	cfg, err := LoadConfig(func(k string) string { return values[k] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DKIMPrivateKey != "PEM contents" {
		t.Fatalf("DKIM private key = %q", cfg.DKIMPrivateKey)
	}
}

func TestSignMessageWithDKIMPrivateKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	msg := []byte("From: sender@example.com\r\nTo: user@example.net\r\nSubject: hi\r\n\r\nhello\r\n")
	signed, err := signMessage(msg, DKIMConfig{Selector: "mail", Domain: "example.com", PrivateKey: pemKey})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(signed), "DKIM-Signature: ") {
		t.Fatalf("missing DKIM header: %q", signed)
	}
}

func TestSignMessageAddsValidDKIMHeader(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/dkim.key"
	b := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	msg := []byte("From: sender@example.com\r\nTo: user@example.net\r\nSubject: hi\r\n\r\nhello\r\n")
	signed, err := signMessage(msg, DKIMConfig{Selector: "mail", Domain: "example.com", PrivateKeyPath: path})
	if err != nil {
		t.Fatal(err)
	}
	text := string(signed)
	if !strings.HasPrefix(text, "DKIM-Signature: ") || !strings.Contains(text, " d=example.com") || !strings.Contains(text, " s=mail") {
		t.Fatalf("missing DKIM header: %q", text)
	}
}
