package app

// gaea_schedule_mpp.go — 进度计划 MPP 导入绑定(v4.140.0 刀1)。
//
// 二进制 MS Project 工程(MPP9/12/14 子集,base64)→ 计划 JSON。解析在
// internal/schedule/mpp.go(蒸馏 ProjectLibre MPXJ 布局:无压缩,CFB 命名流),
// 本文件只做 IO/编码门面——与 gaea_schedule_xlsx.go 同构。导入后排程交回
// CPM 重算(约束日期/日历例外/费率不映射,前端导入提示如实告知)。

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gaea/gaea/internal/schedule"
)

// GaeaScheduleImportMpp 二进制 MS Project 工程 mpp(base64)→ 计划 JSON。
func (a *App) GaeaScheduleImportMpp(mppBase64 string) (string, error) {
	if mppBase64 == "" {
		return "", fmt.Errorf("MPP 内容为空")
	}
	raw, err := base64.StdEncoding.DecodeString(mppBase64)
	if err != nil {
		return "", fmt.Errorf("MPP 数据解码失败:%w", err)
	}
	p, err := schedule.ParseMpp(raw)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
