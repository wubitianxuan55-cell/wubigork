package largefile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// mockProv is a canned provider: each Stream call returns the next reply.
// fails 为第几次调用（1 起）返回错误——重试逻辑的回归钉用。
type mockProv struct {
	calls   int
	replies []string
	fails   map[int]error
}

func (m *mockProv) Name() string { return "mock" }

func (m *mockProv) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	call := m.calls + 1
	m.calls++
	if err, bad := m.fails[call]; bad {
		ch := make(chan provider.Chunk, 1)
		ch <- provider.Chunk{Type: provider.ChunkError, Err: err}
		close(ch)
		return ch, nil
	}
	reply := "摘要块"
	if call-1 < len(m.replies) {
		reply = m.replies[call-1]
	}
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

// 2026-11-28 黄甲反馈：带 base64 图片的素材 md 不该按图片体积切块——实测一份
// 53MB 的素材切出 1055 块、第 4 块超时后整单作废。
func TestExtractText_StripsDataImages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.md")
	body := "# 方案\n\n正文第一段。\n\n![工艺图](data:image/png;base64," + strings.Repeat("QUJD", 200000) + ")\n\n正文第二段。\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	text, _, err := ExtractText(path, 0)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if strings.Contains(text, "data:image/") || len(text) > 200 {
		t.Fatalf("内嵌图片的 base64 应在分块前去掉：len=%d head=%.120q", len(text), text)
	}
	if !strings.Contains(text, "正文第一段") || !strings.Contains(text, "正文第二段") {
		t.Fatalf("正文应保留：%q", text)
	}
}

// 量级闸：块数超上限直接报错，一次模型调用都不发起。
func TestSummarize_ChunkCapFailsFast(t *testing.T) {
	text := strings.Repeat("段落内容。\n\n", 12)
	prov := &mockProv{}
	_, _, err := Summarize(context.Background(), prov, text, Options{ChunkRunes: 20, MaxChunks: 3})
	if err == nil || !strings.Contains(err.Error(), "超过单次摘要上限 3 块") || !strings.Contains(err.Error(), "format_convert") {
		t.Fatalf("超限应报错并给出路：%v", err)
	}
	if prov.calls != 0 {
		t.Fatalf("超限不该发起模型调用，got %d", prov.calls)
	}
}

// 单块瞬时失败重试一次：失败一次整单照成；连败两次才作废；调用方已取消不重试。
func TestSummarize_RetriesChunkOnce(t *testing.T) {
	text := "第一段。\n\n第二段。\n\n第三段。\n\n第四段。"
	transient := errors.New("context deadline exceeded")

	prov := &mockProv{fails: map[int]error{2: transient}}
	if _, chunks, err := Summarize(context.Background(), prov, text, Options{ChunkRunes: 6}); err != nil || chunks != 4 {
		t.Fatalf("失败一次应重试后成功：chunks=%d err=%v", chunks, err)
	}
	if prov.calls != 5 {
		t.Fatalf("应为 4 块 + 1 次重试，got %d", prov.calls)
	}

	prov = &mockProv{fails: map[int]error{2: transient, 3: transient}}
	_, _, err := Summarize(context.Background(), prov, text, Options{ChunkRunes: 6})
	if err == nil || !strings.Contains(err.Error(), "第 2/4 块摘要失败") {
		t.Fatalf("连败两次应作废并点名块号：%v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	prov = &mockProv{fails: map[int]error{1: ctx.Err()}}
	if _, _, err := Summarize(ctx, prov, text, Options{ChunkRunes: 6}); err == nil {
		t.Fatal("已取消应报错")
	}
	if prov.calls != 1 {
		t.Fatalf("已取消不该重试，got %d", prov.calls)
	}
}
