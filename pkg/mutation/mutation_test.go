package mutation_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kubevirtv1 "kubevirt.io/api/core/v1"

	"github.com/jaevans/harvester-enable-nested-virt/pkg/mutation"
)

// MockCPUFeatureDetector is a mock implementation of CPUFeatureDetector for testing
type MockCPUFeatureDetector struct {
	feature mutation.CPUFeature
	err     error
}

func (m *MockCPUFeatureDetector) DetectFeature() (mutation.CPUFeature, error) {
	return m.feature, m.err
}

var _ = Describe("Mutation", func() {
	Describe("VMFeatureMutator", func() {
		Context("when CPU feature is VMX", func() {
			It("should add VMX feature to VM", func() {
				detector := &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
				mutator := mutation.NewVMFeatureMutator(detector)

				vm := &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{},
				}

				err := mutator.MutateVM(vm)
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Spec.Template).NotTo(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.CPU).NotTo(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features[0].Name).To(BeEquivalentTo(mutation.CPUFeatureVMX))
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features[0].Policy).To(Equal("require"))
			})

			It("should not add VMX feature if already present", func() {
				detector := &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
				mutator := mutation.NewVMFeatureMutator(detector)

				vm := &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									CPU: &kubevirtv1.CPU{
										Features: []kubevirtv1.CPUFeature{
											{Name: string(mutation.CPUFeatureVMX), Policy: "require"},
										},
									},
								},
							},
						},
					},
				}

				err := mutator.MutateVM(vm)
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features).To(HaveLen(1))
			})
		})

		Context("when CPU feature is SVM", func() {
			It("should add SVM feature to VM", func() {
				detector := &MockCPUFeatureDetector{feature: mutation.CPUFeatureSVM}
				mutator := mutation.NewVMFeatureMutator(detector)

				vm := &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{},
				}

				err := mutator.MutateVM(vm)
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Spec.Template).NotTo(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.CPU).NotTo(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features[0].Name).To(BeEquivalentTo(mutation.CPUFeatureSVM))
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features[0].Policy).To(Equal("require"))
			})
		})

		Context("when VM already has CPU features", func() {
			It("should append the virtualization feature to existing features", func() {
				detector := &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
				mutator := mutation.NewVMFeatureMutator(detector)

				vm := &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{
						Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
							Spec: kubevirtv1.VirtualMachineInstanceSpec{
								Domain: kubevirtv1.DomainSpec{
									CPU: &kubevirtv1.CPU{
										Features: []kubevirtv1.CPUFeature{
											{Name: "sse4.2", Policy: "require"},
										},
									},
								},
							},
						},
					},
				}

				err := mutator.MutateVM(vm)
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features).To(HaveLen(2))
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features[0].Name).To(Equal("sse4.2"))
				Expect(vm.Spec.Template.Spec.Domain.CPU.Features[1].Name).To(BeEquivalentTo(mutation.CPUFeatureVMX))
			})
		})

		Context("when VM is nil", func() {
			It("should return error", func() {
				detector := &MockCPUFeatureDetector{feature: mutation.CPUFeatureVMX}
				mutator := mutation.NewVMFeatureMutator(detector)

				err := mutator.MutateVM(nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when CPU feature detection fails", func() {
			It("should return error", func() {
				detector := &MockCPUFeatureDetector{err: fmt.Errorf("detection failed")}
				mutator := mutation.NewVMFeatureMutator(detector)

				vm := &kubevirtv1.VirtualMachine{
					Spec: kubevirtv1.VirtualMachineSpec{},
				}

				err := mutator.MutateVM(vm)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("detection failed"))
			})
		})
	})
})
