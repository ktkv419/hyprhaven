//go:build linux
// +build linux

package wallpaper

import (
	"os"
	"os/exec"
	"strconv"
)

// Desktop contains the current desktop environment on Linux.
var Desktop = os.Getenv("XDG_CURRENT_DESKTOP")

// SetFromFile sets wallpaper from a file path.
func SetFromFile(file string) error {
	if isGNOMECompliant() {
		return exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", strconv.Quote("file://"+file)).Run()
	}

	switch Desktop {
	case "KDE":
		return setKDE(file)
	case "X-Cinnamon":
		return exec.Command("dconf", "write", "/org/cinnamon/desktop/background/picture-uri", strconv.Quote("file://"+file)).Run()
	case "MATE":
		return exec.Command("dconf", "write", "/org/mate/desktop/background/picture-filename", strconv.Quote(file)).Run()
	case "XFCE":
		return setXFCE(file)
	case "LXDE":
		return exec.Command("pcmanfm", "-w", file).Run()
	case "Deepin":
		return exec.Command("dconf", "write", "/com/deepin/wrap/gnome/desktop/background/picture-uri", strconv.Quote("file://"+file)).Run()
	default:
		err := exec.Command("swaybg", "-i", file).Start()
		if err == nil {
			return nil
		}

		return exec.Command("feh", "--bg-fill", file).Run()
	}
}
