package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/graph"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
)

// ── 打桩：本地 OpenAI 兼容 mock（httptest），打桩方式对齐 plot_branch_handler_test.go ──

// consistencyDeepTestEnv 一致性深检测试最小环境：可计数假 LLM + 临时小说项目
type consistencyDeepTestEnv struct {
	a     *App
	pm    *project.Manager
	calls *int32 // AI 请求计数
}

// newConsistencyDeepEnv replies 按调用次序回放（SSE 分片回放 delta.content）；超出后重复最后一条。
// clientNil=true 时不注册任何引擎/客户端，用于测试 client nil 降级路径。
func newConsistencyDeepEnv(t *testing.T, replies []string, clientNil bool) *consistencyDeepTestEnv {
	t.Helper()

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	var calls int32
	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	var client *ai.Client

	if !clientNil {
		client = ai.NewClient(cfg)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idx := int(atomic.AddInt32(&calls, 1)) - 1
			if idx >= len(replies) {
				idx = len(replies) - 1
			}
			payload, _ := json.Marshal(map[string]interface{}{
				"choices": []map[string]interface{}{{
					"delta":         map[string]interface{}{"content": replies[idx]},
					"finish_reason": nil,
				}},
			})
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			fmt.Fprintf(w, "data: %s\n\n", payload)
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
		}))
		t.Cleanup(srv.Close)

		if err := engMgr.SaveEngine(modelengine.EngineConfig{
			ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
			BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
		}); err != nil {
			t.Fatalf("SaveEngine: %v", err)
		}
		client.SetEngineManager(engMgr)
	}
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "深检测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	a.ctx = ctx
	a.setPM(pm)

	return &consistencyDeepTestEnv{a: a, pm: pm, calls: &calls}
}

// seedChapter 写一章正文（走 pm.WriteChapter，主线）
func (e *consistencyDeepTestEnv) seedChapter(t *testing.T, num int, content string) {
	t.Helper()
	if err := e.pm.WriteChapter(num, content); err != nil {
		t.Fatalf("写第%d章: %v", num, err)
	}
}

// stateCardReply 构造一段状态卡 AI 回复（```json 围栏，测 ExtractJSON 容错）
func stateCardReply(t *testing.T, card deepStateCard) string {
	t.Helper()
	b, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal card: %v", err)
	}
	return "```json\n" + string(b) + "\n```"
}

// ── 纯函数：状态卡解析 ───────────────────────────────────────

// TestDeepParseCard 状态卡 JSON 解析：裸 JSON / ```json 围栏 / 非 JSON 三种回复
func TestDeepParseCard(t *testing.T) {
	raw := `{"chapter":3,"branch":"a","time_mark":"第三日夜","time_relation":"later",
		"scene_notes":["夜探宗门"],
		"travel_notes":["连夜赶往北境"],
		"characters":[{"name":"林晚","status":"alive","location":"北境","items":["玄铁剑"]}],
		"items_lost":["护身符"],"items_regained":[]}`

	card, err := deepParseCard(raw)
	if err != nil {
		t.Fatalf("裸 JSON 解析失败: %v", err)
	}
	if card.Chapter != 3 || card.Branch != "a" || card.TimeMark != "第三日夜" || card.TimeRelation != "later" {
		t.Fatalf("字段不符: %+v", card)
	}
	if len(card.Characters) != 1 || card.Characters[0].Name != "林晚" || card.Characters[0].Status != "alive" {
		t.Fatalf("characters 不符: %+v", card.Characters)
	}
	if len(card.Characters[0].Items) != 1 || card.Characters[0].Items[0] != "玄铁剑" {
		t.Fatalf("items 不符: %+v", card.Characters[0].Items)
	}
	if len(card.ItemsLost) != 1 || card.ItemsLost[0] != "护身符" {
		t.Fatalf("items_lost 不符: %+v", card.ItemsLost)
	}

	fenced, err := deepParseCard("前置废话\n```json\n" + raw + "\n```\n后置废话")
	if err != nil {
		t.Fatalf("围栏 JSON 解析失败: %v", err)
	}
	if fenced.TimeMark != "第三日夜" {
		t.Fatalf("围栏解析字段丢失: %+v", fenced)
	}

	if _, err := deepParseCard("这不是 JSON"); err == nil {
		t.Fatalf("非 JSON 回复应报错")
	}
}

