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
	errNotFound = errors.New("path does not exist")
	errBadPath  = errors.New("bad path")
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
