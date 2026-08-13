package app

import (
	"os"
	"path/filepath"
	"testing"
)

// 一次性校验：用真实 CLI 输出（.tmp/herdsman-models-list.json）验证解析器。
func TestParseHerdsmanModelList_LiveFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", ".tmp", "herdsman-models-list.json"))
	if err != nil {
		t.Skip("缺少实时 fixture")
	}
	models, err := parseHerdsmanModelList(data)
	if err != nil {
		t.Fatalf("解析实时输出失败: %v", err)
	}
	if len(models) < 80 {
		t.Fatalf("实时目录应 ≥80 条，got %d", len(models))
	}
	t.Logf("解析 %d 个模型，首个: %s (%s)", len(models), models[0].Name, models[0].Type)
}
