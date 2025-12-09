# Harvester Nested Virtualization Webhook - Copilot Instructions

## Project Overview

This is a Kubernetes mutating webhook that automatically enables nested virtualization on KubeVirt VirtualMachines in Harvester. The webhook intercepts VirtualMachine create/update operations and adds the appropriate CPU virtualization features (VMX for Intel or SVM for AMD) based on configurable namespace and VM name patterns.

## Technology Stack

- **Language**: Go 1.24
- **Framework**: Kubernetes admission webhooks
- **Key Dependencies**:
  - KubeVirt API for VirtualMachine resources
  - Kubernetes client-go for API interactions
  - Ginkgo/Gomega for testing
- **Build System**: Make
- **CI/CD**: GitHub Actions

## Project Structure

```
.
├── cmd/webhook/          # Main application entry point
├── pkg/
│   ├── config/          # ConfigMap parsing and VM pattern matching
│   ├── mutation/        # CPU feature detection and VM mutation logic
│   └── webhook/         # HTTP server and admission webhook handler
├── deploy/              # Kubernetes manifests
├── scripts/             # Helper scripts (e.g., certificate generation)
├── Makefile            # Build and test automation
└── .github/workflows/   # CI/CD workflows
```

## Build, Test, and Lint Commands

### Essential Commands

- **Build**: `make build` - Compiles the webhook binary to `bin/webhook`
- **Test**: `make test` - Runs all tests with race detection and coverage
- **Lint**: `make lint` - Runs golangci-lint (requires golangci-lint installed)
- **Format**: `make fmt` - Formats Go code
- **Tidy**: `make tidy` - Tidies Go module dependencies
- **Coverage**: `make test-coverage` - Generates HTML coverage report

### Running Tests

- Run all tests: `make test`
- Run specific package tests: `go test -v ./pkg/config/...`
- View coverage: `make test-coverage` (opens coverage.html)

## Code Style and Requirements

### General Guidelines

1. **Minimal Changes**: Make the smallest possible changes to achieve the goal
2. **Test Coverage**: All new functionality must have test coverage using Ginkgo/Gomega
3. **Error Handling**: Always handle errors appropriately
4. **Documentation**: Update documentation for significant changes
5. **Code Formatting**: Run `make fmt` before committing

### Testing Standards

- Use **Ginkgo** BDD-style test framework with **Gomega** matchers
- Test files use `package_test` naming convention (e.g., `mutation_test`)
- Use descriptive `Describe`, `Context`, and `It` blocks
- Mock external dependencies (see `MockCPUFeatureDetector` pattern)
- Test both success and error cases

### Example Test Structure

```go
var _ = Describe("ComponentName", func() {
    Context("when condition X", func() {
        It("should do Y", func() {
            // Arrange
            // Act
            // Assert using Gomega matchers
            Expect(result).To(Equal(expected))
        })
    })
})
```

## Making Changes

### Before Making Changes

1. Run existing tests to understand baseline: `make test`
2. Run linter to see any existing issues: `make lint`
3. Review relevant code in `pkg/` directory

### Development Workflow

1. Make focused, minimal changes
2. Write or update tests immediately
3. Run `make test` to verify tests pass
4. Run `make fmt` and `make tidy`
5. Run `make lint` to check code quality
6. Build to ensure compilation: `make build`

### Areas to Modify

- **Config logic** (`pkg/config/`): ConfigMap parsing and VM name pattern matching
- **Mutation logic** (`pkg/mutation/`): CPU feature detection and VM modification
- **Webhook handler** (`pkg/webhook/`): HTTP server and admission controller logic
- **Main application** (`cmd/webhook/`): Application initialization and flag parsing

## Key Concepts

### CPU Feature Detection

The webhook detects CPU features by reading `/proc/cpuinfo`:
- Intel processors: Looks for `vmx` flag (VT-x)
- AMD processors: Looks for `svm` flag (AMD-V)

### VM Mutation

When a VM matches configured patterns, the webhook adds:
```go
CPU.Features = append(CPU.Features, kubevirtv1.CPUFeature{
    Name:   "vmx" or "svm",
    Policy: "require",
})
```

### Configuration

VMs are matched using regex patterns defined in a ConfigMap:
- Key: namespace name
- Value: comma-separated regex patterns for VM names

## CI/CD

GitHub Actions runs automatically on pull requests:
- **Test Job**: Runs full test suite with coverage
- **Build Job**: Verifies compilation
- **Lint Job**: Checks code quality and dependency tidiness

All jobs must pass before merging.

## Common Pitfalls to Avoid

1. **Don't** remove or modify existing tests unless absolutely necessary
2. **Don't** add new dependencies without careful consideration
3. **Don't** forget to handle errors returned by Kubernetes API calls
4. **Don't** modify deployment manifests unless the change requires it
5. **Don't** commit build artifacts (bin/, coverage files) - they're in .gitignore

## Deployment Context

This webhook runs in a Kubernetes cluster as a mutating admission webhook:
- Requires TLS certificates (via cert-manager or manual generation)
- Needs RBAC permissions to read ConfigMaps
- Listens on port 8443 for admission requests
- Targets `VirtualMachine` resources in the `kubevirt.io/v1` API group

## When Modifying Webhook Behavior

If changing how VMs are mutated:
1. Update `pkg/mutation/` logic
2. Add comprehensive tests covering new behavior
3. Verify backward compatibility with existing VMs
4. Consider impact on deployed ConfigMap format
5. Update README.md with new configuration examples if needed

## Security Considerations

- Validate all input from ConfigMap patterns
- Ensure regex patterns can't cause ReDoS (Regular Expression Denial of Service)
- Don't log sensitive VM configuration data
- Follow principle of least privilege for RBAC

## Getting Help

- **Project README**: Comprehensive usage and deployment guide
- **Test files**: Examples of how components work
- **GitHub Workflows**: CI/CD configuration in `.github/workflows/`
