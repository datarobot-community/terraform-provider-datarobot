package provider

import "testing"

func TestNextLabelFromLatest(t *testing.T) {
	cases := []struct {
		latest string
		want   string
	}{
		// v-prefixed versions
		{"v1", "v2"},
		{"v9", "v10"},
		{"v10", "v11"},
		{"v99", "v100"},
		{"v0", "v1"},

		// bare numbers
		{"1", "v2"},
		{"9", "v10"},
		{"123", "v124"},

		// edge cases
		{"", "v1"},
		{"v", "v1"},
		{"abc", "v1"},
		{"version1", "v1"},
		{"0", "v1"},
	}

	for _, c := range cases {
		got := nextLabelFromLatest(c.latest)
		if got != c.want {
			t.Fatalf("nextLabelFromLatest(%q) = %q, want %q", c.latest, got, c.want)
		}
	}
}
