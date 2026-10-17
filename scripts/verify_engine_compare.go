//go:build ignore
// +build ignore

package main

// verify_engine_compare.go — 双引擎同模型真机对擂（MoE4All vs Strata，同一份
// Qwen3.8-Flash-Next GSQ-RCO IQ2_XS GGUF），测量口径与 scripts/verify_strata_speed.go
// 完全一致（gaea 自家 ai 客户端 + 聊天页 tok/s 口径），新增并发 x2 档。
// 用法：
//   go run scripts/verify_engine_compare.go -label MoE4All -base http://127.0.0.1:8092/v1 \
//     -model Qwen3.8-Flash-Next-GSQ-RCO-IQ2_XS-00001-of-00002 \
//     -enginelog C:\...\infr.log -logpattern decode_tps -out temp\cmp_moe4all.json
// 引擎未起：如实退出 3。零污染：内存态引擎表，不落盘不写用户数据。
import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

var (
	flagBase = flag.String("base", "http://127.0.0.1:8091/v1", "引擎 OpenAI /v1 根地址")
	flagModel = flag.String("model", "", "模型 id（空=取 /v1/models 首个）")
	flagLabel = flag.String("label", "engine", "展示标签")
	flagEngineLog = flag.String("enginelog", "", "引擎日志路径（尾部旁证）")
	flagLogPattern = flag.String("logpattern", "generated in", "日志速度行匹配串")
	flagOut = flag.String("out", "", "结果 JSON 落盘路径")
	flagSkipConc = flag.Bool("skip-conc", false, "跳过并发 x2 档")
)

type runResult struct {
	Name       string  `json:"name"`
	TTFTms     int64   `json:"ttft_ms"`
	GenMs      int64   `json:"gen_ms"`
	PromptTok  int64   `json:"prompt_tokens"`
	Completion int64   `json:"completion_tokens"`
	TPS        float64 `json:"tok_per_s"`
	ContentLen int     `json:"content_chars"`
	ReasonLen  int     `json:"reasoning_chars"`
	Err        string  `json:"err,omitempty"`
}

type concResult struct {
	Streams []runResult `json:"streams"`
	WallMs  int64       `json:"wall_ms"`
	AggTPS  float64     `json:"agg_tok_per_s"`
}

type report struct {
	Label     string      `json:"label"`
	Base      string      `json:"base"`
	Model     string      `json:"model"`
	Warmup    runResult   `json:"warmup"`
	Runs      []runResult `json:"runs"`
	Conc2     *concResult `json:"conc2,omitempty"`
	LogLines  []string    `json:"engine_log_lines"`
	StartedAt string      `json:"started_at"`
}

var client *ai.Client
var model string

func main() {
	flag.Parse()
	started := time.Now()

	// 0) 探活。
	mid, err := probeEngine()
	if err != nil {
		fmt.Println("SKIP: engine not reachable at", *flagBase, "—", err)
		os.Exit(3)
	}
	model = *flagModel
	if model == "" {
		model = mid
	}
	fmt.Printf("[ok] %s reachable (%s), model=%s\n", *flagLabel, *flagBase, model)

	// 1) gaea 客户端装配：引擎 ID 固定 "strata"（复用思考通道 chat_template_kwargs——
	//    strata 必需、MoE4All 已实测接受，两侧同一客户端路径保证口径一致）。
	mgr := modelengine.NewManager("", "")
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "strata", BaseURL: *flagBase, Enabled: true}); err != nil {
		fmt.Println("FATAL: SaveEngine:", err)
		os.Exit(2)
	}
	client = ai.NewClient(&config.Config{})
	client.SetEngineManager(mgr)

	rep := report{Label: *flagLabel, Base: *flagBase, Model: model, StartedAt: started.Format(time.RFC3339)}

	// 2) 预热（页缓存/管线首编译一次性成本，不计入实测）。
	rep.Warmup = runOnce("warmup", "你是验证脚本助手。", "只回复：好。",
		ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, MaxTokens: 64})
	if rep.Warmup.Err != "" {
		fmt.Println("FATAL: warmup:", rep.Warmup.Err)
		exitWithReport(&rep, 2)
	}
	fmt.Printf("[warmup] ok (ttft=%dms completion=%d)\n", rep.Warmup.TTFTms, rep.Warmup.Completion)

	// 3) 四档实测（提示词与 verify_strata_speed.go 逐字节一致）。
	shortQ := "用一句话解释什么是潮汐锁定。"
	longQ := "写一段约 400 字的短文，描写秋天清晨的城市街道，情景交融。"
	longCtx := strings.Repeat("城市热岛效应是指城市气温明显高于周围郊区的现象，其主要成因包括建筑材料吸热、"+
		"绿地减少、人为热排放以及通风条件变差。缓解手段包括增加植被覆盖、采用高反射率屋顶、"+
		"优化建筑布局和发展公共交通。", 40) + "\n\n请用一句话概括以上材料的核心主题。"

	type profile struct {
		name string
		sys  string
		usr  string
		opts ai.ChatSimpleOptions
	}
	profiles := []profile{
		{"短答·关思考", "你是验证脚本助手。", shortQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, TimeoutMinutes: 10}},
		{"长答·关思考", "你是验证脚本助手。", longQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, TimeoutMinutes: 10}},
		{"思考缺省开·短答", "你是验证脚本助手。", shortQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", TimeoutMinutes: 10}},
		{"长提示词·关思考", "你是验证脚本助手。", longCtx,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, MaxTokens: 512, TimeoutMinutes: 10}},
		// 复测档：短答再跑一遍，观察页缓存变热后的稳定性。
		{"短答·关思考·复测", "你是验证脚本助手。", shortQ,
			ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true, TimeoutMinutes: 10}},
	}
	for _, p := range profiles {
		r := runOnce(p.name, p.sys, p.usr, p.opts)
		rep.Runs = append(rep.Runs, r)
		if r.Err != "" {
			fmt.Printf("  %-16s ERR: %s\n", r.Name, r.Err)
			continue
		}
		fmt.Printf("  %-16s ttft=%-6dms prompt=%-5d completion=%-4d gen=%-6dms %6.1f tok/s  正文=%d字 思考=%d字\n",
			r.Name, r.TTFTms, r.PromptTok, r.Completion, r.GenMs, r.TPS, r.ContentLen, r.ReasonLen)
	}

	// 4) 并发 x2（两路同型短问同时发）。
	if !*flagSkipConc {
		rep.Conc2 = runConc2()
		if rep.Conc2 != nil {
			fmt.Printf("  并发x2           wall=%dms agg=%0.1f tok/s（两路各自 %.1f / %.1f）\n",
				rep.Conc2.WallMs, rep.Conc2.AggTPS,
				rep.Conc2.Streams[0].TPS, rep.Conc2.Streams[1].TPS)
		}
	}

	// 5) 引擎日志旁证。
	rep.LogLines = lastLogLines(4)
	fmt.Println("\n=== 引擎日志旁证 ===")
	for _, l := range rep.LogLines {
		fmt.Println(" ", l)
	}

	if *flagOut != "" {
		if raw, err := json.MarshalIndent(rep, "", " "); err == nil {
			_ = os.WriteFile(*flagOut, raw, 0644)
			fmt.Println("\nresult ->", *flagOut)
		}
	}
	bad := 0
	for _, r := range rep.Runs {
		if r.Err != "" || r.Completion <= 0 {
			bad++
		}
	}
	if bad > 0 {
		exitWithReport(&rep, 1)
	}
	fmt.Printf("\nCOMPARE_OK: %s 采集完成\n", *flagLabel)
}

