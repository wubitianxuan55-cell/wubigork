package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/whisper"
	wdb "github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 众神殿 action-1 实弹基线探针（进度计划/gaea-chat-persona-optimization-20261105.md
// §1.5）：五桶探针 + 15 轮连续闲聊 + 重启差分 + 并发首轮，全字段记录。
//
// 环境门控：GAEA_PROBE=1 才运行（CI 与常规测试面跳过）。引擎=本地 llama 自管档
//（GAEA_PROBE_BASE_URL / GAEA_PROBE_MODEL，缺省 8124/qwen35——与 herdsman 槽位
// 同一份 Qwen3.6-35B GGUF，同模型对照成立）；数据根走测试 TempDir 隔离，绝不
// 触碰生产会话；搜索腿用固定罐头结果（离线确定性，baseline/HEAD 两轮不变量）。
// 产出：GAEA_PROBE_OUT 目录下 report.json + summary.md。

const probePersona = "deredere" // 非 gaea 内置人格（压制面/生日面对象，论证钉住的「同一角色」）

type probeTurn struct {
	Group           string  `json:"group"`
	Msg             string  `json:"msg"`
	FinalType       string  `json:"finalType"`
	Error           string  `json:"error,omitempty"`
	Reply           string  `json:"reply"`
	ReplyChars      int     `json:"replyChars"`
	EventType       string  `json:"eventType"`
	GateLevel       string  `json:"gateLevel"`
	StrategyLevel   string  `json:"strategyLevel"`
	ClampInjected   bool    `json:"clampInjected"`
	SilentCalc      bool    `json:"silentCalc"`
	Stage           string  `json:"stage"`
	Aff             float64 `json:"aff"`
	Trust           float64 `json:"trust"`
	Label           string  `json:"label"`
	FactsInjected   int     `json:"factsInjected"`
	SearchPredicate bool    `json:"searchPredicate"`
	TotalMs         int64   `json:"totalMs"`
}

func (p *probeTurn) suppressionHit() bool {
	return p.ClampInjected || p.GateLevel == "silent" || p.GateLevel == "whisper" ||
		p.StrategyLevel == "silent" || p.StrategyLevel == "whisper"
}

// newProbeApp 与 newChatServiceTestAppWithHandler 同套装配纪律，但 herdsman 指向
// 真实本地引擎（探针专用；SSE 桩版见 chat_service_test.go）。
func newChatProbeApp(t *testing.T, baseURL, model string) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	// OCR 引擎面隔离（防本机真拉 llama-server，同 chat_service_test 纪律）
	t.Setenv("GAEA_UOCR2_DIR", filepath.Join(t.TempDir(), "no-such-uocr2"))
	t.Setenv("GAEA_UOCR2_PORT", "1")
	t.Setenv("GAEA_UOCR2_LLAMA_DIR", filepath.Join(t.TempDir(), "no-such-llama"))
	t.Setenv("GAEA_UOCR2_LLAMA_URL", "http://127.0.0.1:1")
	t.Setenv("GAEA_OCR_DIR", filepath.Join(t.TempDir(), "no-such-ocr"))
	t.Setenv("GAEA_OCR_PORT", "1")
	root := filepath.Join(t.TempDir(), "data")

	c := &core{
		cfg:       &config.Config{Model: "probe", XaiAPIBaseURL: "http://127.0.0.1:1"},
		engineMgr: modelengine.NewManager("", ""),
		chatStore: chat.NewStore(filepath.Join(root, "chat")),
	}
	t.Cleanup(func() { _ = c.chatStore.Close() })
	t.Cleanup(func() { _ = wdb.CloseDatabase(root) })
	if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Enabled: true, BaseURL: baseURL,
		Models: []modelengine.ModelInfo{{ID: model}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	if err := c.SetFeatureModel("chat", "herdsman", model); err != nil {
		t.Fatalf("SetFeatureModel: %v", err)
	}
	a := &App{core: c}
	a.ctx = context.Background()
	a.client = ai.NewClient(a.cfg)
	a.client.SetEngineManager(a.engineMgr)
	a.whisperState = &whisperState{
		core:            c,
		app:             a,
		whisperDataRoot: root,
		weixinServers:   map[string]*weixin.Server{},
	}
	a.assistantMgr = assistant.NewEmpty(root)
	return a
}

