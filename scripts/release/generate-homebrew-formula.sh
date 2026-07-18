#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: generate-homebrew-formula.sh TAG DIST_DIR OUTPUT [RELEASE_BASE_URL]

Generate Formula/elyro.rb from GoReleaser archives and checksums. TAG must be a
v-prefixed semantic version. RELEASE_BASE_URL defaults to the Elyro GitHub
Releases download endpoint; an explicit file:// URL is useful for local smoke.
EOF
}

if [[ $# -lt 3 || $# -gt 4 ]]; then
  usage >&2
  exit 2
fi

tag="$1"
dist_dir="$2"
output="$3"
release_base_url="${4:-https://github.com/cofy-x/elyro/releases/download}"

if [[ ! "${tag}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)([+-][0-9A-Za-z.-]+)?$ ]]; then
  printf 'tag must be a v-prefixed semantic version, got %q\n' "${tag}" >&2
  exit 2
fi

version="${tag#v}"
checksums_file="${dist_dir}/checksums.txt"
if [[ ! -f "${checksums_file}" ]]; then
  printf 'missing GoReleaser checksums: %s\n' "${checksums_file}" >&2
  exit 1
fi

checksum_for() {
  local filename="$1"
  local checksum
  checksum="$(awk -v name="${filename}" '$2 == name { print $1 }' "${checksums_file}")"
  if [[ ! "${checksum}" =~ ^[0-9a-f]{64}$ ]]; then
    printf 'missing or invalid SHA-256 for %s\n' "${filename}" >&2
    exit 1
  fi
  printf '%s' "${checksum}"
}

darwin_amd64="elyro_${version}_darwin_amd64.tar.gz"
darwin_arm64="elyro_${version}_darwin_arm64.tar.gz"
linux_amd64="elyro_${version}_linux_amd64.tar.gz"
linux_arm64="elyro_${version}_linux_arm64.tar.gz"

darwin_amd64_sha="$(checksum_for "${darwin_amd64}")"
darwin_arm64_sha="$(checksum_for "${darwin_arm64}")"
linux_amd64_sha="$(checksum_for "${linux_amd64}")"
linux_arm64_sha="$(checksum_for "${linux_arm64}")"

mkdir -p "$(dirname "${output}")"
cat >"${output}" <<EOF
class Elyro < Formula
  desc "Local Linux workspaces for developers and coding agents"
  homepage "https://github.com/cofy-x/elyro"
  version "${version}"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "${release_base_url}/${tag}/${darwin_arm64}"
      sha256 "${darwin_arm64_sha}"
    else
      url "${release_base_url}/${tag}/${darwin_amd64}"
      sha256 "${darwin_amd64_sha}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "${release_base_url}/${tag}/${linux_arm64}"
      sha256 "${linux_arm64_sha}"
    else
      url "${release_base_url}/${tag}/${linux_amd64}"
      sha256 "${linux_amd64_sha}"
    end
  end

  def install
    bin.install "elyro"
  end

  test do
    assert_match %Q("version": "v#{version}"), shell_output("#{bin}/elyro version --json")
    assert_match "use-elyro-workspace", shell_output("#{bin}/elyro skill show")
  end
end
EOF

printf 'generated %s for %s\n' "${output}" "${tag}"
