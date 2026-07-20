#!/usr/bin/env bash
set -euo pipefail

REPOSITORY="cofy-x/elyro"
version="${ELYRO_INSTALL_VERSION:-}"
install_dir="${ELYRO_INSTALL_DIR:-${HOME}/.local/bin}"
release_url="${ELYRO_RELEASE_URL:-}"
release_dir="${ELYRO_RELEASE_DIR:-}"

usage() {
  cat <<'EOF'
Install an Elyro GitHub Release archive.

Usage:
  scripts/install.sh --version VERSION [--install-dir DIR]
                     [--release-url URL | --release-dir DIR]

Options:
  --version VERSION   Exact v-prefixed release tag, for example v0.1.4.
  --install-dir DIR   Destination for elyro.
                      Default: $ELYRO_INSTALL_DIR or $HOME/.local/bin.
  --release-url URL   Exact release asset directory. Intended for mirrors and
                      validation; defaults to the matching GitHub Release.
  --release-dir DIR   Local directory containing release assets. Intended for
                      offline validation and mutually exclusive with URL.
  -h, --help          Show this help.
EOF
}

require_value() {
  if [[ $# -lt 2 || -z "$2" ]]; then
    echo "error: $1 requires a value" >&2
    exit 2
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      require_value "$@"
      version="$2"
      shift 2
      ;;
    --install-dir)
      require_value "$@"
      install_dir="$2"
      shift 2
      ;;
    --release-url)
      require_value "$@"
      release_url="${2%/}"
      shift 2
      ;;
    --release-dir)
      require_value "$@"
      release_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ ! "${version}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "error: --version must be an exact v-prefixed release tag, for example v0.1.4" >&2
  exit 2
fi

if [[ -n "${release_url}" && -n "${release_dir}" ]]; then
  echo "error: --release-url and --release-dir are mutually exclusive" >&2
  exit 2
fi

for command_name in tar install mktemp awk; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "error: required command not found: ${command_name}" >&2
    exit 1
  fi
done

if [[ -n "${release_dir}" ]]; then
  if ! command -v cp >/dev/null 2>&1; then
    echo "error: required command not found: cp" >&2
    exit 1
  fi
  if [[ ! -d "${release_dir}" ]]; then
    echo "error: release directory not found: ${release_dir}" >&2
    exit 1
  fi
elif ! command -v curl >/dev/null 2>&1; then
  echo "error: required command not found: curl" >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "error: unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "error: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [[ -z "${release_dir}" ]]; then
  release_url="${release_url:-https://github.com/${REPOSITORY}/releases/download/${version}}"
fi
archive="elyro_${version#v}_${os}_${arch}.tar.gz"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/elyro-install.XXXXXX")"
stage_dir=""

cleanup() {
  rm -rf "${work_dir}"
  if [[ -n "${stage_dir}" ]]; then
    rm -rf "${stage_dir}"
  fi
}
trap cleanup EXIT

download() {
  local name="$1"
  if [[ -n "${release_dir}" ]]; then
    cp "${release_dir}/${name}" "${work_dir}/${name}"
    return
  fi
  curl --fail --location --silent --show-error \
    --retry 3 --connect-timeout 15 \
    --output "${work_dir}/${name}" \
    "${release_url}/${name}"
}

download "checksums.txt"
download "${archive}"

expected_checksum="$(awk -v archive="${archive}" '$2 == archive { print $1; exit }' "${work_dir}/checksums.txt")"
if [[ ! "${expected_checksum}" =~ ^[0-9a-fA-F]{64}$ ]]; then
  echo "error: ${archive} is missing from checksums.txt" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum="$(sha256sum "${work_dir}/${archive}" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum="$(shasum -a 256 "${work_dir}/${archive}" | awk '{ print $1 }')"
else
  echo "error: sha256sum or shasum is required" >&2
  exit 1
fi

if [[ "${actual_checksum}" != "${expected_checksum}" ]]; then
  echo "error: checksum mismatch for ${archive}" >&2
  exit 1
fi

extract_dir="${work_dir}/extract"
mkdir -p "${extract_dir}"
tar -xzf "${work_dir}/${archive}" -C "${extract_dir}" elyro

for binary in elyro; do
  if [[ ! -f "${extract_dir}/${binary}" || -L "${extract_dir}/${binary}" ]]; then
    echo "error: ${binary} is missing or is not a regular archive file" >&2
    exit 1
  fi
done

mkdir -p "${install_dir}"
stage_dir="$(mktemp -d "${install_dir}/.elyro-install.XXXXXX")"
install -m 0755 "${extract_dir}/elyro" "${stage_dir}/elyro"
mv -f "${stage_dir}/elyro" "${install_dir}/elyro"

echo "installed elyro ${version} to ${install_dir}/elyro"
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) echo "add ${install_dir} to PATH before running elyro" ;;
esac
