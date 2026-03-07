//go:build !darwin

package tray

// Init is a no-op on non-macOS platforms.
func Init(version string, icon []byte, onShow func(), onQuit func()) {}

// SetWindowVisible is a no-op on non-macOS platforms.
func SetWindowVisible(visible bool) {}
