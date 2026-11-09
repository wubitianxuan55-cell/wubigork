package app

// P2 条件视觉首渲测试（2026-11-10 众神殿 pan_1794234593182 裁决 action-5 后半；
// 备忘 :42-48 形状条款继承）：无基线（首次生成）交付物不再是无视觉复核——
// 渲染链路 seam 注入 fake，纯函数、无 soffice/poppler 依赖（同
// gaea_verify_channel_b_test.go 先例）。覆盖=抽样封顶/整页/降级/不可渲染四态，
// 全部透明落账（visualPagesReviewed/visualPagesTotal/visualSkippedReason）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

// firstReviewSeams 首渲 seam 注入：convertErr 非 nil=soffice 降级；pages 非
// nil=pdftoppm 产出（假 PDF + PNG 路径清单）。返回恢复函数。
func firstReviewSeams(t *testing.T, convertErr error, pages []string) func() {
	t.Helper()
	oldConv, oldRender := verifyConvertToPdf, verifyRenderPages
	verifyConvertToPdf = func(src, out string) error {
		if convertErr != nil {
			return convertErr
		}
		return os.WriteFile(out, []byte("fake pdf"), 0o644)
	}
	verifyRenderPages = func(pdf, prefix string, dpi int) ([]string, error) {
		if pages != nil {
			return pages, nil
		}
		return nil, fmt.Errorf("pdftoppm missing")
	}
	return func() {
		verifyConvertToPdf, verifyRenderPages = oldConv, oldRender
	}
}

// writeFirstReviewRecord office_generate 证据卡（无基线，目标为真 docx 字节）。
func writeFirstReviewRecord(t *testing.T, ws, id string) string {
	t.Helper()
	targetRel := "交付/报告.docx"
	target := filepath.Join(ws, filepath.FromSlash(targetRel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, abBuildDocx(t, []string{"# 报告", "正文第一页。"}), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Append(evidence.ChangeRecord{
		ID:           id,
		SessionID:    "fr-session",
		Space:        "work",
		Turn:         1,
		Tool:         "office_generate",
		Target:       targetRel,
		AfterSummary: "format=docx size=1024 bytes",
		OpsJSON:      "",
		BaselinePath: "", // 无基线=首渲场景
		Status:       evidence.StatusPendingVerify,
		At:           1000,
	}); err != nil {
		t.Fatal(err)
	}
	return targetRel
}

func TestGaeaVerifyRecord_FirstRenderReviewSampling(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	writeFirstReviewRecord(t, ws, "ev_fr_1")
	defer verifyChannelBFakeSeams(t, 0, 12)() // 12 页 > 8 → 抽样

	v, err := (&App{}).GaeaVerifyRecord("ev_fr_1")
	if err != nil {
		t.Fatalf("GaeaVerifyRecord: %v", err)
	}
	if !strings.Contains(v.ChannelB, "first-review: 首渲覆盖 8/12 页") {
		t.Fatalf("ChannelB 缺首渲覆盖口径: %q", v.ChannelB)
	}
	if v.VisualPagesReviewed != 8 || v.VisualPagesTotal != 12 {
		t.Fatalf("覆盖字段=%d/%d, want 8/12", v.VisualPagesReviewed, v.VisualPagesTotal)
	}
	if !strings.Contains(v.VisualSkippedReason, "抽样") {
		t.Fatalf("跳过原因应透明: %q", v.VisualSkippedReason)
	}
	if v.Status != evidence.VerdictVerified || !strings.Contains(v.Note, "条件视觉首渲") {
		t.Fatalf("首渲态汇总错误: status=%s note=%s", v.Status, v.Note)
	}
	if v.ChannelBArtifacts == "" {
		t.Fatal("首渲产物目录应随 verdict 落库")
	}
}

func TestGaeaVerifyRecord_FirstRenderReviewAllPages(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	writeFirstReviewRecord(t, ws, "ev_fr_2")
	defer verifyChannelBFakeSeams(t, 0, 5)() // 5 页 ≤ 8 → 全页覆盖无跳过

	v, err := (&App{}).GaeaVerifyRecord("ev_fr_2")
	if err != nil {
		t.Fatal(err)
	}
	if v.VisualPagesReviewed != 5 || v.VisualPagesTotal != 5 || v.VisualSkippedReason != "" {
		t.Fatalf("整页覆盖应无跳过原因: %d/%d %q", v.VisualPagesReviewed, v.VisualPagesTotal, v.VisualSkippedReason)
	}
}

func TestGaeaVerifyRecord_FirstRenderDegraded(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	writeFirstReviewRecord(t, ws, "ev_fr_3")
	defer firstReviewSeams(t, fmt.Errorf("soffice not found"), nil)()

	v, err := (&App{}).GaeaVerifyRecord("ev_fr_3")
	if err != nil {
		t.Fatal(err)
	}
	if v.ChannelB != "n/a" {
		t.Fatalf("渲染降级 ChannelB 应保持 n/a（不冒充审过）: %q", v.ChannelB)
	}
	if v.VisualPagesReviewed != 0 {
		t.Fatalf("降级 reviewed 应零: %d", v.VisualPagesReviewed)
	}
	if !strings.Contains(v.VisualSkippedReason, "渲染降级") {
		t.Fatalf("降级原因应透明: %q", v.VisualSkippedReason)
	}
}

func TestGaeaVerifyRecord_FirstRenderRenderFail(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	writeFirstReviewRecord(t, ws, "ev_fr_4")
	defer firstReviewSeams(t, nil, nil)() // convert OK，pdftoppm 失败

	v, err := (&App{}).GaeaVerifyRecord("ev_fr_4")
	if err != nil {
		t.Fatal(err)
	}
	if v.ChannelB != "n/a" || v.VisualPagesReviewed != 0 {
		t.Fatalf("渲染失败应 n/a+reviewed 零: %q %d", v.ChannelB, v.VisualPagesReviewed)
	}
	if !strings.Contains(v.VisualSkippedReason, "pdftoppm") {
		t.Fatalf("渲染失败原因应透明: %q", v.VisualSkippedReason)
	}
}

func TestGaeaVerifyRecord_FirstReviewNotRenderable(t *testing.T) {
	ws := isolateOfficeWorkspace(t)
	// md 交付：不在首渲格式集 → ChannelB n/a、零覆盖字段（诚实缺省，不冒充）
	targetRel := "交付/总结.md"
	target := filepath.Join(ws, filepath.FromSlash(targetRel))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# 总结\n\n正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := evidence.OpenJournal(filepath.Join(ws, ".gaea", "work", "journal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Append(evidence.ChangeRecord{
		ID: "ev_fr_5", SessionID: "fr-session", Space: "work", Turn: 1,
		Tool: "office_generate", Target: targetRel,
		AfterSummary: "format=md", BaselinePath: "", Status: evidence.StatusPendingVerify, At: 1000,
	}); err != nil {
		t.Fatal(err)
	}
	v, err := (&App{}).GaeaVerifyRecord("ev_fr_5")
	if err != nil {
		t.Fatal(err)
	}
	if v.ChannelB != "n/a" || v.VisualPagesReviewed != 0 || v.VisualPagesTotal != 0 || v.VisualSkippedReason != "" {
		t.Fatalf("不可渲染格式应零覆盖字段（诚实缺省）: %q %+v", v.ChannelB, v)
	}
}
