package app

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/config"
)

// SetActiveOCRModel 设置办公「提取文字」使用的 OCR 引擎与模型。
// 两个参数都为空时清除绑定，回退到自动选择。
func (c *core) SetActiveOCRModel(engineID, modelID string) error {
	if c.engineMgr == nil {
		return errNoEngineMgr
	}

	if strings.TrimSpace(engineID) == "" && strings.TrimSpace(modelID) == "" {
		c.activeOCREngine = ""
		c.activeOCRModel = ""
		_ = config.Save(config.KeyActiveOCREngine, "")
		_ = config.Save(config.KeyActiveOCRModel, "")
		c.emit("ocr-model-changed", map[string]interface{}{"engine": "", "model": ""})
		return nil
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
	return map[string]string{
		"engine": c.activeOCREngine,
		"model":  c.activeOCRModel,
	}
}
