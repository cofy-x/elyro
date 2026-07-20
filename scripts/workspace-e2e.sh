#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE_EXAMPLES_DIR="${ELYRO_WORKSPACE_EXAMPLES_DIR:-${ROOT_DIR}/examples/workspace}"
BIN="${ELYRO_WORKSPACE_E2E_BIN:-${ROOT_DIR}/bin/elyro}"
ELYRO_VERSION="${ELYRO_WORKSPACE_E2E_VERSION:-dev}"
TMP_ROOT="${ELYRO_WORKSPACE_E2E_TMP_ROOT:-$(mktemp -d /tmp/elyro-workspace-e2e.XXXXXX)}"
TMP_ROOT="$(cd "${TMP_ROOT}" && pwd -P)"
RUN_ID="${ELYRO_WORKSPACE_E2E_RUN_ID:-$(date +%s)-$$}"
HOST_HOME="${HOME}"
export HOME="${TMP_ROOT}/home"
export DOCKER_CONFIG="${DOCKER_CONFIG:-${HOST_HOME}/.docker}"
SSH_CONFIG="${HOME}/.ssh/config"
CASES="${ELYRO_WORKSPACE_E2E_CASES:-python go node java environment image runtime-environment}"
CUSTOM_IMAGE="elyro/workspace-e2e-custom:${RUN_ID}"
PROJECT_IMAGE="elyro/workspace-e2e-project:${RUN_ID}"
WORKSPACE_EXEC_PID=""
WORKSPACE_EXEC_LOG=""
export XDG_STATE_HOME="${TMP_ROOT}/state"

log() {
  printf '[workspace-e2e] %s\n' "$*"
}

require_cmd() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    printf '[workspace-e2e] missing required command: %s\n' "${name}" >&2
    exit 1
  fi
}

find_free_port() {
  python3 - <<'PY'
import socket

with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
}

wait_http() {
  local url="$1"
  local attempts="${2:-30}"
  local _

  for _ in $(seq 1 "${attempts}"); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    if [ -n "${WORKSPACE_EXEC_PID}" ] && ! kill -0 "${WORKSPACE_EXEC_PID}" 2>/dev/null; then
      printf '[workspace-e2e] workspace service exited before %s became ready\n' "${url}" >&2
      tail -n 50 "${WORKSPACE_EXEC_LOG}" >&2 || true
      return 1
    fi
    sleep 1
  done
  printf '[workspace-e2e] timed out waiting for %s\n' "${url}" >&2
  if [ -n "${WORKSPACE_EXEC_LOG}" ]; then
    tail -n 50 "${WORKSPACE_EXEC_LOG}" >&2 || true
  fi
  return 1
}

copy_example() {
  local name="$1"
  local src="${WORKSPACE_EXAMPLES_DIR}/${name}"
  local dest="${TMP_ROOT}/${name}"

  rm -rf "${dest}"
  mkdir -p "${dest}"
  cp -R "${src}/." "${dest}/"
  prepare_project_dir "${dest}"
  printf '%s\n' "${dest}"
}

prepare_project_dir() {
  local project_dir="$1"

  # Official Workspace images use UID/GID 1000. GitHub-hosted Linux runners
  # create fixtures as a different UID, so make only these disposable projects
  # writable by the same user that exercises the bind mount inside the container.
  chmod -R a+rwX "${project_dir}"
}

container_for_project() {
  local project_dir="$1"
  docker ps -aq --filter "label=elyro.workspace.project_dir=${project_dir}"
}

workspace_down() {
  local project_dir="$1"

  "${BIN}" down --project-dir "${project_dir}" >/dev/null
  assert_workspace_removed "${project_dir}"
}

assert_workspace_removed() {
  local project_dir="$1"

  if [ -n "$(container_for_project "${project_dir}")" ]; then
    printf '[workspace-e2e] workspace container remains after down: %s\n' "${project_dir}" >&2
    return 1
  fi
  if "${BIN}" list --json | jq -e --arg project "${project_dir}" \
    '.workspaces[] | select(.workspace.project_dir == $project)' >/dev/null; then
    printf '[workspace-e2e] workspace registry entry remains after down: %s\n' "${project_dir}" >&2
    return 1
  fi
}

