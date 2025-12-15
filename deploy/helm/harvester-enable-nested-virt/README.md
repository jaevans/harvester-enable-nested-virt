# Harvester Enable Nested Virt Helm Chart

This Helm chart deploys the Harvester Enable Nested Virt webhook, a mutating admission webhook that automatically enables nested virtualization on KubeVirt VirtualMachines (VMs) based on namespace and VM name patterns defined in a ConfigMap.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- Harvester 1.6+
- cert-manager (optional but recommended for automatic certificate management)

## Installation Methods

### From OCI Registry (Recommended)

```bash
helm install enable-nested-virt oci://ghcr.io/jaevans/harvester-enable-nested-virt/charts/enable-nested-virt \
  --namespace enable-nested-virt \
  --create-namespace
```

### From Git Repository

```bash
git clone https://github.com/jaevans/harvester-enable-nested-virt.git
cd harvester-enable-nested-virt

helm install enable-nested-virt ./deploy/helm/harvester-enable-nested-virt \
  --namespace enable-nested-virt \
  --create-namespace
```

## Installing the Chart

### With cert-manager (Recommended)

1. Install cert-manager if not already installed:
```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.0/cert-manager.yaml
```

2. Install the chart (it will automatically create a self-signed Issuer):
```bash
helm install enable-nested-virt ./deploy/helm/harvester-enable-nested-virt \
  --namespace enable-nested-virt \
  --create-namespace
```

The chart will automatically create a self-signed `Issuer` in the namespace to provision certificates. No additional setup required!

3. Configure which VMs should have nested virtualization enabled:
```bash
# Option 1: Via helm values file
cat > values.yaml <<EOF
config:
  rules:
    - namespace: default
      patterns:
        - "^vm-.*"
        - "^test-.*"
    - namespace: production
      patterns:
        - "^prod-.*"
EOF

helm upgrade enable-nested-virt ./deploy/helm/harvester-enable-nested-virt \
  --namespace enable-nested-virt \
  -f values.yaml

# Option 2: Via --set flags
helm upgrade enable-nested-virt ./deploy/helm/harvester-enable-nested-virt \
  --namespace enable-nested-virt \
  --set 'config.rules[0].namespace=default' \
  --set 'config.rules[0].patterns[0]=^vm-.*'

# Option 3: Edit the ConfigMap directly
kubectl edit configmap enable-nested-virt-config -n enable-nested-virt
```

### With an existing cert-manager Issuer

If you want to use an existing ClusterIssuer or Issuer:

```bash
helm install enable-nested-virt ./deploy/helm/enable-nested-virt \
  --namespace enable-nested-virt \
  --create-namespace \
  --set certificates.certManager.createIssuer=false \
  --set certificates.certManager.issuerKind=ClusterIssuer \
  --set certificates.certManager.issuerName=my-existing-issuer
```

### Without cert-manager (Manual Certificates)

1. Generate certificates:
```bash
# Generate CA
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -subj "/CN=enable-nested-virt-ca" -days 3650 -out ca.crt

# Generate server certificate
openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -subj "/CN=enable-nested-virt-webhook.enable-nested-virt.svc" -out tls.csr

# Sign the certificate
cat > csr.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = enable-nested-virt-webhook
DNS.2 = enable-nested-virt-webhook.enable-nested-virt
DNS.3 = enable-nested-virt-webhook.enable-nested-virt.svc
DNS.4 = enable-nested-virt-webhook.enable-nested-virt.svc.cluster.local
EOF

openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out tls.crt -days 365 -extensions v3_req -extfile csr.conf

# Base64 encode
CA_BUNDLE=$(cat ca.crt | base64 -w 0)
TLS_CRT=$(cat tls.crt | base64 -w 0)
TLS_KEY=$(cat tls.key | base64 -w 0)
```

2. Install with manual certificates:
```bash
helm install enable-nested-virt ./deploy/helm/enable-nested-virt \
  --namespace enable-nested-virt \
  --create-namespace \
  --set certificates.certManager.enabled=false \
  --set certificates.manual.caCert="$CA_BUNDLE" \
  --set certificates.manual.tlsCert="$TLS_CRT" \
  --set certificates.manual.tlsKey="$TLS_KEY"
```

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter                               | Description                         | Default                                         |
| --------------------------------------- | ----------------------------------- | ----------------------------------------------- |
| `replicaCount`                          | Number of webhook replicas          | `1`                                             |
| `image.repository`                      | Webhook image repository            | `ghcr.io/jaevans/harvester-enable-nested-virt`  |
| `image.tag`                             | Image tag                           | Chart appVersion                                |
| `image.pullPolicy`                      | Image pull policy                   | `IfNotPresent`                                  |
| `config.rules`                          | Namespace and VM name pattern rules | `[]` (empty, see values.yaml for examples)      |
| `config.debug`                          | Enable debug logging                | `false`                                         |
| `webhook.port`                          | Webhook server port                 | `8443`                                          |
| `webhook.certDir`                       | Certificate directory               | `/etc/webhook/certs`                            |
| `certificates.certManager.enabled`      | Use cert-manager                    | `true`                                          |
| `certificates.certManager.createIssuer` | Create a self-signed Issuer         | `true`                                          |
| `certificates.certManager.issuerKind`   | Issuer kind (if createIssuer=false) | `ClusterIssuer`                                 |
| `certificates.certManager.issuerName`   | Issuer name (if createIssuer=false) | `kubevirt-enable-nested-virt-selfsigned-issuer` |
| `resources.limits.cpu`                  | CPU limit                           | `200m`                                          |
| `resources.limits.memory`               | Memory limit                        | `128Mi`                                         |
| `resources.requests.cpu`                | CPU request                         | `100m`                                          |
| `resources.requests.memory`             | Memory request                      | `64Mi`                                          |

## Uninstalling the Chart

```bash
helm uninstall enable-nested-virt --namespace enable-nested-virt
```

## Troubleshooting

### Check webhook logs:
```bash
kubectl logs -n enable-nested-virt -l app.kubernetes.io/name=enable-nested-virt
```

### Check certificate status (with cert-manager):
```bash
kubectl get certificate -n enable-nested-virt
kubectl describe certificate -n enable-nested-virt enable-nested-virt-cert
```

### Test webhook manually:
```bash
kubectl run test-vm --image=nginx --dry-run=server -o yaml
```

### Verify webhook is receiving requests:
```bash
kubectl logs -n enable-nested-virt -l app.kubernetes.io/name=enable-nested-virt --tail=100 -f
```

## Development

To test changes locally:

```bash
# Lint the chart
helm lint ./deploy/helm/harvester-enable-nested-virt

# Template the chart
helm template enable-nested-virt ./deploy/helm/harvester-enable-nested-virt \
  --namespace enable-nested-virt

# Install in dry-run mode
helm install enable-nested-virt ./deploy/helm/harvester-enable-nested-virt \
  --namespace enable-nested-virt \
  --dry-run --debug
```

## Uninstalling the Chart

### Standard Uninstall

```bash
helm uninstall enable-nested-virt --namespace enable-nested-virt
```

**Note:** This may leave behind some cluster-scoped resources (MutatingWebhookConfiguration, ClusterRole, ClusterRoleBinding).

### Quick Manual Cleanup

If you need to manually remove the webhook configuration (e.g., it's preventing cluster operations):

```bash
kubectl delete mutatingwebhookconfiguration enable-nested-virt
kubectl delete clusterrolebinding enable-nested-virt
kubectl delete clusterrole enable-nested-virt
kubectl delete namespace enable-nested-virt
```

## License

See the [LICENSE](../../../LICENSE) file for details.
