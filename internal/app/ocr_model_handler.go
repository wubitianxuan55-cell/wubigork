package app

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/docmd"
)

// SetActiveOCRModel 设置办公「提取文字」使用的 OCR 引擎与模型。
// 两个参数都为空时清除绑定，回退到自动选择。
func (c *core) SetActiveOCRModel(engineID, modelID string) error {
	// 本地服务目录（电脑操控配套刀）：engineID=local:<docmd kind>。定点
	// docmd 本地链（见 gaea_ocr.go 本地腿），不占 herdsman 腿 1——两链配置
	// 名词同名不同物（docmd/ocr.go 口径边界段），持久化键独立。该支不依赖
	// engineMgr，置于判空之前。
	if strings.HasPrefix(engineID, "local:") {
		kind := strings.TrimPrefix(engineID, "local:")
		switch kind {
		case docmd.OCRKindUocr2:
		case "ovis", "tesseract":
			// 2026-11-26 退役的本地引擎：如实拒绝并指路，不静默换成 uocr2。
			return &appError{"本地 OCR 引擎 " + kind + " 已退役（本地只保留 Unlimited-OCR-2），请选 local:uocr2"}
		default:
			return &appError{"未知本地 OCR 引擎: " + kind}
		}
		c.activeOCRLocal = kind
		if err := config.Save(config.KeyActiveOCRLocal, kind); err != nil {
			slog.Warn("保存本地 OCR 引擎配置失败", "error", err)
		}
		c.emit("ocr-model-changed", map[string]interface{}{"engine": engineID, "model": modelID})
		slog.Info("本地 OCR 引擎已定点", "kind", kind)
		return nil
	}

	if strings.TrimSpace(engineID) == "" && strings.TrimSpace(modelID) == "" {
		c.activeOCREngine = ""
		c.activeOCRModel = ""
		c.activeOCRLocal = ""
		_ = config.Save(config.KeyActiveOCREngine, "")
		_ = config.Save(config.KeyActiveOCRModel, "")
		_ = config.Save(config.KeyActiveOCRLocal, "")
		c.emit("ocr-model-changed", map[string]interface{}{"engine": "", "model": ""})
		return nil
	}

	if c.engineMgr == nil {
		return errNoEngineMgr
	}

	eng, ok := c.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}
	if modelID == "" {
		return &appError{"OCR 模型不能为空"}
	}
	found := false
	for _, m := range eng.Models {
		l := strings.ToLower(m.ID)
		if m.ID == modelID && (strings.Contains(l, "ocr") || strings.Contains(l, "paddle") || strings.Contains(l, "mineru") || strings.Contains(l, "parse")) {
			found = true
			break
		}
	}
	if len(eng.Models) > 0 && !found {
		return &appError{fmt.Sprintf("模型 %s 不是引擎 %s 的 OCR 模型", modelID, engineID)}
	}

	c.activeOCREngine = engineID
	c.activeOCRModel = modelID
	if err := config.Save(config.KeyActiveOCREngine, engineID); err != nil {
		slog.Warn("保存 OCR 引擎配置失败", "error", err)
	}
	if err := config.Save(config.KeyActiveOCRModel, modelID); err != nil {
		slog.Warn("保存 OCR 模型配置失败", "error", err)
	}
	c.emit("ocr-model-changed", map[string]interface{}{"engine": engineID, "model": modelID})
	slog.Info("OCR 模型已切换", "engine", engineID, "model", modelID)
	return nil
}

// GetActiveOCRModel 获取当前 OCR 激活模型（空=自动选择）。
func (c *core) GetActiveOCRModel() map[string]string {
	if c.activeOCRLocal != "" {
		// 本地定点：engine 回 local:<kind>，前端专业页按卡面 engineId 高亮。
		return map[string]string{
			"engine": "local:" + c.activeOCRLocal,
			"model":  c.activeOCRLocal,
		}
	}
	return map[string]string{
		"engine": c.activeOCREngine,
		"model":  c.activeOCRModel,
	}
}

// normalizeLocalOCRKind 本地引擎定点只认 uocr2（2026-11-26 精简后本地唯一引擎）。
// 配置里残留的已退役值（ovis / tesseract）在启动装载时清空并如实记日志——不带着
// 坏值去调 docmd（那会让「提取文字」的本地腿每次都报「已退役」而失效）。
func normalizeLocalOCRKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" || kind == docmd.OCRKindUocr2 {
		return kind
	}
	slog.Warn("本地 OCR 引擎定点已失效：该引擎已退役，改回默认（Unlimited-OCR-2）", "configured", kind)
	return ""
}