latest_host_alias() {
  awk '/^Host elyro-/ { alias = $2 } END { print alias }' "${SSH_CONFIG}"
}

run_ssh() {
  ssh \
    -F "${SSH_CONFIG}" \
    -o BatchMode=yes \
    -o ConnectTimeout=10 \
    "$@"
}

start_workspace_service() {
  local project_dir="$1"
  local log_file="$2"
  shift 2

  "${BIN}" exec --project-dir "${project_dir}" -- "$@" >"${log_file}" 2>&1 &
  WORKSPACE_EXEC_PID=$!
  WORKSPACE_EXEC_LOG="${log_file}"
}

stop_workspace_service() {
  if [ -z "${WORKSPACE_EXEC_PID}" ]; then
    return
  fi
  if kill -0 "${WORKSPACE_EXEC_PID}" 2>/dev/null; then
    kill -INT "${WORKSPACE_EXEC_PID}" 2>/dev/null || true
  fi
  wait "${WORKSPACE_EXEC_PID}" 2>/dev/null || true
  WORKSPACE_EXEC_PID=""
  WORKSPACE_EXEC_LOG=""
}

file_fingerprint() {
  local path="$1"

  if [ ! -e "${path}" ]; then
    printf 'absent\n'
    return
  fi
  cksum "${path}"
}

cleanup() {
  local project_dir container_id

  stop_workspace_service

  for project_dir in \
    "${TMP_ROOT}/python-http-service" \
    "${TMP_ROOT}/go-http-service" \
    "${TMP_ROOT}/go-custom-image-environment" \
    "${TMP_ROOT}/go-project-image" \
    "${TMP_ROOT}/go-runtime-environment" \
    "${TMP_ROOT}/node-test" \
    "${TMP_ROOT}/java-test"; do
    if [ -d "${project_dir}" ]; then
      while IFS= read -r container_id; do
        if [ -n "${container_id}" ]; then
          docker rm -f "${container_id}" >/dev/null 2>&1 || true
        fi
      done < <(docker ps -aq --filter "label=elyro.workspace.project_dir=${project_dir}")
    fi
  done
  docker image rm -f "${CUSTOM_IMAGE}" >/dev/null 2>&1 || true
  docker image rm -f "${PROJECT_IMAGE}" >/dev/null 2>&1 || true
  rm -rf "${TMP_ROOT}"
}
trap cleanup EXIT

build_workspace() {
  log "building elyro binary"
  mkdir -p "$(dirname "${BIN}")"
  (
    cd "${ROOT_DIR}"
    # Isolate Elyro state below the temporary HOME without relocating Go's
    # read-only module cache into a directory that this test must delete.
    HOME="${HOST_HOME}" go build \
      -ldflags "-X github.com/cofy-x/elyro/internal/version.Version=${ELYRO_VERSION}" \
      -o "${BIN}" ./cmd/elyro
  )
}

assert_status_field() {
  local project_dir="$1"
  local field="$2"
  local expected="$3"

  "${BIN}" status --project-dir "${project_dir}" --json | \
    jq -e --arg field "${field}" --arg expected "${expected}" \
      '.workspace[$field] | if type == "array" then index($expected) != null else . == $expected end' >/dev/null
}

run_python_case() {
  local project_dir host_alias port container_id remote_dir

  project_dir="$(copy_example python-http-service)"
  port="$(find_free_port)"
  remote_dir="/home/elyro/python-http-service"

  log "python example: workspace up"
  (cd "${project_dir}" && "${BIN}" init --yes)
  "${BIN}" up \
    --project-dir "${project_dir}" \
    --publish "${port}:8000"

  host_alias="$(latest_host_alias)"

  assert_status_field "${project_dir}" toolchain python
  assert_status_field "${project_dir}" published_ports "${port}:8000"
  test -f "${project_dir}/.vscode/extensions.json"
  test -f "${project_dir}/.vscode/settings.json"
  grep -Fq "Host ${host_alias}" "${SSH_CONFIG}"

  container_id="$(container_for_project "${project_dir}")"
  test -n "${container_id}"
  docker exec "${container_id}" bash -lc 'python --version >/dev/null && uv --version >/dev/null'
  start_workspace_service "${project_dir}" "${TMP_ROOT}/python-service.log" \
    env APP_GREETING="hello from python e2e" python app.py
  wait_http "http://127.0.0.1:${port}/healthz" 30
  curl -fsS "http://127.0.0.1:${port}/" | grep -Fq "hello from python e2e"
  run_ssh "${host_alias}" "cd '${remote_dir}' && python --version >/dev/null"
  stop_workspace_service

  log "python example: workspace down"
  workspace_down "${project_dir}"
}

