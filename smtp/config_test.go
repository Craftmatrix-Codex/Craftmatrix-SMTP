package smtp

import "testing"

func TestLoadConfigRequiresCredentials(t *testing.T) {
	_, err := LoadConfig(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected missing configuration error")
	}
}

func TestLoadConfigReadsSubmissionCredentials(t *testing.T) {
	values := map[string]string{"SMTP_HOSTNAME": "smtp.example.com", "SMTP_AUTH_USERNAME": "noreply@example.com", "SMTP_AUTH_PASSWORD": "secret"}
	c, err := LoadConfig(func(k string) string { return values[k] })
	if err != nil {
		t.Fatal(err)
	}
	if c.Hostname != "smtp.example.com" || c.Username != "noreply@example.com" || c.Password != "secret" {
		t.Fatalf("unexpected config: %+v", c)
	}
}
