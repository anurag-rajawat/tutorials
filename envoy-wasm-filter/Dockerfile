FROM rust:1.81.0 AS builder

WORKDIR /envoy-plugin

COPY . .

RUN make toolchain build

FROM scratch

COPY --from=builder /envoy-plugin/target/wasm32-unknown-unknown/release/httpfilter.wasm ./plugin.wasm