run_go_case() {
  local project_dir host_alias port container_id recreated_container_id exec_status nested_dir root_id nested_id
  local ssh_before known_hosts_before registry_before

  project_dir="$(copy_example go-http-service)"
  port="$(find_free_port)"

  log "go example: create plan is side-effect free"
  ssh_before="$(file_fingerprint "${SSH_CONFIG}")"
  known_hosts_before="$(file_fingerprint "${HOME}/.ssh/elyro_known_hosts")"
  registry_before="$(file_fingerprint "${XDG_STATE_HOME}/elyro/workspaces.json")"
  "${BIN}" up --project-dir "${project_dir}" --publish "${port}:8000" --dry-run --json | \
    jq -e --arg root "${project_dir}" '
      .kind == "workspace_plan" and .operation == "up" and .action == "create" and
      .reasons == ["workspace_absent"] and .project.root == $root
    ' >/dev/null
  test -z "$(container_for_project "${project_dir}")"
  test "$(file_fingerprint "${SSH_CONFIG}")" = "${ssh_before}"
  test "$(file_fingerprint "${HOME}/.ssh/elyro_known_hosts")" = "${known_hosts_before}"
  test "$(file_fingerprint "${XDG_STATE_HOME}/elyro/workspaces.json")" = "${registry_before}"

  log "go example: workspace up"
  "${BIN}" up \
    --project-dir "${project_dir}" \
    --publish "${port}:8000"

  host_alias="$(latest_host_alias)"

  assert_status_field "${project_dir}" toolchain go
  assert_status_field "${project_dir}" published_ports "${port}:8000"
  test ! -e "${project_dir}/elyro.yaml"
  test ! -e "${project_dir}/.vscode"
  grep -Fq "Host ${host_alias}" "${SSH_CONFIG}"

  container_id="$(container_for_project "${project_dir}")"
  test -n "${container_id}"
  docker exec "${container_id}" bash -lc 'go version >/dev/null && ! command -v golangci-lint'

  log "go example: reuse and start plans"
  "${BIN}" up --project-dir "${project_dir}" --publish "${port}:8000" --dry-run --json | \
    jq -e '.action == "reuse" and .reasons == [] and .workspace.current_status == "running"' >/dev/null
  docker stop "${container_id}" >/dev/null
  "${BIN}" up --project-dir "${project_dir}" --publish "${port}:8000" --dry-run --json | \
    jq -e '.action == "start" and .reasons == ["workspace_stopped"] and .workspace.current_status == "exited"' >/dev/null
  "${BIN}" up --project-dir "${project_dir}" --publish "${port}:8000" --json | \
    jq -e '.action == "started" and .reasons == ["workspace_stopped"]' >/dev/null
  test "$(container_for_project "${project_dir}")" = "${container_id}"

  nested_dir="${project_dir}/internal/http"
  mkdir -p "${nested_dir}"
  root_id="$("${BIN}" status --project-dir "${project_dir}" --json | jq -r '.workspace.id')"
  nested_id="$(cd "${nested_dir}" && "${BIN}" status --json | jq -r '.workspace.id')"
  test "${nested_id}" = "${root_id}"
  (cd "${nested_dir}" && "${BIN}" doctor --json) | \
    jq -e --arg root "${project_dir}" '.schema_version == 2 and .healthy == true and .project.root == $root and .project.source == "registry"' >/dev/null
  (cd "${nested_dir}" && "${BIN}" exec -- /usr/local/go/bin/go version) >/dev/null

  log "go example: recreate from nested directory"
  (cd "${nested_dir}" && "${BIN}" up --recreate --publish "${port}:8000" --dry-run --json) | \
    jq -e '.action == "recreate" and .reasons == ["explicit_recreate"]' >/dev/null
  (cd "${nested_dir}" && "${BIN}" up --recreate --publish "${port}:8000" --json) | \
    jq -e '.action == "recreated" and .reasons == ["explicit_recreate"] and .workspace.status == "running"' >/dev/null
  recreated_container_id="$(container_for_project "${project_dir}")"
  test -n "${recreated_container_id}"
  test "${recreated_container_id}" != "${container_id}"

  start_workspace_service "${project_dir}" "${TMP_ROOT}/go-service.log" \
    env APP_GREETING="hello from go e2e" go run .
  wait_http "http://127.0.0.1:${port}/healthz" 40
  curl -fsS "http://127.0.0.1:${port}/" | grep -Fq "hello from go e2e"
  "${BIN}" exec \
    --project-dir "${project_dir}" -- \
    /usr/local/go/bin/go version >/dev/null
  if "${BIN}" exec \
    --project-dir "${project_dir}" -- \
    sh -c 'exit 23'; then
    printf '[workspace-e2e] expected workspace exec to preserve a failing remote status\n' >&2
    return 1
  else
    exec_status=$?
  fi
  if [ "${exec_status}" -ne 23 ]; then
    printf '[workspace-e2e] workspace exec status = %s, want 23\n' "${exec_status}" >&2
    return 1
  fi
  stop_workspace_service

  log "go example: removal, cleanup, and empty plans"
  (cd "${nested_dir}" && "${BIN}" down --dry-run --json) | jq -e '
    .action == "remove" and
    .removes == ["container_writable_layer", "managed_ssh", "known_hosts", "registry_record"] and
    .preserves == ["project_files", "mounted_host_data", "local_images"]
  ' >/dev/null
  docker rm -f "${recreated_container_id}" >/dev/null
  (cd "${nested_dir}" && "${BIN}" down --dry-run --json) | \
    jq -e '.action == "cleanup" and (.removes | index("container_writable_layer") | not)' >/dev/null
  (cd "${nested_dir}" && "${BIN}" down) >/dev/null
  assert_workspace_removed "${project_dir}"
  (cd "${nested_dir}" && "${BIN}" down --dry-run --json) | \
    jq -e '.action == "none" and .removes == []' >/dev/null
}

