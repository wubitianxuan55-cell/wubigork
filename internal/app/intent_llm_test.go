package app

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/intent"
)

// LLM 兜底分类（v4.8）：默认关（宁漏勿误姿态 + 语音回路延迟）；开启后走
// seam 注入（测试不打真 LLM），命中复用既有执行层；dryRun 恒不调用。
func TestIntentLLMFallback(t *testing.T) {
	a := newChatServiceTestApp(t)

	ruleMiss := "把绘梦整出来" // 板块别名命中但无强导航动词——规则引擎确实漏
	if _, handled := a.routeIntent(ruleMiss); handled {
		t.Fatal("兜底关闭时规则未命中应走聊天（handled=false）")
	}

	// dry-run 恒不触发兜底（计数=0）
	var calls int32
	a.intentClassifierFn = func(text string) *intent.Intent {
		atomic.AddInt32(&calls, 1)
		return &intent.Intent{Action: intent.ActionNavigate, Target: "imagegen"}
	}
	a.cfg.SetIntentsLLMFallback(true)
	res := a.GaeaRouteIntent(ruleMiss, true)
	if res.Handled {
		t.Errorf("dry-run 不应预览兜底命中（预览-确认制口径）: %+v", res)
	}
	if n := atomic.LoadInt32(&calls); n != 0 {
		t.Fatalf("dry-run 不应调用分类器，实际 %d 次", n)
	}

	// 开启 + 兜底命中 navigate → 复用执行层（板块校验 + emit）
	reply, handled := a.routeIntent(ruleMiss)
	if !handled || !strings.Contains(reply, "绘梦") {
		t.Errorf("兜底命中应执行导航，reply=%q handled=%v", reply, handled)
	}
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Errorf("分类器应被调用 1 次，实际 %d", n)
	}

	// 兜底返回 navigate 但 target 不在 manifest → 按未命中
	a.intentClassifierFn = func(text string) *intent.Intent {
		atomic.AddInt32(&calls, 1)
		return &intent.Intent{Action: intent.ActionNavigate, Target: "nope"}
	}
	if _, handled := a.routeIntent(ruleMiss); handled {
		t.Error("兜底导航 target 不在 manifest 应按未命中")
	}

	// 兜底返回生图（白名单外）→ 按未命中
	a.intentClassifierFn = func(text string) *intent.Intent {
		return &intent.Intent{Action: intent.ActionGenerateImage, Target: "猫"}
	}
	if _, handled := a.routeIntent(ruleMiss); handled {
		t.Error("兜底不应执行白名单外动作")
	}

	// 开关再关闭 → 不再调用
	a.cfg.SetIntentsLLMFallback(false)
	a.intentClassifierFn = func(text string) *intent.Intent {
		atomic.AddInt32(&calls, 1)
		return &intent.Intent{Action: intent.ActionStatus, Target: "model"}
	}
	if _, handled := a.routeIntent(ruleMiss); handled {
		t.Error("开关关闭后兜底不应命中")
	}
	if n := atomic.LoadInt32(&calls); n != 2 {
		t.Errorf("关闭后分类器不应被调用（累计 2），实际 %d", n)
	}
}

// 默认配置：兜底关（测试夹具手工构造 Config，零值即关——宁漏勿误姿态
// 的安全侧；Load 默认 2000ms 由 config round-trip 测试覆盖）。
func TestIntentLLMFallbackDefaults(t *testing.T) {
	if config.KeyIntentsLLMFallback == "" {
		t.Fatal("config key 不应为空")
	}
	a := newChatServiceTestApp(t)
	if a.cfg.GetIntentsLLMFallback() {
		t.Error("意图 LLM 兜底默认应为关闭")
	}
}
