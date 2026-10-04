package xlsxedit

// apply_one.go — applyOne 的表驱动实现体（2026-10-04 自 253 行 switch 逐字
// 搬迁，配方照 internal/schedule/ops.go opHandlers 先例：契约先行→表驱动→
// 零编辑复跑；summary 回执文案由 apply_one_test.go 冻结）。

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// opHandlers 把 Op.Type 映射到对应 apply 函数：键集由
// TestOpHandlersKeySetLocked 冻结，新增 op 类型须注册于此并补 summary 契约例。
var opHandlers = map[string]func(*excelize.File, Op) (string, error){
	"set_formula":   applySetFormula,
	"set_value":     applySetValue,
	"fill_range":    applyFillRange,
	"transform":     applyTransform,
	"replace":       applyReplace,
	"split_column":  applySplitColumn,
	"clean":         applyClean,
	"set_style":     applySetStyle,
	"merge_cells":   applyMergeCells,
	"unmerge_cells": applyUnmergeCells,
	"set_col_width": applySetColWidth,
}

func applySetFormula(f *excelize.File, op Op) (string, error) {
	if op.Target == "" || op.Formula == "" {
		return "", fmt.Errorf("target/formula 缺失")
	}
	if err := f.SetCellFormula(op.Sheet, op.Target, strings.TrimPrefix(op.Formula, "=")); err != nil {
		return "", err
	}
	return fmt.Sprintf("写入公式 %s=%s", op.Target, op.Formula), nil
}

func applySetValue(f *excelize.File, op Op) (string, error) {
	if op.Target == "" {
		return "", fmt.Errorf("target 缺失")
	}
	if err := f.SetCellValue(op.Sheet, op.Target, op.Value); err != nil {
		return "", err
	}
	return fmt.Sprintf("写入值 %s=%v", op.Target, op.Value), nil
}

func applyFillRange(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	n := 0
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			if err := f.SetCellValue(op.Sheet, axis, op.Value); err != nil {
				return "", err
			}
			if n++; n > maxCellsPerOp {
				return "", fmt.Errorf("填充区域过大（超过 %d 格）", maxCellsPerOp)
			}
		}
	}
	return fmt.Sprintf("填充 %s = %v", op.Range, op.Value), nil
}

func applyTransform(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	if op.Formula == "" {
		return "", fmt.Errorf("formula 缺失")
	}
	if coords[1] != coords[3] {
		return "", fmt.Errorf("transform 仅支持单列区域（列方向公式调整暂不支持）")
	}
	n := 0
	for r := coords[0]; r <= coords[2]; r++ {
		axis, _ := excelize.CoordinatesToCellName(coords[1], r)
		formula := adjustRowRefs(op.Formula, r-coords[0])
		if err := f.SetCellFormula(op.Sheet, axis, strings.TrimPrefix(formula, "=")); err != nil {
			return "", err
		}
		if n++; n > maxCellsPerOp {
			return "", fmt.Errorf("变换区域过大（超过 %d 格）", maxCellsPerOp)
		}
	}
	return fmt.Sprintf("逐行公式 %s（%d 行）", op.Range, coords[2]-coords[0]+1), nil
}

func applyReplace(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	count := 0
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			val, err := f.GetCellValue(op.Sheet, axis)
			if err != nil || val == "" {
				continue
			}
			if strings.Contains(val, op.Find) {
				if err := f.SetCellValue(op.Sheet, axis, strings.ReplaceAll(val, op.Find, op.Replace)); err != nil {
					return "", err
				}
				count++
			}
		}
	}
	return fmt.Sprintf("替换 %s：%q → %q（%d 格）", op.Range, op.Find, op.Replace, count), nil
}

func applySplitColumn(f *excelize.File, op Op) (string, error) {
	if op.Col == "" || op.Sep == "" || len(op.NewCols) == 0 {
		return "", fmt.Errorf("col/sep/newCols 缺失")
	}
	colNum, err := excelize.ColumnNameToNumber(op.Col)
	if err != nil {
		return "", err
	}
	if len(op.Headers) > 0 {
		for i, h := range op.Headers {
			if i >= len(op.NewCols) {
				break
			}
			if err := f.SetCellValue(op.Sheet, op.NewCols[i]+"1", h); err != nil {
				return "", err
			}
		}
	}
	rows, err := f.GetRows(op.Sheet)
	if err != nil {
		return "", err
	}
	count := 0
	for ri := 1; ri < len(rows); ri++ {
		row := rows[ri]
		if colNum-1 >= len(row) {
			continue
		}
		parts := strings.Split(row[colNum-1], op.Sep)
		for i, part := range parts {
			if i >= len(op.NewCols) {
				break
			}
			axis := op.NewCols[i] + strconv.Itoa(ri+1)
			if err := f.SetCellValue(op.Sheet, axis, strings.TrimSpace(part)); err != nil {
				return "", err
			}
			count++
		}
	}
	return fmt.Sprintf("拆分 %s 列 → %s（%d 格）", op.Col, strings.Join(op.NewCols, ","), count), nil
}

