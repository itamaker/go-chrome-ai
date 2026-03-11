#!/usr/bin/env bash

set -euo pipefail

PROJECT="go-chrome-ai"
DESCRIPTION="Patch Chrome Local State to enable Ask Gemini and other AI features"
CHECKSUMS_FILE="dist/SHA256SUMS"
OWNER=""
REPO="${PROJECT}"
VERSION=""

usage() {
  echo "Render a cask for https://github.com/itamaker/homebrew-tap" >&2
  echo "Usage: $0 --owner <project-repo-owner> --version <v1.0.3> [--repo <project-repo-name>] [--checksums <path>]" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --owner)
      OWNER="$2"
      shift 2
      ;;
    --repo)
      REPO="$2"
      shift 2
      ;;
    --version)
      VERSION="${2#v}"
      shift 2
      ;;
    --checksums)
      CHECKSUMS_FILE="$2"
      shift 2
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

if [[ -z "${OWNER}" || -z "${VERSION}" ]]; then
  usage
  exit 1
fi

if [[ ! -f "${CHECKSUMS_FILE}" ]]; then
  echo "checksums file not found: ${CHECKSUMS_FILE}" >&2
  exit 1
fi

checksum_for() {
  local arch="$1"
  local artifact="${PROJECT}-darwin-${arch}.tar.gz"
  awk -v artifact="${artifact}" '$2 == artifact { print $1 }' "${CHECKSUMS_FILE}"
}

darwin_arm64="$(checksum_for arm64)"
darwin_amd64="$(checksum_for amd64)"

for checksum in "${darwin_arm64}" "${darwin_amd64}"; do
  if [[ -z "${checksum}" ]]; then
    echo "missing required checksum in ${CHECKSUMS_FILE}" >&2
    exit 1
  fi
done

cat <<EOF
cask "${PROJECT}" do
  arch arm: "arm64", intel: "amd64"

  version "${VERSION}"
  sha256 arm: "${darwin_arm64}",
         intel: "${darwin_amd64}"

  url "https://github.com/${OWNER}/${REPO}/releases/download/v#{version}/${PROJECT}-darwin-#{arch}.tar.gz"
  name "${PROJECT}"
  desc "${DESCRIPTION}"
  homepage "https://github.com/${OWNER}/${REPO}"

  binary "${PROJECT}"

  livecheck do
    url :url
    strategy :github_latest
  end
end
EOF