// ── 纯函数：跨章比对 ─────────────────────────────────────────

// TestDeepClampMaxChapters 深检窗口夹取
func TestDeepClampMaxChapters(t *testing.T) {
	cases := map[int]int{-5: 50, 0: 50, 20: 20, 51: 50, 49: 49}
	for in, want := range cases {
		if got := deepClampMaxChapters(in); got != want {
			t.Fatalf("deepClampMaxChapters(%d) = %d, want %d", in, got, want)
		}
	}
}

// TestDeepCompareStateLineDeadReappear 死亡角色后期以存活状态再出场 → error，且复活只报一次
func TestDeepCompareStateLineDeadReappear(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "dead", Location: ""}}},
		{Chapter: 3, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 4, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
	}
	issues := deepCompareStateLine(cards)

	var hits []graph.ConsistencyIssue
	for _, iss := range issues {
		if iss.Category == "status" && iss.Severity == "error" && strings.Contains(iss.EntityName, "林晚") {
			hits = append(hits, iss)
		}
	}
	if len(hits) != 1 {
		t.Fatalf("期望恰好 1 条死亡后再出场告警，得到 %d: %+v", len(hits), issues)
	}
	if !strings.Contains(hits[0].Description, "第2章已死亡") || !strings.Contains(hits[0].Description, "第3章") {
		t.Fatalf("告警描述缺章节信息: %s", hits[0].Description)
	}
	if hits[0].Branch != "" || hits[0].Location != "第3章" {
		t.Fatalf("分支/位置标签不符: %+v", hits[0])
	}
}

// TestDeepCompareStateLineDeadInBranch 分支线死亡角色在本分支后续章再出场 → 带分支标记
func TestDeepCompareStateLineDeadInBranch(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 3, Branch: "a", Characters: []deepCharacterState{{Name: "陈九", Status: "dead"}}},
		{Chapter: 5, Branch: "a", Characters: []deepCharacterState{{Name: "陈九", Status: "alive", Location: "黑市"}}},
	}
	issues := deepCompareStateLine(cards)
	if len(issues) != 1 {
		t.Fatalf("期望 1 条告警: %+v", issues)
	}
	if issues[0].Branch != "a" || !strings.Contains(issues[0].Location, "分支a") {
		t.Fatalf("分支标记丢失: %+v", issues[0])
	}
}

// TestDeepCompareStateLineItemVanishAndConjure 物品凭空消失（warning）→ 无中生有（error）→ 有重新获得交代则不报
func TestDeepCompareStateLineItemVanishAndConjure(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"玄铁剑"}}}},
		// 玄铁剑消失且无 items_lost 交代 → warning
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		// 玄铁剑又出现但无 items_regained 交代 → error
		{Chapter: 3, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"玄铁剑"}}}},
	}
	issues := deepCompareStateLine(cards)

	var vanish, conjure *graph.ConsistencyIssue
	for i := range issues {
		iss := issues[i]
		if iss.Category == "item" && strings.Contains(iss.EntityName, "玄铁剑") {
			if iss.Severity == "warning" && vanish == nil {
				v := iss
				vanish = &v
			}
			if iss.Severity == "error" && conjure == nil {
				c := iss
				conjure = &c
			}
		}
	}
	if vanish == nil {
		t.Fatalf("缺「凭空消失」告警: %+v", issues)
	}
	if !strings.Contains(vanish.Description, "凭空消失") || !strings.Contains(vanish.Location, "第2章") {
		t.Fatalf("凭空消失告警不符: %+v", *vanish)
	}
	if conjure == nil {
		t.Fatalf("缺「无中生有」告警: %+v", issues)
	}
	if !strings.Contains(conjure.Description, "无中生有") || !strings.Contains(conjure.Description, "第2章已标记失去") {
		t.Fatalf("无中生有告警不符: %+v", *conjure)
	}

	// 有 items_regained 交代 → 无中生有不再报
	cards[2].ItemsRegained = []string{"玄铁剑"}
	for _, iss := range deepCompareStateLine(cards) {
		if iss.Category == "item" && iss.Severity == "error" {
			t.Fatalf("有重新获得交代仍报无中生有: %+v", iss)
		}
	}

	// 上一章 items_lost 有交代（本章交出）→ 凭空消失不再报
	cards[1].Characters = []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}
	cards[1].ItemsLost = []string{"玄铁剑"}
	cards[2].Characters = []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}
	cards[2].ItemsRegained = nil
	for _, iss := range deepCompareStateLine(cards) {
		if iss.Category == "item" && iss.Severity == "warning" {
			t.Fatalf("items_lost 有交代仍报凭空消失: %+v", iss)
		}
	}
}

