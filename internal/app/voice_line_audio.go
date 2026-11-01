package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/characterlib"
)

// 角色台词语音文件化（内容寻址缓存）：台词+音色+描述哈希 → voices/ 下的
// wav/mp3 文件。命中直接读文件（试听秒开），未命中合成一次落盘；台词/音色
// 任一改动 → 键变 → 旧文件自然失效，下次试听/批量补齐按新键重生成。
// 启动时后台为全量角色补齐初始文件（bootstrapVoiceLineAudio），保存台词/
// 音色变更后单角色后台重生成——保证「改完之后点试听=直接播文件」。

// genderDefaultVoiceDescText 音色描述的性别缺省（与前端 genderDefaultVoiceDesc、
// 聊天链 voicePersonaPreset ③ 同口径）。
func genderDefaultVoiceDescText(gender string) string {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "male":
		return "沉稳清晰的青年男性声线"
	case "female":
		return "温柔清晰的青年女性声线"
	}
	return ""
}

// resolveVoiceLineText 试听台词解析（与前端 resolveVoiceLine 同口径）：
// 专属台词 > 对话样本首条 > 姓名问候。
func resolveVoiceLineText(name, ttsLine string, dialogueSamples []string) string {
	line := strings.TrimSpace(ttsLine)
	if line != "" {
		return line
	}
	for _, s := range dialogueSamples {
		if s = strings.TrimSpace(s); s != "" {
			return s
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "…"
	}
	return fmt.Sprintf("你好，我是「%s」，很高兴认识你。", name)
}

// characterVoiceKey 角色台词语音文件键（内容寻址）。
func characterVoiceKey(line, voiceID, desc string) string {
	return characterlib.VoiceFileKey(line, voiceID, desc)
}

// TTSLineAudioFor 角色台词语音（文件缓存优先）：探键命中直接回读文件字节
// （cached=true，试听秒开），未命中合成一次并落盘（cached=false）。chJSON 为
// 角色 JSON（编辑器可传未保存表单值；新角色无 id 时拒绝——文件按 id 落盘，
// 先保存再生成）。
func (a *mediaState) TTSLineAudioFor(chJSON string) (map[string]interface{}, error) {
	var c characterlib.Character
	if err := json.Unmarshal([]byte(chJSON), &c); err != nil {
		return nil, fmt.Errorf("解析角色数据失败: %w", err)
	}
	if a.charLib == nil {
		return nil, fmt.Errorf("角色库未初始化")
	}
	if strings.TrimSpace(c.ID) == "" {
		return nil, fmt.Errorf("请先保存角色，再生成语音文件")
	}
	line := resolveVoiceLineText(c.Name, c.TTSLine, c.DialogueSamples)
	voiceID := strings.TrimSpace(c.TTSVoiceID)
	desc := strings.TrimSpace(c.TTSDesc)
	if desc == "" {
		desc = genderDefaultVoiceDescText(c.Gender)
	}
	key := characterVoiceKey(line, voiceID, desc)

	// 缓存命中：读文件回传（不碰引擎，试听零等待）
	if path := a.charLib.VoiceLineAudioPath(key); path != "" {
		if data, err := a.charLib.ReadVoiceLineAudio(path); err == nil && len(data) > 0 {
			mime := "audio/wav"
			if strings.HasSuffix(path, ".mp3") {
				mime = "audio/mp3"
			}
			return map[string]interface{}{
				"audio": base64.StdEncoding.EncodeToString(data), "mimeType": mime,
				"cached": true, "path": path,
			}, nil
		}
	}

	audio, mime, err := ttsVoicePreviewAudioFn(a, line, voiceID, desc)
	if err != nil {
		return nil, err
	}
	path := a.charLib.SaveVoiceLineAudio(key, mime, audio)
	if path == "" {
		slog.Warn("台词语音文件落盘失败（本次试听不受影响，下次会重试合成）", "character", c.ID)
	}
	return map[string]interface{}{
		"audio": base64.StdEncoding.EncodeToString(audio), "mimeType": mime,
		"cached": false, "path": path,
	}, nil
}

// voicePresetChanged 判断台词语音的内容键是否变化（台词/克隆音色/音色描述任一
// 变更 → 旧文件失效需重生成）。
func voicePresetChanged(old, cur *characterlib.Character) bool {
	if old.TTSLine != cur.TTSLine || old.TTSVoiceID != cur.TTSVoiceID || old.TTSDesc != cur.TTSDesc {
		return true
	}
	// 对话样本/性别也参与台词与描述回退口径：影响生效键也算变更
	if len(old.DialogueSamples) != len(cur.DialogueSamples) {
		return true
	}
	for i := range cur.DialogueSamples {
		if i >= len(old.DialogueSamples) || old.DialogueSamples[i] != cur.DialogueSamples[i] {
			return true
		}
	}
	return old.Gender != cur.Gender
}

// regenerateCharacterVoiceLine 单角色台词语音重生成（保存钩子后台消费）：
// 按新键合成落盘；旧文件成孤儿不清理（无副作用）。失败只 log——下次试听
// 缓存未命中会现场合成兜底。
func (a *mediaState) regenerateCharacterVoiceLine(c characterlib.Character) {
	if a.charLib == nil || strings.TrimSpace(c.ID) == "" {
		return
	}
	line := resolveVoiceLineText(c.Name, c.TTSLine, c.DialogueSamples)
	voiceID := strings.TrimSpace(c.TTSVoiceID)
	desc := strings.TrimSpace(c.TTSDesc)
	if desc == "" {
		desc = genderDefaultVoiceDescText(c.Gender)
	}
	key := characterVoiceKey(line, voiceID, desc)
	if a.charLib.VoiceLineAudioPath(key) != "" {
		return // 已有（并发竞态兜底），无需重生成
	}
	audio, mime, err := ttsVoicePreviewAudioFn(a, line, voiceID, desc)
	if err != nil || len(audio) == 0 {
		slog.Warn("台词语音重生成失败（下次试听会现场合成兜底）", "character", c.ID, "error", err)
		return
	}
	if p := a.charLib.SaveVoiceLineAudio(key, mime, audio); p != "" {
		slog.Info("台词语音已按新设定重生成", "character", c.ID)
	}
}

// bootstrapVoiceLineAudio 启动后台为全量角色补齐初始台词语音文件：已存在的键
// 跳过（幂等），失败逐角色容忍（log 后继续）。任何角色都合成不出来时整批早退
// （引擎不可用只吵一次，不逐角色重试冷启动）。
func (a *mediaState) bootstrapVoiceLineAudio() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("台词语音批量生成 panic", "panic", r)
		}
	}()
	if a.charLib == nil {
		return
	}
	items, _, err := a.charLib.List("", "", false, 500, 0)
	if err != nil {
		slog.Warn("台词语音批量生成：角色清单读取失败", "error", err)
		return
	}
	// 引擎可用性整批探一次：合成核心的首步（聊天语音绑定/全局模型）若不可表达
	// 预设，末步必是 breeze——它探不活就整批无意义。
	if !a.voicePreviewEngineAlive() {
		slog.Info("台词语音批量生成跳过：无可用的角色音色语音引擎（Breeze 未就绪）")
		return
	}
	done, skip, fail := 0, 0, 0
	for i := range items {
		c := items[i]
		line := resolveVoiceLineText(c.Name, c.TTSLine, c.DialogueSamples)
		voiceID := strings.TrimSpace(c.TTSVoiceID)
		desc := strings.TrimSpace(c.TTSDesc)
		if desc == "" {
			desc = genderDefaultVoiceDescText(c.Gender)
		}
		key := characterVoiceKey(line, voiceID, desc)
		if a.charLib.VoiceLineAudioPath(key) != "" {
			skip++
			continue
		}
		audio, mime, err := ttsVoicePreviewAudioFn(a, line, voiceID, desc)
		if err != nil || len(audio) == 0 {
			fail++
			slog.Warn("台词语音生成失败（角色级容忍）", "character", c.ID, "error", err)
			continue
		}
		if p := a.charLib.SaveVoiceLineAudio(key, mime, audio); p != "" {
			done++
		} else {
			fail++
		}
	}
	slog.Info("台词语音批量生成完成", "total", len(items), "generated", done, "existing", skip, "failed", fail)
}

// voicePreviewEngineAlive 整批前置探测：breeze 探活一次（含按需拉起短等）。
func (a *mediaState) voicePreviewEngineAlive() bool {
	if a.engineMgr == nil {
		return false
	}
	if breezeStreamProbe(a) {
		return true
	}
	breezeStreamEnsure(a)
	for i := 0; i < breezeStreamPollAttempts && !breezeStreamProbe(a); i++ {
		time.Sleep(breezeStreamPollInterval)
	}
	return breezeStreamProbe(a)
}
