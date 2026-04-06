# Import development related environment variables from dev.env
ifneq ("$(wildcard dev.env)","")
    include dev.env
endif

PROJECT_VERSION ?= v1.1.0

DOCKER_REGISTRY ?= docker.io/rocm
IMAGE_NAME ?= k8s-network-node-labeller
IMAGE_TAG_BASE ?= $(DOCKER_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG ?= $(PROJECT_VERSION)
IMG ?= $(IMAGE_TAG_BASE):$(IMAGE_TAG)

IMG_FILE_NAME ?= $(IMAGE_NAME)-$(IMAGE_TAG)
BINARY_NAME ?= network-node-labeller
BUILDDIR=$(CURDIR)/build
IMG_DIR=$(BUILDDIR)/docker
DIRS=$(BUILDDIR) $(IMG_DIR)

HELM_CHART_DIR ?= $(CURDIR)/helm-charts-k8s
HELM_CHART_VERSION ?= $(PROJECT_VERSION)
HELM_CHART_APP_VERSION ?= $(IMAGE_TAG)
HELM_CHART_NAME ?= network-node-labeller-charts
HELM_RELEASE_NAME ?= amd-network-node-labeller
HELM_RELEASE_NAMESPACE ?= kube-amd-network
HELM_OUTPUT_FILE_PREFIX ?= k8s-network-node-labeller-helm-k8s
HELM_OUTPUT_FILE_NAME ?= $(HELM_OUTPUT_FILE_PREFIX)-$(PROJECT_VERSION).tgz
CHART_DEST ?= $(HELM_CHART_DIR)/$(HELM_OUTPUT_FILE_NAME)

AINIC_VERSION ?= 1.117.5-a-56
DOCKER_ARGS ?= AINIC_VERSION=$(AINIC_VERSION)

GO_BUILD_OPTS ?=

# Build environment container variables
DOCKER_BUILDER_IMAGE_NAME ?= k8s-network-node-labeller-build
DOCKER_BUILDER_IMAGE_NAME_BASE ?= $(DOCKER_REGISTRY)/$(DOCKER_BUILDER_IMAGE_NAME)
DOCKER_BUILDER_IMAGE_TAG ?= v1.1
DOCKER_BUILDER_IMAGE ?= $(DOCKER_BUILDER_IMAGE_NAME_BASE):$(DOCKER_BUILDER_IMAGE_TAG)
CONTAINER_WORKDIR ?= /k8s-network-node-labeller
CUR_USER:=$(shell whoami)
CONTAINER_NAME:=${CUR_USER}-k8s-network-node-labeller-build

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin GOFLAGS=-mod=mod go install $(2);\
}
endef

.PHONY: all
all: build test

.PHONY: create-dirs
create-dirs:
	$(info Creating directories $(DIRS)...)
	mkdir -p $(DIRS)
	$(info Done!)

.PHONY: docker-build
docker-build:
	$(eval DOCKER_LABELS_OPTION := $(if $(DOCKER_LABELS),--label "$(DOCKER_LABELS)"))
	$(eval DOCKER_ARGS_OPTION := $(if $(DOCKER_ARGS),--build-arg "$(DOCKER_ARGS)"))
	docker build $(DOCKER_LABELS_OPTION) $(DOCKER_ARGS_OPTION) -t ${IMG} .

.PHONY: docker-save
docker-save: create-dirs
	$(info Saving docker image to $(IMG_FILE_NAME).tar.gz...)
	docker save $(IMG) | gzip > $(IMG_DIR)/$(IMG_FILE_NAME).tar.gz

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: docker-build-env
docker-build-env: ## Build the build environment container
	$(info Building build environment container $(DOCKER_BUILDER_IMAGE)...)
	docker build -t $(DOCKER_BUILDER_IMAGE) -f tools/base-image/Dockerfile tools/base-image/
	$(info Done!)