func exitWithReport(rep *report, code int) {
	if *flagOut != "" {
		if raw, err := json.MarshalIndent(rep, "", " "); err == nil {
			_ = os.WriteFile(*flagOut, raw, 0644)
		}
	}
	os.Exit(code)
}

// runConc2 两路并发：独立 goroutine 各自流式，汇总墙钟聚合吞吐。
func runConc2() *concResult {
	var wg sync.WaitGroup
	out := make([]runResult, 2)
	t0 := time.Now()
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("并发#%d", i)
			out[i] = runOnce(name, "你是验证脚本助手。",
				fmt.Sprintf("从 1 数到 %d，用顿号分隔，不要其他内容。", 25+i*5),
				ai.ChatSimpleOptions{EngineID: "strata", Feature: "verify", DisableThinking: true,
					MaxTokens: 256, TimeoutMinutes: 10})
		}(i)
	}
	wg.Wait()
	wall := time.Since(t0)
	agg := float64(out[0].Completion+out[1].Completion) / wall.Seconds()
	return &concResult{Streams: out, WallMs: wall.Milliseconds(), AggTPS: round1(agg)}
}

// runOnce 单档：流式消费，口径同聊天页（completion ÷ 首 delta→流结束）。
func runOnce(name, system, user string, opts ai.ChatSimpleOptions) runResult {
	r := runResult{Name: name}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	chunks, canc, err := client.ChatStreamChunks(ctx, model, system, user, opts)
	if err != nil {
		r.Err = err.Error()
		return r
	}
	defer canc()

	var firstDelta time.Time
	var usage *ai.ChatUsage
	var content, reasoning strings.Builder
	for chunk := range chunks {
		if chunk.Error != "" {
			r.Err = chunk.Error
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
	r.ContentLen = len([]rune(content.String()))
	r.ReasonLen = len([]rune(reasoning.String()))
	if firstDelta.IsZero() {
		r.Err = "没有任何内容/思考 delta（无正文）"
		return r
	}
	r.TTFTms = firstDelta.Sub(started).Milliseconds()
	r.GenMs = time.Since(firstDelta).Milliseconds()
	if usage != nil {
		r.PromptTok, r.Completion = usage.PromptTokens, usage.CompletionTokens
	}
	r.TPS = round1(float64(r.Completion) / (float64(r.GenMs) / 1000))
	return r
}

func round1(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0
	}
	return math.Round(v*10) / 10
}

func probeEngine() (string, error) {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(*flagBase + "/models")
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
	_ = json.Unmarshal(raw, &out)
	if len(out.Data) > 0 {
		return out.Data[0].ID, nil
	}
	return "", nil
}

func lastLogLines(n int) []string {
	if *flagEngineLog == "" {
		return []string{"(未指定引擎日志)"}
	}
	raw, err := os.ReadFile(*flagEngineLog)
	if err != nil {
		return []string{"(日志不可读: " + err.Error() + ")"}
	}
	lines := strings.Split(string(raw), "\n")
	var picked []string
	for i := len(lines) - 1; i >= 0 && len(picked) < n; i-- {
		if strings.Contains(lines[i], *flagLogPattern) {
			picked = append([]string{strings.TrimSpace(lines[i])}, picked...)
		}
	}
	if len(picked) == 0 {
		return []string{"(日志中暂无匹配行: " + *flagLogPattern + ")"}
	}
	return picked
}
