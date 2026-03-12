package main

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bps      uint64
		expected string
	}{
		{500, "500 B/s"},
		{1024, "1024 B/s"},
		{1500, "1.5 KB/s"},
		{1024 * 1024, "1024.0 KB/s"},
		{1024 * 1024 * 2, "2.0 MB/s"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bps)
		if result != tt.expected {
			t.Errorf("formatBytes(%d): expected %s, got %s", tt.bps, tt.expected, result)
		}
	}
}

func TestGetTopBandwidthHogs(t *testing.T) {
	// This function depends on OS state, but we can at least ensure it doesn't crash
	// and returns a non-empty string.
	res := GetTopBandwidthHogs()
	if res == "" {
		t.Error("GetTopBandwidthHogs returned an empty string")
	}
}
