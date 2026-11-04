package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newVoiceSettingsTestState 构造测试用 mediaState（voiceManager 为 nil，原
// voice_settings_resolve_test.go 同名 helper 迁此——tts_seam_test 等沿用）。
func newVoiceSettingsTestState(t *testing.T, models []modelengine.ModelInfo) *mediaState {
	t.Helper()
	c := &core{engineMgr: modelengine.NewManager("", "")}
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID:      "herdsman",
		Enabled: true,
		Models:  models,
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	return &mediaState{core: c}
}

// newVoiceRouteTestState 构造测试用 mediaState：engineMgr 预置 herdsman 引擎
// 模型列表（对齐 ttsRouteHealth 对 Herdsman 系的可达性判定输入），并注入
// voiceManager（VoiceHealth 走 manager.HealthCheck 路径）。
func newVoiceRouteTestState(t *testing.T, models []modelengine.ModelInfo) *mediaState {
	t.Helper()
	a := newVoiceSettingsTestState(t, models)
	// 探活隔离：塞死端口防误连真机本地 TTS 服务（breeze 8012 常驻在跑，
	// 不隔离则离线用例被真服务喂成假绿——同 uocr2 测试塞死值坑律）。
	a.cfg = &config.Config{BreezePort: 1}
	return a
}

// TestTTSRouteHealth_AutoRoute：无聊天绑定、无全局 TTS → 扫描链自动路由，恒可用。
func TestTTSRouteHealth_AutoRoute(t *testing.T) {
	a := newVoiceRouteTestState(t, nil)
	ready, detail := a.ttsRouteHealth()
	if !ready {
		t.Fatalf("auto route ready = false, want true（detail=%s）", detail)
	}
	if !strings.Contains(detail, "自动路由") {
		t.Errorf("detail = %q, want 含「自动路由」", detail)
	}
}

// TestTTSRouteHealth_ChatBindingXai：聊天绑定 xai 云端（客户端就绪）→ 立即可用，
// detail 如实标注绑定引擎（这正是「绑定短路角色克隆链」的可见性来源）。
func TestTTSRouteHealth_ChatBindingXai(t *testing.T) {
	a := newVoiceRouteTestState(t, nil)
	a.chatVoiceEngine = "xai"
	a.chatVoiceModel = "grok-tts"
	// clientRef 为 nil（未注入 AI 客户端）→ 云端未就绪
	if ready, detail := a.ttsRouteHealth(); ready || !strings.Contains(detail, "云端客户端未就绪") {
		t.Errorf("xai 无客户端: ready=%v detail=%q, want false+未就绪", ready, detail)
	}
}

// TestTTSRouteHealth_ChatBindingBreezeOffline：聊天绑定 breeze、服务离线
// （测试环境无本地服务）→ 立即可用=false，detail 标注「朗读时自动拉起」。
func TestTTSRouteHealth_ChatBindingBreezeOffline(t *testing.T) {
	a := newVoiceRouteTestState(t, nil)
	a.chatVoiceEngine = "breeze"
	a.chatVoiceModel = "breeze-tts-2"
	ready, detail := a.ttsRouteHealth()
	if ready {
		t.Fatalf("breeze 离线 ready = true, want false")
	}
	if !strings.Contains(detail, "breeze/breeze-tts-2") || !strings.Contains(detail, "自动拉起") {
		t.Errorf("detail = %q, want 含引擎/模型与「自动拉起」", detail)
	}
}

// TestTTSRouteHealth_GlobalHerdsman：全局 TTS 走 Herdsman 系、引擎已装 → 可用。
func TestTTSRouteHealth_GlobalHerdsman(t *testing.T) {
	a := newVoiceRouteTestState(t, []modelengine.ModelInfo{{ID: "qwen3-tts-voicedesign"}})
	a.activeTTSEngine = "herdsman"
	a.activeTTSModel = "qwen3-tts-voicedesign"
	if ready, detail := a.ttsRouteHealth(); !ready || !strings.Contains(detail, "Herdsman") {
		t.Errorf("herdsman 已装: ready=%v detail=%q, want true+Herdsman", ready, detail)
	}
}

// TestTTSRouteHealth_GlobalHerdsmanNotInstalled：全局 Herdsman 未装模型 →
// 不可用并说明滑落。
func TestTTSRouteHealth_GlobalHerdsmanNotInstalled(t *testing.T) {
	a := newVoiceRouteTestState(t, nil) // herdsman 无已装模型
	a.activeTTSEngine = "herdsman"
	a.activeTTSModel = "qwen3-tts-voicedesign"
	if ready, detail := a.ttsRouteHealth(); ready || !strings.Contains(detail, "滑落") {
		t.Errorf("herdsman 未装: ready=%v detail=%q, want false+滑落说明", ready, detail)
	}
}

// TestVoiceHealth_HonestTTS：VoiceHealth 覆写 ttsReady 为路由探测结果，
// 并带 browserAsr/ttsDetail/asrDetail 诚实化标注。
// （TestVoiceHealth_HonestTTS 的 asrReady/browserAsr 断言随 V1 退役删除；
// ttsReady 路由探测诚实化由 TestTTSRouteHealth_* 系列继续覆盖）
func TestVoiceGetSettings_NoDeadHerdsmanModel(t *testing.T) {
	a := newVoiceRouteTestState(t, nil)
	got := a.VoiceGetSettings()
	for _, k := range []string{"ttsHerdsmanModel", "ttsHerdsmanModelFallback", "ttsHerdsmanModelFromInstalled"} {
		if _, ok := got[k]; ok {
			t.Errorf("settings 仍含退役键 %s", k)
		}
	}
}
