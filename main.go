package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    addr := os.Getenv("HTTP_ADDR")
    if addr == "" {
        addr = ":8080"
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = fmt.Fprint(w, `{"status":"ok"}`)
    })

    log.Printf("health server listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, mux))
}
