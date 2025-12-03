package config_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
)

var _ = Describe("Config", func() {
	Describe("ParseConfigMap", func() {
		Context("when given a valid ConfigMap", func() {
			It("should parse single namespace with single pattern", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"namespace1": "^vm-.*",
					},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Rules).To(HaveLen(1))
				Expect(cfg.Rules[0].Namespace).To(Equal("namespace1"))
				Expect(cfg.Rules[0].Patterns).To(HaveLen(1))
			})

			It("should parse single namespace with multiple patterns", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"namespace1": "^vm-.*,^test-.*,^prod-.*",
					},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Rules).To(HaveLen(1))
				Expect(cfg.Rules[0].Namespace).To(Equal("namespace1"))
				Expect(cfg.Rules[0].Patterns).To(HaveLen(3))
			})

			It("should parse multiple namespaces", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"namespace1": "^vm-.*",
						"namespace2": "^test-.*",
						"namespace3": ".*-prod$",
					},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Rules).To(HaveLen(3))
			})

			It("should handle patterns with spaces", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"namespace1": "^vm-.* , ^test-.* , ^prod-.*",
					},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Rules).To(HaveLen(1))
				Expect(cfg.Rules[0].Patterns).To(HaveLen(3))
			})

			It("should skip empty patterns", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"namespace1": "^vm-.*,,,^test-.*",
						"namespace2": "",
					},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Rules).To(HaveLen(1))
				Expect(cfg.Rules[0].Namespace).To(Equal("namespace1"))
				Expect(cfg.Rules[0].Patterns).To(HaveLen(2))
			})
		})

		Context("when given invalid input", func() {
			It("should return error for nil ConfigMap", func() {
				cfg, err := config.ParseConfigMap(nil)
				Expect(err).To(HaveOccurred())
				Expect(cfg).To(BeNil())
			})

			It("should return error for invalid regex pattern", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"namespace1": "^vm-[.*",
					},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).To(HaveOccurred())
				Expect(cfg).To(BeNil())
			})
		})

		Context("when ConfigMap is empty", func() {
			It("should return empty config", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-config",
						Namespace: "default",
					},
					Data: map[string]string{},
				}

				cfg, err := config.ParseConfigMap(cm)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Rules).To(HaveLen(0))
			})
		})
	})

	Describe("Matches", func() {
		var cfg *config.Config

		BeforeEach(func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-config",
					Namespace: "default",
				},
				Data: map[string]string{
					"namespace1": "^vm-.*,^test-.*",
					"namespace2": ".*-prod$",
				},
			}

			var err error
			cfg, err = config.ParseConfigMap(cm)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when VM matches a rule", func() {
			It("should return true for matching VM name in namespace1", func() {
				result := cfg.Matches("namespace1", "vm-test-123")
				Expect(result).To(BeTrue())
			})

			It("should return true for matching VM name with test prefix", func() {
				result := cfg.Matches("namespace1", "test-vm")
				Expect(result).To(BeTrue())
			})

			It("should return true for matching VM name in namespace2", func() {
				result := cfg.Matches("namespace2", "my-vm-prod")
				Expect(result).To(BeTrue())
			})
		})

		Context("when VM does not match any rule", func() {
			It("should return false for non-matching VM name", func() {
				result := cfg.Matches("namespace1", "prod-vm")
				Expect(result).To(BeFalse())
			})

			It("should return false for non-existent namespace", func() {
				result := cfg.Matches("namespace3", "vm-test")
				Expect(result).To(BeFalse())
			})

			It("should return false for partially matching name", func() {
				result := cfg.Matches("namespace2", "prod-vm-test")
				Expect(result).To(BeFalse())
			})
		})

		Context("when config is nil", func() {
			It("should return false", func() {
				var nilCfg *config.Config
				result := nilCfg.Matches("namespace1", "vm-test")
				Expect(result).To(BeFalse())
			})
		})
	})
})
