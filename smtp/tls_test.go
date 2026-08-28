package smtp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func TestTLSConfigLoadsBase64CertificateAndKey(t *testing.T) {
	certPEM, keyPEM := makeTLSCertificate(t)
	cfg := Config{
		TLSCertBase64: base64.StdEncoding.EncodeToString(certPEM),
		TLSKeyBase64:  base64.StdEncoding.EncodeToString(keyPEM),
	}
	got, err := TLSConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got.Certificates) != 1 {
		t.Fatalf("TLS config certificates = %#v", got)
	}
	parsed, err := x509.ParseCertificate(got.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := parsed.VerifyHostname("smtp.example.com"); err != nil {
		t.Fatalf("loaded certificate does not verify hostname: %v", err)
	}
}

func makeTLSCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "smtp.example.com"}, DNSNames: []string{"smtp.example.com"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
}
