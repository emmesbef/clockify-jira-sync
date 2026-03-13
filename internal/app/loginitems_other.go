//go:build !darwin

package app

func setLaunchOnStartup(enabled bool) error {
	return nil
}
