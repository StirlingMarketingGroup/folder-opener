package main

import (
	"net/url"
	"os/exec"
	"path/filepath"
)

func openDir(path string) error {
	return runQuiet(exec.Command("xdg-open", path))
}

// revealFile asks the desktop's file manager to select the file via the
// org.freedesktop.FileManager1 D-Bus interface (supported by Nautilus,
// Dolphin, Thunar, …), falling back to just opening the parent directory.
func revealFile(path string) error {
	uri := (&url.URL{Scheme: "file", Path: path}).String()
	err := runQuiet(exec.Command("dbus-send", "--session", "--print-reply",
		"--dest=org.freedesktop.FileManager1",
		"/org/freedesktop/FileManager1",
		"org.freedesktop.FileManager1.ShowItems",
		"array:string:"+uri, "string:"))
	if err == nil {
		return nil
	}
	return openDir(filepath.Dir(path))
}
