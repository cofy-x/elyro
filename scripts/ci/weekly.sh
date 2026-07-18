#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MAKE_BIN="${MAKE:-make}"
cd "${ROOT_DIR}"
"${MAKE_BIN}" ci-nightly
printf '[ci-weekly] completed\n'
