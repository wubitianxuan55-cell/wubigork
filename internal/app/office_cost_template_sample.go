package app

// office_cost_template_sample.go — 测算样表的取用单点（2026-11-25 用户拍板
// 「删除渲染器思路，提供样本给模型，再进行约束」）。双层来源：
//  1. 工作区覆盖（改版式换文件、不等发版）；
//  2. 内嵌种子（skill 包 go:embed，随程序走，开箱即得）。
//
// 出口两处共用：cost_library action=template 与绑定 GaeaCostLibraryExport
// kind=template（板块「导出 → 测算模版」）。

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// costEstimateSample 拿样表字节：工作区覆盖优先，内嵌种子兜底。返回
// （字节, 来源说明, error）。
func costEstimateSample() ([]byte, string, error) {
	if p, perr := wspath.ResolveRelWithin(gaeaCwd(), skill.CostEstimateSampleOverrideRel); perr == nil {
		if b, rerr := os.ReadFile(p); rerr == nil && len(b) > 0 {
			return b, "工作区样表 " + skill.CostEstimateSampleOverrideRel, nil
		}
	}
	b := skill.CostEstimateSampleXLSX()
	if len(b) == 0 {
		return nil, "", fmt.Errorf("内置测算样表缺失（打包缺件，请反馈）")
	}
	return b, "内置样表", nil
}

// writeCostEstimateSample 把样表落到 out（自动建父目录）。
func writeCostEstimateSample(out string) ([]byte, string, error) {
	data, source, err := costEstimateSample()
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return nil, "", fmt.Errorf("创建样表目录失败: %w", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return nil, "", fmt.Errorf("写出样表失败: %w", err)
	}
	return data, source, nil
}
