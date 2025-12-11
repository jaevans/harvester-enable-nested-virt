package config_test

import (
	"log/slog"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
)

var _ = BeforeSuite(func() {
	l := slog.New(slog.NewTextHandler(GinkgoWriter, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(l)
})

var _ = Describe("Config", func() {
	Describe("LoadConfig", func() {
		It("should load configuration from a valid file", func() {
			tmpFile := GinkgoT().TempDir() + "/config.yaml"
			configContent := `
port: 1234
`
			err := os.WriteFile(tmpFile, []byte(configContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := config.LoadConfig(tmpFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Port).To(Equal(1234))
		})

		It("should return an error for a non-existent file", func() {
			_, err := config.LoadConfig("non_existent_file.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("should return an error for invalid YAML content", func() {
			tmpFile := GinkgoT().TempDir() + "/invalid_config.yaml"
			invalidContent := `
port: not_a_number
`
			err := os.WriteFile(tmpFile, []byte(invalidContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = config.LoadConfig(tmpFile)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetParsedRules", func() {

		It("should compile regex patterns correctly", func() {
			cfg := &config.Config{
				Rules: []config.NamespaceRuleConfig{
					{
						Namespace: "test-namespace",
						Patterns:  []string{"^vm-.*", "^test-.*"},
					},
				},
			}

			parsedRules := cfg.GetParsedRules()
			Expect(parsedRules).To(HaveLen(1))
			Expect(parsedRules[0].Namespace).To(Equal("test-namespace"))
			Expect(parsedRules[0].Patterns).To(HaveLen(2)) // Two valid regexes
		})

		It("should handle invalid regex patterns gracefully", func() {
			cfg := &config.Config{
				Rules: []config.NamespaceRuleConfig{
					{
						Namespace: "test-namespace",
						Patterns:  []string{"^vm-[.*", "^test-.*"},
					},
				},
			}

			parsedRules := cfg.GetParsedRules()
			Expect(parsedRules).To(HaveLen(1))
			Expect(parsedRules[0].Namespace).To(Equal("test-namespace"))
			Expect(parsedRules[0].Patterns).To(HaveLen(1)) // One valid regex, one invalid skipped
		})

		It("should return cached parsed rules on subsequent calls", func() {
			cfg := &config.Config{
				Rules: []config.NamespaceRuleConfig{
					{
						Namespace: "test-namespace",
						Patterns:  []string{"^vm-.*"},
					},
				},
			}

			firstCall := cfg.GetParsedRules()
			secondCall := cfg.GetParsedRules()
			// Verify the second call returns the exact same slice header (same underlying array)
			Expect(&secondCall[0]).To(BeIdenticalTo(&firstCall[0]))
		})

		It("should compile with no rules if no rules are provided", func() {
			cfg := &config.Config{}

			parsedRules := cfg.GetParsedRules()
			Expect(parsedRules).To(HaveLen(0))
		})

		It("should compile multiple namespaces and patterns", func() {
			cfg := &config.Config{
				Rules: []config.NamespaceRuleConfig{
					{
						Namespace: "namespace1",
						Patterns:  []string{"^vm-.*", "^test-.*"},
					},
					{
						Namespace: "namespace2",
						Patterns:  []string{".*-prod$"},
					},
				},
			}

			parsedRules := cfg.GetParsedRules()
			Expect(parsedRules).To(HaveLen(2))

			Expect(parsedRules[0].Namespace).To(Equal("namespace1"))
			Expect(parsedRules[0].Patterns).To(HaveLen(2))

			Expect(parsedRules[1].Namespace).To(Equal("namespace2"))
			Expect(parsedRules[1].Patterns).To(HaveLen(1))
		})
	})

	Describe("Matches", func() {
		It("should match a VM in a namespace", func() {
			cfg := &config.Config{
				Rules: []config.NamespaceRuleConfig{
					{
						Namespace: "test-namespace",
						Patterns:  []string{"^vm-.*"},
					},
				},
			}

			Expect(cfg.Matches("test-namespace", "vm-1234")).To(BeTrue())
			Expect(cfg.Matches("test-namespace", "not-a-vm")).To(BeFalse())
			Expect(cfg.Matches("other-namespace", "vm-1234")).To(BeFalse())
		})

		It("should return false if config is nil", func() {
			var cfg *config.Config = nil
			Expect(cfg.Matches("any-namespace", "any-vm")).To(BeFalse())
		})

		It("should handle multiple rules and patterns", func() {
			cfg := &config.Config{
				Rules: []config.NamespaceRuleConfig{
					{
						Namespace: "namespace1",
						Patterns:  []string{"^vm-.*", "^test-.*"},
					},
					{
						Namespace: "namespace2",
						Patterns:  []string{".*-prod$"},
					},
				},
			}

			Expect(cfg.Matches("namespace1", "vm-001")).To(BeTrue())
			Expect(cfg.Matches("namespace1", "test-abc")).To(BeTrue())
			Expect(cfg.Matches("namespace1", "other-vm")).To(BeFalse())

			Expect(cfg.Matches("namespace2", "my-vm-prod")).To(BeTrue())
			Expect(cfg.Matches("namespace2", "my-vm-dev")).To(BeFalse())

			Expect(cfg.Matches("unknown-namespace", "vm-001")).To(BeFalse())
		})
	})
})
