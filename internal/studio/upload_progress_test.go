package studio

import "testing"

func TestUploadProgressFromLabel(t *testing.T) {
	tests := []struct {
		label      string
		complete   bool
		inProgress bool
	}{
		{"Upload complete", true, false},
		{"Upload complete … Processing will begin shortly", true, false},
		{"  Upload complete  ", true, false},
		{"Processing will begin shortly", true, false},
		{"Processing up to HD", true, false},
		{"Uploading 45%", false, true},
		{"Uploading… 100%", false, true},
		{"Processing HD version", false, false},
		{"", false, false},
	}
	for _, tt := range tests {
		complete, inProgress := uploadProgressFromLabel(tt.label)
		if complete != tt.complete || inProgress != tt.inProgress {
			t.Fatalf("uploadProgressFromLabel(%q) = (%v, %v), want (%v, %v)",
				tt.label, complete, inProgress, tt.complete, tt.inProgress)
		}
	}
}
