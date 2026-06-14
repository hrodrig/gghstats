#!/usr/bin/env sh
# Install gghstats from GitHub releases.
# Usage: curl -fsSL https://raw.githubusercontent.com/hrodrig/gghstats/main/scripts/install.sh | sh
# Or: BINDIR=~/bin sh install.sh
# Pin a release: VERSION=v0.7.4 sh install.sh  (or VERSION=0.7.4)

set -e

REPO="hrodrig/gghstats"
BINDIR="${BINDIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

detect_platform() {
  _os=""
  _arch=""
  _ext="tar.gz"

  case "$(uname -s)" in
    Linux)   _os="linux" ;;
    Darwin)  _os="darwin" ;;
    FreeBSD) _os="freebsd" ;;
    OpenBSD) _os="openbsd" ;;
    MINGW*|MSYS*|CYGWIN*) _os="windows" ;;
    *)
      echo "Unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) _arch="amd64" ;;
    aarch64|arm64) _arch="arm64" ;;
    *)
      echo "Unsupported arch: $(uname -m)" >&2
      exit 1
      ;;
  esac

  if [ "$_os" = "windows" ]; then
    _ext="zip"
  fi

  echo "${_os} ${_arch} ${_ext}"
}

get_latest_tag() {
  _api="https://api.github.com/repos/${REPO}/releases/latest"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${_api}" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/'
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "${_api}" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/'
  else
    echo "curl or wget required" >&2
    exit 1
  fi
}

normalize_tag() {
  case "$1" in
    v*) echo "$1" ;;
    *) echo "v$1" ;;
  esac
}

tag_to_version() {
  echo "$1" | sed 's/^v//'
}

main() {
  _platform=$(detect_platform)
  _os=$(echo "${_platform}" | awk '{print $1}')
  _arch=$(echo "${_platform}" | awk '{print $2}')
  _ext=$(echo "${_platform}" | awk '{print $3}')

  if [ "$VERSION" = "latest" ]; then
    _tag=$(get_latest_tag)
  else
    _tag=$(normalize_tag "$VERSION")
  fi

  _ver=$(tag_to_version "${_tag}")
  _name="gghstats_${_ver}_${_os}_${_arch}"
  _url="https://github.com/${REPO}/releases/download/${_tag}/${_name}.${_ext}"
  _bin_name="gghstats"
  if [ "$_os" = "windows" ]; then
    _bin_name="gghstats.exe"
  fi

  echo "Installing gghstats ${_tag} (${_os}/${_arch}) to ${BINDIR}"

  _tmpdir=$(mktemp -d)
  trap 'rm -rf "${_tmpdir}"' EXIT

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "${_tmpdir}/archive.${_ext}" "${_url}"
  else
    wget -q -O "${_tmpdir}/archive.${_ext}" "${_url}"
  fi

  if [ "$_ext" = "zip" ]; then
    if command -v unzip >/dev/null 2>&1; then
      unzip -q -o "${_tmpdir}/archive.${_ext}" -d "${_tmpdir}"
    else
      echo "unzip required for Windows" >&2
      exit 1
    fi
  else
    tar -xzf "${_tmpdir}/archive.${_ext}" -C "${_tmpdir}"
  fi

  _binary="${_tmpdir}/${_bin_name}"
  if [ ! -f "${_binary}" ]; then
    _binary="${_tmpdir}/${_name}/${_bin_name}"
  fi
  if [ ! -f "${_binary}" ]; then
    echo "Binary not found in archive" >&2
    exit 1
  fi

  mkdir -p "${BINDIR}"
  if [ -w "${BINDIR}" ]; then
    cp "${_binary}" "${BINDIR}/${_bin_name}"
    chmod +x "${BINDIR}/${_bin_name}"
  else
    echo "Need sudo to write to ${BINDIR}"
    sudo cp "${_binary}" "${BINDIR}/${_bin_name}"
    sudo chmod +x "${BINDIR}/${_bin_name}"
  fi

  echo "Installed: ${BINDIR}/${_bin_name}"
  "${BINDIR}/${_bin_name}" version 2>/dev/null || true
}

main
