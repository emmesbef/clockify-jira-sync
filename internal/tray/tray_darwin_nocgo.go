//go:build darwin && !cgo

package tray

// Init is a no-op when building for macOS without cgo.
func Init(
	version string,
	icon []byte,
	onShow func(),
	onQuit func(),
	onCheckUpdates func(),
	onStartTimer func(ticketKey, description string),
	onStopTimer func(comment string),
	onCancelTimer func(),
	onLoadAssignedTickets func() string,
	onSearchTickets func(query string) string,
) {
}

// SetWindowVisible is a no-op on macOS builds without cgo.
func SetWindowVisible(visible bool) {}

// SetStatusText is a no-op on macOS builds without cgo.
func SetStatusText(text string) {}

// SetTooltip is a no-op on macOS builds without cgo.
func SetTooltip(text string) {}

// SetTimerRunning is a no-op on macOS builds without cgo.
func SetTimerRunning(running bool) {}

// SetAppBackgroundMode is a no-op on macOS builds without cgo.
func SetAppBackgroundMode() {}

// SetAppForegroundMode is a no-op on macOS builds without cgo.
func SetAppForegroundMode() {}
