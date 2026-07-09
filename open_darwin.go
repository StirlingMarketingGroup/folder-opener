package main

import "os/exec"

func openDir(path string) error {
	return runQuiet(exec.Command("open", path))
}

func revealFile(path string) error {
	return runQuiet(exec.Command("open", "-R", path))
}

// openDenied: no recovery exists on macOS — the server runs as the desktop
// user (login item), so Finder would be denied exactly the same way.
func openDenied(string) (string, bool) { return "", false }
