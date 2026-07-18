#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-pr}"

log() {
  printf '[ci-suite] %s\n' "$*"
}

case "${MODE}" in
  pr)
    exec "${SCRIPT_DIR}/pr-smoke.sh"
    ;;
  nightly)
    exec "${SCRIPT_DIR}/nightly.sh"
    ;;
  weekly)
    exec "${SCRIPT_DIR}/weekly.sh"
    ;;
  all)
    log "running pr suite"
    "${SCRIPT_DIR}/pr-smoke.sh"
    log "running nightly suite"
    "${SCRIPT_DIR}/nightly.sh"
    log "running weekly suite"
    "${SCRIPT_DIR}/weekly.sh"
    ;;
  *)
    echo "usage: ${0} [pr|nightly|weekly|all]" >&2
    exit 1
    ;;
esac
