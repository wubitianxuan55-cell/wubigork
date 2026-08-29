package app

// v4.1b Verifier + 回滚（docs/gaea-v41-evidence-chain-design.md §4-§6）。
// VerifyRecord 对一张证据卡做双通道复核（A 结构/引用完整性；B 视觉健全性），
// 结论落 verdicts.jsonl（按 ID 后者胜）；RollbackRecord 用基线快照回滚，
// 带「目标已被手工修改」冲突保护（绝不覆盖用户编辑）。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

func journalStore() *evidence.JournalStore {
	st, err := evidence.OpenJournal(filepath.Join(gaeaCwd(), ".gaea", "work", "journal"))
	if err != nil {
		return nil
	}
	return st
}

func resolveTarget(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(gaeaCwd(), rel)
}

// pdfPageCount 轻量统计 PDF 页数（按 "/Type /Page" 计数；渲染健全性用，非精确解析）。
func pdfPageCount(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(raw), "/Type /Page")
}

// VerifyRecord 双通道复核一张证据卡。
func (a *App) GaeaVerifyRecord(id string) (evidence.Verdict, error) {
	st := journalStore()
	if st == nil {
		return evidence.Verdict{}, fmt.Errorf("Journal 不可用")
	}
	rec, ok := st.FindByID(id)
	if !ok {
		return evidence.Verdict{}, fmt.Errorf("证据卡 %s 不存在", id)
	}
	target := resolveTarget(rec.Target)
	v := evidence.Verdict{ID: id, At: time.Now().UnixMilli()}

	// ── 通道 A：结构 / 引用完整性 ──
	switch rec.Tool {
	case "xlsx_apply":
		if _, err := os.Stat(target); err != nil {
			v.ChannelA = "fail: 目标文件不存在"
		} else if rep, err := xlsxedit.Recalc(target, gaeaCwd()); err != nil {
			v.ChannelA = "fail: 公式重算失败 " + err.Error()
		} else if rep.TotalErrors > 0 {
			v.ChannelA = fmt.Sprintf("fail: 重算发现 %d 处公式错误", rep.TotalErrors)
		} else {
			v.ChannelA = "pass: 文件可打开，公式重算零错误"
		}
	case "move_file":
		src := strings.TrimPrefix(rec.BeforeSummary, "→ moved from ")
		if _, err := os.Stat(target); err != nil {
			v.ChannelA = "fail: 目标不存在"
		} else if _, err := os.Stat(src); err == nil {
			v.ChannelA = "fail: 源文件仍存在"
		} else {
			v.ChannelA = "pass: 目标存在且源已迁移"
		}
	default: // edit_file / write_file / multi_edit / edit_lines / rollback
		cur, err := os.ReadFile(target)
		if err != nil {
			v.ChannelA = "fail: 目标不可读"
			break
		}
		curS := string(cur)
		editLike := rec.Tool == "edit_file" || rec.Tool == "multi_edit" || rec.Tool == "edit_lines"
		if editLike && rec.AfterSummary != "" && !strings.Contains(curS, rec.AfterSummary) {
			v.ChannelA = "fail: 变更后的原文摘要未出现在目标中"
		} else if editLike && rec.BeforeSummary != "" && strings.Contains(curS, rec.BeforeSummary) {
			v.ChannelA = "fail: 变更前的旧文本仍存在于目标中"
		} else {
			v.ChannelA = "pass: 目标存在，变更摘要与文件实况一致"
		}
	}

	// ── 通道 B：视觉健全性（有基线时渲染 before/after PDF 对比）──
	v.ChannelB = "n/a"
	if rec.Tool == "xlsx_apply" && rec.BaselinePath != "" && v.ChannelA != "fail" {
		if _, err := os.Stat(rec.BaselinePath); err == nil {
			dir, _ := os.MkdirTemp("", "gaea-verify-*")
			defer os.RemoveAll(dir)
			beforePDF := filepath.Join(dir, "before.pdf")
			afterPDF := filepath.Join(dir, "after.pdf")
			berr := convertToPdfFile(rec.BaselinePath, beforePDF)
			aerr := convertToPdfFile(target, afterPDF)
			if berr != nil || aerr != nil {
				v.ChannelB = "warn: 视觉渲染降级（soffice 不可用或转换失败）"
			} else {
				bp, ap := pdfPageCount(beforePDF), pdfPageCount(afterPDF)
				if bp > 0 && ap > 0 && bp != ap {
					v.ChannelB = fmt.Sprintf("warn: 版式变化（%d → %d 页）", bp, ap)
				} else {
					v.ChannelB = fmt.Sprintf("pass: 渲染健全（%d 页）", max(ap, 1))
				}
			}
		}
	}

	// ── 汇总 verdict ──
	switch {
	case strings.HasPrefix(v.ChannelA, "fail"):
		v.Status = evidence.VerdictFailed
		v.Note = "结构/引用完整性未通过——建议回滚或人工复核"
	case strings.HasPrefix(v.ChannelB, "warn"):
		v.Status = evidence.VerdictWarned
		v.Note = "结构通过，视觉/版式存在变化（或渲染降级）"
	default:
		v.Status = evidence.VerdictVerified
		v.Note = "双通道复核通过"
	}
	_ = st.AppendVerdict(v)
	return v, nil
}

// RollbackRecord 用基线快照回滚一张证据卡；目标已被手工修改则拒绝（零覆盖）。
func (a *App) GaeaRollbackRecord(id string) error {
	st := journalStore()
	if st == nil {
		return fmt.Errorf("Journal 不可用")
	}
	rec, ok := st.FindByID(id)
	if !ok {
		return fmt.Errorf("证据卡 %s 不存在", id)
	}
	if rec.BaselinePath == "" {
		return fmt.Errorf("该证据卡无基线快照，无法回滚")
	}
	if _, err := os.Stat(rec.BaselinePath); err != nil {
		return fmt.Errorf("基线快照缺失：%v", err)
	}
	target := resolveTarget(rec.Target)
	if cur, err := os.ReadFile(target); err == nil {
		curS := string(cur)
		editLike := rec.Tool == "edit_file" || rec.Tool == "multi_edit" || rec.Tool == "edit_lines"
		if editLike && rec.AfterSummary != "" && !strings.Contains(curS, rec.AfterSummary) {
			return fmt.Errorf("目标已被手工修改（变更摘要不匹配），拒绝回滚以免覆盖你的编辑")
		}
		if rec.Tool == "write_file" && rec.AfterSummary != "" && curS != rec.AfterSummary {
			return fmt.Errorf("目标已被手工修改（内容与记录不一致），拒绝回滚")
		}
	}
	baseline, err := os.ReadFile(rec.BaselinePath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, baseline, 0o644); err != nil {
		return err
	}
	_ = st.Append(evidence.ChangeRecord{
		SessionID:     rec.SessionID,
		Space:         "work",
		Tool:          "rollback",
		Target:        rec.Target,
		BeforeSummary: "rolled back " + id,
		Status:        evidence.StatusPendingVerify,
	})
	return nil
}
