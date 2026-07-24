#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/elyro-candidate-test.XXXXXX")"
trap 'rm -rf "${tmp_dir}"' EXIT
mock_bin="${tmp_dir}/bin"
log_file="${tmp_dir}/docker.log"
mkdir -p "${mock_bin}"

cat >"${mock_bin}/docker" <<'EOF_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${MOCK_DOCKER_LOG}"
case " $* " in
  ' image inspect '*) exit 0 ;;
  ' buildx imagetools inspect '*) [[ "${MOCK_CANDIDATE_EXISTS:-0}" == 1 ]] ;;
  ' tag '*|' push '*|' buildx imagetools create '*) exit 0 ;;
  *) printf 'unexpected docker command: %s\n' "$*" >&2; exit 2 ;;
esac
EOF_DOCKER
chmod 0755 "${mock_bin}/docker"

candidate_tag=candidate-0123456789abcdef0123456789abcdef01234567-4242
common_env=(
  PATH="${mock_bin}:${PATH}"
  MOCK_DOCKER_LOG="${log_file}"
  ELYRO_BUILD_IMAGE_PREFIX=ghcr.io/cofy-x/elyro
  ELYRO_CANDIDATE_TAG="${candidate_tag}"
)

env "${common_env[@]}" ELYRO_BUILD_PLATFORM=linux/arm64 ELYRO_BUILD_VERSION=dev \
  "${ROOT_DIR}/scripts/ci/push-candidate-images.sh"
[[ "$(grep -c '^push ' "${log_file}")" -eq 4 ]]

: >"${log_file}"
env "${common_env[@]}" "${ROOT_DIR}/scripts/ci/merge-candidate-images.sh"
[[ "$(grep -c '^buildx imagetools create ' "${log_file}")" -eq 4 ]]

if env "${common_env[@]}" MOCK_CANDIDATE_EXISTS=1 ELYRO_BUILD_PLATFORM=linux/amd64 ELYRO_BUILD_VERSION=dev \
  "${ROOT_DIR}/scripts/ci/push-candidate-images.sh" >/dev/null 2>&1; then
  printf '%s\n' 'existing immutable candidate unexpectedly passed' >&2
  exit 1
fi

if env "${common_env[@]}" ELYRO_CANDIDATE_TAG=candidate-main-4242 \
  "${ROOT_DIR}/scripts/ci/merge-candidate-images.sh" >/dev/null 2>&1; then
  printf '%s\n' 'invalid candidate tag unexpectedly passed' >&2
  exit 1
fi

printf '%s\n' 'candidate image script tests passed'
