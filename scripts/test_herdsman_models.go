//go:build ignore
// +build ignore

// test_herdsman_models runs a functional battery against the local Herdsman
// LLM models (http://localhost:8080/v1 by default; override with
// HERDSMAN_BASE_URL). It auto-detects the LLM models from /v1/models and runs
// 20 fixed tasks against each: 常识/公文/翻译/代码/逻辑/数学/长文摘要/JSON
// 抽取/多轮对话/代码调试/创意写作/中译英/格式遵循/长文输出/知识边界/思考
// 模式/流式输出/识图（表格/图表/流程图）. Image inputs follow the same
// OpenAI-compatible text+image_url format the app's vision module uses. Every
// task runs under thinking and non-thinking mode (HERDSMAN_MODE=
// think|nothink|both) with identical prompts/params so modes are comparable;
// replies are saved to .tmp-herdsman-<mode>-*.txt.
package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	baseURL = strings.TrimRight(getenv("HERDSMAN_BASE_URL", "http://localhost:8080/v1"), "/")
	client  = &http.Client{Timeout: 300 * time.Second}
)

type modelsResp struct {
	Data []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
		Status  string `json:"status"`
	} `json:"data"`
}

type chatResp struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
			Reasoning        string `json:"reasoning,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Timings *struct {
		PredictedPerSecond float64 `json:"predicted_per_second"`
	} `json:"timings"`
}

type task struct {
	ID        string
	Label     string
	Prompt    string
	Turns     []string // multi-turn user prompts; empty means single turn
	Image     string   // relative image path for vision tasks (repo root)
	MaxTokens int
	Stream    bool
	Thinking  bool
	// Check returns "" on success, or a short failure reason.
	Check func(content string) string
}

func defaultCheck(content string) string {
	if strings.TrimSpace(content) == "" {
		return "空回复"
	}
	return ""
}

func containsAll(ss ...string) func(string) string {
	return func(content string) string {
		for _, s := range ss {
			if !strings.Contains(content, s) {
				return fmt.Sprintf("缺少关键内容 %q", s)
			}
		}
		return ""
	}
}

func lengthBetween(min, max int) func(string) string {
	return func(content string) string {
		n := len([]rune(strings.TrimSpace(content)))
		if n < min {
			return fmt.Sprintf("字数 %d < %d", n, min)
		}
		if max > 0 && n > max {
			return fmt.Sprintf("字数 %d > %d", n, max)
		}
		return ""
	}
}

func containsCJK(content string) string {
	for _, r := range content {
		if unicode.Is(unicode.Han, r) {
			return ""
		}
	}
	return "无中文字符"
}

var alphaWord = regexp.MustCompile(`[A-Za-z]{2,}`)

func hasEnglish(content string) string {
	if len(alphaWord.FindAllString(content, -1)) >= 8 {
		return ""
	}
	return "英文单词过少"
}

func hasDigit(content string) string {
	for _, r := range content {
		if r >= '0' && r <= '9' {
			return ""
		}
	}
	return "无数字"
}

func hasAny(keywords ...string) func(string) string {
	return func(content string) string {
		for _, k := range keywords {
			if strings.Contains(content, k) {
				return ""
			}
		}
		return fmt.Sprintf("未命中关键词 %v", keywords)
	}
}

func validJSON(content string) string {
	s := extractJSON(content)
	if !json.Valid([]byte(s)) {
		return "非合法 JSON"
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "JSON 解析失败"
	}
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			return "JSON 为空对象"
		}
	case []any:
		if len(t) == 0 {
			return "JSON 为空数组"
		}
	default:
		return "JSON 不是对象或数组"
	}
	return ""
}

// extractJSON pulls the JSON payload out of a model reply that may contain
// markdown fences and/or a leading explanation sentence.
func extractJSON(content string) string {
	s := strings.TrimSpace(content)
	if i := strings.Index(s, "```"); i >= 0 {
		if j := strings.LastIndex(s, "```"); j > i {
			inner := strings.TrimSpace(s[i+3 : j])
			inner = strings.TrimPrefix(inner, "json")
			inner = strings.TrimPrefix(inner, "JSON")
			s = strings.TrimSpace(inner)
		}
	}
	start := -1
	for i, r := range s {
		if r == '{' || r == '[' {
			start = i
			break
		}
	}
	if start < 0 {
		return s
	}
	s = s[start:]
	end := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '}' || s[i] == ']' {
			end = i
			break
		}
	}
	if end >= 0 {
		s = s[:end+1]
	}
	return strings.TrimSpace(s)
}

