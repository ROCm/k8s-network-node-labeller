# Kubernetes (Helm) installation

This page explains how to install AMD Kubernetes Network Node Labeller using Helm.

## System requirements

- Kubernetes cluster v1.29.0 or later
- Helm v3.2.0 or later
- `kubectl` command-line tool configured with access to the cluster
- Nodes equipped with AMD AINICs
- AMD AINIC drivers installed on the nodes

### Multi-Version Image Support

Starting with v1.3.0, container images support multiple nicctl versions bundled together. This allows a single image to work across clusters with different NIC firmware versions through automatic firmware detection at startup. See the [README Multi-Version nicctl Support](../../README.md#multi-version-nicctl-support) section for build details.

## Installation

### Add the AMD Helm Repository

```bash
helm repo add rocm-network-nl https://rocm.github.io/k8s-network-node-labeller
helm repo update
```

### Install the Helm chart

```bash
helm install amd-network-node-labeller rocm-network-nl/network-node-labeller-charts \
  --namespace kube-amd-network \
  --create-namespace \
  --version v1.2.0
```

### Verify the installation

Check that the DaemonSet pods are running on the expected nodes:

```bash
kubectl get pods -l app=amd-network-node-labeller -n kube-amd-network
```

Once the pods are running, verify that node labels have been applied:

```bash
kubectl get nodes --show-labels | grep amd.com/nic
```

Or inspect a specific node:

```bash
kubectl describe node <node-name>
```

## Configuration

To override default values, pass `--set key=value` on the command line or supply a custom values file via `-f my-values.yaml`. Refer to [`helm-charts-k8s/README.md`](../../helm-charts-k8s/README.md) for the full list of configurable values.

> **Note:** Ensure the Node Labeller image (`image.tag`) matches the AINIC firmware version installed on your nodes. Refer to the [Compatibility Matrix](../../README.md#compatibility-matrix) in the main README for the correct image version to use and configure the correct `image.tag` value in the values file.

### Example: custom tolerations

To schedule the labeller DaemonSet pods on nodes with a custom taint, add the required toleration:

```bash
helm install amd-network-node-labeller rocm-network-nl/network-node-labeller-charts \
  --namespace kube-amd-network \
  --create-namespace \
  --version v1.2.0 \
  --set tolerations[0].key=example.com/foo \
  --set tolerations[0].operator=Exists \
  --set tolerations[0].effect=NoSchedule
```

Or via a values file:

```yaml
tolerations:
  - key: example.com/foo
    operator: Exists
    effect: NoSchedule
```

```bash
helm install amd-network-node-labeller rocm-network-nl/network-node-labeller-charts \
  --namespace kube-amd-network \
  --create-namespace \
  --version v1.2.0 \
  -f my-values.yaml
```

### Debugging

Use the `--debug` flag with `helm install` or `helm template` to see the fully rendered Kubernetes manifests with all values applied. This is useful for verifying that overrides are taking effect before deploying:

```bash
helm template amd-network-node-labeller rocm-network-nl/network-node-labeller-charts \
  --version v1.2.0 \
  --debug \
  -f my-values.yaml
```

Use `--dry-run` with `helm install` to simulate the installation without applying any resources to the cluster. Unlike `helm template`, a dry run communicates with the Kubernetes API server to validate the manifests against the cluster:

```bash
helm install amd-network-node-labeller rocm-network-nl/network-node-labeller-charts \
  --namespace kube-amd-network \
  --create-namespace \
  --version v1.2.0 \
  --dry-run \
  -f my-values.yaml
```

## Uninstallation

```bash
helm uninstall amd-network-node-labeller -n kube-amd-network
```

This removes all Kubernetes resources created by the chart, including the DaemonSet, RBAC rules, and ServiceAccount. Node labels applied by the labeller are cleaned up automatically by the labeller before the pods terminate.
