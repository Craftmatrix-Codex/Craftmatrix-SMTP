package smtp

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Hostname, Username, Password, Addr, TLSCertFile, TLSKeyFile string
	QueueDir, RelayHost, RelayUsername, RelayPassword           string
	RelayPort                                                   int
}

func LoadConfig(getenv func(string) string) (Config, error) {
	port, _ := strconv.Atoi(getenv("SMTP_RELAY_PORT"))
	c := Config{Hostname: getenv("SMTP_HOSTNAME"), Username: getenv("SMTP_AUTH_USERNAME"), Password: getenv("SMTP_AUTH_PASSWORD"), Addr: getenv("SMTP_ADDR"), TLSCertFile: getenv("SMTP_TLS_CERT_FILE"), TLSKeyFile: getenv("SMTP_TLS_KEY_FILE"), QueueDir: getenv("SMTP_QUEUE_DIR"), RelayHost: getenv("SMTP_RELAY_HOST"), RelayUsername: getenv("SMTP_RELAY_USERNAME"), RelayPassword: getenv("SMTP_RELAY_PASSWORD"), RelayPort: port}
	if c.Hostname == "" || c.Username == "" || c.Password == "" {
		return Config{}, errors.New("SMTP_HOSTNAME, SMTP_AUTH_USERNAME, and SMTP_AUTH_PASSWORD are required")
	}
	if c.Addr == "" {
		c.Addr = ":587"
	}
	return c, nil
}
func FromEnv() (Config, error) { return LoadConfig(os.Getenv) }
