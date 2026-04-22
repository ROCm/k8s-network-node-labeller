# network-node-labeller-charts

![Version: v1.2.0](https://img.shields.io/badge/Version-v1.2.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v1.2.0](https://img.shields.io/badge/AppVersion-v1.2.0-informational?style=flat-square)

A Helm chart for AMD AINIC Node Labeller

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Shiv Tyagi | <Shiv.Tyagi@amd.com> |  |
| Shrey Ajmera | <sajmera@amd.com> |  |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| annotations | object | `{}` | Additional annotations to add to the DaemonSet pods |
| hooks | object | `{"utilsImage":"docker.io/rocm/network-operator-utils:v1.1.0"}` | Hook configuration |
| hooks.utilsImage | string | `"docker.io/rocm/network-operator-utils:v1.1.0"` | Image used by pre-delete hook to gracefully shut down labeller pods |
| image | object | `{"pullPolicy":"IfNotPresent","repository":"docker.io/rocm/k8s-network-node-labeller","tag":"v1.2.0"}` | Container image configuration |
| image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy |
| image.repository | string | `"docker.io/rocm/k8s-network-node-labeller"` | Container image repository |
| image.tag | string | `"v1.2.0"` | Container image tag |
| imagePullSecrets | list | `[]` | Image pull secrets for private registries |
| nodeSelector | object | `{}` | Node selector to constrain pods to specific nodes |
| resources | object | `{}` | Resource limits and requests for the containers |
| securityContext | object | `{"privileged":true}` | Security context for the DaemonSet pods |
| securityContext.privileged | bool | `true` | Run containers in privileged mode. nicctl requires the container to run in privileged mode to detect AINICs. It is recommended to set this to true for bare metal nodes. For VM nodes, nicctl is not used and this can be set to false. |
| serviceAccountName | string | `""` | Service account name to use. If not set, defaults to Release name |
| tolerations | list | `[{"key":"CriticalAddonsOnly","operator":"Exists"}]` | Tolerations for pod assignment to nodes with taints |
| updateStrategy | object | `{"rollingUpdate":{"maxUnavailable":1},"type":"RollingUpdate"}` | Update strategy for the DaemonSet |
| updateStrategy.rollingUpdate.maxUnavailable | int | `1` | Maximum number of pods that can be unavailable during update |
| updateStrategy.type | string | `"RollingUpdate"` | Type of update strategy (RollingUpdate or OnDelete) |

