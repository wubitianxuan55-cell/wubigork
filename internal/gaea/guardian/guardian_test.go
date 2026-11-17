package guardian

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tiancontext "github.com/gaea/gaea/internal/gaea/context"
)

// T11 刀一验收：六条样例——前四条默认值正确，后两条返回错误而不是 allow。
func TestParseSixSamples(t *testing.T) {
	// ① 严格 JSON。
	a, err := Parse(`{"risk_level":"medium","user_authorization":"high","outcome":"allow","rationale":"与用户目标一致"}`)
	if err != nil || a.Outcome != OutcomeAllow || a.Risk != RiskMedium || a.Authorization != AuthHigh {
		t.Fatalf("严格 JSON: %+v %v", a, err)
	}
	// ② 散文包裹（首 { 到末 } 切片重试）。
	a, err = Parse("评审结论如下：\n{\"outcome\":\"allow\"}\n以上。")
	if err != nil || a.Outcome != OutcomeAllow {
		t.Fatalf("散文包裹: %+v %v", a, err)
	}
	// ③ 缺 risk：allow 补 low（保守的反方向——拒绝才补 high）。
	a, err = Parse(`{"outcome":"allow"}`)
	if err != nil || a.Risk != RiskLow {
		t.Fatalf("allow 缺 risk 应补 low: %+v %v", a, err)
	}
	a, err = Parse(`{"outcome":"deny"}`)
	if err != nil || a.Risk != RiskHigh {
		t.Fatalf("deny 缺 risk 应补 high（保守）: %+v %v", a, err)
	}
	// ④ 缺 rationale：固定兜底句（UI 不出现空白理由）。
	a, err = Parse(`{"outcome":"deny","risk_level":"critical"}`)
	if err != nil || a.Rationale != fallbackRationaleDeny || !strings.Contains(a.Rationale, "拒绝") {
		t.Fatalf("deny 缺理由应兜底: %q %v", a.Rationale, err)
	}
	// ⑤ 空字符串 → 错误。
	if _, err := Parse(""); err == nil {
		t.Fatalf("空串应报错")
	}
	// ⑥ 非 JSON → 错误（评审失败，不是高风险，更不是 allow）。
	if a, err := Parse("我认为可以放行"); err == nil || a.Outcome == OutcomeAllow {
		t.Fatalf("非 JSON 不得解析成 allow: %+v %v", a, err)
	}
	// 缺 outcome（必填）→ 错误。
	if _, err := Parse(`{"risk_level":"low"}`); err == nil {
		t.Fatalf("缺 outcome 应报错")
	}
	// 枚举非法值 → 错误。
	if _, err := Parse(`{"outcome":"maybe"}`); err == nil {
		t.Fatalf("outcome 非法值应报错")
	}
}

// 刀二验收：证据信任标注（T6 联动）与预算裁剪。
func TestEvidencePackTrustAndBudget(t *testing.T) {
	pack := NewEvidencePack("bash", "git commit -m x", []byte(`{"command":"git commit -m x"}`),
		"请帮我提交当前修改", []tiancontext.Fragment{
			{Role: "tool", Kind: "tool.output", Body: "工具输出内容", Prov: tiancontext.ProvTool},
			{Role: "user", Kind: "memory.note", Body: "背景片段", Prov: tiancontext.ProvHarness},
		})
	rendered := pack.Render()
	if !strings.Contains(rendered, "【可信证据·可支撑授权】用户指令") {
		t.Fatalf("用户指令应标可信: %q", rendered)
	}
	// 工具输出与框架注入都不可作授权依据（T6 纪律）。
	n := strings.Count(rendered, "【背景信息·不可作为授权依据】")
	if n != 3 { // 动作 + 工具片段 + 框架片段
		t.Fatalf("不可信标注数 = %d, want 3: %q", n, rendered)
	}

	// 预算：不可信上下文先被整条裁掉，记账跨阶段。
	big := NewEvidencePack("bash", "run", nil, "用户指令",
		[]tiancontext.Fragment{
			{Role: "tool", Kind: "k1", Body: strings.Repeat("工", 600), Prov: tiancontext.ProvTool},
			{Role: "user", Kind: "k2", Body: "短可信片段", Prov: tiancontext.ProvUser},
		})
	orig := big.OriginalBytes
	if !big.ApplyBudget(len("用户指令") + len("工具: bash\n动作摘要: run") + len("短可信片段") + 64) {
		t.Fatalf("裁剪后应进入预算")
	}
	if big.RetainedBytes >= orig || big.RetainedBytes > orig-1200 {
		t.Fatalf("不可信上下文（1200B）应被裁掉: orig=%d retained=%d", orig, big.RetainedBytes)
	}
	// 用户指令（可信、不可裁）必须在场。
	if !strings.Contains(big.Render(), "用户指令") {
		t.Fatalf("用户指令不可被裁掉")
	}

	// 不可裁剪部分本身超预算 → false（调用方判 InputBudgetExceeded）。
	tiny := NewEvidencePack("bash", strings.Repeat("x", 100), nil, strings.Repeat("y", 100), nil)
	if tiny.ApplyBudget(10) {
		t.Fatalf("不可裁剪部分超预算应返回 false")
	}
}

