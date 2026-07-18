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

docker run --rm --platform "${platform}" "${prefix}/workspace-base:${tag}" bash -lc \
  'id elyro && command -v sshd git curl jq pgrep pkill install awk mktemp'
docker run --rm --platform "${platform}" "${prefix}/workspace-python:${tag}" bash -lc \
  "python --version && uv --version | grep -F 'uv ${ELYRO_UV_VERSION}'"
docker run --rm --platform "${platform}" "${prefix}/workspace-go:${tag}" bash -lc \
  "go version | grep -F 'go${ELYRO_GO_VERSION}' && ! command -v golangci-lint"
docker run --rm --platform "${platform}" "${prefix}/workspace-java:${tag}" bash -lc \
  "java -version && mvn --version && gradle --version | grep -F 'Gradle ${ELYRO_GRADLE_VERSION}'"
docker run --rm --platform "${platform}" "${prefix}/workspace-node:${tag}" bash -lc \
  "node --version | grep -F 'v${ELYRO_NODE_VERSION}' && npm --version && npx --version && corepack --version && python3 --version && command -v make g++ && test ! -e /usr/bin/dpkg-buildpackage"

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
