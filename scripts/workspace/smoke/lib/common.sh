#!/usr/bin/env bash
# Shared helpers for Elyro Workspace smoke checks. Sourced by verify.sh and toolchain scripts.

workspace_smoke_build_binary() {
  local root_dir="$1"
  local bin_path="$2"
  echo "building elyro binary for smoke..."
  (
    cd "${root_dir}" || exit 1
    mkdir -p "$(dirname "${bin_path}")"
    go build -o "${bin_path}" ./cmd/elyro
  )
}

workspace_smoke_build_images() {
  local root_dir="$1"
  local arch
  arch="$(go env GOARCH)"
  case "${arch}" in amd64|arm64) ;; *) printf 'unsupported smoke architecture: %s\n' "${arch}" >&2; return 1 ;; esac
  (
    cd "${root_dir}" || exit 1
    make "workspace-python-image-build-${arch}" "workspace-go-image-build-${arch}" \
      "workspace-node-image-build-${arch}" "workspace-java-image-build-${arch}"
  )
  export ELYRO_IMAGE_PREFIX=elyro
}

# Poll until curl succeeds or attempts exhausted. Returns non-zero if still failing.
workspace_smoke_wait_http() {
  local url="$1"
  local max_attempts="${2:-15}"

  local _
  for _ in $(seq 1 "${max_attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}