// fencedCode returns the content of the first markdown code fence block
// (```lang ... ```), or "" when none is present.
func fencedCode(content string) string {
	i := strings.Index(content, "```")
	if i < 0 {
		return ""
	}
	rest := content[i+3:]
	// skip optional language tag on the opening line
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[nl+1:]
	} else {
		return ""
	}
	j := strings.Index(rest, "```")
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

func numberedList(content string) string {
	lines := nonEmptyLines(content)
	if len(lines) < 3 {
		return fmt.Sprintf("行数 %d < 3", len(lines))
	}
	ok := 0
	for _, ln := range lines {
		if regexp.MustCompile(`^\d+[.、)]`).MatchString(ln) ||
			strings.HasPrefix(ln, "- ") || strings.HasPrefix(ln, "* ") {
			ok++
		}
	}
	if ok < 3 {
		return "列表格式不符合要求"
	}
	return ""
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			out = append(out, strings.TrimSpace(ln))
		}
	}
	return out
}

var longPassage = `近年来，大语言模型的参数规模与训练数据量持续增长，推动了自然语言处理能力的跨越式进步。
从早期的统计语言模型，到基于 Transformer 架构的预训练模型，再到具备多轮对话、工具调用与推理能力的
新一代模型，技术演进的核心始终围绕“让机器更好地理解并生成人类语言”。与此同时，模型的落地方式也出现
了明显分化：云端大模型凭借海量算力与集中式训练，在通用能力和复杂推理上占据优势，但存在数据外传、
按量计费与网络延迟等顾虑；本地大模型则将模型部署在用户自己的设备上，数据不出内网，响应更快，且一次
部署后边际成本极低，尤其适合处理敏感文档、离线环境与高频小任务。不过本地模型也受限于显存与算力，
在模型规模、知识更新速度和长文本处理能力上往往弱于云端头部模型。因此，越来越多企业开始采用“混合
路由”的策略：简单、私密的任务交给本地模型，复杂推理与创意任务交给云端模型，从而在成本、隐私与质量
之间取得平衡。`

