package app

// office_recovery_contract_test.go — R2 分档诚实恢复契约静态契约测试
// （众神殿 pan_1795058214142 裁决 action-1，用户已签注 accepted）：
// ① office_edit/office_save 描述钉住分档边界（Word=当前会话一次 Ctrl+Z 整批；
//    Excel/PPT=不承诺整批/不承诺逐操作；三档均无跨会话回滚）；
// ② save 回执恒带恢复边界行，Journal 卡诚实缺省（BaselinePath 空+摘要自陈
//    无基线快照）；
// ③ 源级守卫：OfficeHelper.cs 的 undo/UndoRecord 仅 Word——未来有人给
//    Excel/PPT 加整批撤销时此测试红=提醒同步更新 R2 文案（dsh-r3 断言④）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/comhelper"
)

func TestOfficeComRecoveryContractPinned(t *testing.T) {
	editDesc := (officeComEditTool{}).Description()
	// Word 档：整批撤销+会话域限定。
	for _, pin := range []string{"一次 Ctrl+Z 可整体撤销", "仅限当前文档与当前会话", "不能恢复编辑前版本"} {
		if !strings.Contains(editDesc, pin) {
			t.Errorf("office_edit 描述缺钉句 %q:\n%s", pin, editDesc)
		}
	}
	// Excel/PPT 档：只声明助手不提供整批撤销，粒度未核验不落事实。
	for _, pin := range []string{"不提供整批撤销", "不要向用户保证可撤销"} {
		if !strings.Contains(editDesc, pin) {
			t.Errorf("office_edit 描述缺钉句 %q:\n%s", pin, editDesc)
		}
	}
	if strings.Contains(editDesc, "逐操作撤销") || strings.Contains(editDesc, "可逐操作") {
		t.Errorf("Excel/PPT 撤销粒度未核验，office_edit 描述不得写成逐操作可撤:\n%s", editDesc)
	}

	saveDesc := (officeComSaveTool{}).Description()
	for _, pin := range []string{"仅可复核", "没有跨会话回滚"} {
		if !strings.Contains(saveDesc, pin) {
			t.Errorf("office_save 描述缺钉句 %q:\n%s", pin, saveDesc)
		}
	}
	// 两描述禁词：COM 档无基线快照，不得暗示 gaea 侧回滚能力（dsh-r3 断言①）。
	for name, desc := range map[string]string{"office_edit": editDesc, "office_save": saveDesc} {
		for _, banned := range []string{"可回滚", "可恢复"} {
			if strings.Contains(desc, banned) {
				t.Errorf("%s 描述含禁词 %q（虚假恢复承诺）:\n%s", name, banned, desc)
			}
		}
	}
}

func TestOfficeComSaveReceiptRecoveryBoundary(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	stub := &stubCaller{reply: func(cmd string, args map[string]any) (json.RawMessage, error) {
		switch cmd {
		case "open":
			return stubJSON(docInfoStub("opened", ws)), nil
		case "edit": // 保存前的 fill_gaps 补空（无需填空）
			return stubJSON(comhelper.EditResult{Done: []string{"no page needed filling"}}), nil
		case "save":
			return stubJSON(map[string]any{"path": filepath.ToSlash(filepath.Join(ws, "报告.docx")), "bytes": 1234}), nil
		default:
			return nil, os.ErrInvalid
		}
	}}
	resetOfficeComForTest(stub)
	ctx := context.Background()
	if _, err := (officeComOpenTool{}).ExecuteWithContext(ctx, tool.ToolContext{SessionID: "s-r2"}, json.RawMessage(`{"path":"报告.docx"}`)); err != nil {
		t.Fatalf("open: %v", err)
	}
	out, err := (officeComSaveTool{}).Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	for _, pin := range []string{"恢复边界：", "仅可复核、无基线快照", "没有跨会话回滚", "仅 Word 一次 Ctrl+Z 整批、限当前会话"} {
		if !strings.Contains(out, pin) {
			t.Errorf("save 回执缺恢复边界钉句 %q:\n%s", pin, out)
		}
	}
	if strings.Contains(out, "可回滚") {
		t.Errorf("save 回执含禁词「可回滚」:\n%s", out)
	}
	// Journal 诚实缺省上锁：BaselinePath 空 + 摘要自陈无基线快照（dsh-r3 断言③）。
	for _, rec := range readOfficeJournal(t, ws) {
		if rec.Tool != "office_save" {
			continue
		}
		if rec.BaselinePath != "" {
			t.Errorf("COM 档证据卡 BaselinePath 应为空（诚实缺省），got %q", rec.BaselinePath)
		}
		if !strings.Contains(rec.BeforeSummary, "无基线快照") {
			t.Errorf("证据卡摘要应自陈无基线快照: %q", rec.BeforeSummary)
		}
		return
	}
	t.Fatal("office_save 证据卡未落账")
}

// TestOfficeHelperUndoGuardPinned 源级守卫：OfficeHelper.cs 的 undo 命令与
// UndoRecord 必须保持仅 Word——Excel/PPT 若未来获得整批撤销，R2 文案（描述/
// 回执/本文件断言）必须同批更新，此测试红即提醒。
func TestOfficeHelperUndoGuardPinned(t *testing.T) {
	root := guardRepoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "internal", "office", "comhelper", "OfficeHelper.cs"))
	if err != nil {
		t.Fatalf("读 OfficeHelper.cs: %v", err)
	}
	src := string(raw)
	pins := []string{
		// undo 命令：非 word 直接 fail。
		`if (kind != "word") throw new Fail("BAD_ARGS", "Only Word keeps the agent's edits on its undo stack.");`,
		// 整批撤销记录：仅 word 分支开启/闭合。
		`if (kind == "word") { try { undo = app.UndoRecord; undo.StartCustomRecord("AI edit"); } catch (COMException) { undo = null; } }`,
		`if (undo != null) { try { undo.EndCustomRecord(); } catch (COMException) { } }`,
	}
	for _, pin := range pins {
		if !strings.Contains(src, pin) {
			t.Errorf("OfficeHelper.cs 撤销分档守卫漂移（若已给 Excel/PPT 加整批撤销，须同批更新 R2 契约文案与测试）,缺:\n%s", pin)
		}
	}
}
