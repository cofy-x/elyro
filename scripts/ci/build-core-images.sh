#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/release/versions.env"

platform="${ELYRO_BUILD_PLATFORM:-linux/amd64}"
version="${ELYRO_BUILD_VERSION:-ci}"
prefix="${ELYRO_BUILD_IMAGE_PREFIX:-elyro}"
mirror_source="${ELYRO_MIRROR_SOURCE:-official}"
proxy_url="$(bash "${ROOT_DIR}/scripts/lib/resolve-elyro-proxy.sh" "${ELYRO_PROXY_URL-}")"
proxy_no_proxy="${ELYRO_PROXY_NO_PROXY:-localhost,127.0.0.1,::1,host.docker.internal,.local}"
goproxy="${ELYRO_GOPROXY:-https://proxy.golang.org,direct}"

case "${platform}" in linux/amd64) arch=amd64 ;; linux/arm64) arch=arm64 ;; *) printf 'unsupported build platform: %s\n' "${platform}" >&2; exit 1 ;; esac
tag="${version}-${arch}"
workspace_base_image="${prefix}/workspace-base:${tag}"

build() {
  local image="$1" dockerfile="$2" context="$3"
  local started=${SECONDS}
  local -a docker_args=(
    --platform "${platform}"
    --build-arg "HTTP_PROXY=${proxy_url}"
    --build-arg "HTTPS_PROXY=${proxy_url}"
    --build-arg "ALL_PROXY=${proxy_url}"
    --build-arg "NO_PROXY=${proxy_no_proxy}"
    --build-arg "MIRROR_SOURCE=${mirror_source}"
    --build-arg "ELYRO_PROXY_URL=${proxy_url}"
    --build-arg "ELYRO_PROXY_NO_PROXY=${proxy_no_proxy}"
    --file "${ROOT_DIR}/${dockerfile}"
    --tag "${image}"
  )
  shift 3
  if [[ -n "${ELYRO_BUILD_REVISION:-}" ]]; then
    [[ -n "${ELYRO_BUILD_SOURCE:-}" && -n "${ELYRO_BUILD_VERSION_LABEL:-}" ]] || {
      printf 'ELYRO_BUILD_SOURCE and ELYRO_BUILD_VERSION_LABEL are required with ELYRO_BUILD_REVISION\n' >&2
      exit 2
    }
    docker_args+=(
      --label "org.opencontainers.image.source=${ELYRO_BUILD_SOURCE}"
      --label "org.opencontainers.image.url=${ELYRO_BUILD_SOURCE}"
      --label "org.opencontainers.image.version=${ELYRO_BUILD_VERSION_LABEL}"
      --label "org.opencontainers.image.revision=${ELYRO_BUILD_REVISION}"
      --label "org.opencontainers.image.licenses=Apache-2.0"
    )
  fi
  printf 'building %s\n' "${image}"
  DOCKER_BUILDKIT=1 docker build "${docker_args[@]}" "$@" "${ROOT_DIR}/${context}"
  printf 'built %s in %ss\n' "${image}" "$((SECONDS - started))"
}

build "${workspace_base_image}" images/workspace-base/Dockerfile images/workspace-base
build "${prefix}/workspace-python:${tag}" images/workspace-python/Dockerfile images/workspace-python \
  --build-arg "WORKSPACE_BASE_IMAGE=${workspace_base_image}" --build-arg "UV_VERSION=${ELYRO_UV_VERSION}" \
  --build-arg "UV_SHA256_AMD64=${ELYRO_UV_SHA256_AMD64}" --build-arg "UV_SHA256_ARM64=${ELYRO_UV_SHA256_ARM64}"
build "${prefix}/workspace-go:${tag}" images/workspace-go/Dockerfile images/workspace-go \
  --build-arg "WORKSPACE_BASE_IMAGE=${workspace_base_image}" --build-arg "GO_VERSION=${ELYRO_GO_VERSION}" \
  --build-arg "GO_SHA256_AMD64=${ELYRO_GO_SHA256_AMD64}" --build-arg "GO_SHA256_ARM64=${ELYRO_GO_SHA256_ARM64}" \
  --build-arg "GOPROXY=${goproxy}"
build "${prefix}/workspace-java:${tag}" images/workspace-java/Dockerfile images/workspace-java \
  --build-arg "WORKSPACE_BASE_IMAGE=${workspace_base_image}" --build-arg "GRADLE_VERSION=${ELYRO_GRADLE_VERSION}" \
  --build-arg "GRADLE_SHA256=${ELYRO_GRADLE_SHA256}"
build "${prefix}/workspace-node:${tag}" images/workspace-node/Dockerfile images/workspace-node \
  --build-arg "WORKSPACE_BASE_IMAGE=${workspace_base_image}" --build-arg "NODE_VERSION=${ELYRO_NODE_VERSION}" \
  --build-arg "NODE_SHA256_AMD64=${ELYRO_NODE_SHA256_AMD64}" --build-arg "NODE_SHA256_ARM64=${ELYRO_NODE_SHA256_ARM64}"
