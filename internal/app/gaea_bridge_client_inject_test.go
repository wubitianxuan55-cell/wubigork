package app

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// TestBridgeClientInjectedWithoutGaeaInit 回归：未进入办公板块（GaeaInit
// 未执行）时，configureClient 也必须注入桥接 client，否则办公侧的 AI 解析
// 会报 "bridge: ai.LLMClient 未注入"。
// （本文件原名 gaea_cost_marshal_test.go；旧成本库条目的 Wails 参数契约用例与
// 成本导入 PDF 文本判定用例随旧成本库退役拆除，2026-11-26。）
func TestBridgeClientInjectedWithoutGaeaInit(t *testing.T) {
	a := &App{core: &core{}}
	a.client = ai.NewClient(config.Load())
	a.configureClient()

	p, err := provider.NewLLM("", provider.Config{Name: "cost-import-ai", Model: "", Engine: ""})
	if err != nil {
		t.Fatalf("桥接 provider 创建失败（client 未注入）: %v", err)
	}
	if p == nil {
		t.Fatal("provider nil")
	}
}