var tasks = []task{
	{"T1", "常识", "请用 200 字以内解释量子计算的原理，以及它与经典计算的根本区别。", nil, "", 800, false, false, lengthBetween(50, 400)},
	{"T2", "公文", "请写一份 150 字左右的工作周报摘要，包含本周完成事项、遇到的问题与下周计划，使用正式的公文语气。", nil, "", 800, false, false, lengthBetween(80, 500)},
	{"T3", "翻译", "把下面这句英文翻译成地道的中文：The quick brown fox jumps over the lazy dog, but the dog was not lazy at all — it simply preferred to stay.", nil, "", 800, false, false, containsCJK},
	{"T4", "代码", "用 Python 写一个函数 find_most_frequent(nums)，返回列表中出现次数最多的元素；若有多个并列，返回全部。附 2 个示例。", nil, "", 800, false, false, containsAll("def find_most_frequent", "def ")},
	{"T5", "逻辑", "三个盒子分别标着“苹果”“橘子”“苹果和橘子”，但所有标签都是错的。你只能从一个盒子里摸一个水果，如何确定三个盒子的真实内容？请解释推理过程。", nil, "", 800, false, false, containsAll("苹果", "橘子", "标签")},
	{"T6", "数学", "一个蓄水池有进水管和排水管。单独开进水管 6 小时可注满，单独开排水管 9 小时可排空。如果先只开进水管 2 小时，然后进水管和排水管同时打开，还需要多久注满？请给出计算过程。", nil, "", 800, false, false, func(c string) string {
		if r := hasDigit(c); r != "" {
			return r
		}
		if !strings.Contains(c, "小时") {
			return "缺少“小时”结论"
		}
		return ""
	}},
	{"T7", "长文摘要", "请用 100 字以内概括下面这段文字的核心观点：\n\n" + longPassage, nil, "", 600, false, false, func(c string) string {
		if r := lengthBetween(20, 200)(c); r != "" {
			return r
		}
		return hasAny("大模型", "人工智能", "模型", "云端", "本地")(c)
	}},
	{"T8", "JSON抽取", "从以下报销记录中提取结构化数据，输出一个 JSON 对象，字段为 name、date、amount、department，不要输出 JSON 以外的文字：\n张三于2026年8月1日报销差旅费1200元，所属部门为市场部；李四于2026年8月3日报销办公用品320.5元，所属部门为行政部。", nil, "", 600, false, false, func(c string) string {
		if r := validJSON(c); r != "" {
			return r
		}
		return hasAny("name", "张三", "李四")(c)
	}},
	{"T9", "多轮对话", "", []string{
		"我下周要在公司内部做一次 30 分钟的汇报，主题是“本地大模型在企业内的落地”。请先给我列出汇报大纲。",
		"很好。请基于你刚才的大纲，帮我写一段开场白，并补充一个可行的落地案例。",
	}, "", 800, false, false, hasAny("开场白", "大家好", "汇报", "本地", "案例")},
	{"T10", "代码调试", "下面的 Python 代码想统计字符串中每个字符出现的次数，但结果不对。请找出 bug 并给出修复后的完整代码：\n```python\ndef char_count(s):\n    counts = {}\n    for c in s:\n        if c in counts:\n            counts[c] = 1\n        else:\n            counts[c] += 1\n    return counts\n```", nil, "", 800, false, false, func(c string) string {
		if hasAny("get(", "defaultdict", "Counter", "not in counts")(c) == "" {
			return ""
		}
		// 另一种正确修法是颠倒 if/else 分支。只在代码块内判断顺序，
		// 避免分析文字先提到 buggy 行造成误判。
		code := c
		if fenced := fencedCode(c); fenced != "" {
			code = fenced
		}
		i1 := strings.Index(code, "counts[c] += 1")
		i2 := strings.Index(code, "counts[c] = 1")
		if i1 >= 0 && i2 >= 0 && i1 < i2 {
			return ""
		}
		return "未识别出正确修复（期望 get()/Counter 或颠倒 if/else 分支）"
	}},
	{"T11", "创意写作", "请写一段 300 字左右的科幻微小说开头，必须包含“雨”和“老式电话亭”两个元素。", nil, "", 1000, false, false, func(c string) string {
		if r := lengthBetween(200, 0)(c); r != "" {
			return r
		}
		return containsAll("雨", "电话亭")(c)
	}},
	{"T12", "中译英", "把下面这句中文翻译成自然流畅的英文：好的工具不会替你做决定，而是让你的每一个决定都更清晰。然后给出另一种更口语化的译法。", nil, "", 800, false, false, hasEnglish},
	{"T13", "格式遵循", "请输出一个三行的待办清单，每行格式为：编号. 内容。不要输出其他任何文字。", nil, "", 400, false, false, numberedList},
	{"T14", "长文输出", "请写一篇 800 字以上的短文，主题：“本地大模型与云端大模型的优缺点对比”，要求有开头、三个分论点、结尾。", nil, "", 1800, false, false, lengthBetween(600, 0)},
	{"T15", "知识边界", "2026 年发布的最新 GPT 模型是什么？如果无法确认，请直接说明你不确定。", nil, "", 400, false, false, defaultCheck},
	{"T16", "思考模式", "一个房间里有 100 盏灯，编号 1 到 100，初始全关。第 k 个人把所有编号能被 k 整除的灯各按一次开关（开变关、关变开）。100 个人依次操作后，哪些灯是亮的？请说明理由。", nil, "", 2000, false, true, defaultCheck},
	{"T17", "流式输出", "请简要解释什么是 RAG（检索增强生成），300 字以内。", nil, "", 600, true, false, defaultCheck},
	{"T18", "识图-表格", "请逐列描述这张工程成本测算表：读出表头、费用项目名称和关键数字。", nil, ".gaea/exports/预览-成本测算表.png", 1024, false, false, func(c string) string {
		if r := hasDigit(c); r != "" {
			return r
		}
		if r := hasAny("成本测算", "测算表", "市政道路")(c); r != "" {
			return r
		}
		return hasAny("沥青", "混凝土", "税金", "路缘石")(c)
	}},
	{"T19", "识图-图表", "这张图包含哪些内容？请说明图表类型（饼图/柱状图）以及费用汇总表中各项金额。", nil, ".gaea/exports/预览-费用汇总与柱状图.png", 1024, false, false, func(c string) string {
		if r := hasDigit(c); r != "" {
			return r
		}
		return hasAny("费用汇总", "直接费", "柱状", "饼图", "占比")(c)
	}},
	{"T20", "识图-流程", "请描述这张流程图的结构，并尽量读出图中每个方框里的文字。", nil, ".tmp-diagram.png", 1024, false, false, hasAny("流程", "gaea", "SQLite")},
}