run_node_case() {
  local project_dir host_alias

  project_dir="${TMP_ROOT}/node-test"
  mkdir -p "${project_dir}"
  cat >"${project_dir}/package.json" <<'EOF'
{"scripts":{"test":"node --test"}}
EOF
  cat >"${project_dir}/ready.test.js" <<'EOF'
const test = require('node:test');
const assert = require('node:assert/strict');
test('node workspace is ready', () => assert.equal(process.platform, 'linux'));
EOF
  prepare_project_dir "${project_dir}"

  log "node project: workspace up and exec"
  "${BIN}" up --project-dir "${project_dir}"
  host_alias="$(latest_host_alias)"
  assert_status_field "${project_dir}" toolchain node
  "${BIN}" exec --project-dir "${project_dir}" -- npm test
  run_ssh "${host_alias}" 'node --version >/dev/null && npm --version >/dev/null'

  log "node project: workspace down"
  workspace_down "${project_dir}"
}

run_java_case() {
  local project_dir host_alias

  project_dir="${TMP_ROOT}/java-test"
  mkdir -p "${project_dir}"
  cat >"${project_dir}/Ready.java" <<'EOF'
public final class Ready {
  public static void main(String[] args) {
    if (!System.getProperty("os.name").toLowerCase().contains("linux")) {
      throw new IllegalStateException("expected Linux");
    }
    System.out.println("java workspace is ready");
  }
}
EOF
  prepare_project_dir "${project_dir}"

  log "java project: workspace up and exec"
  "${BIN}" up --toolchain java --project-dir "${project_dir}"
  host_alias="$(latest_host_alias)"
  assert_status_field "${project_dir}" toolchain java
  "${BIN}" exec --project-dir "${project_dir}" -- bash -lc 'javac Ready.java && java Ready'
  run_ssh "${host_alias}" 'java -version >/dev/null && mvn --version >/dev/null && gradle --version >/dev/null'

  log "java project: workspace down"
  workspace_down "${project_dir}"
}

