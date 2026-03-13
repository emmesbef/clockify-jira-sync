//go:build darwin

package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const loginItemName = "JiraFy Clockwork"

func setLaunchOnStartup(enabled bool) error {
	launchPath, err := currentLaunchPath()
	if err != nil {
		return err
	}

	var script string
	if enabled {
		script = fmt.Sprintf(`tell application "System Events"
if exists login item "%s" then
	set path of login item "%s" to "%s"
	set hidden of login item "%s" to false
else
	make login item at end with properties {name:"%s", path:"%s", hidden:false}
end if
end tell`, loginItemName, loginItemName, escapeAppleScriptString(launchPath), loginItemName, loginItemName, escapeAppleScriptString(launchPath))
	} else {
		script = fmt.Sprintf(`tell application "System Events"
if exists login item "%s" then
	delete login item "%s"
end if
end tell`, loginItemName, loginItemName)
	}

	if err := runAppleScript(script); err != nil {
		return fmt.Errorf("failed to apply launch-on-startup setting: %w", err)
	}
	return nil
}

func currentLaunchPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return "", fmt.Errorf("failed to resolve executable path: empty path")
	}

	if idx := strings.Index(exePath, ".app/Contents/MacOS/"); idx >= 0 {
		return exePath[:idx+4], nil
	}

	return exePath, nil
}

func runAppleScript(script string) error {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func escapeAppleScriptString(input string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(input)
}
