package app

// revolution 刀8 验收·线 B：新建生成主轴跨模块端到端测试。
//
// 主轴 = CreateChapter（桩 LLM）→ 章落盘+场景重置物化 → done（aiTaste）→
// 异步自动门 runAutoGateAfterGeneration → analysisAgent.Analyze（一次 LLM）→
// analysis-v2.json 落盘 + memories 回填 + annotations 落盘 + 伏笔同步。
// 另附全书体检 50 章聚合（零 LLM）。helper 一律 axis 前缀（同包防撞），
// 复用 waitFor/waitGensDone/mustWriteChapter，零生产代码改动。

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/httpbridge"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// ── 正文与分析载荷构造 ───────────────────────────────────────

// 分析 keyword 锚点（8-25 rune，正文逐字包含，保证标注 pos≥0 命中）。
const (
	axisHookAnchor        = "把那半块虎符收进袖中"
	axisForeshadowAnchor  = "巷口的黑影一闪，裴照按住了刀柄"
	axisPlotAnchor        = "虎符合璧之日，便是旧案重开之时"
	axisForeshadowTitle   = "虎符之谜"
	axisForeshadowContent = "半块虎符现世，另半块下落不明，暗指十年前旧案。"
)

// axisChapterBody 程序化拼 ≥1300 rune 的多样正文（叙述+对话+多段落），
// 三个 keyword 锚点均逐字埋入（目标字数 1000 → 单次生成即达标，无续写循环）。
func axisChapterBody() string {
	paras := []string{
		"林晚把那半块虎符收进袖中，指腹压住裂口，没有回头看裴照。",
		"「城南的茶棚今晚不点灯。」裴照把伞压低了些，「你最好当作什么都没看见。」",
		"更夫的梆子敲过三下，雨点开始密集地砸在青石板上，溅起细小的水花。",
		"她想起父亲临终前留下的那句话：虎符合璧之日，便是旧案重开之时。",
		"巷口的黑影一闪，裴照按住了刀柄，两人在雨里无声对峙了半晌。",
		"「你到底站在哪边？」林晚终于开口，声音比雨水还要冷。",
		"半块虎符在她袖中发烫，像一块不肯熄灭的炭。",
		"老周头缩在门槛后面，只敢露出半只眼睛朝巷口张望。",
		"裴照收了刀，转身走进雨幕深处，脚印很快被水淹没。",
		"林晚立在原地，把方才每一个细节在心里过了一遍，又一遍。",
	}
	var sb strings.Builder
	for i := 0; utf8.RuneCountInString(sb.String()) < 1300; i++ {
		sb.WriteString(paras[i%len(paras)])
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

// axisAnalysisJSON 构造一次 Unmarshal 即过的合法分析 V2 JSON：
// 1 hook + 1 planted 伏笔 + 1 角色状态 + 最小合法 conflict/emotional_arc，
// 三维分 (8/8.5/8)→overall 8.2≥8 → 建议上限 1 条（联动裁剪不裁光）。
// 外层由桩包 ```json 围栏（RetryJSON 的 ExtractJSON 对「裸 JSON=全文」
// 判为未找到，围栏保证一次成功）。
func axisAnalysisJSON() string {
	return fmt.Sprintf(`{
  "hooks": [{"type": "悬念", "content": "林晚收起半块虎符，与裴照雨夜对峙。", "strength": 8, "position": "开头", "keyword": {"text": %q, "pos": -1, "length": 0}}],
  "foreshadows": [{"title": %q, "content": %q, "type": "planted", "strength": 7, "subtlety": 6, "category": "item", "is_long_term": true, "related_characters": ["林晚"], "estimated_resolve_chapter": 5, "keyword": {"text": %q, "pos": -1, "length": 0}}],
  "conflict": {"types": ["人与环境"], "parties": ["林晚", "裴照"], "level": 8, "description": "雨夜对峙，各怀心思。", "resolution_progress": 0.3},
  "emotional_arc": {"primary_emotion": "警觉", "intensity": 7, "curve": "由静转紧", "secondary_emotions": ["疑虑"]},
  "character_states": [{"name": "林晚", "old_state": "旁观", "new_state": "决意追查", "psychological_change": "由避事转向担责", "key_event": "收起半块虎符"}],
  "plot_points": [{"content": "虎符合璧的传言被点破", "type": "revelation", "importance": 0.8, "impact": "旧案重启", "keyword": {"text": %q, "pos": -1, "length": 0}}],
  "scenes": [{"location": "城南巷口", "atmosphere": "雨夜压抑", "duration": "一夜"}],
  "pacing": "varied",
  "dialogue_ratio": 0.4,
  "description_ratio": 0.6,
  "scores": {"pacing": 8.0, "engagement": 8.5, "coherence": 8.0, "justification": "节奏张弛有度"},
  "plot_stage": "第一幕",
  "suggestions": ["中段可再压一笔裴照的迟疑"],
  "summary": "雨夜，林晚收起半块虎符，与裴照对峙后决意追查旧案。"
}`, axisHookAnchor, axisForeshadowTitle, axisForeshadowContent, axisForeshadowAnchor, axisPlotAnchor)
}

// ── fixture：分流桩 + App 装配 ──────────────────────────────

// axisStubStats 桩分流计数（验证生成单次、分析恰好一次）。
type axisStubStats struct {
	genCalls      atomic.Int64
	analysisCalls atomic.Int64
}

// newAxisTestApp 构造主轴测试 App：httptest SSE 桩读完整请求体分流——
// 含「小说分析编辑」（analysis-chapter 系统提示词开头）回分析 V2 JSON，
// 否则回章节正文（单 delta+flush，仿 newRewriteTestApp）。App 装配仿
// cancel 测试，再挂 analysisAgent（同一桩 client → 分析走成功路径）。
func newAxisTestApp(t *testing.T) (*App, *project.Manager, *axisStubStats) {
	t.Helper()
	stats := &axisStubStats{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		var reply string
		if strings.Contains(string(body), "小说分析编辑") {
			stats.analysisCalls.Add(1)
			reply = "```json\n" + axisAnalysisJSON() + "\n```"
		} else {
			stats.genCalls.Add(1)
			reply = axisChapterBody()
		}
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", strconv.Quote(reply))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "主轴测试小说", "悬疑", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	t.Cleanup(func() { _ = pm.Close() })

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.ctx = context.Background()
	a.setPM(pm)
	a.analysisAgent = analysis.New(client, pm, cfg, prompt.NewEngine("../../prompts"))
	return a, pm, stats
}

// ── chapter-gate 事件订阅（httpbridge 全局 hub 的 SSE 面）────

// axisSubscribeGate 经 httpbridge.New(...).Handler() 订阅 /api/stream?id=
// chapter-gate（hub 是包级 globalHub，app.emit→Publish 即达）。同步消费
// connected 帧确保订阅已登记后才放行；返回事件快照函数。
func axisSubscribeGate(t *testing.T) func() []map[string]interface{} {
	t.Helper()
	srv := httptest.NewServer(httpbridge.New(nil).Handler())
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/stream?id=chapter-gate", nil)
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		srv.Close()
		t.Fatalf("打开 chapter-gate SSE: %v", err)
	}
	t.Cleanup(func() { cancel(); resp.Body.Close(); srv.Close() })

	br := bufio.NewReader(resp.Body)
	// 消费 connected 帧（event: connected / data: {...} / 空行）= 订阅已登记
	for i := 0; i < 3; i++ {
		if _, err := br.ReadString('\n'); err != nil {
			t.Fatalf("读 connected 帧: %v", err)
		}
	}

	var mu sync.Mutex
	var events []map[string]interface{}
	go func() {
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var m map[string]interface{}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &m) == nil {
				mu.Lock()
				events = append(events, m)
				mu.Unlock()
			}
		}
	}()
	return func() []map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]interface{}(nil), events...)
	}
}

