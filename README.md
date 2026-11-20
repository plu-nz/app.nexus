# Nexus

Nexus is a TV panel for Linux, made for a couch and a remote. It slides up from the bottom of the screen over whatever is playing and offers:

- the TV apps: Lounge and VacuumTube, with their own icons, when they are installed;
- volume, mute, and the audio output (HDMI, speakers, Bluetooth…);
- display brightness: a laptop backlight through `brightnessctl`, or a monitor or TV through DDC/CI with `ddcutil`;
- a sleep timer that suspends the machine;
- suspend, restart, and power off, with a second press to confirm restart and power off;
- the language (English, Portuguese, Spanish) and the theme.

Everything works with the arrow keys, Enter, and Back (Escape or Backspace), so any remote that sends keyboard keys can drive it. Opening an app hides the panel. On Hyprland, an app that is already open is brought to the front, fullscreen, instead of starting again; a second press while it is still starting is ignored.

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

On Wayland compositors with the layer shell (Hyprland, Sway, KDE, and other wlroots compositors) the panel is an overlay layer: it covers the whole output, above fullscreen video, and takes the keyboard while shown. Clicking outside the panel hides it. The compositor puts it on the focused output; set `NEXUS_OUTPUT` to an output name (such as `HDMI-A-1`) to pin it to the TV. Where there is no layer shell, Nexus shows a normal window.

The panel is translucent, so what is playing stays visible behind it. Blurring what is behind it is up to the compositor. On Hyprland, add a layer rule for the `nexus` namespace (with `decoration.blur.enabled`):

```lua
hl.layer_rule({ match = { namespace = "^nexus$" }, blur = true, ignore_alpha = 0.3 })
```

`ignore_alpha` keeps the nearly clear dimming around the panel sharp.

## Theme

By default Nexus follows the desktop's GTK theme: it reads the theme's colors through GTK and updates when the theme or its light/dark preference changes. The Theme button switches to Nexus's own colors instead.

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
