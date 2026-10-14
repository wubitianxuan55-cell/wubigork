//go:build ignore
// +build ignore

package main

// verify_strata_speed.go — 真机验证本机 Strata 引擎（127.0.0.1:8091）的输出速度。
// 走 gaea 自家 ai 客户端路径（ChatStreamChunks + EngineID=strata，聊天页同款装配），
// tok/s 口径与 internal/app/chat_service.go streamTokensPerSecond 一致：
//   completion tokens（流结束块 usage）÷（首个内容/思考 delta → 流结束实耗），
//   四舍五入 0.1；usage 缺失或耗时非正值=0（不虚报）。
// 四档实测：短答（关思考）/ 长答（关思考）/ 思考缺省开 / 长提示词（首字延迟 TTFT）。
// 另附引擎自身日志的「generated in … tok/s」作独立旁证（只读）。
// 零污染：引擎只读；内存态引擎表（空 statePath 不落盘），不写任何用户数据。
// 引擎未起：如实退出 3（verify_strata_engine.go / verify_subagent_parallel.go 同款）。
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

const (
	engineBase    = "http://127.0.0.1:8091/v1"
	fallbackModel = "qwen3.8-flash-next-iq2_xs"
	engineLog     = `C:\AI\strata\repo\strata-iq2_xs.log`
	runTimeout    = 10 * time.Minute
)

// runResult 单档实测结果（tok/s 口径同聊天页）。
type runResult struct {
	name       string
	ttft       time.Duration // 发起 → 首个内容/思考 delta（含排队与预填充）
	genElapsed time.Duration // 首个 delta → 流结束（tok/s 计时的分母）
	promptTok  int64
	completion int64
	tps        float64
	content    string
	reasoning  string
	err        string
}

var client *ai.Client
var model string

func main() {
	// 0) 探活：引擎未起如实退出 3，绝不误报。
	mid, err := probeEngine()
	if err != nil {
		fmt.Println("SKIP: strata engine not reachable at", engineBase, "—", err)
		os.Exit(3)
	}
	model = mid
	if model == "" {
		model = fallbackModel
	}
	fmt.Printf("[ok] strata engine reachable, model=%s\n", model)

	// 1) gaea 客户端装配：内存态引擎表指向真实引擎（空 statePath=不落盘）。
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "strata", BaseURL: engineBase, Enabled: true}); err != nil {
		fmt.Println("FATAL: SaveEngine:", err)
		os.Exit(2)
	}
	client = ai.NewClient(&config.Config{})
	client.SetEngineManager(mgr)

	// 2) 预热一枪（首请求含草稿表/SSD 预热等一次性成本，不计入实测）。
	warm := runOnce(runResult{name: "warmup"}, "你是验证脚本助手。", "只回复：好。",
		ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, MaxTokens: 64})
	if warm.err != "" {
		fmt.Println("FATAL: warmup:", warm.err)
		os.Exit(2)
	}
	fmt.Printf("[warmup] ok (ttft=%s gen=%s completion=%d)\n",
		warm.ttft.Round(time.Millisecond), warm.genElapsed.Round(time.Millisecond), warm.completion)

	// 3) 四档实测。
	shortQ := "用一句话解释什么是潮汐锁定。"
	longQ := "写一段约 400 字的短文，描写秋天清晨的城市街道，情景交融。"
	longCtx := strings.Repeat("城市热岛效应是指城市气温明显高于周围郊区的现象，其主要成因包括建筑材料吸热、"+
		"绿地减少、人为热排放以及通风条件变差。缓解手段包括增加植被覆盖、采用高反射率屋顶、"+
		"优化建筑布局和发展公共交通。", 40) + "\n\n请用一句话概括以上材料的核心主题。"

	runs := []runResult{
		runOnce(runResult{name: "短答·关思考"}, "你是验证脚本助手。", shortQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, TimeoutMinutes: 10}),
		runOnce(runResult{name: "长答·关思考"}, "你是验证脚本助手。", longQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, TimeoutMinutes: 10}),
		runOnce(runResult{name: "思考缺省开·短答"}, "你是验证脚本助手。", shortQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", TimeoutMinutes: 10}),
		runOnce(runResult{name: "长提示词·关思考"}, "你是验证脚本助手。", longCtx,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, MaxTokens: 512, TimeoutMinutes: 10}),
	}

	// 4) 结果与判定。
	fails := 0
	check := func(name string, ok bool, detail string) {
		if ok {
			fmt.Println("[PASS]", name)
		} else {
			fails++
			fmt.Println("[FAIL]", name, "—", detail)
		}
	}

	fmt.Println("\n=== 实测（gaea 客户端口径 tok/s，即聊天页显示值）===")
	for _, r := range runs {
		if r.err != "" {
			fmt.Printf("  %-14s ERR: %s\n", r.name, r.err)
			continue
		}
		fmt.Printf("  %-14s ttft=%-8s prompt=%-5d completion=%-4d gen=%-8s %6.1f tok/s  正文=%d字 思考=%d字\n",
			r.name, r.ttft.Round(time.Millisecond), r.promptTok, r.completion,
			r.genElapsed.Round(time.Millisecond), r.tps,
			len([]rune(r.content)), len([]rune(r.reasoning)))
	}
	for _, r := range runs {
		check(r.name+"·请求无错", r.err == "", r.err)
		check(r.name+"·usage 齐备（completion>0）", r.completion > 0,
			fmt.Sprintf("completion=%d", r.completion))
		check(r.name+"·输出速度 ≥10 tok/s", r.tps >= 10.0,
			fmt.Sprintf("%.1f tok/s", r.tps))
	}
	check("短答·首字延迟 <30s", runs[0].err == "" && runs[0].ttft < 30*time.Second,
		runs[0].ttft.Round(time.Millisecond).String())
	check("长提示词·首字（预填充）延迟 <2min", runs[3].err == "" && runs[3].ttft < 2*time.Minute,
		runs[3].ttft.Round(time.Millisecond).String())

	// 5) 引擎自身日志旁证（独立口径：引擎记账的 generated tok/s）。
	fmt.Println("\n=== 引擎日志旁证（strata serve，最近若干条）===")
	for _, line := range lastEngineSpeedLines(4) {
		fmt.Println(" ", line)
	}

	if fails > 0 {
		fmt.Printf("\nVERIFY_FAIL: %d 项未过\n", fails)
		os.Exit(1)
	}
	fmt.Println("\nVERIFY_OK: strata 输出速度四档实测通过")
}

