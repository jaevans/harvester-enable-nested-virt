# Harvester Nested Virtualization Webhook

A Kubernetes mutating webhook that automatically enables nested virtualization on KubeVirt VirtualMachines in Harvester based on namespace and VM name patterns.

## Overview

This webhook intercepts KubeVirt VirtualMachine create and update operations and automatically adds the appropriate CPU virtualization feature (`vmx` for Intel or `svm` for AMD) when the VM matches configured namespace and name patterns.

## Features

- ✅ Automatic CPU feature detection (Intel VT-x/VMX or AMD-V/SVM)
- ✅ ConfigMap-based configuration for namespace and VM name patterns
- ✅ Regex pattern matching for flexible VM selection
- ✅ Non-invasive: only mutates VMs that match configured patterns
- ✅ Comprehensive test coverage using Ginkgo/Gomega
- ✅ Production-ready with health checks and graceful shutdown

## How It Works

1. A ConfigMap defines which VMs should have nested virtualization enabled using namespace-based regex patterns
2. When a VirtualMachine is created or updated, the webhook checks if it matches any configured pattern
3. If matched, the webhook detects the host CPU type and adds the appropriate feature:
   - `vmx` for Intel processors (VT-x)
   - `svm` for AMD processors (AMD-V)
4. The modified VirtualMachine is then created with nested virtualization support

## Prerequisites

- Kubernetes cluster with KubeVirt installed
- Go 1.21 or later (for building)
- TLS certificates for webhook server

## Building

### Local Build

```bash
make build
```

### Docker Build

```bash
make docker-build
```

Or with a specific version:

```bash
make docker-build VERSION=v1.0.0
```

## Configuration

The webhook is configured via a ConfigMap with the following format:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nested-virt-config
  namespace: harvester-nested-virt
data:
  # Format: namespace: "regex1,regex2,regex3"
  default: "^vm-.*,^test-.*"
  production: "^prod-.*"
  staging: ".*-staging$"
```

Each key is a namespace name, and the value is a comma-separated list of regex patterns to match VM names.

### Example Patterns

- `^vm-.*` - Matches VMs starting with "vm-"
- `.*-prod$` - Matches VMs ending with "-prod"
- `^nested-.*` - Matches VMs starting with "nested-"
- `.*` - Matches all VMs in the namespace

## Deployment

### 1. Generate TLS Certificates

You need to generate TLS certificates for the webhook server. You can use cert-manager or create them manually:

```bash
# Using OpenSSL
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes \
  -subj "/CN=nested-virt-webhook.harvester-nested-virt.svc"

# Create Kubernetes secret
kubectl create secret tls nested-virt-webhook-certs \
  --cert=tls.crt \
  --key=tls.key \
  -n harvester-nested-virt
```

### 2. Deploy the Webhook

```bash
# Create namespace and RBAC
kubectl apply -f deploy/configmap.yaml
kubectl apply -f deploy/rbac.yaml

# Deploy the webhook
kubectl apply -f deploy/deployment.yaml

# Configure the MutatingWebhookConfiguration
# Note: Update the caBundle field in webhook.yaml with your CA certificate
kubectl apply -f deploy/webhook.yaml
```

### 3. Configure VM Matching Rules

Edit the ConfigMap to define which VMs should have nested virtualization enabled:

```bash
kubectl edit configmap nested-virt-config -n harvester-nested-virt
```

## Testing

### Run All Tests

```bash
make test
```

### Run Tests with Coverage

```bash
make test-coverage
```

This generates `coverage.html` with a visual coverage report.

### Run Tests for Specific Packages

```bash
# Config package
go test -v ./pkg/config/...

# Mutation package
go test -v ./pkg/mutation/...

# Webhook package
go test -v ./pkg/webhook/...
```

## Development

### Project Structure

```
.
├── cmd/
│   └── webhook/          # Main application entry point
├── pkg/
│   ├── config/          # ConfigMap parsing and rule matching
│   ├── mutation/        # CPU feature detection and VM mutation
│   └── webhook/         # Webhook server and handler
├── deploy/              # Kubernetes manifests
├── Dockerfile           # Container image definition
├── Makefile            # Build and test automation
└── README.md           # This file
```

### Code Organization

- **pkg/config**: Parses ConfigMap and matches VMs against regex patterns
- **pkg/mutation**: Detects CPU features and mutates VirtualMachine objects
- **pkg/webhook**: HTTP server and admission webhook handler
- **cmd/webhook**: Main application that ties everything together

### Running Locally

```bash
# Build the binary
make build

# Run with custom configuration (requires valid kubeconfig and certificates)
./bin/webhook \
  --port=8443 \
  --cert-file=/path/to/tls.crt \
  --key-file=/path/to/tls.key \
  --configmap-name=nested-virt-config \
  --configmap-namespace=default \
  --kubeconfig=$HOME/.kube/config
```

## Command-Line Flags

- `--port` - Webhook server port (default: 8443)
- `--cert-file` - Path to TLS certificate file (default: /etc/webhook/certs/tls.crt)
- `--key-file` - Path to TLS key file (default: /etc/webhook/certs/tls.key)
- `--configmap-name` - ConfigMap name (default: nested-virt-config)
- `--configmap-namespace` - ConfigMap namespace (default: default)
- `--kubeconfig` - Path to kubeconfig file (optional, uses in-cluster config if not provided)

## Troubleshooting

### Webhook Not Mutating VMs

1. Check webhook logs:
   ```bash
   kubectl logs -n harvester-nested-virt deployment/nested-virt-webhook
   ```

2. Verify ConfigMap is correctly formatted:
   ```bash
   kubectl get configmap nested-virt-config -n harvester-nested-virt -o yaml
   ```

3. Check if VM name matches any pattern:
   ```bash
   # Test regex patterns
   echo "vm-test-123" | grep -E "^vm-.*"
   ```

### Webhook Failing to Start

1. Check certificates are valid:
   ```bash
   kubectl get secret nested-virt-webhook-certs -n harvester-nested-virt
   ```

2. Verify RBAC permissions:
   ```bash
   kubectl get clusterrole nested-virt-webhook
   kubectl get clusterrolebinding nested-virt-webhook
   ```

### CPU Feature Detection Issues

The webhook detects CPU features by reading `/proc/cpuinfo`. On some systems, this may require additional permissions or may not expose the virtualization flags correctly.

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `make test`
2. Code is formatted: `make fmt`
3. Go modules are tidy: `make tidy`
4. Add tests for new functionality
