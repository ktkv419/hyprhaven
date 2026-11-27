//go:build linux
// +build linux

package wallpaper

import (
	"os/exec"
	"path"
	"strings"
)

func getXFCEProps(key string) ([]string, error) {
	output, err := exec.Command("xfconf-query", "--channel", "xfce4-desktop", "--list").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.Trim(string(output), "\n"), "\n")
	var desktops []string

	for _, line := range lines {
		if path.Base(line) == key {
			desktops = append(desktops, line)
		}
	}

	return desktops, nil
}

func setXFCE(file string) error {
	desktops, err := getXFCEProps("last-image")
	if err != nil {
		return err
	}
	for _, desktop := range desktops {
		err := exec.Command("xfconf-query", "--channel", "xfce4-desktop", "--property", desktop, "--set", file).Run()
		if err != nil {
			return err
		}
	}
	return nil
}
