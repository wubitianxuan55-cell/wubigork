// Package app — 语音 API 端点（Wails 绑定）
//
// 对齐 Ackem 的 voiceManager.ts IPC 通道：
//   - VoiceStart / VoiceStop → 启停语音管道
//   - VoicePushAudio → 推送麦克风 PCM 块
//   - VoiceSetMode / VoiceApplySettings → 配置管理
//   - VoiceCancelTTS → 打断
//   - VoiceSetPTTActive → 按键说话
//   - VoiceHealth → 健康检查
package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/characterlib"
	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/tts"
)

// ── 初始化 ──

// initVoice 初始化语音管理器（在 Startup 中调用）

func (a *mediaState) ttsProviderForEngine(engineID, model, voiceDescription string) (tts.TTSProvider, bool) {
	if a.engineMgr == nil {
		return nil, false
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok || !eng.Enabled {
		return nil, false
	}
	if engineID == "xai" {
		if a.clientRef() == nil {
			return nil, false
		}
		voice := strings.TrimSpace(a.activeTTSVoice)
		if !tts.IsXaiVoice(voice) {
			voice = "eve"
		}
		p, err := tts.NewTTSProvider("xai", tts.TTSConfig{
			BaseURL: eng.BaseURL, Voice: voice, GetToken: a.clientRef().GetToken,
		})
		if err != nil {
			return nil, false
		}
		return p, true
	}

	// Breeze：OpenAI 兼容 JSON 路径但字段口径不同（instructions/voice，无
	// /v1/audio/info 音色探测），走 breeze kind。音色/描述解析单一来源在
	// breezeVoiceAndInstruction（流播路径共用同一优先级链）。
	if engineID == "breeze" {
		voice, desc := a.breezeVoiceAndInstruction(voiceDescription)
		p, err := tts.NewTTSProvider("breeze", tts.TTSConfig{
			BaseURL: eng.BaseURL, Model: model,
			Voice:            voice,
			VoiceDescription: desc,
		})
		if err != nil {
			return nil, false
		}
		return p, true
	}

	// 其余引擎：OpenAI 兼容 /v1/audio/speech（Herdsman 风格）
	cfg := tts.TTSConfig{BaseURL: eng.BaseURL, Model: model, VoiceDescription: voiceDescription}
	l := strings.ToLower(model)
	switch {
	case strings.Contains(l, "voiceclone"):
		// voiceclone 参考音频/文本来自环境变量（与历史 tryEngineTTS 一致）
		cfg.RefAudio = strings.TrimSpace(os.Getenv("HERDSMAN_TTS_REF_AUDIO"))
		cfg.RefText = strings.TrimSpace(os.Getenv("HERDSMAN_TTS_REF_TEXT"))
	case strings.Contains(l, "voicedesign"), strings.Contains(l, "voxcpm"):
		// voicedesign/voxcpm 无描述时用空音色（工厂按模型收敛；历史行为一致）
	default:
		cfg.Voice = a.ttsVoiceForModel(model)
	}
	p, err := tts.NewTTSProvider("herdsman", cfg)
	if err != nil {
		return nil, false
	}
	return p, true
}

// ttsProviderForEngineWithVoice 试听专用 provider 构造（TTSVoicePreviewBase64
// 消费）：音色/描述显式入参，不读当前人格预设与全局 breeze 描述。仅覆盖
// breeze 与 herdsman voicedesign/voxcpm 两种能表达预设的形态
// （voicePreviewCapable 已先行过滤，这里不再重复判定）。
func (a *mediaState) ttsProviderForEngineWithVoice(engineID, model, voiceID, voiceDescription string) (tts.TTSProvider, bool) {
	if a.engineMgr == nil {
		return nil, false
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok || !eng.Enabled {
		return nil, false
	}
	if engineID == "breeze" {
		p, err := tts.NewTTSProvider("breeze", tts.TTSConfig{
			BaseURL: eng.BaseURL, Model: model,
			Voice:            voiceID,
			VoiceDescription: voiceDescription,
		})
		if err != nil {
			return nil, false
		}
		return p, true
	}
	// herdsman voicedesign/voxcpm：描述即音色（无 Voice 字段，工厂按模型收敛）
	p, err := tts.NewTTSProvider("herdsman", tts.TTSConfig{
		BaseURL: eng.BaseURL, Model: model, VoiceDescription: voiceDescription,
	})
	if err != nil {
		return nil, false
	}
	return p, true
}

// ── Wails API 端点 ──

// VoiceStart 启动语音管道。browserASR 为 true 时使用浏览器端
// Web Speech API 识别（无需 herdsman STT 模型），后端只负责对话与 TTS。
func (a *mediaState) VoiceApplySettings(settings map[string]interface{}) error {
	// V1 拼接链退役（2026-11-05 用户拍板）后，本绑定只剩朗读/音色/人格设定面；
	// V2 实时档运行参数不经此（VAD 阈值走 session.update，模型路由走功能绑定）。
	if v, ok := settings["ttsVoice"].(string); ok {
		a.activeTTSVoice = v
		if err := appconfig.Save(appconfig.KeyTTSVoice, v); err != nil {
			slog.Warn("保存 TTS 音色配置失败", "error", err)
		}
	}
	// P0a-A3：人格双读者同点生效——setVoicePersonality（V2 读者）与
	// KeyVoicePersonality 落盘（朗读/台词读者）在同一成功点翻转。
	if v, ok := settings["personalityPresetId"].(string); ok {
		if err := appconfig.Save(appconfig.KeyVoicePersonality, v); err != nil {
			slog.Warn("保存语音角色配置失败", "error", err)
		}
		a.setVoicePersonality(v)
	}
	if v, ok := settings["breezeVoiceDesc"].(string); ok {
		// Breeze voice design 音色描述：只落盘（provider 在 ttsProviderForEngine
		// 处实时读 cfg，不进 voice 运行时配置）
		if err := appconfig.Save(appconfig.KeyBreezeVoiceDesc, v); err != nil {
			return fmt.Errorf("保存 Breeze 音色描述失败: %w", err)
		}
		if a.cfg != nil {
			a.cfg.BreezeVoiceDesc = v
		}
	}
	if v, ok := settings["personaVoicePresets"]; ok {
		// 角色音色预设（人格→音色）：整体规范化落盘（形状不符整批拒绝——
		// 半套预设会让人格间音色跳变难以排查）
		normalized := map[string]map[string]string{}
		if m, mok := v.(map[string]interface{}); mok {
			for pid, pv := range m {
				pm, pok := pv.(map[string]interface{})
				if !pok {
					return fmt.Errorf("角色音色预设形状不符: %s", pid)
				}
				vid, _ := pm["voiceId"].(string)
				dsc, _ := pm["desc"].(string)
				normalized[pid] = map[string]string{"voiceId": strings.TrimSpace(vid), "desc": strings.TrimSpace(dsc)}
			}
		} else if s, sok := v.(string); !sok || s != "" {
			return fmt.Errorf("personaVoicePresets 须为对象")
		}
		b, err := json.Marshal(normalized)
		if err != nil {
			return fmt.Errorf("序列化角色音色预设失败: %w", err)
		}
		if err := appconfig.Save(appconfig.KeyVoicePersonaPresets, string(b)); err != nil {
			return fmt.Errorf("保存角色音色预设失败: %w", err)
		}
		if a.cfg != nil {
			a.cfg.VoicePersonaPresets = string(b)
		}
	}
	return nil
}

func (a *mediaState) VoiceGetSettings() map[string]interface{} {
	m := map[string]interface{}{
		"ttsVoice": a.activeTTSVoice,
	}
	if a.cfg != nil {
		m["breezeVoiceDesc"] = a.cfg.BreezeVoiceDesc
		m["personaVoicePresets"] = a.cfg.VoicePersonaPresets
	}
	return m
}

// voicePersonaPreset 解析当前人格的音色预设（voice_persona_presets JSON）。
// 坏 JSON/无预设/人格为 plain（无人格）一律返回空——调用方走全局缺省链。
// voicePersonaPreset 解析当前人格（=角色库角色 id）的音色预设，三级独立回退：
// ① 角色卡字段（characterlib.Character.TTSVoiceID/TTSDesc，各自非空即用）；
// ② 全局映射 cfg.VoicePersonaPresets（旧版面板数据，兼容保留）；
// ③ gender 缺省描述（角色卡未配置时出厂不哑——仅 desc，voice 无缺省）。
// plain（无人格）/空 id/库失败 → 空，走全局缺省链。
func (a *mediaState) voicePersonaPreset(personalityID string) (voiceID, desc string) {
	if personalityID == "" || personalityID == "plain" {
		return "", ""
	}
	// ① 角色卡（一次 Get 同时取音色字段与 gender 缺省底料）
	var ch *characterlib.Character
	if a.charLib != nil {
		// Get 无行时返回 (nil, nil)——必须双守卫
		if c, err := a.charLib.Get(personalityID); err == nil && c != nil {
			ch = c
			voiceID = strings.TrimSpace(c.TTSVoiceID)
			desc = strings.TrimSpace(c.TTSDesc)
		}
	}
	// ② 全局映射（旧版面板数据兼容）
	if a.cfg != nil && a.cfg.VoicePersonaPresets != "" && (voiceID == "" || desc == "") {
		var m map[string]struct {
			VoiceID string `json:"voiceId"`
			Desc    string `json:"desc"`
		}
		if err := json.Unmarshal([]byte(a.cfg.VoicePersonaPresets), &m); err == nil {
			if p, ok := m[personalityID]; ok {
				if voiceID == "" {
					voiceID = strings.TrimSpace(p.VoiceID)
				}
				if desc == "" {
					desc = strings.TrimSpace(p.Desc)
				}
			}
		}
	}
	// ③ gender 缺省描述（角色卡/映射均未配置描述时，出厂不哑）
	if desc == "" && ch != nil {
		switch strings.ToLower(ch.Gender) {
		case "male":
			desc = "沉稳清晰的青年男性声线"
		case "female":
			desc = "温柔清晰的青年女性声线"
		}
	}
	return voiceID, desc
}

// herdsmanInstalledModelIDs 返回 Herdsman 引擎当前已安装（可用）模型 ID 列表。
// 引擎未初始化/未启用/无模型列表时返回 nil（消费方 ttsRouteHealth 按
// 「Herdsman 路由不可达」处理）。
func (a *mediaState) herdsmanInstalledModelIDs() []string {
	if a.engineMgr == nil {
		return nil
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok || !eng.Enabled {
		return nil
	}
	if len(eng.Models) == 0 {
		return nil
	}
	ids := make([]string, 0, len(eng.Models))
	for _, m := range eng.Models {
		if strings.TrimSpace(m.ID) != "" {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

// VoiceHealth 健康检查（V1 退役后只剩朗读/TTS 路由面；ttsReady=真实合成路由
// 第一跳探测，2026-11-01 诚实化口径不变）。
func (a *mediaState) VoiceHealth() map[string]interface{} {
	ready, detail := a.ttsRouteHealth()
	return map[string]interface{}{
		"ttsReady":  ready,
		"state":     "idle",
		"ttsDetail": detail,
	}
}

func (a *mediaState) ttsRouteHealth() (bool, string) {
	engineID, modelID, hop := "", "", ""
	switch {
	case a.chatVoiceEngine != "" && a.chatVoiceModel != "":
		engineID, modelID, hop = a.chatVoiceEngine, a.chatVoiceModel, "聊天语音绑定"
	case a.activeTTSModel != "":
		engineID, modelID, hop = a.activeTTSEngine, a.activeTTSModel, "全局 TTS"
	default:
		return true, "自动路由：本地引擎优先，Edge 云端兜底"
	}
	label := hop + "：" + engineID + "/" + modelID
	switch {
	case isLocalTTSAssetEngine(engineID):
		if a.ttsReady() {
			return true, label + "（本地服务在线）"
		}
		if engineID == "breeze" {
			if _, err := breezeCmdFor(a.cfg); err != nil {
				return false, label + "——服务未启动且未检测到安装（朗读将滑落 Edge 兜底）"
			}
		}
		return false, label + "（本地服务离线，朗读时自动拉起）"
	case engineID == "xai":
		// 云端：绑定在位即可用（不实拨 API 花配额；合成失败自动滑落全局/兜底）。
		if a.clientRef() != nil {
			return true, label + "（xai 云端）"
		}
		return false, label + "——云端客户端未就绪（朗读将滑落全局/Edge 兜底）"
	default:
		// Herdsman 系：引擎启用且有已装模型即路由可达。
		if len(a.herdsmanInstalledModelIDs()) > 0 {
			return true, label + "（Herdsman）"
		}
		return false, label + "——Herdsman 未启用/无已装模型（朗读将滑落 Edge 兜底）"
	}
}

// VoiceRestartService 重启语音服务（重新检测 ASR/TTS 可用性）
// ── 工具函数 ──
