package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// autostartCLI handles `folder-opener autostart enable|disable|status`.
// Note: on Windows the release binary is built with -H=windowsgui, so these
// commands print nothing to a console; GPO deployments should set the
// HKCU\...\Run value directly instead.
func autostartCLI(args []string) {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}
	switch sub {
	case "enable":
		if err := enableAutostart(); err != nil {
			fmt.Fprintf(os.Stderr, "enable autostart: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("autostart enabled")
	case "disable":
		if err := disableAutostart(); err != nil {
			fmt.Fprintf(os.Stderr, "disable autostart: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("autostart disabled")
	case "status":
		if autostartEnabled() {
			fmt.Println("enabled")
		} else {
			fmt.Println("disabled")
		}
	default:
		fmt.Fprintf(os.Stderr, "usage: folder-opener autostart enable|disable|status\n")
		os.Exit(2)
	}
}

// executablePath resolves the real path of the running binary for use in
// autostart entries, so a symlinked invocation doesn't break at next login.
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}