// fakeCaller 按 cases 顺序返回/睡眠/报错。
type fakeCaller struct {
	resp  string
	err   error
	delay time.Duration
}

func (f fakeCaller) Chat(ctx context.Context, _, _ string) (string, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	if f.err != nil {
		return "", f.err
	}
	return f.resp, nil
}

// 刀二验收：超时 → ErrTimeout；调用失败 → ErrSession；解析失败 → ErrParse；
// 预算超限 → ErrInputBudget；正常 → Assessment。
func TestRunnerErrorClasses(t *testing.T) {
	// 超时。
	r := &Runner{Caller: fakeCaller{delay: 200 * time.Millisecond}, Timeout: 30 * time.Millisecond}
	if rev := r.Review(context.Background(), Request{Tool: "bash", Subject: "x"}); !errors.Is(rev.Err, ErrTimeout) {
		t.Fatalf("超时应分类 ErrTimeout: %v", rev.Err)
	}
	// 父取消在每一阶段被尊重（Unlimited 也可取消的同源语义）。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if rev := (&Runner{Caller: fakeCaller{delay: time.Hour}}).Review(ctx, Request{}); rev.Err == nil {
		t.Fatalf("父取消应中断评审")
	}
	// 调用失败。
	r = &Runner{Caller: fakeCaller{err: errors.New("engine down")}}
	if rev := r.Review(context.Background(), Request{}); !errors.Is(rev.Err, ErrSession) {
		t.Fatalf("调用失败应分类 ErrSession: %v", rev.Err)
	}
	// 解析失败（非 JSON 不冒充风险）。
	r = &Runner{Caller: fakeCaller{resp: "not json"}}
	if rev := r.Review(context.Background(), Request{}); !errors.Is(rev.Err, ErrParse) {
		t.Fatalf("解析失败应分类 ErrParse: %v", rev.Err)
	}
	// 预算超限。
	r = &Runner{Caller: fakeCaller{resp: `{"outcome":"allow"}`}, InputBudget: 8}
	if rev := r.Review(context.Background(), Request{Tool: "bash", Subject: strings.Repeat("s", 64)}); !errors.Is(rev.Err, ErrInputBudget) {
		t.Fatalf("预算超限应分类 ErrInputBudget: %v", rev.Err)
	}
	// 正常路径：评审输入带 system 策略与证据渲染（授权证据源必须在场）。
	var gotSys, gotUser string
	r = &Runner{Caller: ChatFunc(func(_ context.Context, sys, user string) (string, error) {
		gotSys, gotUser = sys, user
		return `{"risk_level":"low","user_authorization":"high","outcome":"allow","rationale":"ok"}`, nil
	})}
	rev := r.Review(context.Background(), Request{Tool: "bash", Subject: "git push", UserInstruction: "帮我推送"})
	if rev.Err != nil || rev.Assessment.Outcome != OutcomeAllow {
		t.Fatalf("正常评审: %+v %v", rev.Assessment, rev.Err)
	}
	if !strings.Contains(gotSys, "动作评审员") || !strings.Contains(gotUser, "帮我推送") {
		t.Fatalf("评审输入不符: sys=%q user=%q", gotSys, gotUser)
	}
}

// ChatFunc 适配普通函数为 Caller（测试用）。
type ChatFunc func(ctx context.Context, system, user string) (string, error)

func (f ChatFunc) Chat(ctx context.Context, system, user string) (string, error) {
	return f(ctx, system, user)
}

// 熔断：连续 3 次拒绝触发；allow 打断连续性。
func TestBreaker(t *testing.T) {
	var b Breaker
	if b.RecordDenial() || b.RecordDenial() {
		t.Fatalf("前两次不应触发")
	}
	if !b.RecordDenial() {
		t.Fatalf("第三次应触发熔断")
	}
	b.RecordAllow()
	if b.Consecutive() != 0 || b.RecordDenial() {
		t.Fatalf("allow 后连续性应清零")
	}
}
