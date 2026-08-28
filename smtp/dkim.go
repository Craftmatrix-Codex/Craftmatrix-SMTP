package smtp

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/emersion/go-msgauth/dkim"
)

type DKIMConfig struct{ Selector, Domain, PrivateKeyPath string }

func loadDKIMSigner(path string) (crypto.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read DKIM private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode DKIM private key: PEM block not found")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if signer, ok := key.(crypto.Signer); ok {
			return signer, nil
		}
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse DKIM private key: %w", err)
	}
	return key, nil
}

func signMessage(message []byte, cfg DKIMConfig) ([]byte, error) {
	if cfg.Selector == "" && cfg.Domain == "" && cfg.PrivateKeyPath == "" {
		return message, nil
	}
	if cfg.Selector == "" || cfg.Domain == "" || cfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("DKIM selector, domain, and private key path must all be configured")
	}
	signer, err := loadDKIMSigner(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err = dkim.Sign(&out, bytes.NewReader(message), &dkim.SignOptions{Domain: cfg.Domain, Selector: cfg.Selector, Signer: signer, Hash: crypto.SHA256, HeaderKeys: []string{"From", "To", "Subject", "Date", "Message-ID"}})
	if err != nil {
		return nil, fmt.Errorf("sign message with DKIM: %w", err)
	}
	return out.Bytes(), nil
}
