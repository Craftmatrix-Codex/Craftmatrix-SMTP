package smtp

import (
	"errors"
	"os"
)

type Config struct {
	Hostname string
	Username string
	Password string
}

func LoadConfig(getenv func(string) string) (Config, error) {
	c := Config{Hostname: getenv("SMTP_HOSTNAME"), Username: getenv("SMTP_AUTH_USERNAME"), Password: getenv("SMTP_AUTH_PASSWORD")}
	if c.Hostname == "" || c.Username == "" || c.Password == "" {
		return Config{}, errors.New("SMTP_HOSTNAME, SMTP_AUTH_USERNAME, and SMTP_AUTH_PASSWORD are required")
	}
	return c, nil
}

func FromEnv() (Config, error) { return LoadConfig(os.Getenv) }
