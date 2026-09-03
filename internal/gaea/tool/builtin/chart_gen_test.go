package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseChartInputValidation(t *testing.T) {
	valid := func(s string) json.RawMessage { return json.RawMessage(s) }

	cases := []struct {
		name    string
		args    string
		wantErr string // "" = 应通过
	}{
		{"labels 缺失", `{"values":[1,2]}`, "labels 不能为空"},
		{"单系列长度不一致", `{"labels":["a","b"],"values":[1]}`, "长度不一致"},
		{"values 缺失", `{"labels":["a","b"]}`, "values 不能为空"},
		{"未知类型", `{"labels":["a"],"values":[1],"chart_type":"radar"}`, "未知 chart_type"},
		{"grouped_bar 缺 series", `{"labels":["a","b"],"values":[1,2],"chart_type":"grouped_bar"}`, "需要 series"},
		{"stacked_bar 缺 series", `{"labels":["a","b"],"values":[1,2],"chart_type":"stacked_bar"}`, "需要 series"},
		{"pie 不支持 series", `{"labels":["a","b"],"series":[{"values":[1,2]}],"chart_type":"pie"}`, "不支持 series"},
		{"hbar 不支持 series", `{"labels":["a","b"],"series":[{"values":[1,2]}],"chart_type":"hbar"}`, "不支持 series"},
		{"series 长度不一致", `{"labels":["a","b"],"chart_type":"grouped_bar","series":[{"name":"X","values":[1]},{"name":"Y","values":[1,2]}]}`, "series[0]"},
		{"单系列 bar 合法", `{"labels":["a","b"],"values":[1,2]}`, ""},
		{"多系列 grouped_bar 合法", `{"labels":["a","b"],"chart_type":"grouped_bar","series":[{"name":"X","values":[1,2]},{"name":"Y","values":[3,4]}]}`, ""},
		{"多系列 stacked_bar 合法", `{"labels":["a","b"],"chart_type":"stacked_bar","series":[{"values":[1,2]}]}`, ""},
		{"bar+series 升级并列", `{"labels":["a","b"],"chart_type":"bar","series":[{"name":"X","values":[1,2]},{"name":"Y","values":[3,4]}]}`, ""},
		{"line+series 升级多折线", `{"labels":["a","b"],"chart_type":"line","series":[{"name":"X","values":[1,2]}]}`, ""},
		{"缺省类型为 bar", `{"labels":["a"],"values":[1]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := parseChartInput(valid(tc.args))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("应通过却报错: %v", err)
				}
				if p.ChartType == "" {
					t.Fatalf("chart_type 应有缺省值")
				}
				return
			}
			if err == nil {
				t.Fatalf("应报错却通过了: %+v", p)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("错误信息应含 %q，实际: %v", tc.wantErr, err)
			}
		})
	}
}

// 冒烟：真实渲染链（Python+matplotlib）跑一张多系列分组柱状图。
// 环境缺 Python/matplotlib 时跳过（与 diagram_gen 冒烟同口径）。
func TestChartGenMultiSeriesSmoke(t *testing.T) {
	if _, err := lookPathFirst([]string{"python", "python3"}); err != nil {
		t.Skip("python 不可用，跳过渲染冒烟")
	}
	out := filepath.Join(t.TempDir(), "grouped.png")
	args := map[string]interface{}{
		"title":      "分项单价对比",
		"chart_type": "grouped_bar",
		"labels":     []string{"人工费", "材料费", "机械费"},
		"series": []map[string]interface{}{
			{"name": "2025年", "values": []float64{120, 300, 80}},
			{"name": "2026年", "values": []float64{135, 340, 92}},
		},
		"output": out,
	}
	raw, _ := json.Marshal(args)
	res, err := (chartGen{}).Execute(context.Background(), raw)
	if err != nil {
		if strings.Contains(err.Error(), "matplotlib not installed") {
			t.Skip("matplotlib 不可用，跳过渲染冒烟")
		}
		t.Fatalf("Execute: %v", err)
	}
	var r struct {
		OK  bool   `json:"ok"`
		Msg string `json:"message"`
	}
	if err := json.Unmarshal([]byte(res), &r); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if !r.OK || !strings.Contains(r.Msg, "系列: 2") {
		t.Fatalf("结果应含 系列: 2，实际: %s", res)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("输出为空文件")
	}
}
