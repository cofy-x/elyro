#!/usr/bin/env bash
set -euo pipefail

prefix="${ELYRO_BUILD_IMAGE_PREFIX:?ELYRO_BUILD_IMAGE_PREFIX is required}"
candidate_tag="${ELYRO_CANDIDATE_TAG:?ELYRO_CANDIDATE_TAG is required}"

[[ "${candidate_tag}" =~ ^candidate-[0-9a-f]{40}-[0-9]+$ ]] || {
  printf 'invalid immutable candidate tag: %s\n' "${candidate_tag}" >&2
  exit 2
}

for image in workspace-base workspace-python workspace-go workspace-node workspace-java; do
  target="${prefix}/${image}:${candidate_tag}"
  if docker buildx imagetools inspect "${target}" >/dev/null 2>&1; then
    printf 'candidate index already exists and will not be overwritten: %s\n' "${target}" >&2
    exit 1
  fi
  docker buildx imagetools create \
    --tag "${target}" \
    "${target}-amd64" \
    "${target}-arm64"
done
