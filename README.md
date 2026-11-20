# Nexus

Nexus is a TV panel for Linux, made for a couch and a remote. It slides up from the bottom of the screen over whatever is playing and offers:

- the TV apps: Lounge and VacuumTube, with their own icons, when they are installed;
- a single horizontal row for apps, volume, audio output, brightness, and the sleep timer;
- Up/Down on volume adjusts by 5%; Enter mutes or unmutes;
- Up/Down on output switches between audio devices; Enter selects the next one;
- Up/Down on brightness adjusts by 10%; Enter increases it by 10%;
- Up/Down on sleep changes the remaining timer by 15 minutes, from off to four hours; Enter opens a confirmation dialog to suspend immediately;
- automatic language selection (English, Portuguese, or Spanish) from the system;
- colors that follow the desktop's GTK theme.

Everything works with the arrow keys, Enter, and Back (Escape or Backspace), so any remote that sends keyboard keys can drive it. Opening an app hides the panel. On Hyprland, an app that is already open is brought to the front, fullscreen, instead of starting again; a second press while it is still starting is ignored.
The panel closes with Back or Escape.

## Install

```sh
./scripts/install.sh
```

This builds Nexus and installs `nexus`, the `nexus-toggle` command, and a desktop entry for the current user. Bind `nexus-toggle` to a key or a remote button; on this Omarchy setup it is the air mouse's Power button, which udev remaps to F19. The first press starts Nexus, later presses show and hide the panel.

Nexus keeps running while hidden, so the sleep timer survives. To load a new version, stop the running one with `systemctl --user stop plunz-nexus`.

## Requirements

- Build: Go, pnpm, and the development files for `gtk4`, `webkitgtk-6.0`, and `gtk4-layer-shell`.
- Run: PipeWire with WirePlumber (`wpctl`, `pw-dump`) for audio, `brightnessctl` or `ddcutil` for brightness, and `systemctl` for power. `ddcutil` needs access to `/dev/i2c-*`, usually through the `i2c` group. A control whose tool is missing shows as unavailable.

## How it appears

On Wayland compositors with the layer shell (Hyprland, Sway, KDE, and other wlroots compositors) the panel is an overlay layer: it covers the whole output, above fullscreen video, and takes the keyboard while shown. The compositor puts it on the focused output; set `NEXUS_OUTPUT` to an output name (such as `HDMI-A-1`) to pin it to the TV. Where there is no layer shell, Nexus shows a normal window.

The panel is translucent, so what is playing stays visible behind it. Blurring what is behind it is up to the compositor. On Hyprland, add a layer rule for the `nexus` namespace (with `decoration.blur.enabled`):

```lua
hl.layer_rule({ match = { namespace = "^nexus$" }, blur = true, ignore_alpha = 0.3 })
```

`ignore_alpha` keeps the nearly clear dimming around the panel sharp.

## Theme

Nexus follows the desktop's GTK theme: it reads the theme's colors through GTK and updates when the theme or its light/dark preference changes. Its built-in colors are used when GTK colors are unavailable.

## Development

```sh
pnpm install
pnpm build    # frontend/dist and build/nexus
pnpm check    # tsc, go vet, go test
NEXUS_WINDOWED=1 ./build/nexus
```

`NEXUS_WINDOWED=1` shows a normal window instead of the overlay, and `NEXUS_DEV=1` enables the web inspector's context menu.

## Scope

The app catalog is fixed in `internal/apps`: Nexus only starts the apps listed there. Packaging beyond the per-user install is a next step.
