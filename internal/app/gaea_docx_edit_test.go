package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/evidence"
)

// docxWithText 构造一个含指定段落文本的最小 docx。
func docxWithText(t *testing.T, text string) []byte {
	t.Helper()
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>` + text + `</w:t></w:r></w:p>
  </w:body>
</w:document>`
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": docXML,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestGaeaDocxApplyEdit(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "edit-test.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "合同期限为 30 天。"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天")
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "docx" {
		t.Fatalf("kind = %q, want docx", got.Kind)
	}
	if !strings.HasPrefix(got.DataURL, "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,") {
		t.Error("预览 dataUrl 缺失")
	}

	// 落盘文件应包含修订标记
	r, err := zip.OpenReader(rel)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var docXML []byte
	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		docXML, _ = io.ReadAll(rc)
		rc.Close()
	}
	s := string(docXML)
	if !strings.Contains(s, "<w:del ") || !strings.Contains(s, "<w:ins ") {
		t.Errorf("docx 未写入修订标记: %s", s)
	}
	if !strings.Contains(s, "<w:delText>30 天</w:delText>") || !strings.Contains(s, ">60 天</w:t>") {
		t.Errorf("修订内容不正确: %s", s)
	}
}

func TestGaeaDocxApplyEdit_NotFound(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "edit-miss.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "原始文本"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "找不到", "替换"); err == nil {
		t.Fatal("期望未命中错误")
	}
}

func TestGaeaDocxAcceptChanges(t *testing.T) {
	t.Chdir(t.TempDir())
	rel := filepath.Join(".gaea", "uploads", "accept-test.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "合同期限为 30 天。"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天"); err != nil {
		t.Fatal(err)
	}
	got, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "docx" {
		t.Fatalf("kind = %q", got.Kind)
	}
	r, err := zip.OpenReader(rel)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var docXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			docXML, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	s := string(docXML)
	if strings.Contains(s, "<w:del ") || strings.Contains(s, "<w:ins ") {
		t.Error("接受后仍有修订标记")
	}
	if !strings.Contains(s, ">60 天</w:t>") {
		t.Error("接受后新文未生效")
	}
	// 无修订时再次接受应报错
	if _, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true); err == nil {
		t.Fatal("无修订时接受应报错")
	}
}

// ── v4.157 小刀：证据链（快照 + Journal）测试 ─────────────────────────────
// 对齐 pptx_apply/xlsx_apply 口径：docx_apply/docx_accept 落盘前快照进
// .gaea/work/rollback/docx-*.before、成功后落 Journal 证据卡（pending_verify，
// 回滚走 GaeaRollbackRecord）。

// injectWorkSpace 注入「work 空间生效」的内存配置。gaeaEffectiveSpace 在
// ga.cfg==nil（引擎未初始化，单测缺省态）时返回 ""（≠"work"），appendDocxEvidence
// 的空间守卫会直接 return，故证据链用例必须先注入。注入先例=gaea_pptx_test.go /
// gaea_export_test.go 的 ga.cfg 直赋；Workspace 刻意留空使 gaeaCwd() 回退
// os.Getwd()（= t.Chdir 后的临时目录），t.Cleanup 还原原值，避免 stale
// Workspace 让后续用例的 gaeaCwd 指向已删除目录。
func injectWorkSpace(t *testing.T) {
	t.Helper()
	orig := ga.cfg
	ga.cfg = &gaeaConfig.Config{}
	t.Cleanup(func() { ga.cfg = orig })
}

// readDocxJournal 读取 .gaea/work/journal 下全部 JSONL 证据卡。测试里
// ga.ctrl 未初始化 → SessionID 恒 "unsaved"（文件名 unsaved.jsonl）；按 glob
// 全扫，不依赖该文件名细节。
func readDocxJournal(t *testing.T) []evidence.ChangeRecord {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(".gaea", "work", "journal", "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var recs []evidence.ChangeRecord
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var rec evidence.ChangeRecord
			if err := json.Unmarshal([]byte(line), &rec); err != nil {
				t.Fatalf("证据卡解析失败: %v: %s", err, line)
			}
			recs = append(recs, rec)
		}
	}
	return recs
}

// findDocxRecord 按 Tool+Target 过滤证据卡（docx 绑定面的 Target=传入 rel，
// slash 形态）。
func findDocxRecord(recs []evidence.ChangeRecord, tool, target string) *evidence.ChangeRecord {
	for i := range recs {
		if recs[i].Tool == tool && recs[i].Target == target {
			return &recs[i]
		}
	}
	return nil
}

// countDocxSnapshots 统计 .gaea/work/rollback/docx-*.before 快照数。
func countDocxSnapshots(t *testing.T) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(".gaea", "work", "rollback", "docx-*.before"))
	if err != nil {
		t.Fatal(err)
	}
	return len(matches)
}

// TestGaeaDocxApplyEdit_EvidenceChain v4.157 小刀主用例：应用编辑后
//   ① .gaea/work/rollback/ 出现 docx-*.before 且内容=应用前字节；
//   ② .gaea/work/journal 的 JSONL 出现 Tool=docx_apply 记录，BaselinePath 非空
//     且指向该快照，两个摘要含目标/替换文本。
// 返回值宽断言：GaeaDocxApplyEdit 尾部调 a.GaeaPreview(rel)，docx 预览走
// base64 dataUrl（无 soffice 依赖），当前测试环境 err==nil；若 preview 实现
// 变化导致 binding 返回 error，不影响上方已断言的副作用（编辑+证据链均已发生）。
func TestGaeaDocxApplyEdit_EvidenceChain(t *testing.T) {
	t.Chdir(t.TempDir())
	injectWorkSpace(t)
	rel := filepath.Join(".gaea", "uploads", "evidence-apply.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	before := docxWithText(t, "合同期限为 30 天。")
	if err := os.WriteFile(rel, before, 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	got, perr := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天")
	if perr == nil && got.Kind != "docx" {
		t.Errorf("kind = %q, want docx", got.Kind)
	}

	// ① 落盘前快照：唯一一张、内容=应用前字节
	matches, err := filepath.Glob(filepath.Join(".gaea", "work", "rollback", "docx-*.before"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("rollback 快照数 = %d, want 1", len(matches))
	}
	// Glob 用相对模式 → 相对路径；BaselinePath 由 gaeaCwd() 拼接为绝对路径，
	// 比对前统一 Abs 归一。
	snapPath, err := filepath.Abs(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	snap, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(snap, before) {
		t.Error("快照内容应等于应用前字节")
	}

	// ② Journal 证据卡：Tool/Target/BaselinePath/摘要/状态口径
	rec := findDocxRecord(readDocxJournal(t), "docx_apply", filepath.ToSlash(rel))
	if rec == nil {
		t.Fatal("journal 缺少 docx_apply 记录")
	}
	if rec.BaselinePath == "" {
		t.Error("BaselinePath 应非空")
	} else if rec.BaselinePath != snapPath {
		t.Errorf("BaselinePath = %q, want 快照路径 %q", rec.BaselinePath, snapPath)
	}
	if rec.Space != "work" || rec.Status != evidence.StatusPendingVerify {
		t.Errorf("Space/Status = %q/%q, want work/%q", rec.Space, rec.Status, evidence.StatusPendingVerify)
	}
	if !strings.Contains(rec.BeforeSummary, "30 天") {
		t.Errorf("BeforeSummary 应含目标文本: %q", rec.BeforeSummary)
	}
	if !strings.Contains(rec.AfterSummary, "60 天") || !strings.Contains(rec.AfterSummary, "修订制写入") {
		t.Errorf("AfterSummary 应含替换文本与口径: %q", rec.AfterSummary)
	}
}

// TestGaeaDocxApplyEdit_EvidenceSummaryTruncation 截断口径：超长目标/替换文本
// 的摘要按 truncateStr(,120) 字节截断（120 字节 + 省略号；「长」「替」各
// 3 字节/字 → 恰 40 字 + …，字节截断落在 rune 边界上，结果确定）。
func TestGaeaDocxApplyEdit_EvidenceSummaryTruncation(t *testing.T) {
	t.Chdir(t.TempDir())
	injectWorkSpace(t)
	rel := filepath.Join(".gaea", "uploads", "evidence-trunc.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	longTarget := strings.Repeat("长", 200)
	longRepl := strings.Repeat("替", 200)
	if err := os.WriteFile(rel, docxWithText(t, longTarget), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &App{}
	// 返回值不校验（preview 降级不影响副作用断言；应用失败会以 rec==nil 显形）
	_, _ = a.GaeaDocxApplyEdit(filepath.ToSlash(rel), longTarget, longRepl)
	rec := findDocxRecord(readDocxJournal(t), "docx_apply", filepath.ToSlash(rel))
	if rec == nil {
		t.Fatal("journal 缺少 docx_apply 记录")
	}
	wantBefore := strings.Repeat("长", 40) + "…"
	if rec.BeforeSummary != wantBefore {
		t.Errorf("BeforeSummary 截断口径不符: got %d bytes, want %d bytes", len(rec.BeforeSummary), len(wantBefore))
	}
	wantAfter := "→ " + strings.Repeat("替", 40) + "…（修订制写入）"
	if rec.AfterSummary != wantAfter {
		t.Errorf("AfterSummary 截断口径不符: got %d bytes, want %d bytes", len(rec.AfterSummary), len(wantAfter))
	}
}

// TestGaeaDocxAcceptChanges_EvidenceChain v4.157 小刀：accept=true（接受修订=
// 清除修订标记的不可逆整理）加快照 + Tool=docx_accept 记录；accept=false
//（拒绝=恢复原文，天然回滚语义）既不加快照也不落记录。
func TestGaeaDocxAcceptChanges_EvidenceChain(t *testing.T) {
	t.Chdir(t.TempDir())
	injectWorkSpace(t)
	rel := filepath.Join(".gaea", "uploads", "evidence-accept.docx")
	if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rel, docxWithText(t, "合同期限为 30 天。"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "30 天", "60 天"); err != nil {
		t.Fatal(err)
	}
	beforeAccept, err := os.ReadFile(rel)
	if err != nil {
		t.Fatal(err)
	}
	if n := countDocxSnapshots(t); n != 1 {
		t.Fatalf("应用后快照数 = %d, want 1", n)
	}

	// accept=true：新快照（内容=接受前带修订标记的字节）+ docx_accept 记录
	if _, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), true); err != nil {
		t.Fatal(err)
	}
	if n := countDocxSnapshots(t); n != 2 {
		t.Errorf("接受后快照数 = %d, want 2", n)
	}
	rec := findDocxRecord(readDocxJournal(t), "docx_accept", filepath.ToSlash(rel))
	if rec == nil {
		t.Fatal("journal 缺少 docx_accept 记录")
	}
	if rec.BaselinePath == "" {
		t.Error("accept BaselinePath 应非空")
	} else if snap, err := os.ReadFile(rec.BaselinePath); err != nil {
		t.Fatalf("读 accept 快照失败: %v", err)
	} else if !bytes.Equal(snap, beforeAccept) {
		t.Error("accept 快照内容应等于接受前字节")
	}
	if rec.BeforeSummary != "接受 gaea 修订（清除修订标记）" {
		t.Errorf("BeforeSummary = %q", rec.BeforeSummary)
	}
	if rec.AfterSummary != "修订已接受" {
		t.Errorf("AfterSummary = %q", rec.AfterSummary)
	}

	// accept=false：再写一笔修订供拒绝；断言快照数与 docx_accept 记录数均不增
	if _, err := a.GaeaDocxApplyEdit(filepath.ToSlash(rel), "60 天", "90 天"); err != nil {
		t.Fatal(err)
	}
	snapsBefore := countDocxSnapshots(t)
	acceptsBefore := len(readDocxJournal(t))
	if _, err := a.GaeaDocxAcceptChanges(filepath.ToSlash(rel), false); err != nil {
		t.Fatal(err)
	}
	if n := countDocxSnapshots(t); n != snapsBefore {
		t.Errorf("拒绝后快照数 = %d, want 不变 %d（拒绝=天然回滚，不加快照）", n, snapsBefore)
	}
	recs := readDocxJournal(t)
	if n := len(recs); n != acceptsBefore {
		t.Errorf("拒绝后证据卡数 = %d, want 不变 %d（拒绝=天然回滚，不落记录）", n, acceptsBefore)
	}
	acceptCount := 0
	for i := range recs {
		if recs[i].Tool == "docx_accept" {
			acceptCount++
		}
	}
	if acceptCount != 1 {
		t.Errorf("docx_accept 记录数 = %d, want 1（拒绝分支不得新增）", acceptCount)
	}
}
