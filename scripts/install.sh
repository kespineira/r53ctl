#!/bin/sh
set -eu

repo="${R53CTL_REPO:-kespineira/r53ctl}"
version="${R53CTL_VERSION:-latest}"
install_dir="${R53CTL_INSTALL_DIR:-/usr/local/bin}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

need_cmd tar
need_cmd sed
need_cmd grep
need_cmd awk

if command -v curl >/dev/null 2>&1; then
  download() {
    curl -fsSL "$1" -o "$2"
  }
elif command -v wget >/dev/null 2>&1; then
  download() {
    wget -qO "$2" "$1"
  }
else
  echo "error: curl or wget is required" >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "error: unsupported OS: $(uname -s)" >&2
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

if [ "$version" = "latest" ]; then
  release_json="$tmp_dir/release.json"
  download "https://api.github.com/repos/$repo/releases/latest" "$release_json"
  version="$(sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' "$release_json" | head -n 1)"
  if [ -z "$version" ]; then
    echo "error: could not resolve latest release for $repo" >&2
    exit 1
  fi
fi

version_no_v="$(printf '%s' "$version" | sed 's/^v//')"
asset="r53ctl_${version_no_v}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$version"
archive="$tmp_dir/$asset"
checksums="$tmp_dir/checksums.txt"

download "$base_url/$asset" "$archive"
download "$base_url/checksums.txt" "$checksums"

# Verify the checksums file's cosign signature when cosign is available. This
# authenticates checksums.txt itself (the per-asset SHA-256 check below only
# guards transport integrity). Set R53CTL_SKIP_COSIGN=1 to bypass.
sig="$tmp_dir/checksums.txt.sig"
cert="$tmp_dir/checksums.txt.pem"
if [ "${R53CTL_SKIP_COSIGN:-0}" != "1" ] && command -v cosign >/dev/null 2>&1; then
  if download "$base_url/checksums.txt.sig" "$sig" && download "$base_url/checksums.txt.pem" "$cert"; then
    if cosign verify-blob \
      --certificate "$cert" \
      --signature "$sig" \
      --certificate-identity-regexp "^https://github.com/$repo/\.github/workflows/release\.yml@refs/tags/" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      "$checksums" >/dev/null 2>&1; then
      echo "verified checksums.txt signature with cosign"
    else
      echo "error: cosign signature verification failed for checksums.txt" >&2
      echo "       set R53CTL_SKIP_COSIGN=1 to bypass (not recommended)" >&2
      exit 1
    fi
  else
    echo "note: no cosign signature published for this release; skipping signature check" >&2
  fi
fi

expected="$(grep "  $asset\$" "$checksums" | awk '{print $1}')"
if [ -z "$expected" ]; then
  echo "error: checksum for $asset was not found" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$archive" | awk '{print $1}')"
else
  actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
fi

if [ "$expected" != "$actual" ]; then
  echo "error: checksum mismatch for $asset" >&2
  exit 1
fi

tar -xzf "$archive" -C "$tmp_dir"

if [ ! -d "$install_dir" ]; then
  mkdir -p "$install_dir"
fi

if [ -w "$install_dir" ]; then
  mv "$tmp_dir/r53ctl" "$install_dir/r53ctl"
else
  need_cmd sudo
  sudo mv "$tmp_dir/r53ctl" "$install_dir/r53ctl"
fi

echo "r53ctl $version installed to $install_dir/r53ctl"
