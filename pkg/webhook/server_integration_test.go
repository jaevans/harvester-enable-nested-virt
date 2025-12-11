package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
)

var _ = Describe("Server Integration", func() {
	var (
		cfg         *config.Config
		mutator     *mutation.VMFeatureMutator
		handler     *WebhookHandler
		certFile    string
		keyFile     string
		httpClient  *http.Client
		port        int
		testDataDir string
	)

	BeforeEach(func() {
		// Setup config and handler
		cfg = &config.Config{
			Rules: []config.NamespaceRuleConfig{
				{
					Namespace: "test-namespace",
					Patterns:  []string{"^vm-.*"},
				},
			},
		}
		detector := &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
		mutator = mutation.NewVMFeatureMutator(detector)
		handler = NewWebhookHandler(cfg, mutator)

		// Get path to test certificates
		testDataDir = filepath.Join(os.Getenv("PWD"), "pkg", "webhook", "testdata")
		if _, err := os.Stat(testDataDir); os.IsNotExist(err) {
			// Try relative to current directory
			testDataDir = filepath.Join(".", "testdata")
		}

		certFile = filepath.Join(testDataDir, "test-cert.pem")
		keyFile = filepath.Join(testDataDir, "test-key.pem")

		// Skip test if certificates don't exist
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			Skip("test certificates not found")
		}

		// Find an available port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// Setup HTTP client with TLS skip verification for self-signed cert
		httpClient = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec
				},
			},
		}
	})

	Describe("Start", func() {
		It("should start the server and accept HTTPS connections", func() {
			server := NewServer(ServerConfig{Port: port}, handler)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
			}()

			// Start server in a goroutine
			errCh := make(chan error, 1)
			go func() {
				errCh <- server.Start(certFile, keyFile)
			}()

			// Give server time to start
			time.Sleep(200 * time.Millisecond)

			// Make a health check request
			url := fmt.Sprintf("https://localhost:%d/healthz", port)
			resp, err := httpClient.Get(url)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(Equal("OK"))
			resp.Body.Close()
		})

		It("should handle admission webhook requests", func() {
			server := NewServer(ServerConfig{Port: port}, handler)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
			}()

			errCh := make(chan error, 1)
			go func() {
				errCh <- server.Start(certFile, keyFile)
			}()

			time.Sleep(200 * time.Millisecond)

			// Create a test VM that matches the config
			vm := &kubevirtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-test-123",
					Namespace: "test-namespace",
				},
				Spec: kubevirtv1.VirtualMachineSpec{
					Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
						Spec: kubevirtv1.VirtualMachineInstanceSpec{
							Domain: kubevirtv1.DomainSpec{
								CPU: &kubevirtv1.CPU{},
							},
						},
					},
				},
			}

			vmBytes, err := json.Marshal(vm)
			Expect(err).NotTo(HaveOccurred())

			// Create admission review
			admissionReview := &admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "admission.k8s.io/v1",
					Kind:       "AdmissionReview",
				},
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid-123",
					Kind: metav1.GroupVersionKind{
						Group:   "kubevirt.io",
						Version: "v1",
						Kind:    "VirtualMachine",
					},
					Namespace: "test-namespace",
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: vmBytes,
					},
				},
			}

			reviewBytes, err := json.Marshal(admissionReview)
			Expect(err).NotTo(HaveOccurred())

			// Make request to mutation endpoint
			url := fmt.Sprintf("https://localhost:%d/mutate", port)
			resp, err := httpClient.Post(url, "application/json", bytes.NewReader(reviewBytes))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var responseReview admissionv1.AdmissionReview
			err = json.NewDecoder(resp.Body).Decode(&responseReview)
			resp.Body.Close()
			Expect(err).NotTo(HaveOccurred())

			// Verify response
			Expect(responseReview.Response).NotTo(BeNil())
			Expect(string(responseReview.Response.UID)).To(Equal("test-uid-123"))
			Expect(responseReview.Response.Allowed).To(BeTrue())
			Expect(responseReview.Response.Patch).NotTo(BeNil())
		})

		It("should reject requests with invalid method", func() {
			server := NewServer(ServerConfig{Port: port}, handler)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
			}()

			errCh := make(chan error, 1)
			go func() {
				errCh <- server.Start(certFile, keyFile)
			}()

			time.Sleep(200 * time.Millisecond)

			url := fmt.Sprintf("https://localhost:%d/mutate", port)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			Expect(err).NotTo(HaveOccurred())

			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
			resp.Body.Close()
		})

		It("should return error when certificate files are invalid", func() {
			server := NewServer(ServerConfig{Port: port}, handler)
			invalidCert := "/nonexistent/cert.pem"
			invalidKey := "/nonexistent/key.pem"

			err := server.Start(invalidCert, invalidKey)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load TLS certificates"))
		})

		It("should return error when certificate is malformed", func() {
			server := NewServer(ServerConfig{Port: port}, handler)
			malformedCert := filepath.Join(testDataDir, "malformed-cert.pem")

			err := server.Start(malformedCert, keyFile)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load TLS certificates"))
		})

		It("should return error when port is already in use", func() {
			// Start first server on the chosen port
			server1 := NewServer(ServerConfig{Port: port}, handler)
			errCh1 := make(chan error, 1)
			go func() {
				errCh1 <- server1.Start(certFile, keyFile)
			}()

			time.Sleep(200 * time.Millisecond)

			// Try to start second server on the same port
			server2 := NewServer(ServerConfig{Port: port}, handler)
			err := server2.Start(certFile, keyFile)

			// Should get "address already in use" or similar error
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bind"))

			// Cleanup first server
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server1.Shutdown(ctx)
		})
	})
})
