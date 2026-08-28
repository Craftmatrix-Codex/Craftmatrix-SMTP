package smtp

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
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

func TestNormalizeMessageAddsRequiredRFC5322Headers(t *testing.T) {
	data, err := normalizeMessage(Message{From: "sender@example.com", To: "user@example.net", Data: []byte("Subject: hi\r\n\r\nhello\r\n")}, "smtp.example.com")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, header := range []string{"Date:", "Message-ID:", "From: sender@example.com", "To: user@example.net", "Subject: hi"} {
		if !strings.Contains(text, header) {
			t.Fatalf("normalized message missing %q: %q", header, text)
		}
	}
}

func TestNormalizeMessagePreservesExistingHeaders(t *testing.T) {
	data, err := normalizeMessage(Message{From: "sender@example.com", To: "user@example.net", Data: []byte("Date: Thu, 01 Jan 1970 00:00:00 +0000\r\nMessage-ID: <existing@example.com>\r\nFrom: existing@example.com\r\nTo: existing@example.net\r\nSubject: existing\r\n\r\nhello\r\n")}, "smtp.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "Message-ID:"); got != 1 {
		t.Fatalf("Message-ID count = %d, want 1", got)
	}
	if !strings.Contains(string(data), "Subject: existing") {
		t.Fatalf("existing headers not preserved: %q", data)
	}
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

func TestDirectMXDeliveryUsesSTARTTLSWhenAdvertised(t *testing.T) {
	cert, err := tls.X509KeyPair([]byte(testCertificate), []byte(testPrivateKey))
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	seenTLS := make(chan bool, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 mx.test ESMTP\r\n")
		line, _ := r.ReadString('\n')
		if strings.HasPrefix(line, "EHLO") {
			fmt.Fprint(conn, "250-mx.test\r\n250-STARTTLS\r\n250 OK\r\n")
		}
		line, _ = r.ReadString('\n')
		if strings.TrimSpace(line) != "STARTTLS" {
			seenTLS <- false
			return
		}
		fmt.Fprint(conn, "220 ready\r\n")
		tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
		if err := tlsConn.Handshake(); err != nil {
			seenTLS <- false
			return
		}
		seenTLS <- true
		r = bufio.NewReader(tlsConn)
		for {
			line, err = r.ReadString('\n')
			if err != nil {
				return
			}
			s := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(s, "EHLO"):
				fmt.Fprint(tlsConn, "250 mx.test\r\n")
			case strings.HasPrefix(s, "MAIL FROM"), strings.HasPrefix(s, "RCPT TO"):
				fmt.Fprint(tlsConn, "250 ok\r\n")
			case s == "DATA":
				fmt.Fprint(tlsConn, "354 end\r\n")
				for {
					line, _ = r.ReadString('\n')
					if strings.TrimSpace(line) == "." {
						break
					}
				}
				fmt.Fprint(tlsConn, "250 queued\r\n")
			case s == "QUIT":
				fmt.Fprint(tlsConn, "221 bye\r\n")
				return
			}
		}
	}()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(testCertificate))
	d := DirectMXDelivery{Resolver: staticMXResolver{hosts: []string{"127.0.0.1"}}, Port: ln.Addr().(*net.TCPAddr).Port, Timeout: time.Second, TLSConfig: &tls.Config{RootCAs: pool}}
	if err := d.Deliver(Message{From: "sender@example.com", To: "user@example.net", Data: []byte("Subject: hi\r\n\r\nhello\r\n")}); err != nil {
		t.Fatal(err)
	}
	if !<-seenTLS {
		t.Fatal("server did not observe STARTTLS")
	}
}

