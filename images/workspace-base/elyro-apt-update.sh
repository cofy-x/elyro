#!/bin/sh
set -eu

retries="${ELYRO_APT_RETRIES:-3}"
timeout="${ELYRO_APT_TIMEOUT:-30}"
set -- apt-get -o "Acquire::Retries=${retries}" -o "Acquire::http::Timeout=${timeout}" -o "Acquire::https::Timeout=${timeout}" -o APT::Update::Error-Mode=any
if [ -n "${ELYRO_PROXY_URL:-}" ]; then
  set -- "$@" -o "Acquire::http::Proxy=${ELYRO_PROXY_URL}" -o "Acquire::https::Proxy=${ELYRO_PROXY_URL}"
fi
exec "$@" update
