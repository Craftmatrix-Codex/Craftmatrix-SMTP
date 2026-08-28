package smtp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

type staticMXResolver struct{ hosts []string }

func (r staticMXResolver) LookupMX(string) ([]*net.MX, error) {
	var out []*net.MX
	for i, h := range r.hosts {
		out = append(out, &net.MX{Host: h, Pref: uint16(i)})
	}
	return out, nil
}

func TestDirectMXDeliveryResolvesMXAndSendsMessage(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	got := make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		w := func(s string) { _, _ = fmt.Fprintf(conn, "%s\r\n", s) }
		w("220 mx.test ESMTP")
		r := bufio.NewReader(conn)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			s := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(s, "EHLO"):
				w("250 mx.test")
			case strings.HasPrefix(s, "MAIL FROM"):
				w("250 ok")
			case strings.HasPrefix(s, "RCPT TO"):
				w("250 ok")
			case s == "DATA":
				w("354 end")
				var b strings.Builder
				for {
					line, _ = r.ReadString('\n')
					if strings.TrimSpace(line) == "." {
						break
					}
					b.WriteString(line)
				}
				got <- b.String()
				w("250 queued")
			case s == "QUIT":
				w("221 bye")
				return
			}
		}
	}()

	d := DirectMXDelivery{Resolver: staticMXResolver{hosts: []string{strings.Split(ln.Addr().String(), ":")[0]}}, Port: ln.Addr().(*net.TCPAddr).Port, Timeout: time.Second}
	if err := d.Deliver(Message{From: "sender@example.com", To: "user@example.net", Data: []byte("Subject: hi\r\n\r\nhello\r\n")}); err != nil {
		t.Fatal(err)
	}
	select {
	case body := <-got:
		if !strings.Contains(body, "hello") {
			t.Fatalf("body = %q", body)
		}
	case <-time.After(time.Second):
		t.Fatal("message was not received")
	}
}
