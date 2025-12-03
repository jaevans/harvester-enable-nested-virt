package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/webhook"
)

var (
	port              int
	certFile          string
	keyFile           string
	configMapName     string
	configMapNamespace string
	kubeconfig        string
)

func init() {
	flag.IntVar(&port, "port", 8443, "Webhook server port")
	flag.StringVar(&certFile, "cert-file", "/etc/webhook/certs/tls.crt", "TLS certificate file")
	flag.StringVar(&keyFile, "key-file", "/etc/webhook/certs/tls.key", "TLS key file")
	flag.StringVar(&configMapName, "configmap-name", "nested-virt-config", "ConfigMap name containing VM matching rules")
	flag.StringVar(&configMapNamespace, "configmap-namespace", "default", "ConfigMap namespace")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (optional, uses in-cluster config if not provided)")
}

func main() {
	flag.Parse()

	// Create Kubernetes client
	k8sConfig, err := getK8sConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Kubernetes config: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Load ConfigMap
	cm, err := loadConfigMap(clientset, configMapNamespace, configMapName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load ConfigMap: %v\n", err)
		os.Exit(1)
	}

	// Parse configuration
	cfg, err := config.ParseConfigMap(cm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse ConfigMap: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded configuration with %d rules\n", len(cfg.Rules))

	// Create mutator
	mutator := mutation.NewVMFeatureMutator(nil)

	// Create webhook handler
	handler := webhook.NewWebhookHandler(cfg, mutator)

	// Create server
	serverCfg := webhook.ServerConfig{
		Port:     port,
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	server := webhook.NewServer(serverCfg, handler)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting webhook server on port %d\n", port)
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

func getK8sConfig() (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func loadConfigMap(clientset *kubernetes.Clientset, namespace, name string) (*corev1.ConfigMap, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, name, err)
	}
	return cm, nil
}
