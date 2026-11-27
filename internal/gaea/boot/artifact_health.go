package boot

// artifact_health.go — 会话交接卡的逐件体检钩子（2026-11-27 AI 反馈 #14）：
// sessionquery 不认识文件类型，xlsx 的打开性与缓存值在这里接上 xlsxedit。

import (
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/sessionquery"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

// wireArtifactHealth 给 session_search 的交接卡接上体检：xlsx 打不开、公式格没有
// 缓存值这类「合法但退化」的交棒件，下一场开场就看得见。
func wireArtifactHealth() {
	sessionquery.SetArtifactHealth(func(path string) string {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".xlsx", ".xlsm":
		default:
			return ""
		}
		h, err := xlsxedit.Inspect(path)
		switch {
		case err != nil:
			return "⚠ 打不开（多半没写完整）——先用 Excel/WPS 确认，别接着用"
		case h.NoCachedValue > 0:
			return "⚠ " + h.CacheLine() + "——没重算，用 Excel/WPS 打开是空值；接着用之前先 xlsx_recalc"
		case h.ErrorValues > 0:
			return "⚠ " + h.CacheLine() + "——先用 xlsx_audit 看是哪些格"
		}
		return "可打开，公式都有缓存值"
	})
}
