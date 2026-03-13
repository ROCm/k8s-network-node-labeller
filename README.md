# AMD Kubernetes Network Node Labeller

## Introduction

This tool automatically labels nodes with AINIC properties if a node has one or more AMD AINICs installed.
## Prerequisites

* The Node Labeller must be run inside a Kubernetes Pod.
* The node's hostname should be made available inside the container as an environment variable named `DS_NODE_NAME`. This can be set in the Pod spec.
* The Pod containing the Labeller must be deployed by a service account with sufficient API access. This can be achieved through the use of `ClusterRole` and `ClusterRoleBinding`.
  * apiGroups: core ("")
  * resources: `nodes`
  * verbs: `watch`, `get`, `list`, `update`

Please refer to this [example manifest file](./examples/k8s-network-node-labeller-ds.yaml) to know more about setting the environment variable and configuring the API access for the Node Labeller pod.

## Deployment

The Labeller must be run on all nodes equipped with AMD AINICs. The simplest way to do this is to create a Kubernetes DaemonSet, which runs a copy of the pod on all (or some) nodes in the cluster. An example configuration is available [here](./examples/k8s-network-node-labeller-ds.yaml). This labeller requires privileged mode for NIC feature discovery. It is recommended to consult with your cluster administrator or security expert to ensure appropriate security measures are in place.

## Compatibility Matrix

The following matrix summarizes supported NICs and the required AINIC firmware / tooling for each container image version.

| AINIC Firmware Version | Image Version | Supported NICs |
| --- | --- | --- |
| N/A (host `nicctl`) | `v1.0.0` | Pollara 400 |
| `1.117.5-a-56` | `v1.1.0` | Pollara 400 |

## Labels
The Labeller currently creates node labels for the following AMD AINIC properties:

* Count (-count)
* Product Name (-product-name)
* Port Count (-port-count)
* Port Speed (-port-speed)
* Firmware Version (-firmware-version)
* Driver Version (-driver-version)
* Driver Name (-driver-name)

Example result:

    $ kubectl describe node cluster-node-23
    Name:               cluster-node-23
    Roles:              <none>
    Labels:             amd.com/nic.count=2
                        amd.com/nic.product-name=POLLARA_1x400G_QSFP112
                        amd.com/nic.port-count=2
                        amd.com/nic.port-speed=100G
                        amd.com/nic.firmware-version=1.117.1-a-7
                        amd.com/driver-name=ionic
                        amd.com/driver-version=25.06.4.001
                        beta.kubernetes.io/arch=amd64
                        beta.kubernetes.io/os=linux
                        kubernetes.io/hostname=cluster-node-23
    Annotations:        kubeadm.alpha.kubernetes.io/cri-socket: /var/run/dockershim.sock
                        node.alpha.kubernetes.io/ttl: 0
    ......

**Note:** When running on VMs, the labeller has limited access to hardware information and will only publish the following labels: Product Name, Driver Version, and Driver Name. Other hardware-specific properties like port count, port speed, and firmware version may not be available in virtualized environments.

#### Label Key Format for Homogeneous vs Heterogeneous Nodes

Homogeneous nodes are nodes in which all NICs are of the same model. This means every NIC installed on the node shares identical hardware characteristics, such as vendor, model, and supported features.

Heterogeneous nodes, on the other hand, have NICs of different models. These nodes may contain NICs from various vendors or with varying capabilities, speeds, and features. This diversity requires a more granular approach to labeling and feature identification.

For heterogeneous nodes, the label prefix includes a differentiator to distinguish between different NIC models and their respective features.

Example labels on a heterogeneous node:
```
amd.com/nic.count=2
amd.com/nic.pollara-1q400p.count=1
amd.com/nic.<some-other-nic>.count=1
amd.com/nic.pollara-1q400p.product-name=POLLARA_1x400G_QSFP112
amd.com/nic.<some-other-nic>.product-name=<some-other-nic-full-name>
...
```

## Usage example with label selector

Once the Node Labeller is deployed and functional, you can select specific nodes via Kubernetes' label selector. For example, to select nodes that have NICs with 100G port speed attached:

    $ kubectl get nodes -l amd.com/nic.port-speed=100G

## Building From Source

### Prerequisites

Before building from source, ensure the following dependencies are installed:

- Go (>= 1.24)
- Docker
- Make

After installing the prerequisites, you can build the project binary by running:

```bash
make build
```
The built binary will be located in the `build/` directory.

To build a Docker image for the project, run:

```bash
make docker-build
```
This will build the Docker image using the current configuration.

To push the built Docker image to the configured registry, use:

```bash
make docker-push
```

### Configurable Environment Variables
You can configure the image name, tag, and registry using environment variables:

- `DOCKER_REGISTRY`: Docker registry (default: `docker.io/rocm`)
- `IMAGE_NAME`: Image name (default: `k8s-network-node-labeller`)
- `IMAGE_TAG`: Image tag (default: `dev`)

For example:
```bash
DOCKER_REGISTRY=myregistry IMAGE_NAME=myimage IMAGE_TAG=latest make docker-build
```

Refer to the [Makefile](./Makefile) for more variables and options that can be configured for your build and deployment workflow.

## Running Unit Tests
To run all tests, use:

```bash
make test
```
This will execute all Go tests in the repository and display verbose output.
