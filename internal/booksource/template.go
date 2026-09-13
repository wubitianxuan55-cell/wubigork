package booksource

import (
	_ "embed"
	"os"
	"path/filepath"
)

// 出厂内置规则模板（disabled=true 不会被拾取）；用户照抄自编。
// **不出厂任何真实站点规则集**（书源规格 §6 拍板池第 1 条：上游 AGPL 数据
// 不随 gaea 分发；引擎规则为自编功能数据，owllook 规格书源搜索 §6）。
//
//go:embed rules/rule-template.json
var TemplateJSON []byte

//go:embed rules/websearch-engines.json
var EnginesTemplateJSON []byte

// EnsureTemplate 把模板落到规则目录（已存在则不动，幂等）。t2 app 接线时首用调用。
func EnsureTemplate(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for name, raw := range map[string][]byte{
		"rule-template.json":     TemplateJSON,
		"websearch-engines.json": EnginesTemplateJSON,
	} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}
