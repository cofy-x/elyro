#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAKE_BIN="${MAKE:-make}"
cd "${ROOT_DIR}"
"${MAKE_BIN}" ci-pr-smoke
"${MAKE_BIN}" workspace-e2e
printf '[ci-nightly] completed\n'