func runProbeTurn(a *App, group, msg string) probeTurn {
	pt := probeTurn{Group: group, Msg: msg, SearchPredicate: shouldSearchWeb(msg)}
	start := time.Now()
	out, err := a.WhisperChatWithSearch(msg, probePersona, false, false)
	pt.TotalMs = time.Since(start).Milliseconds()
	if err != nil {
		pt.FinalType = "error"
		pt.Error = err.Error()
		return pt
	}
	pt.FinalType = "done"
	reply, _ := out["reply"].(string)
	pt.Reply = reply
	pt.ReplyChars = len([]rune(reply))
	pt.Stage, _ = out["stage"].(string)
	pt.Aff, _ = out["aff"].(float64)
	pt.Trust, _ = out["trust"].(float64)
	if tr, ok := out["trace"].(whisper.TurnTrace); ok {
		pt.EventType = string(tr.L0.Type)
		pt.GateLevel = tr.L3.GateLevel
		pt.StrategyLevel = tr.L3.StrategyLevel
		pt.ClampInjected = tr.L3.ClampInjected
		pt.SilentCalc = tr.L3.Silent
		pt.Label = tr.L2.Label
		pt.FactsInjected = len(tr.L3.InjectedFactIDs)
	}
	return pt
}

type probeReport struct {
	Label      string                 `json:"label"`
	Model      string                 `json:"model"`
	BaseURL    string                 `json:"baseUrl"`
	Persona    string                 `json:"persona"`
	StartedAt  string                 `json:"startedAt"`
	DurationS  float64                `json:"durationS"`
	Turns      []probeTurn            `json:"turns"`
	Continuous []probeTurn            `json:"continuous"`
	FollowUps  []probeTurn            `json:"followUps"`
	Restart    *probeRestart          `json:"restart"`
	Concurrent []probeTurn            `json:"concurrent"`
	Birthday   map[string]interface{} `json:"birthday"`
	FinalState map[string]interface{} `json:"finalState"`
}

type probeRestart struct {
	FactAnswer    string `json:"factAnswer"`
	FactPass      bool   `json:"factPass"`
	ContinueReply string `json:"continueReply"`
	ContinuePass  bool   `json:"continuePass"`
}

