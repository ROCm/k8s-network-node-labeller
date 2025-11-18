# Copyright (c) 2025 Advanced Micro Devices, Inc.  All rights reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

FROM docker.io/golang:1.24-alpine3.21 AS builder
RUN apk add --no-cache git make
RUN mkdir -p /go/src/github.com/ROCm/k8s-network-node-labeller
ADD . /go/src/github.com/ROCm/k8s-network-node-labeller
WORKDIR /go/src/github.com/ROCm/k8s-network-node-labeller
RUN make build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.5
LABEL \
    org.opencontainers.image.source="https://github.com/ROCm/k8s-network-node-labeller" \
    org.opencontainers.image.authors="Shiv Tyagi <Shiv.Tyagi@amd.com>" \
    org.opencontainers.image.vendor="Advanced Micro Devices, Inc." \
    org.opencontainers.image.licenses="Apache-2.0"

RUN microdnf update -y && microdnf install -y pciutils jq kmod && \
    microdnf clean all
WORKDIR /root/
COPY --from=builder /go/src/github.com/ROCm/k8s-network-node-labeller/build/network-node-labeller .
CMD ["./network-node-labeller"]
