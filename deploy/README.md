# cert-manager Deployment

This directory contains deployment manifests for the nested virtualization webhook.

## Files

- `cert-manager-deployment.yaml` - All-in-one deployment using cert-manager for automatic certificate management
- `configmap.yaml` - ConfigMap for VM matching rules (manual deployment)
- `rbac.yaml` - RBAC resources (manual deployment)
- `deployment.yaml` - Webhook deployment (manual deployment)
- `webhook.yaml` - MutatingWebhookConfiguration (manual deployment)

## Deployment with cert-manager

The `cert-manager-deployment.yaml` file includes everything needed for deployment when cert-manager is installed:

1. **Namespace** - Creates `harvester-nested-virt` namespace
2. **Issuer** - Self-signed certificate issuer for cert-manager
3. **Certificate** - Defines the certificate to be generated with proper DNS names
4. **ConfigMap** - VM matching rules configuration
5. **RBAC** - ServiceAccount, ClusterRole, and ClusterRoleBinding
6. **Service** - Exposes the webhook on port 443
7. **Deployment** - Runs the webhook container
8. **MutatingWebhookConfiguration** - Configures Kubernetes to call the webhook with automatic CA injection

### Key Features

- **Automatic certificate generation**: cert-manager creates and manages TLS certificates
- **Automatic CA injection**: The `cert-manager.io/inject-ca-from` annotation automatically populates the `caBundle` field
- **Certificate renewal**: Certificates are automatically renewed before expiration (15 days before 90-day expiry)
- **Proper DNS names**: Certificate includes all necessary DNS names for the service

### Usage

```bash
# Prerequisites: cert-manager must be installed
# Install cert-manager if needed: https://cert-manager.io/docs/installation/

# Deploy everything
kubectl apply -f cert-manager-deployment.yaml

# Verify certificate was created
kubectl get certificate -n harvester-nested-virt
kubectl get secret nested-virt-webhook-certs -n harvester-nested-virt

# Configure VM matching rules
kubectl edit configmap nested-virt-config -n harvester-nested-virt
```

## Manual Deployment

For clusters without cert-manager, use the individual manifest files:

```bash
# Generate certificates
../scripts/generate-certs.sh

# Apply manifests
kubectl apply -f configmap.yaml
kubectl apply -f rbac.yaml
kubectl apply -f deployment.yaml
kubectl apply -f webhook.yaml
```

See the main [README](../README.md) for detailed instructions.
