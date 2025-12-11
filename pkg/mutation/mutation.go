package mutation

import (
	"fmt"
	"os"
	"strings"

	kubevirtv1 "kubevirt.io/api/core/v1"
)

type CPUFeature string

const (
	// CPU feature flags
	CPUFeatureNil = CPUFeature("")
	CPUFeatureVMX = CPUFeature("vmx") // Intel VT-x
	CPUFeatureSVM = CPUFeature("svm") // AMD-V
)

// CPUFeatureDetector defines the interface for detecting CPU features
type CPUFeatureDetector interface {
	DetectFeature() (CPUFeature, error)
}

// DefaultCPUFeatureDetector implements CPUFeatureDetector using /proc/cpuinfo
type DefaultCPUFeatureDetector struct {
	cpuInfoPath string
}

// NewDefaultCPUFeatureDetector creates a new DefaultCPUFeatureDetector
func NewDefaultCPUFeatureDetector() *DefaultCPUFeatureDetector {
	return &DefaultCPUFeatureDetector{
		cpuInfoPath: "/proc/cpuinfo",
	}
}

// DetectFeature reads /proc/cpuinfo and returns the appropriate CPU feature
func (d *DefaultCPUFeatureDetector) DetectFeature() (CPUFeature, error) {
	data, err := os.ReadFile(d.cpuInfoPath)
	if err != nil {
		return CPUFeatureNil, fmt.Errorf("failed to read %s: %w", d.cpuInfoPath, err)
	}

	cpuInfo := string(data)

	// Look for the flags line in cpuinfo
	lines := strings.Split(cpuInfo, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "flags") || strings.HasPrefix(line, "Features") {
			// Check for virtualization features
			if strings.Contains(line, " vmx ") || strings.Contains(line, " vmx\t") || strings.HasSuffix(line, " vmx") {
				return CPUFeatureVMX, nil
			}
			if strings.Contains(line, " svm ") || strings.Contains(line, " svm\t") || strings.HasSuffix(line, " svm") {
				return CPUFeatureSVM, nil
			}
		}
	}

	return CPUFeatureNil, fmt.Errorf("no virtualization feature (vmx or svm) found in CPU info")
}

// VMFeatureMutator handles mutation of VirtualMachine objects
type VMFeatureMutator struct {
	detector CPUFeatureDetector
}

// NewVMFeatureMutator creates a new VMFeatureMutator
func NewVMFeatureMutator(detector CPUFeatureDetector) *VMFeatureMutator {
	if detector == nil {
		detector = NewDefaultCPUFeatureDetector()
	}
	return &VMFeatureMutator{
		detector: detector,
	}
}

// MutateVM adds the appropriate CPU feature to a VirtualMachine
func (m *VMFeatureMutator) MutateVM(vm *kubevirtv1.VirtualMachine) error {
	if vm == nil {
		return fmt.Errorf("vm is nil")
	}

	feature, err := m.detector.DetectFeature()
	if err != nil {
		return fmt.Errorf("failed to detect CPU feature: %w", err)
	}

	// Ensure the CPU features structure exists
	if vm.Spec.Template == nil {
		vm.Spec.Template = &kubevirtv1.VirtualMachineInstanceTemplateSpec{}
	}
	if vm.Spec.Template.Spec.Domain.CPU == nil {
		vm.Spec.Template.Spec.Domain.CPU = &kubevirtv1.CPU{}
	}
	if vm.Spec.Template.Spec.Domain.CPU.Features == nil {
		vm.Spec.Template.Spec.Domain.CPU.Features = make([]kubevirtv1.CPUFeature, 0)
	}

	// Check if the feature already exists
	featureExists := false
	for _, f := range vm.Spec.Template.Spec.Domain.CPU.Features {
		if f.Name == string(feature) {
			featureExists = true
			break
		}
	}

	// Add the feature if it doesn't exist
	if !featureExists {
		vm.Spec.Template.Spec.Domain.CPU.Features = append(
			vm.Spec.Template.Spec.Domain.CPU.Features,
			kubevirtv1.CPUFeature{
				Name:   string(feature),
				Policy: "require",
			},
		)
	}

	return nil
}
