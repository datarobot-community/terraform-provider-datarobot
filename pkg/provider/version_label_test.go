package provider

import "testing"

func TestNextLabelFromLatest(t *testing.T) {
	cases := []struct {
		latest string
		want   string
	}{
		{"v1", "v2"},
		{"v9", "v10"},
		{"v10", "v11"},
		{"9", "v10"},
		{"", "v1"},
		{"alpha", "v1"},
		{"v", "v1"},
	}

	for _, c := range cases {
		got := nextLabelFromLatest(c.latest)
		if got != c.want {
			t.Fatalf("nextLabelFromLatest(%q) = %q, want %q", c.latest, got, c.want)
		}
	}
}
