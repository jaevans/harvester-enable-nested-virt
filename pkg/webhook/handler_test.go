package webhook_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/webhook"
)

// MockCPUFeatureDetector for testing
type MockCPUFeatureDetector struct {
	feature string
	err     error
}

func (m *MockCPUFeatureDetector) DetectFeature() (string, error) {
	return m.feature, m.err
}

var _ = Describe("WebhookHandler", func() {
	var (
		handler  *webhook.WebhookHandler
		cfg      *config.Config
		mutator  *mutation.VMFeatureMutator
		detector *MockCPUFeatureDetector
	)

	BeforeEach(func() {
		detector = &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
		mutator = mutation.NewVMFeatureMutator(detector)
		cfg = &config.Config{
			Rules: []config.NamespaceRule{
				{
					Namespace: "test-namespace",
					Patterns:  []*regexp.Regexp{regexp.MustCompile("^vm-.*")},
				},
			},
		}
		handler = webhook.NewWebhookHandler(cfg, mutator)
	})

	Describe("Handle", func() {
		Context("when receiving a valid admission request", func() {
			It("should mutate VM that matches the config", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm-test-123",
						Namespace: "test-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{},
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
			})

			It("should not mutate VM that does not match the config", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-123",
						Namespace: "test-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{},
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
				Expect(responseReview.Response.Patch).To(BeNil())
			})

			It("should not mutate VM in different namespace", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm-test-123",
						Namespace: "other-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{},
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
						Namespace: "other-namespace",
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
				Expect(responseReview.Response.Patch).To(BeNil())
			})
		})

		Context("when receiving invalid requests", func() {
			It("should reject non-POST requests", func() {
				req := httptest.NewRequest(http.MethodGet, "/mutate", nil)
				w := httptest.NewRecorder()

				handler.Handle(w, req)

				Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			})

			It("should return error for invalid admission review", func() {
				req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader([]byte("invalid json")))
				w := httptest.NewRecorder()

				handler.Handle(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})

			It("should return error for admission review with nil request", func() {
				admissionReview := &admissionv1.AdmissionReview{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "admission.k8s.io/v1",
						Kind:       "AdmissionReview",
					},
					Request: nil,
				}

				reviewBytes, err := json.Marshal(admissionReview)
				Expect(err).NotTo(HaveOccurred())

				req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(reviewBytes))
				w := httptest.NewRecorder()

				handler.Handle(w, req)

				Expect(w.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("when CPU feature detection fails", func() {
			BeforeEach(func() {
				detector.err = fmt.Errorf("detection failed")
			})

			It("should allow but not mutate the VM", func() {
				vm := &kubevirtv1.VirtualMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "vm-test-123",
						Namespace: "test-namespace",
					},
					Spec: kubevirtv1.VirtualMachineSpec{},
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
			})
		})
	})

	Describe("NewWebhookHandler", func() {
		It("should create a new webhook handler", func() {
			handler := webhook.NewWebhookHandler(cfg, mutator)
			Expect(handler).NotTo(BeNil())
		})
	})
})
