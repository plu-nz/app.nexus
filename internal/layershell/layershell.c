#include <string.h>
#include <gtk/gtk.h>
#include <gtk4-layer-shell.h>
#include "layershell.h"

static GdkMonitor *find_monitor(const char *connector) {
    if (!connector || !*connector) return NULL;
    GListModel *monitors = gdk_display_get_monitors(gdk_display_get_default());
    for (guint i = 0; i < g_list_model_get_n_items(monitors); i++) {
        GdkMonitor *m = g_list_model_get_item(monitors, i);
        const char *name = gdk_monitor_get_connector(m);
        if (name && strcmp(name, connector) == 0) {
            g_object_unref(m);  // The display keeps the monitor alive.
            return m;
        }
        g_object_unref(m);
    }
    return NULL;
}

int nexus_layer_init(void *gtk_window, const char *connector) {
    GtkWindow *w = GTK_WINDOW(gtk_window);
    if (!gtk_layer_is_supported()) return 0;
    if (gtk_layer_is_layer_window(w)) return 1;
    if (gtk_widget_get_mapped(GTK_WIDGET(w))) return 0;
    // A layer surface must be set up before GTK creates the window's surface.
    if (gtk_widget_get_realized(GTK_WIDGET(w))) gtk_widget_unrealize(GTK_WIDGET(w));
    gtk_layer_init_for_window(w);
    gtk_layer_set_namespace(w, "nexus");
    // Overlay sits above fullscreen video, where a TV panel is needed most.
    gtk_layer_set_layer(w, GTK_LAYER_SHELL_LAYER_OVERLAY);
    gtk_layer_set_anchor(w, GTK_LAYER_SHELL_EDGE_TOP, TRUE);
    gtk_layer_set_anchor(w, GTK_LAYER_SHELL_EDGE_BOTTOM, TRUE);
    gtk_layer_set_anchor(w, GTK_LAYER_SHELL_EDGE_LEFT, TRUE);
    gtk_layer_set_anchor(w, GTK_LAYER_SHELL_EDGE_RIGHT, TRUE);
    // Cover bars too, instead of fitting between them.
    gtk_layer_set_exclusive_zone(w, -1);
    // A remote only has the keyboard, so the panel takes it while shown.
    gtk_layer_set_keyboard_mode(w, GTK_LAYER_SHELL_KEYBOARD_MODE_EXCLUSIVE);
    GdkMonitor *m = find_monitor(connector);
    if (m) gtk_layer_set_monitor(w, m);
    return 1;
}

void nexus_window_transparent(void *gtk_window) {
    GtkWidget *w = GTK_WIDGET(gtk_window);
    static GtkCssProvider *provider = NULL;
    if (!provider) {
        provider = gtk_css_provider_new();
        // Not fully transparent: with an alpha of 0 WebKit stops painting
        // the page at all (WebKitGTK 2.54, GTK 4.24).
        gtk_css_provider_load_from_string(provider, "window.nexus-transparent { background: rgba(0,0,0,0.01); }");
        gtk_style_context_add_provider_for_display(gtk_widget_get_display(w), GTK_STYLE_PROVIDER(provider),
                                                   GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
    }
    gtk_widget_add_css_class(w, "nexus-transparent");
}
