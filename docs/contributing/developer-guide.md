# Developer Guide

This guide provides information for developers who want to build, test, or modify the AMD Kubernetes Network Node Labeller.

```{warning}
This project is not currently accepting external contributions.
```

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Setup](#development-setup)
3. [Making Changes](#making-changes)
4. [Additional Resources](#additional-resources)
5. [License](#license)

## Getting Started

### Prerequisites

Before you begin, ensure you have:

* **Git** 2.25+
* **Go** 1.24+ (see `go.mod`)
* **Docker** (for container builds and testing)
* **Kubernetes** familiarity
* **Make**

### Fork & Clone

1. Fork the repository on GitHub

2. Clone your fork locally:

   ```bash
   git clone https://github.com/<your-username>/k8s-network-node-labeller.git
   cd k8s-network-node-labeller
   ```

3. Add the upstream repository:

   ```bash
   git remote add upstream https://github.com/ROCm/k8s-network-node-labeller.git
   ```

4. Fetch the latest changes:

   ```bash
   git fetch upstream main
   ```

## Development Setup

### Build the Project

```bash
# Build the node labeller binary
make build

# Build Docker image
make docker-build

# Run tests
make test
```

> `make docker-build` builds the container image using the [`Dockerfile`](../../Dockerfile) at the root of the repository.

### Development Environment

```bash
# Install dependencies
go mod download

# Run linting
make lint

# Format code
make fmt
```

## Making Changes

* Create a feature or fix branch from `main`
* Keep changes focused and minimal
* Follow existing coding and naming conventions
* Add or update tests when applicable

### Before Committing

1. **Run tests locally**

   ```bash
   make test
   make lint
   ```

2. **Update documentation** if user-facing behavior changes

3. **Add tests** for new features or bug fixes

4. **Format code**

   ```bash
   make fmt
   gofmt -s -w .
   ```

## Additional Resources

* [Kubernetes Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
* [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
* [Effective Go](https://golang.org/doc/effective_go)

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](../../LICENSE) file for details.
