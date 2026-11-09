package app

// P1a 结构化写入口测试（2026-11-09 众神殿 pan_1794234593182 裁决 action-3）：
//   - office_generate 声明路径与实际落点逐字节一致（裁决验收线）+ 无声明/越界/
//     扩展名不一致拒绝 + Journal 证据卡落账（新文件无基线=诚实缺省）；
//   - 四类写 WritePathResolver（DAG write_paths 声明面地基）+ 非只读 +
//     不标 PersistWrite（FilterRegistry 不摘新入口的验收线，接口断言在
//     office_tools.go，本处钉工具面形状）；
//   - 通道 A 文案修复：docx_apply/pptx_apply 等非 editLike 工具未比对时不得
//     回「变更摘要与文件实况一致」（裁决风险条：未比对不得回 pass）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// isolateOfficeWorkspace 同 verify 测试基建：工作区指向临时目录（gaeaCwd 跟随）
// 且生效空间归一 work（Journal 守卫放行）。
func isolateOfficeWorkspace(t *testing.T) string {
	t.Helper()
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	return ws
}

func readOfficeJournal(t *testing.T, ws string) []evidence.ChangeRecord {
	t.Helper()
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	recs, err := st.List("unsaved") // 测试态无 controller，appendOfficeEvidence 记 unsaved
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return recs
}

func TestOfficeGenerateDeclaredPathByteExact(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	md := "# 季度总结\n\n正文第一段。"
	args, _ := json.Marshal(map[string]string{
		"markdown": md, "format": "md", "output_path": "报告/季度总结.md", "title": "季度总结",
	})
	out, err := officeGenerateTool{a: &App{}}.Execute(t.Context(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 验收线：声明路径与实际落点逐字节一致
	declared := filepath.Join(ws, "报告", "季度总结.md")
	raw, rerr := os.ReadFile(declared)
	if rerr != nil {
		t.Fatalf("声明路径落点不存在: %v（out=%s）", rerr, out)
	}
	if string(raw) != md {
		t.Fatalf("落点内容逐字节不一致: %q", string(raw))
	}
	if !strings.Contains(out, "报告/季度总结.md") {
		t.Fatalf("回执未报告声明路径: %s", out)
	}
	// Journal 证据卡（office_generate，无基线快照）
	recs := readOfficeJournal(t, ws)
	var found *evidence.ChangeRecord
	for i := range recs {
		if recs[i].Tool == "office_generate" {
			found = &recs[i]
		}
	}
	if found == nil {
		t.Fatalf("office_generate 证据卡未落账: %+v", recs)
	}
	if found.Target != "报告/季度总结.md" {
		t.Fatalf("证据卡 Target=%q, want 声明相对路径", found.Target)
	}
	if found.BaselinePath != "" {
		t.Fatalf("新文件无基线快照，BaselinePath 应空（通道 B 记 n/a）: %q", found.BaselinePath)
	}
}

func TestOfficeGenerateRejectsUndeclaredUnsafeMismatched(t *testing.T) {
	isolateOfficeWorkspace(t)
	tl := officeGenerateTool{a: &App{}}
	cases := []struct {
		name string
		args string
		want string
	}{
		{"缺 output_path", `{"markdown":"x","format":"md"}`, "缺少 output_path"},
		{"越界逃逸", `{"markdown":"x","format":"md","output_path":"../escape.md"}`, "非法 output_path"},
		{"扩展名不一致", `{"markdown":"x","format":"md","output_path":"报告/总结.docx"}`, "不一致"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tl.Execute(t.Context(), json.RawMessage(tc.args))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want 含 %q", err, tc.want)
			}
		})
	}
}

