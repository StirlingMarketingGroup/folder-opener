package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveSMB turns an smb:// URL into a path inside the user's gvfs FUSE
// mount, mounting the share on demand via `gio mount`.
func resolveSMB(raw string) (string, error) {
	u, share, rest, err := parseSMB(raw)
	if err != nil {
		return "", err
	}
	host := u.Hostname()

	dir, ok := gvfsShareDir(host, share)
	if !ok {
		// gio also exits non-zero when the share is already mounted, so only
		// fail if the share still isn't visible afterwards.
		mountErr := runQuiet(exec.Command("gio", "mount", smbShareURL(u, share)))
		if dir, ok = gvfsShareDir(host, share); !ok {
			if mountErr != nil {
				return "", fmt.Errorf("mount %s: %w", smbShareURL(u, share), mountErr)
			}
			return "", fmt.Errorf("mount %s: mounted but gvfs directory not found", smbShareURL(u, share))
		}
	}
	return filepath.Join(append([]string{dir}, rest...)...), nil
}

// gvfsShareDir finds the gvfs FUSE directory for an already-mounted smb
// share, e.g. /run/user/1000/gvfs/smb-share:server=host,share=name.
func gvfsShareDir(host, share string) (string, bool) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = fmt.Sprintf("/run/user/%d", os.Getuid())
	}
	gvfs := filepath.Join(runtimeDir, "gvfs")
	entries, err := os.ReadDir(gvfs)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		fields, found := strings.CutPrefix(entry.Name(), "smb-share:")
		if !found {
			continue
		}
		var entryHost, entryShare string
		for field := range strings.SplitSeq(fields, ",") {
			key, value, _ := strings.Cut(field, "=")
			switch key {
			case "server":
				entryHost = value
			case "share":
				entryShare = value
			}
		}
		if strings.EqualFold(entryHost, host) && strings.EqualFold(entryShare, share) {
			return filepath.Join(gvfs, entry.Name()), true
		}
	}
	return "", false
}
