#ifndef NEXUS_LAYERSHELL_H
#define NEXUS_LAYERSHELL_H

// nexus_layer_init turns a GTK window into a full-screen overlay layer that
// takes the keyboard while shown, on the output named by connector, or on
// the one the compositor picks when connector is empty. It returns 1 when
// the window is a layer surface, 0 when the compositor has no layer shell
// or the window is already on screen. Call it on the GTK main thread.
int nexus_layer_init(void *gtk_window, const char *connector);

// nexus_window_transparent stops GTK from painting the theme's window
// background behind the page, so only what the page paints is opaque.
// Wails does not do this for GTK 4. Call it on the GTK main thread, after
// nexus_layer_init.
void nexus_window_transparent(void *gtk_window);

#endif
