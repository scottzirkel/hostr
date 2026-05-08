#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
repo="$PWD"

if (($# == 0)); then
	versions=(8.2.30 8.3.30 8.4.20 8.5.5)
else
	versions=("$@")
fi

case "$(uname -m)" in
	x86_64)
		spc_arch="x86_64"
		asset_arch="amd64"
		;;
	aarch64 | arm64)
		spc_arch="aarch64"
		asset_arch="arm64"
		;;
	*)
		echo "unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

dist="dist"
work="${RUNNER_TEMP:-/tmp}/routa-php-xdebug-${asset_arch}"
spc_target="${SPC_TARGET:-native-native-gnu.2.17}"
base_extensions="${ROUTA_PHP_XDEBUG_BASE_EXTENSIONS:-bcmath,zlib}"

mkdir -p "$dist"
rm -rf "$work"
mkdir -p "$work"

curl -fsSL \
	-o "$work/spc.tgz" \
	"https://dl.static-php.dev/static-php-cli/spc-bin/nightly/spc-linux-${spc_arch}.tar.gz"
tar -xzf "$work/spc.tgz" -C "$work"
spc="$(find "$work" -maxdepth 2 -type f -name spc | head -n 1)"
if [[ -z "$spc" ]]; then
	echo "spc binary not found in archive" >&2
	exit 1
fi
chmod +x "$spc"

for version in "${versions[@]}"; do
	build_dir="$work/php-$version"
	rm -rf "$build_dir"
	mkdir -p "$build_dir"
	cp "$spc" "$build_dir/spc"

	echo "building Xdebug for PHP $version linux/$asset_arch"
	(
		cd "$build_dir"
		export SPC_TARGET="$spc_target"
		./spc download --with-php="$version" --for-extensions="${base_extensions},xdebug" --prefer-pre-built --retry=2
		./spc build "$base_extensions" --build-cli --build-shared=xdebug --debug
		test -f buildroot/modules/xdebug.so
		tar -C buildroot/modules -czf "$repo/$dist/routa_php_xdebug_${version}_linux_${asset_arch}.tar.gz" xdebug.so
	)
done

(
	cd "$dist"
	sha256sum routa_php_xdebug_*_linux_"$asset_arch".tar.gz > "routa_php_xdebug_linux_${asset_arch}_checksums.txt"
)

echo "wrote $dist/routa_php_xdebug_*_linux_${asset_arch}.tar.gz"
