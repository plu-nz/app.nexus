// Package layershell shows the Nexus window as a wlr-layer-shell overlay,
// so the compositor places it over the whole output, above fullscreen
// apps, without per-compositor positioning.
package layershell

/*
#cgo pkg-config: gtk4-layer-shell-0 gtk4
#include <stdlib.h>
#include "layershell.h"
*/
import "C"

import "unsafe"

// Init makes window, a GtkWindow pointer, a full-screen overlay on the
// output named connector ("" lets the compositor choose). It reports false
// when the compositor has no layer shell; the window then stays a normal
// window. Call it on the GTK main thread before the window is shown.
func Init(window unsafe.Pointer, connector string) bool {
	c := C.CString(connector)
	defer C.free(unsafe.Pointer(c))
	return C.nexus_layer_init(window, c) == 1
}
