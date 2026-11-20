#ifndef NEXUS_GTKTHEME_H
#define NEXUS_GTKTHEME_H

// nexus_theme_colors lists the named colors the active GTK theme defines,
// one "name=r,g,b,a" line each, plus "@name=" and "@dark=" lines. Call it on
// the GTK main thread. The caller frees the result.
char *nexus_theme_colors(void *gtk_widget);

// nexus_theme_watch calls goGtkThemeChanged when the GTK theme or its dark
// preference changes. Call it once on the GTK main thread.
void nexus_theme_watch(void);

#endif
