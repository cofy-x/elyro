#!/usr/bin/env bash
set -euo pipefail

platform="${ELYRO_BUILD_PLATFORM:?ELYRO_BUILD_PLATFORM is required}"
prefix="${ELYRO_BUILD_IMAGE_PREFIX:?ELYRO_BUILD_IMAGE_PREFIX is required}"
candidate_tag="${ELYRO_CANDIDATE_TAG:?ELYRO_CANDIDATE_TAG is required}"
source_version="${ELYRO_BUILD_VERSION:-dev}"

case "${platform}" in
  linux/amd64) arch=amd64 ;;
  linux/arm64) arch=arm64 ;;
  *) printf 'unsupported candidate platform: %s\n' "${platform}" >&2; exit 2 ;;
esac

[[ "${candidate_tag}" =~ ^candidate-[0-9a-f]{40}-[0-9]+$ ]] || {
  printf 'invalid immutable candidate tag: %s\n' "${candidate_tag}" >&2
  exit 2
}

for image in workspace-base workspace-python workspace-go workspace-node workspace-java; do
  source_ref="${prefix}/${image}:${source_version}-${arch}"
  candidate_ref="${prefix}/${image}:${candidate_tag}-${arch}"
  docker image inspect "${source_ref}" >/dev/null
  if docker buildx imagetools inspect "${candidate_ref}" >/dev/null 2>&1; then
    printf 'candidate image already exists and will not be overwritten: %s\n' "${candidate_ref}" >&2
    exit 1
  fi
  docker tag "${source_ref}" "${candidate_ref}"
  docker push "${candidate_ref}"
done
