package smtp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"sort"
	"strings"
	"time"
)

type MXResolver interface {
	LookupMX(string) ([]*net.MX, error)
}
type systemMXResolver struct{}

func (systemMXResolver) LookupMX(domain string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(context.Background(), domain)
}

type DirectMXDelivery struct {
	Resolver MXResolver
	Dialer   *net.Dialer
	Port     int
	Timeout  time.Duration
	From     string
	Hostname string
}

func normalizeMessage(m Message, hostname string) ([]byte, error) {
	data := strings.ReplaceAll(string(m.Data), "\r\n", "\n")
	parts := strings.SplitN(data, "\n\n", 2)
	headers := parts[0]
	body := ""
	if len(parts) == 2 {
		body = parts[1]
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(headers, "\n") {
		if i := strings.IndexByte(line, ':'); i > 0 {
			seen[strings.ToLower(line[:i])] = true
		}
	}
	var additions []string
	if !seen["date"] {
		additions = append(additions, "Date: "+time.Now().UTC().Format(time.RFC1123Z))
	}
	if !seen["message-id"] {
		if hostname == "" {
			hostname = addressDomain(m.From)
		}
		if hostname == "" {
			hostname = "localhost"
		}
		var id [16]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, fmt.Errorf("generate Message-ID: %w", err)
		}
		additions = append(additions, "Message-ID: <"+hex.EncodeToString(id[:])+"@"+hostname+">")
	}
	if !seen["from"] && m.From != "" {
		additions = append(additions, "From: "+m.From)
	}
	if !seen["to"] && m.To != "" {
		additions = append(additions, "To: "+m.To)
	}
	if !seen["subject"] {
		additions = append(additions, "Subject:")
	}
	if len(additions) > 0 && headers != "" {
		headers += "\n"
	}
	headers += strings.Join(additions, "\n")
	return []byte(strings.ReplaceAll(headers+"\n\n"+body, "\n", "\r\n")), nil
}

func addressDomain(address string) string {
	at := strings.LastIndex(address, "@")
	if at >= 0 && at+1 < len(address) {
		return strings.TrimSpace(address[at+1:])
	}
	return ""
}

func (d DirectMXDelivery) Deliver(m Message) error {
	if d.Port == 0 {
		d.Port = 25
	}
	if d.Timeout == 0 {
		d.Timeout = 30 * time.Second
	}
	resolver := d.Resolver
	if resolver == nil {
		resolver = systemMXResolver{}
	}
	dialer := d.Dialer
	if dialer == nil {
		dialer = &net.Dialer{Timeout: d.Timeout}
	}
	at := strings.LastIndex(m.To, "@")
	if at <= 0 || at == len(m.To)-1 || strings.Contains(m.To[at+1:], "@") {
		return errors.New("invalid recipient address")
	}
	domain := m.To[at+1:]
	mx, err := resolver.LookupMX(domain)
	if err != nil {
		return fmt.Errorf("lookup MX for %s: %w", domain, err)
	}
	if len(mx) == 0 {
		return fmt.Errorf("no MX records for %s", domain)
	}
	sort.SliceStable(mx, func(i, j int) bool { return mx[i].Pref < mx[j].Pref })
	from := m.From
	if d.From != "" {
		from = d.From
	}
	messageData, err := normalizeMessage(m, d.Hostname)
	if err != nil {
		return err
	}
	var errs []string
	for _, record := range mx {
		host := strings.TrimSuffix(record.Host, ".")
		conn, dialErr := dialer.Dial("tcp", net.JoinHostPort(host, fmt.Sprint(d.Port)))
		if dialErr != nil {
			errs = append(errs, host+": "+dialErr.Error())
			continue
		}
		_ = conn.SetDeadline(time.Now().Add(d.Timeout))
		client, clientErr := smtp.NewClient(conn, host)
		if clientErr == nil {
			clientErr = client.Mail(from)
			if clientErr == nil {
				clientErr = client.Rcpt(m.To)
			}
			if clientErr == nil {
				w, dataErr := client.Data()
				clientErr = dataErr
				if clientErr == nil {
					_, clientErr = w.Write(messageData)
					if clientErr == nil {
						clientErr = w.Close()
					}
				}
			}
			if clientErr == nil {
				clientErr = client.Quit()
			} else {
				_ = client.Close()
			}
		}
		_ = conn.Close()
		if clientErr == nil {
			return nil
		}
		errs = append(errs, host+": "+clientErr.Error())
	}
	return fmt.Errorf("direct MX delivery failed: %s", strings.Join(errs, "; "))
}
