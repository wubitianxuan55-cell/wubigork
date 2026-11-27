package app

// office_cost_template_sample.go — 测算样表的取用单点（2026-11-25 用户拍板
// 「删除渲染器思路，提供样本给模型，再进行约束」）。双层来源：
//  1. 工作区覆盖（改版式换文件、不等发版）；
//  2. 内嵌种子（skill 包 go:embed，随程序走，开箱即得）。
//
// 出口两处共用：cost_library action=template 与绑定 GaeaCostLibraryExport
// kind=template（板块「导出 → 测算模版」）。
//
// 2026-11-27 用户令「给成本测算表增加一个使用说明书 md，一个和他绑定」：样表
// 落盘时，旁边同时落一份《使用说明书》（<样表名>-使用说明.md）——样表的
// 「2 块 × 4 明细」只是示例槽位、加行配方、两道门与入库红线都在里面。说明书
// 也可被工作区同名文件覆盖。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// costEstimateManual 拿使用说明书字节：工作区覆盖优先，内嵌兜底。
func costEstimateManual() []byte {
	if p, perr := wspath.ResolveRelWithin(gaeaCwd(), skill.CostEstimateManualOverrideRel); perr == nil {
		if b, rerr := os.ReadFile(p); rerr == nil && len(b) > 0 {
			return b
		}
	}
	return skill.CostEstimateManual()
}

// writeCostEstimateSample 把样表落到 out（自动建父目录），并把《使用说明书》
// 绑着落在旁边（<样表名>-使用说明.md）。返回（样表来源说明, 说明书工作区相对
// 路径, error）；说明书路径落在工作区外时给绝对路径。
func writeCostEstimateSample(out string) (string, string, error) {
	data, source, err := costEstimateSample()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", "", fmt.Errorf("创建样表目录失败: %w", err)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", "", fmt.Errorf("写出样表失败: %w", err)
	}
	manual := strings.TrimSuffix(out, filepath.Ext(out)) + "-使用说明.md"
	if err := os.WriteFile(manual, costEstimateManual(), 0o644); err != nil {
		// 说明书是样表的配套：写不出来不拦样表，只是回执里没有它。
		return source, "", nil
	}
	return source, workspaceRelative(gaeaCwd(), manual), nil
}

// workspaceRelative 能表示成工作区相对路径就给相对（正斜杠），否则给原路径。
func workspaceRelative(root, p string) string {
	if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return p
}
