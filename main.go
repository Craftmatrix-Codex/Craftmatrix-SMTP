package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Craftmatrix-Codex/Go-Smtp/smtp"
	gosmtp "github.com/emersion/go-smtp"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		resp, err := http.Get("http://127.0.0.1:8080/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return
	}
	cfg, err := smtp.FromEnv()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.QueueDir == "" {
		cfg.QueueDir = "/var/lib/go-smtp/queue"
	}
	queue, err := smtp.NewQueue(cfg.QueueDir)
	if err != nil {
		log.Fatal(err)
	}
	go runDeliveryWorker(queue, cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "Hello World")
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"ok"}`)
	})
	go func() {
		addr := envOr("HTTP_ADDR", ":8080")
		log.Printf("health server listening on %s", addr)
		log.Fatal(http.ListenAndServe(addr, mux))
	}()
	tlsConfig, err := smtp.TLSConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	s := gosmtp.NewServer(smtp.NewBackend(cfg))
	s.Addr = cfg.Addr
	s.Domain = cfg.Hostname
	s.TLSConfig = tlsConfig
	s.AllowInsecureAuth = tlsConfig == nil
	log.Printf("SMTP submission listening on %s", cfg.Addr)
	log.Fatal(s.ListenAndServe())
}
func runDeliveryWorker(queue *smtp.Queue, cfg smtp.Config) {
	var delivery smtp.Delivery
	if cfg.RelayHost != "" {
		delivery = smtp.RelayDelivery{Host: cfg.RelayHost, Port: cfg.RelayPort, Username: cfg.RelayUsername, Password: cfg.RelayPassword}
	} else {
		delivery = smtp.DirectMXDelivery{Port: 25, Timeout: cfg.DeliveryTimeout, Hostname: cfg.Hostname, DKIM: smtp.DKIMConfig{Selector: cfg.DKIMSelector, Domain: cfg.DKIMDomain, PrivateKeyPath: cfg.DKIMPrivateKeyPath, PrivateKey: cfg.DKIMPrivateKey}}
	}
	for {
		if err := (smtp.Worker{Queue: queue, Delivery: delivery}).ProcessOnce(); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Printf("outbound delivery failed: %v", err)
			}
			time.Sleep(time.Second)
		}
	}
}
func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
