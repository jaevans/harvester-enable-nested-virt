# Quick Start Guide

This guide will help you get the nested virtualization webhook up and running quickly.

## Prerequisites

- Kubernetes cluster with KubeVirt installed
- kubectl configured to access your cluster
- openssl (for certificate generation)

## Step 1: Clone the Repository

```bash
git clone https://github.com/jaevans/harvester-enable-nested-virt.git
cd harvester-enable-nested-virt
```

## Step 2: Generate TLS Certificates

```bash
./scripts/generate-certs.sh
```

This will:
- Generate TLS certificates for the webhook
- Create the `harvester-nested-virt` namespace
- Create a Kubernetes secret with the certificates
- Output the CA bundle for the webhook configuration

## Step 3: Update Webhook Configuration

Edit `deploy/webhook.yaml` and replace the empty `caBundle` field with the base64-encoded CA certificate from Step 2:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: nested-virt-webhook
webhooks:
- name: nested-virt.kubevirt.io
  admissionReviewVersions: ["v1"]
  clientConfig:
    service:
      name: nested-virt-webhook
      namespace: harvester-nested-virt
      path: "/mutate"
    caBundle: "<PASTE_CA_BUNDLE_HERE>"  # Replace with output from Step 2
  # ... rest of configuration
```

## Step 4: Configure VM Matching Rules

Edit `deploy/configmap.yaml` to define which VMs should have nested virtualization enabled:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nested-virt-config
  namespace: harvester-nested-virt
data:
  # Example: enable for all VMs starting with 'vm-' in the default namespace
  default: "^vm-.*"
  # Add more namespace patterns as needed
```

See `examples/configmap-examples.yaml` for more configuration examples.

## Step 5: Deploy the Webhook

```bash
# Deploy ConfigMap and RBAC
kubectl apply -f deploy/configmap.yaml
kubectl apply -f deploy/rbac.yaml

# Deploy the webhook service and deployment
kubectl apply -f deploy/deployment.yaml

# Configure the mutating webhook
kubectl apply -f deploy/webhook.yaml
```

## Step 6: Verify Installation

Check that the webhook is running:

```bash
kubectl get pods -n harvester-nested-virt
kubectl logs -n harvester-nested-virt deployment/nested-virt-webhook
```

You should see output like:
```
Loaded configuration with X rules
Starting webhook server on port 8443
```

## Step 7: Test the Webhook

Create a test VirtualMachine that matches your configured patterns:

```bash
kubectl apply -f examples/virtualmachine-example.yaml
```

Verify the VM has the CPU feature added:

```bash
kubectl get vm vm-nested-test -o yaml | grep -A 5 "cpu:"
```

You should see the `vmx` (Intel) or `svm` (AMD) feature:

```yaml
cpu:
  features:
  - name: vmx
    policy: require
```

## Updating Configuration

To update the matching rules without restarting the webhook:

1. Edit the ConfigMap:
   ```bash
   kubectl edit configmap nested-virt-config -n harvester-nested-virt
   ```

2. Restart the webhook to reload the configuration:
   ```bash
   kubectl rollout restart deployment nested-virt-webhook -n harvester-nested-virt
   ```

## Troubleshooting

### Webhook Not Mutating VMs

1. Check webhook logs:
   ```bash
   kubectl logs -n harvester-nested-virt deployment/nested-virt-webhook
   ```

2. Verify the VM name matches a pattern in the ConfigMap:
   ```bash
   kubectl get configmap nested-virt-config -n harvester-nested-virt -o yaml
   ```

3. Check webhook endpoint health:
   ```bash
   kubectl get endpoints -n harvester-nested-virt
   ```

### Certificate Issues

If you see TLS errors, regenerate the certificates:

```bash
./scripts/generate-certs.sh
kubectl rollout restart deployment nested-virt-webhook -n harvester-nested-virt
```

### Permission Issues

Verify RBAC is correctly configured:

```bash
kubectl get clusterrole nested-virt-webhook
kubectl get clusterrolebinding nested-virt-webhook
```

## Next Steps

- Review the [README](../README.md) for detailed documentation
- Check [examples](../examples/) for more configuration patterns
- Customize the deployment for your environment

## Uninstalling

To remove the webhook:

```bash
kubectl delete -f deploy/webhook.yaml
kubectl delete -f deploy/deployment.yaml
kubectl delete -f deploy/rbac.yaml
kubectl delete -f deploy/configmap.yaml
kubectl delete namespace harvester-nested-virt
```
