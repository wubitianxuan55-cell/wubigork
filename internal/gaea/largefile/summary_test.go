package largefile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// mockProv is a canned provider: each Stream call returns the next reply.
type mockProv struct {
	calls   int
	replies []string
}

func (m *mockProv) Name() string { return "mock" }

func (m *mockProv) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	reply := "摘要块"
	if m.calls < len(m.replies) {
		reply = m.replies[m.calls]
	}
	m.calls++
	ch := make(chan provider.Chunk, 1)
	ch <- provider.Chunk{Type: provider.ChunkText, Text: reply}
	close(ch)
	return ch, nil
}

func TestChunkText(t *testing.T) {
	// 段落感知：两个小段落合成一块，超预算时切块
	text := "第一段内容。\n\n第二段内容。\n\n" + strings.Repeat("长段落", 5000)
	chunks := ChunkText(text, 1000)
	if len(chunks) == 0 {
		t.Fatal("no chunks")
	}
	joined := strings.Join(chunks, "\n\n")
	if !strings.Contains(joined, "第一段内容") || !strings.Contains(joined, "第二段内容") {
		t.Errorf("chunks dropped paragraph content")
	}
	// 大段落被硬切：不应存在超过预算的块
	for i, c := range chunks {
		if runeLen(c) > 1100 {
			t.Errorf("chunk %d exceeds budget: %d runes", i, runeLen(c))
		}
	}
	if got := ChunkText("   \n\n  ", 100); got != nil {
		t.Errorf("whitespace text should yield nil, got %v", got)
	}
}

func TestExtractText_PDFAndPlain(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "doc.pdf")
	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&sb, "/Type /Page\nBT /F1 12 Tf 72 720 Td (第 %d 页) Tj ET\n", i)
	}
	if err := os.WriteFile(pdf, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	text, total, err := ExtractText(pdf, 20)
	if err != nil {
		t.Fatalf("ExtractText pdf: %v", err)
	}
	if total != 30 {
		t.Errorf("total = %d, want 30", total)
	}
	if !strings.Contains(text, "第 1 页") || strings.Contains(text, "第 21 页") {
		t.Errorf("pdf text should be capped at first 20 pages")
	}

	txt := filepath.Join(dir, "note.md")
	if err := os.WriteFile(txt, []byte("# 标题\n正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	text2, total2, err := ExtractText(txt, 0)
	if err != nil || total2 != 0 || !strings.Contains(text2, "标题") {
		t.Errorf("plain text extraction wrong: total=%d err=%v", total2, err)
	}
}

func TestSummarize_MapReduce(t *testing.T) {
	// 4 个 chunk，每块摘要足够短 → 无二次合并
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&sb, "第 %d 段：项目周报数据与结论。\n\n", i)
	}
	prov := &mockProv{replies: []string{"要点一", "要点二", "要点三", "要点四"}}
	summary, chunks, err := Summarize(context.Background(), prov, sb.String(), Options{ChunkRunes: 40})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if chunks != 4 {
		t.Errorf("chunks = %d, want 4", chunks)
	}
	if prov.calls != 4 {
		t.Errorf("provider calls = %d, want 4 (map-reduce, no merge pass)", prov.calls)
	}
	if !strings.Contains(summary, "要点一") || !strings.Contains(summary, "要点四") {
		t.Errorf("merged summary missing chunk summaries: %q", summary)
	}
}

func TestSummarize_MergePass(t *testing.T) {
	// 多 chunk 且块摘要足够长 → 触发「摘要的摘要」合并 pass
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&sb, "第 %d 段：项目周报数据与结论。\n\n", i)
	}
	prov := &mockProv{replies: []string{
		strings.Repeat("要点甲。", 2000), // 8000 runes × 4 = 32000 > 合并阈值
		strings.Repeat("要点乙。", 2000),
		strings.Repeat("要点丙。", 2000),
		strings.Repeat("要点丁。", 2000),
		"合并总览：以上四块要点。",
	}}
	summary, chunks, err := Summarize(context.Background(), prov, sb.String(), Options{ChunkRunes: 40})
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if chunks != 4 {
		t.Errorf("chunks = %d, want 4", chunks)
	}
	if prov.calls != 5 {
		t.Errorf("provider calls = %d, want 5 (4 chunks + 1 merge pass)", prov.calls)
	}
	if !strings.Contains(summary, "合并总览") {
		t.Errorf("merge-pass summary missing: %q", summary)
	}
}

func TestSummarizeFile_AndTool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.md")
	if err := os.WriteFile(path, []byte("# 报告\n\n第一段。\n\n第二段。\n\n第三段。\n\n第四段。"), 0o644); err != nil {
		t.Fatal(err)
	}
	prov := &mockProv{replies: []string{"要点一", "要点二", "要点三", "要点四"}}
	tool := NewSummarizeTool(prov)
	raw := fmt.Sprintf(`{"path":%q}`, path)
	out, err := tool.Execute(context.Background(), json.RawMessage(raw))
	if err != nil {
		t.Fatalf("tool Execute: %v", err)
	}
	if !strings.Contains(out, `"ok":true`) || !strings.Contains(out, "要点一") || !strings.Contains(out, "分 1 块") {
		t.Errorf("tool output missing summary envelope: %q", out)
	}

	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("empty path should error")
	}
	if _, err := SummarizeFile(context.Background(), prov, filepath.Join(dir, "missing.md"), Options{}); err == nil {
		t.Error("missing file should error")
	}
}

func TestSummarizeFiles_MultiMerge(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.md")
	b := filepath.Join(dir, "b.md")
	if err := os.WriteFile(a, []byte("# 报告A\n\n第一段内容。"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# 报告B\n\n第二段内容。"), 0o644); err != nil {
		t.Fatal(err)
	}
	prov := &mockProv{replies: []string{"A 摘要", "B 摘要", "合并总览：A+B"}}
	res, err := SummarizeFiles(context.Background(), prov, []string{a, b}, Options{})
	if err != nil {
		t.Fatalf("SummarizeFiles: %v", err)
	}
	if res.Files != 2 || len(res.Paths) != 2 {
		t.Errorf("Files/Paths = %d/%v, want 2/2", res.Files, res.Paths)
	}
	if prov.calls != 3 {
		t.Errorf("provider calls = %d, want 3 (2 files + 1 merge pass)", prov.calls)
	}
	if !strings.Contains(res.Summary, "合并总览") {
		t.Errorf("merged summary missing: %q", res.Summary)
	}

	// 工具多文件路径
	tool := NewSummarizeTool(prov)
	raw := fmt.Sprintf(`{"paths":[%q,%q]}`, a, b)
	out, err := tool.Execute(context.Background(), json.RawMessage(raw))
	if err != nil {
		t.Fatalf("tool multi Execute: %v", err)
	}
	if !strings.Contains(out, "2 个文件") || !strings.Contains(out, "合并总览") {
		t.Errorf("multi-file tool output wrong: %q", out)
	}
}

func TestSummarize_NilProvider(t *testing.T) {
	if _, _, err := Summarize(context.Background(), nil, "内容", Options{}); err == nil {
		t.Error("nil provider should error")
	}
}
