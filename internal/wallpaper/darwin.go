//go:build darwin
// +build darwin

package wallpaper

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// macOS 15 broke the old `System Events` / `Finder` "set picture" AppleScript
// commands: they silently rewrite the wallpaper store to a relative path and
// fall back to the system wallpaper. NSWorkspace is the supported API and does
// not require Automation permissions.
const setWallpaperScript = `
use framework "AppKit"
use scripting additions

on run argv
	set imagePath to item 1 of argv
	set imageURL to current application's NSURL's fileURLWithPath:imagePath
	set workspace to current application's NSWorkspace's sharedWorkspace()
	set options to current application's NSDictionary's dictionary()
	set screens to current application's NSScreen's screens()

	repeat with screen in screens
		set {ok, err} to workspace's setDesktopImageURL:imageURL forScreen:screen options:options |error|:(reference)
		if not ok then error (err as text)
	end repeat
end run
`

// SetFromFile sets the desktop wallpaper for every screen to the given file.
func SetFromFile(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}

	out, err := exec.Command("osascript", "-e", setWallpaperScript, file).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("osascript: %w: %s", err, msg)
		}
		return fmt.Errorf("osascript: %w", err)
	}
	return nil
}
