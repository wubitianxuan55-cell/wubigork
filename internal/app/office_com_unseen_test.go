package app

// office_com_unseen_test.go — 波3.5 unseen 交付前检查闭环测试：
// edit 置位 → save 触发总览+Final check 段 → save 后清除；
// render 清除（看过再 save 不多调 render）；close 清除。
// preferLive 引导段文案钉死 + 探测下沉冒烟。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/tool"
	comhelper "github.com/gaea/gaea/internal/office/comhelper"
)

func TestOfficeComUnseenLifecycle(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	docPath := filepath.Join(ws, "报告.docx")
	var renderCalls int
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		switch cmd {
		case "open":
			return stubJSON(docInfoStub("opened", ws)), nil
		case "edit":
			return stubJSON(comhelper.EditResult{Done: []string{"set_text OK"}, Total: 1}), nil
		case "render":
			renderCalls++
			if outPath, _ := args["out"].(string); outPath != "" {
				_ = os.WriteFile(outPath, []byte("png"), 0o644)
			}
			from := 1
			if v, ok := args["from"].(float64); ok {
				from = int(v)
			}
			return stubJSON(comhelper.RenderResult{What: fmt.Sprintf("Word pages %d-%d of 3", from, from+5), Total: 3, Width: 640, Height: 800}), nil
		case "save":
			return stubJSON(map[string]any{"path": filepath.ToSlash(docPath), "bytes": 42}), nil
		case "close":
			return stubJSON("ok"), nil
		default:
			return nil, fmt.Errorf("unexpected cmd %s", cmd)
		}
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()
	docJSON := `{"doc":"` + filepath.ToSlash(docPath) + `"`
	editJSON := func(expect, text string) json.RawMessage {
		return json.RawMessage(docJSON + `,"ops":[{"op":"set_text","para":1,"expect":"` + expect + `","text":"` + text + `"}]}`)
	}

	// edit 成功 → unseen 置位。
	if _, err := (officeComEditTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-u"}, editJSON("x", "y")); err != nil {
		t.Fatalf("edit: %v", err)
	}
	officeCom.unseenMu.Lock()
	wasUnseen := officeCom.unseen[officeComKey("word", docPath)]
	officeCom.unseenMu.Unlock()
	if !wasUnseen {
		t.Fatal("edit 成功后 unseen 应置位")
	}

	// save（未看过）→ 保存后自动 overview（render 被调）+ Final check 段，随后清除。
	out, err := (officeComSaveTool{}).Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if renderCalls == 0 {
		t.Fatal("unseen 置位时 save 应自动跑总览 render")
	}
	if !strings.Contains(out, "Final check") || !strings.Contains(out, "vision") {
		t.Fatalf("save 回执缺交付前检查段: %.300s", out)
	}
	officeCom.unseenMu.Lock()
	_, still := officeCom.unseen[officeComKey("word", docPath)]
	officeCom.unseenMu.Unlock()
	if still {
		t.Fatal("save 后 unseen 应清除")
	}

	// 再 edit → render（看过）→ save：不自动总览。
	if _, err = (officeComEditTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-u"}, editJSON("y", "z")); err != nil {
		t.Fatalf("edit2: %v", err)
	}
	if _, err = (officeComRenderTool{}).Execute(ctx, json.RawMessage(docJSON+`,"page":1}`)); err != nil {
		t.Fatalf("render: %v", err)
	}
	before := renderCalls
	if _, err = (officeComSaveTool{}).Execute(ctx, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("save2: %v", err)
	}
	if renderCalls != before {
		t.Fatal("看过渲染图后 save 不应再自动总览")
	}

	// 再 edit → close：unseen 不残留（close 前须 open 收编 own——用户文档拒绝
	// 关闭是 own 守卫语义，测试同守卫）。
	if _, err = (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-u"}, json.RawMessage(`{"path":"`+filepath.ToSlash(docPath)+`"}`)); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err = (officeComEditTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-u"}, editJSON("z", "w")); err != nil {
		t.Fatalf("edit3: %v", err)
	}
	if _, err = (officeComCloseTool{}).Execute(ctx, json.RawMessage(docJSON+`,"save":false}`)); err != nil {
		t.Fatalf("close: %v", err)
	}
	officeCom.unseenMu.Lock()
	_, left := officeCom.unseen[officeComKey("word", docPath)]
	officeCom.unseenMu.Unlock()
	if left {
		t.Fatal("close 后 unseen 应清除")
	}
}

func TestOfficeLivePromptSectionPinned(t *testing.T) {
	// preferLive 引导段文案钉死（boot 拼接段；关键引导句必须在场）。
	sec := boot.OfficeLivePromptSection()
	for _, want := range []string{"office_open", "office_edit", "office_render {overview:true}", "不要用 docx/pptx 技能", "office_batch"} {
		if !strings.Contains(sec, want) {
			t.Fatalf("office 实时协同引导段缺关键句 %q", want)
		}
	}
	// 探测下沉冒烟：测试机装了 WPS 应为 true；语义=单次探测缓存（不 assert，
	// 环境差异只记录——探测本体逻辑由注册条件装配守卫覆盖）。
	if !comhelper.ProbeSuiteAvailable() {
		t.Log("本机未探测到 Office/WPS（ProbeSuiteAvailable=false）——记录环境")
	}
}
