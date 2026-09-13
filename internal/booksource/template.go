package booksource

import (
	_ "embed"
	"os"
	"path/filepath"
)

// rule-template.json 出厂唯一内置规则文件（模板，disabled=true 不会被聚合
// 搜索拾取）；用户照抄自编目标站点规则。**不出厂任何真实站点规则集**
// （规格 §6 拍板池第 1 条：上游 AGPL 数据不随 gaea 分发）。
//
//go:embed rules/rule-template.json
var TemplateJSON []byte

// EnsureTemplate 把模板落到规则目录（已存在则不动）。t2 app 接线时首用调用。
func EnsureTemplate(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "rule-template.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, TemplateJSON, 0o644)
}
