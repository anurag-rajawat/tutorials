IMG ?= anuragrajawat/httpfilter
TAG ?= final
CONTAINER_TOOL ?= docker

.PHONY: build
build:
	@cargo build --target wasm32-unknown-unknown --release

.PHONY: toolchain
toolchain: ## Install Rust WASM toolchain
	@test rustup || curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
	@rustup target add wasm32-unknown-unknown

.PHONY: clean
clean:
	@cargo clean

.PHONY: imagex
imagex:
	$(CONTAINER_TOOL) buildx build --push --platform=linux/arm64,linux/amd64 --tag ${IMG}:${TAG} .

.PHONY: push
push:
	$(CONTAINER_TOOL) push ${IMG}:${TAG}
