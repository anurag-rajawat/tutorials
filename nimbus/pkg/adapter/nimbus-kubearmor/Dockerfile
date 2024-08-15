FROM golang:1.22 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /nimbus

# relative deps requried by the adapter
COPY api/ api/
COPY pkg/ pkg/
COPY go.mod go.mod
COPY go.sum go.sum

# nimbus-kubearmor directory
ARG ADAPTER_DIR=pkg/adapter/nimbus-kubearmor
WORKDIR /nimbus/$ADAPTER_DIR

# # Copy Go modules and manifests
COPY $ADAPTER_DIR/go.mod go.mod
COPY $ADAPTER_DIR/go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY $ADAPTER_DIR/builder builder
COPY $ADAPTER_DIR/cmd cmd
COPY $ADAPTER_DIR/manager manager
COPY $ADAPTER_DIR/watcher watcher

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -ldflags="-s" -o nimbus-kubearmor cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /nimbus/pkg/adapter/nimbus-kubearmor .
USER 65532:65532

ENTRYPOINT ["/nimbus-kubearmor"]
