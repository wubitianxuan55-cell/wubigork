package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/intent"
)

// newVoiceV2ActionTestApp 构造可执行 navigate 的最小 App（manifest 里须有目标板块）。
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
	a := &App{core: &core{cfg: &config.Config{}}}
	a.mediaState = &mediaState{core: a.core}
	return a
}

func TestVoiceV2PreviewIntentWhitelistAndLedger(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	// navigate=白名单内：登记台账并返回 preview 凭据
	res := a.VoiceV2PreviewIntent("打开运行监测")
	if res["handled"] != true || res["executable"] != true {
		t.Fatalf("navigate 应可执行: %v", res)
	}
	id, _ := res["previewId"].(string)
	hash, _ := res["intentHash"].(string)
	if id == "" || hash == "" || res["ttlMs"] == float64(0) {
		t.Fatalf("预览凭据缺失: %v", res)
	}
	voiceV2PreviewMu.Lock()
	p := voiceV2PreviewLedger[id]
	voiceV2PreviewMu.Unlock()
	if p == nil || p.text != "打开运行监测" || p.action != string(intent.ActionNavigate) {
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

func TestVoiceV2ExecuteActionExactlyOnce(t *testing.T) {
	a := newVoiceV2ActionTestApp(t)
	res := a.VoiceV2PreviewIntent("打开运行监测")
	id, _ := res["previewId"].(string)
	hash, _ := res["intentHash"].(string)

	first := a.VoiceV2ExecuteAction(id, hash)
	if first["executed"] != true || !strings.Contains(first["reply"].(string), "运行监测") {
		t.Fatalf("首次执行应成功且带结果回执: %v", first)
	}
	// 幂等：同凭据第二次调用=already-executed（roadmap「确认后最多执行 1 次」）
	second := a.VoiceV2ExecuteAction(id, hash)
	if second["executed"] != false || second["reason"] != "already-executed" {
		t.Fatalf("重复执行必须被拒: %v", second)
	}
	// hash 不符（防篡改）
	res2 := a.VoiceV2PreviewIntent("打开运行监测")
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
	res := a.VoiceV2PreviewIntent("打开运行监测")
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