const testCertificate = `-----BEGIN CERTIFICATE-----
MIIDGjCCAgKgAwIBAgIUS+nyZOpgQ5xBQKi1I6nMaloRBjcwDQYJKoZIhvcNAQEL
BQAwFDESMBAGA1UEAwwJMTI3LjAuMC4xMB4XDTI2MDgyODA5MzIwMVoXDTI2MDgy
OTA5MzIwMVowFDESMBAGA1UEAwwJMTI3LjAuMC4xMIIBIjANBgkqhkiG9w0BAQEF
AAOCAQ8AMIIBCgKCAQEArgkjr0JBkrzmFAKUsmjBP74Py4SHaMITJVIk4W1Cyjt6
+IeFva5zZJ5FV2j1WaXSBuX3QIjPHGjlULZ5bTiXzhrfoFIynbY9cllr54KJvk+7
xkxxx//5nx0d/pzCtU0bOxV/ibjh2IudoCbPeLtM6DEF3b1M5s7mPshSIoIyiLE3
n4yLhPW4LukWTp9/8fcnXCEdjY4iGzfs+T5SepfTyYZLtv2izeKSK9YNo9dAIiwN
zgawwcY+ZsoQAS5wmATXJFGw/x4DcEUKP1X/BHaxoZsUyoN13LrK8N8UFt9KZRO6
eiN7BAfGW8rjOejSRTlR+FVAxa+UzhungTAE8b4IhwIDAQABo2QwYjAdBgNVHQ4E
FgQUGnV+ML69zdWiS52pwyhxj9mt+tgwHwYDVR0jBBgwFoAUGnV+ML69zdWiS52p
wyhxj9mt+tgwDwYDVR0TAQH/BAUwAwEB/zAPBgNVHREECDAGhwR/AAABMA0GCSqG
SIb3DQEBCwUAA4IBAQBjlDecvTXYxLjQHOE6f6LMfjnBkX6Ta8IyA7k3LetAgHvG
LbNc/FXW8JBKOeMbJn6PAh2hDNCTbU6T5XuqLeml7JFWXf+krB/cUJRlqg25xFhV
kvXLJdhBizUmkLhUl6m5+hTqPaWsmUov1MO9ZIjdzBJE2/rAvaFdn+BSR1hxYeM0
cXOzwebtQXBKUWMiOGHwztEqhfx8nK3ItqcE93KYLXeLKTJLlVGWp3cAGhJoBDPr
S9Mnxyvnu5L72UJrRwNxa3YNBydkXIwVY6imE5YMSW5XPpt9/fkkBactAjxrnAdQ
Q9deVcw0yJvRu/2KFds5BEn4thxGbkLfDZmqRQ38
-----END CERTIFICATE-----`
const testPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCuCSOvQkGSvOYU
ApSyaME/vg/LhIdowhMlUiThbULKO3r4h4W9rnNknkVXaPVZpdIG5fdAiM8caOVQ
tnltOJfOGt+gUjKdtj1yWWvngom+T7vGTHHH//mfHR3+nMK1TRs7FX+JuOHYi52g
Js94u0zoMQXdvUzmzuY+yFIigjKIsTefjIuE9bgu6RZOn3/x9ydcIR2NjiIbN+z5
PlJ6l9PJhku2/aLN4pIr1g2j10AiLA3OBrDBxj5myhABLnCYBNckUbD/HgNwRQo/
Vf8EdrGhmxTKg3Xcusrw3xQW30plE7p6I3sEB8ZbyuM56NJFOVH4VUDFr5TOG6eB
MATxvgiHAgMBAAECggEANefvzSvVGsn27BDGlSJcZU2OH+hLdvfSLVetw8Nvkitt
UgjMNK4R4QMjEEma4Ws6zFczyCH4IOEe0mrK3rlOUBFD3ilHs1EM5FvtT9a/qpqQ
24G3Jo9TduTycviEWNrqydjFWdfR4JySNNaAofFGB4+TtRSa+szwRVcQFc9FLtBN
e4qYpnUPMnml5/LykVxJ1OA7Sytz+HdQqLRWh6c5i1x3CvUbkf+nz4q3dt8jawA/
j6QwBdI1D5EPVBsJ6e5ddIbOK1cKW3jl68dRdAGo9gRJB2plF1gjHZ3gfE2O86gT
C4uGKeJNDEEYK2syfLCamNGljuWPHJItmW/9CzLdmQKBgQDz+XwOGFvsh8xPU9M+
xLGhQp7XLoR3ZSrk3/7+Se98R/srC5fF1xfV1TkqZ38W2/Rwq8vuQI+onb0smX6u
/ZhyCoKbjCdzRuNTyK3q/Af7XoRBHWt1gefpBOExgk0oBi8qz1VowoVruaJjcevK
pTHT/f9Q0aLBX+pqT1ghQuQD3wKBgQC2nSdVJlw+r1fX07Ln7h+BTip68SqUhdj1
aJ2c+BfnYrJMjqr1ksbAwym3XESpJko++VWx89bf0M3/cZ6XnWIua5Um6ecMhr0h
3tPzE28sNKI5Pmm4ttG3a0hO+I0sxA8iRs9v3p6MD+eo7mQaFTxy8bALpzV6UyGn
483u4R1QWQKBgE7V6jE/y8xloS5s/tDEjEV4mo0b2fAev7qPav1OdNVrCQ1CxLtI
IsqVCNDb+qQvVRBnYxFMyV8KAYv82YPALFeFb+jFZCYK3QBA03ogEJA4XXIRCJ1C
6eZRDleKLFZnkSw7LPUaDjTeGkwaDsA/mxdOdwbthrMHxF6v3uF4lOdvAoGBALYb
VB0DCxxr6XLOp2vIuFxPYeeGVosUSz632+2sLtJNRzc3Ut9gRpn2RcSX29S+3W2d
Ycr7On2qEbO6T4gsp7tZB71tpj6Eo1mCh+Swrb3soxXo2q8ciVibQNmX3dkVj24E
JVsPKxbLyZ5aVTL5mHWb9Y45aggZnMd7UmL01THJAoGBAMa+NEKZXqdRZrq1jMyL
BPxV2tsjhSlnkquwRlqt5m9IMJtjSY0SjKqWgC5r/NNOf5PjmxFI9PBoRwRPxANp
+UvvZS9aeyytkfoYjqqq4UlrLHTDK2fOG3k2+bjuCAoPkoE/JT1X1yS0cgpocB//
VxohMG+cPgNNFie2f66Az9gL
-----END PRIVATE KEY-----`
