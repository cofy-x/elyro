#!/usr/bin/env bash
# From-scratch custom-image Workspace environment smoke checks.

workspace_smoke_environment_custom_image() {
  local BIN="$1"
  local ROOT_DIR="$2"

  local TMP_PROJECT TMP_HOME TMP_SSH_CONFIG TMP_STATE_HOME
  local HOST_ALIAS ENVIRONMENT_NAME CUSTOM_IMAGE PLATFORM CONFIG_TMP REMOTE_DIR PROXY_URL
  TMP_PROJECT="$(mktemp -d /tmp/elyro-workspace-smoke-environment.XXXXXX)"
  TMP_HOME="$(mktemp -d /tmp/elyro-workspace-home-environment.XXXXXX)"
  TMP_SSH_CONFIG="${TMP_HOME}/.ssh/config"
  TMP_STATE_HOME="$(mktemp -d /tmp/elyro-workspace-state-environment.XXXXXX)"
  ENVIRONMENT_NAME="custom"
  CUSTOM_IMAGE="elyro/workspace-smoke-from-scratch:$$"
  PLATFORM="${ELYRO_WORKSPACE_SMOKE_PLATFORM:-linux/$(go env GOARCH)}"
  CONFIG_TMP="${TMP_PROJECT}/elyro.yaml.tmp"
  REMOTE_DIR="/home/elyro/$(basename "${TMP_PROJECT}")"
  PROXY_URL="$(bash "${ROOT_DIR}/scripts/lib/resolve-elyro-proxy.sh" "${ELYRO_PROXY_URL-}")"
  case "${PLATFORM}" in
    linux/amd64|linux/arm64) ;;
    *) printf '[workspace-smoke] unsupported platform: %s\n' "${PLATFORM}" >&2; return 1 ;;
  esac

  _workspace_smoke_environment_cleanup() {
    HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down \
      --project-dir "${TMP_PROJECT}" >/dev/null 2>&1 || true
    docker image rm -f "${CUSTOM_IMAGE}" >/dev/null 2>&1 || true
    rm -rf "${TMP_PROJECT}" "${TMP_HOME}" "${TMP_STATE_HOME}"
  }
  trap _workspace_smoke_environment_cleanup EXIT

  cp -R "${ROOT_DIR}/examples/workspace/custom-image-from-scratch/." "${TMP_PROJECT}/"
  chmod 0777 "${TMP_PROJECT}"
  printf '%s\n' 'host-to-container' >"${TMP_PROJECT}/workspace.txt"

  if grep -Fq 'ghcr.io/cofy-x/elyro' "${TMP_PROJECT}/Dockerfile"; then
    printf '[workspace-smoke] from-scratch Dockerfile references an official Elyro image\n' >&2
    return 1
  fi
  awk -v image="${CUSTOM_IMAGE}" '
    $1 == "image:" { print "    image: " image; next }
    { print }
  ' "${TMP_PROJECT}/elyro.yaml" >"${CONFIG_TMP}"
  mv "${CONFIG_TMP}" "${TMP_PROJECT}/elyro.yaml"

  docker build \
    --platform "${PLATFORM}" \
    --build-arg "MIRROR_SOURCE=${ELYRO_MIRROR_SOURCE:-official}" \
    --build-arg "ELYRO_PROXY_URL=${PROXY_URL}" \
    -t "${CUSTOM_IMAGE}" "${TMP_PROJECT}"

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" up \
    --environment "${ENVIRONMENT_NAME}" \
    --project-dir "${TMP_PROJECT}"

  local STATUS_OUTPUT CONTAINER_ID
  STATUS_OUTPUT="$(HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" status --project-dir "${TMP_PROJECT}" --json)"
  jq -e \
    --arg environment "${ENVIRONMENT_NAME}" \
    --arg platform "${PLATFORM}" \
    --arg image "${CUSTOM_IMAGE}" \
    '.workspace.environment == $environment and (.workspace.toolchain // "") == "" and .workspace.platform == $platform and .workspace.image == $image' \
    <<<"${STATUS_OUTPUT}" >/dev/null

  CONTAINER_ID="$(docker ps -q --filter "label=elyro.workspace.project_dir=${TMP_PROJECT}")"
  test -n "${CONTAINER_ID}"
  test "$(docker inspect --format '{{.Config.User}}' "${CONTAINER_ID}")" = "elyro"
  test "$(docker exec "${CONTAINER_ID}" id -un)" = "elyro"
  test "$(docker exec --user 0 "${CONTAINER_ID}" id -u)" = "0"

  HOST_ALIAS="$(awk '/^Host elyro-/ {print $2; exit}' "${TMP_SSH_CONFIG}")"
  test -n "${HOST_ALIAS}"
  grep -Fq "IdentityFile \"${TMP_HOME}/.ssh/elyro_workspace_ed25519\"" "${TMP_SSH_CONFIG}"
  ssh -F "${TMP_SSH_CONFIG}" -o BatchMode=yes -o ConnectTimeout=10 "${HOST_ALIAS}" \
    'test "$(id -un)" = elyro'
  ssh -F "${TMP_SSH_CONFIG}" -o BatchMode=yes -o ConnectTimeout=10 "${HOST_ALIAS}" \
    "cd '${REMOTE_DIR}' && grep -Fq host-to-container workspace.txt"

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" exec \
    --project-dir "${TMP_PROJECT}" -- \
    sh -c 'printf "%s\n" container-to-host >container-created.txt'
  grep -Fq container-to-host "${TMP_PROJECT}/container-created.txt"

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}"

  trap - EXIT
  _workspace_smoke_environment_cleanup
}
