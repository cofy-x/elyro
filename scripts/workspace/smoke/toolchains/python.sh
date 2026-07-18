#!/usr/bin/env bash
# Python workspace toolchain smoke checks.

workspace_smoke_toolchain_python() {
  local BIN="$1"
  local ROOT_DIR="$2"

  local TMP_PROJECT TMP_HOME TMP_SSH_CONFIG TMP_STATE_HOME HOST_ALIAS HTTP_PORT EXPECTED_PLATFORM
  local REMOTE_DIR
  TMP_PROJECT="$(mktemp -d /tmp/elyro-workspace-smoke.XXXXXX)"
  TMP_HOME="$(mktemp -d /tmp/elyro-workspace-home.XXXXXX)"
  TMP_SSH_CONFIG="${TMP_HOME}/.ssh/config"
  TMP_STATE_HOME="$(mktemp -d /tmp/elyro-workspace-state.XXXXXX)"
  HTTP_PORT="18000"
  EXPECTED_PLATFORM="${ELYRO_WORKSPACE_SMOKE_PLATFORM:-linux/$(go env GOARCH)}"
  REMOTE_DIR="/home/elyro/$(basename "${TMP_PROJECT}")"

  _workspace_smoke_python_cleanup() {
    HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}" >/dev/null 2>&1 || true
    rm -rf "${TMP_PROJECT}" "${TMP_HOME}" "${TMP_STATE_HOME}"
  }
  trap _workspace_smoke_python_cleanup EXIT

  cat >"${TMP_PROJECT}/README.md" <<'EOF_README'
# smoke
EOF_README
  cat >"${TMP_PROJECT}/pyproject.toml" <<'EOF_PYPROJECT'
[project]
name = "elyro-smoke"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = []
EOF_PYPROJECT

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" up \
    --toolchain python \
    --platform "${EXPECTED_PLATFORM}" \
    --project-dir "${TMP_PROJECT}" \
    --publish "${HTTP_PORT}:8000"

  local STATUS_OUTPUT
  STATUS_OUTPUT="$(HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" status --project-dir "${TMP_PROJECT}" --json)"
  jq -e --arg published "${HTTP_PORT}:8000" \
    '.workspace.toolchain == "python" and (.workspace.published_ports | index($published)) != null' <<<"${STATUS_OUTPUT}" >/dev/null
  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" list --project-dir "${TMP_PROJECT}" --json | \
    jq -e '.schema_version == 1 and (.workspaces | length) > 0 and .workspaces[0].workspace.name != ""' >/dev/null

  local CONTAINER_ID IMAGE_ID
  CONTAINER_ID="$(docker ps -q --filter "label=elyro.workspace.project_dir=${TMP_PROJECT}")"
  test -n "${CONTAINER_ID}"

  IMAGE_ID="$(docker inspect --format '{{.Image}}' "${CONTAINER_ID}")"
  grep -q "^${EXPECTED_PLATFORM}$" <<<"$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "${IMAGE_ID}")"
  grep -q "^${REMOTE_DIR}\$" <<<"$(docker inspect --format '{{(index .Mounts 0).Destination}}' "${CONTAINER_ID}")"
  docker exec "${CONTAINER_ID}" bash -lc "cd '${REMOTE_DIR}' && uv --version >/dev/null && python --version >/dev/null && uv venv .venv && uv sync --offline && uv run python --version >/dev/null && python -m venv .venv-standard"
  docker exec "${CONTAINER_ID}" bash -lc "cd '${REMOTE_DIR}' && python -m http.server 8000 >/tmp/workspace-http.log 2>&1 &"
  workspace_smoke_wait_http "http://127.0.0.1:${HTTP_PORT}" 15
  curl -fsS "http://127.0.0.1:${HTTP_PORT}" >/dev/null

  test ! -e "${TMP_PROJECT}/elyro.yaml"
  test ! -e "${TMP_PROJECT}/.vscode"
  HOST_ALIAS="$(awk '/^Host elyro-/ {print $2; exit}' "${TMP_SSH_CONFIG}")"
  test -n "${HOST_ALIAS}"
  grep -Fq "IdentityFile \"${TMP_HOME}/.ssh/elyro_workspace_ed25519\"" "${TMP_SSH_CONFIG}"

  docker exec "${CONTAINER_ID}" bash -lc 'grep -Fq "PasswordAuthentication no" /etc/ssh/sshd_config.d/99-elyro-workspace.conf'
  ssh -F "${TMP_SSH_CONFIG}" -o BatchMode=yes -o ConnectTimeout=10 "${HOST_ALIAS}" 'python --version >/dev/null'

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}"

  trap - EXIT
  _workspace_smoke_python_cleanup
}
