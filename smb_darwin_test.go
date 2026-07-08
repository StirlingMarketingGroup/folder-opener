package main

import "testing"

func TestSplitMntFromName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from      string
		wantHost  string
		wantShare string
		wantOK    bool
	}{
		{"//brian@storage/Art", "storage", "Art", true},
		{"//storage/Art", "storage", "Art", true},
		{"//GUEST:@storage/Signature%20Coins", "storage", "Signature Coins", true},
		{"//DOMAIN;brian@storage:445/Art", "storage", "Art", true},
		{"/dev/disk3s1", "", "", false},
		{"//storage", "", "", false},
	}
	for _, test := range tests {
		host, share, ok := splitMntFromName(test.from)
		if host != test.wantHost || share != test.wantShare || ok != test.wantOK {
			t.Errorf("splitMntFromName(%q) = (%q, %q, %v), want (%q, %q, %v)",
				test.from, host, share, ok, test.wantHost, test.wantShare, test.wantOK)
		}
	}
}
