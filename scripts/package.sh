#!/usr/bin/env bash
# Package a built Nexus (pnpm build) as a release archive:
# build/release/nexus-linux-x86_64.tar.gz, whose fixed name lets
# /releases/latest/download/ link to it, and its SHA-256 sum. The archive
# holds a nexus-<version> directory.
set -euo pipefail

version="${1:?usage: scripts/package.sh <version>}"
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

name="nexus-linux-x86_64"
stage="build/release/nexus-$version"
rm -rf build/release
mkdir -p "$stage"
install -m755 build/nexus scripts/install.sh "$stage/"
install -m755 scripts/toggle.sh "$stage/nexus-toggle"
install -m644 assets/nz.plu.nexus.desktop assets/nz.plu.nexus.svg README.md "$stage/"
tar -C build/release -czf "build/release/$name.tar.gz" "nexus-$version"
(cd build/release && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
echo "build/release/$name.tar.gz"
