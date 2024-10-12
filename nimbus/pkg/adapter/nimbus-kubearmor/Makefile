# Image URL to use all building/pushing image targets
IMG ?= anuragrajawat/nimbus-kubearmor
# Image Tag to use all building/pushing image targets
TAG ?= v0.1

CONTAINER_TOOL ?= docker
BINARY ?= bin/nimbus-kubearmor
CONTROLLER_TOOLS_VERSION ?= v0.14.0
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build nimbus-kubearmor executable.
	@go build -ldflags="-s" -o ${BINARY} cmd/main.go

run: build ## Run nimbus-kubearmor.
	@./${BINARY}

.PHONY: image
image: ## Build nimbus-kubearmor container image.
	$(CONTAINER_TOOL) build -t ${IMG}:${TAG} --build-arg VERSION=${TAG} -f ./Dockerfile ../../../

.PHONY: push-image
push-image: ## Push nimbus-kubearmor container image.
	$(CONTAINER_TOOL) push ${IMG}:${TAG}

PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: imagex
imagex: ## Build and push container image for cross-platform support
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --build-arg VERSION=${TAG} --tag ${IMG}:${TAG} -f Dockerfile.cross ../../../ || { $(CONTAINER_TOOL) buildx rm project-v3-builder; rm Dockerfile.cross; exit 1; }
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

.PHONY: generate
generate: controller-gen ## Generate ClusterRole.
	$(CONTROLLER_GEN) rbac:roleName=nimbus-kubearmor paths="./..." output:dir=../../../deployment/nimbus-kubearmor/templates/

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
