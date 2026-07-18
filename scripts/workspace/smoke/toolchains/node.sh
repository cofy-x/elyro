#!/usr/bin/env bash
# Node.js workspace toolchain smoke checks.

workspace_smoke_toolchain_node() {
  local BIN="$1"
  local TMP_PROJECT TMP_HOME TMP_STATE_HOME STATUS_OUTPUT CONTAINER_ID
  TMP_PROJECT="$(mktemp -d /tmp/elyro-workspace-smoke-node.XXXXXX)"
  TMP_HOME="$(mktemp -d /tmp/elyro-workspace-home-node.XXXXXX)"
  TMP_STATE_HOME="$(mktemp -d /tmp/elyro-workspace-state-node.XXXXXX)"

  _workspace_smoke_node_cleanup() {
    HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}" >/dev/null 2>&1 || true
    rm -rf "${TMP_PROJECT}" "${TMP_HOME}" "${TMP_STATE_HOME}"
  }
  trap _workspace_smoke_node_cleanup EXIT

  cat >"${TMP_PROJECT}/package.json" <<'EOF_PACKAGE'
{"name":"elyro-node-smoke","private":true,"scripts":{"test":"node --test"}}
EOF_PACKAGE
  cat >"${TMP_PROJECT}/smoke.test.js" <<'EOF_TEST'
const test = require('node:test');
const assert = require('node:assert/strict');
test('node workspace', () => assert.equal(process.platform, 'linux'));
EOF_TEST

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" up --toolchain node --project-dir "${TMP_PROJECT}"
  STATUS_OUTPUT="$(HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" status --project-dir "${TMP_PROJECT}" --json)"
  jq -e '.schema_version == 1 and .workspace.toolchain == "node" and .workspace.status == "running"' <<<"${STATUS_OUTPUT}" >/dev/null
  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" exec --project-dir "${TMP_PROJECT}" -- npm test

  CONTAINER_ID="$(docker ps -q --filter "label=elyro.workspace.project_dir=${TMP_PROJECT}")"
  test -n "${CONTAINER_ID}"
  docker exec "${CONTAINER_ID}" bash -lc 'node --version && npm --version && corepack --version && python3 --version && command -v make g++'

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}"
  trap - EXIT
  _workspace_smoke_node_cleanup
}
