package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	errNotFound     = errors.New("path does not exist")
	errBadPath      = errors.New("bad path")
	errAccessDenied = errors.New("access denied")
)

// openPath validates the path and opens it in the system file browser.
// Directories open directly; files are revealed (selected) in their parent
// folder. Unlike Windows Explorer's default behavior, a missing path is a
// real error — we never silently open a fallback location.
//
// smb:// URLs are resolved to a local path first (mounting the share on
// demand where the platform needs it) and then opened like any other path.
func openPath(path string) (action string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%w: empty path", errBadPath)
	}
	if isSMBURL(path) {
		if path, err = resolveSMB(path); err != nil {
			return "", err
		}
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: path must be absolute: %q", errBadPath, path)
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %q", errNotFound, path)
		}
		if errors.Is(err, os.ErrPermission) {
			// This process's identity can be narrower than the desktop
			// user's — e.g. an elevated or IT-assisted install launched the
			// server as the installer, whose network-share access differs
			// from the signed-in user's. openDenied (per-platform) decides
			// whether the file browser, which DOES run as the desktop user,
			// can still handle the path.
			if action, ok := openDenied(path); ok {
				return action, nil
			}
			return "", fmt.Errorf("%w: %q", errAccessDenied, path)
		}
		return "", fmt.Errorf("stat %q: %w", path, err)
	}

	if info.IsDir() {
		if err := openDir(path); err != nil {
			return "", err
		}
		return "opened", nil
	}
	if err := revealFile(path); err != nil {
		return "", err
	}
	return "revealed", nil
}

func runQuiet(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", cmd.Args[0], err, strings.TrimSpace(string(out)))
	}
	return nil
}
