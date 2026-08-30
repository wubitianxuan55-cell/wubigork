package control

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// reqPermStubTool 是 request_permission 测试用的最小注册表工具桩。
type reqPermStubTool struct{ name string }

func (t reqPermStubTool) Name() string            { return t.name }
func (t reqPermStubTool) Description() string     { return "stub" }
func (t reqPermStubTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t reqPermStubTool) ReadOnly() bool          { return true }
func (t reqPermStubTool) Execute(_ context.Context, _ json.RawMessage) (string, error) {
	return "", nil
}

// requestPermHarness 装配一个只服务 request_permission 测试的控制器：
// 审批卡事件进入 apCh（带缓冲，测超时/拒绝路径不会塞满），Notice 计数。
type requestPermHarness struct {
	c       *Controller
	cards   chan event.Approval
	notices chan struct{}
}

func newRequestPermHarness(t *testing.T, mutate func(*Options)) *requestPermHarness {
	t.Helper()
	h := &requestPermHarness{
		cards:   make(chan event.Approval, 8),
		notices: make(chan struct{}, 8),
	}
	opts := Options{Sink: event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.ApprovalRequest:
			h.cards <- e.Approval
		case event.Notice:
			select {
			case h.notices <- struct{}{}:
			default:
			}
		}
	})}
	if mutate != nil {
		mutate(&opts)
	}
	h.c = New(opts)
	return h
}

// waitCard 等一张审批卡（超时即失败）；when 为空串时断言不再有新卡。
func (h *requestPermHarness) waitCard(t *testing.T, when string) event.Approval {
	t.Helper()
	select {
	case ap := <-h.cards:
		return ap
	case <-time.After(2 * time.Second):
		t.Fatalf("request_permission %s: 未产生审批卡", when)
		return event.Approval{}
	}
}

// assertNoCard 断言短时间内没有新审批卡弹出。
func (h *requestPermHarness) assertNoCard(t *testing.T, when string) {
	t.Helper()
	select {
	case ap := <-h.cards:
		t.Fatalf("request_permission %s: 不应弹卡，却弹出了 %q", when, ap.Tool)
	case <-time.After(150 * time.Millisecond):
	}
}

// TestRequestPermissionAllowSession 覆盖主链路：工具触发 → 审批卡带 Request
// 标记与 reason → allow_session → 会话规则生效（glob 匹配），同 subject 二次
// 申请短路不弹卡。
func TestRequestPermissionAllowSession(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	ctx := context.Background()

	type result struct {
		granted  bool
		decision string
		err      error
	}
	res := make(chan result, 1)
	go func() {
		g, d, err := h.c.RequestPermission(ctx, "bash", "go build*", "需要跑 go build 验证改动")
		res <- result{g, d, err}
	}()

	card := h.waitCard(t, "allow_session")
	if card.Tool != "bash" || card.Subject != "go build*" {
		t.Fatalf("卡片目标不符: %+v", card)
	}
	if !card.Request {
		t.Fatal("卡片缺少 Request 标记（前端无法识别为权限升级申请）")
	}
	if card.Reason == "" || card.Reason != "需要跑 go build 验证改动" {
		t.Fatalf("卡片 reason 应原样携带申请理由，got %q", card.Reason)
	}
	h.c.Approve(card.ID, DecisionAllowSession)

	r := <-res
	if r.err != nil || !r.granted || r.decision != DecisionAllowSession {
		t.Fatalf("RequestPermission = (%v,%q,%v), want (true, allow_session, nil)", r.granted, r.decision, r.err)
	}

	// 会话规则已生效：同规则二次申请短路（不弹第二张卡）。
	h.c.mu.Lock()
	covered := h.c.ruleGrantedLocked("bash", "go build*")
	coveredCall := h.c.ruleGrantedLocked("bash", "go build ./...")
	uncovered := h.c.ruleGrantedLocked("bash", "rm -rf /")
	coversOtherTool := h.c.ruleGrantedLocked("write_file", "go build ./...")
	h.c.mu.Unlock()
	if !covered || !coveredCall {
		t.Fatal("allow_session 后规则应覆盖相同与 glob 匹配的 subject")
	}
	if uncovered {
		t.Fatal("规则不应覆盖不匹配的 subject")
	}
	if coversOtherTool {
		t.Fatal("规则不应外溢到其它工具")
	}

	// 二次申请：命中会话规则，立即 granted，不弹卡。
	granted, decision, err := h.c.RequestPermission(ctx, "bash", "go build*", "重复申请")
	if err != nil || !granted || decision != DecisionAllowSession {
		t.Fatalf("二次申请 = (%v,%q,%v), want 短路 allow_session", granted, decision, err)
	}
	h.assertNoCard(t, "二次申请")
}

