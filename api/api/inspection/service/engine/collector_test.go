package engine

import "testing"

func TestParseMemoryToKB(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"", 0},
		{"0", 0},
		{"16000000kB", 16000000},
		{"16000000KB", 16000000},
		{"16000000kb", 16000000},
		{"1GB", 1 * 1024 * 1024},
		{"2 GB", 2 * 1024 * 1024},
		{"2048MB", 2048 * 1024},
		{" 2048 MB ", 2048 * 1024},
		{"512K", 512},
		{"1024", 1024}, // bare number, assumed kB
		{"16.5GB", int64(16.5 * 1024 * 1024)},
	}
	for _, tc := range cases {
		got := parseMemoryToKB(tc.input)
		if got != tc.expected {
			t.Errorf("parseMemoryToKB(%q): expected %d, got %d", tc.input, tc.expected, got)
		}
	}
}
