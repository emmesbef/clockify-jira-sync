//go:build darwin && cgo

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
	onShow               func()
	onQuit               func()
	onCheckUpdates       func()
	onStartTimer         func(ticketKey, description string)
	onStopTimer          func(comment string)
	onCancelTimer        func()
	onLoadAssignedTicket func() string
	onSearchTicket       func(query string) string
)

// Init creates the macOS status bar icon and context menu.
// It must be called after the main run loop has started (e.g., from Wails OnStartup).
// onShowFn is called when "Show/Hide Window" is clicked.
// onQuitFn is called when "Quit" is clicked.
// onCheckUpdatesFn is called when "Check for Updates…" is clicked.
// onStartTimerFn is called when "Start Timer…" is submitted.
// onStopTimerFn is called when "Stop Timer" is confirmed, with optional comment text.
// onCancelTimerFn is called when "Cancel Timer" is clicked.
// onLoadAssignedTicketFn loads the top assigned tickets for empty/focus state.
// onSearchTicketFn loads matching tickets for a non-empty query.
func Init(
	version string,
	icon []byte,
	onShowFn func(),
	onQuitFn func(),
	onCheckUpdatesFn func(),
	onStartTimerFn func(ticketKey, description string),
	onStopTimerFn func(comment string),
	onCancelTimerFn func(),
	onLoadAssignedTicketFn func() string,
	onSearchTicketFn func(query string) string,
) {
	onShow = onShowFn
	onQuit = onQuitFn
	onCheckUpdates = onCheckUpdatesFn
	onStartTimer = onStartTimerFn
	onStopTimer = onStopTimerFn
	onCancelTimer = onCancelTimerFn
	onLoadAssignedTicket = onLoadAssignedTicketFn
	onSearchTicket = onSearchTicketFn

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

// SetStatusText sets timer status text next to the tray icon.
func SetStatusText(text string) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.setTrayStatusText(cText)
}

// SetTooltip sets the hover tooltip text for the tray icon.
func SetTooltip(text string) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.setTrayTooltip(cText)
}

// SetTimerRunning updates the tray timer action item title/state (Start vs Stop).
func SetTimerRunning(running bool) {
	v := C.int(0)
	if running {
		v = 1
	}
	C.setTrayTimerRunning(v)
}

// SetAppBackgroundMode switches to accessory mode so the app behaves as a tray/background app.
func SetAppBackgroundMode() {
	C.setTrayAppBackgroundMode()
}

// SetAppForegroundMode switches to regular app mode so the main window can be foregrounded.
func SetAppForegroundMode() {
	C.setTrayAppForegroundMode()
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

//export goTrayCheckUpdates
func goTrayCheckUpdates() {
	if onCheckUpdates != nil {
		onCheckUpdates()
	}
}

//export goTrayStartTimer
func goTrayStartTimer(ticketKey *C.char, description *C.char) {
	if onStartTimer != nil {
		onStartTimer(C.GoString(ticketKey), C.GoString(description))
	}
}

//export goTrayStopTimer
func goTrayStopTimer(comment *C.char) {
	if onStopTimer != nil {
		if comment == nil {
			onStopTimer("")
			return
		}
		onStopTimer(C.GoString(comment))
	}
}

//export goTrayCancelTimer
func goTrayCancelTimer() {
	if onCancelTimer != nil {
		onCancelTimer()
	}
}

//export goTrayLoadAssignedTickets
func goTrayLoadAssignedTickets() *C.char {
	if onLoadAssignedTicket == nil {
		return C.CString("[]")
	}
	return C.CString(onLoadAssignedTicket())
}

//export goTraySearchTickets
func goTraySearchTickets(query *C.char) *C.char {
	if onSearchTicket == nil {
		return C.CString("[]")
	}
	return C.CString(onSearchTicket(C.GoString(query)))
}
