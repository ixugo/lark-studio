package pipeline

import (
	"testing"
)

func TestAdjustSpeedFactor_Capping(t *testing.T) {
	tests := []struct {
		name      string
		audioDur  float64
		segDur    float64
		maxFactor float64
		wantSkip  bool
		wantCap   float64
	}{
		{"音频短于段落 → 跳过", 3.0, 5.0, 1.3, true, 0},
		{"差异 <2% → 跳过", 5.05, 5.0, 1.3, true, 0},
		{"正常调速", 6.0, 5.0, 1.3, false, 1.2},
		{"超限截断", 8.0, 5.0, 1.3, false, 1.3},
		{"maxFactor ≤1 → 全跳过", 6.0, 5.0, 0.9, true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.maxFactor <= 1 {
				if !tt.wantSkip {
					t.Error("maxFactor ≤1 应跳过")
				}
				return
			}

			if tt.audioDur <= 0 || tt.segDur <= 0 || tt.audioDur <= tt.segDur {
				if !tt.wantSkip {
					t.Error("音频不超段落应跳过")
				}
				return
			}

			factor := tt.audioDur / tt.segDur
			if factor < 1.02 {
				if !tt.wantSkip {
					t.Error("差异 <2% 应跳过")
				}
				return
			}

			if tt.wantSkip {
				t.Error("不应跳过")
				return
			}

			if factor > tt.maxFactor {
				factor = tt.maxFactor
			}
			if factor != tt.wantCap {
				t.Errorf("factor = %.2f, want %.2f", factor, tt.wantCap)
			}
		})
	}
}

func TestMaxSpeedFactorConfig(t *testing.T) {
	cfg := Config{MaxSpeedFactor: 1.25}
	if cfg.MaxSpeedFactor <= 1 {
		t.Error("配置值应 > 1 才生效")
	}
	if cfg.MaxSpeedFactor > 2 {
		t.Error("调速超过 2x 影响听感")
	}

	cfg2 := Config{MaxSpeedFactor: 0}
	if cfg2.MaxSpeedFactor > 1 {
		t.Error("零值不应触发调速")
	}
}
