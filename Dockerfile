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

ARG BASE_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.5

FROM docker.io/golang:1.24-alpine3.21 AS gobuilder
RUN apk add --no-cache git make
RUN mkdir -p /go/src/github.com/ROCm/k8s-network-node-labeller
ADD . /go/src/github.com/ROCm/k8s-network-node-labeller
WORKDIR /go/src/github.com/ROCm/k8s-network-node-labeller
RUN make build

FROM ${BASE_IMAGE} AS nicctlbuilder
ARG AINIC_VERSION=1.117.5-a-56
RUN echo "${AINIC_VERSION}" > /etc/dnf/vars/amdainicver
ENV DIST=el9
RUN curl -o /etc/yum.repos.d/amdainic.repo \
    https://repo.radeon.com/amdainic/pensando/${DIST}/amdainic-${DIST}.repo
RUN curl -L -o /tmp/dtc.rpm https://download.rockylinux.org/pub/rocky/9/AppStream/x86_64/os/Packages/d/dtc-1.6.0-7.el9.x86_64.rpm && \
    rpm -ivh --nodeps /tmp/dtc.rpm && \
    rm -f /tmp/dtc.rpm
RUN microdnf update -y && \
    microdnf install -y nicctl && \
    rm -rf /var/cache/yum && rm -rf /var/cache/dnf && microdnf clean all

# Export binaries and needed libs
RUN mkdir -p /export/bin /export/lib64
RUN cp -v /usr/sbin/nicctl /export/bin/
RUN cp -v /lib64/libpci* /export/lib64/

FROM ${BASE_IMAGE}
LABEL \
    org.opencontainers.image.source="https://github.com/ROCm/k8s-network-node-labeller" \
    org.opencontainers.image.authors="Shiv Tyagi <Shiv.Tyagi@amd.com>" \
    org.opencontainers.image.vendor="Advanced Micro Devices, Inc." \
    org.opencontainers.image.licenses="Apache-2.0" \
    ainic_version=${AINIC_VERSION}

COPY --from=nicctlbuilder /export/bin/nicctl /usr/sbin/nicctl
COPY --from=nicctlbuilder /export/lib64/libpci* /lib64/

RUN microdnf update -y && microdnf install -y pciutils jq kmod && \
    rm -rf /var/cache/yum && rm -rf /var/cache/dnf && microdnf clean all
WORKDIR /root/
COPY --from=gobuilder /go/src/github.com/ROCm/k8s-network-node-labeller/build/network-node-labeller .
CMD ["./network-node-labeller"]
