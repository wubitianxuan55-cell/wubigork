package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/evidence"
	"github.com/gaea/gaea/internal/office/xlsxedit"
	"github.com/gaea/gaea/internal/office/xlsxpreview"
	"github.com/gaea/gaea/internal/util"
	"github.com/xuri/excelize/v2"
)

// XlsxEditResult 是单元格编辑结果：更新后的预览 + 应用摘要。
type XlsxEditResult struct {
	Preview string `json:"preview"` // xlsxpreview JSON（前端重渲染）
	Summary string `json:"summary"`
	Applied int    `json:"applied"`
}

// XlsxPlanResult 是「先规划后应用」的规划结果：ops 原样带回（应用时透传），
// 附单元格级变更清单供用户审阅批准（对标 Copilot Plan/Show Changes 范式）。
type XlsxPlanResult struct {
	Ops       string            `json:"ops"`     // 操作集 JSON（GaeaXlsxApplyEdit 透传）
	Summary   string            `json:"summary"` // 操作描述（分号连接）
	Changes   []xlsxedit.Change `json:"changes"` // 变更清单（截断到上限）
	Total     int               `json:"total"`   // 已确认变更格数（读取预算内，截断时为下界）
	Truncated bool              `json:"truncated"`
}

// GaeaXlsxPlanEdit 单元格操作规划（不落盘）：上下文 → AI 规划操作 →
// 临时副本试运行（合法性与真实摘要）→ 与原文件 diff 出变更清单。
// 返回的 ops 原样带回，由 GaeaXlsxApplyEdit 在用户批准后执行。
func (a *App) GaeaXlsxPlanEdit(rel, sheet, instruction, selection string) (XlsxPlanResult, error) {
	if rel == "" {
		return XlsxPlanResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return XlsxPlanResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	if sheet == "" {
		sheet = firstSheetName(path)
	}
	if instruction == "" {
		return XlsxPlanResult{}, fmt.Errorf("编辑指令为空")
	}
	if a.client == nil {
		return XlsxPlanResult{}, fmt.Errorf("AI 客户端未初始化")
	}

	ctxJSON, err := xlsxedit.BuildContext(path, sheet)
	if err != nil {
		return XlsxPlanResult{}, err
	}

	// 2026-08-28 本地优先强化：Excel AI 编辑属办公功能级调用，优先本地 Herdsman。
	featEng, featModel, _ := a.routeOfficeLocal("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reply, err := a.client.XlsxEditOps(ctx, featEng, featModel, ctxJSON, selection, instruction)
	if err != nil {
		return XlsxPlanResult{}, err
	}
	var ops []xlsxedit.Op
	if err := json.Unmarshal([]byte(util.ExtractJSON(reply)), &ops); err != nil || len(ops) == 0 {
		return XlsxPlanResult{}, fmt.Errorf("AI 未返回有效操作，请换个说法重试")
	}

	summary, changes, total, truncated, err := xlsxedit.PlanOps(path, ops)
	if err != nil {
		return XlsxPlanResult{}, err
	}
	opsJSON, err := json.Marshal(ops)
	if err != nil {
		return XlsxPlanResult{}, err
	}
	return XlsxPlanResult{
		Ops:       string(opsJSON),
		Summary:   strings.Join(summary, "；"),
		Changes:   changes,
		Total:     total,
		Truncated: truncated,
	}, nil
}

// GaeaXlsxApplyEdit 应用已批准的操作集（GaeaXlsxPlanEdit 的产物原样透传）：
// excelize 执行 → LibreOffice 重算公式 → 返回更新预览。
func (a *App) GaeaXlsxApplyEdit(rel, opsJSON string) (XlsxEditResult, error) {
	if rel == "" {
		return XlsxEditResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return XlsxEditResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	if strings.TrimSpace(opsJSON) == "" {
		return XlsxEditResult{}, fmt.Errorf("操作集为空，请重新规划")
	}
	var ops []xlsxedit.Op
	if err := json.Unmarshal([]byte(opsJSON), &ops); err != nil || len(ops) == 0 {
		return XlsxEditResult{}, fmt.Errorf("操作集无效，请重新规划")
	}

	// v4.1b：应用前整文件基线快照（Verifier 通道 B / 回滚原料）。
	baseline := ""
	if raw, rerr := os.ReadFile(path); rerr == nil {
		rbDir := filepath.Join(gaeaCwd(), ".gaea", "work", "rollback")
		_ = os.MkdirAll(rbDir, 0o755)
		bp := filepath.Join(rbDir, fmt.Sprintf("xlsx-%d.before", time.Now().UnixNano()))
		if werr := os.WriteFile(bp, raw, 0o644); werr == nil {
			baseline = bp
		}
	}
	summary, err := xlsxedit.ApplyOps(path, ops)
	if err != nil {
		return XlsxEditResult{}, err
	}
	// v4.1 证据链：xlsx 应用后记录证据卡（Target=工作区相对路径；Before 为操作
	// 载荷摘要，After 为应用摘要；单元格级 old/new 由 Verifier 对基线快照复核）。
	appendXlsxEvidence(rel, ops, summary, baseline)
	recalcNote := ""
	if rep, rerr := xlsxedit.Recalc(path, gaeaCwd()); rerr == nil {
		if rep.TotalErrors > 0 {
			recalcNote = fmt.Sprintf("；重算发现 %d 处公式错误（%s）", rep.TotalErrors, rep.Status)
		}
	} else if strings.Contains(rerr.Error(), "未找到 recalc.py") {
		recalcNote = "；公式重算跳过（未找到 recalc.py）"
	}
	preview, err := xlsxpreview.Render(path)
	if err != nil {
		return XlsxEditResult{}, err
	}
	return XlsxEditResult{
		Preview: preview,
		Summary: strings.Join(summary, "；") + recalcNote,
		Applied: len(ops),
	}, nil
}

// appendXlsxEvidence 把一次 xlsx 应用写入 work 空间 Journal（JSONL）。
// 红线：非 work 空间（play）不落证据链；journal 目录不可用/写失败静默。
func appendXlsxEvidence(rel string, ops []xlsxedit.Op, summary []string, baseline string) {
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
	var before strings.Builder
	for _, op := range ops {
		fmt.Fprintf(&before, "%s!%s %s=%v", op.Sheet, op.Target, op.Type, op.Value)
		if op.Find != "" || op.Replace != "" {
			fmt.Fprintf(&before, " find=%q replace=%q", op.Find, op.Replace)
		}
		before.WriteString("; ")
	}
	// opsJson 随卡落盘（v4.9.1 通道 A 引用级「声明↔实况」比对原料）。截断会
	// 把 JSON 切坏——超限干脆不落（空 = 引用级声明比对干净跳过，宁漏勿误）。
	opsJSON := ""
	if b, err := json.Marshal(ops); err == nil && len(b) <= evidence.SummaryLimit {
		opsJSON = string(b)
	}
	_ = st.Append(evidence.ChangeRecord{
		SessionID:     sid,
		Space:         "work",
		Tool:          "xlsx_apply",
		Target:        rel,
		BeforeSummary: strings.TrimSpace(before.String()),
		AfterSummary:  strings.Join(summary, "；"),
		BaselinePath:  baseline,
		OpsJSON:       opsJSON,
		Status:        evidence.StatusPendingVerify,
	})
}

func firstSheetName(path string) string {
	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return ""
	}
	defer f.Close()
	if list := f.GetSheetList(); len(list) > 0 {
		return list[0]
	}
	return ""
}

// GaeaXlsxSetCell 直接写单元格（Excel 式双击编辑）：写入值或公式 →
// LibreOffice 重算 → 返回更新预览。
func (a *App) GaeaXlsxSetCell(rel, sheet, ref, value string) (XlsxEditResult, error) {
	if rel == "" || ref == "" {
		return XlsxEditResult{}, fmt.Errorf("缺少文件路径或单元格")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return XlsxEditResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	if _, _, err := excelize.CellNameToCoordinates(ref); err != nil {
		return XlsxEditResult{}, fmt.Errorf("无效单元格引用：%s", ref)
	}
	if sheet == "" {
		sheet = firstSheetName(path)
	}

	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return XlsxEditResult{}, fmt.Errorf("打开文件失败：%w", err)
	}
	defer f.Close()

	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "=") {
		if err := f.SetCellFormula(sheet, ref, strings.TrimPrefix(trimmed, "=")); err != nil {
			return XlsxEditResult{}, fmt.Errorf("写入公式失败：%w", err)
		}
	} else if trimmed == "" {
		if err := f.SetCellValue(sheet, ref, ""); err != nil {
			return XlsxEditResult{}, fmt.Errorf("清空单元格失败：%w", err)
		}
	} else if n, perr := strconv.ParseFloat(trimmed, 64); perr == nil {
		// 纯数字按数值写入，保持可计算
		if err := f.SetCellValue(sheet, ref, n); err != nil {
			return XlsxEditResult{}, fmt.Errorf("写入单元格失败：%w", err)
		}
	} else {
		if err := f.SetCellValue(sheet, ref, value); err != nil {
			return XlsxEditResult{}, fmt.Errorf("写入单元格失败：%w", err)
		}
	}
	if err := f.Save(); err != nil {
		return XlsxEditResult{}, fmt.Errorf("保存文件失败：%w", err)
	}

	return a.renderXlsxAfterChange(path, fmt.Sprintf("已更新 %s!%s", sheet, ref), 1)
}

// GaeaXlsxRecalc 手动重算全部公式（LibreOffice）并返回更新预览。
// 预览自动重算兜底之外，用户也可以主动点“重算”刷新结果。
func (a *App) GaeaXlsxRecalc(rel string) (XlsxEditResult, error) {
	if rel == "" {
		return XlsxEditResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return XlsxEditResult{}, fmt.Errorf("文件不存在：%s", rel)
	}

	rep, rerr := xlsxedit.Recalc(path, gaeaCwd())
	if rerr != nil {
		return XlsxEditResult{}, fmt.Errorf("公式重算失败：%w", rerr)
	}
	return a.renderXlsxAfterChange(path, fmt.Sprintf("已重算 %d 个公式", rep.TotalFormulas), rep.TotalFormulas)
}

// GaeaXlsxRowOps 行级操作（基于选中单元格所在行）：
//
//	insert_before 在选中行上方插入空行；insert_after 在下方插入；delete 删除该行。
//
// 同表公式/合并区域由 excelize 平移，随后 LibreOffice 重算刷新结果。
func (a *App) GaeaXlsxRowOps(rel, sheet, action, ref string) (XlsxEditResult, error) {
	if rel == "" || ref == "" {
		return XlsxEditResult{}, fmt.Errorf("缺少文件路径或单元格")
	}
	_, row, err := excelize.CellNameToCoordinates(ref)
	if err != nil {
		return XlsxEditResult{}, fmt.Errorf("无效单元格引用：%s", ref)
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return XlsxEditResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	if sheet == "" {
		sheet = firstSheetName(path)
	}

	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return XlsxEditResult{}, fmt.Errorf("打开文件失败：%w", err)
	}
	defer f.Close()

	var summary string
	switch action {
	case "insert_before":
		if err := f.InsertRows(sheet, row, 1); err != nil {
			return XlsxEditResult{}, fmt.Errorf("插入行失败：%w", err)
		}
		summary = fmt.Sprintf("已在第 %d 行上方插入空行", row)
	case "insert_after":
		if err := f.InsertRows(sheet, row+1, 1); err != nil {
			return XlsxEditResult{}, fmt.Errorf("插入行失败：%w", err)
		}
		summary = fmt.Sprintf("已在第 %d 行下方插入空行", row)
	case "delete":
		if err := f.RemoveRow(sheet, row); err != nil {
			return XlsxEditResult{}, fmt.Errorf("删除行失败：%w", err)
		}
		summary = fmt.Sprintf("已删除第 %d 行", row)
	default:
		return XlsxEditResult{}, fmt.Errorf("不支持的操作：%s", action)
	}
	if err := f.Save(); err != nil {
		return XlsxEditResult{}, fmt.Errorf("保存文件失败：%w", err)
	}
	return a.renderXlsxAfterChange(path, summary, 1)
}

// GaeaXlsxColOps 列级操作（基于选中单元格所在列）：
//
//	insert_before 在选中列左侧插入空列；insert_after 在右侧插入；delete 删除该列。
func (a *App) GaeaXlsxColOps(rel, sheet, action, ref string) (XlsxEditResult, error) {
	if rel == "" || ref == "" {
		return XlsxEditResult{}, fmt.Errorf("缺少文件路径或单元格")
	}
	colName, _, err := excelize.SplitCellName(ref)
	if err != nil || colName == "" {
		return XlsxEditResult{}, fmt.Errorf("无效单元格引用：%s", ref)
	}
	colNum, err := excelize.ColumnNameToNumber(colName)
	if err != nil {
		return XlsxEditResult{}, fmt.Errorf("无效单元格引用：%s", ref)
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if _, err := os.Stat(path); err != nil {
		return XlsxEditResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	if sheet == "" {
		sheet = firstSheetName(path)
	}

	f, err := excelize.OpenFile(path, excelize.Options{UnzipXMLSizeLimit: 1 << 30})
	if err != nil {
		return XlsxEditResult{}, fmt.Errorf("打开文件失败：%w", err)
	}
	defer f.Close()

	var summary string
	switch action {
	case "insert_before":
		if err := f.InsertCols(sheet, colName, 1); err != nil {
			return XlsxEditResult{}, fmt.Errorf("插入列失败：%w", err)
		}
		summary = fmt.Sprintf("已在 %s 列左侧插入空列", colName)
	case "insert_after":
		next, err := excelize.ColumnNumberToName(colNum + 1)
		if err != nil {
			return XlsxEditResult{}, fmt.Errorf("无效列号：%w", err)
		}
		if err := f.InsertCols(sheet, next, 1); err != nil {
			return XlsxEditResult{}, fmt.Errorf("插入列失败：%w", err)
		}
		summary = fmt.Sprintf("已在 %s 列右侧插入空列", colName)
	case "delete":
		if err := f.RemoveCol(sheet, colName); err != nil {
			return XlsxEditResult{}, fmt.Errorf("删除列失败：%w", err)
		}
		summary = fmt.Sprintf("已删除 %s 列", colName)
	default:
		return XlsxEditResult{}, fmt.Errorf("不支持的操作：%s", action)
	}
	if err := f.Save(); err != nil {
		return XlsxEditResult{}, fmt.Errorf("保存文件失败：%w", err)
	}
	return a.renderXlsxAfterChange(path, summary, 1)
}

// renderXlsxAfterChange 通用收尾：LibreOffice 重算（best-effort）→ 渲染预览。
func (a *App) renderXlsxAfterChange(path, summary string, applied int) (XlsxEditResult, error) {
	recalcNote := ""
	if rep, rerr := xlsxedit.Recalc(path, gaeaCwd()); rerr == nil {
		if rep.TotalErrors > 0 {
			recalcNote = fmt.Sprintf("；重算发现 %d 处公式错误（%s）", rep.TotalErrors, rep.Status)
		}
	} else if strings.Contains(rerr.Error(), "未找到 recalc.py") {
		recalcNote = "；公式重算跳过（未找到 recalc.py）"
	}
	preview, err := xlsxpreview.Render(path)
	if err != nil {
		return XlsxEditResult{}, err
	}
	return XlsxEditResult{Preview: preview, Summary: summary + recalcNote, Applied: applied}, nil
}
