package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/traffino/mcp-server/internal/config"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	apiToken := config.Require("CLOUDFLARE_API_TOKEN")
	targetURL := config.Get("CLOUDFLARE_MCP_URL", "https://mcp.cloudflare.com/mcp")
	addr := config.Get("PORT", ":8000")

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","target":"%s"}`, targetURL)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `{"error":"bad_request"}`, http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
		if err != nil {
			http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
			return
		}

		upstream.Header.Set("Content-Type", coalesce(r.Header.Get("Content-Type"), "application/json"))
		upstream.Header.Set("Accept", coalesce(r.Header.Get("Accept"), "application/json, text/event-stream"))
		upstream.Header.Set("Authorization", "Bearer "+apiToken)
		if sid := r.Header.Get("Mcp-Session-Id"); sid != "" {
			upstream.Header.Set("Mcp-Session-Id", sid)
		}

		resp, err := httpClient.Do(upstream)
		if err != nil {
			log.Printf("[proxy] upstream error: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"bad_gateway","message":"%s"}`, err.Error()), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if ct := resp.Header.Get("Content-Type"); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			w.Header().Set("Mcp-Session-Id", sid)
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	log.Printf("[cloudflare-proxy] listening on %s -> %s", addr, targetURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
