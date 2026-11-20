#!/usr/bin/env bash
# Install Nexus for the current user: nexus and nexus-toggle in
# ~/.local/bin, plus a desktop entry and icon. Inside a release archive it installs
# the prebuilt binary; in the repository it builds first.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -x "$script_dir/nexus" ]]; then
  binary="$script_dir/nexus"
  toggle="$script_dir/nexus-toggle"
  desktop="$script_dir/nz.plu.nexus.desktop"
  icon="$script_dir/nz.plu.nexus.svg"
else
  repo_dir="$(cd "$script_dir/.." && pwd)"
  cd "$repo_dir"
  pnpm install --frozen-lockfile
  pnpm build
  binary="$repo_dir/build/nexus"
  toggle="$repo_dir/scripts/toggle.sh"
  desktop="$repo_dir/assets/nz.plu.nexus.desktop"
  icon="$repo_dir/assets/nz.plu.nexus.svg"
fi

bin_dir="$HOME/.local/bin"
data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
install -Dm755 "$binary" "$bin_dir/nexus"
install -Dm755 "$toggle" "$bin_dir/nexus-toggle"
install -Dm644 "$desktop" "$data_dir/applications/nz.plu.nexus.desktop"
install -Dm644 "$icon" "$data_dir/icons/hicolor/scalable/apps/nz.plu.nexus.svg"
update-desktop-database "$data_dir/applications" 2>/dev/null || true
echo "Installed $bin_dir/nexus"
echo "A Nexus that is already running keeps the old version until it restarts: systemctl --user stop plunz-nexus"
