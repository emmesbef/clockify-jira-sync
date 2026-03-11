//go:build !darwin

package tray

// Init is a no-op on non-macOS platforms.
func Init(
	version string,
	icon []byte,
	onShow func(),
	onQuit func(),
	onCheckUpdates func(),
	onStartTimer func(ticketKey, description string),
	onStopTimer func(),
	onLoadAssignedTickets func() string,
	onSearchTickets func(query string) string,
) {
}

// SetWindowVisible is a no-op on non-macOS platforms.
func SetWindowVisible(visible bool) {}

// SetStatusText is a no-op on non-macOS platforms.
func SetStatusText(text string) {}

// SetTimerRunning is a no-op on non-macOS platforms.
func SetTimerRunning(running bool) {}

// SetAppBackgroundMode is a no-op on non-macOS platforms.
func SetAppBackgroundMode() {}

// SetAppForegroundMode is a no-op on non-macOS platforms.
func SetAppForegroundMode() {}
