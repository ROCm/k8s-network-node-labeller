# Plan: Multi-Version nicctl Bundling for Node Labeller Docker Image

**Date**: 2026-08-17
**Author**: Yuva Shankar
**Status**: Implemented
**Reference**: [pensando/k8s-network-device-plugin#58](https://github.com/pensando/k8s-network-device-plugin/pull/58)

---

## Context

PR [pensando/k8s-network-device-plugin#58](https://github.com/pensando/k8s-network-device-plugin/pull/58) implemented multi-version nicctl bundling in the NIC device-plugin image. PR [pensando/device-metrics-exporter#1542](https://github.com/pensando/device-metrics-exporter/pull/1542) adapted the same pattern for the metrics exporter. This plan adapts it to the k8s-network-node-labeller Docker image so all three images support automatic firmware detection and version selection at container startup.

**Problem**: A single nicctl version is baked into the image. Clusters with mixed NIC firmware versions need separate images per firmware version.

**Solution**: Bundle 1-5 nicctl versions per image. The bootstrap (latest) is stored uncompressed (~73 MB); older versions are xz-compressed (~8.4 MB each). At startup, a wrapper script detects NIC firmware and selects the correct binary.

**Image Size Impact**:
- 1 version: ~80 MB nicctl overhead
- 2 versions (default): ~88 MB (+8.4 MB)
- 5 versions (max): ~113 MB (+34 MB)

---

## Default Flow

**Root Makefile is the single source of truth** for version defaults.

| Location | AINIC_VERSIONS | BOOTSTRAP_VERSION |
|---|---|---|
| Root `Makefile` (defaults) | `1.117.5-a-77,1.117.5-a-147` | `1.117.5-a-147` |
| `Dockerfile` ARGs | `"1.117.5-a-77,1.117.5-a-147"` | `""` (empty triggers "use last in list" fallback) |

---

## Files Changed

### 1. `nicctl-setup.sh` — NEW FILE (repo root)

Entrypoint wrapper adapted from k8s-ndp `images/nicctl-setup.sh`. Logic:

1. If `/usr/sbin/nicctl-bootstrap` doesn't exist → single-version build, skip to CMD via `exec "$@"`
2. Read bootstrap version from `/opt/bootstrap-version.txt`
3. Detect firmware via `nicctl-bootstrap show firmware`, parse `Firmware-[AB]` field
4. Match against bundled versions: bootstrap → symlink, compressed → decompress, unknown → fallback to bootstrap with WARNING
5. Verify `nicctl --version`, then `exec "$@"`

**Key difference from reference repos**: This repo has no shell entrypoint — the Go binary `./network-node-labeller` is the CMD directly. So `nicctl-setup.sh` ends with `exec "$@"` (which runs CMD) instead of chaining to another shell script.

### 2. `Dockerfile` — REWRITE of nicctlbuilder and final stages

**Top-level ARGs** (replace single `AINIC_VERSION`):
```dockerfile
ARG AINIC_VERSIONS="1.117.5-a-77,1.117.5-a-147"
ARG BOOTSTRAP_VERSION=""
```

**Stage 1 (gobuilder)**: Unchanged.

**Stage 2 (nicctlbuilder)**: Rewritten to:
- Add `SHELL ["/bin/bash", "-o", "pipefail", "-c"]`
- Install `binutils` (strip) and `xz` as build deps
- Parse `AINIC_VERSIONS` as CSV, validate (non-empty, max 5, bootstrap in list)
- Single version (COUNT=1): install + strip → `/export/bin/nicctl`
- Multi-version: bootstrap stripped → `/export/bin/nicctl-bootstrap`, others → `/export/nicctl-versions/nicctl-{ver}.xz`
- Record bootstrap version in `/export/bootstrap-version.txt`

**Stage 3 (final)**: Changes:
- `ARG AINIC_VERSIONS` (replaces `ARG AINIC_VERSION`)
- `COPY --from=nicctlbuilder /export/bin/nicctl* /usr/sbin/`
- `COPY --from=nicctlbuilder /export/nicctl-versions /opt/nicctl-versions`
- `COPY --from=nicctlbuilder /export/bootstrap-version.txt /opt/`
- Adds `xz` to microdnf install line
- `COPY nicctl-setup.sh /nicctl-setup.sh` + `chmod +x`
- Label: `ainic_bundled_versions=${AINIC_VERSIONS}` (replaces `ainic_version`)
- `ENTRYPOINT ["/nicctl-setup.sh"]` + `CMD ["./network-node-labeller"]`

### 3. `Makefile` — lines 30-31 and docker-build target

Replace version variables:
```makefile
# Old:
AINIC_VERSION ?= 1.117.5-a-56
DOCKER_ARGS ?= AINIC_VERSION=$(AINIC_VERSION)

# New:
AINIC_VERSIONS ?= 1.117.5-a-77,1.117.5-a-147
BOOTSTRAP_VERSION ?= 1.117.5-a-147
```

Replace docker-build target to pass explicit `--build-arg` flags:
```makefile
docker build $(DOCKER_LABELS_OPTION) \
    --build-arg AINIC_VERSIONS=$(AINIC_VERSIONS) \
    $(if $(BOOTSTRAP_VERSION),--build-arg BOOTSTRAP_VERSION=$(BOOTSTRAP_VERSION)) \
    -t ${IMG} .
```

### 4. No changes needed

- `.job.yml` — CI uses `make docker-build` without overriding AINIC_VERSION; picks up new Makefile defaults automatically
- Go code (`internal/nicctl/client.go`) — `os.Stat("/usr/sbin/nicctl")` follows symlinks; binary always at `/usr/sbin/nicctl` after setup
- `internal/discoverer/nicctldiscoverer.go` — `NicctlBinaryPath = "/usr/sbin/nicctl"` still valid
- Helm charts — no nicctl-related configuration

---

## Backward Compatibility

| Scenario | Behavior |
|----------|----------|
| `make docker-build` (default) | Multi-version: `1.117.5-a-77` (compressed) + `1.117.5-a-147` (bootstrap), runtime detection |
| `make docker-build AINIC_VERSIONS=1.117.5-a-147` | Single version, no `nicctl-bootstrap`, setup script does `exec "$@"` immediately |
| `make docker-build AINIC_VERSIONS=a-56,a-77,a-147 BOOTSTRAP_VERSION=a-147` | 3-version build |
| Go code (`nicctl/client.go`) | Unchanged — `os.Stat(binaryPath)` finds `/usr/sbin/nicctl` in both modes |

---

## Verification

1. **Default build**: `make docker-build` → image builds with 2 versions (`1.117.5-a-77`, `1.117.5-a-147`)
2. **Single-version override**: `make docker-build AINIC_VERSIONS=1.117.5-a-147` → single version, no bootstrap binary
3. **Startup (no NIC)**: `docker run --rm <image>` → uses bootstrap, logs WARNING about no NICs, then starts Go binary
4. **Image size**: default ~88 MB, single ~80 MB (verify with `docker images`)
5. **nicctl-setup.sh short-circuit**: single-version image should NOT have `/usr/sbin/nicctl-bootstrap`
6. **Entrypoint chain**: `docker run --rm <image> echo hello` → prints "hello" (proves `exec "$@"` works)
