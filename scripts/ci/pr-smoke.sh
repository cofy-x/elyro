#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAKE_BIN="${MAKE:-make}"
cd "${ROOT_DIR}"
"${MAKE_BIN}" test
"${MAKE_BIN}" workspace-smoke
printf '[ci-pr-smoke] completed\n'
