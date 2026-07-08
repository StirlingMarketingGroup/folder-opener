package main

import (
	"fmt"
	"net/url"
	"strings"
)

// isSMBURL reports whether the request path is an smb:// share URL rather
// than a local filesystem path.
func isSMBURL(path string) bool {
	return len(path) >= 6 && strings.EqualFold(path[:6], "smb://")
}

// parseSMB splits an smb:// URL into the parsed URL, the share name (first
// path segment), and the path segments below the share. Percent-escapes are
// decoded; literal spaces are accepted as-is.
func parseSMB(raw string) (u *url.URL, share string, rest []string, err error) {
	u, err = url.Parse(raw)
	if err != nil {
		return nil, "", nil, fmt.Errorf("%w: %v", errBadPath, err)
	}
	if u.Hostname() == "" {
		return nil, "", nil, fmt.Errorf("%w: smb URL missing host: %q", errBadPath, raw)
	}
	for segment := range strings.SplitSeq(u.Path, "/") {
		if segment != "" {
			rest = append(rest, segment)
		}
	}
	if len(rest) == 0 {
		return nil, "", nil, fmt.Errorf("%w: smb URL missing share name: %q", errBadPath, raw)
	}
	return u, rest[0], rest[1:], nil
}

// smbShareURL builds a properly escaped smb:// URL for just the share root,
// preserving any user info and port from the original URL.
func smbShareURL(u *url.URL, share string) string {
	root := url.URL{Scheme: "smb", User: u.User, Host: u.Host, Path: "/" + share}
	return root.String()
}
