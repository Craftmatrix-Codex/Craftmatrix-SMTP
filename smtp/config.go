package smtp

import (
	"errors"
	"os"
)

type Config struct{ Hostname, Username, Password, Addr, TLSCertFile, TLSKeyFile string }

func LoadConfig(getenv func(string) string) (Config, error) {
	c := Config{Hostname: getenv("SMTP_HOSTNAME"), Username: getenv("SMTP_AUTH_USERNAME"), Password: getenv("SMTP_AUTH_PASSWORD"), Addr: getenv("SMTP_ADDR"), TLSCertFile: getenv("SMTP_TLS_CERT_FILE"), TLSKeyFile: getenv("SMTP_TLS_KEY_FILE")}
	if c.Hostname == "" || c.Username == "" || c.Password == "" {
		return Config{}, errors.New("SMTP_HOSTNAME, SMTP_AUTH_USERNAME, and SMTP_AUTH_PASSWORD are required")
	}
	if c.Addr == "" {
		c.Addr = ":587"
	}
	return c, nil
}
func FromEnv() (Config, error) { return LoadConfig(os.Getenv) }
