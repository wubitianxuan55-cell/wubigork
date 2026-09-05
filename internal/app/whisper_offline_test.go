package app

import (
	"strings"
	"testing"
)

// ── v4.16 persona 侧离线裂缝收口（对照 v4.15 TestPlainChatOfflineFilter）─────────
//
// 背景：v4.15 只把 plain 聊天主链路收归 routeModel("chat")；persona 侧
// （WhisperChat / 因果解释 / 记忆重述）仍 featureModel("chat") 原始读，全局
// 离线模式对 persona 不生效（绑云端照样发云端）。v4.16 三处全部收敛到
// routeModel("chat")，本测试从 persona 入口端到端验证离线过滤：
//   - 离线 + 绑定云端 + 本地可用 → 路由本地（不误发云端，回复来自本地 mock）；
//   - 离线 + 仅剩云端 → 路由为空（走「未绑定模型」降级，不再误发云端）。

// TestWhisperCausalOfflineFilter 因果解释（persona）离线过滤：云端绑定被滤，
// 本地可用走本地；本地全停则「未绑定模型」降级。
func TestWhisperCausalOfflineFilter(t *testing.T) {
	a, _ := seedCausalApp(t, "pidCausalOffline")
	a.cfg.OfflineMode = true
	// 聊天功能绑定云端 xai（v4.15 同款场景：绑云端 + 离线）
	if err := a.SetFeatureModel("chat", "xai", "grok-4.20"); err != nil {
		t.Fatal(err)
	}
	// 停用 ollama/cosyvoice/modelhub，保留 herdsman 作为唯一本地引擎（xai 云端被滤）；
	// 兜底路由取 DefaultModel（夹具默认空，补上才有模型可发本地 mock）
	if e, ok := a.engineMgr.GetEngine("herdsman"); ok {
		e.DefaultModel = "qwen3-8b"
		if err := a.engineMgr.SaveEngine(*e); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"ollama", "cosyvoice", "modelhub"} {
		if e, ok := a.engineMgr.GetEngine(id); ok {
			e.Enabled = false
			if err := a.engineMgr.SaveEngine(*e); err != nil {
				t.Fatal(err)
			}
		}
	}
	// 本地可用 → 路由本地：回复必须来自本地 mock（herdsman），而非云端 xai
	got, err := a.whisperState.GaeaWhisperCausalExplain("睡不好", "pidCausalOffline")
	if err != nil {
		t.Fatalf("GaeaWhisperCausalExplain: %v", err)
	}
	if !strings.Contains(got, "你好呀") {
		t.Fatalf("离线 + 云端绑定应路由本地 mock, got %q", got)
	}

	// 停用全部本地引擎（仅剩云端 xai）→ 路由为空 →「未绑定模型」降级
	if e, ok := a.engineMgr.GetEngine("herdsman"); ok {
		e.Enabled = false
		if err := a.engineMgr.SaveEngine(*e); err != nil {
			t.Fatal(err)
		}
	}
	_, err = a.whisperState.GaeaWhisperCausalExplain("睡不好", "pidCausalOffline")
	if err == nil || !strings.Contains(err.Error(), "未绑定模型") {
		t.Fatalf("离线 + 仅云端应降级「未绑定模型」, got %v", err)
	}
}

// TestWhisperRetellOfflineFilter 记忆重述（persona）离线过滤：云端绑定被滤，
// 本地可用走本地（覆盖 retell 入口，causal/retell 至少一处）。
func TestWhisperRetellOfflineFilter(t *testing.T) {
	a := newChatServiceTestApp(t)
	seedReplayEpisode(t, a.whisperDataRoot, "whisper_retellOffline", "epRetellOffline")

	a.cfg.OfflineMode = true
	if err := a.SetFeatureModel("chat", "xai", "grok-4.20"); err != nil {
		t.Fatal(err)
	}
	if e, ok := a.engineMgr.GetEngine("herdsman"); ok {
		e.DefaultModel = "qwen3-8b"
		if err := a.engineMgr.SaveEngine(*e); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"ollama", "cosyvoice", "modelhub"} {
		if e, ok := a.engineMgr.GetEngine(id); ok {
			e.Enabled = false
			if err := a.engineMgr.SaveEngine(*e); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, err := a.whisperState.GaeaWhisperMemoryRetell("episode", "epRetellOffline", "p1")
	if err != nil {
		t.Fatalf("GaeaWhisperMemoryRetell: %v", err)
	}
	if !strings.Contains(got, "你好呀") {
		t.Fatalf("离线 + 云端绑定应路由本地 mock, got %q", got)
	}

	// 本地全停 → 路由为空 →「未绑定模型」降级（persona 入口不再误发云端）
	if e, ok := a.engineMgr.GetEngine("herdsman"); ok {
		e.Enabled = false
		if err := a.engineMgr.SaveEngine(*e); err != nil {
			t.Fatal(err)
		}
	}
	_, err = a.whisperState.GaeaWhisperMemoryRetell("episode", "epRetellOffline", "p1")
	if err == nil || !strings.Contains(err.Error(), "未绑定模型") {
		t.Fatalf("离线 + 仅云端应降级「未绑定模型」, got %v", err)
	}
}
