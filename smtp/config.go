package smtp

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Hostname, Username, Password, Addr, TLSCertFile, TLSKeyFile  string
	TLSCertBase64, TLSKeyBase64                                  string
	QueueDir, RelayHost, RelayUsername, RelayPassword            string
	DKIMSelector, DKIMDomain, DKIMPrivateKeyPath, DKIMPrivateKey string
	RateLimitPerMinute                                           int
	RelayPort                                                    int
	DeliveryTimeout                                              time.Duration
	RequireTLS                                                   bool
}

func LoadConfig(getenv func(string) string) (Config, error) {
	port, _ := strconv.Atoi(getenv("SMTP_RELAY_PORT"))
	if port == 0 {
		port = 587
	}
	timeout := 30 * time.Second
	if v := getenv("SMTP_DELIVERY_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			timeout = parsed
		}
	}
	privateKey := getenv("SMTP_DKIM_PRIVATE_KEY")
	if encoded := getenv("SMTP_DKIM_PRIVATE_KEY_BASE64"); encoded != "" {
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return Config{}, fmt.Errorf("decode SMTP_DKIM_PRIVATE_KEY_BASE64: %w", err)
		}
		privateKey = string(decoded)
	}
	requireTLS := strings.EqualFold(getenv("SMTP_REQUIRE_TLS"), "true") || getenv("SMTP_REQUIRE_TLS") == "1"
	rateLimit, _ := strconv.Atoi(getenv("SMTP_RATE_LIMIT_PER_MINUTE"))
	if rateLimit == 0 {
		rateLimit = 25
	}
	c := Config{Hostname: getenv("SMTP_HOSTNAME"), Username: getenv("SMTP_AUTH_USERNAME"), Password: getenv("SMTP_AUTH_PASSWORD"), Addr: getenv("SMTP_ADDR"), TLSCertFile: getenv("SMTP_TLS_CERT_FILE"), TLSKeyFile: getenv("SMTP_TLS_KEY_FILE"), TLSCertBase64: getenv("SMTP_TLS_CERT_BASE64"), TLSKeyBase64: getenv("SMTP_TLS_KEY_BASE64"), QueueDir: getenv("SMTP_QUEUE_DIR"), RelayHost: getenv("SMTP_RELAY_HOST"), RelayUsername: getenv("SMTP_RELAY_USERNAME"), RelayPassword: getenv("SMTP_RELAY_PASSWORD"), DKIMSelector: getenv("SMTP_DKIM_SELECTOR"), DKIMDomain: getenv("SMTP_DKIM_DOMAIN"), DKIMPrivateKeyPath: getenv("SMTP_DKIM_PRIVATE_KEY_PATH"), DKIMPrivateKey: privateKey, RateLimitPerMinute: rateLimit, RelayPort: port, DeliveryTimeout: timeout, RequireTLS: requireTLS}
	if c.Hostname == "" || c.Username == "" || c.Password == "" {
		return Config{}, errors.New("SMTP_HOSTNAME, SMTP_AUTH_USERNAME, and SMTP_AUTH_PASSWORD are required")
	}
	if c.Addr == "" {
		c.Addr = ":587"
	}
	return c, nil
}
func FromEnv() (Config, error) { return LoadConfig(os.Getenv) }
