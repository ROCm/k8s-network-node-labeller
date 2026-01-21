# Import development related environment variables from dev.env
ifneq ("$(wildcard dev.env)","")
    include dev.env
endif

PROJECT_VERSION ?= v1.0.0

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

AINIC_VERSION ?= 1.117.5-a-56
DOCKER_ARGS ?= AINIC_VERSION=$(AINIC_VERSION)

GO_BUILD_OPTS ?=

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
	cd $(HELM_CHART_DIR); helm lint

HELMDOCS = $(shell pwd)/bin/helm-docs
.PHONY: helm-docs
helm-docs: ## Download helm-docs locally if necessary
	$(call go-get-tool,$(HELMDOCS),github.com/norwoodj/helm-docs/cmd/helm-docs@v1.12.0)
	$(HELMDOCS) -c $(shell pwd)/helm-charts-k8s/ -g $(shell pwd)/helm-charts-k8s -u --ignore-non-descriptions

.PHONY: helm-package
helm-package:
	helm package $(HELM_CHART_DIR)/ --destination $(HELM_CHART_DIR)

.PHONY: helm
helm: helm-update-meta helm-lint helm-package helm-docs
	$(eval HELM_OUTPUT_FILE_NAME ?= k8s-network-node-labeller-helm-k8s-$(PROJECT_VERSION).tgz)
	$(eval CHART_DEST := $(HELM_CHART_DIR)/$(HELM_OUTPUT_FILE_NAME))
	cp $(HELM_CHART_DIR)/$(HELM_CHART_NAME)-$(HELM_CHART_VERSION).tgz $(CHART_DEST)
	@echo "Helm chart is ready in $(CHART_DEST)"

.PHONY: helm-install
helm-install:
	helm install $(HELM_RELEASE_NAME) $(HELM_CHART_DIR)/$(HELM_CHART_NAME)-$(HELM_CHART_VERSION).tgz -f $(HELM_CHART_DIR)/values.yaml --namespace $(HELM_RELEASE_NAMESPACE) --create-namespace

.PHONY: helm-uninstall
helm-uninstall:
	helm uninstall $(HELM_RELEASE_NAME) --namespace $(HELM_RELEASE_NAMESPACE)
