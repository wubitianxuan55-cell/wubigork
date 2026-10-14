package app

// ── 资源价批量维护（三期 3.4）绑定面：Preview→Apply 双段式 ──
//
// Preview=解析+匹配预览（宁缺勿误，跳过行带人可读原因）；Apply=作者确认后
// 整批事务落库（写调价历史+推现行价）。传输=xlsxBase64（进度计划导入先例）。

import (
	"encoding/base64"
	"fmt"

	"github.com/gaea/gaea/internal/gaea/workcost"
)

// GaeaCostBulkPricePreview 解析批量调价 Excel 并预览匹配结果（宁缺勿误，
// 跳过行带人可读原因）。
func (a *App) GaeaCostBulkPricePreview(xlsxBase64 string) (workcost.BulkPricePreview, error) {
	raw, err := decodeBulkPriceXlsx(xlsxBase64)
	if err != nil {
		return workcost.BulkPricePreview{}, err
	}
	rows, perr := workcost.ParseBulkPriceXlsx(raw)
	if perr != nil {
		return workcost.BulkPricePreview{}, perr
	}
	matched, skipped, err := a.hubWorkcostStore().MatchBulkPrices(rows)
	if err != nil {
		return workcost.BulkPricePreview{}, err
	}
	if matched == nil {
		matched = []workcost.BulkPriceMatch{}
	}
	if skipped == nil {
		skipped = []workcost.BulkPriceSkip{}
	}
	return workcost.BulkPricePreview{Total: len(rows), Matched: matched, Skipped: skipped, RowError: []workcost.BulkPriceSkip{}}, nil
}

// GaeaCostBulkPriceApply 作者确认后批量应用（整批事务：全部可匹配行一次落库，
// 中途失败整体回滚=干净重试）。收同一 xlsxBase64——后端重解析+重匹配（幂等：
// 同一文件同一匹配结果），Preview 与 Apply 天然一致，前端不搬运整表。
func (a *App) GaeaCostBulkPriceApply(xlsxBase64 string) (*workcost.BulkPriceResult, error) {
	raw, err := decodeBulkPriceXlsx(xlsxBase64)
	if err != nil {
		return nil, err
	}
	rows, err := workcost.ParseBulkPriceXlsx(raw)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("调价行为空：Excel 里没有数据行")
	}
	return a.hubWorkcostStore().ApplyBulkPrices(rows)
}

// decodeBulkPriceXlsx base64 解码（进度计划导入先例同款口径）。
func decodeBulkPriceXlsx(xlsxBase64 string) ([]byte, error) {
	if xlsxBase64 == "" {
		return nil, fmt.Errorf("Excel 内容为空")
	}
	raw, err := base64.StdEncoding.DecodeString(xlsxBase64)
	if err != nil {
		return nil, fmt.Errorf("Excel 数据解码失败：%w", err)
	}
	return raw, nil
}
