package app

// v4.16 Verifier 通道 B（PDF 视觉 diff）结果产品化：
// GaeaVerifyRecord 在通道 B 路径回填 ChannelBRatio / ChannelBPages /
// ChannelBArtifacts 三字段（随 verdict 落 verdicts.jsonl 并返回前端）；
// 无通道 B（无 BaselinePath）时三字段省略（omitempty 零值）。
// 渲染链路 seam（verifyConvertToPdf / verifyRenderPages / verifyPixelDiff）
// 注入 fake 实现，纯函数、无 soffice / poppler 依赖。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
)

// verifyChannelBFakeSeams 注入 fake 渲染链路：转换写假 PDF、渲染按页返回
// PNG 路径、像素比对返回固定差异率。返回恢复函数。
func verifyChannelBFakeSeams(t *testing.T, ratio float64, pageCount int) func() {
	t.Helper()
	oldConv, oldRender, oldDiff := verifyConvertToPdf, verifyRenderPages, verifyPixelDiff
	verifyConvertToPdf = func(src, out string) error {
		return os.WriteFile(out, []byte("fake pdf"), 0o644)
	}
	verifyRenderPages = func(pdf, prefix string, dpi int) ([]string, error) {
		out := make([]string, 0, pageCount)
		for i := 1; i <= pageCount; i++ {
			out = append(out, prefix+string(rune('0'+i))+".png")
		}
		return out, nil
	}
	verifyPixelDiff = func(a, b string) (float64, error) { return ratio, nil }
	return func() {
		verifyConvertToPdf, verifyRenderPages, verifyPixelDiff = oldConv, oldRender, oldDiff
	}
}

// writeVerifyRecord 把一张 write_file 证据卡落进工作区的 journal（模拟
// Apply 后的审计链），目标文件真实可读（通道 A 无需 soffice）。
func writeVerifyRecord(t *testing.T, ws, id, targetRel string, baseline bool) {
	t.Helper()
	target := filepath.Join(ws, targetRel)
	if err := os.WriteFile(target, []byte("after content\n"), 0o644); err != nil {
		t.Fatalf("写目标文件: %v", err)
	}
	rec := evidence.ChangeRecord{
		ID:            id,
		SessionID:     "cb-session",
		Space:         "work",
		Turn:          1,
		Tool:          "write_file",
		Target:        targetRel,
		BeforeSummary: "before content\n",
		AfterSummary:  "after content\n",
		Status:        evidence.StatusPendingVerify,
		At:            time.Now().UnixMilli(),
	}
	if baseline {
		bl := filepath.Join(ws, "baseline.before")
		if err := os.WriteFile(bl, []byte("before content\n"), 0o644); err != nil {
			t.Fatalf("写基线快照: %v", err)
		}
		rec.BaselinePath = bl
	}
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatalf("OpenJournal: %v", err)
	}
	if err := st.Append(rec); err != nil {
		t.Fatalf("Append: %v", err)
	}
}

// isolateWorkspaceTo 把办公工作区指向临时目录（gaeaCwd 跟随），并恢复全局
// 配置，避免污染其他测试。
func isolateWorkspaceTo(t *testing.T, ws string) {
	t.Helper()
	restore := workspaceTestIsolate(t)
	t.Cleanup(restore)
	ga.mu.Lock()
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.cfg = oldCfg
		ga.mu.Unlock()
	})
}

// TestGaeaVerifyRecord_ChannelBBackfill 通道 B 成功路径：三字段回填 +
// verdict 落库往返 + JSON 形状（omitempty 携带）。
func TestGaeaVerifyRecord_ChannelBBackfill(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	writeVerifyRecord(t, ws, "ev_cb_1", "doc.md", true)
	restore := verifyChannelBFakeSeams(t, 0.05, 2)
	defer restore()

	v, err := (&App{}).GaeaVerifyRecord("ev_cb_1")
	if err != nil {
		t.Fatalf("GaeaVerifyRecord: %v", err)
	}
	// 0.05 ∈ (0.02, 0.20] 且页数不变 → warn
	if v.Status != evidence.VerdictWarned {
		t.Errorf("Status = %q, want warned", v.Status)
	}
	if v.ChannelBRatio != 0.05 {
		t.Errorf("ChannelBRatio = %f, want 0.05", v.ChannelBRatio)
	}
	if v.ChannelBPages != 2 {
		t.Errorf("ChannelBPages = %d, want 2（before/after 各 2 页）", v.ChannelBPages)
	}
	wantArtifacts := filepath.ToSlash(filepath.Join(ws, ".gaea", "work", "journal", "verify", "ev_cb_1"))
	if v.ChannelBArtifacts != wantArtifacts {
		t.Errorf("ChannelBArtifacts = %q, want %q", v.ChannelBArtifacts, wantArtifacts)
	}
	if _, err := os.Stat(filepath.Join(ws, ".gaea", "work", "journal", "verify", "ev_cb_1")); err != nil {
		t.Errorf("产物目录应已创建: %v", err)
	}

	// JSON 形状：三字段随 verdict 序列化（omitempty 非零值携带）
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal verdict: %v", err)
	}
	for _, key := range []string{`"channelBRatio":0.05`, `"channelBPages":2`, `"channelBArtifacts":`} {
		if !strings.Contains(string(b), key) {
			t.Errorf("JSON 缺少 %s, got %s", key, b)
		}
	}

	// verdicts.jsonl 落库往返（VerdictOf 后者胜）
	st, _ := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	got, ok := st.VerdictOf("ev_cb_1")
	if !ok {
		t.Fatal("VerdictOf 未找到复核结论")
	}
	if got.ChannelBRatio != 0.05 || got.ChannelBPages != 2 || got.ChannelBArtifacts != wantArtifacts {
		t.Errorf("落库往返字段漂移: %+v", got)
	}
}

// TestGaeaVerifyRecord_NoChannelB 无通道 B（无 BaselinePath）：三字段省略
// （零值），JSON 不带新键（旧 verdict 向后兼容）。
func TestGaeaVerifyRecord_NoChannelB(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	writeVerifyRecord(t, ws, "ev_cb_2", "doc.md", false)

	v, err := (&App{}).GaeaVerifyRecord("ev_cb_2")
	if err != nil {
		t.Fatalf("GaeaVerifyRecord: %v", err)
	}
	if v.ChannelB != "n/a" {
		t.Errorf("ChannelB = %q, want n/a", v.ChannelB)
	}
	if v.ChannelBRatio != 0 || v.ChannelBPages != 0 || v.ChannelBArtifacts != "" {
		t.Errorf("无通道 B 应三字段零值, got %+v", v)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal verdict: %v", err)
	}
	for _, key := range []string{"channelBRatio", "channelBPages", "channelBArtifacts"} {
		if strings.Contains(string(b), key) {
			t.Errorf("无通道 B 时 JSON 不应含 %s, got %s", key, b)
		}
	}
}
