package modelengine

import (
	"math"
	"testing"
)

// TestEstimateCostCNY 表驱动覆盖 EstimateCostCNY 的四种计费口径：
// 本地引擎不计费、未知模型不计费、USD 计价按汇率折算 CNY、CNY 计价直用；
// 另覆盖非法汇率（0）守卫——回退默认 7.2，绝不用 0 汇率把 USD 费用抹成 0。
// 数值断言沿用 stats_test.go 的 ±0.00001 惯例；汇率沿用 SetUsdCnyRate(7.2) 先例。
func TestEstimateCostCNY(t *testing.T) {
	tests := []struct {
		name     string
		engineID string
		model    string
		inTok    int64
		outTok   int64
		usdCny   float64
		want     float64
	}{
		{
			name:     "本地引擎不计费",
			engineID: "herdsman",
			model:    "qwen3-8b",
			inTok:    1000,
			outTok:   500,
			usdCny:   7.2,
			want:     0,
		},
		{
			name:     "未知模型不计费",
			engineID: "xai",
			model:    "no-such-model-x",
			inTok:    1000,
			outTok:   500,
			usdCny:   7.2,
			want:     0,
		},
		{
			// grok-4.20：2 USD/百万 input + 6 USD/百万 output
			// = (2*1000 + 6*500)/1e6 = 0.005 USD × 7.2 = 0.036 CNY
			name:     "USD 计价按汇率折算 CNY",
			engineID: "xai",
			model:    "grok-4.20",
			inTok:    1000,
			outTok:   500,
			usdCny:   7.2,
			want:     0.036,
		},
		{
			// deepseek-chat：1 CNY/百万 input + 2 CNY/百万 output
			// = (1*1000 + 2*500)/1e6 = 0.002 CNY（直用，不乘汇率）
			name:     "CNY 计价直用",
			engineID: "deepseek",
			model:    "deepseek-chat",
			inTok:    1000,
			outTok:   500,
			usdCny:   7.2,
			want:     0.002,
		},
		{
			// 非法汇率（0）回退默认 7.2：结果与显式 7.2 一致，不为 0。
			name:     "USD 计价非法汇率回退默认",
			engineID: "xai",
			model:    "grok-4.20",
			inTok:    1000,
			outTok:   500,
			usdCny:   0,
			want:     0.036,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCostCNY(tt.engineID, tt.model, tt.inTok, tt.outTok, tt.usdCny)
			if math.Abs(got-tt.want) > 0.00001 {
				t.Fatalf("EstimateCostCNY(%q, %q, %d, %d, %.4f) = %.6f, want %.6f",
					tt.engineID, tt.model, tt.inTok, tt.outTok, tt.usdCny, got, tt.want)
			}
		})
	}
}
