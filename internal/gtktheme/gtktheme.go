// Package gtktheme reads the colors of the active GTK theme, so the UI can
// follow the desktop's theme.
package gtktheme

/*
#cgo pkg-config: gtk4
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include "gtktheme.h"
*/
import "C"

import (
	"sync"
	"unsafe"
)

// Read returns the palette of the active GTK theme as seen by widget, a
// GtkWidget pointer. Call it on the GTK main thread.
func Read(widget unsafe.Pointer) Theme {
	out := C.nexus_theme_colors(widget)
	defer C.free(unsafe.Pointer(out))
	return parse(C.GoString(out))
}

var (
	watchOnce sync.Once
	mu        sync.Mutex
	onChange  func()
)

// Watch calls fn on the GTK main thread whenever the GTK theme or its dark
// preference changes. Call it on the GTK main thread; later calls replace fn.
func Watch(fn func()) {
	mu.Lock()
	onChange = fn
	mu.Unlock()
	watchOnce.Do(func() { C.nexus_theme_watch() })
}

//export goGtkThemeChanged
func goGtkThemeChanged() {
	mu.Lock()
	fn := onChange
	mu.Unlock()
	if fn != nil {
		fn()
	}
}
