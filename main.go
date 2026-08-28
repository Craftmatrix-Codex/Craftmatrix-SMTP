package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Craftmatrix-Codex/Go-Smtp/smtp"
	gosmtp "github.com/emersion/go-smtp"
)

func main() {
	cfg, err := smtp.FromEnv()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
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
func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
