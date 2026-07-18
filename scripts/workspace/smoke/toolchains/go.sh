#!/usr/bin/env bash
# Go workspace toolchain smoke checks.

workspace_smoke_toolchain_go() {
  local BIN="$1"
  local ROOT_DIR="$2"

  local TMP_GO_PROJECT TMP_GO_HOME TMP_GO_SSH_CONFIG TMP_GO_STATE_HOME HOST_ALIAS_GO GO_HTTP_PORT
  local REMOTE_DIR_GO
  TMP_GO_PROJECT="$(mktemp -d /tmp/elyro-workspace-smoke-go.XXXXXX)"
  TMP_GO_HOME="$(mktemp -d /tmp/elyro-workspace-home-go.XXXXXX)"
  TMP_GO_SSH_CONFIG="${TMP_GO_HOME}/.ssh/config"
  TMP_GO_STATE_HOME="$(mktemp -d /tmp/elyro-workspace-state-go.XXXXXX)"
  GO_HTTP_PORT="18001"
  REMOTE_DIR_GO="/home/elyro/$(basename "${TMP_GO_PROJECT}")"

  _workspace_smoke_go_cleanup() {
    HOME="${TMP_GO_HOME}" XDG_STATE_HOME="${TMP_GO_STATE_HOME}" "${BIN}" down --project-dir "${TMP_GO_PROJECT}" >/dev/null 2>&1 || true
    rm -rf "${TMP_GO_PROJECT}" "${TMP_GO_HOME}" "${TMP_GO_STATE_HOME}"
  }
  trap _workspace_smoke_go_cleanup EXIT

  cat >"${TMP_GO_PROJECT}/go.mod" <<'EOF_MOD'
module smoke

go 1.23.0
EOF_MOD

  cat >"${TMP_GO_PROJECT}/main.go" <<'EOF_MAIN'
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	log.Fatal(http.ListenAndServe(":8000", mux))
}
EOF_MAIN

  HOME="${TMP_GO_HOME}" XDG_STATE_HOME="${TMP_GO_STATE_HOME}" "${BIN}" up \
    --toolchain go \
    --project-dir "${TMP_GO_PROJECT}" \
    --publish "${GO_HTTP_PORT}:8000"

  local STATUS_GO
  STATUS_GO="$(HOME="${TMP_GO_HOME}" XDG_STATE_HOME="${TMP_GO_STATE_HOME}" "${BIN}" status --project-dir "${TMP_GO_PROJECT}" --json)"
  jq -e --arg published "${GO_HTTP_PORT}:8000" \
    '.workspace.toolchain == "go" and (.workspace.published_ports | index($published)) != null' <<<"${STATUS_GO}" >/dev/null

  local CONTAINER_GO
  CONTAINER_GO="$(docker ps -q --filter "label=elyro.workspace.project_dir=${TMP_GO_PROJECT}")"
  test -n "${CONTAINER_GO}"

  docker exec "${CONTAINER_GO}" bash -lc 'go version >/dev/null && ! command -v golangci-lint'
  docker exec "${CONTAINER_GO}" bash -lc "cd '${REMOTE_DIR_GO}' && nohup go run . >/tmp/workspace-go-http.log 2>&1 &"
  workspace_smoke_wait_http "http://127.0.0.1:${GO_HTTP_PORT}/healthz" 20
  curl -fsS "http://127.0.0.1:${GO_HTTP_PORT}/healthz" >/dev/null

  test ! -e "${TMP_GO_PROJECT}/elyro.yaml"
  test ! -e "${TMP_GO_PROJECT}/.vscode"
  HOST_ALIAS_GO="$(awk '/^Host elyro-/ {print $2; exit}' "${TMP_GO_SSH_CONFIG}")"
  test -n "${HOST_ALIAS_GO}"
  grep -Fq "IdentityFile \"${TMP_GO_HOME}/.ssh/elyro_workspace_ed25519\"" "${TMP_GO_SSH_CONFIG}"

  docker exec "${CONTAINER_GO}" bash -lc 'grep -Fq "PasswordAuthentication no" /etc/ssh/sshd_config.d/99-elyro-workspace.conf'
  ssh -F "${TMP_GO_SSH_CONFIG}" -o BatchMode=yes -o ConnectTimeout=10 "${HOST_ALIAS_GO}" '/usr/local/go/bin/go version >/dev/null'

  HOME="${TMP_GO_HOME}" XDG_STATE_HOME="${TMP_GO_STATE_HOME}" "${BIN}" down --project-dir "${TMP_GO_PROJECT}"

  trap - EXIT
  _workspace_smoke_go_cleanup
}
