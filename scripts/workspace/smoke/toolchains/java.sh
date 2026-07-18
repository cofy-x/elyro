#!/usr/bin/env bash
# Java workspace toolchain smoke checks.

workspace_smoke_toolchain_java() {
  local BIN="$1"
  local TMP_PROJECT TMP_HOME TMP_STATE_HOME STATUS_OUTPUT CONTAINER_ID
  TMP_PROJECT="$(mktemp -d /tmp/elyro-workspace-smoke-java.XXXXXX)"
  TMP_HOME="$(mktemp -d /tmp/elyro-workspace-home-java.XXXXXX)"
  TMP_STATE_HOME="$(mktemp -d /tmp/elyro-workspace-state-java.XXXXXX)"

  _workspace_smoke_java_cleanup() {
    HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}" >/dev/null 2>&1 || true
    rm -rf "${TMP_PROJECT}" "${TMP_HOME}" "${TMP_STATE_HOME}"
  }
  trap _workspace_smoke_java_cleanup EXIT

  cat >"${TMP_PROJECT}/Ready.java" <<'EOF_JAVA'
public final class Ready {
  public static void main(String[] args) {
    if (!System.getProperty("os.name").toLowerCase().contains("linux")) {
      throw new IllegalStateException("expected Linux");
    }
    System.out.println("java workspace is ready");
  }
}
EOF_JAVA

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" up --toolchain java --project-dir "${TMP_PROJECT}"
  STATUS_OUTPUT="$(HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" status --project-dir "${TMP_PROJECT}" --json)"
  jq -e '.schema_version == 1 and .workspace.toolchain == "java" and .workspace.status == "running"' <<<"${STATUS_OUTPUT}" >/dev/null
  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" exec --project-dir "${TMP_PROJECT}" -- \
    bash -lc 'javac Ready.java && java Ready'

  CONTAINER_ID="$(docker ps -q --filter "label=elyro.workspace.project_dir=${TMP_PROJECT}")"
  test -n "${CONTAINER_ID}"
  docker exec "${CONTAINER_ID}" bash -lc 'java -version && mvn --version && gradle --version'

  HOME="${TMP_HOME}" XDG_STATE_HOME="${TMP_STATE_HOME}" "${BIN}" down --project-dir "${TMP_PROJECT}"
  trap - EXIT
  _workspace_smoke_java_cleanup
}
