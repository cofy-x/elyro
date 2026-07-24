#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=../../release/versions.env
source "${ROOT_DIR}/release/versions.env"

platform="${ELYRO_BUILD_PLATFORM:-linux/amd64}"
version="${ELYRO_BUILD_VERSION:-ci}"
prefix="${ELYRO_BUILD_IMAGE_PREFIX:-elyro}"
case "${platform}" in linux/amd64) arch=amd64 ;; linux/arm64) arch=arm64 ;; *) printf 'unsupported smoke platform: %s\n' "${platform}" >&2; exit 1 ;; esac
tag="${version}-${arch}"

smoke_shell_experience() {
  local image="$1"
  local -a run_args=(
    --rm
    --platform "${platform}"
    --hostname demo
    --user elyro
    --workdir /home/elyro/workspace
    --entrypoint zsh
    "${image}"
  )
  local success_prompt failure_prompt cwd_prompt plain_prompt dumb_prompt ls_output

  success_prompt="$(docker run "${run_args[@]}" -ic 'true; print -r -- ${(%):-$PROMPT}')"
  failure_prompt="$(docker run "${run_args[@]}" -ic 'false; print -r -- ${(%):-$PROMPT}')"
  [[ "${success_prompt}" == *'elyro:demo'* && "${success_prompt}" == *'~/workspace'* ]]
  [[ "${success_prompt}" == *$'\033[32m'* ]]
  [[ "${failure_prompt}" == *$'\033[31m'* ]]

  cwd_prompt="$(docker run "${run_args[@]}" -ic 'cd /tmp; print -r -- ${(%):-$PROMPT}')"
  [[ "${cwd_prompt}" == *'/tmp'* ]]

  ls_output="$(docker run --tty "${run_args[@]}" -ic 'mkdir -p /tmp/elyro-color-test/directory; ls /tmp/elyro-color-test')"
  [[ "${ls_output}" == *$'\033['* ]]

  plain_prompt="$(docker run -e NO_COLOR=1 "${run_args[@]}" -ic 'print -r -- ${(%):-$PROMPT}')"
  [[ "${plain_prompt}" == 'elyro:demo ~/workspace ❯ ' ]]
  [[ "${plain_prompt}" != *$'\033['* ]]

  dumb_prompt="$(docker run -e TERM=dumb "${run_args[@]}" -ic 'print -r -- ${(%):-$PROMPT}')"
  [[ "${dumb_prompt}" == 'elyro:demo ~/workspace ❯ ' ]]
  [[ "${dumb_prompt}" != *$'\033['* ]]

  docker run "${run_args[@]}" -ic \
    'alias ls | grep -F -- "--color=auto" && (( ${+functions[_zsh_autosuggest_start]} )) && (( ${+functions[_zsh_highlight]} ))'
  if docker run --rm --platform "${platform}" "${image}" env | grep -q '^DEBIAN_FRONTEND='; then
    printf 'runtime image still exports DEBIAN_FRONTEND: %s\n' "${image}" >&2
    return 1
  fi
}

docker run --rm --platform "${platform}" "${prefix}/workspace-base:${tag}" bash -lc \
  'id elyro && command -v sshd git curl jq pgrep pkill install awk mktemp'
docker run --rm --platform "${platform}" "${prefix}/workspace-python:${tag}" bash -lc \
  "python --version && uv --version | grep -F 'uv ${ELYRO_UV_VERSION}'"
docker run --rm --platform "${platform}" "${prefix}/workspace-go:${tag}" bash -lc \
  "go version | grep -F 'go${ELYRO_GO_VERSION}' && ! command -v golangci-lint"
docker run --rm --platform "${platform}" "${prefix}/workspace-node:${tag}" bash -lc \
  "node --version | grep -F 'v${ELYRO_NODE_VERSION}' && npm --version && npx --version && corepack --version && python3 --version && command -v make g++ && test ! -e /usr/bin/dpkg-buildpackage"

for image in workspace-base workspace-python workspace-go workspace-node; do
  smoke_shell_experience "${prefix}/${image}:${tag}"
done

docker run --rm --interactive --platform "${platform}" "${prefix}/workspace-node:${tag}" bash -s <<'EOF_NODE_GYP'
set -euo pipefail
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT
cd "${tmp_dir}"
cat >binding.gyp <<'EOF_GYP'
{"targets":[{"target_name":"addon","sources":["addon.cc"]}]}
EOF_GYP
cat >addon.cc <<'EOF_CPP'
#include <node_api.h>

napi_value Answer(napi_env env, napi_callback_info info) {
  (void)info;
  napi_value result;
  napi_create_int32(env, 42, &result);
  return result;
}

napi_value Init(napi_env env, napi_value exports) {
  napi_value answer;
  napi_create_function(env, "answer", NAPI_AUTO_LENGTH, Answer, nullptr, &answer);
  napi_set_named_property(env, exports, "answer", answer);
  return exports;
}

NAPI_MODULE(NODE_GYP_MODULE_NAME, Init)
EOF_CPP
node_gyp=/usr/local/lib/node_modules/npm/node_modules/node-gyp/bin/node-gyp.js
test -f "${node_gyp}"
node "${node_gyp}" configure build --nodedir=/usr/local
node -e 'const addon=require("./build/Release/addon.node"); if (addon.answer() !== 42) process.exit(1)'
EOF_NODE_GYP
