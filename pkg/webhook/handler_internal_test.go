package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
)

// MockCPUFeatureDetector for testing
type MockCPUFeatureDetector struct {
	feature mutation.CPUFeature
	err     error
}

func (m *MockCPUFeatureDetector) DetectFeature() (mutation.CPUFeature, error) {
	return m.feature, m.err
}

var _ = Describe("Handler internal methods", func() {
	var (
		handler  *WebhookHandler
		cfg      *config.Config
		mutator  *mutation.VMFeatureMutator
		detector *MockCPUFeatureDetector
	)

	BeforeEach(func() {
		detector = &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
		mutator = mutation.NewVMFeatureMutator(detector)

		cfg = &config.Config{
			Rules: []config.NamespaceRuleConfig{
				{
					Namespace: "test-namespace",
					Patterns:  []string{"^vm-.*"},
				},
			},
		}
		handler = NewWebhookHandler(cfg, mutator)
	})

	Describe("createJSONPatch", func() {
		Context("when JSON parsing fails", func() {
			DescribeTable("should return error for invalid JSON",
				func(original, mutated []byte) {
					patch, err := createJSONPatch(original, mutated)
					Expect(err).To(HaveOccurred())
					Expect(patch).To(BeNil())
				},
				Entry("invalid original JSON", []byte("{invalid"), []byte(`{}`)),
				Entry("invalid mutated JSON", []byte(`{}`), []byte("{invalid")),
				Entry("both invalid JSON", []byte("{invalid"), []byte("{invalid")),
			)
		})

		Context("when documents are identical", func() {
			It("returns nil patch and no error", func() {
				vm := map[string]interface{}{"spec": map[string]interface{}{}}
				original, err := json.Marshal(vm)
				Expect(err).NotTo(HaveOccurred())

				patch, err := createJSONPatch(original, original)
				Expect(err).NotTo(HaveOccurred())
				Expect(patch).To(BeNil())
			})
		})

		Context("when VM structure is missing expected maps", func() {
			DescribeTable("should return nil patch for malformed structures",
				func(mutated []byte) {
					patch, err := createJSONPatch([]byte(`{}`), mutated)
					Expect(err).NotTo(HaveOccurred())
					Expect(patch).To(BeNil())
				},
				Entry("spec is not a map", []byte(`{"spec":"invalid"}`)),
				Entry("template is not a map", []byte(`{"spec":{"template":"invalid"}}`)),
				Entry("template spec is not a map", []byte(`{"spec":{"template":{"spec":"invalid"}}}`)),
				Entry("domain is not a map", []byte(`{"spec":{"template":{"spec":{"domain":"invalid"}}}}`)),
				Entry("cpu is not a map", []byte(`{"spec":{"template":{"spec":{"domain":{"cpu":"invalid"}}}}}`)),
				Entry("features are missing", []byte(`{"spec":{"template":{"spec":{"domain":{"cpu":{}}}}}}`)),
			)
		})

		Context("when VM has CPU but no features", func() {
			It("should generate an add patch for features", func() {
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

				admissionReview := &admissionv1.AdmissionReview{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "admission.k8s.io/v1",
						Kind:       "AdmissionReview",
					},
					Request: &admissionv1.AdmissionRequest{
						UID: "test-uid",
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

				req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(reviewBytes))
				w := httptest.NewRecorder()

				handler.Handle(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var responseReview admissionv1.AdmissionReview
				err = json.Unmarshal(w.Body.Bytes(), &responseReview)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseReview.Response).NotTo(BeNil())
				Expect(responseReview.Response.Allowed).To(BeTrue())
				Expect(responseReview.Response.Patch).NotTo(BeNil())

				// Verify the patch is an add operation for features
				var patches []map[string]interface{}
				err = json.Unmarshal(responseReview.Response.Patch, &patches)
				Expect(err).NotTo(HaveOccurred())
				Expect(patches).NotTo(BeEmpty())
				Expect(patches[0]["op"]).To(Equal("add"))
				Expect(patches[0]["path"]).To(Equal("/spec/template/spec/domain/cpu/features"))
			})
		})

		Context("when VM has CPU with existing features", func() {
			It("should generate a replace patch for features", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm-test-123",
						Namespace: "test-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									CPU: &kubevirtv1.CPU{
										Features: []kubevirtv1.CPUFeature{
											{
												Name:   "test-feature",
												Policy: "require",
											},
										},
									},
								},
							},
						},
					},
				}

				vmBytes, err := json.Marshal(vm)
				Expect(err).NotTo(HaveOccurred())

				admissionReview := &admissionv1.AdmissionReview{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "admission.k8s.io/v1",
						Kind:       "AdmissionReview",
					},
					Request: &admissionv1.AdmissionRequest{
						UID: "test-uid",
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

				req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(reviewBytes))
				w := httptest.NewRecorder()

				handler.Handle(w, req)

				Expect(w.Code).To(Equal(http.StatusOK))

				var responseReview admissionv1.AdmissionReview
				err = json.Unmarshal(w.Body.Bytes(), &responseReview)
				Expect(err).NotTo(HaveOccurred())
				Expect(responseReview.Response).NotTo(BeNil())
				Expect(responseReview.Response.Allowed).To(BeTrue())
				Expect(responseReview.Response.Patch).NotTo(BeNil())

				// Verify the patch is a replace operation for features
				var patches []map[string]interface{}
				err = json.Unmarshal(responseReview.Response.Patch, &patches)
				Expect(err).NotTo(HaveOccurred())
				Expect(patches).NotTo(BeEmpty())
				Expect(patches[0]["op"]).To(Equal("replace"))
				Expect(patches[0]["path"]).To(Equal("/spec/template/spec/domain/cpu/features"))
			})
		})
	})

	Describe("mutate", func() {
		Context("when VirtualMachine decoding fails", func() {
			It("should return error response with failed to decode message", func() {
				req := &admissionv1.AdmissionRequest{
					UID: "test-uid",
					Kind: metav1.GroupVersionKind{
						Group:   "kubevirt.io",
						Version: "v1",
						Kind:    "VirtualMachine",
					},
					Namespace: "test-namespace",
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: []byte("{invalid json}"),
					},
				}

				response := handler.mutate(req)

				Expect(response).NotTo(BeNil())
				Expect(response.Allowed).To(BeTrue())
				Expect(response.Result).NotTo(BeNil())
				Expect(response.Result.Message).To(ContainSubstring("failed to decode VirtualMachine"))
				Expect(response.Patch).To(BeNil())
			})
		})

		Context("when mutator fails", func() {
			BeforeEach(func() {
				// Replace mutator with one that will fail
				failingDetector := &MockCPUFeatureDetector{
					err: fmt.Errorf("CPU detection failed"),
				}
				handler.mutator = mutation.NewVMFeatureMutator(failingDetector)
			})

			It("should return error response with failed to mutate message", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm-test-123",
						Namespace: "test-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{},
				}

				vmBytes, err := json.Marshal(vm)
				Expect(err).NotTo(HaveOccurred())

				req := &admissionv1.AdmissionRequest{
					UID: "test-uid",
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
				}

				response := handler.mutate(req)

				Expect(response).NotTo(BeNil())
				Expect(response.Allowed).To(BeTrue())
				Expect(response.Result).NotTo(BeNil())
				Expect(response.Result.Message).To(ContainSubstring("failed to mutate VirtualMachine"))
				Expect(response.Patch).To(BeNil())
			})
		})

		Context("when VM does not match config rules", func() {
			It("should return allowed response without modification", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "no-match",
						Namespace: "test-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{},
				}

				vmBytes, err := json.Marshal(vm)
				Expect(err).NotTo(HaveOccurred())

				req := &admissionv1.AdmissionRequest{
					UID: "test-uid",
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
				}

				response := handler.mutate(req)

				Expect(response).NotTo(BeNil())
				Expect(response.Allowed).To(BeTrue())
				Expect(response.Result).To(BeNil())
				Expect(response.Patch).To(BeNil())
			})
		})
	})
})
