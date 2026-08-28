package smtp

import (
	"strings"
	"testing"

	"github.com/emersion/go-sasl"
)

func TestSubmissionSessionAuthenticatesPlainCredentials(t *testing.T) {
	s := NewSession(Config{Username: "user", Password: "pass"})
	if got := s.AuthMechanisms(); len(got) != 2 || got[0] != sasl.Plain || got[1] != sasl.Login {
		t.Fatalf("mechanisms = %v", got)
	}
	a, err := s.Auth(sasl.Plain)
	if err != nil {
		t.Fatal(err)
	}
	_, done, err := a.Next([]byte("\x00user\x00pass"))
	if err != nil || !done || !s.authenticated {
		t.Fatalf("auth result done=%v err=%v authenticated=%v", done, err, s.authenticated)
	}
}

func TestSubmissionSessionRejectsBadCredentials(t *testing.T) {
	s := NewSession(Config{Username: "user", Password: "pass"})
	a, err := s.Auth(sasl.Plain)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = a.Next([]byte("\x00user\x00wrong"))
	if err == nil {
		t.Fatal("expected authentication failure")
	}
}

func TestSubmissionSessionAcceptsMessageInMemory(t *testing.T) {
	backend := NewBackend(Config{Username: "user", Password: "pass"})
	s, err := backend.NewSession(nil)
	if err != nil {
		t.Fatal(err)
	}
	session := s.(*Session)
	a, err := session.Auth(sasl.Plain)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = a.Next([]byte("\x00user\x00pass")); err != nil {
		t.Fatal(err)
	}
	if err := s.Mail("from@example.com", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Rcpt("to@example.com", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Data(strings.NewReader("Subject: test\r\n\r\nhello")); err != nil {
		t.Fatal(err)
	}
	if len(backend.Messages()) != 1 || string(backend.Messages()[0].Data) != "Subject: test\r\n\r\nhello" {
		t.Fatalf("messages = %+v", backend.Messages())
	}
}
