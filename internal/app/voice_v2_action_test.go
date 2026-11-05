package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/intent"
)

// newVoiceV2ActionTestApp 构造最小 App：执行层桩化（禁连真实执行面——
// GetModelMonitor 等会静默连真引擎/真实库，App 测试语义管线地雷）。
func newVoiceV2ActionTestApp(t *testing.T) *App {
	t.Helper()
	// 台账是包级全局：每例清空防跨例泄漏
	voiceV2PreviewMu.Lock()
	voiceV2PreviewLedger = map[string]*voiceV2ActionPreview{}
	voiceV2PreviewMu.Unlock()
	t.Cleanup(func() {
		voiceV2PreviewMu.Lock()
		voiceV2PreviewLedger = map[string]*voiceV2ActionPreview{}
		voiceV2PreviewMu.Unlock()
	})
	// 执行桩：返回固定回执（编排断言与真实副作用解耦）
	prevExec := execVoiceV2Action
	execVoiceV2Action = func(a *App, action string, it *intent.Intent) (string, bool) {
		switch action {
		case string(intent.ActionNavigate):
			return "好，已打开运行监测。", true
		case string(intent.ActionStatus):
			return "当前状态：测试引擎（本地）。", true
		case string(intent.ActionReminder):
			return "好，已设提醒：开会。", true
		case string(intent.ActionSaveTask):
			return "好，任务已保存。", true
		}
		return "", false
	}
	t.Cleanup(func() { execVoiceV2Action = prevExec })
	a := &App{core: &core{cfg: &config.Config{}}}
	a.mediaState = &mediaState{core: a.core}
	return a
}

func TestVoiceV2PreviewIntentWhitelistAndLedger(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	// M5 刀一：navigate=T0 免确认——预览时直接执行并回执，不入台账
	res := a.VoiceV2PreviewIntent("打开运行监测")
	if res["handled"] != true || res["executed"] != true {
		t.Fatalf("T0 navigate 应直执行: %v", res)
	}
	if !strings.Contains(res["reply"].(string), "运行监测") {
		t.Fatalf("T0 回执缺失: %v", res)
	}
	if _, hasID := res["previewId"]; hasID {
		t.Fatalf("T0 不得发预览凭据: %v", res)
	}
	voiceV2PreviewMu.Lock()
	n := len(voiceV2PreviewLedger)
	voiceV2PreviewMu.Unlock()
	if n != 0 {
		t.Fatalf("T0 不得写台账: %d", n)
	}
	// T1 save_task=白名单内：登记台账并返回 preview 凭据（确认后才执行）。
	// （reminder 预览依赖 whisperState 域，最小 App 缺域=未命中，故夹具用 save_task）
	res1 := a.VoiceV2PreviewIntent("帮我记个任务:写周报")
	if res1["handled"] != true || res1["executable"] != true || res1["executed"] != nil {
		t.Fatalf("T1 save_task 应走预览台账: %v", res1)
	}
	id, _ := res1["previewId"].(string)
	hash, _ := res1["intentHash"].(string)
	if id == "" || hash == "" || res1["ttlMs"] == float64(0) {
		t.Fatalf("预览凭据缺失: %v", res1)
	}
	voiceV2PreviewMu.Lock()
	p := voiceV2PreviewLedger[id]
	voiceV2PreviewMu.Unlock()
	if p == nil || p.action != string(intent.ActionSaveTask) {
		t.Fatalf("台账未登记: %+v", p)
	}
	// 未命中：handled=false 零台账
	before := len(voiceV2PreviewLedger)
	if r := a.VoiceV2PreviewIntent("今晚想听故事"); r["handled"] != false {
		t.Fatalf("闲聊不应命中: %v", r)
	}
	if len(voiceV2PreviewLedger) != before {
		t.Fatalf("未命中不得写台账")
	}
}

func TestVoiceV2T0StatusDirectExec(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	// status=T0 只读：直执行回执（话术按 reStatus 规则命中）
	res := a.VoiceV2PreviewIntent("现在用的是什么模型")
	if res["handled"] != true || res["executed"] != true {
		t.Fatalf("T0 status 应直执行: %v", res)
	}
	if !strings.Contains(res["reply"].(string), "状态") {
		t.Fatalf("status 回执异常: %v", res)
	}
}

func TestVoiceV2ExecuteActionExactlyOnce(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	res := a.VoiceV2PreviewIntent("帮我记个任务:写周报")
	id, _ := res["previewId"].(string)
	hash, _ := res["intentHash"].(string)

	first := a.VoiceV2ExecuteAction(id, hash)
	if first["executed"] != true {
		t.Fatalf("首次执行应成功: %v", first)
	}
	// 幂等：同凭据第二次调用=already-executed（roadmap「确认后最多执行 1 次」）
	second := a.VoiceV2ExecuteAction(id, hash)
	if second["executed"] != false || second["reason"] != "already-executed" {
		t.Fatalf("重复执行必须被拒: %v", second)
	}
	// hash 不符（防篡改）
	res2 := a.VoiceV2PreviewIntent("帮我记个任务:回访客户")
	id2, _ := res2["previewId"].(string)
	if r := a.VoiceV2ExecuteAction(id2, "deadbeef"); r["reason"] != "hash-mismatch" {
		t.Fatalf("hash 校验失败应拒: %v", r)
	}
	// 未知 id
	if r := a.VoiceV2ExecuteAction("vap_none", "x"); r["reason"] != "no-preview" {
		t.Fatalf("未知预览应拒: %v", r)
	}
}

func TestVoiceV2ExecuteActionTTLExpiry(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	res := a.VoiceV2PreviewIntent("帮我记个任务:季度盘点")
	id, _ := res["previewId"].(string)
	hash, _ := res["intentHash"].(string)
	voiceV2PreviewMu.Lock()
	voiceV2PreviewLedger[id].expiresAt = time.Now().Add(-time.Second)
	voiceV2PreviewMu.Unlock()
	if r := a.VoiceV2ExecuteAction(id, hash); r["reason"] != "expired" {
		t.Fatalf("过期应拒: %v", r)
	}
	voiceV2PreviewMu.Lock()
	_, alive := voiceV2PreviewLedger[id]
	voiceV2PreviewMu.Unlock()
	if alive {
		t.Fatalf("过期条目应被清理")
	}
}

func TestVoiceV2PreviewNonWhitelistNotExecutable(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	// 生图=白名单外（roadmap：留到单独权限设计之后）：可预览不可执行、不写台账
	res := a.VoiceV2PreviewIntent("画一张日落图")
	if res["handled"] != true || res["executable"] != false {
		t.Fatalf("生图应可预览不可执行: %v", res)
	}
	if _, hasID := res["previewId"]; hasID {
		t.Fatalf("白名单外不得发预览凭据: %v", res)
	}
	voiceV2PreviewMu.Lock()
	n := len(voiceV2PreviewLedger)
	voiceV2PreviewMu.Unlock()
	if n != 0 {
		t.Fatalf("白名单外不得写台账: %d", n)
	}
}