func main() {
	models, err := listModels()
	if err != nil {
		fmt.Printf("FATAL: 无法获取模型列表: %v\n", err)
		os.Exit(1)
	}
	llms := filterLLMs(models)
	if len(llms) == 0 {
		fmt.Println("FATAL: 服务没有发现 LLM 模型")
		os.Exit(1)
	}
	if f := os.Getenv("HERDSMAN_MODEL"); f != "" {
		var keep []string
		for _, m := range llms {
			if strings.Contains(m, f) {
				keep = append(keep, m)
			}
		}
		llms = keep
		if len(llms) == 0 {
			fmt.Println("FATAL: HERDSMAN_MODEL 过滤后没有模型")
			os.Exit(1)
		}
	}
	taskSet := tasks
	if f := os.Getenv("HERDSMAN_TASKS"); f != "" {
		want := map[string]bool{}
		for _, id := range strings.Split(f, ",") {
			want[strings.TrimSpace(id)] = true
		}
		var keep []task
		for _, t := range tasks {
			if want[t.ID] {
				keep = append(keep, t)
			}
		}
		taskSet = keep
		if len(taskSet) == 0 {
			fmt.Println("FATAL: HERDSMAN_TASKS 过滤后没有任务")
			os.Exit(1)
		}
	}

	fmt.Printf("本地模型测试（OpenAI 兼容 %s）—— %d 个任务 × %d 个模型\n", baseURL, len(taskSet), len(llms))
	fmt.Printf("模型总数: %d，LLM 模型: %d\n\n", len(models), len(llms))
	for i, m := range llms {
		fmt.Printf("  %d. %s\n", i+1, m)
	}
	fmt.Println()

	modeFilter := strings.ToLower(getenv("HERDSMAN_MODE", "both"))
	modes := []string{"nothink"}
	switch modeFilter {
	case "think":
		modes = []string{"think"}
	case "nothink":
		modes = []string{"nothink"}
	default:
		modes = []string{"nothink", "think"}
	}
	fmt.Printf("测试模式: %s（%d 个任务 × %d 个模型 × %d 种模式，参数一致）\n\n",
		strings.Join(modes, " + "), len(taskSet), len(llms), len(modes))

	type modeAgg struct {
		pass, fail int
		elapsed    float64
		chars      float64
		reasoned   int
	}
	totalPass, totalFail := 0, 0
	compare := make(map[string]map[string]*modeAgg) // modelShort -> mode -> agg
	for _, md := range modes {
		for _, m := range llms {
			short := shortName(m)
			fmt.Printf("===== %s [%s] =====\n", m, md)
			a := &modeAgg{}
			for _, t := range taskSet {
				ok, elapsed, chars, tps, detail := runOne(md, m, t)
				if ok {
					a.pass++
					totalPass++
				} else {
					a.fail++
					totalFail++
				}
				a.elapsed += elapsed.Seconds()
				a.chars += float64(chars)
				if strings.Contains(detail, "推理") {
					a.reasoned++
				}
				status := "PASS"
				if !ok {
					status = "FAIL"
				}
				tpsTxt := "-"
				if tps > 0 {
					tpsTxt = fmt.Sprintf("%.1f", tps)
				}
				fmt.Printf("  %s %-6s %-4s %9s %6d 字 %7s tok/s%s\n",
					t.ID, t.Label, status, elapsed.Round(time.Millisecond), chars, tpsTxt, detail)
			}
			fmt.Printf("  -- %s [%s]: %d/%d 通过，总耗时 %.0fs，平均 %.0f 字/任务\n\n",
				short, md, a.pass, a.pass+a.fail, a.elapsed, a.chars/float64(len(taskSet)))
			if compare[short] == nil {
				compare[short] = map[string]*modeAgg{}
			}
			compare[short][md] = a
		}
	}

	fmt.Printf("RESULT: %d pass, %d fail\n\n", totalPass, totalFail)
	fmt.Println("对比（同一套任务与参数，仅 thinking_enabled 不同）：")
	fmt.Printf("%-18s %-9s %-7s %-10s %-10s %-10s\n", "模型", "模式", "通过", "平均耗时", "平均字数", "推理任务")
	for _, m := range llms {
		short := shortName(m)
		for _, md := range modes {
			a := compare[short][md]
			n := len(taskSet)
			fmt.Printf("%-18s %-9s %d/%-4d %-10s %-10.0f %d/%d\n",
				short, md, a.pass, n,
				fmt.Sprintf("%.1fs", a.elapsed/float64(n)),
				a.chars/float64(n), a.reasoned, n)
		}
	}
}

