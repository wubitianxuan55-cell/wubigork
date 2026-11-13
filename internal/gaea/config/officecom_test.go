package config

import "testing"

// [office_com] 段（dsh-office COM 实时协同档波3）：Default() 播上游 DEFAULTS
// 口径 + OfficeComNormalized 归一（nil 安全、非法 suite 回 auto、宽度缺省
// 1100 / 越界夹 [600,2000]、布尔原样透传）。
func TestOfficeComNormalized(t *testing.T) {
	if d := Default().OfficeComNormalized(); d.Suite != "auto" || d.Silent || !d.Follow || !d.Typing || d.RenderWidth != 1100 {
		t.Fatalf("Default() 应播上游 DEFAULTS 口径: %+v", d)
	}
	var nilCfg *Config
	if n := nilCfg.OfficeComNormalized(); n.Suite != "auto" || !n.Follow || !n.Typing || n.RenderWidth != 1100 {
		t.Fatalf("nil Config 应安全回默认: %+v", n)
	}
	// 合法值透传（suite 大小写不敏感；布尔显式 false 保留）。
	c := &Config{OfficeCom: OfficeComConfig{Suite: "Office", Silent: true, Follow: false, Typing: false, RenderWidth: 800}}
	if n := c.OfficeComNormalized(); n.Suite != "office" || !n.Silent || n.Follow || n.Typing || n.RenderWidth != 800 {
		t.Fatalf("合法值应透传: %+v", n)
	}
	// 非法 suite 回 auto；越界宽度夹边（<600 → 600）；裸 Config 布尔零值按
	// 显式关闭透传（Default() 播种才回 true）。
	c = &Config{OfficeCom: OfficeComConfig{Suite: "onlyoffice", RenderWidth: 42}}
	if n := c.OfficeComNormalized(); n.Suite != "auto" || n.Follow || n.Typing || n.RenderWidth != 600 {
		t.Fatalf("非法值应归一: %+v", n)
	}
	// 越界宽度夹边（>2000 → 2000）；suite 空 = auto。
	c = &Config{OfficeCom: OfficeComConfig{RenderWidth: 9999}}
	if n := c.OfficeComNormalized(); n.Suite != "auto" || n.RenderWidth != 2000 {
		t.Fatalf("越界宽度应夹回 2000: %+v", n)
	}
	// 零值段：宽度缺省 1100、suite auto；布尔零值按显式关闭透传（不强行回填，
	// 生产路径恒经 Default() 播种 follow/typing=true）。
	if n := (&Config{}).OfficeComNormalized(); n.Suite != "auto" || n.Silent || n.Follow || n.Typing || n.RenderWidth != 1100 {
		t.Fatalf("零值段宽度应缺省 1100: %+v", n)
	}
}
