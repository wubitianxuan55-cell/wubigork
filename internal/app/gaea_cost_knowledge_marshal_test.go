package app

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// TestCostKnowledgeWritePayloadsWithoutTimestamps 固化 Wails 绑定层参数契约：
// 前端新建/导入成本与知识时不得发送 updatedAt/createdAt 空串——
// Go 端 time.Time 用 encoding/json 解 "" 会报错，导致整次调用在绑定层失败。
func TestCostKnowledgeWritePayloadsWithoutTimestamps(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"CostEntry 不发送时间戳", `{"name":"hp300","title":"HP300","category":"机械","unit":"台班","price":3200,"spec":"300kW","source":"测试","tags":[],"status":"现行","body":""}`},
		{"KnowledgeEntry 不发送时间戳", `{"name":"test-k","title":"测试知识","category":"规范标准","phase":"施工","discipline":"土木","tags":[],"status":"现行","version":1,"author":"","reviewer":"","source":"","body":"正文"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ce CostEntry
			if err := json.Unmarshal([]byte(tc.body), &ce); err != nil {
				t.Fatalf("CostEntry 解码失败: %v", err)
			}
			var ke KnowledgeEntry
			if err := json.Unmarshal([]byte(tc.body), &ke); err != nil {
				t.Fatalf("KnowledgeEntry 解码失败: %v", err)
			}
		})
	}
}

// TestWailsArgsRejectEmptyTimestamps 复现 Wails ParseArgs 的失败路径：
// 只要负载里带 updatedAt:""，json.Unmarshal 就会整体失败。
func TestWailsArgsRejectEmptyTimestamps(t *testing.T) {
	bad := `[{"name":"hp300","title":"HP300","price":3200,"updatedAt":"","createdAt":""}]`
	var rows []CostEntry
	if err := json.Unmarshal([]byte(bad), &rows); err == nil {
		t.Fatal("期望空字符串时间戳解码失败（前端必须省略字段），当前竟然成功")
	}
}

// TestBridgeClientInjectedWithoutGaeaInit 回归：未进入办公板块（GaeaInit
// 未执行）时，configureClient 也必须注入桥接 client，否则成本/知识导入的
// AI 解析会报 "bridge: ai.LLMClient 未注入"。
func TestBridgeClientInjectedWithoutGaeaInit(t *testing.T) {
	a := &App{core: &core{}}
	a.client = ai.NewClient(config.Load())
	a.configureClient()

	p, err := provider.New("wubigrok", provider.Config{Name: "cost-import-ai", Model: "", Engine: ""})
	if err != nil {
		t.Fatalf("桥接 provider 创建失败（client 未注入）: %v", err)
	}
	if p == nil {
		t.Fatal("provider nil")
	}
}

// TestPDFTextUsable 乱码/空内容不应进入 AI 提示词，可读 OCR 文本应放行。
func TestPDFTextUsable(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"项目周报：本周完成 8 项需求，营收 120 万元，同比增长 18%，修复目标砷 ≤ 60 mg/kg。", true},
		{"", false},
		{"? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ? ?", false},
		{"水泥单价 450 元/吨 膨润土单价 900 元/吨 外加剂 3500 元/吨", true},
		{"abcd", false},
	}
	for _, c := range cases {
		if got := pdfTextUsable(c.in); got != c.want {
			t.Errorf("pdfTextUsable(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