// TestDeepCompareStateLineTeleportAndTime 位置瞬移（无 travel_notes 交代）+ 时间倒流
func TestDeepCompareStateLineTeleportAndTime(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 1, TimeMark: "第一日", TimeRelation: "unknown", Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 2, TimeMark: "第一日晨", TimeRelation: "earlier", Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "北境荒原"}}},
	}
	issues := deepCompareStateLine(cards)

	var teleport, rewind bool
	for _, iss := range issues {
		if iss.Category == "status" && strings.Contains(iss.Description, "位置瞬移") {
			teleport = true
		}
		if iss.Category == "timeline" && iss.Severity == "error" && strings.Contains(iss.Description, "时间倒流") {
			rewind = true
		}
	}
	if !teleport {
		t.Fatalf("缺「位置瞬移」告警: %+v", issues)
	}
	if !rewind {
		t.Fatalf("缺「时间倒流」告警: %+v", issues)
	}

	// 有 travel_notes 交代 → 不报瞬移；time_relation 改 later → 不报倒流
	cards[1].TravelNotes = []string{"连夜御剑赶往北境"}
	cards[1].TimeRelation = "later"
	if issues = deepCompareStateLine(cards); len(issues) != 0 {
		t.Fatalf("有交代后不应再报: %+v", issues)
	}
}

// ── httptest 桩 AI：端到端提取 + 比对 + 合并 ─────────────────

// TestCheckConsistencyDeepAIStub 三章正文：AI 逐章返回状态卡，第2章林晚死亡、第3章复活 → AI 告警 + 规则告警合并
func TestCheckConsistencyDeepAIStub(t *testing.T) {
	replies := []string{
		stateCardReply(t, deepStateCard{
			Chapter: 1, TimeMark: "第一日", TimeRelation: "unknown", SceneNotes: []string{"林晚在青云宗练剑"},
			Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"玄铁剑"}}},
		}),
		stateCardReply(t, deepStateCard{
			Chapter: 2, TimeMark: "第二日", TimeRelation: "later", SceneNotes: []string{"林晚力战妖兽身亡"},
			Characters: []deepCharacterState{{Name: "林晚", Status: "dead", Location: "青云宗"}},
		}),
		stateCardReply(t, deepStateCard{
			Chapter: 3, TimeMark: "第三日", TimeRelation: "later", SceneNotes: []string{"林晚活蹦乱跳出现"},
			Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"玄铁剑"}}},
		}),
	}
	env := newConsistencyDeepEnv(t, replies, false)
	for n := 1; n <= 3; n++ {
		env.seedChapter(t, n, fmt.Sprintf("第%d章正文：林晚的冒险故事，青云宗的清晨。", n))
	}

	res, err := env.a.CheckConsistencyDeep(20)
	if err != nil {
		t.Fatalf("CheckConsistencyDeep: %v", err)
	}

	if v, _ := res["ai_available"].(bool); !v {
		t.Fatalf("ai_available 应为 true: %v（note=%v）", res["ai_available"], res["ai_note"])
	}
	if v, _ := res["chapters_scanned"].(int); v != 3 {
		t.Fatalf("chapters_scanned = %v, want 3", res["chapters_scanned"])
	}
	if got := *env.calls; got != 3 {
		t.Fatalf("AI 调用次数 = %d, want 3（逐章一次）", got)
	}

	issues, _ := res["issues"].([]map[string]interface{})
	if len(issues) == 0 {
		t.Fatalf("应有合并告警: %v", res)
	}
	var aiDead bool
	ruleCount := 0
	for _, iss := range issues {
		source, _ := iss["source"].(string)
		switch source {
		case "ai":
			if iss["entity_name"] == "林晚" && iss["severity"] == "error" && strings.Contains(iss["description"].(string), "第2章已死亡") {
				aiDead = true
			}
		case "rule":
			ruleCount++
		default:
			t.Fatalf("告警缺 source 字段: %v", iss)
		}
	}
	if !aiDead {
		t.Fatalf("缺 AI 死亡后再出场告警: %v", issues)
	}
	if v, _ := res["total_issues"].(int); v != len(issues) {
		t.Fatalf("total_issues = %d, len(issues) = %d", v, len(issues))
	}
	if s, _ := res["summary"].(string); !strings.Contains(s, "AI 深检已扫描 3 章") {
		t.Fatalf("summary 缺深检信息: %s", s)
	}
}

