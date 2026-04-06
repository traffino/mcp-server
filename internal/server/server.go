package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
	name    string
	version string
	mcp     *mcp.Server
}

func New(name, version string) *Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: name, Version: version},
		nil,
	)
	return &Server{name: name, version: version, mcp: s}
}

func (s *Server) MCPServer() *mcp.Server {
	return s.mcp
}

func (s *Server) ListenAndServe(addr string) {
	mux := http.NewServeMux()

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return s.mcp
	}, nil)

	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","server":"%s","version":"%s"}`, s.name, s.version)
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("[%s] listening on %s", s.name, addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[%s] server error: %v", s.name, err)
		}
	}()

	<-ctx.Done()
	log.Printf("[%s] shutting down...", s.name)
	srv.Shutdown(context.Background())
}
