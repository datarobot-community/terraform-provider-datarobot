package provider

import (
	"testing"
)

func TestParseMemoryBytes(t *testing.T) {
	cases := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{input: "0", want: 0},
		{input: "1024", want: 1024},
		{input: "536870912", want: 536870912},

		// SI (1000-based)
		{input: "1B", want: 1},
		{input: "1KB", want: 1000},
		{input: "1MB", want: 1_000_000},
		{input: "1GB", want: 1_000_000_000},
		{input: "4GB", want: 4_000_000_000},
		{input: "512MB", want: 512_000_000},
		{input: "15GB", want: 15_000_000_000},

		// IEC (1024-based)
		{input: "1Ki", want: 1024},
		{input: "1KiB", want: 1024},
		{input: "1Mi", want: 1 << 20},
		{input: "1MiB", want: 1 << 20},
		{input: "1Gi", want: 1 << 30},
		{input: "1GiB", want: 1 << 30},
		{input: "4096Mi", want: 4096 * (1 << 20)},
		{input: "4096MiB", want: 4096 * (1 << 20)},

		// Short IEC aliases
		{input: "1K", want: 1024},
		{input: "1M", want: 1 << 20},
		{input: "1G", want: 1 << 30},

		// Whitespace tolerance
		{input: "  4GB  ", want: 4_000_000_000},
		{input: "4 GB", want: 4_000_000_000},

		// Errors
		{input: "", wantErr: true},
		{input: "abc", wantErr: true},
		{input: "4XB", wantErr: true},
	}
	for _, tc := range cases {
		got, err := parseMemoryBytes(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseMemoryBytes(%q) expected error, got %d", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseMemoryBytes(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseMemoryBytes(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