func TestOfficeGenerateWritePathsResolves(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	got, err := officeGenerateTool{}.WritePaths(json.RawMessage(`{"output_path":"交付/a.docx"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != filepath.Join(ws, "交付", "a.docx") {
		t.Fatalf("WritePaths=%v, want 声明路径绝对化", got)
	}
	if _, err := (officeGenerateTool{}).WritePaths(json.RawMessage(`{}`)); err == nil {
		t.Fatal("缺声明路径应报错（WritePathResolver 与 Execute 同判据）")
	}
}

func TestOfficeApplyToolsWritePathsAndShape(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	four := []tool.Tool{
		officeDocxApplyTool{},
		officeDocxAcceptTool{},
		officePptxApplyTool{},
		officeXlsxApplyTool{},
	}
	for _, tl := range four {
		if tl.ReadOnly() {
			t.Fatalf("%s 写工具不得自报只读", tl.Name())
		}
		if tool.IsPersistWrite(tl) {
			t.Fatalf("%s 不得标 PersistWrite（FilterRegistry 须保留，write_paths 声明面可用）", tl.Name())
		}
		resolver, ok := tl.(tool.WritePathResolver)
		if !ok {
			t.Fatalf("%s 未实现 WritePathResolver", tl.Name())
		}
		got, err := resolver.WritePaths(json.RawMessage(`{"path":"文档/合同.docx"}`))
		if err != nil {
			t.Fatalf("%s WritePaths: %v", tl.Name(), err)
		}
		if len(got) != 1 || got[0] != filepath.Join(ws, "文档", "合同.docx") {
			t.Fatalf("%s WritePaths=%v", tl.Name(), got)
		}
		if _, err := resolver.WritePaths(json.RawMessage(`{"path":"../outside.docx"}`)); err == nil {
			t.Fatalf("%s 越界路径应拒绝", tl.Name())
		}
	}
	// docx_accept 仅 accept=true：schema 无 accept 参数（拒绝路径留 UI，裁决口径）。
	schema := string(officeDocxAcceptTool{}.Schema())
	if strings.Contains(schema, `"accept"`) {
		t.Fatalf("docx_accept schema 不得暴露 accept 参数: %s", schema)
	}
}

func TestOfficeApplyExecuteTargetsMissingFileFailsLoud(t *testing.T) {
	isolateOfficeWorkspace(t)
	_, err := officeDocxApplyTool{a: &App{}}.Execute(t.Context(),
		json.RawMessage(`{"path":"不存在.docx","selected_text":"a","replacement":"b"}`))
	if err == nil {
		t.Fatal("目标缺失应 fail-loud")
	}
	_, err = officeXlsxApplyTool{a: &App{}}.Execute(t.Context(),
		json.RawMessage(`{"path":"不存在.xlsx","ops_json":"[]"}`))
	if err == nil {
		t.Fatal("xlsx 目标缺失应 fail-loud")
	}
}

// TestVerifyChannelAOfficeToolsNoFakePass 通道 A 文案修复钉死：docx_apply 走
// default 分支时未比对摘要，不得回「变更摘要与文件实况一致」（旧文案=虚假一致
// 声明，裁决风险条）；edit_like 三工具语义不变。
func TestVerifyChannelAOfficeToolsNoFakePass(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	writeVerifyRecordTool(t, ws, "ev_docx_1", "合同.docx", "docx_apply", "这段文字不应出现在文件里")
	v, err := (&App{}).GaeaVerifyRecord("ev_docx_1")
	if err != nil {
		t.Fatalf("GaeaVerifyRecord: %v", err)
	}
	if strings.Contains(v.ChannelA, "变更摘要与文件实况一致") {
		t.Fatalf("未比对不得回一致性 pass: %q", v.ChannelA)
	}
	if !strings.Contains(v.ChannelA, "未做引用级比对") {
		t.Fatalf("应诚实声明未比对: %q", v.ChannelA)
	}
	// edit_like 仍走摘要比对（旧语义回归锚）
	writeVerifyRecordTool(t, ws, "ev_edit_1", "note.md", "edit_file", "after content")
	v2, err := (&App{}).GaeaVerifyRecord("ev_edit_1")
	if err != nil {
		t.Fatalf("GaeaVerifyRecord edit: %v", err)
	}
	if !strings.Contains(v2.ChannelA, "变更摘要与文件实况一致") {
		t.Fatalf("editLike 语义不得变: %q", v2.ChannelA)
	}
}

// writeVerifyRecordTool writeVerifyRecord 的任意工具名变体（目标内容=摘要，
// 供 editLike 分支比对；docx_apply 用例的摘要故意不在文件里）。
func writeVerifyRecordTool(t *testing.T, ws, id, targetRel, toolName, afterSummary string) {
	t.Helper()
	target := filepath.Join(ws, targetRel)
	if err := os.WriteFile(target, []byte(afterSummary+"\n"), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	if err := st.Append(evidence.ChangeRecord{
		ID:            id,
		SessionID:     "t-session",
		Space:         "work",
		Turn:          1,
		Tool:          toolName,
		Target:        targetRel,
		BeforeSummary: "before content\n",
		AfterSummary:  afterSummary,
		Status:        evidence.StatusPendingVerify,
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
}
