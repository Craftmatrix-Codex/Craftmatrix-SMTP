package smtp

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
)

type Message struct {
	From, To string
	Data     []byte
}
type Backend struct {
	cfg      Config
	queue    *Queue
	mu       sync.RWMutex
	messages []Message
}

func NewBackend(cfg Config) *Backend {
	b := &Backend{cfg: cfg}
	if cfg.QueueDir != "" {
		b.queue, _ = NewQueue(cfg.QueueDir)
	}
	return b
}
func (b *Backend) NewSession(_ *gosmtp.Conn) (gosmtp.Session, error) {
	return NewSessionWithBackend(b), nil
}
func (b *Backend) Messages() []Message {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Message, len(b.messages))
	copy(out, b.messages)
	return out
}

type Session struct {
	backend       *Backend
	cfg           Config
	authenticated bool
	from, to      string
}

func NewSession(cfg Config) *Session            { return &Session{cfg: cfg} }
func NewSessionWithBackend(b *Backend) *Session { return &Session{backend: b, cfg: b.cfg} }
func (s *Session) AuthMechanisms() []string     { return []string{sasl.Plain, sasl.Login} }
func (s *Session) Auth(mech string) (sasl.Server, error) {
	switch strings.ToUpper(mech) {
	case sasl.Plain:
		return sasl.NewPlainServer(s.authenticate), nil
	case sasl.Login:
		return &loginServer{session: s}, nil
	default:
		return nil, gosmtp.ErrAuthUnsupported
	}
}
func (s *Session) authenticate(identity, username, password string) error {
	if identity != "" && identity != username {
		return errors.New("invalid identity")
	}
	if username != s.cfg.Username || password != s.cfg.Password {
		return errors.New("invalid credentials")
	}
	s.authenticated = true
	return nil
}
func (s *Session) Reset()        { s.from = ""; s.to = "" }
func (s *Session) Logout() error { return nil }
func (s *Session) Mail(from string, _ *gosmtp.MailOptions) error {
	if !s.authenticated {
		return errors.New("authentication required")
	}
	s.from = from
	return nil
}
func (s *Session) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	if s.from == "" {
		return errors.New("MAIL required")
	}
	s.to = to
	return nil
}
func (s *Session) Data(r io.Reader) error {
	if s.to == "" {
		return errors.New("recipient required")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if s.backend == nil {
		return errors.New("no backend")
	}
	s.backend.mu.Lock()
	s.backend.messages = append(s.backend.messages, Message{s.from, s.to, data})
	s.backend.mu.Unlock()
	if s.backend.queue != nil {
		if _, err := s.backend.queue.Enqueue(Message{From: s.from, To: s.to, Data: data}); err != nil {
			return err
		}
	}
	return nil
}

type loginServer struct {
	session  *Session
	step     int
	username string
}

func (a *loginServer) Next(response []byte) ([]byte, bool, error) {
	if a.step == 0 {
		a.step++
		return []byte("Username:"), false, nil
	}
	if a.step == 1 {
		a.username = string(response)
		a.step++
		return []byte("Password:"), false, nil
	}
	if err := a.session.authenticate("", a.username, string(response)); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}
func TLSConfig(cfg Config) (*tls.Config, error) {
	if cfg.TLSCertFile == "" && cfg.TLSKeyFile == "" {
		return nil, nil
	}
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		return nil, fmt.Errorf("SMTP_TLS_CERT_FILE and SMTP_TLS_KEY_FILE must both be set")
	}
	cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}, nil
}
