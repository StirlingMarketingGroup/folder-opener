package main

import (
	"errors"
	"slices"
	"testing"
)

func TestIsSMBURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"smb://storage/Art", true},
		{"SMB://storage/Art", true},
		{`\\storage\Art`, false},
		{"/Volumes/Art", false},
		{"smb:/storage/Art", false},
		{"", false},
	}
	for _, test := range tests {
		if got := isSMBURL(test.path); got != test.want {
			t.Errorf("isSMBURL(%q) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestParseSMB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		wantHost  string
		wantShare string
		wantRest  []string
	}{
		{"plain", "smb://storage/Art/12345", "storage", "Art", []string{"12345"}},
		{"spaces", "smb://storage/Signature Coins/2026/Order 99", "storage", "Signature Coins", []string{"2026", "Order 99"}},
		{"percent encoded", "smb://storage/Signature%20Coins/2026", "storage", "Signature Coins", []string{"2026"}},
		{"share only", "smb://storage/Art", "storage", "Art", nil},
		{"trailing slash", "smb://storage/Art/", "storage", "Art", nil},
		{"user and port", "smb://brian@storage:445/Art/x", "storage", "Art", []string{"x"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			u, share, rest, err := parseSMB(test.raw)
			if err != nil {
				t.Fatalf("parseSMB(%q) error = %v", test.raw, err)
			}
			if u.Hostname() != test.wantHost {
				t.Errorf("host = %q, want %q", u.Hostname(), test.wantHost)
			}
			if share != test.wantShare {
				t.Errorf("share = %q, want %q", share, test.wantShare)
			}
			if !slices.Equal(rest, test.wantRest) {
				t.Errorf("rest = %q, want %q", rest, test.wantRest)
			}
		})
	}
}

// Invalid smb URLs must fail in parseSMB as errBadPath, well before any
// platform mount logic could run (tests must never mount a share).
func TestParseSMBInvalid(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"smb://", "smb:///Art", "smb://storage", "smb://storage/"} {
		if _, _, _, err := parseSMB(raw); !errors.Is(err, errBadPath) {
			t.Errorf("parseSMB(%q) error = %v, want errBadPath", raw, err)
		}
	}
	for _, raw := range []string{"smb://storage", "smb://storage/"} {
		if _, err := openPath(raw); !errors.Is(err, errBadPath) {
			t.Errorf("openPath(%q) error = %v, want errBadPath", raw, err)
		}
	}
}