// probeEngine GET /v1/models，返回首个模型 id；不可达报错。
func probeEngine() (string, error) {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(engineBase + "/models")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &out) // 宽容：解析失败走回退模型名
	if len(out.Data) > 0 {
		return out.Data[0].ID, nil
	}
	return "", nil
}

// runOnce 跑一档：流式消费直到流结束块，按聊天页口径计时计 tokens。
func runOnce(r runResult, system, user string, opts ai.ChatSimpleOptions) runResult {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	chunks, canc, err := client.ChatStreamChunks(ctx, model, system, user, opts)
	if err != nil {
		r.err = err.Error()
		return r
	}
	defer canc()

	var firstDelta time.Time
	var usage *ai.ChatUsage
	var content, reasoning strings.Builder
	for chunk := range chunks {
		if chunk.Error != "" {
			r.err = chunk.Error
			return r
		}
		if chunk.Done {
			usage = chunk.Usage
			break
		}
		if (chunk.Content != "" || chunk.Reasoning != "") && firstDelta.IsZero() {
			firstDelta = time.Now()
		}
		content.WriteString(chunk.Content)
		reasoning.WriteString(chunk.Reasoning)
	}
	r.content, r.reasoning = content.String(), reasoning.String()
	if firstDelta.IsZero() {
		r.err = "没有任何内容/思考 delta（无正文）"
		return r
	}
	r.ttft = firstDelta.Sub(started)
	r.genElapsed = time.Since(firstDelta)
	if usage != nil {
		r.promptTok, r.completion = usage.PromptTokens, usage.CompletionTokens
	}
	r.tps = streamTokensPerSecond(r.completion, r.genElapsed)
	return r
}

// streamTokensPerSecond 与 internal/app/chat_service.go 同口径（测试性复刻，
// 常量断言防漂移：7.666→7.7）。
func streamTokensPerSecond(tokens int64, elapsed time.Duration) float64 {
	if tokens <= 0 || elapsed <= 0 {
		return 0
	}
	return math.Round(float64(tokens)/elapsed.Seconds()*10) / 10
}

// lastEngineSpeedLines 读引擎日志尾部，取最近 n 条含「generated in」的记账行；
// 读失败（文件锁等）返回占位说明，不致命。
func lastEngineSpeedLines(n int) []string {
	raw, err := os.ReadFile(engineLog)
	if err != nil {
		return []string{"(日志不可读: " + err.Error() + ")"}
	}
	lines := strings.Split(string(raw), "\n")
	var picked []string
	for i := len(lines) - 1; i >= 0 && len(picked) < n; i-- {
		if strings.Contains(lines[i], "generated in") {
			picked = append([]string{strings.TrimSpace(lines[i])}, picked...)
		}
	}
	if len(picked) == 0 {
		return []string{"(日志中暂无 generated 记账行)"}
	}
	return picked
}
