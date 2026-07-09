package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Only invalid paths are exercised here: openPath on a real directory would
// pop a file browser window on the machine running the tests.
func TestOpenPathInvalid(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "does-not-exist")

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{"empty", "", errBadPath},
		{"whitespace", "   ", errBadPath},
		{"relative", "some/relative/path", errBadPath},
		{"missing", missing, errNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := openPath(test.path)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("openPath(%q) error = %v, want %v", test.path, err, test.wantErr)
			}
		})
	}
}

// A permission-denied stat must surface as errAccessDenied, not a raw stat
// error. Unix-only: an unsearchable parent makes stat fail with EACCES, and
// openDenied is a no-op off Windows, so no file browser is invoked. Root
// ignores permission bits, so skip when the setup doesn't actually deny.
func TestOpenPathAccessDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("cannot force a permission-denied stat on Windows without ACL setup")
	}
	t.Parallel()

	parent := filepath.Join(t.TempDir(), "locked")
	target := filepath.Join(parent, "child")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	if _, err := os.Stat(target); !errors.Is(err, os.ErrPermission) {
		t.Skipf("stat under a 0o000 parent did not fail with permission denied (running as root?): %v", err)
	}

	if _, err := openPath(target); !errors.Is(err, errAccessDenied) {
		t.Fatalf("openPath(%q) error = %v, want %v", target, err, errAccessDenied)
	}
}