// TestCheckConsistencyDeepWindow 窗口夹取：5 章只深检最近 2 章（maxChapters=2）
func TestCheckConsistencyDeepWindow(t *testing.T) {
	replies := []string{stateCardReply(t, deepStateCard{Chapter: 4, TimeRelation: "unknown"}), stateCardReply(t, deepStateCard{Chapter: 5, TimeRelation: "later"})}
	env := newConsistencyDeepEnv(t, replies, false)
	for n := 1; n <= 5; n++ {
		env.seedChapter(t, n, fmt.Sprintf("第%d章正文：普通推进。", n))
	}

	res, err := env.a.CheckConsistencyDeep(2)
	if err != nil {
		t.Fatalf("CheckConsistencyDeep: %v", err)
	}
	if v, _ := res["chapters_scanned"].(int); v != 2 {
		t.Fatalf("chapters_scanned = %v, want 2", res["chapters_scanned"])
	}
	if got := *env.calls; got != 2 {
		t.Fatalf("AI 调用次数 = %d, want 2", got)
	}
}

// TestCheckConsistencyDeepPartialFailure 单章 AI 失败跳过不中断：3 章中第 2 章 RetryJSON 三个
// 回合（初次 + 2 次重试）全部回非 JSON
func TestCheckConsistencyDeepPartialFailure(t *testing.T) {
	replies := []string{
		stateCardReply(t, deepStateCard{Chapter: 1, TimeRelation: "unknown"}),
		"抱歉，我无法输出 JSON。", "实在没有 JSON", "仍然不是 JSON", // 第 2 章：耗尽重试
		stateCardReply(t, deepStateCard{Chapter: 3, TimeRelation: "later"}),
		stateCardReply(t, deepStateCard{Chapter: 3, TimeRelation: "later"}), // 重试兜底
	}
	env := newConsistencyDeepEnv(t, replies, false)
	for n := 1; n <= 3; n++ {
		env.seedChapter(t, n, fmt.Sprintf("第%d章正文：普通推进。", n))
	}

	res, err := env.a.CheckConsistencyDeep(20)
	if err != nil {
		t.Fatalf("单章失败不应中断整体: %v", err)
	}
	if v, _ := res["chapters_scanned"].(int); v != 2 {
		t.Fatalf("chapters_scanned = %v, want 2", res["chapters_scanned"])
	}
	if v, _ := res["chapters_failed"].(int); v != 1 {
		t.Fatalf("chapters_failed = %v, want 1", res["chapters_failed"])
	}
	if v, _ := res["ai_available"].(bool); !v {
		t.Fatalf("部分成功时 ai_available 应为 true")
	}
	if note, _ := res["ai_note"].(string); !strings.Contains(note, "1 章 AI 提取失败已跳过") {
		t.Fatalf("ai_note 缺跳过说明: %q", note)
	}
}

// ── 诚实降级路径 ─────────────────────────────────────────────