// TestRequestPermissionGrantFeedsNormalGate 硬纪律主验证：批准授予的是「规则」，
// 后续真实工具调用仍走正常闸门——规则满足则自然放行（无新卡），不满足则照常
// 弹卡。
func TestRequestPermissionGrantFeedsNormalGate(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	ctx := context.Background()

	done := make(chan bool, 1)
	go func() {
		g, _, _ := h.c.RequestPermission(ctx, "bash", "go build*", "构建验证")
		done <- g
	}()
	card := h.waitCard(t, "gate 联动")
	h.c.Approve(card.ID, DecisionAllowSession)
	if !<-done {
		t.Fatal("申请应被批准")
	}

	// 闸门路径：匹配规则的调用自然放行，且没有新卡（走 granted 短路而非弹卡）。
	allow, remember, err := gateApprover{h.c}.Approve(ctx, "bash", "go build ./...", nil)
	if err != nil || !allow {
		t.Fatalf("规则覆盖的闸门调用应放行: (%v,%v)", allow, err)
	}
	_ = remember
	h.assertNoCard(t, "规则覆盖的闸门调用")

	// 不匹配的 subject：仍走正常闸门弹卡（规则没有放大授权面）。
	blocked := make(chan bool, 1)
	go func() {
		allow, _, _ := gateApprover{h.c}.Approve(ctx, "bash", "curl evil.example", nil)
		blocked <- allow
	}()
	card2 := h.waitCard(t, "未覆盖 subject")
	h.c.Approve(card2.ID, DecisionDeny)
	if <-blocked {
		t.Fatal("规则未覆盖的闸门调用不应被放行")
	}
}

// TestRequestPermissionAllowOnce 覆盖规则申请语义下 allow_once 与 allow_session
// 同效（写入会话规则，否则「允许一次」无可落地效果）。
func TestRequestPermissionAllowOnce(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		_, _, err := h.c.RequestPermission(ctx, "bash", "go vet*", "静态检查")
		done <- err
	}()
	card := h.waitCard(t, "allow_once")
	h.c.Approve(card.ID, DecisionAllowOnce)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	h.c.mu.Lock()
	covered := h.c.ruleGrantedLocked("bash", "go vet ./...")
	h.c.mu.Unlock()
	if !covered {
		t.Fatal("allow_once 对规则申请应写入会话 granted")
	}
}

// TestRequestPermissionPersistAllow 回写策略：persist_allow → PersistAllowRule
// 收到 "Tool(subject)" 规则串 + 会话规则生效。
func TestRequestPermissionPersistAllow(t *testing.T) {
	var gotRule string
	var gotErr error
	h := newRequestPermHarness(t, func(o *Options) {
		o.PersistAllowRule = func(rule string) error {
			gotRule = rule
			return gotErr
		}
	})
	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		_, _, err := h.c.RequestPermission(ctx, "bash", "go build*", "构建验证")
		done <- err
	}()
	card := h.waitCard(t, "persist_allow")
	h.c.Approve(card.ID, DecisionPersistAllow)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if gotRule != "bash(go build*)" {
		t.Fatalf("回写规则 = %q, want %q", gotRule, "bash(go build*)")
	}
	h.c.mu.Lock()
	covered := h.c.ruleGrantedLocked("bash", "go build ./...")
	h.c.mu.Unlock()
	if !covered {
		t.Fatal("persist_allow 后会话内规则应生效")
	}
}

