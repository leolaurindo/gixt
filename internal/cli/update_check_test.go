package cli

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "1.2.3", 0},
		{"1.2.4", "1.2.3", 1},
		{"1.2", "1.2.1", -1},
		{"1.10.0", "1.2.9", 1},
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Fatalf("compareVersions(%q,%q)=%d want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTrimVersion(t *testing.T) {
	if got := trimVersion("v1.2.3"); got != "1.2.3" {
		t.Fatalf("trimVersion: got %q", got)
	}
	if got := trimVersion(" 1.0 "); got != "1.0" {
		t.Fatalf("trimVersion: got %q", got)
	}
}
