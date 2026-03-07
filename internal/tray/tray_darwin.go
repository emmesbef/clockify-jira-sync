//go:build darwin

package tray

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa

#include <stdlib.h>
#include "tray_darwin.h"
*/
import "C"
import "unsafe"

var (
	onShow func()
	onQuit func()
)

// Init creates the macOS status bar icon and context menu.
// It must be called after the main run loop has started (e.g., from Wails OnStartup).
// onShowFn is called when "Show/Hide Window" is clicked.
// onQuitFn is called when "Quit" is clicked.
func Init(version string, icon []byte, onShowFn func(), onQuitFn func()) {
	onShow = onShowFn
	onQuit = onQuitFn

	cVersion := C.CString(version)
	defer C.free(unsafe.Pointer(cVersion))

	var iconPtr unsafe.Pointer
	var iconLen C.int
	if len(icon) > 0 {
		iconPtr = unsafe.Pointer(&icon[0])
		iconLen = C.int(len(icon))
	}

	C.initTray(cVersion, iconPtr, iconLen)
}

// SetWindowVisible updates the tray menu Show/Hide label.
func SetWindowVisible(visible bool) {
	v := C.int(0)
	if visible {
		v = 1
	}
	C.setTrayWindowVisible(v)
}

//export goTrayShow
func goTrayShow() {
	if onShow != nil {
		onShow()
	}
}

//export goTrayQuit
func goTrayQuit() {
	if onQuit != nil {
		onQuit()
	}
}
