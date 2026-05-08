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
		asset_arch="amd64"
		;;
	aarch64 | arm64)
		asset_arch="arm64"
		;;
	*)
		echo "unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

dist="dist"
work="${RUNNER_TEMP:-/tmp}/routa-php-xdebug-${asset_arch}"
base_extensions="${ROUTA_PHP_XDEBUG_BASE_EXTENSIONS:-bcmath,zlib}"
spc_repo="${ROUTA_PHP_XDEBUG_SPC_REPO:-https://github.com/crazywhalecc/static-php-cli.git}"

mkdir -p "$dist"
rm -rf "$work"
mkdir -p "$work"

for version in "${versions[@]}"; do
	build_dir="$work/php-$version"
	rm -rf "$build_dir"
	git clone --depth=1 "$spc_repo" "$build_dir"
	chmod +x "$build_dir/bin/spc-gnu-docker"

	echo "building Xdebug for PHP $version linux/$asset_arch"
	(
		cd "$build_dir"
		bin/spc-gnu-docker download --with-php="$version" --for-extensions="${base_extensions},xdebug" --prefer-pre-built --retry=2
		bin/spc-gnu-docker build "$base_extensions" --build-cli --build-shared=xdebug --debug
		test -f buildroot/modules/xdebug.so
		tar -C buildroot/modules -czf "$repo/$dist/routa_php_xdebug_${version}_linux_${asset_arch}.tar.gz" xdebug.so
	)
done

(
	cd "$dist"
	sha256sum routa_php_xdebug_*_linux_"$asset_arch".tar.gz > "routa_php_xdebug_linux_${asset_arch}_checksums.txt"
)

echo "wrote $dist/routa_php_xdebug_*_linux_${asset_arch}.tar.gz"
