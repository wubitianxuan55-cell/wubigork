package app

// wps_dock.go — P1-4 WPS 面板停靠（dock 档）：把 WPS 文档窗口摆放到前端
// 预览面板矩形上。实现收编自 tools/wpsdock 探针（internal/office/wpshost）；
// embed（SetParent）是第二步，默认不做，须用户拍板另开。
//
// 前端分工：FilePreview 停靠钮列出 WpsDockList 的文档窗口 → 选一个
// WpsDockApply(面板屏幕矩形) → ResizeObserver 跟随 → 还原走 WpsDockRestore；
// 组件卸载自动还原（防面板关闭后窗口留在原地）。

import (
	"fmt"

	"github.com/gaea/gaea/internal/office/wpshost"
)

// WpsWindowInfo 是给前端选窗口用的最小面（样式位不上桥）。
type WpsWindowInfo struct {
	HWND     uint64 `json:"hwnd"`
	Proc     string `json:"proc"`
	Class    string `json:"class"`
	Title    string `json:"title"`
	Visible  bool   `json:"visible"`
	IsDocWin bool   `json:"isDocWin"`
}

// WpsDockList 列出本机 WPS 相关窗口（可见窗口优先展示，前端过滤）。
func (a *App) WpsDockList() ([]WpsWindowInfo, error) {
	wins := wpshost.List()
	if len(wins) == 0 {
		return []WpsWindowInfo{}, fmt.Errorf("没发现 WPS 窗口——先打开 WPS 文字/表格再停靠")
	}
	out := make([]WpsWindowInfo, 0, len(wins))
	for _, w := range wins {
		out = append(out, WpsWindowInfo{
			HWND: w.HWND, Proc: w.Proc, Class: w.Class,
			Title: w.Title, Visible: w.Visible, IsDocWin: w.IsDocWin,
		})
	}
	return out, nil
}

// WpsDockApply 停靠：x/y/w/h 为屏幕物理像素矩形（前端由 DOM rect×DPR 换算）。
func (a *App) WpsDockApply(hwnd uint64, x, y, w, h int, borderless bool) (string, error) {
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("矩形尺寸无效（%dx%d）——面板可能已折叠", w, h)
	}
	if err := wpshost.Dock(hwnd, x, y, w, h, borderless); err != nil {
		return "", err
	}
	return fmt.Sprintf("已停靠 WPS 窗口 %d → %d,%d %dx%d（borderless=%v）", hwnd, x, y, w, h, borderless), nil
}

// WpsDockRestore 还原窗口样式与可见性（停靠档不改父子关系，还原即回顶层）。
func (a *App) WpsDockRestore(hwnd uint64) (string, error) {
	if err := wpshost.Restore(hwnd); err != nil {
		return "", err
	}
	return fmt.Sprintf("已还原 WPS 窗口 %d", hwnd), nil
}
