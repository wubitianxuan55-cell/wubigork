package app

import (
	"strconv"

	"github.com/gaea/gaea/internal/config"
)

// 设置中枢尾批「高级组收编」（2026-11-19）：此前无 UI 的 config 键（引擎路径/
// 空闲退出/读屏/意图兜底等）统一入设置页高级组。写侧复用既有 SaveConfig
// （66 键宽写面，见 saveconfig_keys_test）；读侧 GetConfig 仅返回 3 个会话键，
// 故补本读绑对。键集与前端 AdvancedSection 的 ADVANCED_KEYS 单源对齐（新增键
// 两处同批）。Realtime 档三键不入组：provider/model 为孤儿键（全仓零消费、
// Config 结构无字段），api_key 为 secure 密文不可回显——语音侧入口见
// VoiceSettingsPanel 实时档。
func (a *App) AdvancedConfigGet() map[string]string {
	c := a.cfg
	return map[string]string{
		config.KeyStrataPath:           c.StrataPath,
		config.KeyWhirlPath:            c.WhirlPath,
		config.KeyBreezeDir:            c.BreezeDir,
		config.KeyBreezePort:           strconv.Itoa(c.BreezePort),
		config.KeyComfyUIPath:          c.ComfyUIPath,
		config.KeyModelIdleExitMinutes: strconv.Itoa(c.ModelIdleExitMinutes),
		config.KeyHTTPTimeoutSeconds:   strconv.Itoa(c.HTTPTimeoutSeconds),
		config.KeyReadScreenSummary:    strconv.FormatBool(c.ReadScreenSummary),
		config.KeyReadScreenKeepLast:   strconv.FormatBool(c.ReadScreenKeepLast),
		config.KeyIntentsLLMFallback:   strconv.FormatBool(c.IntentsLLMFallback),
		config.KeyIntentsLLMTimeoutMS:  strconv.Itoa(c.IntentsLLMTimeoutMS),
		config.KeyGLMCatalogURL:        c.GLMCatalogURL,
	}
}
