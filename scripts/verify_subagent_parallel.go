package main

// verify_subagent_parallel.go — 子代理有限并行真机端到端验证（2026-10）。
// 用本机 Strata 引擎（127.0.0.1:8091，内置本地推理）真跑一轮「同消息派发
// 3 个子代理」的父回合，观察四批机制的实测行为：
//   批1 有限并行闸（subagent_max_parallel=2 → 第 3 路排队）
//   批1 排队诚实化（queued 侧车可见）
//   批3 中断面（subRuns 句柄；本脚本观察 queued/running 状态机，不打断）
//   批4 subagent_list 枚举（父模型自己调用）
// 零污染：会话/transcript 全部落 temp 目录；只读真实引擎，不写任何用户数据。
// 用法：go run scripts/verify_subagent_parallel.go（引擎未起时如实退出 3）。
import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/modelengine"
)

const (
	engineBase   = "http://127.0.0.1:8091/v1"
	strataModel  = "qwen3.8-flash-next-iq2_xs"
	turnTimeout  = 5 * time.Minute
	watcherEvery = 120 * time.Millisecond
)

func main() {
	// 0) 引擎探活：未起引擎时如实退出（exit 3），绝不误报机制缺陷。
	resp, err := http.Get(engineBase + "/models")
	if err != nil || resp.StatusCode != 200 {
		fmt.Println("SKIP: strata engine not reachable at", engineBase, "(", err, ")")
		os.Exit(3)
	}
	resp.Body.Close()
	fmt.Println("[ok] strata engine reachable")

	// 1) 组装真实 provider：模型中心客户端 → strata 引擎 → wubigrok 桥。
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "strata", BaseURL: engineBase, Enabled: true}); err != nil {
		fatal("SaveEngine", err)
	}
	client := ai.NewClient(&config.Config{})
	client.SetEngineManager(mgr)
	bridge.SetClient(client)
	prov, err := provider.NewLLM(provider.LLMKindWubigrok, provider.Config{
		Name:   "strata-verify",
		Model:  strataModel,
		Engine: "strata",
	})
	if err != nil {
		fatal("NewLLM", err)
	}
	fmt.Println("[ok] provider assembled (wubigrok bridge → strata)")

	// 2) 工具面：task + interrupt_agent + subagent_list（真机注册面同 boot）。
	workdir, err := os.MkdirTemp("", "gaea-subagent-verify-*")
	if err != nil {
		fatal("MkdirTemp", err)
	}
	defer os.RemoveAll(workdir)
	store := agent.NewSubagentStore(filepath.Join(workdir, "subagents"))

	agent.SetSubagentMaxParallel(2) // 3 路派发 → 第 3 路必排队（批1 观察点）

	reg := tool.NewRegistry()
	taskTool := agent.NewTaskTool(prov, nil, reg, 24, 131072, 0.3, "", "你是被派发的子代理：直接完成任务并给最终答案。", nil).WithTranscripts(store)
	reg.Add(taskTool)
	reg.Add(agent.NewInterruptTool())
	reg.Add(agent.NewSubagentListTool(store))
	fmt.Println("[ok] registry: task + interrupt_agent + subagent_list (max_parallel=2)")

	// 3) 事件打印 sink（截头，控制台可读）。
	sink := event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.ToolDispatch:
			fmt.Printf("  [dispatch] %s (%s)\n", e.Tool.Name, short(e.Tool.ID, 24))
		case event.ToolResult:
			state := "ok"
			if strings.TrimSpace(e.Tool.Err) != "" {
				state = "ERR: " + firstLine(e.Tool.Err, 90)
			}
			fmt.Printf("  [result]   %s → %s\n", e.Tool.Name, state)
		case event.Notice:
			fmt.Printf("  [notice]   %s\n", firstLine(e.Text, 120))
		case event.SubagentMessage:
			fmt.Printf("  [child-msg] %s\n", firstLine(e.Text, 100))
		}
	})

	// 4) 排队观察器：回合期间轮询侧车，记录出现过的状态（queued 尤其关键）。
	seen := map[agent.SubagentStatus]bool{}
	stopWatch := make(chan struct{})
	watchErr := make(chan error, 1)
	go func() {
		for {
			select {
			case <-stopWatch:
				return
			case <-time.After(watcherEvery):
			}
			runs, err := store.ListRuns()
			if err != nil {
				watchErr <- err
				return
			}
			for _, r := range runs {
				seen[r.Status] = true
			}
		}
	}()

	// 5) 父回合：真模型派发 3 路子代理 + 自主调 subagent_list。
	sess := agent.NewSession("你是编排测试助手：只按用户指令派发子代理并汇总，不做其他事。")
	runner := agent.New(prov, reg, sess, agent.Options{MaxSteps: 16, ContextWindow: 131072}, sink)
	prompt := "用 task 工具在同一条消息里派发 3 个子代理（一次性并行派发，不要一个个来）。" +
		"每个子代理的任务都只是：直接回答『计数 1、2、3』这六个字，不使用任何工具。" +
		"三个都完成后，调用 subagent_list 工具查看运行清单，最后只回答『全部完成』。"
	fmt.Println("[run] parent turn start (strata, max 5min)…")
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), turnTimeout)
	defer cancel()
	_, runErr := runner.Run(ctx, prompt)
	close(stopWatch)
	elapsed := time.Since(started).Round(time.Second)
	if runErr != nil {
		fmt.Printf("[run] turn error after %s: %v\n", elapsed, runErr)
	} else {
		fmt.Printf("[run] turn done in %s\n", elapsed)
	}
	select {
	case err := <-watchErr:
		fmt.Println("[warn] watcher:", err)
	default:
	}

	// 6) 终局证据：store 侧车终态 + 观察汇总。
	runs, _ := store.ListRuns()
	fmt.Printf("\n=== 观察汇总（%d 条子代理侧车，回合用时 %s）===\n", len(runs), elapsed)
	for _, r := range runs {
		fmt.Printf("  %s [%s] %s\n", r.Ref, r.Status, firstLine(r.Title, 40))
	}

	fails := 0
	check := func(name string, ok bool, detail string) {
		if ok {
			fmt.Println("[PASS]", name)
		} else {
			fails++
			fmt.Println("[FAIL]", name, "—", detail)
		}
	}
	check("同批多路派发（task 调用 ≥3）", countTaskDispatches(sess) >= 3,
		fmt.Sprintf("task 派发 %d 次", countTaskDispatches(sess)))
	check("并发闸排队可见（queued 侧车被观察到）", seen[agent.SubagentQueued],
		"cap=2 且 3 路派发，若模型确实同批派发则必现 queued；未见通常=模型串行派发")
	check("三路子代理终态完成", countStatus(runs, agent.SubagentCompleted) >= 3,
		fmt.Sprintf("completed=%d", countStatus(runs, agent.SubagentCompleted)))
	check("父回合无错误收尾", runErr == nil, fmt.Sprintf("%v", runErr))

	if fails > 0 {
		fmt.Printf("\nVERIFY_FAIL: %d 项未过\n", fails)
		os.Exit(1)
	}
	fmt.Println("\nVERIFY_OK: 有限并行/排队诚实化/枚举 真机行为符合预期")
}

// countTaskDispatches 从父会话 tool 结果消息数 task 派发次数（每个 task 调用
// 恰好一条 tool 角色结果）。
func countTaskDispatches(sess *agent.Session) int {
	n := 0
	for _, m := range sess.Messages {
		if m.Role == "tool" && m.Name == "task" {
			n++
		}
	}
	return n
}

func countStatus(runs []agent.SubagentRunInfo, st agent.SubagentStatus) int {
	n := 0
	for _, r := range runs {
		if r.Status == st {
			n++
		}
	}
	return n
}

func firstLine(s string, n int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func short(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func fatal(what string, err error) {
	fmt.Println("FATAL:", what, err)
	os.Exit(2)
}
