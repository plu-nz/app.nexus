#!/usr/bin/env bash
# Shows or hides the Nexus panel, starting Nexus when it is not running.
# Bind it to a key or a remote button.
set -euo pipefail
app_path="$(command -v nexus || true)"
if [[ -z "$app_path" ]]; then app_path="$(cd "$(dirname "$0")/.." && pwd)/build/nexus"; fi
if "$app_path" --toggle 2>/dev/null; then exit; fi
# Its own user unit keeps Nexus (and its sleep timer) running after the
# caller exits, with logs in the journal.
exec systemd-run --user --unit=plunz-nexus --collect "$app_path" >/dev/null
