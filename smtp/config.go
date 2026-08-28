package smtp

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Hostname, Username, Password, Addr, TLSCertFile, TLSKeyFile  string
	QueueDir, RelayHost, RelayUsername, RelayPassword            string
	DKIMSelector, DKIMDomain, DKIMPrivateKeyPath, DKIMPrivateKey string
	RelayPort                                                    int
	DeliveryTimeout                                              time.Duration
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
	c := Config{Hostname: getenv("SMTP_HOSTNAME"), Username: getenv("SMTP_AUTH_USERNAME"), Password: getenv("SMTP_AUTH_PASSWORD"), Addr: getenv("SMTP_ADDR"), TLSCertFile: getenv("SMTP_TLS_CERT_FILE"), TLSKeyFile: getenv("SMTP_TLS_KEY_FILE"), QueueDir: getenv("SMTP_QUEUE_DIR"), RelayHost: getenv("SMTP_RELAY_HOST"), RelayUsername: getenv("SMTP_RELAY_USERNAME"), RelayPassword: getenv("SMTP_RELAY_PASSWORD"), DKIMSelector: getenv("SMTP_DKIM_SELECTOR"), DKIMDomain: getenv("SMTP_DKIM_DOMAIN"), DKIMPrivateKeyPath: getenv("SMTP_DKIM_PRIVATE_KEY_PATH"), DKIMPrivateKey: getenv("SMTP_DKIM_PRIVATE_KEY"), RelayPort: port, DeliveryTimeout: timeout}
	if c.Hostname == "" || c.Username == "" || c.Password == "" {
		return Config{}, errors.New("SMTP_HOSTNAME, SMTP_AUTH_USERNAME, and SMTP_AUTH_PASSWORD are required")
	}
	if c.Addr == "" {
		c.Addr = ":587"
	}
	return c, nil
}
func FromEnv() (Config, error) { return LoadConfig(os.Getenv) }
