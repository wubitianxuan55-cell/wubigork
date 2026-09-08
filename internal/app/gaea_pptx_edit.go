package app

// GaeaPptxApplyEdit — pptx 真编辑刀1 数据层绑定（docs/gaea-pptx-edit-design-2026-09.md
// §4 刀1；选型 A=Go 自研 pptxedit 同构 docxedit，范式 P1 对齐 xlsx_apply）：
// 大纲/agent 给「页码 + 目标文本 + 替换文本」→ run 级直接替换（rPr 原字节继承）
// → 落盘前快照 + Journal 证据卡 → 返回新预览。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/pptxedit"
)

func (a *App) GaeaPptxApplyEdit(rel string, slideIdx int, target string, replacement string) (PreviewResult, error) {
	if rel == "" {
		return PreviewResult{}, fmt.Errorf("缺少文件路径")
	}
	if slideIdx < 1 {
		return PreviewResult{}, fmt.Errorf("页码从 1 起")
	}
	if strings.TrimSpace(target) == "" {
		return PreviewResult{}, fmt.Errorf("替换目标为空")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return PreviewResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	// 落盘前基线快照（同 xlsx_apply 口径；Verifier 通道 B 对有快照的写盘记录复核）
	baseline := ""
	if raw, rerr := os.ReadFile(path); rerr == nil {
		rbDir := filepath.Join(gaeaCwd(), ".gaea", "work", "rollback")
		_ = os.MkdirAll(rbDir, 0o755)
		bp := filepath.Join(rbDir, fmt.Sprintf("pptx-%d.before", time.Now().UnixNano()))
		if werr := os.WriteFile(bp, raw, 0o644); werr == nil {
			baseline = bp
		}
	}
	summary, err := pptxedit.ApplyTextReplace(path, slideIdx, target, replacement)
	if err != nil {
		return PreviewResult{}, err
	}
	appendPptxEvidence(rel, slideIdx, target, replacement, summary, baseline)
	return a.GaeaPreview(rel), nil
}

// GaeaPptxSlideText — pptx 真编辑刀2 编辑面绑定（docs/gaea-pptx-edit-design-2026-09.md
// §4 刀2 的后端增量）：大纲 GaeaPptxOutline 的 texts 为 200 rune 截断预览
//（附省略号），而 GaeaPptxApplyEdit 是精确匹配——拿截断文本当替换 target
// 必匹配失败拒写（宁拒不误改口径）；本绑定供编辑面板取段落全文作为替换
// target。走 pptxedit.LocateText（纯 Go 零 python）：ApplyTextReplace 匹配
// 单段落，此处段落粒度即 target 粒度；页码序 = presentation.xml sldIdLst，
// 与大纲 Index 一致。路径解析与 GaeaPptxOutline 同款（resolvePreviewPath：
// Join/Clean 防穿越 + 裸文件名回退常见输出目录）；仅接受 .pptx。
func (a *App) GaeaPptxSlideText(rel string) ([]pptxedit.SlideText, error) {
	path, _ := resolvePreviewPath(rel)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("文件不存在：%s", rel)
	}
	if ext := strings.ToLower(filepath.Ext(path)); ext != ".pptx" {
		return nil, fmt.Errorf("仅支持 .pptx 文本提取（收到 %s；.ppt 旧格式请先另存为 .pptx）", ext)
	}
	return pptxedit.LocateText(path)
}

// appendPptxEvidence 把一次 pptx 编辑写入 work 空间 Journal（JSONL）。
// 红线：非 work 空间（play）不落证据链；journal 目录不可用/写失败静默
//（对齐 appendXlsxEvidence 口径）。
func appendPptxEvidence(rel string, slideIdx int, target, replacement, summary, baseline string) {
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
		Tool:          "pptx_apply",
		Target:        rel,
		BeforeSummary: fmt.Sprintf("p%d %q", slideIdx, target),
		AfterSummary:  fmt.Sprintf("p%d → %q（%s）", slideIdx, replacement, summary),
		BaselinePath:  baseline,
		Status:        evidence.StatusPendingVerify,
	})
}
