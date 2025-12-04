package webhook_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/webhook"
)

var _ = Describe("Server", func() {
	var (
		cfg     *config.Config
		mutator *mutation.VMFeatureMutator
		handler *webhook.WebhookHandler
	)

	BeforeEach(func() {
		cfg = &config.Config{
			Rules: []config.NamespaceRule{},
		}
		detector := &MockCPUFeatureDetector{feature: "vmx"}
		mutator = mutation.NewVMFeatureMutator(detector)
		handler = webhook.NewWebhookHandler(cfg, mutator)
	})

	Describe("NewServer", func() {
		It("should create a new server", func() {
			serverCfg := webhook.ServerConfig{
				Port:     8443,
				CertFile: "/tmp/cert.pem",
				KeyFile:  "/tmp/key.pem",
			}

			server := webhook.NewServer(serverCfg, handler)
			Expect(server).NotTo(BeNil())
		})
	})

	Describe("Shutdown", func() {
		It("should shutdown gracefully", func() {
			serverCfg := webhook.ServerConfig{
				Port:     8443,
				CertFile: "/tmp/cert.pem",
				KeyFile:  "/tmp/key.pem",
			}

			server := webhook.NewServer(serverCfg, handler)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := server.Shutdown(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// Note: We cannot easily test the Start method in a unit test
	// as it requires valid TLS certificates and would block.
	// Integration tests would be needed to fully test the server.
})