// TestRequestPermissionDeny 覆盖拒绝语义：不写 granted、回合继续（不取消）。
func TestRequestPermissionDeny(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	ctx := context.Background()

	type result struct {
		granted  bool
		decision string
	}
	res := make(chan result, 1)
	go func() {
		g, d, _ := h.c.RequestPermission(ctx, "bash", "curl *", "拉取依赖")
		res <- result{g, d}
	}()
	card := h.waitCard(t, "deny")
	h.c.Approve(card.ID, DecisionDeny)
	r := <-res
	if r.granted || r.decision != DecisionDeny {
		t.Fatalf("deny = (%v,%q), want (false, deny)", r.granted, r.decision)
	}
	h.c.mu.Lock()
	covered := h.c.ruleGrantedLocked("bash", "curl https://x")
	h.c.mu.Unlock()
	if covered {
		t.Fatal("deny 后不得写入会话 granted")
	}
}

// TestRequestPermissionAbort 覆盖 abort 语义：决策串如实返回且取消当前回合。
func TestRequestPermissionAbort(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	ctx := context.Background()

	cancelled := make(chan struct{}, 1)
	h.c.mu.Lock()
	h.c.cancel = func() { cancelled <- struct{}{} }
	h.c.mu.Unlock()

	type result struct {
		granted  bool
		decision string
	}
	res := make(chan result, 1)
	go func() {
		g, d, _ := h.c.RequestPermission(ctx, "bash", "rm -rf build/*", "清理产物")
		res <- result{g, d}
	}()
	card := h.waitCard(t, "abort")
	h.c.Approve(card.ID, DecisionAbort)
	r := <-res
	if r.granted || r.decision != DecisionAbort {
		t.Fatalf("abort = (%v,%q), want (false, abort)", r.granted, r.decision)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("abort 未触发回合取消")
	}
}

// TestRequestPermissionTimeout 覆盖 C4 超时语义：无人响应按拒绝处理，发 Notice，
// 不写 granted。
func TestRequestPermissionTimeout(t *testing.T) {
	h := newRequestPermHarness(t, func(o *Options) {
		o.ApprovalTimeout = 40 * time.Millisecond
	})
	ctx := context.Background()

	type result struct {
		granted  bool
		decision string
	}
	res := make(chan result, 1)
	go func() {
		g, d, _ := h.c.RequestPermission(ctx, "bash", "go test*", "跑测试")
		res <- result{g, d}
	}()
	h.waitCard(t, "timeout") // 卡片已弹出但无人应答
	r := <-res
	if r.granted || r.decision != DecisionTimeout {
		t.Fatalf("timeout = (%v,%q), want (false, timeout)", r.granted, r.decision)
	}
	select {
	case <-h.notices:
	case <-time.After(time.Second):
		t.Fatal("超时应发 Notice 让用户回来能看到")
	}
	h.c.mu.Lock()
	covered := h.c.ruleGrantedLocked("bash", "go test ./...")
	h.c.mu.Unlock()
	if covered {
		t.Fatal("超时拒绝不得写入会话 granted")
	}
}

// TestRequestPermissionHardAskRefused 硬纪律：hardAsk 逐条确认工具不接受升级
// 申请——不弹卡、不写 granted（任何级别都不自动放行）。
func TestRequestPermissionHardAskRefused(t *testing.T) {
	h := newRequestPermHarness(t, nil) // 包级默认 hardAsk 集含 remember
	granted, decision, err := h.c.RequestPermission(
		context.Background(), "remember", "", "想把这件事记成长期记忆")
	if err != nil || granted || decision != "refused_hardask" {
		t.Fatalf("hardAsk 目标 = (%v,%q,%v), want (false, refused_hardask, nil)", granted, decision, err)
	}
	h.assertNoCard(t, "hardAsk 拒绝")

	// yolo 下同样拒绝：hardAsk 纪律凌驾于权限级别之上。
	h.c.SetPermLevel("yolo")
	granted, decision, _ = h.c.RequestPermission(context.Background(), "remember", "", "再试一次")
	if granted || decision != "refused_hardask" {
		t.Fatalf("yolo 下 hardAsk 目标 = (%v,%q), want 拒绝", granted, decision)
	}
}

