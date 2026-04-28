# AMD Kubernetes Network Node Labeller

## Introduction

The **AMD Kubernetes Network Node Labeller** automatically labels Kubernetes nodes with AINIC properties when one or more AMD AINICs are installed. This enables intelligent pod scheduling and resource management based on specific network hardware capabilities.

## Key Features

### Automatic Node Labeling

* Discovers AMD AI NICs on cluster nodes
* Automatically applies labels for hardware properties
* Updates labels when hardware configuration changes
* Supports both homogeneous and heterogeneous node configurations

### AINIC Properties Labeled

The labeller creates node labels for the following AMD AINIC properties:

* **Count** (`-count`) - Number of AINICs installed
* **Product Name** (`-product-name`) - NIC model identifier
* **Port Count** (`-port-count`) - Number of network ports
* **Port Speed** (`-port-speed`) - Network port speed
* **Firmware Version** (`-firmware-version`) - AINIC firmware version
* **Driver Version** (`-driver-version`) - Driver version in use
* **Driver Name** (`-driver-name`) - Driver name

### Label Format

For homogeneous nodes (all NICs of the same model):
```
amd.com/nic.count=2
amd.com/nic.product-name=POLLARA_1x400G_QSFP112
amd.com/nic.port-count=2
amd.com/nic.port-speed=100G
amd.com/nic.firmware-version=1.117.1-a-7
amd.com/driver-name=ionic
amd.com/driver-version=25.06.4.001
```

For heterogeneous nodes (different NIC models):
```
amd.com/nic.count=2
amd.com/nic.pollara-1q400p.count=1
amd.com/nic.pollara-1q400p.product-name=POLLARA_1x400G_QSFP112
```

## Compatibility

### Supported Hardware

| Hardware | Status |
|-----------|---------|
| AMD Pensando™ Pollara AI NIC | ✅ Supported |

### Version Compatibility Matrix

The following matrix summarizes supported NICs and the required AINIC firmware / tooling for each container image version.

| Image Version | AINIC Firmware Version           | Supported NICs |
| ------------- | -------------------------------- | -------------- |
| `v1.0.0`      | N/A (host `nicctl`)              | Pollara 400    |
| `v1.1.0`      | `1.117.5-a-56`                   | Pollara 400    |
| `v1.2.0`      | `1.117.5-a-56`<br>`1.117.5-a-77` | Pollara 400    |

**Note:** When running on VMs, the labeller has limited access to hardware information and will only publish Product Name, Driver Version, and Driver Name labels. Hardware-specific properties like port count, port speed, and firmware version may not be available in virtualized environments.

## Prerequisites

* Kubernetes v1.29.0+
* The Node Labeller must be run inside a Kubernetes Pod
* Node hostname available as environment variable `DS_NODE_NAME`
* Service account with appropriate RBAC permissions:
  * apiGroups: core ("")
  * resources: `nodes`
  * verbs: `watch`, `get`, `list`, `update`
* Privileged mode for NIC feature discovery

## Deployment

### Quick Start

The node labeller can be deployed using either DaemonSet or Helm:

#### DaemonSet Deployment

```bash
kubectl apply -f ./examples/k8s-network-node-labeller-ds.yaml
```

#### Helm Deployment

```bash
helm repo add rocm-network-nl https://rocm.github.io/k8s-network-node-labeller
helm repo update
helm install amd-network-node-labeller rocm-network-nl/network-node-labeller-charts \
  --namespace kube-amd-network \
  --create-namespace \
  --version v1.2.0
```

For detailed installation instructions, see the [Kubernetes (Helm) Installation Guide](installation/kubernetes-helm.md).

## Usage

### Node Selection with Label Selectors

Once deployed, you can use Kubernetes label selectors to target nodes with specific AINIC properties:

```bash
# Select nodes with 100G port speed
kubectl get nodes -l amd.com/nic.port-speed=100G

# Select nodes with Pollara NICs
kubectl get nodes -l amd.com/nic.product-name=POLLARA_1x400G_QSFP112
```

### Pod Scheduling

Use node selectors in pod specifications to schedule workloads on nodes with specific network capabilities:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: network-intensive-app
spec:
  nodeSelector:
    amd.com/nic.port-speed: "100G"
  containers:
  - name: app
    image: my-network-app:latest
```

## Documentation

* [Installation Guide](installation/kubernetes-helm.md)
* [Release Notes](releasenotes.md)

## Support

For bugs and feature requests, please file an issue on our [GitHub Issues](https://github.com/ROCm/k8s-network-node-labeller/issues) page.

## Summary

The AMD Kubernetes Network Node Labeller enables automatic discovery and labeling of AMD AI NIC properties on Kubernetes nodes. This facilitates intelligent workload placement based on network hardware capabilities, ensuring optimal performance for network-intensive AI and HPC applications.
