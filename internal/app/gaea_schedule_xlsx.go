package app

// gaea_schedule_xlsx.go — 进度计划 Excel 导入导出绑定（v4.134.0 刀D3）。
//
// 导出：板块当前计划 JSON → 上报口径 xlsx，base64 返回（前端 <a download>
// 触发下载，与导出 XML 同一 UX）；导入：上报 xlsx（base64）→ 计划 JSON
// （前端 importProject 走 normalize+Save 的既有校验链）。解析与渲染在
// internal/schedule/xlsx.go，本文件只做 IO/编码门面。

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/schedule"
)

// GaeaScheduleExportXlsx 计划 JSON → 上报口径 Excel（base64）。
func (a *App) GaeaScheduleExportXlsx(projectJSON string) (string, error) {
	var p schedule.Project
	if err := json.Unmarshal([]byte(projectJSON), &p); err != nil {
		return "", fmt.Errorf("计划 JSON 解析失败：%w", err)
	}
	data, err := schedule.ExportXlsx(p)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// GaeaScheduleImportXlsx 上报 Excel（base64）→ 计划 JSON。
func (a *App) GaeaScheduleImportXlsx(xlsxBase64 string) (string, error) {
	if xlsxBase64 == "" {
		return "", fmt.Errorf("Excel 内容为空")
	}
	raw, err := base64.StdEncoding.DecodeString(xlsxBase64)
	if err != nil {
		return "", fmt.Errorf("Excel 数据解码失败：%w", err)
	}
	p, err := schedule.ImportXlsx(raw)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
