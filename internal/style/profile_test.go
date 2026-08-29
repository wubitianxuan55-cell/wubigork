package style

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
)

// TestClampParams S1.5-B 护栏钳制透传（platform_handler AnalyzeStyle 生成点）：
// 零值 = 现状逐字节；超上限钳到 cap；边界不变；低于上限不抬高。
func TestClampParams(t *testing.T) {
	// 零值 clamp：0.3/1024 原样（现状逐字节）。
	got := clampParams(ai.ChatSimpleOptions{Temperature: 0.3, MaxTokens: 1024}, ParamClamp{})
	if got.Temperature != 0.3 || got.MaxTokens != 1024 {
		t.Errorf("零值 clamp 应原样: %+v", got)
	}
	// 钳制生效。
	got = clampParams(ai.ChatSimpleOptions{Temperature: 0.3, MaxTokens: 1024},
		ParamClamp{TemperatureMax: 0.2, MaxOutputTokens: 512})
	if got.Temperature != 0.2 || got.MaxTokens != 512 {
		t.Errorf("应钳制到上限: %+v", got)
	}
	// 边界 == cap 不变；低于 cap 不抬高。
	got = clampParams(ai.ChatSimpleOptions{Temperature: 0.3, MaxTokens: 1024},
		ParamClamp{TemperatureMax: 0.3, MaxOutputTokens: 1024})
	if got.Temperature != 0.3 || got.MaxTokens != 1024 {
		t.Errorf("边界应不变: %+v", got)
	}
	got = clampParams(ai.ChatSimpleOptions{Temperature: 0.1, MaxTokens: 128},
		ParamClamp{TemperatureMax: 0.9, MaxOutputTokens: 4096})
	if got.Temperature != 0.1 || got.MaxTokens != 128 {
		t.Errorf("低于上限不应抬高: %+v", got)
	}
}
