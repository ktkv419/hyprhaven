//go:build linux
// +build linux

package wallpaper

import "strings"

func isGNOMECompliant() bool {
	return strings.Contains(Desktop, "GNOME") || Desktop == "Unity" || Desktop == "Pantheon"
}