run_environment_case() {
  local project_dir host_alias port container_id recreated_container_id config_tmp platform arch image_prefix base_image docker_context

  project_dir="$(copy_example go-custom-image-environment)"
  port="$(find_free_port)"
  config_tmp="${project_dir}/elyro.yaml.tmp"
  case "$(uname -m)" in
    x86_64|amd64) platform="linux/amd64"; arch="amd64" ;;
    arm64|aarch64) platform="linux/arm64"; arch="arm64" ;;
    *) printf '[workspace-e2e] unsupported architecture: %s\n' "$(uname -m)" >&2; return 1 ;;
  esac
  image_prefix="${ELYRO_IMAGE_PREFIX:-ghcr.io/cofy-x/elyro}"
  image_prefix="${image_prefix%/}"
  base_image="${image_prefix}/workspace-go:dev-${arch}"
  docker_context="$(docker context show)"

  awk -v image="${CUSTOM_IMAGE}" -v platform="${platform}" '
    $1 == "image:" { print "    image: " image; next }
    $1 == "platform:" { print "    platform: " platform; next }
    { print }
  ' "${project_dir}/elyro.yaml" >"${config_tmp}"
  mv "${config_tmp}" "${project_dir}/elyro.yaml"

  log "environment example: docker build ${CUSTOM_IMAGE}"
  # The daemon builder for the current Docker context can consume the locally
  # loaded Workspace image. CI also selects an isolated container builder,
  # which cannot see daemon images and would try to pull this development tag.
  docker buildx build --builder "${docker_context}" --load \
    --platform "${platform}" \
    --build-arg "WORKSPACE_GO_IMAGE=${base_image}" \
    -t "${CUSTOM_IMAGE}" "${project_dir}"

  log "environment example: workspace up"
  "${BIN}" up \
    --environment api \
    --project-dir "${project_dir}" \
    --publish "${port}:8000"

  host_alias="$(latest_host_alias)"

  assert_status_field "${project_dir}" environment api
  assert_status_field "${project_dir}" image "${CUSTOM_IMAGE}"
  assert_status_field "${project_dir}" published_ports "${port}:8000"
  grep -Fq "redhat.vscode-yaml" "${project_dir}/.vscode/extensions.json"
  grep -Fq "Host ${host_alias}" "${SSH_CONFIG}"

  container_id="$(container_for_project "${project_dir}")"
  test -n "${container_id}"

  log "environment example: recreate custom image with unchanged tag"
  "${BIN}" up \
    --environment api \
    --project-dir "${project_dir}" \
    --publish "${port}:8000" \
    --recreate \
    --json | jq -e '.action == "recreated"' >/dev/null
  recreated_container_id="$(container_for_project "${project_dir}")"
  test -n "${recreated_container_id}"
  test "${recreated_container_id}" != "${container_id}"
  docker exec "${recreated_container_id}" bash -lc 'elyro-example-tool | grep -Fq elyro-example-tool'
  start_workspace_service "${project_dir}" "${TMP_ROOT}/environment-service.log" \
    go run .
  wait_http "http://127.0.0.1:${port}/healthz" 40
  stop_workspace_service
  run_ssh "${host_alias}" 'elyro-example-tool | grep -Fq elyro-example-tool'

  log "environment example: workspace down"
  workspace_down "${project_dir}"
}

