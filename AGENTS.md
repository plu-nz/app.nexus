# Agent instructions

Nexus (`app.nexus`) is a PLUNZ standalone Linux TV panel. Shared PLUNZ decisions are in the parent workspace. Lounge (`app.lounge`) is its sibling and shares its stack and conventions.

## Architecture

The app is one Go binary built with Wails 3. The window is GTK 4 with WebKitGTK 6.0, and the UI is SolidJS + TypeScript built by Vite.

- `main.go` wires the services, makes the window an overlay, and runs the single-instance socket. `services.go` holds the Wails services the UI calls by name (`main.<Service>.<Method>`); they are thin adapters.
- `internal/apps`: the fixed app catalog, install detection, launch-or-focus (on Hyprland it focuses an open window of the app's class through `hyprctl` and makes it fullscreen; otherwise it starts the app in its own session, reaped), and icon lookup from `.desktop` files and the hicolor theme.
- `internal/audio`: volume, mute, and the default output through `wpctl` and `pw-dump`.
- `internal/brightness`: `brightnessctl` for backlights, `ddcutil` (DDC/CI) for monitors and TVs.
- `internal/sleeptimer`: the suspend countdown. It lives in the process so it survives hiding the panel.
- `internal/instance`: the per-user Unix socket; a second launch toggles the running panel.
- `internal/layershell`: gtk4-layer-shell setup, a full-screen overlay layer with exclusive keyboard.
- `internal/gtktheme`: reads the active GTK theme's named colors through GTK and reports theme changes; the UI gets them as `theme` events.
- `frontend/src`:
  - `main.tsx`: the panel;
  - `api.ts`: the typed service calls;
  - `nav.ts`: spatial navigation for remotes (arrow keys move to the nearest `[data-nav]` element) and the `.focused` class;
  - `theme.ts`: maps the GTK palette onto the `--plz-*` tokens;
  - `i18n.ts`: the en/pt/es strings.
- `scripts/toggle.sh` toggles the panel or starts Nexus as the `plunz-nexus` user unit.

## Conventions

- Write code, comments, commits, and docs in English. Every user-facing string goes through `t()`, with entries in all three dictionaries.
- Focusable controls are real `<button>`s marked `data-nav`. Style focus with the `.focused` class, not `:focus`: WebKit drops `:focus` while the window is unfocused. No sliders or `<select>`s; a remote cannot drag, and GTK popups break the panel's focus.
- Use `rem` for sizes. The root font size scales with the screen width.
- Style with the `--plz-*` tokens in `frontend/src/style.css`, never hard-coded colors, so the panel follows the GTK theme. Black is fine only for the scrim over other content.
- Only run fixed commands with fixed arguments; never pass user input to a shell. The app catalog stays explicit.
- Keep services thin and put logic in `internal/` packages with Go tests.
- Pin dependencies. Use pnpm only.

## Commands

```bash
pnpm install          # dependencies
pnpm build            # frontend/dist + build/nexus
pnpm check            # tsc, go vet, go test
./scripts/install.sh  # install for the current user
```

## Testing in a real window

UI changes need a check in the real window. On Hyprland you can test without disturbing the user's screen:

1. Create a headless output: `hyprctl output create headless`.
2. Run the test instance with its own `XDG_RUNTIME_DIR` (a short path, for the socket) and under `dbus-run-session`. Otherwise it would toggle the user's running Nexus, whose GTK application ID it shares. Set `WAYLAND_DISPLAY` to the absolute socket path.
3. For key tests, use `NEXUS_WINDOWED=1`, launch with `hl.dsp.exec_cmd(cmd, { workspace = '<n> silent' })`, and send keys with `hl.dsp.send_key_state({ ..., window = 'pid:<test pid>' })`.
4. For the overlay, set `NEXUS_OUTPUT` to the headless output and keep the run short: the overlay takes the user's keyboard while it is shown.
5. Capture the output with `grim -o <output>`.

Never press the power buttons in a test. Brightness and volume act on the real machine.

## Completion checks

- `pnpm check` and `pnpm build` pass.
- The changed flow works in the window with arrow keys, Enter, and Back.
- New strings exist in en, pt, and es.
- `README.md` and this file still describe the app.
