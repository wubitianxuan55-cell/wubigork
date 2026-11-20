package agent

// W0 工具产出图片随行测试（电脑操控施工单 2026-11-19 W0-4/W0-5）：
// 工具经 ctx ImageSink 登记路径 → executeBatch 结果图片同长同序；
// 压缩摘要不吃图（只读 Content）。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

type imagingTool struct{ path string }

func (i imagingTool) Name() string            { return "imaging_x" }
func (i imagingTool) Description() string     { return "" }
func (i imagingTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (i imagingTool) ReadOnly() bool          { return true }
func (i imagingTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) {
	tool.ImageSinkFromContext(ctx).AddImage(i.path)
	return "shot saved", nil
}

func TestExecuteBatchCarriesToolImages(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(imagingTool{path: `C:\shot.png`})
	a := New(nil, reg, NewSession(""), Options{}, event.Discard)
	results, imgs := a.executeBatch(context.Background(), []provider.ToolCall{
		{ID: "1", Name: "imaging_x", Arguments: `{}`},
		{ID: "2", Name: "imaging_x", Arguments: `{}`},
	})
	if len(results) != 2 || len(imgs) != 2 {
		t.Fatalf("结果与图片同长: results=%d imgs=%d", len(results), len(imgs))
	}
	for i := range imgs {
		if len(imgs[i]) != 1 || imgs[i][0] != `C:\shot.png` {
			t.Fatalf("调用 %d 应带回 1 张路径图: %v", i, imgs[i])
		}
	}
}

func TestExecuteBatchNoImagesNil(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(countingTool{name: "writer_x"})
	a := New(nil, reg, NewSession(""), Options{}, event.Discard)
	_, imgs := a.executeBatch(context.Background(), []provider.ToolCall{
		{ID: "1", Name: "writer_x", Arguments: `{}`},
	})
	if len(imgs) != 1 || imgs[0] != nil {
		t.Fatalf("无图调用 images 必须为 nil: %v", imgs)
	}
}

func TestImageSinkNilSafe(t *testing.T) {
	var nilSink *tool.ImageSink
	nilSink.AddImage("x.png") // 不得 panic
	if got := nilSink.Paths(); got != nil {
		t.Fatalf("nil sink Paths 应为 nil: %v", got)
	}
	sink := &tool.ImageSink{}
	sink.AddImage("")
	sink.AddImage("a.png")
	sink.AddImage("a.png") // 重复登记照收（去重在 bridge/上层职责）
	got := sink.Paths()
	if len(got) != 2 || got[0] != "a.png" {
		t.Fatalf("Paths 快照错: %v", got)
	}
	// 快照是副本：追加不影响已取出的切片。
	sink.AddImage("b.png")
	if len(got) != 2 {
		t.Fatalf("快照必须为副本: %v", got)
	}
}

func TestBuildCompactSummaryIgnoresImages(t *testing.T) {
	summary := BuildCompactSummary([]provider.Message{{
		Role:    provider.RoleTool,
		Content: "screenshot saved to uploads/x.png",
		Images:  []string{"data:image/jpeg;base64,AAAAAAAAAAAAAAAA"},
	}})
	if strings.Contains(summary, "base64") || strings.Contains(summary, "data:image") {
		t.Fatalf("压缩摘要不得吃进图片: %s", summary)
	}
}
