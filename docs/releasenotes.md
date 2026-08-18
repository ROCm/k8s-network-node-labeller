# Release Notes

## v1.3.0

### Release Highlights

- Multi-version nicctl bundling with automatic firmware detection at container startup
- Images now support up to 5 nicctl versions, enabling a single image to work across clusters with mixed NIC firmware versions

### Hardware Support

- **AMD Pensando™ Pollara AI NIC**
  - Supported AINIC firmware: `1.117.5-a-77`, `1.117.5-a-147`

### Platform Support

- **Kubernetes 1.29+**

## v1.2.0

### Release Highlights

- Added Helm charts for standalone installations, enabling deployment of the Node Labeller independently of the AMD Network Operator

### Hardware Support

- **AMD Pensando™ Pollara AI NIC**
  - Supported AINIC firmware: `1.117.5-a-56`, `1.117.5-a-77`

### Platform Support

- **Kubernetes 1.29+**

## v1.1.0

### Release Highlights

- The NICCTL tool is now bundled within the Node Labeller image, allowing the component to run independently of host OS versions

### Hardware Support

- **AMD Pensando™ Pollara AI NIC**
  - Supported AINIC firmware: `1.117.5-a-56`

### Platform Support

- **Kubernetes 1.29+**

## v1.0.0

This is the initial release of the AMD Kubernetes Network Node Labeller. The Node Labeller automatically discovers AMD AINICs on Kubernetes cluster nodes and applies labels describing NIC properties, enabling workload scheduling based on network hardware capabilities.

### Release Highlights

- Automatic discovery and labeling of AMD AINIC properties on Kubernetes nodes
- Support for both homogeneous and heterogeneous node configurations
- DaemonSet-based deployment for cluster-wide node labeling

### Hardware Support

- **AMD Pensando™ Pollara AI NIC**
  - AINIC firmware: N/A (requires host `nicctl`)

### Platform Support

- **Kubernetes 1.29+**
