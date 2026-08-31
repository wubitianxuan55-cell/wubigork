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
	"github.com/gaea/gaea/internal/office/docmd"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/xuri/excelize/v2"
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

// v4.6 Verifier 通道 B 渲染/比对 seam（测试注入）：
//   - verifyConvertToPdf：soffice 无头把 xlsx/docx/txt 转 PDF；
//   - verifyRenderPages：poppler pdftoppm 把 PDF 逐页渲染 PNG；
//   - verifyPixelDiff：纯 Go 像素差异率。
var (
	verifyConvertToPdf = convertToPdfFile
	verifyRenderPages  = docmd.RenderPDFPages
	verifyPixelDiff    = pixelDiffRatio
)

// visualDiffThresholds 是通道 B 判定阈值（像素差异率，0..1）：
//   - ≤ passThreshold：视觉一致（仅渲染噪声/微小变化）→ pass；
//   - ≤ warnThreshold：视觉变化但页数未变（内容/数据大改，预期内）→ warn；
//   - > warnThreshold 且页数变化：视觉大改 + 版式变化（A 通道无法解释的
//     结构破坏信号）→ fail。
const (
	visualDiffPassThreshold = 0.02
	visualDiffWarnThreshold = 0.20
)

// runVisualDiff 对基线/目标做真视觉 diff（v4.6 补课：页数对比 → 像素对比 +
// 页数联合判定）。verifyDir 是审计产物目录（before/after PDF + 逐页 PNG 落盘，
// 供事后人工复核差异页）。返回通道 B 文案、建议状态、像素差异率（0-1，渲染
// 降级时 0）与渲染页数（before/after 较大者；降级时 0）——v4.16 起 ratio/pages
// 随 verdict 结构化回填，前端「视觉复核」行直接展示。
func runVisualDiff(baselinePath, target, verifyDir string) (msg string, status string, ratio float64, pages int) {
	beforePDF := filepath.Join(verifyDir, "before.pdf")
	afterPDF := filepath.Join(verifyDir, "after.pdf")
	if err := verifyConvertToPdf(baselinePath, beforePDF); err != nil {
		return "warn: 视觉渲染降级（soffice 不可用或转换失败，仅结构复核）", "warn", 0, 0
	}
	if err := verifyConvertToPdf(target, afterPDF); err != nil {
		return "warn: 视觉渲染降级（soffice 不可用或转换失败，仅结构复核）", "warn", 0, 0
	}
	bp, berr := verifyRenderPages(beforePDF, filepath.Join(verifyDir, "before"), 0)
	ap, aerr := verifyRenderPages(afterPDF, filepath.Join(verifyDir, "after"), 0)
	if berr != nil || aerr != nil {
		return "warn: 视觉渲染降级（poppler pdftoppm 不可用，仅结构复核）", "warn", 0, 0
	}

	maxPages := max(len(bp), len(ap))
	totalRatio := 0.0
	changedPages := 0
	for i := 0; i < maxPages; i++ {
		var pageRatio float64
		if i >= len(bp) || i >= len(ap) {
			pageRatio = 1.0 // 页数变化：缺失页整页视为差异
		} else {
			r, err := verifyPixelDiff(bp[i], ap[i])
			if err != nil {
				return "warn: 视觉 diff 像素解析失败（" + err.Error() + "）", "warn", 0, 0
			}
			pageRatio = r
		}
		totalRatio += pageRatio
		if pageRatio > visualDiffPassThreshold {
			changedPages++
		}
	}
	avg := totalRatio / float64(maxPages)
	pageShift := len(bp) != len(ap)
	switch {
	case avg <= visualDiffPassThreshold:
		return fmt.Sprintf("pass: 渲染健全（%d 页，像素差异 %.1f%%）", maxPages, avg*100), "pass", avg, maxPages
	case pageShift && avg > visualDiffWarnThreshold:
		return fmt.Sprintf("fail: 视觉大改（%d→%d 页，%d 页差异，像素差异 %.1f%%）——建议回滚或人工复核",
			len(bp), len(ap), changedPages, avg*100), "fail", avg, maxPages
	default:
		return fmt.Sprintf("warn: 视觉变化（%d 页中 %d 页差异，像素差异 %.1f%%）",
			maxPages, changedPages, avg*100), "warn", avg, maxPages
	}
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
			// v4.9.1 引用级深化：重算零错误之上，回读工作簿逐条核对证据卡
			// 声明（opsJson）——「声明了什么就核对什么」；fail 前缀升级整条。
			f, ferr := excelize.OpenFile(target, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
			if ferr != nil {
				v.ChannelA = "pass: 文件可打开，公式重算零错误（引用级复核无法打开工作簿，跳过声明比对）"
				break
			}
			deep := verifyXlsxChannelADeep(f, rec)
			f.Close()
			if strings.HasPrefix(deep, "；fail:") {
				v.ChannelA = "fail: 公式重算零错误，" + strings.TrimPrefix(deep, "；")
			} else {
				v.ChannelA = "pass: 文件可打开，公式重算零错误" + deep
			}
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

	// ── 通道 B：真视觉 diff（v4.6 补课：页数对比 → 像素 diff + 页数联合
	// 判定）。有基线快照的写盘记录（xlsx_apply / edit_file / write_file /
	// multi_edit / edit_lines / move_file 等）都可复核——soffice 转 PDF +
	// poppler 逐页渲染 + 像素差异率；渲染链路任一环不可用降级 warn（仅结构
	// 复核），不误判失败。审计产物（before/after PDF 与逐页 PNG）落
	// .gaea/work/journal/verify/<id>/，事后可人工查差异页。──
	v.ChannelB = "n/a"
	verifyArtifacts := ""
	if rec.BaselinePath != "" && v.ChannelA != "fail" {
		if _, err := os.Stat(rec.BaselinePath); err == nil {
			verifyDir := filepath.Join(gaeaCwd(), ".gaea", "work", "journal", "verify", id)
			_ = os.MkdirAll(verifyDir, 0o755)
			msg, _, ratio, pages := runVisualDiff(rec.BaselinePath, target, verifyDir)
			v.ChannelB = msg
			// v4.16 通道 B 结果产品化：像素差异率/渲染页数/产物目录随 verdict
			// 结构化返回（前端「视觉复核」行直接展示；渲染降级时 ratio=0 经
			// omitempty 省略，前端不渲染该行）
			v.ChannelBRatio = ratio
			v.ChannelBPages = pages
			v.ChannelBArtifacts = filepath.ToSlash(verifyDir)
			// 审计产物路径随 verdict 落库（事后人工复核差异页）
			verifyArtifacts = fmt.Sprintf("视觉产物：%s", filepath.ToSlash(verifyDir))
		}
	}

	// ── 汇总 verdict ──
	switch {
	case strings.HasPrefix(v.ChannelA, "fail"):
		v.Status = evidence.VerdictFailed
		v.Note = "结构/引用完整性未通过——建议回滚；xlsx 可回滚后重新规划（办公面板）"
	case strings.HasPrefix(v.ChannelB, "fail"):
		v.Status = evidence.VerdictFailed
		v.Note = "视觉大改且版式变化——建议回滚后重新规划；人工复核为准"
	case strings.HasPrefix(v.ChannelB, "warn"):
		v.Status = evidence.VerdictWarned
		v.Note = "结构通过，视觉/版式存在变化（或渲染降级）"
	default:
		v.Status = evidence.VerdictVerified
		v.Note = "双通道复核通过"
	}
	if verifyArtifacts != "" {
		v.Note += "｜" + verifyArtifacts
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
