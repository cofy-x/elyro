#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  requested_proxy=""
else
  requested_proxy="$1"
fi

if [ "${requested_proxy}" != "auto" ]; then
  printf '%s\n' "${requested_proxy}"
  exit 0
fi

if command -v nc >/dev/null 2>&1 && nc -z 127.0.0.1 7890 >/dev/null 2>&1; then
  printf '%s\n' "http://host.docker.internal:7890"
  exit 0
fi

if command -v python3 >/dev/null 2>&1 && \
  python3 -c 'import socket,sys; s=socket.socket(); s.settimeout(0.5); rc=s.connect_ex(("127.0.0.1",7890)); s.close(); sys.exit(0 if rc == 0 else 1)' >/dev/null 2>&1; then
  printf '%s\n' "http://host.docker.internal:7890"
  exit 0
fi

printf '\n'
