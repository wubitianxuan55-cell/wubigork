package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ── 2026-11 P2 执行批：按调用路由授权 + send_message 中期回报通道 ──────────

func TestSubagentRoutesAuth(t *testing.T) {
	t.Cleanup(func() { SetSubagentRoutes(nil) })

	// 清单空=路由整体关闭（上游「无记录政策时只许父路由」）
	SetSubagentRoutes(nil)
	if err := AuthorizeSubagentRoute("strata/qwen3-32b"); err == nil {
		t.Fatal("空清单必须拒绝一切按调用路由")
	} else if !strings.Contains(err.Error(), "subagent_routes") {
		t.Fatalf("拒绝文案应指路配置键：%v", err)
	}

	SetSubagentRoutes([]string{"strata/qwen3-32b", "", "xai/grok-4.3", "strata/qwen3-32b"})
	got := SubagentRoutes()
	if len(got) != 2 || got[0] != "strata/qwen3-32b" || got[1] != "xai/grok-4.3" {
		t.Fatalf("清单应剔空保序去重：%v", got)
	}

	if err := AuthorizeSubagentRoute(" xai/grok-4.3 "); err != nil {
		t.Fatalf("清单内路由（trim 后逐字匹配）应通过：%v", err)
	}
	if err := AuthorizeSubagentRoute("ollama/llama3"); err == nil {
		t.Fatal("清单外路由必须拒绝")
	} else if !strings.Contains(err.Error(), "strata/qwen3-32b") {
		t.Fatalf("fail-loud 应点名可用路由：%v", err)
	}
}

// 路由校验发生在任何 LLM 调用之前：factory 报错/授权拒绝都当场返回。
func TestTaskRouteArgsValidation(t *testing.T) {
	parentReg := tool.NewRegistry()
	var factoryEngine, factoryModel string
	task := NewTaskTool(&mockProvider{name: "p"}, nil, parentReg, 10, 0, 0.0, "", "sys", nil)
	task.SetSubagentRouteFactory(func(engine, model string) (provider.LLMProvider, error) {
		factoryEngine, factoryModel = engine, model
		return nil, errors.New("bridge not wired")
	})
	t.Cleanup(func() { SetSubagentRoutes(nil) })

	// 只给一个参数：须同时给
	_, err := task.Execute(context.Background(), []byte(`{"prompt":"do it","engine":"strata"}`))
	if err == nil || !strings.Contains(err.Error(), "together") {
		t.Fatalf("engine/model 须同时给：%v", err)
	}

	// 清单空（未授权）：拒绝且不触 factory
	SetSubagentRoutes(nil)
	_, err = task.Execute(context.Background(), []byte(`{"prompt":"do it","engine":"strata","model":"qwen3-32b"}`))
	if err == nil || !strings.Contains(err.Error(), "not authorized") && !strings.Contains(err.Error(), "routing is not authorized") {
		t.Fatalf("空清单应拒绝：%v", err)
	}
	if factoryEngine != "" {
		t.Fatal("拒绝路径不得触达 factory")
	}

	// 授权路由：factory 被调（此处 factory 报错=当场如实失败，运行不开始）
	SetSubagentRoutes([]string{"strata/qwen3-32b"})
	_, err = task.Execute(context.Background(), []byte(`{"prompt":"do it","engine":"strata","model":"qwen3-32b"}`))
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("授权路由应触达 factory 并如实报错：%v", err)
	}
	if factoryEngine != "strata" || factoryModel != "qwen3-32b" {
		t.Fatalf("factory 参数异常：%s/%s", factoryEngine, factoryModel)
	}

	// CompactDescription 发现面：配置后动态列出授权路由
	if cd := task.CompactDescription(); !strings.Contains(cd, "strata/qwen3-32b") {
		t.Fatalf("CompactDescription 应列出授权路由：%s", cd)
	}
	SetSubagentRoutes(nil)
	if cd := task.CompactDescription(); strings.Contains(cd, "可按授权路由") {
		t.Fatalf("清单空时发现面应缺席：%s", cd)
	}
}

func TestSendMessageDelivery(t *testing.T) {
	parent := testAgentC(4000, 0.8, 5)
	ref := "sa_20261102_test"

	// 署名信封：带 ref、带消息体、带防洗白条款（不是用户输入/无授权效力）
	parent.SteerFromSubagent(ref, "中间发现：表 A 有三列缺失")
	parent.steerMu.Lock()
	items := append([]steerItem(nil), parent.steerQueue...)
	parent.steerMu.Unlock()
	if len(items) != 1 {
		t.Fatalf("入队数异常：%d", len(items))
	}
	if items[0].origin != steerFromSubagent {
		t.Fatalf("来源应为 steerFromSubagent：%v", items[0].origin)
	}
	env := items[0].text
	for _, want := range []string{ref, "三列缺失", "NOT from the user"} {
		if !strings.Contains(env, want) {
			t.Fatalf("信封缺 %q：%s", want, env)
		}
	}

	// 工具面：ctx 兜底取宿主执行器 + ref，返回送达回执非答案
	tl := &sendMessageTool{}
	res, err := tl.Execute(WithRunner(WithSubagentRef(context.Background(), ref), parent), []byte(`{"message":"进度 50%，表 B 可跳过"}`))
	if err != nil {
		t.Fatalf("送达失败：%v", err)
	}
	if !strings.Contains(res, "delivered") || strings.Contains(res, "进度 50%") {
		t.Fatalf("回执不应回显消息全文：%s", res)
	}
	parent.steerMu.Lock()
	n := len(parent.steerQueue)
	parent.steerMu.Unlock()
	if n != 2 {
		t.Fatalf("第二次投递后队列长度异常：%d", n)
	}

	// 空消息/无宿主：诚实拒绝
	if _, err := tl.Execute(context.Background(), []byte(`{"message":"  "}`)); err == nil {
		t.Fatal("空消息必须拒绝")
	}
	_, err = (&sendMessageTool{}).Execute(context.Background(), []byte(`{"message":"hi"}`))
	if err == nil || !strings.Contains(err.Error(), "requires a live parent agent") {
		t.Fatalf("无宿主应如实拒绝：%v", err)
	}
}
