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
ARG AINIC_VERSIONS="1.117.5-a-77,1.117.5-a-147"
ARG BOOTSTRAP_VERSION=""

FROM docker.io/golang:1.24-alpine3.21 AS gobuilder
RUN apk add --no-cache git make
RUN mkdir -p /go/src/github.com/ROCm/k8s-network-node-labeller
ADD . /go/src/github.com/ROCm/k8s-network-node-labeller
WORKDIR /go/src/github.com/ROCm/k8s-network-node-labeller
RUN make build

FROM ${BASE_IMAGE} AS nicctlbuilder
ARG AINIC_VERSIONS
ARG BOOTSTRAP_VERSION
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
ENV DIST=el9

RUN mkdir -p /export/bin /export/lib64 /export/nicctl-versions && \
    IFS=',' read -ra VERSIONS <<< "${AINIC_VERSIONS}" && \
    COUNT="${#VERSIONS[@]}" && \
    if [[ "$COUNT" -eq 0 ]]; then echo "ERROR: AINIC_VERSIONS empty" && exit 1; fi && \
    if [[ "$COUNT" -gt 5 ]]; then echo "ERROR: Max 5 versions, got ${COUNT}" && exit 1; fi && \
    BOOTSTRAP="${BOOTSTRAP_VERSION:-${VERSIONS[-1]}}" && \
    if [[ -n "$BOOTSTRAP_VERSION" ]]; then \
        FOUND=false; for v in "${VERSIONS[@]}"; do [[ "$v" == "$BOOTSTRAP" ]] && FOUND=true; done; \
        if [[ "$FOUND" != "true" ]]; then echo "ERROR: BOOTSTRAP_VERSION '$BOOTSTRAP' not in AINIC_VERSIONS" && exit 1; fi; \
    fi && \
    echo "$BOOTSTRAP" > /etc/dnf/vars/amdainicver && \
    curl -sSL -o /etc/yum.repos.d/amdainic.repo "https://repo.radeon.com/amdainic/pensando/${DIST}/amdainic-${DIST}.repo" && \
    curl -L -o /tmp/dtc.rpm https://download.rockylinux.org/pub/rocky/9/AppStream/x86_64/os/Packages/d/dtc-1.6.0-7.el9.x86_64.rpm && \
    rpm -ivh --nodeps /tmp/dtc.rpm && rm -f /tmp/dtc.rpm && \
    microdnf update -y && \
    microdnf install -y binutils xz && microdnf clean all && \
    echo "$BOOTSTRAP" > /export/bootstrap-version.txt && \
    if [[ "$COUNT" -eq 1 ]]; then \
        echo "Single version: ${VERSIONS[0]}"; \
        echo "${VERSIONS[0]}" > /etc/dnf/vars/amdainicver && \
        microdnf install -y "nicctl-${VERSIONS[0]//-/.}" && \
        strip /usr/sbin/nicctl -o /export/bin/nicctl; \
    else \
        echo "Multi-version build (bootstrap: $BOOTSTRAP)"; \
        echo "$BOOTSTRAP" > /etc/dnf/vars/amdainicver && \
        microdnf install -y "nicctl-${BOOTSTRAP//-/.}" && \
        strip /usr/sbin/nicctl -o /export/bin/nicctl-bootstrap && \
        microdnf remove -y "nicctl-${BOOTSTRAP//-/.}"; \
        for ver in "${VERSIONS[@]}"; do \
            [[ "$ver" == "$BOOTSTRAP" ]] && continue; \
            echo "Compressing: ${ver}"; \
            echo "$ver" > /etc/dnf/vars/amdainicver && \
            microdnf install -y "nicctl-${ver//-/.}" && \
            strip /usr/sbin/nicctl -o /tmp/nicctl && \
            xz -9 -c /tmp/nicctl > "/export/nicctl-versions/nicctl-${ver}.xz" && \
            microdnf remove -y "nicctl-${ver//-/.}"; \
        done; \
    fi && \
    cp /lib64/libpci* /export/lib64/ && \
    microdnf clean all && rm -rf /var/cache/yum /var/cache/dnf

FROM ${BASE_IMAGE}
ARG AINIC_VERSIONS
LABEL \
    org.opencontainers.image.source="https://github.com/ROCm/k8s-network-node-labeller" \
    org.opencontainers.image.authors="Shiv Tyagi <Shiv.Tyagi@amd.com>" \
    org.opencontainers.image.vendor="Advanced Micro Devices, Inc." \
    org.opencontainers.image.licenses="Apache-2.0" \
    ainic_bundled_versions=${AINIC_VERSIONS}

COPY --from=nicctlbuilder /export/bin/nicctl* /usr/sbin/
COPY --from=nicctlbuilder /export/nicctl-versions /opt/nicctl-versions
COPY --from=nicctlbuilder /export/bootstrap-version.txt /opt/
COPY --from=nicctlbuilder /export/lib64/libpci* /lib64/

RUN microdnf update -y && microdnf install -y pciutils jq kmod xz && \
    rm -rf /var/cache/yum && rm -rf /var/cache/dnf && microdnf clean all
WORKDIR /root/
COPY --from=gobuilder /go/src/github.com/ROCm/k8s-network-node-labeller/build/network-node-labeller .
COPY nicctl-setup.sh /nicctl-setup.sh
RUN chmod +x /nicctl-setup.sh

ENTRYPOINT ["/nicctl-setup.sh"]
CMD ["./network-node-labeller"]
