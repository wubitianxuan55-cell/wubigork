// Package app — 语音模型选择（模型中心 → 语音管道）
//
// 三段模型（STT 识别 / LLM 对话 / TTS 合成）独立选型、分别持久化，
// 对齐 Open WebUI audio.* 键体系的行业惯例：
//   - STT/TTS 由模型中心选择（引擎 + 模型），写 ~/.gaea_config.json
//   - LLM 复用主模型激活（GetActiveEngine / GetActiveModel）
//   - 切换后 emit voice-model-changed 通知前端刷新激活态
package app

import (
	"log/slog"

	"github.com/gaea/gaea/internal/config"
)

// ── 语音识别 (STT) 模型选择 ──────────────────────────────────

// SetActiveASRModel 设置语音识别激活模型（模型中心 STT 模型）
func (a *mediaState) SetActiveASRModel(engineID, modelID string) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}

	a.activeASREngine = engineID
	a.activeASRModel = modelID

	// 重建 ASR 客户端（语音管道立即生效）
	a.applyASRClient()

	// 持久化（重启恢复）
	if err := config.Save(config.KeyActiveASREngine, engineID); err != nil {
		slog.Warn("保存 ASR 引擎配置失败", "error", err)
	}
	if err := config.Save(config.KeyActiveASRModel, modelID); err != nil {
		slog.Warn("保存 ASR 模型配置失败", "error", err)
	}

	a.emit("voice-model-changed", map[string]interface{}{"asrEngine": engineID, "asrModel": modelID})
	slog.Info("语音识别模型已切换", "engine", engineID, "model", modelID)
	return nil
}

// GetActiveASRModel 获取语音识别激活模型
func (a *mediaState) GetActiveASRModel() map[string]string {
	return map[string]string{
		"engine": a.activeASREngine,
		"model":  a.activeASRModel,
	}
}

// ── 语音合成 (TTS) 模型选择 ──────────────────────────────────

// SetActiveTTSModel 设置语音合成激活模型（模型中心 TTS 模型），并持久化
func (a *mediaState) SetActiveTTSModel(engineID, modelID string) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}

	a.activeTTSEngine = engineID
	a.activeTTSModel = modelID

	// 持久化（重启恢复）
	if err := config.Save(config.KeyActiveTTSEngine, engineID); err != nil {
		slog.Warn("保存 TTS 引擎配置失败", "error", err)
	}
	if err := config.Save(config.KeyActiveTTSModel, modelID); err != nil {
		slog.Warn("保存 TTS 模型配置失败", "error", err)
	}

	a.emit("voice-model-changed", map[string]interface{}{"ttsEngine": engineID, "ttsModel": modelID})
	slog.Info("语音合成模型已切换", "engine", engineID, "model", modelID)
	return nil
}

// GetActiveTTSModel 获取语音合成激活模型
func (a *mediaState) GetActiveTTSModel() map[string]string {
	return map[string]string{
		"engine": a.activeTTSEngine,
		"model":  a.activeTTSModel,
	}
}

// ── 三段模型汇总 ─────────────────────────────────────────────

// GetVoicePipelineConfig 获取语音管道三段模型配置（STT / LLM / TTS）
func (a *mediaState) GetVoicePipelineConfig() map[string]interface{} {
	asr := a.GetActiveASRModel()
	tts := a.GetActiveTTSModel()
	return map[string]interface{}{
		"stt": map[string]string{"engine": asr["engine"], "model": asr["model"]},
		"llm": map[string]string{"engine": a.GetActiveEngine(), "model": a.GetActiveModel()},
		"tts": map[string]string{"engine": tts["engine"], "model": tts["model"]},
	}
}
