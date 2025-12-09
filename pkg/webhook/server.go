package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Server represents the webhook server
type Server struct {
	server  *http.Server
	handler *WebhookHandler
}

// ServerConfig holds configuration for the webhook server
type ServerConfig struct {
	Port     int
	CertFile string
	KeyFile  string
}

// NewServer creates a new webhook server
func NewServer(cfg ServerConfig, handler *WebhookHandler) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", handler.Handle)
	mux.HandleFunc("/healthz", healthzHandler)

	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
		},
		handler: handler,
	}
}

// Start starts the webhook server with TLS
func (s *Server) Start(certFile, keyFile string) error {
	// Load TLS certificates
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	s.server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return s.server.ListenAndServeTLS("", "")
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// healthzHandler handles health check requests
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}