run_project_image_case() {
  local project_dir nested_dir dockerfile report running_container recreated_container

  project_dir="$(copy_example go-http-service)"
  mv "${project_dir}" "${TMP_ROOT}/go-project-image"
  project_dir="$(cd "${TMP_ROOT}/go-project-image" && pwd -P)"
  git -C "${project_dir}" init -q
  nested_dir="${project_dir}/internal/e2e/nested"
  mkdir -p "${nested_dir}"

  log "project image: initialize from nested directory"
  (cd "${nested_dir}" && "${BIN}" image init --toolchain go --image "${PROJECT_IMAGE}" --yes) | grep -Fq 'elyro image build'
  dockerfile="${project_dir}/.elyro/Dockerfile"
  test -f "${dockerfile}"
  grep -Fq "image: ${PROJECT_IMAGE}" "${project_dir}/elyro.yaml"

  report="${TMP_ROOT}/project-image-doctor.json"
  if (cd "${nested_dir}" && "${BIN}" doctor --json) >"${report}" 2>/dev/null; then
    printf '[workspace-e2e] expected missing project image to make doctor fail\n' >&2
    return 1
  fi
  jq -e '
    .schema_version == 2 and .healthy == false and
    .project.image_build.context == "." and
    .project.image_build.dockerfile == ".elyro/Dockerfile" and
    any(.checks[]; .name == "workspace_image" and .status == "fail")
  ' "${report}" >/dev/null

  sed -e 's/^# RUN /RUN /' -e 's/^#     /    /' "${dockerfile}" >"${dockerfile}.tmp"
  mv "${dockerfile}.tmp" "${dockerfile}"

  log "project image: cold and warm builds"
  (cd "${nested_dir}" && "${BIN}" image build --json) >"${TMP_ROOT}/project-image-cold.json" 2>"${TMP_ROOT}/project-image-cold.log"
  test -s "${TMP_ROOT}/project-image-cold.log"
  jq -e --arg image "${PROJECT_IMAGE}" '.kind == "image" and .action == "built" and .image.reference == $image' "${TMP_ROOT}/project-image-cold.json" >/dev/null
  NO_COLOR=1 "${BIN}" image build --project-dir "${project_dir}" >"${TMP_ROOT}/project-image-warm.out" 2>"${TMP_ROOT}/project-image-warm.log"
  grep -Fq 'Workspace image built' "${TMP_ROOT}/project-image-warm.out"
  if LC_ALL=C grep -q "$(printf '\033')" "${TMP_ROOT}/project-image-warm.out"; then
    printf '[workspace-e2e] plain project image receipt contains ANSI\n' >&2
    return 1
  fi

  log "project image: start and verify derived tool"
  (cd "${nested_dir}" && "${BIN}" up --json) | jq -e --arg image "${PROJECT_IMAGE}" '.action == "created" and .workspace.image == $image' >/dev/null
  test "$(cd "${nested_dir}" && "${BIN}" exec -- sqlite3 :memory: 'select 42;')" = 42
  (cd "${nested_dir}" && "${BIN}" open --print) | grep -Fq 'vscode-remote://ssh-remote+elyro-'
  running_container="$(container_for_project "${project_dir}")"
  test -n "${running_container}"

  log "project image: failed build preserves old image and running Workspace"
  cp "${dockerfile}" "${dockerfile}.good"
  printf '%s\n' 'RUN false' >>"${dockerfile}"
  if "${BIN}" image build --project-dir "${project_dir}" --json >"${TMP_ROOT}/project-image-fail.json" 2>"${TMP_ROOT}/project-image-fail.log"; then
    printf '[workspace-e2e] intentionally broken project image build passed\n' >&2
    return 1
  fi
  test ! -s "${TMP_ROOT}/project-image-fail.json"
  assert_status_field "${project_dir}" status running
  test "$("${BIN}" exec --project-dir "${project_dir}" -- sqlite3 :memory: 'select 42;')" = 42
  mv "${dockerfile}.good" "${dockerfile}"

  log "project image: same-tag rebuild requires explicit recreate"
  printf '%s\n' "RUN printf '%s\\n' '#!/bin/sh' 'echo elyro-image-e2e-v2' > /usr/local/bin/elyro-image-e2e-tool && chmod +x /usr/local/bin/elyro-image-e2e-tool" >>"${dockerfile}"
  "${BIN}" image build --project-dir "${project_dir}" --json >"${TMP_ROOT}/project-image-rebuild.json" 2>"${TMP_ROOT}/project-image-rebuild.log"
  if "${BIN}" exec --project-dir "${project_dir}" -- elyro-image-e2e-tool >/dev/null 2>&1; then
    printf '[workspace-e2e] running Workspace changed without recreate\n' >&2
    return 1
  fi
  "${BIN}" up --project-dir "${project_dir}" --recreate --json | jq -e '.action == "recreated"' >/dev/null
  recreated_container="$(container_for_project "${project_dir}")"
  test -n "${recreated_container}"
  test "${recreated_container}" != "${running_container}"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- elyro-image-e2e-tool)" = elyro-image-e2e-v2
  "${BIN}" exec --project-dir "${project_dir}" -- go test ./...

  log "project image: workspace down"
  workspace_down "${project_dir}"
}

