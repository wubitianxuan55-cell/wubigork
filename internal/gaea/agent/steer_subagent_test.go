package agent

// 在跑子代理直穿登记测试（v4.243 运行中直穿改向）：SteerSubagent 寻址语义——
// 查无登记如实报错；登记存在时注入 steer 队列（消费侧 shouldMidTurnSteer/
// consumeSteer 已有独立测试，这里钉注册/注销与寻址）。

import (
	"strings"
	"testing"
)

func TestSteerSubagentAddressing(t *testing.T) {
	// 查无此 ref：如实报错不静默
	if err := SteerSubagent("sa_ghost", "改一下口径"); err == nil || !strings.Contains(err.Error(), "不在运行") {
		t.Fatalf("未知 ref 应报错: %v", err)
	}
	// 空 ref 拒绝
	if err := SteerSubagent("", "x"); err == nil {
		t.Fatal("空 ref 应拒绝")
	}
	// 登记存在：注入成功（用真 AgentRunner 走 Store/Load 全链，不 mock 内部字段）
	sub := New(nil, nil, nil, Options{}, nil)
	subRuns.Store("sa_live", &subRunHandle{runner: sub, cancel: func() {}})
	defer subRuns.Delete("sa_live")
	if err := SteerSubagent("sa_live", "补充指引"); err != nil {
		t.Fatalf("在跑登记应注入成功: %v", err)
	}
	// 注入进的是该 runner 的 steer 队列（consumeSteer 可取回）
	text, ok := sub.consumeSteer()
	if !ok || text != "补充指引" {
		t.Fatalf("steer 队列内容 = %q ok=%v", text, ok)
	}
	// 注销后寻址失败（runSubAgentInternal defer Delete 的语义等价）
	subRuns.Delete("sa_live")
	if err := SteerSubagent("sa_live", "again"); err == nil {
		t.Fatal("注销后应寻址失败")
	}
	// 排队句柄（runner=nil）：不构成可改向的在跑登记
	subRuns.Store("sa_queued", &subRunHandle{cancel: func() {}})
	defer subRuns.Delete("sa_queued")
	if err := SteerSubagent("sa_queued", "x"); err == nil {
		t.Fatal("排队句柄（runner=nil）应寻址失败")
	}
}
