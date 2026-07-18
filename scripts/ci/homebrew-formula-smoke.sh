#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/elyro-homebrew-formula.XXXXXX")"
TAP_NAME=""
FORMULA_INSTALLED=0

cleanup() {
  if [[ "${FORMULA_INSTALLED}" == "1" ]]; then
    HOMEBREW_NO_AUTO_UPDATE=1 brew uninstall --formula elyro >/dev/null 2>&1 || true
  fi
  if [[ -n "${TAP_NAME}" ]]; then
    HOMEBREW_NO_AUTO_UPDATE=1 brew untap --force "${TAP_NAME}" >/dev/null 2>&1 || true
  fi
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

version="v0.0.0-ci"
release_version="${version#v}"
release_root="${TMP_DIR}/releases"
dist_dir="${release_root}/${version}"
formula="${TMP_DIR}/Formula/elyro.rb"
mkdir -p "${dist_dir}"

for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  archive="elyro_${release_version}_${platform}.tar.gz"
  printf 'fixture for %s\n' "${platform}" >"${dist_dir}/${archive}"
done

if [[ "${ELYRO_HOMEBREW_INSTALL_SMOKE:-0}" == "1" ]]; then
  if [[ "$(uname -s)" != "Darwin" ]]; then
    printf 'Homebrew installation smoke currently requires macOS\n' >&2
    exit 2
  fi
  case "$(uname -m)" in
    arm64) host_platform="darwin_arm64" ;;
    x86_64) host_platform="darwin_amd64" ;;
    *)
      printf 'unsupported Homebrew smoke architecture: %s\n' "$(uname -m)" >&2
      exit 2
      ;;
  esac
  make -C "${ROOT_DIR}" \
    ELYRO_BIN_DIR="${TMP_DIR}/bin" ELYRO_VERSION="${version}" elyro-build
  tar -czf "${dist_dir}/elyro_${release_version}_${host_platform}.tar.gz" \
    -C "${TMP_DIR}/bin" elyro
fi

for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  archive="elyro_${release_version}_${platform}.tar.gz"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "${dist_dir}" && sha256sum "${archive}")
  else
    (cd "${dist_dir}" && shasum -a 256 "${archive}")
  fi
done >"${dist_dir}/checksums.txt"

"${ROOT_DIR}/scripts/release/generate-homebrew-formula.sh" \
  "${version}" "${dist_dir}" "${formula}" "file://${release_root}"

grep -q '^class Elyro < Formula$' "${formula}"
grep -q 'bin.install "elyro"' "${formula}"
grep -q 'on_macos do' "${formula}"
grep -q 'on_linux do' "${formula}"
grep -q 'test do' "${formula}"

build_metadata_formula="${TMP_DIR}/Formula/elyro-build-metadata.rb"
for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  cp "${dist_dir}/elyro_${release_version}_${platform}.tar.gz" \
    "${dist_dir}/elyro_0.0.0+ci.1_${platform}.tar.gz"
done
for platform in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64; do
  archive="elyro_0.0.0+ci.1_${platform}.tar.gz"
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "${dist_dir}" && sha256sum "${archive}")
  else
    (cd "${dist_dir}" && shasum -a 256 "${archive}")
  fi
done >>"${dist_dir}/checksums.txt"
"${ROOT_DIR}/scripts/release/generate-homebrew-formula.sh" \
  "v0.0.0+ci.1" "${dist_dir}" "${build_metadata_formula}" "file://${release_root}"
grep -q 'version "0.0.0+ci.1"' "${build_metadata_formula}"

if [[ "${ELYRO_HOMEBREW_INSTALL_SMOKE:-0}" == "1" ]]; then
  if brew list --formula elyro >/dev/null 2>&1; then
    printf 'refusing to replace an existing Homebrew elyro installation\n' >&2
    exit 2
  fi
  TAP_NAME="cofy-x/elyro-ci-${GITHUB_RUN_ID:-$$}"
  HOMEBREW_NO_AUTO_UPDATE=1 brew tap-new "${TAP_NAME}" >/dev/null
  tap_dir="$(brew --repository "${TAP_NAME}")"
  mkdir -p "${tap_dir}/Formula"
  cp "${formula}" "${tap_dir}/Formula/elyro.rb"
  HOMEBREW_NO_AUTO_UPDATE=1 brew style --formula "${TAP_NAME}/elyro"
  HOMEBREW_NO_AUTO_UPDATE=1 brew install --formula "${TAP_NAME}/elyro"
  FORMULA_INSTALLED=1
  HOMEBREW_NO_AUTO_UPDATE=1 brew test "${TAP_NAME}/elyro"
  elyro version --json | grep -F '"version": "v0.0.0-ci"' >/dev/null
  elyro --help | grep -F 'Edit on Mac. Build and test in Linux.' >/dev/null
  elyro skill show | grep -F 'use-elyro-workspace' >/dev/null
fi

if "${ROOT_DIR}/scripts/release/generate-homebrew-formula.sh" \
  latest "${dist_dir}" "${TMP_DIR}/invalid.rb" >/dev/null 2>&1; then
  printf 'floating Homebrew version unexpectedly accepted\n' >&2
  exit 1
fi

printf 'homebrew formula generation smoke passed\n'
