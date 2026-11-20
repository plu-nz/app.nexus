#include <math.h>
#include <gtk/gtk.h>
#include "gtktheme.h"

extern void goGtkThemeChanged(void);

// Color names from libadwaita first, then the older GTK theme names, so
// both kinds of theme resolve.
static const char *names[] = {
    "window_bg_color", "window_fg_color", "view_bg_color", "view_fg_color",
    "card_bg_color", "headerbar_bg_color", "accent_bg_color", "accent_fg_color",
    "accent_color", "destructive_bg_color", "success_color", "warning_color",
    "error_color", "theme_bg_color", "theme_fg_color", "theme_base_color",
    "theme_text_color", "theme_selected_bg_color", "theme_selected_fg_color",
    "insensitive_fg_color", "borders", NULL,
};

static int channel(float v) { return (int)lroundf(CLAMP(v, 0.0f, 1.0f) * 255.0f); }

char *nexus_theme_colors(void *gtk_widget) {
    GString *out = g_string_new(NULL);
    // GtkStyleContext is deprecated, but it is still the only way to read a
    // theme's named colors, and GTK 4 keeps it working.
    G_GNUC_BEGIN_IGNORE_DEPRECATIONS
    GtkStyleContext *ctx = gtk_widget_get_style_context(GTK_WIDGET(gtk_widget));
    for (int i = 0; names[i]; i++) {
        GdkRGBA c;
        if (gtk_style_context_lookup_color(ctx, names[i], &c)) {
            g_string_append_printf(out, "%s=%d,%d,%d,%.3f\n", names[i], channel(c.red), channel(c.green), channel(c.blue),
                                   CLAMP(c.alpha, 0.0f, 1.0f));
        }
    }
    G_GNUC_END_IGNORE_DEPRECATIONS
    char *theme = NULL;
    gboolean dark = FALSE;
    g_object_get(gtk_settings_get_default(), "gtk-theme-name", &theme, "gtk-application-prefer-dark-theme", &dark, NULL);
    // GTK_THEME overrides the setting, and GTK does not update the setting.
    const char *forced = g_getenv("GTK_THEME");
    const char *name = forced && *forced ? forced : theme;
    g_string_append_printf(out, "@name=%s\n@dark=%d\n", name ? name : "", dark ? 1 : 0);
    g_free(theme);
    return g_string_free(out, FALSE);
}

static void on_notify(GObject *obj, GParamSpec *spec, gpointer data) { goGtkThemeChanged(); }

void nexus_theme_watch(void) {
    GtkSettings *s = gtk_settings_get_default();
    g_signal_connect(s, "notify::gtk-theme-name", G_CALLBACK(on_notify), NULL);
    g_signal_connect(s, "notify::gtk-application-prefer-dark-theme", G_CALLBACK(on_notify), NULL);
}
