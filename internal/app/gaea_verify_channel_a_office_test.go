package app

// P2 通道 A 引用级深化测试（2026-11-10 众神殿 pan_1794234593182 裁决 action-5
// 前半）：docx_apply/docx_accept/pptx_apply/office_generate 从「目标可读」
// 升级为「声明↔实况」引用级比对——修复 P1a 文案修复时记录的「复核面弱通过」
// 缺口。造件复用 abeval hermetic 造件器（abBuildDocx/abBuildDocxRevs/
// abBuildPptx，同包测试件）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/docxedit"
	"github.com/gaea/gaea/internal/office/pptxedit"
)

func TestVerifyChannelADocxApplyDeep(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	path := filepath.Join(ws, "合同.docx")
	if err := os.WriteFile(path, abBuildDocx(t, []string{"服务合同", "合同期限为 30 天。"}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := docxedit.ApplyTrackedReplace(path, "30 天", "60 天", "gaea AI"); err != nil {
		t.Fatalf("ApplyTrackedReplace: %v", err)
	}
	rec := evidence.ChangeRecord{Tool: "docx_apply", OpsJSON: docxApplyOpsJSON("30 天", "60 天")}
	got := verifyDocxApplyChannelADeep(path, rec)
	if !strings.Contains(got, "pass") || !strings.Contains(got, "引用级") {
		t.Fatalf("修订在册应 pass 引用级: %q", got)
	}
	// 接受修订后：替换文本仍在文档（w:ins 扁平化），但修订不在册 → fail
	if err := docxedit.AcceptChanges(path, "gaea AI"); err != nil {
		t.Fatalf("AcceptChanges: %v", err)
	}
	got2 := verifyDocxApplyChannelADeep(path, rec)
	if !strings.Contains(got2, "fail") || !strings.Contains(got2, "无本作者待处理修订") {
		t.Fatalf("修订已接受后应 fail（无修订在册）: %q", got2)
	}
	// 声明与实况不一致：替换文本不在文档 → fail
	path2 := filepath.Join(ws, "合同2.docx")
	if err := os.WriteFile(path2, abBuildDocx(t, []string{"服务合同", "合同期限为 30 天。"}), 0o644); err != nil {
		t.Fatal(err)
	}
	rec2 := evidence.ChangeRecord{Tool: "docx_apply", OpsJSON: docxApplyOpsJSON("30 天", "90 天")}
	got3 := verifyDocxApplyChannelADeep(path2, rec2)
	if !strings.Contains(got3, "fail") || !strings.Contains(got3, "替换后文本未出现") {
		t.Fatalf("替换文本缺失应 fail: %q", got3)
	}
}

func TestVerifyChannelADocxApplyDeepFallbackSummary(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	path := filepath.Join(ws, "合同.docx")
	if err := os.WriteFile(path, abBuildDocx(t, []string{"服务合同", "合同期限为 30 天。"}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := docxedit.ApplyTrackedReplace(path, "30 天", "60 天", "gaea AI"); err != nil {
		t.Fatal(err)
	}
	// 旧卡无 opsJSON：回退解析 AfterSummary「→ 60 天（修订制写入）」取替换文本
	rec := evidence.ChangeRecord{Tool: "docx_apply", AfterSummary: "→ 60 天（修订制写入）"}
	got := verifyDocxApplyChannelADeep(path, rec)
	if !strings.Contains(got, "pass") || !strings.Contains(got, "引用级") {
		t.Fatalf("摘要回退应完成引用级比对: %q", got)
	}
	// 双缺（无 opsJSON 且摘要不可解析）：诚实跳过，不冒充核对
	rec2 := evidence.ChangeRecord{Tool: "docx_apply"}
	got2 := verifyDocxApplyChannelADeep(path, rec2)
	if !strings.Contains(got2, "pass") || !strings.Contains(got2, "旧卡无声明载荷") {
		t.Fatalf("声明双缺应诚实跳过: %q", got2)
	}
}

func TestVerifyChannelADocxAcceptDeep(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	path := filepath.Join(ws, "reviewed.docx")
	if err := os.WriteFile(path, abBuildDocxRevs(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := verifyDocxAcceptChannelADeep(path); !strings.Contains(got, "fail") {
		t.Fatalf("接受前应有待处理修订 → fail: %q", got)
	}
	if err := docxedit.AcceptChanges(path, "gaea AI"); err != nil {
		t.Fatalf("AcceptChanges: %v", err)
	}
	if got := verifyDocxAcceptChannelADeep(path); !strings.Contains(got, "pass") || !strings.Contains(got, "引用级") {
		t.Fatalf("接受后应 pass 引用级: %q", got)
	}
}

func TestVerifyChannelAPptxApplyDeep(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	path := filepath.Join(ws, "deck.pptx")
	if err := os.WriteFile(path, abBuildPptx(t, []string{"预算总额：120 万元", "交付计划：2027 年 Q1 交付"}), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := pptxedit.ApplyTextReplace(path, 1, "预算总额：120 万元", "预算总额：180 万元"); err != nil {
		t.Fatalf("ApplyTextReplace: %v", err)
	}
	rec := evidence.ChangeRecord{Tool: "pptx_apply"}
	rec.OpsJSON, _ = mustPptxOpsJSON(1, "预算总额：120 万元", "预算总额：180 万元")
	got := verifyPptxApplyChannelADeep(path, rec)
	if !strings.Contains(got, "pass") || !strings.Contains(got, "目标文本已不在") {
		t.Fatalf("替换在+目标不在应 pass 引用级: %q", got)
	}
	// 错页 → fail
	rec2 := evidence.ChangeRecord{Tool: "pptx_apply"}
	rec2.OpsJSON, _ = mustPptxOpsJSON(3, "预算总额：120 万元", "预算总额：180 万元")
	if got := verifyPptxApplyChannelADeep(path, rec2); !strings.Contains(got, "fail") || !strings.Contains(got, "页不存在") {
		t.Fatalf("页码越界应 fail: %q", got)
	}
	// 声明双缺 → 诚实跳过
	if got := verifyPptxApplyChannelADeep(path, evidence.ChangeRecord{Tool: "pptx_apply"}); !strings.Contains(got, "旧卡无声明载荷") {
		t.Fatalf("声明双缺应诚实跳过: %q", got)
	}
}

// mustPptxOpsJSON pptx 声明载荷构造（与 appendPptxEvidence 同形）。
func mustPptxOpsJSON(slide int, target, replacement string) (string, bool) {
	b, err := json.Marshal(map[string]any{"slide": slide, "target": target, "replacement": replacement})
	return string(b), err == nil
}

func TestVerifyChannelAOfficeGenerateDeep(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	// md：存在非空 → pass
	mdPath := filepath.Join(ws, "总结.md")
	if err := os.WriteFile(mdPath, []byte("# 总结\n\n正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := verifyOfficeGenerateChannelA(mdPath); !strings.Contains(got, "pass") {
		t.Fatalf("md 交付应 pass: %q", got)
	}
	// docx：OOXML 容器头 → pass
	docxPath := filepath.Join(ws, "交付.docx")
	if err := os.WriteFile(docxPath, abBuildDocx(t, []string{"x"}), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := verifyOfficeGenerateChannelA(docxPath); !strings.Contains(got, "pass") || !strings.Contains(got, "OOXML") {
		t.Fatalf("docx 容器应 pass: %q", got)
	}
	// 空文件 → fail
	emptyPath := filepath.Join(ws, "空.docx")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := verifyOfficeGenerateChannelA(emptyPath); !strings.Contains(got, "fail") {
		t.Fatalf("空交付应 fail: %q", got)
	}
	// 非 OOXML（文本字节冒充 .docx）→ fail
	fakePath := filepath.Join(ws, "假.docx")
	if err := os.WriteFile(fakePath, []byte("this is not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := verifyOfficeGenerateChannelA(fakePath); !strings.Contains(got, "fail") || !strings.Contains(got, "OOXML") {
		t.Fatalf("非容器冒充应 fail: %q", got)
	}
	// 缺失 → fail
	if got := verifyOfficeGenerateChannelA(filepath.Join(ws, "不存在.docx")); !strings.Contains(got, "fail") {
		t.Fatalf("缺失应 fail: %q", got)
	}
}
