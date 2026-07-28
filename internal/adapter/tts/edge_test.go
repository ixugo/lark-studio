package tts

import "testing"

// TestEdgeRate 验证语速倍率转换为 edge-tts 百分比。
func TestEdgeRate(t *testing.T) {
	tests := map[float64]string{
		0:    "+0%",
		0.8:  "-20%",
		1:    "+0%",
		1.25: "+25%",
	}
	for speed, want := range tests {
		if got := edgeRate(speed); got != want {
			t.Fatalf("edgeRate(%v) = %q，期望 %q", speed, got, want)
		}
	}
}