run_runtime_environment_case() {
  local project_dir nested_dir host_alias first_container second_container report
  local sentinel_file="runtime-file-sentinel-${RUN_ID}"
  local sentinel_inline="runtime-inline-sentinel-${RUN_ID}"

  project_dir="$(copy_example go-runtime-environment)"
  nested_dir="${project_dir}/internal/e2e/nested"
  mkdir -p "${nested_dir}"
  {
    printf 'APP_MODE=%s\nFILE_ONLY=file-value\n' "${sentinel_file}"
    printf '%s\n' 'EQUALS_VALUE=first=second' 'HASH_VALUE=value#fragment' 'QUOTED_VALUE="literal-quotes"' "INTERPOLATED_VALUE=\${HOME}"
  } >"${project_dir}/.elyro/dev.env"
  printf 'APP_MODE=local-wins\nLOCAL_ONLY=local-value\n' >"${project_dir}/.elyro/user.local.env"
  python3 - "${project_dir}/elyro.yaml" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
text = path.read_text()
text = text.replace("        - .elyro/dev.env\n", "        - .elyro/dev.env\n        - .elyro/user.local.env\n")
path.write_text(text)
PY
  prepare_project_dir "${project_dir}"

  log "runtime environment: nested up and precedence"
  (cd "${nested_dir}" && "${BIN}" up --json) | jq -e '.action == "created"' >/dev/null
  first_container="$(container_for_project "${project_dir}")"
  test -n "${first_container}"
  test "$(cd "${nested_dir}" && "${BIN}" exec -- printenv APP_MODE)" = "inline-wins"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv FILE_ONLY)" = "file-value"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv LOCAL_ONLY)" = "local-value"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv EQUALS_VALUE)" = "first=second"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv HASH_VALUE)" = "value#fragment"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv QUOTED_VALUE)" = '"literal-quotes"'
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv INTERPOLATED_VALUE)" = "\${HOME}"
  host_alias="$(latest_host_alias)"
  test "$(run_ssh "${host_alias}" 'printenv APP_MODE')" = "inline-wins"
  (cd "${nested_dir}" && "${BIN}" open --print) | grep -Fq 'vscode-remote://ssh-remote+elyro-'

  report="${TMP_ROOT}/runtime-environment-doctor.json"
  (cd "${nested_dir}" && "${BIN}" doctor --json) >"${report}"
  jq -e '
    .schema_version == 2 and .healthy == true and
    .project.runtime_environment.variables == ["APP_MODE", "EMPTY_VALUE", "EQUALS_VALUE", "FILE_ONLY", "HASH_VALUE", "INTERPOLATED_VALUE", "LOCAL_ONLY", "QUOTED_VALUE"] and
    .project.runtime_environment.env_files == [".elyro/dev.env", ".elyro/user.local.env"] and
    any(.checks[]; .name == "runtime_environment" and .status == "ok")
  ' "${report}" >/dev/null
  if grep -Fq "${sentinel_file}" "${report}" || grep -Fq "${sentinel_inline}" "${report}"; then
    printf '[workspace-e2e] doctor leaked a runtime environment value\n' >&2
    return 1
  fi
  if docker inspect "${first_container}" --format '{{json .Config.Labels}}' | grep -Fq "${sentinel_file}"; then
    printf '[workspace-e2e] container labels leaked a runtime environment value\n' >&2
    return 1
  fi

  log "runtime environment: unchanged effective values reuse Workspace"
  {
    printf '%s\n' 'APP_MODE=source-changed-but-overridden' 'FILE_ONLY=file-value'
    printf '%s\n' 'EQUALS_VALUE=first=second' 'HASH_VALUE=value#fragment' 'QUOTED_VALUE="literal-quotes"' "INTERPOLATED_VALUE=\${HOME}"
  } >"${project_dir}/.elyro/dev.env"
  (cd "${nested_dir}" && "${BIN}" up --dry-run --json) | \
    jq -e '.action == "reuse" and .reasons == []' >/dev/null
  (cd "${nested_dir}" && "${BIN}" up --json) | jq -e '.action == "reused" and .reasons == []' >/dev/null
  test "$(container_for_project "${project_dir}")" = "${first_container}"

  log "runtime environment: effective value change recreates Workspace"
  printf 'APP_MODE=local-wins\nLOCAL_ONLY=changed-value\n' >"${project_dir}/.elyro/user.local.env"
  (cd "${nested_dir}" && "${BIN}" up --dry-run --json) | \
    jq -e '.action == "recreate" and .reasons == ["runtime_environment_changed"]' >/dev/null
  (cd "${nested_dir}" && "${BIN}" up --json) | \
    jq -e '.action == "recreated" and .reasons == ["runtime_environment_changed"]' >/dev/null
  second_container="$(container_for_project "${project_dir}")"
  test -n "${second_container}"
  test "${second_container}" != "${first_container}"
  test "$("${BIN}" exec --project-dir "${project_dir}" -- printenv LOCAL_ONLY)" = "changed-value"

  log "runtime environment: workspace down"
  (cd "${nested_dir}" && "${BIN}" down) >/dev/null
  assert_workspace_removed "${project_dir}"
}

