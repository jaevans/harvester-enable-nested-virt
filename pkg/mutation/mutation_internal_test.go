package mutation

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mutation Internal tests", func() {
	Describe("DefaultCPUFeatureDetector", func() {
		Context("DetectFeature", func() {
			var (
				detector CPUFeatureDetector
			)

			BeforeEach(func() {
				detector = NewDefaultCPUFeatureDetector()
			})

			It("should be created with default constructor", func() {
				Expect(detector).NotTo(BeNil())
			})

			It("should return an error if /proc/cpuinfo cannot be read", func() {
				invalidDetector := &DefaultCPUFeatureDetector{cpuInfoPath: "/nonexistent/path"}
				feature, err := invalidDetector.DetectFeature()
				Expect(err).To(HaveOccurred())
				Expect(feature).To(BeEquivalentTo(CPUFeatureNil))
			})

			It("should detect VMX on Intel feature", func() {
				fakecpuInfoFile := GinkgoT().TempDir() + "/cpuinfo"
				cpuInfoContent := `
processor   : 0
vendor_id   : GenuineIntel
cpu family  : 6
model       : 79
model name  : Intel(R) Xeon(R) CPU E5-2609 v4 @ 1.70GHz
stepping    : 1
microcode   : 0xb000038
cpu MHz     : 1700.111
cache size  : 20480 KB
physical id : 0
siblings    : 8
core id     : 0
cpu cores   : 8
apicid      : 0
initial apicid  : 0
fpu     : yes
fpu_exception   : yes
cpuid level : 20
wp      : yes
flags       : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm constant_tsc arch_perfmon pebs bts rep_good nopl xtopology nonstop_tsc cpuid aperfmperf pni pclmulqdq dtes64 monitor ds_cpl vmx smx est tm2 ssse3 sdbg fma cx16 xtpr pdcm pcid dca sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch cpuid_fault epb cat_l3 cdp_l3 pti intel_ppin ssbd ibrs ibpb stibp tpr_shadow flexpriority ept vpid ept_ad fsgsbase tsc_adjust bmi1 hle avx2 smep bmi2 erms invpcid rtm cqm rdt_a rdseed adx smap intel_pt xsaveopt cqm_llc cqm_occup_llc cqm_mbm_total cqm_mbm_local dtherm arat pln pts vnmi md_clear flush_l1d
vmx flags   : vnmi preemption_timer posted_intr invvpid ept_x_only ept_ad ept_1gb flexpriority apicv tsc_offset vtpr mtf vapic ept vpid unrestricted_guest vapic_reg vid ple shadow_vmcs pml ept_violation_ve
bugs        : cpu_meltdown spectre_v1 spectre_v2 spec_store_bypass l1tf mds swapgs taa itlb_multihit mmio_stale_data
bogomips    : 3400.21
clflush size    : 64
cache_alignment : 64
address sizes   : 46 bits physical, 48 bits virtual
power management:`

				err := os.WriteFile(fakecpuInfoFile, []byte(cpuInfoContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				detector = &DefaultCPUFeatureDetector{cpuInfoPath: fakecpuInfoFile}

				feature, err := detector.DetectFeature()
				Expect(err).NotTo(HaveOccurred())
				Expect(feature).To(BeEquivalentTo(CPUFeatureVMX))
			})

			It("should detect SVM on AMD feature", func() {
				fakecpuInfoFile := GinkgoT().TempDir() + "/cpuinfo"
				cpuInfoContent := `
processor	: 0
vendor_id	: AuthenticAMD
cpu family	: 26
model		: 68
model name	: AMD EPYC 4545P 16-Core Processor
stepping	: 0
microcode	: 0xb404023
cpu MHz		: 5131.972
cache size	: 1024 KB
physical id	: 0
siblings	: 32
core id		: 14
cpu cores	: 16
apicid		: 29
initial apicid	: 29
fpu		: yes
fpu_exception	: yes
cpuid level	: 16
wp		: yes
flags		: fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx mmxext fxsr_opt pdpe1gb rdtscp lm constant_tsc rep_good amd_lbr_v2 nopl nonstop_tsc cpuid extd_apicid aperfmperf rapl pni pclmulqdq monitor ssse3 fma cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx f16c rdrand lahf_lm cmp_legacy svm extapic cr8_legacy abm sse4a misalignsse 3dnowprefetch osvw ibs skinit wdt tce topoext perfctr_core perfctr_nb bpext perfctr_llc mwaitx cpb cat_l3 cdp_l3 hw_pstate ssbd mba perfmon_v2 ibrs ibpb stibp ibrs_enhanced vmmcall fsgsbase tsc_adjust bmi1 avx2 smep bmi2 erms invpcid cqm rdt_a avx512f avx512dq rdseed adx smap avx512ifma clflushopt clwb avx512cd sha_ni avx512bw avx512vl xsaveopt xsavec xgetbv1 xsaves cqm_llc cqm_occup_llc cqm_mbm_total cqm_mbm_local avx_vnni avx512_bf16 clzero irperf xsaveerptr rdpru wbnoinvd cppc amd_ibpb_ret arat npt lbrv svm_lock nrip_save tsc_scale vmcb_clean flushbyasid decodeassists pausefilter pfthreshold avic v_vmsave_vmload vgif x2avic v_spec_ctrl avx512vbmi umip pku ospke avx512_vbmi2 gfni vaes vpclmulqdq avx512_vnni avx512_bitalg avx512_vpopcntdq rdpid bus_lock_detect movdiri movdir64b overflow_recov succor smca fsrm avx512_vp2intersect flush_l1d
bugs		: sysret_ss_attrs spectre_v1 spectre_v2 spec_store_bypass
bogomips	: 5988.35
TLB size	: 192 4K pages
clflush size	: 64
cache_alignment	: 64
address sizes	: 48 bits physical, 48 bits virtual
power management: ts ttp tm hwpstate cpb eff_freq_ro [13] [14]
`

				err := os.WriteFile(fakecpuInfoFile, []byte(cpuInfoContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				detector = &DefaultCPUFeatureDetector{cpuInfoPath: fakecpuInfoFile}

				feature, err := detector.DetectFeature()
				Expect(err).NotTo(HaveOccurred())
				Expect(feature).To(BeEquivalentTo(CPUFeatureSVM))
			})

			It("should return an error on a non X86 processor", func() {
				fakecpuInfoFile := GinkgoT().TempDir() + "/cpuinfo"
				cpuInfoContent := `
processor	: 3
BogoMIPS	: 108.00
Features	: fp asimd evtstrm crc32 cpuid
CPU implementer	: 0x41
CPU architecture: 8
CPU variant	: 0x0
CPU part	: 0xd08
CPU revision	: 3
`

				err := os.WriteFile(fakecpuInfoFile, []byte(cpuInfoContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				detector = &DefaultCPUFeatureDetector{cpuInfoPath: fakecpuInfoFile}

				feature, err := detector.DetectFeature()
				Expect(err).To(HaveOccurred())
				Expect(feature).To(BeEquivalentTo(CPUFeatureNil))
			})
			// Note: We can't easily test the actual CPU feature detection
			// in a unit test as it depends on the host CPU.
			// Integration tests would be needed to verify this functionality.
		})
	})

	Describe("NewVMFeatureMutator", func() {
		It("should create a VMFeatureMutator with default detector when nil is provided", func() {
			mutator := NewVMFeatureMutator(nil)
			Expect(mutator).NotTo(BeNil())
			Expect(mutator.detector).NotTo(BeNil())
		})

		It("should create a VMFeatureMutator with the provided detector", func() {
			mockDetector := &DefaultCPUFeatureDetector{}
			mutator := NewVMFeatureMutator(mockDetector)
			Expect(mutator).NotTo(BeNil())
			Expect(mutator.detector).To(Equal(mockDetector))
		})
	})
})
