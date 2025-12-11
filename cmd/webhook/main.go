package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/webhook"

	"log/slog"
)

func init() {
	pflag.String("config", "/etc/webhook/config.yaml", "Path to the configuration file")
	pflag.Int("port", 8443, "Webhook server port")
	pflag.String("cert-dir", "/etc/webhook/certs", "The directory containing TLS certificates (overrides CERT_DIR env var)")
	pflag.Bool("debug", false, "Enable debug logging")
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to bind command line flags: %v\n", err)
		os.Exit(1)
	}
	viper.SetEnvPrefix("nested_virt")
	// Allow environment variables like NESTED_VIRT_CERT_DIR to map to key "cert-dir"
	// Hyphens are replaced with underscores for env var compatibility
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func main() {
	pflag.Parse()
	configFile := viper.GetString("config")

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Merge environment variables and CLI flag overrides with correct precedence.
	cfg = config.MergeWithOverrides(viper.GetViper(), cfg)

	// Set up logging
	var logLevel slog.Level
	if cfg.Debug {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Get the certificate files
	certFile := fmt.Sprintf("%s/tls.crt", cfg.CertDir)
	keyFile := fmt.Sprintf("%s/tls.key", cfg.CertDir)

	logger.Info("Loaded configuration", "rules_count", len(cfg.Rules))

	// Create mutator
	mutator := mutation.NewVMFeatureMutator(nil)

	// Create webhook handler
	handler := webhook.NewWebhookHandler(cfg, mutator)

	// Create server
	serverCfg := webhook.ServerConfig{
		Port:     cfg.Port,
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	server := webhook.NewServer(serverCfg, handler)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting webhook server on port %d\n", cfg.Port)
		if err := server.Start(certFile, keyFile); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down webhook server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Webhook server stopped")
}
