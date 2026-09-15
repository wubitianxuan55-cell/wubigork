package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/docxedit"
)

// GaeaOfficeEditText 框选即改：按自然语言指令生成选中文本的替换内容
// （办公向提示词：信息不变、措辞严谨、纯文本输出）。
func (a *App) GaeaOfficeEditText(selectedText, instruction string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	if selectedText == "" {
		return nil, fmt.Errorf("选中文本为空")
	}
	if instruction == "" {
		return nil, fmt.Errorf("编辑指令为空")
	}

	// 2026-08-28 本地优先强化：Word 框选即改属办公功能级调用，优先本地
	// Herdsman（数据不出本机、省 token），不可用时回退常规路由。
	featEng, featModel, _ := a.routeOfficeLocal("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	edited, err := a.client.OfficeEditText(ctx, featEng, featModel, "office", selectedText, instruction)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"edited": edited}, nil
}

// GaeaDocxApplyEdit 把替换以修订模式（w:del + w:ins）写入 docx，
// 返回更新后的预览（前端直接重渲染，修订样式可见）。
//
// v4.157 小刀（docs/gaea-pptx-edit-design-2026-09.md §6 拍板项 4：补 docx_apply
// 不落证据链的既有欠账，不动 docx 编辑行为本身）：ApplyTrackedReplace 调用
// 语义零改动，只在其之前加落盘前基线快照、成功后加 Journal 证据卡——对齐
// pptx_apply/xlsx_apply 证据链口径，回滚走 GaeaRollbackRecord（VersionTimeline
// kind "docx" 已支持）。
func (a *App) GaeaDocxApplyEdit(rel, selectedText, replacement string) (PreviewResult, error) {
	if rel == "" {
		return PreviewResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	// v4.157 小刀：落盘前基线快照（同 pptx_apply/xlsx_apply 口径；Verifier
	// 通道 B 对有快照的写盘记录复核）。快照失败不阻断编辑（BaselinePath 如实
	// 留空，静默降级），ApplyTrackedReplace 的报错行为与改造前逐字节一致。
	baseline := docxBaselineSnapshot(path)
	if err := docxedit.ApplyTrackedReplace(path, selectedText, replacement, "gaea AI"); err != nil {
		return PreviewResult{}, err
	}
	// v4.157 小刀：应用成功后落证据卡（摘要 120 截断口径，防 JSONL 膨胀）。
	appendDocxEvidence(rel, "docx_apply",
		truncateStr(selectedText, 120),
		"→ "+truncateStr(replacement, 120)+"（修订制写入）",
		baseline)
	return a.GaeaPreview(rel), nil
}

// GaeaDocxAcceptChanges 接受/拒绝 gaea 的待处理修订（accept=true 接受全部，
// false 拒绝全部），返回更新后的预览。
//
// v4.157 小刀：accept=true（接受修订=清除修订标记的不可逆整理）加落盘前
// 快照 + Journal 证据卡（Tool=docx_accept）；accept=false 不加——拒绝=把文本
// 恢复为修订前原文，天然回滚语义，无需快照/证据链兜底。
func (a *App) GaeaDocxAcceptChanges(rel string, accept bool) (PreviewResult, error) {
	if rel == "" {
		return PreviewResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if accept {
		baseline := docxBaselineSnapshot(path)
		if err := docxedit.AcceptChanges(path, "gaea AI"); err != nil {
			return PreviewResult{}, err
		}
		appendDocxEvidence(rel, "docx_accept",
			"接受 gaea 修订（清除修订标记）",
			"修订已接受",
			baseline)
	} else {
		if err := docxedit.RejectChanges(path, "gaea AI"); err != nil {
			return PreviewResult{}, err
		}
	}
	return a.GaeaPreview(rel), nil
}

// docxBaselineSnapshot 把 path 当前内容快照进 work 回滚目录（文件名前缀
// docx-，同 pptx-*/xlsx-* 口径），返回快照绝对路径；读不到/写失败返回 ""
// （快照尽力而为，不阻断编辑主意图）。v4.157 小刀新增，仅服务 docx 两个
// 绑定面，不动 docxedit 包。
func docxBaselineSnapshot(path string) string {
	baseline := ""
	raw, err := os.ReadFile(path)
	if err != nil {
		return baseline
	}
	rbDir := filepath.Join(gaeaCwd(), ".gaea", "work", "rollback")
	_ = os.MkdirAll(rbDir, 0o755)
	bp := filepath.Join(rbDir, fmt.Sprintf("docx-%d.before", time.Now().UnixNano()))
	if werr := os.WriteFile(bp, raw, 0o644); werr == nil {
		baseline = bp
	}
	return baseline
}

// appendDocxEvidence 把一次 docx 编辑（docx_apply 应用 / docx_accept 接受修订）
// 写入 work 空间 Journal（JSONL）。
// v4.157 小刀：对齐 pptx_apply/xlsx_apply 证据链口径，回滚走
// GaeaRollbackRecord（VersionTimeline kind "docx" 已支持）。
// 红线：非 work 空间（play）不落证据链；journal 目录不可用/写失败静默
//（对齐 appendPptxEvidence/appendXlsxEvidence 口径）。
func appendDocxEvidence(rel, tool, beforeSummary, afterSummary, baseline string) {
	if gaeaEffectiveSpace() != "work" {
		return
	}
	st, err := evidence.OpenJournal(filepath.Join(gaeaCwd(), ".gaea", "work", "journal"))
	if err != nil {
		return
	}
	sid := ""
	if c := gaeaCtrl(); c != nil {
		sid = c.SessionPath()
	}
	if sid == "" {
		sid = "unsaved"
	}
	_ = st.Append(evidence.ChangeRecord{
		SessionID:     sid,
		Space:         "work",
		Tool:          tool,
		Target:        rel,
		BeforeSummary: beforeSummary,
		AfterSummary:  afterSummary,
		BaselinePath:  baseline,
		Status:        evidence.StatusPendingVerify,
	})
}
