//go:build windows
// +build windows

package wallpaper

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724947.aspx
const (
	spiSetDeskWallpaper = 0x0014

	uiParam = 0x0000

	spifUpdateINIFile = 0x01
	spifSendChange    = 0x02
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms724947.aspx
var (
	user32               = syscall.NewLazyDLL("user32.dll")
	systemParametersInfo = user32.NewProc("SystemParametersInfoW")
)

// isAdmin checks if the script is running as Administrator.
func isAdmin() bool {
	cmd := exec.Command("net", "session")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Run()
	return err == nil
}

// runAsAdmin relaunches the script with Administrator privileges.
func runAsAdmin() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	cmd := exec.Command("powershell", "Start-Process", exe, "-Verb", "RunAs")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run as administrator: %v", err)
	}

	os.Exit(0) // Exit current process, new one will start with admin rights
}

func checkRegistryValues(filename string) bool {
	expectedValues := map[string]interface{}{
		`SOFTWARE\Policies\Microsoft\Windows\Personalization`: map[string]interface{}{
			"LockScreenImage": filename,
		},
		`SOFTWARE\Microsoft\Windows\CurrentVersion\PersonalizationCSP`: map[string]interface{}{
			"LockScreenImageStatus": uint32(1),
			"LockScreenImagePath":   filename,
			"LockScreenImageUrl":    filename,
		},
	}

	for keyPath, values := range expectedValues {
		// Open registry key
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil {
			return false
		}
		defer key.Close()

		// Iterate through expected values
		for valueName, expected := range values.(map[string]interface{}) {
			switch expected := expected.(type) {
			case string:
				val, _, err := key.GetStringValue(valueName)
				if err != nil || val != expected {
					return false
				}
			case uint32:
				val, _, err := key.GetIntegerValue(valueName)
				if err != nil || val != uint64(expected) {
					return false
				}
			}
		}
	}

	return true
}

func setLockscreen(filename string) error {
	if !isAdmin() {
		runAsAdmin()
		return nil // Exit after requesting elevation
	}

	// Set lockscreen
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Policies\Microsoft\Windows\Personalization`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if err := key.SetStringValue("LockScreenImage", filename); err != nil {
		return err
	}

	// Set PersonalizationCSP settings
	cspKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\PersonalizationCSP`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer cspKey.Close()

	if err := cspKey.SetDWordValue("LockScreenImageStatus", 1); err != nil {
		return err
	}
	if err := cspKey.SetStringValue("LockScreenImagePath", filename); err != nil {
		return err
	}
	if err := cspKey.SetStringValue("LockScreenImageUrl", filename); err != nil {
		return err
	}
	return nil
}

// SetFromFile sets the wallpaper for the current user and mirrors it to the
// lock screen.
func SetFromFile(filename string) error {
	filenameUTF16, err := syscall.UTF16PtrFromString(filename)
	if err != nil {
		return err
	}

	if !checkRegistryValues(filename) {
		err := setLockscreen(filename)

		if err != nil {
			return err
		}
	}

	systemParametersInfo.Call(
		uintptr(spiSetDeskWallpaper),
		uintptr(uiParam),
		uintptr(unsafe.Pointer(filenameUTF16)),
		uintptr(spifUpdateINIFile|spifSendChange),
	)

	return nil
}
