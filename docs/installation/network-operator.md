# Kubernetes (Network Operator) installation

This page explains how to install AMD Kubernetes Network Node Labeller using the AMD Network Operator.

## System requirements

- Kubernetes cluster v1.29.0 or later
- Helm v3.2.0 or later
- `kubectl` command-line tool configured with access to the cluster
- Nodes equipped with AMD AINICs
- AMD AINIC drivers installed on the nodes

## Installation

See the [Network Operator Documentation](https://instinct.docs.amd.com/projects/network-operator/en/latest/installation/kubernetes-helm.html) for installation instructions.

## Enabling and configuring the Node Labeller

Once the Network Operator is installed, the Node Labeller can be enabled and configured through the `spec.devicePlugin` section of the `NetworkConfig` custom resource. See [Device Plugin & Node Labeller](https://instinct.docs.amd.com/projects/network-operator/en/latest/device_plugin/deviceplugin.html) for details.