// TestRequestPermissionDenyRuleRefused 硬纪律：deny 规则命中的目标不弹卡直接
// 拒绝——批准的规则盖不过 Decide 的 Deny 优先级，弹卡只会误导。
func TestRequestPermissionDenyRuleRefused(t *testing.T) {
	h := newRequestPermHarness(t, func(o *Options) {
		o.Policy = permission.New("ask", nil, nil, []string{"bash(rm -rf*)"})
	})
	granted, decision, err := h.c.RequestPermission(
		context.Background(), "bash", "rm -rf /tmp/x", "想清理目录")
	if err != nil || granted || decision != "refused_deny_rule" {
		t.Fatalf("deny 规则目标 = (%v,%q,%v), want (false, refused_deny_rule, nil)", granted, decision, err)
	}
	h.assertNoCard(t, "deny 规则拒绝")

	// 不命中 deny 规则的 subject 正常走审批卡。
	done := make(chan bool, 1)
	go func() {
		g, _, _ := h.c.RequestPermission(context.Background(), "bash", "go build*", "构建验证")
		done <- g
	}()
	card := h.waitCard(t, "deny 规则未命中")
	h.c.Approve(card.ID, DecisionAllowSession)
	if !<-done {
		t.Fatal("未命中 deny 规则的申请应正常弹卡获批")
	}
}

// TestRequestPermissionUnknownToolRefused：未注册工具的申请直接拒绝（规则永远
// 空转，不误导用户）。
func TestRequestPermissionUnknownToolRefused(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Add(reqPermStubTool{name: "bash"})
	h := newRequestPermHarness(t, func(o *Options) { o.Registry = reg })

	granted, decision, err := h.c.RequestPermission(
		context.Background(), "nonexistent_tool", "", "不存在的工具")
	if err != nil || granted || decision != "refused_unknown_tool" {
		t.Fatalf("未注册工具 = (%v,%q,%v), want (false, refused_unknown_tool, nil)", granted, decision, err)
	}
	h.assertNoCard(t, "未注册工具")

	// 已注册工具正常弹卡。
	done := make(chan bool, 1)
	go func() {
		g, _, _ := h.c.RequestPermission(context.Background(), "bash", "", "整工具授权")
		done <- g
	}()
	card := h.waitCard(t, "已注册工具")
	if card.Subject != "" {
		t.Fatalf("空 subject 应原样为空，got %q", card.Subject)
	}
	h.c.Approve(card.ID, DecisionAllowSession)
	if !<-done {
		t.Fatal("已注册工具的申请应获批")
	}
}

// TestRequestPermissionAutoLevelNoPrompt：auto/yolo 级别下真实调用本就不再
// 询问，申请直接按会话规则生效（decision=auto），不打扰用户。
func TestRequestPermissionAutoLevelNoPrompt(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	h.c.SetPermLevel("auto")

	granted, decision, err := h.c.RequestPermission(
		context.Background(), "bash", "go build*", "构建验证")
	if err != nil || !granted || decision != "auto" {
		t.Fatalf("auto 级别 = (%v,%q,%v), want (true, auto, nil)", granted, decision, err)
	}
	h.assertNoCard(t, "auto 级别")
	h.c.mu.Lock()
	covered := h.c.ruleGrantedLocked("bash", "go build ./...")
	h.c.mu.Unlock()
	if !covered {
		t.Fatal("auto 级别下批准的规则应写入会话规则表")
	}
}

// TestRequestPermissionCtxCancel：ctx 取消（回合终止）时返回 err 且清理挂起卡。
func TestRequestPermissionCtxCancel(t *testing.T) {
	h := newRequestPermHarness(t, nil)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, _, err := h.c.RequestPermission(ctx, "bash", "go build*", "构建验证")
		done <- err
	}()
	card := h.waitCard(t, "ctx cancel")
	cancel()
	if err := <-done; err == nil {
		t.Fatal("ctx 取消应返回 err")
	}
	h.c.mu.Lock()
	_, pending := h.c.approvals[card.ID]
	h.c.mu.Unlock()
	if pending {
		t.Fatal("ctx 取消后挂起审批未清理")
	}
}