func applyClean(f *excelize.File, op Op) (string, error) {
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	count := 0
	for r := coords[0]; r <= coords[2]; r++ {
		for c := coords[1]; c <= coords[3]; c++ {
			axis, _ := excelize.CoordinatesToCellName(c, r)
			val, err := f.GetCellValue(op.Sheet, axis)
			if err != nil || val == "" {
				continue
			}
			orig := val
			if op.Trim {
				val = strings.TrimSpace(val)
			}
			if op.Upper {
				val = strings.ToUpper(val)
			}
			if op.Lower {
				val = strings.ToLower(val)
			}
			if val != orig {
				if err := f.SetCellValue(op.Sheet, axis, val); err != nil {
					return "", err
				}
				count++
			}
		}
	}
	return fmt.Sprintf("清洗 %s（%d 格）", op.Range, count), nil
}

func applySetStyle(f *excelize.File, op Op) (string, error) {
	if op.Style == nil {
		return "", fmt.Errorf("style 缺失")
	}
	var cells []string
	if op.Target != "" {
		cells = []string{op.Target}
	} else if op.Range != "" {
		coords, err := parseRange(op.Range)
		if err != nil {
			return "", err
		}
		for r := coords[0]; r <= coords[2]; r++ {
			for c := coords[1]; c <= coords[3]; c++ {
				axis, _ := excelize.CoordinatesToCellName(c, r)
				cells = append(cells, axis)
			}
		}
		if len(cells) > maxCellsPerOp {
			return "", fmt.Errorf("样式区域过大（超过 %d 格）", maxCellsPerOp)
		}
	} else {
		return "", fmt.Errorf("target/range 缺失")
	}
	// 旧样式ID → 叠加后样式ID 的缓存：区域里相同起点的样式只新建一次
	styleCache := map[int]int{}
	for _, axis := range cells {
		oldID, err := f.GetCellStyle(op.Sheet, axis)
		if err != nil {
			oldID = 0
		}
		nid, ok := styleCache[oldID]
		if !ok {
			nid, err = mergeStyle(f, oldID, op.Style)
			if err != nil {
				return "", err
			}
			styleCache[oldID] = nid
		}
		if err := f.SetCellStyle(op.Sheet, axis, axis, nid); err != nil {
			return "", err
		}
	}
	scope := op.Range
	if op.Target != "" {
		scope = op.Target
	}
	return fmt.Sprintf("设置样式 %s（%d 格）", scope, len(cells)), nil
}

func applyMergeCells(f *excelize.File, op Op) (string, error) {
	if op.Range == "" {
		return "", fmt.Errorf("range 缺失")
	}
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	tl, _ := excelize.CoordinatesToCellName(coords[1], coords[0])
	br, _ := excelize.CoordinatesToCellName(coords[3], coords[2])
	if err := f.MergeCell(op.Sheet, tl, br); err != nil {
		return "", err
	}
	return fmt.Sprintf("合并 %s", op.Range), nil
}

func applyUnmergeCells(f *excelize.File, op Op) (string, error) {
	if op.Range == "" {
		return "", fmt.Errorf("range 缺失")
	}
	coords, err := parseRange(op.Range)
	if err != nil {
		return "", err
	}
	tl, _ := excelize.CoordinatesToCellName(coords[1], coords[0])
	br, _ := excelize.CoordinatesToCellName(coords[3], coords[2])
	if err := f.UnmergeCell(op.Sheet, tl, br); err != nil {
		return "", err
	}
	return fmt.Sprintf("取消合并 %s", op.Range), nil
}

func applySetColWidth(f *excelize.File, op Op) (string, error) {
	if op.Col == "" || op.Width <= 0 {
		return "", fmt.Errorf("col/width 缺失或非法")
	}
	if err := f.SetColWidth(op.Sheet, op.Col, op.Col, op.Width); err != nil {
		return "", err
	}
	return fmt.Sprintf("列宽 %s = %.1f", op.Col, op.Width), nil
}
