# routa-bin AUR Package

This directory contains the AUR metadata for the binary package.

Release flow:

1. Publish a GitHub release tag. The `release artifacts` workflow uploads:
   - `routa_<version>_linux_amd64.tar.gz`
   - `routa_<version>_linux_arm64.tar.gz`
   - `routa_<version>_checksums.txt`
2. Update `pkgver` in `PKGBUILD`.
3. Replace `sha256sums_*` with the matching hashes from the release checksum
   file.
4. Regenerate `.SRCINFO`:

   ```bash
   makepkg --printsrcinfo > .SRCINFO
   ```

The package installs only `/usr/bin/routa` and docs. Runtime setup still happens
through `routa init` and `routa install`.
