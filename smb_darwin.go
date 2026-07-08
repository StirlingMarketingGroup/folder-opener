package main

/*
#cgo LDFLAGS: -framework NetFS -framework CoreFoundation
#include <errno.h>
#include <stdlib.h>
#include <CoreFoundation/CoreFoundation.h>
#include <NetFS/NetFS.h>

// mountSMB mounts an smb:// share URL the way Finder does: under /Volumes,
// using Keychain credentials — without opening any Finder window. With
// allowUI false all NetFS UI is suppressed, so a failed mount comes back as
// an error code instead of the system "There was a problem connecting to
// the server" alert; pass allowUI true to let the standard authentication
// dialog appear. On success the mountpoint is returned in *mountpoint
// (caller frees).
static int mountSMB(const char *url, int allowUI, char **mountpoint) {
	CFStringRef urlString = CFStringCreateWithCString(NULL, url, kCFStringEncodingUTF8);
	if (urlString == NULL) {
		return EINVAL;
	}
	CFURLRef shareURL = CFURLCreateWithString(NULL, urlString, NULL);
	CFRelease(urlString);
	if (shareURL == NULL) {
		return EINVAL;
	}

	CFMutableDictionaryRef openOptions = CFDictionaryCreateMutable(NULL, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	if (openOptions != NULL && !allowUI) {
		CFDictionarySetValue(openOptions, kNAUIOptionKey, kNAUIOptionNoUI);
	}

	CFArrayRef mountpoints = NULL;
	int rc = NetFSMountURLSync(shareURL, NULL, NULL, NULL, openOptions, NULL, &mountpoints);
	CFRelease(shareURL);
	if (openOptions != NULL) {
		CFRelease(openOptions);
	}

	if (rc == 0) {
		if (mountpoints == NULL || CFArrayGetCount(mountpoints) == 0) {
			rc = EIO;
		} else {
			CFStringRef mp = CFArrayGetValueAtIndex(mountpoints, 0);
			CFIndex size = CFStringGetMaximumSizeForEncoding(CFStringGetLength(mp), kCFStringEncodingUTF8) + 1;
			*mountpoint = malloc(size);
			if (*mountpoint == NULL || !CFStringGetCString(mp, *mountpoint, size, kCFStringEncodingUTF8)) {
				free(*mountpoint);
				*mountpoint = NULL;
				rc = EIO;
			}
		}
	}
	if (mountpoints != NULL) {
		CFRelease(mountpoints);
	}
	return rc;
}
*/
import "C"

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// resolveSMB turns an smb:// URL into a local /Volumes path, mounting the
// share on demand so a single click works even when it isn't mounted yet.
func resolveSMB(raw string) (string, error) {
	u, share, rest, err := parseSMB(raw)
	if err != nil {
		return "", err
	}

	mountpoint, err := mountedShare(u.Hostname(), share)
	if err != nil {
		return "", err
	}
	if mountpoint == "" {
		if mountpoint, err = mountShare(u, share); err != nil {
			return "", err
		}
	}
	return filepath.Join(append([]string{mountpoint}, rest...)...), nil
}

func mountShare(u *url.URL, share string) (string, error) {
	shareURL := smbShareURL(u, share)

	// Mount silently first (Keychain or saved credentials), so failures come
	// back as clean errors for the caller instead of the system "There was a
	// problem connecting to the server" alert. Only when the mount fails for
	// lack of credentials retry with UI allowed, letting the standard macOS
	// authentication dialog appear (and save to the Keychain).
	mountpoint, rc := mountOnce(shareURL, false)
	if needsAuthUI(rc) {
		mountpoint, rc = mountOnce(shareURL, true)
	}

	switch rc {
	case 0:
		return mountpoint, nil
	case C.EEXIST:
		// Lost a mount race; the share is there now, find its mountpoint.
		mountpoint, err := mountedShare(u.Hostname(), share)
		if err == nil && mountpoint != "" {
			return mountpoint, nil
		}
		return "", fmt.Errorf("mount %s: already mounted but mountpoint not found", shareURL)
	case C.ENOENT:
		return "", fmt.Errorf("%w: no such share: %s", errNotFound, shareURL)
	case C.ECANCELED:
		return "", fmt.Errorf("mount %s: authentication canceled", shareURL)
	default:
		if rc > 0 {
			return "", fmt.Errorf("mount %s: %w", shareURL, syscall.Errno(rc))
		}
		// Negative codes are NetFS-specific (auth/UI errors from NetFS.h).
		return "", fmt.Errorf("mount %s: NetFS error %d", shareURL, int(rc))
	}
}

func mountOnce(shareURL string, allowUI bool) (mountpoint string, rc C.int) {
	urlC := C.CString(shareURL)
	defer C.free(unsafe.Pointer(urlC))

	allowUIC := C.int(0)
	if allowUI {
		allowUIC = 1
	}
	var mountpointC *C.char
	rc = C.mountSMB(urlC, allowUIC, &mountpointC)
	if mountpointC != nil {
		defer C.free(unsafe.Pointer(mountpointC))
	}
	return C.GoString(mountpointC), rc
}

// needsAuthUI reports whether a silent mount failed specifically because
// credentials are missing or rejected — the cases the auth dialog can fix.
func needsAuthUI(rc C.int) bool {
	switch rc {
	case C.EAUTH, C.ENEEDAUTH, C.EACCES, C.EPERM:
		return true
	}
	return false
}

// mountedShare scans the mount table for an existing smbfs mount of
// //host/share and returns its mountpoint, or "" when it isn't mounted.
func mountedShare(host, share string) (string, error) {
	n, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	if err != nil {
		return "", fmt.Errorf("getfsstat: %w", err)
	}
	stats := make([]unix.Statfs_t, n)
	if _, err := unix.Getfsstat(stats, unix.MNT_NOWAIT); err != nil {
		return "", fmt.Errorf("getfsstat: %w", err)
	}

	for _, stat := range stats {
		if unix.ByteSliceToString(stat.Fstypename[:]) != "smbfs" {
			continue
		}
		mountHost, mountShare, ok := splitMntFromName(unix.ByteSliceToString(stat.Mntfromname[:]))
		if ok && strings.EqualFold(mountHost, host) && strings.EqualFold(mountShare, share) {
			return unix.ByteSliceToString(stat.Mntonname[:]), nil
		}
	}
	return "", nil
}

// splitMntFromName extracts host and share from an smbfs mount source like
// "//user@host/share" or "//GUEST:@host:445/share"; both parts may be
// percent-encoded.
func splitMntFromName(from string) (host, share string, ok bool) {
	from, ok = strings.CutPrefix(from, "//")
	if !ok {
		return "", "", false
	}
	authority, share, ok := strings.Cut(from, "/")
	if !ok || share == "" {
		return "", "", false
	}
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		authority = authority[at+1:]
	}
	if colon := strings.LastIndex(authority, ":"); colon >= 0 {
		authority = authority[:colon]
	}
	return pathUnescaped(authority), pathUnescaped(share), true
}

func pathUnescaped(s string) string {
	if unescaped, err := url.PathUnescape(s); err == nil {
		return unescaped
	}
	return s
}