func runOne(mode, model string, t task) (ok bool, elapsed time.Duration, chars int, tps float64, detail string) {
	thinking := mode == "think"
	start := time.Now()
	turns := t.Turns
	if len(turns) == 0 {
		turns = []string{t.Prompt}
	}
	var content, reasoning string
	var firstToken time.Duration
	messages := make([]map[string]string, 0, len(turns)*2)
	for i, prompt := range turns {
		messages = append(messages, map[string]string{"role": "user", "content": prompt})
		var err error
		var ft time.Duration
		content, reasoning, ft, tps, err = callModel(model, t, messages, start, thinking)
		if err != nil {
			return false, time.Since(start), 0, 0, fmt.Sprintf(" (%v)", err)
		}
		if i == 0 {
			firstToken = ft
		}
		if i < len(turns)-1 {
			messages = append(messages, map[string]string{"role": "assistant", "content": content})
		}
	}
	elapsed = time.Since(start)

	content = strings.TrimSpace(content)
	reasoning = strings.TrimSpace(reasoning)
	reasoningChars := len([]rune(reasoning))
	chars = len([]rune(content))
	if reasoningChars > 0 {
		detail = fmt.Sprintf(" (推理%d字)", reasoningChars)
	}
	if t.Stream {
		if chars == 0 && reasoningChars == 0 {
			return false, elapsed, 0, 0, " (流式空回复)"
		}
		if t.Check != nil {
			if r := t.Check(content); r != "" {
				return false, elapsed, chars, 0, fmt.Sprintf(" (%s)%s", r, detail)
			}
		}
		if chars > 0 {
			saveReply(mode, model, t.ID, t.Label, content)
		}
		return true, elapsed, chars, 0, fmt.Sprintf(" (首token %.0fms)%s", float64(firstToken.Microseconds())/1000, detail)
	}

	checkErr := ""
	if t.Check != nil {
		checkErr = t.Check(content)
	}
	// A thinking model may spend its whole token budget on reasoning; that still
	// proves the capability even when the final content is empty.
	if chars == 0 && thinking && reasoningChars > 0 {
		return true, elapsed, 0, tps, " (仅有推理，无最终回答)" + detail
	}
	if chars == 0 {
		return false, elapsed, 0, 0, " (空回复)" + detail
	}
	if checkErr != "" {
		saveReply(mode, model, t.ID, t.Label, content) // keep failed replies for inspection
		return false, elapsed, chars, tps, fmt.Sprintf(" (%s)%s", checkErr, detail)
	}
	saveReply(mode, model, t.ID, t.Label, content)
	return true, elapsed, chars, tps, detail
}

