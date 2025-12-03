package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/config"
	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

func init() {
	_ = kubevirtv1.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
}

// WebhookHandler handles admission webhook requests
type WebhookHandler struct {
	config  *config.Config
	mutator *mutation.VMFeatureMutator
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(cfg *config.Config, mutator *mutation.VMFeatureMutator) *WebhookHandler {
	return &WebhookHandler{
		config:  cfg,
		mutator: mutator,
	}
}

// Handle processes admission webhook requests
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decode the admission review request
	admissionReview := &admissionv1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode admission review: %v", err), http.StatusBadRequest)
		return
	}

	if admissionReview.Request == nil {
		http.Error(w, "admission review request is nil", http.StatusBadRequest)
		return
	}

	// Process the request
	response := h.mutate(admissionReview.Request)

	// Construct the response
	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

	// Encode and send the response
	responseBytes, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

// mutate processes the admission request and returns an admission response
func (h *WebhookHandler) mutate(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	response := &admissionv1.AdmissionResponse{
		Allowed: true,
	}

	// Parse the VirtualMachine object
	vm := &kubevirtv1.VirtualMachine{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(req.Object.Raw, nil, vm); err != nil {
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("failed to decode VirtualMachine: %v", err),
		}
		return response
	}

	// Check if the VM matches any rule
	if !h.config.Matches(req.Namespace, vm.Name) {
		// No match, allow without modification
		return response
	}

	// Create a copy of the VM for mutation
	vmCopy := vm.DeepCopy()

	// Mutate the VM
	if err := h.mutator.MutateVM(vmCopy); err != nil {
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("failed to mutate VirtualMachine: %v", err),
		}
		return response
	}

	// Create JSON patch
	originalBytes, err := json.Marshal(vm)
	if err != nil {
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("failed to marshal original VM: %v", err),
		}
		return response
	}

	mutatedBytes, err := json.Marshal(vmCopy)
	if err != nil {
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("failed to marshal mutated VM: %v", err),
		}
		return response
	}

	// Generate JSON patch
	patchBytes, err := createJSONPatch(originalBytes, mutatedBytes)
	if err != nil {
		response.Result = &metav1.Status{
			Message: fmt.Sprintf("failed to create JSON patch: %v", err),
		}
		return response
	}

	if len(patchBytes) > 0 {
		patchType := admissionv1.PatchTypeJSONPatch
		response.Patch = patchBytes
		response.PatchType = &patchType
	}

	return response
}

// createJSONPatch creates a JSON patch between two JSON documents
func createJSONPatch(original, mutated []byte) ([]byte, error) {
	// Simple implementation: unmarshal both, compare, and create patch
	var origMap, mutMap map[string]interface{}
	
	if err := json.Unmarshal(original, &origMap); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(mutated, &mutMap); err != nil {
		return nil, err
	}

	// Check if they're the same
	origJSON, _ := json.Marshal(origMap)
	mutJSON, _ := json.Marshal(mutMap)
	
	if string(origJSON) == string(mutJSON) {
		return nil, nil
	}

	// For this use case, we need to patch the spec.template.spec.domain.cpu.features field
	// Extract the CPU features from mutated
	spec, ok := mutMap["spec"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	
	template, ok := spec["template"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	
	templateSpec, ok := template["spec"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	
	domain, ok := templateSpec["domain"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	
	cpu, ok := domain["cpu"].(map[string]interface{})
	if !ok {
		return nil, nil
	}

	features, ok := cpu["features"]
	if !ok {
		return nil, nil
	}

	// Create patch operations
	patches := []map[string]interface{}{}
	
	// Check if the path exists in original
	origSpec, _ := origMap["spec"].(map[string]interface{})
	origTemplate, _ := origSpec["template"].(map[string]interface{})
	origTemplateSpec, _ := origTemplate["spec"].(map[string]interface{})
	origDomain, _ := origTemplateSpec["domain"].(map[string]interface{})
	origCPU, _ := origDomain["cpu"].(map[string]interface{})
	
	if origCPU == nil {
		// Need to add the entire CPU structure
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/template/spec/domain/cpu",
			"value": cpu,
		})
	} else if origCPU["features"] == nil {
		// Need to add just the features
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/template/spec/domain/cpu/features",
			"value": features,
		})
	} else {
		// Replace the features
		patches = append(patches, map[string]interface{}{
			"op":    "replace",
			"path":  "/spec/template/spec/domain/cpu/features",
			"value": features,
		})
	}

	return json.Marshal(patches)
}
