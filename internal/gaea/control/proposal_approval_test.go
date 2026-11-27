package control

import (
	"encoding/json"
	"strings"
	"testing"
)

// 方案库维护工具的审批口径：update/delete 与 save 同属逐条确认；save 的演练
// （dry_run=true）不写库，不进确认卡。
func TestProposalApprovalSubjectsAndDryRun(t *testing.T) {
	for _, name := range []string{"proposal_save", "proposal_update", "proposal_delete"} {
		if !hardAskTools[name] {
			t.Fatalf("%s 应属逐条确认工具", name)
		}
	}
	if hardAskTools["proposal_export"] {
		t.Fatal("proposal_export 只写工作区文件，不属于持久化写入确认集")
	}

	dryRun := json.RawMessage(`{"title":"甲","dry_run":true,"outline":[{"level":1,"title":"一"}]}`)
	if !proposalArgsDryRun(dryRun) {
		t.Fatal("dry_run=true 的录入应判为演练")
	}
	for _, args := range []string{
		`{"title":"甲","dry_run":false}`,
		`{"title":"甲","dry_run":true,"action":"status"}`,
		`{"title":"甲"}`,
		`not json`,
	} {
		if proposalArgsDryRun(json.RawMessage(args)) {
			t.Fatalf("%s 不应判为演练", args)
		}
	}
	if s := proposalSaveApprovalSubject(dryRun); !strings.Contains(s, "演练，不写库") {
		t.Fatalf("演练的确认摘要应写明不写库：%q", s)
	}
	if s := proposalSaveApprovalSubject(json.RawMessage(`{"title":"甲","mode":"append","outline":[{"level":1,"title":"一"}]}`)); !strings.Contains(s, "续录") {
		t.Fatalf("续录摘要：%q", s)
	}

	// 不写库的调用（2026-11-27 AI 反馈 #12）：演练、查进度、认标题都不弹卡；带了
	// 录入内容的照旧逐条确认。
	for _, args := range []string{
		`{"title":"甲","dry_run":true,"outline":[{"level":1,"title":"一"}]}`,
		`{"title":"甲","action":"status"}`,
		`{"source_file":"a.md","action":"detect_headings"}`,
		`{"source_file":"a.md","action":"detect_headings","outline_file":".gaea/work/o.json"}`,
	} {
		if !proposalArgsNoWrite(json.RawMessage(args)) {
			t.Fatalf("%s 应判为不写库", args)
		}
	}
	for _, args := range []string{
		`{"title":"甲","dry_run":false}`,
		`{"title":"甲","outline":[{"level":1,"title":"一"}]}`,
		`{"title":"甲","source_file":"a.md","outline_file":"o.json"}`,
		`{"title":"甲"}`,
		`not json`,
	} {
		if proposalArgsNoWrite(json.RawMessage(args)) {
			t.Fatalf("%s 写库，不应放行", args)
		}
	}

	upd := json.RawMessage(`{"proposal":"甲方案","title":"乙方案","section_id":3,"section_title":"1.1 新","body":"x"}`)
	s := approvalSubjectFor("proposal_update", upd)
	for _, want := range []string{"修改方案库方案：甲方案", "改标题为「乙方案」", "修正小节 #3", "标题→「1.1 新」", "整节替换正文"} {
		if !strings.Contains(s, want) {
			t.Fatalf("proposal_update 摘要缺 %q：%q", want, s)
		}
	}
	if s := approvalSubjectFor("proposal_update", json.RawMessage(`{"proposal":"甲方案","year":"2027","summary":"新摘要"}`)); !strings.Contains(s, "改年度/摘要") {
		t.Fatalf("元数据改动摘要：%q", s)
	}
	if s := approvalSubjectFor("proposal_update", json.RawMessage(`{"proposal":"甲方案"}`)); strings.Contains(s, "改标题为") || strings.Contains(s, "改年度") || strings.Contains(s, "修正小节") {
		t.Fatalf("没有改动项时摘要不该说改了什么：%q", s)
	}
	if s := approvalSubjectFor("proposal_delete", json.RawMessage(`{"proposal":"甲方案","reason":"录错"}`)); !strings.Contains(s, "从方案库删除（不可恢复）：甲方案") || !strings.Contains(s, "理由：录错") {
		t.Fatalf("proposal_delete 摘要：%q", s)
	}
}