.PHONY: docker-shell
docker-shell: docker-build-env ## Start a shell in the Docker build container
	@echo "Starting a shell in the Docker build container..."
	@docker run --rm -it --privileged \
		--name ${CONTAINER_NAME} \
		-e "USER_NAME=${CUR_USER}" \
		-e "USER_UID=$(shell id -u)" \
		-e "USER_GID=$(shell id -g)" \
		-v $(CURDIR):$(CONTAINER_WORKDIR) \
		-v $(CURDIR):/home/${CUR_USER}/go/src/github.com/ROCm/k8s-network-node-labeller \
		-v $(HOME)/.ssh:/home/${CUR_USER}/.ssh \
		-w $(CONTAINER_WORKDIR) \
		$(DOCKER_BUILDER_IMAGE) \
		'bash -ic "source ~/.bash_profile && cd $(CONTAINER_WORKDIR) && git config --global --add safe.directory $(CONTAINER_WORKDIR) && bash"'

.PHONY: build
build: create-dirs
	$(info Building $(BINARY_NAME)...)
	cd $(CURDIR)/cmd && $(GO_BUILD_OPTS) go build -o $(BUILDDIR)/$(BINARY_NAME) -v ./main.go
	$(info Done!)

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	$(info Running tests...)
	go test ./... -v
	$(info Done!)

copyrights:
	GOFLAGS=-mod=mod go run tools/build/copyright/main.go && ${MAKE} fmt && ./tools/build/check-local-files.sh

.PHONY: helm-update-meta
helm-update-meta:
	sed -i -e 's|appVersion:.*$$|appVersion: ${HELM_CHART_APP_VERSION}|' $(HELM_CHART_DIR)/Chart.yaml
	sed -i '0,/version:/s|version:.*|version: ${HELM_CHART_VERSION}|' $(HELM_CHART_DIR)/Chart.yaml
	sed -i -e 's|name: network-node-labeller-charts|name: ${HELM_CHART_NAME}|' $(HELM_CHART_DIR)/Chart.yaml
	sed -i -e 's|repository:.*$$|repository: ${IMAGE_TAG_BASE}|' $(HELM_CHART_DIR)/values.yaml
	sed -i -e 's|tag:.*$$|tag: ${IMAGE_TAG}|' $(HELM_CHART_DIR)/values.yaml

.PHONY: helm-lint
helm-lint:
	helm lint $(HELM_CHART_DIR)

.PHONY: lint
lint:
	@[ -f $(PROJECT_DIR)/bin/golangci-lint ] || { \
		echo "Installing golangci-lint to $(PROJECT_DIR)/bin..." ;\
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(PROJECT_DIR)/bin v2.7.2 ;\
	}
	$(PROJECT_DIR)/bin/golangci-lint run ./...

HELMDOCS = $(shell pwd)/bin/helm-docs
.PHONY: helm-docs
helm-docs: ## Download helm-docs locally if necessary
	$(call go-get-tool,$(HELMDOCS),github.com/norwoodj/helm-docs/cmd/helm-docs@v1.12.0)
	$(HELMDOCS) -c $(shell pwd)/helm-charts-k8s/ -g $(shell pwd)/helm-charts-k8s -u --ignore-non-descriptions

.PHONY: cleanup-stale-charts
cleanup-stale-charts:
	rm -f $(HELM_CHART_DIR)/$(HELM_CHART_NAME)-*.tgz $(HELM_CHART_DIR)/$(HELM_OUTPUT_FILE_PREFIX)-*.tgz

.PHONY: helm-package
helm-package: cleanup-stale-charts
	helm package $(HELM_CHART_DIR)/ --destination $(HELM_CHART_DIR)

.PHONY: helm
helm: helm-update-meta helm-lint helm-package helm-docs
	cp $(HELM_CHART_DIR)/$(HELM_CHART_NAME)-$(HELM_CHART_VERSION).tgz $(CHART_DEST)
	@echo "Helm chart is ready in $(CHART_DEST)"

.PHONY: helm-install
helm-install:
	helm install $(HELM_RELEASE_NAME) $(HELM_CHART_DIR)/$(HELM_CHART_NAME)-$(HELM_CHART_VERSION).tgz -f $(HELM_CHART_DIR)/values.yaml --namespace $(HELM_RELEASE_NAMESPACE) --create-namespace

.PHONY: helm-uninstall
helm-uninstall:
	helm uninstall $(HELM_RELEASE_NAME) --namespace $(HELM_RELEASE_NAMESPACE)