// TestCheckConsistencyDeepClientNil client 未初始化 → 仅规则层结果 + ai_available=false + 说明
func TestCheckConsistencyDeepClientNil(t *testing.T) {
	env := newConsistencyDeepEnv(t, nil, true)
	env.seedChapter(t, 1, "第1章正文：开局。")

	res, err := env.a.CheckConsistencyDeep(20)
	if err != nil {
		t.Fatalf("降级不应报错: %v", err)
	}
	if v, _ := res["ai_available"].(bool); v {
		t.Fatalf("client nil 时 ai_available 应为 false")
	}
	if note, _ := res["ai_note"].(string); !strings.Contains(note, "AI 客户端未初始化") {
		t.Fatalf("ai_note 缺降级说明: %q", note)
	}
	if v, _ := res["chapters_scanned"].(int); v != 0 {
		t.Fatalf("chapters_scanned 应为 0: %v", res["chapters_scanned"])
	}
	issues, _ := res["issues"].([]map[string]interface{})
	for _, iss := range issues {
		if s, _ := iss["source"].(string); s != "rule" {
			t.Fatalf("降级时只允许规则告警: %v", iss)
		}
	}
	if v, _ := res["total_issues"].(int); v != len(issues) {
		t.Fatalf("total_issues = %d, len(issues) = %d", v, len(issues))
	}
}

// TestCheckConsistencyDeepAIAllFail 模型全失败（HTTP 500，RetryJSON 重试耗尽）→ 诚实降级为规则层
func TestCheckConsistencyDeepAIAllFail(t *testing.T) {
	env := newConsistencyDeepEnv(t, []string{"内部错误"}, false)
	// 把正常桩换成恒 500 的服务：重新注册引擎指向坏地址
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server exploded", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	if err := env.a.engineMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine(500): %v", err)
	}

	env.seedChapter(t, 1, "第1章正文：开局。")
	env.seedChapter(t, 2, "第2章正文：推进。")

	res, err := env.a.CheckConsistencyDeep(20)
	if err != nil {
		t.Fatalf("全失败应降级而非报错: %v", err)
	}
	if v, _ := res["ai_available"].(bool); v {
		t.Fatalf("全失败时 ai_available 应为 false")
	}
	if note, _ := res["ai_note"].(string); !strings.Contains(note, "AI 逐章提取全部失败") {
		t.Fatalf("ai_note 缺全失败说明: %q", note)
	}
	if v, _ := res["chapters_scanned"].(int); v != 0 {
		t.Fatalf("chapters_scanned 应为 0: %v", res["chapters_scanned"])
	}
	issues, _ := res["issues"].([]map[string]interface{})
	for _, iss := range issues {
		if s, _ := iss["source"].(string); s != "rule" {
			t.Fatalf("降级时只允许规则告警: %v", iss)
		}
	}
}

// TestCheckConsistencyDeepNoProject 未打开项目 → 报错
func TestCheckConsistencyDeepNoProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	a := &App{core: &core{cfg: &config.Config{}, client: nil, engineMgr: modelengine.NewManager("", "")}}
	a.writingState = &writingState{core: a.core, app: a}
	if _, err := a.CheckConsistencyDeep(20); err == nil {
		t.Fatalf("未打开项目应报错")
	}
}

// TestDeepListChapters 枚举：主线 + 分支排序、忽略无关文件（含分支章节直写文件）
func TestDeepListChapters(t *testing.T) {
	env := newConsistencyDeepEnv(t, nil, true)
	env.seedChapter(t, 2, "第二章")
	env.seedChapter(t, 1, "第一章")
	branchDir := filepath.Join(env.pm.Dir, "chapters")
	if err := os.WriteFile(filepath.Join(branchDir, "003a.md"), []byte("分支a"), 0o644); err != nil {
		t.Fatalf("写分支章: %v", err)
	}
	if err := os.WriteFile(filepath.Join(branchDir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("写无关文件: %v", err)
	}

	got := deepListChapters(env.pm)
	if len(got) != 3 {
		t.Fatalf("应枚举 3 章: %+v", got)
	}
	want := []deepChapterRef{{1, ""}, {2, ""}, {3, "a"}}
	for i, ref := range got {
		if ref != want[i] {
			t.Fatalf("排序不符 [%d]: got %+v want %+v", i, ref, want[i])
		}
	}
	if p := deepPlace(3, "a"); p != "第3章分支a" {
		t.Fatalf("分支位置标签不符: %s", p)
	}
	if p := deepPlace(3, ""); p != "第3章" {
		t.Fatalf("主线位置标签不符: %s", p)
	}
}
