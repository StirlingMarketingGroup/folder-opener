package main

import "path/filepath"

// resolveSMB translates an smb:// URL to the equivalent UNC path, which
// Windows resolves natively (connecting and authenticating as needed).
func resolveSMB(raw string) (string, error) {
	u, share, rest, err := parseSMB(raw)
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{`\\` + u.Hostname(), share}, rest...)...), nil
}
