package update

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v0.0.127", "0.1.0", -1}, {"1.1000.0", "1.999.999", 1},
		{"999999999999999999999999.0.0", "2.0.0", 1}, {"v1.2.3", "01.002.3", 0},
	}
	for _, tt := range tests {
		got, err := CompareVersions(tt.a, tt.b)
		if err != nil || got != tt.want {
			t.Errorf("CompareVersions(%q,%q) = %d,%v", tt.a, tt.b, got, err)
		}
	}
	for _, v := range []string{"", "dev", "1.2", "1.2.3.4", "1.-2.3", "+1.2.3", "1.2.3-beta", " 1.2.3", "vv1.2.3", "1.2.３"} {
		if _, err := CompareVersions(v, "1.0.0"); err == nil {
			t.Errorf("accepted invalid version %q", v)
		}
	}
}

func TestShouldPrompt(t *testing.T) {
	for _, tt := range []struct {
		current, latest, ignored string
		manual, want             bool
	}{
		{"0.0.1", "v0.1.0", "", false, true}, {"0.0.1", "v0.1.0", "0.1.0", false, false},
		{"0.0.1", "v0.1.0", "0.1.0", true, true}, {"0.1.0", "0.1.0", "", true, false},
		{"0.0.1", "v0.1.0", "0.2.0", false, false}, {"0.0.1", "v0.1.0", "0.2.0", true, true},
		{"2.0.0", "1.0.0", "", false, false},
	} {
		got, err := ShouldPrompt(tt.current, tt.latest, tt.ignored, tt.manual)
		if err != nil || got != tt.want {
			t.Errorf("ShouldPrompt(%+v)=%v,%v", tt, got, err)
		}
	}
}
