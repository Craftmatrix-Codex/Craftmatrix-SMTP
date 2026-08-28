package smtp

import (
	"context"
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
					_, clientErr = w.Write(m.Data)
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
