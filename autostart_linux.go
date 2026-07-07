package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// XDG autostart is the right hook here rather than a systemd unit: the tray
// icon needs a desktop session anyway, and .desktop autostart entries work
// across desktop environments.
func autostartDesktopPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "folder-opener.desktop"), nil
}

func enableAutostart() error {
	exe, err := executablePath()
	if err != nil {
		return err
	}
	path, err := autostartDesktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Folder Opener
Comment=Local "open folder in file browser" companion server
Exec=%s
X-GNOME-Autostart-enabled=true
`, exe)
	return os.WriteFile(path, []byte(desktop), 0o644)
}

func disableAutostart() error {
	path, err := autostartDesktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func autostartEnabled() bool {
	path, err := autostartDesktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