// callModel performs one chat request. For streaming tasks it consumes SSE and
// returns the accumulated text; thinking adds thinking_enabled so the response
// may carry reasoning_content. start is the run's wall-clock start used to
// compute first-token latency.
func callModel(model string, t task, messages []map[string]string, start time.Time, thinking bool) (content string, reasoning string, firstToken time.Duration, tps float64, err error) {
	var msgs any = messages
	if t.Image != "" {
		dataURL, derr := imageDataURL(filepath.Join(repoRoot(), t.Image))
		if derr != nil {
			return "", "", 0, 0, fmt.Errorf("读取图片: %v", derr)
		}
		msgs = []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": t.Prompt},
					{"type": "image_url", "image_url": map[string]string{"url": dataURL}},
				},
			},
		}
	}
	payload := map[string]any{
		"model":       model,
		"messages":    msgs,
		"temperature": 0.3,
	}
	// Identical token budget in both modes so only thinking_enabled differs;
	// 2048 leaves room for reasoning on top of the final answer.
	maxTokens := t.MaxTokens
	if maxTokens < 2048 {
		maxTokens = 2048
	}
	payload["max_tokens"] = maxTokens
	if t.Stream {
		payload["stream"] = true
	}
	if thinking {
		payload["thinking_enabled"] = true
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("marshal: %v", err)
	}

	resp, err := client.Post(baseURL+"/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return "", "", 0, 0, fmt.Errorf("HTTP %d: %.160s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	if t.Stream {
		content, reasoning, ft, err := consumeSSE(resp.Body, start)
		return content, reasoning, ft, 0, err
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("读取响应: %v", err)
	}
	var out chatResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", 0, 0, fmt.Errorf("解析响应: %v", err)
	}
	if len(out.Choices) == 0 {
		return "", "", 0, 0, fmt.Errorf("无 choices")
	}
	if out.Timings != nil {
		tps = out.Timings.PredictedPerSecond
	}
	rc := out.Choices[0].Message.ReasoningContent
	if rc == "" {
		rc = out.Choices[0].Message.Reasoning
	}
	return out.Choices[0].Message.Content, rc, 0, tps, nil
}

func consumeSSE(r io.Reader, start time.Time) (string, string, time.Duration, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	var parts []string
	var reasonParts []string
	sawData := false
	firstToken := time.Duration(0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		if !sawData {
			firstToken = time.Since(start)
		}
		sawData = true
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			d := chunk.Choices[0].Delta
			if d.Content != "" {
				parts = append(parts, d.Content)
			}
			if d.ReasoningContent != "" {
				reasonParts = append(reasonParts, d.ReasoningContent)
			} else if d.Reasoning != "" {
				reasonParts = append(reasonParts, d.Reasoning)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return strings.Join(parts, ""), strings.Join(reasonParts, ""), firstToken, fmt.Errorf("流式读取: %v", err)
	}
	if !sawData {
		return "", strings.Join(reasonParts, ""), firstToken, fmt.Errorf("非 SSE 响应")
	}
	return strings.Join(parts, ""), strings.Join(reasonParts, ""), firstToken, nil
}

func saveReply(mode, model, id, label, content string) {
	name := fmt.Sprintf(".tmp-herdsman-%s-%s-%s_%s.txt", mode, shortName(model), id, label)
	_ = os.WriteFile(name, []byte(content), 0o644)
}

func listModels() ([]string, error) {
	resp, err := client.Get(baseURL + "/models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out modelsResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(out.Data))
	for _, d := range out.Data {
		ids = append(ids, d.ID)
	}
	return ids, nil
}

// filterLLMs keeps chat-capable LLM models, dropping embedding/rerank/TTS/STT/
// document/image services (bge-m3, edge-tts, sherpa-*, mineru, zimage-*, ...).
func filterLLMs(models []string) []string {
	var out []string
	for _, id := range models {
		l := strings.ToLower(id)
		skip := false
		for _, kw := range []string{"bge", "rerank", "tts", "voice", "edge", "whisper",
			"sherpa", "zipformer", "asr", "mineru", "ocr", "image", "zimage",
			"flux", "krea", "sd", "dalle"} {
			if strings.Contains(l, kw) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, id)
		}
	}
	return out
}

func shortName(id string) string {
	s := id
	for _, prefix := range []string{"Hermes3.6-35B-A3B-", "Qwen3.6-35B-A3B-"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			break
		}
	}
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "@", "")
	s = strings.ReplaceAll(s, " ", "")
	if len(s) > 16 {
		s = s[:16]
	}
	return s
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func repoRoot() string {
	if d := os.Getenv("HERDSMAN_IMG_DIR"); d != "" {
		return d
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

func imageDataURL(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	case ".bmp":
		mime = "image/bmp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}
