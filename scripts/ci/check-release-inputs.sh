#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
VERSIONS_FILE="${ROOT_DIR}/release/versions.env"

fail() {
  printf 'release input check failed: %s\n' "$*" >&2
  exit 1
}

require_line() {
  local file="$1"
  local expected="$2"
  grep -Fqx "${expected}" "${file}" || fail "${file} is not synchronized: ${expected}"
}

[[ -f "${VERSIONS_FILE}" ]] || fail "missing release/versions.env"

while IFS='=' read -r key value; do
  [[ -z "${key}" || "${key}" == \#* ]] && continue
  [[ "${key}" =~ ^ELYRO_[A-Z0-9_]+$ ]] || fail "invalid key ${key}"
  if [[ -z "${value}" && "${key}" != "ELYRO_COMPARE_VERSION" ]]; then
    fail "${key} is empty"
  fi
  lower_value="$(printf '%s' "${value}" | tr '[:upper:]' '[:lower:]')"
  if [[ "${lower_value}" =~ (^|[-_.:/])(latest|stable|main|master|nightly|edge)($|[-_.:/]) ]]; then
    fail "${key} uses floating value ${value}"
  fi
done <"${VERSIONS_FILE}"

# shellcheck disable=SC1090
source "${VERSIONS_FILE}"
[[ "${ELYRO_RELEASE_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]] || \
  fail "ELYRO_RELEASE_VERSION must be a v-prefixed semantic version"
if [[ -n "${ELYRO_COMPARE_VERSION}" ]]; then
  [[ "${ELYRO_COMPARE_VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]] || \
    fail "ELYRO_COMPARE_VERSION must be empty or a v-prefixed semantic version"
  [[ "${ELYRO_COMPARE_VERSION}" != "${ELYRO_RELEASE_VERSION}" ]] || \
    fail "ELYRO_COMPARE_VERSION must differ from ELYRO_RELEASE_VERSION"
fi

if [[ -n "${1:-}" && "${1}" != "${ELYRO_RELEASE_VERSION}" ]]; then
  fail "tag ${1} does not match reviewed version ${ELYRO_RELEASE_VERSION}"
fi

core_files=(
  images/workspace-base/Dockerfile
  images/workspace-python/Dockerfile
  images/workspace-go/Dockerfile
  images/workspace-java/Dockerfile
  images/workspace-node/Dockerfile
)

budget_file=release/image-budgets.json

cd "${ROOT_DIR}"
for file in "${core_files[@]}"; do
  [[ -f "${file}" ]] || fail "missing ${file}"
  if grep -Eq '^(FROM|ARG [A-Z0-9_]+)=?.*:(latest|stable)([^A-Za-z0-9]|$)' "${file}"; then
    fail "${file} contains a floating image or build default"
  fi
done

[[ -f "${budget_file}" ]] || fail "missing ${budget_file}"
for image in workspace-base workspace-python workspace-go workspace-node workspace-java; do
  grep -Eq '"'"${image}"'"[[:space:]]*:[[:space:]]*[1-9][0-9]*(\.[0-9]+)?' "${budget_file}" || \
    fail "${budget_file} has no positive MiB budget for ${image}"
done

first_from="$(awk '/^FROM / {print; exit}' images/workspace-base/Dockerfile)"
[[ "${first_from}" == *'@sha256:'* ]] || fail "workspace-base Ubuntu image is not digest-pinned"
[[ "${first_from#FROM }" == "${ELYRO_UBUNTU_IMAGE}" ]] || fail "Ubuntu digest differs from release/versions.env"

require_line images/workspace-go/Dockerfile "ARG GO_VERSION=${ELYRO_GO_VERSION}"
require_line images/workspace-go/Dockerfile "ARG GO_SHA256_AMD64=${ELYRO_GO_SHA256_AMD64}"
require_line images/workspace-go/Dockerfile "ARG GO_SHA256_ARM64=${ELYRO_GO_SHA256_ARM64}"
require_line images/workspace-python/Dockerfile "ARG UV_VERSION=${ELYRO_UV_VERSION}"
require_line images/workspace-python/Dockerfile "ARG UV_SHA256_AMD64=${ELYRO_UV_SHA256_AMD64}"
require_line images/workspace-python/Dockerfile "ARG UV_SHA256_ARM64=${ELYRO_UV_SHA256_ARM64}"
require_line images/workspace-java/Dockerfile "ARG GRADLE_VERSION=${ELYRO_GRADLE_VERSION}"
require_line images/workspace-java/Dockerfile "ARG GRADLE_SHA256=${ELYRO_GRADLE_SHA256}"
require_line images/workspace-node/Dockerfile "ARG NODE_VERSION=${ELYRO_NODE_VERSION}"
require_line images/workspace-node/Dockerfile "ARG NODE_SHA256_AMD64=${ELYRO_NODE_SHA256_AMD64}"
require_line images/workspace-node/Dockerfile "ARG NODE_SHA256_ARM64=${ELYRO_NODE_SHA256_ARM64}"
for file in images/workspace-python/Dockerfile images/workspace-go/Dockerfile images/workspace-java/Dockerfile images/workspace-node/Dockerfile; do
  require_line "${file}" "ARG WORKSPACE_BASE_IMAGE=ghcr.io/cofy-x/elyro/workspace-base:${ELYRO_RELEASE_VERSION}"
done
if grep -Eh '^ARG (GO|UV|NODE)_SHA256_(AMD64|ARM64)=' \
  images/workspace-go/Dockerfile images/workspace-python/Dockerfile images/workspace-node/Dockerfile | \
  grep -Ev '=[0-9a-f]{64}$'; then
	fail "Go, uv, or Node.js checksum is not a lowercase SHA-256"
fi
[[ "${ELYRO_GRADLE_SHA256}" =~ ^[0-9a-f]{64}$ ]] || fail "Gradle checksum is not a lowercase SHA-256"

if grep -REn 'raw\.githubusercontent\.com/.+/(main|master)/|git clone .+ --branch (main|master)' \
  "${core_files[@]}"; then
  fail "core image inputs contain an unpinned source checkout"
fi

printf 'release inputs verified for %s\n' "${ELYRO_RELEASE_VERSION}"