run_doctor_unconfigured_case() {
  local project_dir report

  project_dir="${TMP_ROOT}/doctor-unconfigured"
  report="${TMP_ROOT}/doctor-unconfigured.json"
  mkdir -p "${project_dir}"
  project_dir="$(cd "${project_dir}" && pwd -P)"
  git -C "${project_dir}" init -q

  log "doctor: unconfigured Git project is healthy with warnings"
  (cd "${project_dir}" && "${BIN}" doctor --json) >"${report}"
  jq -e --arg root "${project_dir}" '
    .schema_version == 2 and
    .healthy == true and
    .project.root == $root and
    .project.source == "git" and
    any(.checks[]; .name == "workspace_configuration" and .status == "warn")
  ' "${report}" >/dev/null

  printf 'environments: [\n' >"${project_dir}/elyro.yaml"
  if (cd "${project_dir}" && "${BIN}" doctor --json) >"${report}" 2>/dev/null; then
    printf '[workspace-e2e] expected invalid elyro.yaml to make doctor fail\n' >&2
    return 1
  fi
  jq -e '
    .schema_version == 2 and
    .healthy == false and
    any(.checks[]; .name == "workspace_configuration" and .status == "fail")
  ' "${report}" >/dev/null
}

main() {
  require_cmd bash
  require_cmd curl
  require_cmd docker
  require_cmd git
  require_cmd go
  require_cmd jq
  require_cmd python3
  require_cmd ssh

  mkdir -p "${HOME}/.ssh"
  : >"${SSH_CONFIG}"

  build_workspace
  run_doctor_unconfigured_case

  local case_name
  for case_name in ${CASES}; do
    case "${case_name}" in
      python) run_python_case ;;
      go) run_go_case ;;
      node) run_node_case ;;
      java) run_java_case ;;
      environment) run_environment_case ;;
      image) run_project_image_case ;;
      runtime-environment) run_runtime_environment_case ;;
      *) printf '[workspace-e2e] unknown case: %s\n' "${case_name}" >&2; exit 1 ;;
    esac
  done

  log "all requested e2e cases passed: ${CASES}"
}

main "$@"