// TestWhisperProbeBaseline 实弹基线：设 GAEA_PROBE=1 + GAEA_PROBE_OUT=<目录> 运行。
func TestWhisperProbeBaseline(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹探针：GAEA_PROBE=1 才运行（CI/常规测试面跳过）")
	}
	baseURL := getenvDefault("GAEA_PROBE_BASE_URL", "http://127.0.0.1:8124/v1")
	model := getenvDefault("GAEA_PROBE_MODEL", "qwen35")
	outDir := os.Getenv("GAEA_PROBE_OUT")
	if outDir == "" {
		t.Fatal("GAEA_PROBE_OUT 未设置（报告输出目录）")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	rep := probeReport{
		Label:     getenvDefault("GAEA_PROBE_LABEL", "unlabeled"),
		Model:     model,
		BaseURL:   baseURL,
		Persona:   probePersona,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	started := time.Now()

	a := newChatProbeApp(t, baseURL, model)
	cleanupWhisperSession(t, probePersona)
	t.Cleanup(func() { cleanupWhisperSession(t, probePersona) })

	// 搜索腿罐头：离线确定性（两轮对照不变量），标记 [probe-search-canned] 供落库面断言
	origSearch := chatWebSearch
	chatWebSearch = func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "探针", Text: "探针罐头搜索结果：北京今天晴，气温 5 到 14 度。[probe-search-canned]"}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	groups := []struct {
		name string
		msgs []string
	}{
		{"G1_question", []string{"你今天过得怎么样？", "你觉得天气对心情有影响吗？", "你是怎么看待安静的？", "最近有什么让你开心的事吗？", "你觉得人为什么需要聊天？"}},
		{"G2_imperative_le5", []string{"讲个故事", "写首诗", "继续", "唱歌", "逗我笑"}},
		{"G3_imperative_6_9", []string{"讲一个短故事", "写一首小诗", "说个笑话吧", "给我唱首歌", "陪我聊聊天"}},
		{"G4_text_task", []string{"请用一百字左右介绍一下你自己的性格", "帮我写一句晚安短信，语气温柔一点", "写一首关于下雨的五言绝句", "用一句话总结我们刚才的对话", "帮我写一段三十字的自我介绍"}},
		{"G5_smalltalk", []string{"嗨", "在吗", "早呀", "晚上好", "在忙什么"}},
		{"G6_search", []string{"帮我查一下北京今天天气", "搜索一下最近的科技新闻", "今天金价多少"}},
		{"G7_memory", []string{"我叫小林，记住这个名字", "我最喜欢的颜色是蓝色，记一下", "我昨天说喜欢绿色是说错了，其实是蓝色"}},
	}
	for _, g := range groups {
		for _, msg := range g.msgs {
			pt := runProbeTurn(a, g.name, msg)
			rep.Turns = append(rep.Turns, pt)
			t.Logf("[%s/%s] %s -> %s %d字 gate=%s strategy=%s clamp=%v", g.name, pt.FinalType, msg, abbrev(pt.Reply, 40), pt.ReplyChars, pt.GateLevel, pt.StrategyLevel, pt.ClampInjected)
		}
	}

	// 15 轮连续闲聊（可证伪预测：非 gaea 全程 Aff<30、每轮压制行）
	continuous := []string{
		"在吗", "今天有点累", "忙什么呢你", "我下午喝了个咖啡", "你呢你平时干嘛",
		"说点什么让我开心点", "算了不说这个", "你觉得猫和狗哪个好", "我更喜欢猫", "为什么呀",
		"嗯有道理", "对了你会难过吗", "怎么说", "好吧我信了", "那晚安前再陪我聊一句",
	}
	for i, msg := range continuous {
		pt := runProbeTurn(a, fmt.Sprintf("C15_r%d", i+1), msg)
		rep.Continuous = append(rep.Continuous, pt)
		t.Logf("[C15 r%d] aff=%.1f stage=%s strategy=%s gate=%s -> %d字", i+1, pt.Aff, pt.Stage, pt.StrategyLevel, pt.GateLevel, pt.ReplyChars)
	}

	// 压制命中后下一轮中性长问（两臂实验协议要求）
	for _, pt := range append(append([]probeTurn{}, rep.Turns...), rep.Continuous...) {
		if pt.FinalType == "done" && pt.suppressionHit() {
			fu := runProbeTurn(a, "FOLLOWUP_after_"+pt.Group, "那你展开说说吧，我想听详细一点的版本")
			rep.FollowUps = append(rep.FollowUps, fu)
			if len(rep.FollowUps) >= 5 {
				break
			}
		}
	}

	// 重启差分：铺垫建立指代 → 模拟重启（会话缓存清除，同数据根重建）→ 两问
	setup := []string{"我想计划一次去成都的旅行，帮我记着这件事", "顺便记一下，我想吃火锅"}
	for _, msg := range setup {
		runProbeTurn(a, "RESTART_setup", msg)
	}
	whisperSessionsMu.Lock()
	delete(whisperSessions, "whisper_"+probePersona)
	whisperSessionsMu.Unlock()
	time.Sleep(200 * time.Millisecond)
	factTurn := runProbeTurn(a, "RESTART_fact", "我说想去哪里旅行来着？")
	contTurn := runProbeTurn(a, "RESTART_continue", "继续刚才那个")
	rep.Restart = &probeRestart{
		FactAnswer:    factTurn.Reply,
		FactPass:      strings.Contains(factTurn.Reply, "成都"),
		ContinueReply: contTurn.Reply,
		ContinuePass:  strings.Contains(contTurn.Reply, "成都") || strings.Contains(contTurn.Reply, "火锅"),
	}

	// 并发首轮：清会话后两路同时进（恢复发布序验收）
	whisperSessionsMu.Lock()
	delete(whisperSessions, "whisper_"+probePersona)
	whisperSessionsMu.Unlock()
	time.Sleep(200 * time.Millisecond)
	var cw sync.WaitGroup
	cc := make(chan probeTurn, 2)
	for i := 0; i < 2; i++ {
		cw.Add(1)
		go func() {
			defer cw.Done()
			cc <- runProbeTurn(a, "CONCURRENT_first", "你好")
		}()
	}
	cw.Wait()
	close(cc)
	for pt := range cc {
		rep.Concurrent = append(rep.Concurrent, pt)
	}

	time.Sleep(500 * time.Millisecond) // 异步落库收尾
	sessionID := "whisper_" + probePersona
	if st, err := repos.LoadCompanionStateFromDB(a.whisperState.whisperDataRoot, sessionID); err == nil && st != nil {
		bd := map[string]interface{}{"present": true}
		if st.AckemBirthday != nil {
			bd["ackemBirthday"] = st.AckemBirthday.Format("2006-01-02T15:04:05")
		} else {
			bd["ackemBirthday"] = nil
		}
		if st.FirstMetDate != nil {
			bd["firstMetDate"] = st.FirstMetDate.Format("2006-01-02T15:04:05")
		}
		bd["implicit"] = st.AckemBirthday != nil && st.FirstMetDate != nil && st.AckemBirthday.Equal(*st.FirstMetDate)
		rep.Birthday = bd
		rep.FinalState = map[string]interface{}{
			"totalTurns": st.Counters.TotalTurns, "sharedEvents": st.Counters.SharedEventsCount,
			"aff": st.Emotion.Aff, "trust": st.Relationship.Trust, "stage": string(st.Relationship.Stage),
		}
	} else {
		rep.Birthday = map[string]interface{}{"present": false}
	}
	rep.DurationS = time.Since(started).Seconds()

	writeProbeReports(t, outDir, rep)
}

func getenvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func abbrev(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return string(r)
}

func writeProbeReports(t *testing.T, outDir string, rep probeReport) {
	t.Helper()
	jb, err := json.MarshalIndent(rep, "", " ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	jp := filepath.Join(outDir, "report.json")
	if err := os.WriteFile(jp, jb, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# 实弹探针报告：%s\n\n- 模型：%s（%s）\n- 角色：%s（非 gaea）\n- 耗时：%.0fs\n\n", rep.Label, rep.Model, rep.BaseURL, rep.Persona, rep.DurationS)
	summary := func(name string, turns []probeTurn) {
		if len(turns) == 0 {
			return
		}
		done, empty, clamp, silent, gate, search := 0, 0, 0, 0, 0, 0
		var chars int64
		for _, p := range turns {
			if p.FinalType == "done" {
				done++
				chars += int64(p.ReplyChars)
				if p.ReplyChars == 0 {
					empty++
				}
			}
			if p.ClampInjected {
				clamp++
			}
			if p.StrategyLevel == "silent" || p.StrategyLevel == "whisper" {
				silent++
			}
			if p.GateLevel == "silent" || p.GateLevel == "whisper" {
				gate++
			}
			if p.SearchPredicate {
				search++
			}
		}
		fmt.Fprintf(&b, "## %s（%d 轮）\n- done %d / error %d，空回复 %d\n- 平均正文 %.0f 字；钳制命中 %d；strategy 压制 %d；gate 压制 %d\n- 搜索谓词命中 %d\n\n", name, len(turns), done, len(turns)-done, empty, float64(chars)/float64(max(done, 1)), clamp, silent, gate, search)
	}
	summary("五桶探针", rep.Turns)
	summary("15 轮连续闲聊", rep.Continuous)
	summary("压制命中后追问", rep.FollowUps)
	if rep.Restart != nil {
		fmt.Fprintf(&b, "## 重启差分\n- 事实回忆（应含成都）：%v → %s\n- 续接（应含成都/火锅）：%v → %s\n\n", rep.Restart.FactPass, abbrev(rep.Restart.FactAnswer, 60), rep.Restart.ContinuePass, abbrev(rep.Restart.ContinueReply, 60))
	}
	if len(rep.Concurrent) > 0 {
		for _, p := range rep.Concurrent {
			fmt.Fprintf(&b, "- 并发首轮：%s %s（%dms）\n", p.FinalType, p.Error, p.TotalMs)
		}
		b.WriteString("\n")
	}
	bd, _ := json.Marshal(rep.Birthday)
	fmt.Fprintf(&b, "## 生日事实\n- %s\n\n", bd)
	fs, _ := json.Marshal(rep.FinalState)
	fmt.Fprintf(&b, "## 终态\n- %s\n", fs)
	mp := filepath.Join(outDir, "summary.md")
	if err := os.WriteFile(mp, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	t.Logf("探针报告已写出：%s / %s", jp, mp)
}
