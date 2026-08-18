#!/bin/sh
#
# Copyright (c) 2025 Advanced Micro Devices, Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# nicctl version detection and setup for multi-version builds.
# Single-version builds have /usr/sbin/nicctl directly — no setup needed.

set -e

log() {
    echo "[nicctl-setup] $1"
}

# Single-version build: nicctl already in place, skip to CMD
if [ ! -f /usr/sbin/nicctl-bootstrap ]; then
    exec "$@"
fi

# Multi-version build: detect firmware and select correct nicctl
if [ ! -f /opt/bootstrap-version.txt ]; then
    log "ERROR: /opt/bootstrap-version.txt missing in multi-version image"
    exit 1
fi
BOOTSTRAP_VERSION=$(cat /opt/bootstrap-version.txt)
log "Multi-version build (bootstrap: $BOOTSTRAP_VERSION)"

# Detect firmware version from NIC hardware
FIRMWARE_VER=""
if FIRMWARE_OUTPUT=$(/usr/sbin/nicctl-bootstrap show firmware 2>/dev/null); then
    FIRMWARE_VER=$(echo "$FIRMWARE_OUTPUT" | grep -E "^Firmware-[AB]" | awk '{print $3}' | head -1)
    log "Detected firmware: $FIRMWARE_VER"
fi

if [ -z "$FIRMWARE_VER" ]; then
    log "WARNING: Could not detect firmware (no NICs?), using bootstrap"
    ln -sf /usr/sbin/nicctl-bootstrap /usr/sbin/nicctl
elif [ "$FIRMWARE_VER" = "$BOOTSTRAP_VERSION" ]; then
    log "Firmware matches bootstrap"
    ln -sf /usr/sbin/nicctl-bootstrap /usr/sbin/nicctl
else
    COMPRESSED="/opt/nicctl-versions/nicctl-${FIRMWARE_VER}.xz"
    if [ -f "$COMPRESSED" ]; then
        log "Decompressing nicctl $FIRMWARE_VER..."
        unxz -c "$COMPRESSED" > /usr/sbin/.nicctl.tmp || {
            log "ERROR: Failed to decompress nicctl $FIRMWARE_VER"
            exit 1
        }
        chmod +x /usr/sbin/.nicctl.tmp || {
            log "ERROR: Failed to make nicctl $FIRMWARE_VER executable"
            exit 1
        }
        mv /usr/sbin/.nicctl.tmp /usr/sbin/nicctl
    else
        log "WARNING: nicctl $FIRMWARE_VER not bundled, using bootstrap"
        ln -sf /usr/sbin/nicctl-bootstrap /usr/sbin/nicctl
    fi
fi

# Verify nicctl works
if ! /usr/sbin/nicctl --version >/dev/null 2>&1; then
    log "ERROR: nicctl not functional"
    exit 1
fi

log "Ready: $(/usr/sbin/nicctl --version 2>/dev/null)"
exec "$@"