// axisWaitGateReport 轮询等待第 1 章的 chapter-gate 报告事件。
func axisWaitGateReport(t *testing.T, snapshot func() []map[string]interface{}, timeout time.Duration) map[string]interface{} {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, ev := range snapshot() {
			if ev["type"] == "report" && ev["chapterNum"] == float64(1) {
				return ev
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("等待 chapter-gate 报告事件超时")
	return nil
}

// ── 测试 1：生成主轴端到端 ──────────────────────────────────

// TestKnife8_GenerationAxis_EndToEnd 主轴贯通：CreateChapter（目标 1000 字、
// 正文 ≥1200 rune → 单次 LLM 无续写）→ 章落盘 + 场景重置物化 → done →
// 异步自动门 → Analyze 一次 LLM → V2/memories/annotations/伏笔四路落盘 +
// chapter-gate 事件。先订阅事件再生成（hub 无订阅者即丢弃）。
func TestKnife8_GenerationAxis_EndToEnd(t *testing.T) {
	a, pm, stats := newAxisTestApp(t)

	body := axisChapterBody()
	if n := utf8.RuneCountInString(body); n < 1200 {
		t.Fatalf("测试正文应 ≥1200 rune，实际 %d", n)
	}

	// 事件订阅必须在生成前（Publish 对无订阅者是 no-op）
	gateEvents := axisSubscribeGate(t)

	// 预置旧正文+旧场景：done 段 rebuildScenesFromBlob 应删除旧场景并从新
	// blob 物化单场景（重置语义真跑，而非惰性物化兜底）。
	if err := pm.WriteChapter(1, "旧正文，将被整章重写覆盖。"); err != nil {
		t.Fatalf("写旧正文: %v", err)
	}
	sm := pm.SceneManager(1)
	oldSc, err := sm.Create("opening", "旧场景")
	if err != nil {
		t.Fatalf("建旧场景: %v", err)
	}
	oldSc.Content = "旧正文，将被整章重写覆盖。"
	if err := sm.Write(oldSc); err != nil {
		t.Fatalf("写旧场景: %v", err)
	}

	res, err := a.CreateChapter("大陆设定", "", "雨夜夺符", 1, "", "", 1000, 0)
	if err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}
	if res["streaming"] != true || res["chapterNum"] != 1 {
		t.Fatalf("返回负载不对: %v", res)
	}
	waitGensDone(t, a) // done 已发（生成协程整体退出）

	// ① 章落盘：正文含锚点
	data, err := os.ReadFile(pm.ChapterPath(1))
	if err != nil {
		t.Fatalf("读章节文件: %v", err)
	}
	if !strings.Contains(string(data), axisHookAnchor) {
		t.Fatalf("章节文件应含新正文锚点（前 60 字）: %q", axisTruncateRunes(string(data), 60))
	}

	// ② 自动门过水：analysis-v2.json 出现第 1 章条目（V2 类型+归一化生效）
	waitFor(t, 10*time.Second, "自动门分析 V2 落盘", func() bool {
		af, err := pm.ReadAnalysisV2File()
		return err == nil && len(af.Items) == 1 && af.Items[0].ChapterNum == 1 &&
			af.Items[0].AnalyzerSource == "llm"
	})
	af, _ := pm.ReadAnalysisV2File()
	v2 := af.Items[0].Result
	if v2.Scores.Overall != 8.2 { // (8+8.5+8)/3 服务端权威重算
		t.Fatalf("overall 应归一为 8.2: %+v", v2.Scores)
	}
	if len(v2.Hooks) != 1 || len(v2.Foreshadows) != 1 || v2.Foreshadows[0].Type != "planted" {
		t.Fatalf("V2 载荷维度不对: %+v", v2)
	}

	// ③ memories 回填：非空且含 chapter_summary 与 foreshadow 两类
	mems, err := pm.ReadChapterMemories(1)
	if err != nil || len(mems.Items) == 0 {
		t.Fatalf("章节记忆应非空: err=%v items=%d", err, len(mems.Items))
	}
	seenSummary, seenForeshadow := false, false
	for _, m := range mems.Items {
		switch m.Type {
		case types.MemoryTypeChapterSummary:
			seenSummary = true
		case types.MemoryTypeForeshadow:
			seenForeshadow = true
		}
	}
	if !seenSummary || !seenForeshadow {
		t.Fatalf("记忆应含 chapter_summary 与 foreshadow: %+v", mems.Items)
	}

	// ④ annotations 落盘：≥1 条 pos≥0（锚点命中）
	ann, err := pm.ReadChapterAnnotations(1)
	if err != nil || len(ann.Items) == 0 {
		t.Fatalf("章节标注应非空: err=%v items=%d", err, len(ann.Items))
	}
	anchored := 0
	for _, it := range ann.Items {
		if it.Pos >= 0 {
			anchored++
		}
	}
	if anchored < 1 {
		t.Fatalf("应有 ≥1 条锚定标注（pos≥0）: %+v", ann.Items)
	}

	// ⑤ 伏笔同步：桩 JSON 的 planted title 出现且登记在 001.md
	ff, err := pm.ReadForeshadows()
	if err != nil {
		t.Fatalf("读伏笔: %v", err)
	}
	foundFF := false
	for _, f := range ff.Items {
		if f.Title == axisForeshadowTitle && f.Status == types.ForeshadowPlanted && f.PlantedIn == "001.md" {
			foundFF = true
		}
	}
	if !foundFF {
		t.Fatalf("伏笔登记表应含 planted %q: %+v", axisForeshadowTitle, ff.Items)
	}

	// ⑥ 场景重置物化：旧场景被删，从新 blob 物化唯一 done 场景
	metas, err := pm.SceneManager(1).List()
	if err != nil || len(metas) != 1 {
		t.Fatalf("重置后应恰 1 个场景: err=%v n=%d", err, len(metas))
	}
	if !strings.HasSuffix(metas[0].ID, "-chapter") {
		t.Errorf("物化场景 id 应以 -chapter 结尾: %q", metas[0].ID)
	}
	newSc, err := pm.SceneManager(1).Read(metas[0].ID)
	if err != nil || !strings.Contains(newSc.Content, axisHookAnchor) || newSc.Meta.Status != types.SceneDone {
		t.Fatalf("物化场景应为新正文 done 场景: err=%v status=%v", err, newSc.Meta.Status)
	}

	// ⑦ chapter-gate 事件：analysisDone=true + foreshadowSync 计数（事件在
	// 全部落盘之后 emit，等到即四路过水完毕）
	report := axisWaitGateReport(t, gateEvents, 10*time.Second)
	if report["analysisDone"] != true {
		t.Fatalf("chapter-gate 应 analysisDone=true: %+v", report)
	}
	if report["branch"] != "" {
		t.Fatalf("主线章 branch 应为空: %+v", report)
	}
	fsync, ok := report["foreshadowSync"].(map[string]interface{})
	if !ok || fsync["plantedCount"] != float64(1) || fsync["createdCount"] != float64(1) {
		t.Fatalf("foreshadowSync 应 planted=1/created=1: %+v", report["foreshadowSync"])
	}
	if report["aiTasteScore"] == nil {
		t.Fatalf("干净长文应有 AI 味分: %+v", report)
	}

	// ⑧ 桩分流：生成单次（无续写循环）、分析恰好一次（RetryJSON 一次成功）
	if n := stats.genCalls.Load(); n != 1 {
		t.Fatalf("生成应单次 LLM 调用，实际 %d", n)
	}
	if n := stats.analysisCalls.Load(); n != 1 {
		t.Fatalf("分析应恰好一次 LLM 调用，实际 %d", n)
	}
}

// axisTruncateRunes 按 rune 截断（仅失败信息展示用）。
func axisTruncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ── 测试 2：全书体检 50 章 ──────────────────────────────────

// TestKnife8_BookHealth_FiftyChapters 50 章全书体检（零 LLM）：奇数章脏
// （单段 >400 rune 段落触发 novelgate S3 paragraph_too_long）、偶数章干净；
// 预落 3 章 V2；聚合断言章数/脏章数/V2 覆盖/AI 味逐章可打分/耗时上限。
func TestKnife8_BookHealth_FiftyChapters(t *testing.T) {
	a := newFingerprintTestApp(t)
	pm := a.getPM()

	// 脏章样本：300 句同段连排（~6900 rune 单段）——只触发 S3 超长段落
	// （warn 级）：句长/标点/省略号各路均不命中（既有邻域测试同款口径）。
	dirty := strings.Repeat("他缓缓地走向前方那扇门，门后的黑暗仿佛在呼吸。", 300)
	for i := 1; i <= 50; i++ {
		if i%2 == 1 {
			if err := pm.WriteChapter(i, dirty); err != nil {
				t.Fatalf("写脏章 %d: %v", i, err)
			}
		} else {
			mustWriteChapter(t, a, i, fmt.Sprintf("第%d章夜巡。", i))
		}
	}
	// 预落 V2 于 3 章（覆盖率口径：条目章号 ∈ [1,50] 即计入）
	for _, n := range []int{2, 10, 30} {
		if err := pm.UpsertAnalysisV2(types.ChapterAnalysisResult{ChapterNum: n, AnalyzerSource: "manual"}); err != nil {
			t.Fatalf("预落 V2 章 %d: %v", n, err)
		}
	}

	start := time.Now()
	report, err := a.RunBookHealthCheck()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("全书体检: %v", err)
	}

	if report.TotalChapters != 50 || len(report.Chapters) != 50 {
		t.Fatalf("章数不对: total=%d rows=%d", report.TotalChapters, len(report.Chapters))
	}
	if report.QualityIssueChapters != 25 {
		t.Fatalf("脏章应 25（奇数章各 1 条 S3）: %d", report.QualityIssueChapters)
	}
	if report.AnalyzedChapters != 3 {
		t.Fatalf("V2 覆盖应 3 章: %d", report.AnalyzedChapters)
	}
	// 无大纲节点项目：gateOutlineIssues 找不到本章节点 → 空表，契约问题章=0
	if report.ContractIssueChapters != 0 {
		t.Fatalf("无大纲项目不应有契约问题章: %d", report.ContractIssueChapters)
	}
	for _, row := range report.Chapters {
		if row.AiTasteScore < 0 {
			t.Fatalf("第%d章 AI 味分缺失（-1）", row.ChapterNum)
		}
		if row.ChapterNum%2 == 1 {
			if row.QualityIssues < 1 || row.QualityErrors > 0 {
				t.Fatalf("脏章应只触发 warn 级质量问题: %+v", row)
			}
		} else if row.QualityIssues != 0 {
			t.Fatalf("干净章不应有质量问题: %+v", row)
		}
	}
	if elapsed >= 60*time.Second {
		t.Fatalf("全书体检（零 LLM）耗时 %v 超上限", elapsed)
	}
	t.Logf("50 章体检耗时 %s", elapsed)
}
