package main

import (
	"errors"
	"path/filepath"
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
