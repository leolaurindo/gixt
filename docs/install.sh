#!/bin/sh

set -eu

repo="leolaurindo/gixt"
version="${1:-latest}"
install_dir="${GIXT_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

if [ "$version" = "latest" ]; then
	release_url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")
	version=${release_url##*/}
else
	case "$version" in
		v*) ;;
		*) version="v$version" ;;
	esac
fi

case "$version" in
	v[0-9]*) ;;
	*) fail "could not determine a release version" ;;
esac

case "$(uname -s)" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) fail "unsupported architecture: $(uname -m)" ;;
esac

asset="gixt_${version}_${os}_${arch}.tar.gz"
release_base="https://github.com/$repo/releases/download/$version"
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' 0 HUP INT TERM

printf 'Downloading gixt %s for %s/%s...\n' "$version" "$os" "$arch"
curl -fsSL "$release_base/$asset" -o "$temp_dir/$asset"
curl -fsSL "$release_base/checksums.txt" -o "$temp_dir/checksums.txt"

expected=$(awk -v asset="$asset" '$2 == asset { print $1; exit }' "$temp_dir/checksums.txt")
[ -n "$expected" ] || fail "checksum not found for $asset"

if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$temp_dir/$asset" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
	actual=$(shasum -a 256 "$temp_dir/$asset" | awk '{ print $1 }')
else
	fail "sha256sum or shasum is required"
fi

[ "$actual" = "$expected" ] || fail "checksum verification failed"

tar -xzf "$temp_dir/$asset" -C "$temp_dir"
mkdir -p "$install_dir"
cp "$temp_dir/gixt" "$install_dir/gixt"
chmod 0755 "$install_dir/gixt"

printf 'Installed gixt %s to %s/gixt\n' "$version" "$install_dir"
case ":${PATH:-}:" in
	*":$install_dir:"*) ;;
	*) printf 'Add %s to PATH, then open a new shell.\n' "$install_dir" ;;
esac
