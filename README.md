# Nexus

Website and downloads: [nexus.plu.nz](https://nexus.plu.nz/).

Nexus is a TV panel for Linux, made for a couch and a remote. It slides up from the bottom of the screen over whatever is playing and offers:

- the installed TV apps from its catalog, with their own icons: Lounge, VacuumTube, Kodi, Jellyfin, Moonlight, Steam (in Big Picture), and RetroArch;
- a More apps tile that opens a store to install, update, and remove those apps from Flathub;
- a hidden Close apps button to the left of the apps, revealed with Left from the leftmost app; its confirmation closes every open window of the catalog apps;
- on an app that is open, Down reveals a Close button over its tile; Enter closes every window of that app only;
- a single horizontal row for apps, volume, audio output, brightness, and the sleep timer; the apps scroll on their own, so the controls stay in view;
- Up/Down on volume adjusts by 5%; Enter mutes or unmutes;
- when at least two audio outputs are available, Up/Down on output switches between them and Enter selects the next one;
- Up/Down on brightness adjusts by 10%; Enter increases it by 10%;
- Up/Down on sleep changes the remaining timer by 15 minutes, from off to four hours; Enter opens a confirmation dialog to suspend immediately;
- automatic language selection (English, Portuguese, or Spanish) from the system;
- colors that follow the desktop's GTK theme.

Everything works with the arrow keys, Enter, and Back (Escape, Backspace, GoBack, BrowserBack, XF86Back, or Back), so any remote that sends keyboard keys can drive it. Opening an app hides the panel. On Hyprland, an app that is already open is brought to the front, fullscreen, instead of starting again; a second press while it is still starting is ignored.
On Hyprland, confirming Close apps closes every open window of the catalog apps. The panel stays open.
Right, Up, Down, or Back from the Close apps button hides it and returns focus to the first app. Canceling the confirmation hides it too.
Down does nothing on an app that is not open. Up, Left, Right, or Back from an app's Close button hides it and returns focus to that app.
The panel closes with Back or Escape.

## Store

More apps opens a full-screen store with Video and Games tabs. Each app shows whether it is installed, installed for all users, or has an update. Its detail gives the description and, before installing, the download and disk sizes from Flathub, with a note when the install also brings shared runtime components.

- **Install** adds the app to the user's own Flatpak installation (`flatpak install --user` from Flathub). Nexus adds the flathub remote for the user when it is missing. No administrator password is needed.
- **Update** and **Remove** act only on apps in the user installation. Apps installed for all users can be opened but are changed with the system's software manager. Removing keeps the app's data in `~/.var/app`, so a reinstall keeps its settings.
- Operations run one at a time and continue while the panel is hidden. flatpak reports which component it is installing but no percentage, so the progress bar is indeterminate. Cancel stops the operation; partial downloads stay in Flatpak's cache.
- Lounge has its own installer, so the store links to [lounge.plu.nz](https://lounge.plu.nz/) for it.

Without `flatpak`, the store lists the catalog without install buttons.

## Install

Download `nexus-linux-x86_64.tar.gz` from the [latest release](https://github.com/plu-nz/app.nexus/releases/latest), install the libraries, and run the installer inside the archive:

```sh
sudo pacman -S --needed gtk4 webkitgtk-6.0 gtk4-layer-shell brightnessctl flatpak
tar -xzf nexus-linux-x86_64.tar.gz
cd nexus-*/ && ./install.sh
```

Release archives are built on Arch Linux for x86_64. To build from source instead, run the same script from the repository:

```sh
./scripts/install.sh
```

Either way, it installs `nexus`, the `nexus-toggle` command, a desktop entry, and an icon for the current user. Bind `nexus-toggle` to a key or a remote button; on this Omarchy setup it is the air mouse's Power button, which udev remaps to F19. The first press starts Nexus, later presses show and hide the panel.

Nexus keeps running while hidden, so the sleep timer survives. To load a new version, stop the running one with `systemctl --user stop plunz-nexus`.

## Requirements

- Build: Go, pnpm, and the development files for `gtk4`, `webkitgtk-6.0`, and `gtk4-layer-shell`.
- Store: `flatpak`, with network access to Flathub.
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

The app catalog is fixed in `internal/apps`: Nexus only starts, installs, updates, and removes the apps listed there. A catalog entry runs natively on Linux, works with the arrow keys, Enter, and Back, and fills the screen. Nexus never offers Android apps, through Waydroid or any other Android layer. Its window class was checked by running the app, so Nexus can bring it to the front. Packaging beyond the per-user install and the x86_64 release archive is a next step.

## Releases and website

Pushing a `v*` tag runs [the release workflow](.github/workflows/release.yml): it builds Nexus in an Arch Linux container and publishes `nexus-linux-x86_64.tar.gz` with its SHA-256 sum through [scripts/package.sh](scripts/package.sh). The landing page in [site/](site/README.md) links to that archive and is deployed separately to nexus.plu.nz.
