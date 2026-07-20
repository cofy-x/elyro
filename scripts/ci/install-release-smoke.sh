#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
version="${1:-v0.1.4}"
dist_dir="${2:-}"
fixture_mode=0
temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/elyro-install-smoke.XXXXXX")"
trap 'rm -rf "${temp_dir}"' EXIT

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "[install-release-smoke] unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "[install-release-smoke] unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

archive="elyro_${version#v}_${os}_${arch}.tar.gz"

if [[ -z "${dist_dir}" ]]; then
  fixture_mode=1
  dist_dir="${temp_dir}/dist"
  fixture_dir="${temp_dir}/fixture"
  mkdir -p "${dist_dir}" "${fixture_dir}"
  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'set -euo pipefail' \
    '[[ "${1:-}" == "version" && "${2:-}" == "--json" ]]' \
    "printf '%s\\n' '{\"version\":\"${version}\"}'" \
    >"${fixture_dir}/elyro"
  chmod 0755 "${fixture_dir}/elyro"
  tar -czf "${dist_dir}/${archive}" -C "${fixture_dir}" elyro
  if command -v sha256sum >/dev/null 2>&1; then
    checksum="$(sha256sum "${dist_dir}/${archive}" | awk '{ print $1 }')"
  else
    checksum="$(shasum -a 256 "${dist_dir}/${archive}" | awk '{ print $1 }')"
  fi
  printf '%s  %s\n' "${checksum}" "${archive}" >"${dist_dir}/checksums.txt"
else
  dist_dir="$(cd "${dist_dir}" && pwd)"
fi

install_dir="${temp_dir}/install"
"${ROOT_DIR}/scripts/install.sh" \
  --version "${version}" \
  --install-dir "${install_dir}" \
  --release-dir "${dist_dir}"

test -x "${install_dir}/elyro"
"${install_dir}/elyro" version --json | grep -q '"version"'

if [[ "${fixture_mode}" == "0" ]]; then
  project_root="${temp_dir}/projects"
  mkdir -p "${project_root}/existing"
  printf 'module example.com/release-smoke\n' >"${project_root}/existing/go.mod"
  prerequisite_bin="${temp_dir}/prerequisite-bin"
  mkdir -p "${prerequisite_bin}"
  printf '%s\n' \
    '#!/usr/bin/env bash' \
    'set -euo pipefail' \
    '[[ "${1:-}" == "info" ]]' \
    >"${prerequisite_bin}/docker"
  chmod 0755 "${prerequisite_bin}/docker"
  PATH="${prerequisite_bin}:${PATH}" "${install_dir}/elyro" init \
    --project-dir "${project_root}/existing" --yes
  grep -q 'toolchain: go' "${project_root}/existing/elyro.yaml"
fi

if [[ "${fixture_mode}" == "1" ]]; then
  printf 'tampered\n' >>"${dist_dir}/${archive}"
  if "${ROOT_DIR}/scripts/install.sh" \
    --version "${version}" \
    --install-dir "${temp_dir}/tampered-install" \
    --release-dir "${dist_dir}" >"${temp_dir}/tampered.log" 2>&1; then
    echo "[install-release-smoke] tampered archive was accepted" >&2
    exit 1
  fi
  grep -q 'checksum mismatch' "${temp_dir}/tampered.log"

  if "${ROOT_DIR}/scripts/install.sh" \
    --version latest \
    --release-dir "${dist_dir}" >"${temp_dir}/floating.log" 2>&1; then
    echo "[install-release-smoke] floating version was accepted" >&2
    exit 1
  fi
  grep -q 'exact v-prefixed release tag' "${temp_dir}/floating.log"
fi

echo "[install-release-smoke] passed for ${os}/${arch}"
