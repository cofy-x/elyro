#!/usr/bin/env bash
set -euo pipefail

# Unified smoke entry. Per-toolchain checks live under smoke/toolchains/.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BIN="${ELYRO_WORKSPACE_SMOKE_BIN:-${ROOT_DIR}/bin/elyro}"

command -v jq >/dev/null 2>&1 || {
  printf '%s\n' '[workspace-smoke] missing required command: jq' >&2
  exit 1
}

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=toolchains/python.sh
source "${SCRIPT_DIR}/toolchains/python.sh"
# shellcheck source=toolchains/go.sh
source "${SCRIPT_DIR}/toolchains/go.sh"
# shellcheck source=toolchains/node.sh
source "${SCRIPT_DIR}/toolchains/node.sh"
# shellcheck source=environments/custom_image.sh
source "${SCRIPT_DIR}/environments/custom_image.sh"

workspace_smoke_build_binary "${ROOT_DIR}" "${BIN}"
workspace_smoke_build_images "${ROOT_DIR}"

workspace_smoke_toolchain_python "${BIN}" "${ROOT_DIR}"
workspace_smoke_toolchain_go "${BIN}" "${ROOT_DIR}"
workspace_smoke_toolchain_node "${BIN}" "${ROOT_DIR}"
workspace_smoke_environment_custom_image "${BIN}" "${ROOT_DIR}"

echo "Elyro Workspace smoke checks passed."
