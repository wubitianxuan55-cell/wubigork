// Package app — 语音模型选择（模型中心 → 语音管道）
//
// 三段模型（STT 识别 / LLM 对话 / TTS 合成）独立选型、分别持久化，
// 对齐 Open WebUI audio.* 键体系的行业惯例：
//   - STT/TTS 由模型中心选择（引擎 + 模型），写 ~/.gaea_config.json
//   - LLM 复用主模型激活（GetActiveEngine / GetActiveModel）
//   - 切换后 emit voice-model-changed 通知前端刷新激活态
package app

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/tts"
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
	// 校验模型在引擎可用列表（有列表时），避免静默配置无效 ASR
	if len(eng.Models) > 0 {
		found := false
		for _, m := range eng.Models {
			if m.ID == modelID {
				found = true
				break
			}
		}
		if !found {
			return &appError{fmt.Sprintf("模型 %s 不在引擎 %s 的可用列表", modelID, engineID)}
		}
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
		"stt":     map[string]string{"engine": asr["engine"], "model": asr["model"]},
		"llm":     map[string]string{"engine": a.GetActiveEngine(), "model": a.GetActiveModel()},
		"tts":     map[string]string{"engine": tts["engine"], "model": tts["model"], "voice": a.activeTTSVoice},
		"chatTts": map[string]string{"engine": a.chatVoiceEngine, "model": a.chatVoiceModel},
	}
}

// SetChatVoiceModel 设置聊天语音合成模型（功能绑定，空引擎=清除绑定回退全局 TTS）
// 模型中心「功能绑定」区块调用；持久化后语音管道优先使用该绑定。
func (a *mediaState) SetChatVoiceModel(engineID, modelID string) error {
	if a.engineMgr == nil {
		return errNoEngineMgr
	}

	// 清空绑定：回退全局 TTS
	if engineID == "" || modelID == "" {
		a.chatVoiceEngine = ""
		a.chatVoiceModel = ""
		if err := config.Save(config.KeyFuncChatVoiceEngine, ""); err != nil {
			slog.Warn("保存聊天语音引擎失败", "error", err)
		}
		if err := config.Save(config.KeyFuncChatVoiceModel, ""); err != nil {
			slog.Warn("保存聊天语音模型失败", "error", err)
		}
		a.emit("voice-model-changed", map[string]interface{}{"chatTtsEngine": "", "chatTtsModel": ""})
		return nil
	}

	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok {
		return &appError{"引擎不存在: " + engineID}
	}
	if !eng.Enabled {
		return &appError{"引擎未启用: " + engineID}
	}
	// 校验模型在引擎可用列表（有列表时），并确认是 TTS/语音模型
	found := false
	for _, m := range eng.Models {
		l := strings.ToLower(m.ID)
		if m.ID == modelID && (strings.Contains(l, "tts") || strings.Contains(l, "voice") || strings.Contains(l, "speech") || strings.Contains(l, "edge") || strings.Contains(l, "cosyvoice") || strings.Contains(l, "voxcpm")) {
			found = true
			break
		}
	}
	if len(eng.Models) > 0 && !found {
		return &appError{fmt.Sprintf("模型 %s 不是引擎 %s 的语音模型", modelID, engineID)}
	}

	a.chatVoiceEngine = engineID
	a.chatVoiceModel = modelID
	if err := config.Save(config.KeyFuncChatVoiceEngine, engineID); err != nil {
		slog.Warn("保存聊天语音引擎失败", "error", err)
	}
	if err := config.Save(config.KeyFuncChatVoiceModel, modelID); err != nil {
		slog.Warn("保存聊天语音模型失败", "error", err)
	}
	a.emit("voice-model-changed", map[string]interface{}{"chatTtsEngine": engineID, "chatTtsModel": modelID})
	slog.Info("聊天语音模型已绑定", "engine", engineID, "model", modelID)
	return nil
}

// GetChatVoiceModel 获取聊天语音合成绑定（空 = 回退全局 TTS）
func (a *mediaState) GetChatVoiceModel() map[string]string {
	return map[string]string{
		"engine": a.chatVoiceEngine,
		"model":  a.chatVoiceModel,
	}
}

// ttsVoiceForModel 返回语音合成使用的音色（设置面板选择优先，空则按模型默认）
func (a *mediaState) ttsVoiceForModel(model string) string {
	if v := strings.TrimSpace(a.activeTTSVoice); v != "" {
		return v
	}
	l := strings.ToLower(model)
	switch {
	case strings.Contains(l, "edge"):
		return "zh-CN-YunxiNeural"
	case strings.Contains(l, "voicedesign"), strings.Contains(l, "voiceclone"), strings.Contains(l, "voxcpm"):
		return ""
	case strings.Contains(l, "grok-tts"):
		return "eve"
	case strings.Contains(l, "cosyvoice"):
		return "中文女"
	default:
		return "serena"
	}
}

// GetTTSSpeakers 获取指定 Herdsman TTS 模型支持的音色列表（设置面板据此渲染选择器）
func (a *mediaState) GetTTSSpeakers(model string) ([]string, error) {
	if a.engineMgr == nil {
		return nil, fmt.Errorf("引擎管理器未初始化")
	}
	engineID := "herdsman"
	if strings.Contains(strings.ToLower(model), "cosyvoice") {
		engineID = "cosyvoice"
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok || !eng.Enabled {
		return nil, fmt.Errorf("%s 引擎未启用", engineID)
	}
	if model == "" {
		model = a.activeTTSModel
	}
	if model == "" {
		model = "qwen3-tts-customvoice"
	}
	htts := tts.NewHerdsmanTTS(eng.BaseURL, model, "")
	return htts.SupportedSpeakers(), nil
}
