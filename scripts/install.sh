#!/usr/bin/env bash
# Builds Nexus and installs it for the current user.
set -euo pipefail
repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_dir"
pnpm install --frozen-lockfile
pnpm build
install -Dm755 build/nexus "$HOME/.local/bin/nexus"
install -Dm755 scripts/toggle.sh "$HOME/.local/bin/nexus-toggle"
install -Dm644 assets/nz.plu.nexus.desktop "$HOME/.local/share/applications/nz.plu.nexus.desktop"
echo "Installed $HOME/.local/bin/nexus"
echo "A Nexus that is already running keeps the old version until it restarts: systemctl --user stop plunz-nexus"
