package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
	// pflag.StringVar(&configMapName, "configmap-name", "nested-virt-config", "ConfigMap name containing VM matching rules")
	// pflag.StringVar(&configMapNamespace, "configmap-namespace", "default", "ConfigMap namespace")
	// pflag.String("kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if not provided)")
	pflag.Bool("debug", false, "Enable debug logging")
	viper.BindPFlags(pflag.CommandLine)
	viper.SetEnvPrefix("nested_virt")
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

	// Create Kubernetes client
	// k8sConfig, err := getK8sConfig()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Failed to get Kubernetes config: %v\n", err)
	// 	os.Exit(1)
	// }

	// clientset, err := kubernetes.NewForConfig(k8sConfig)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
	// 	os.Exit(1)
	// }

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
