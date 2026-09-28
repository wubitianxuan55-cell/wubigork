package app

// ── 章节计划闭环 端到端演示（规格 docs/gaea-longform-novel-system-2026-09.md §7.6-5「可复现证据」）──
//
// 本文件把「计划缺失 → 硬闸拒绝 → 生成草案 → 作者审批 → 带计划生成 → 偏差回写」
// 六步跑成**一条可复现的证据链**（3 章项目夹具 + 假 LLM），命令：
//
//	go test ./internal/app/ -run 'PlanLoopE2E' -count=1 -v
//
// 断言顺序（每一步都是下一步的前提，失败即证据链断裂）：
//  ① NovelChapterGatePrecheck(2) → Blocking=true 且 Missing 非空；
//  ② CreateChapterWithOverride(..., allowOverride=false) → 拒绝，错误含「先补章节计划」，
//     且**未调用模型**（硬闸在副作用之前）；
//  ③ NovelChapterPlanPropose(2) → 七字段齐备、key_events 2-4 条、**不落盘**；
//     Propose 输入段含本章大纲节点与故事主线（StoryThread）；
//  ④ NovelChapterPlanSave(2, planJSON) → 审批落盘，Get 往返一致；
//  ⑤ NovelChapterGatePrecheck(2) → Allowed=true；
//  ⑥ CreateChapterWithOverride(..., false) → 通过，生成 prompt 含「本章计划」区段，落章 chapters/002.md；
//  ⑦ 写 analysis-v2 夹具（结尾/情绪与计划不符 + 一条未达成事件 + 注入跨章重复）
//     → NovelChapterPlanDeviation(2) 输出四类判据 + summary + nextSuggestion + 落盘。
//
// 纪律：本文件只新增测试，不改实现；发现的实现缺陷在回报里提出，不在此处绕过。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// ── 假 LLM：同一服务同时服务「计划草案」与「章节正文」两条链 ──────

// planLoopStub 记录每次调用的类别（plan|chapter）与 user prompt，供断言取证。
type planLoopStub struct {
	mu      sync.Mutex
	kinds   []string
	prompts []string
}

func (s *planLoopStub) record(kind, userPrompt string) {
	s.mu.Lock()
	s.kinds = append(s.kinds, kind)
	s.prompts = append(s.prompts, userPrompt)
	s.mu.Unlock()
}

func (s *planLoopStub) snapshot() (kinds, prompts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.kinds...), append([]string(nil), s.prompts...)
}

// planLoopPlan 模型返回的七字段草案（关键事件与后续 analysis 夹具一一对应：
// 第 2 条「玄铁剑沉入井底」在后文分析里不出现 → 未达成）。
func planLoopPlan() types.ChapterPlan {
	return types.ChapterPlan{
		SubIndex:       2,
		Title:          "第二章 交易",
		PlotSummary:    "秦昭夜探听风阁，撞见沈砚与黑衣人交易，玄铁剑在争执中坠入井底。",
		KeyEvents:      []string{"秦昭夜探听风阁", "玄铁剑沉入井底"},
		CharacterFocus: []string{"秦昭", "沈砚"},
		EmotionalTone:  "紧张递进",
		NarrativeGoal:  "把暗线冲突推到明面",
		ConflictType:   "人与人",
		EndingType:     string(types.EndingSuspense),
	}
}

// planLoopChapterReply 章节正文回复（带 CHAPTER_SUMMARY 标记，与真实模板输出口径一致）。
const planLoopChapterReply = "秦昭贴着屋檐潜行，听见阁楼里有人在低声交换条件。" +
	"他探头去看，正撞见沈砚把一枚黑衣令牌推到桌上。" +
	"玄铁剑横在两人之间，争执中滑落井口，坠了下去。\n" +
	"---CHAPTER_SUMMARY---\n" +
	"关键事件：秦昭夜探听风阁；玄铁剑沉入井底。\n" +
	"人物变化：沈砚立场存疑。\n" +
	"伏笔/悬念：黑衣人的身份。"

// newPlanLoopApp 3 章夹具 + 假 LLM 的完整 App（模板引擎读磁盘 prompts/，与运行期同源）。
// 判别口径：chapter-plan 模板的 system 文本（「长篇小说的结构编辑」）出现即为计划草案
// 调用，否则视为章节正文调用——空槽位在 prompt 里不渲染（BuildUserPrompt 只渲染非空
// 输入段），所以判别必须落在**恒存在**的 system 文本上，不能靠某个输入段标签。
func newPlanLoopApp(t *testing.T) (*App, *project.Manager, *planLoopStub) {
	t.Helper()

	stub := &planLoopStub{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(body, &req)
		user, system := "", ""
		for _, m := range req.Messages {
			switch m.Role {
			case "user":
				user = m.Content
			case "system":
				system = m.Content
			}
		}

		kind, reply := "chapter", planLoopChapterReply
		if strings.Contains(system, "长篇小说的结构编辑") {
			kind = "plan"
			b, _ := json.Marshal(planLoopPlan())
			reply = string(b)
		}
		stub.record(kind, user)

		payload, _ := json.Marshal(map[string]interface{}{
			"choices": []map[string]interface{}{{
				"delta":         map[string]interface{}{"content": reply},
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

	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "计划闭环演示小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	t.Cleanup(func() { _ = pm.Close() })

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.ctx = context.Background()
	a.setPM(pm)
	return a, pm, stub
}

// planLoopSeedThreeChapters 3 章大纲夹具（标题/摘要/要点/情感齐备——否则预检会因
// 大纲契约缺项阻断，掩盖本演示要证明的「缺计划」这一条原因）。
func planLoopSeedThreeChapters(t *testing.T, pm *project.Manager) {
	t.Helper()
	nodes := []types.OutlineNode{
		{
			ID: "n_1", Title: "第一章 夜行", OrderIndex: 1, Status: types.OutlinePlanned,
			Summary:   "秦昭潜入听风阁，救下守夜的小厮。",
			KeyPoints: []string{"潜入听风阁", "救下小厮"}, Emotion: "压抑",
		},
		{
			ID: "n_2", Title: "第二章 交易", OrderIndex: 2, Status: types.OutlinePlanned,
			Summary:   "秦昭撞见沈砚与黑衣人交易，玄铁剑现世。",
			KeyPoints: []string{"撞见交易", "玄铁剑现世"}, Emotion: "紧张",
		},
		{
			ID: "n_3", Title: "第三章 井底", OrderIndex: 3, Status: types.OutlinePlanned,
			Summary:   "玄铁剑坠井，沈砚的沉默让立场成谜。",
			KeyPoints: []string{"玄铁剑坠井", "沈砚沉默"}, Emotion: "肃杀",
		},
	}
	of := &types.OutlineFile{
		StoryThread: "主线：秦昭查清灭门旧案，玄铁剑是唯一物证",
		Nodes:       nodes,
	}
	if err := pm.WriteOutlines(of); err != nil {
		t.Fatalf("写大纲: %v", err)
	}
}

// planLoopAnalysisFixture 写后分析夹具：结尾钩子=情感（计划悬念）、情绪=欢快
// （计划紧张递进）、情节推进点只命中第 1 条关键事件（第 2 条未达成）。
func planLoopAnalysisFixture() types.ChapterAnalysisResult {
	return types.ChapterAnalysisResult{
		ChapterNum:  2,
		ChapterFile: "002.md",
		Result: types.AnalysisResultV2{
			Summary: "本章以沈砚的沉默收束，井底的水声压过一切。",
			PlotPoints: []types.PlotPoint{{
				Content: "秦昭夜探听风阁，撞见沈砚与黑衣人交易。",
				Type:    "revelation", Importance: 0.8, Impact: "沈砚立场存疑",
			}},
			Hooks:        []types.Hook{{Type: "情感", Content: "沈砚的沉默", Position: "结尾", Strength: 7}},
			EmotionalArc: types.EmotionalArc{PrimaryEmotion: "欢快", Intensity: 4, Curve: "先抑后扬"},
			Pacing:       "fast",
		},
	}
}

// TestPlanLoopE2E_PlanMissingToDeviation 六步证据链（详见文件头注释）。
func TestPlanLoopE2E_PlanMissingToDeviation(t *testing.T) {
	a, pm, stub := newPlanLoopApp(t)
	planLoopSeedThreeChapters(t, pm)
	t.Logf("夹具就绪：3 章大纲（StoryThread 非空）+ chapters/plans.json 不存在（= 未制定任何计划）")

	// ① 计划缺失：写前预检阻断
	rep, err := a.NovelChapterGatePrecheck(2)
	if err != nil {
		t.Fatalf("预检不应报错（缺计划是判定结果，不是故障）: %v", err)
	}
	if rep == nil || !rep.Blocking || rep.Allowed || rep.HasPlan {
		t.Fatalf("缺计划应 Blocking=true / Allowed=false / HasPlan=false: %+v", rep)
	}
	if len(rep.Missing) == 0 {
		t.Fatalf("缺计划应给出中文缺失项: %+v", *rep)
	}
	t.Logf("① Precheck(2) → blocking=%v allowed=%v hasPlan=%v missing=%v",
		rep.Blocking, rep.Allowed, rep.HasPlan, rep.Missing)

	// ② 硬闸拒绝生成（且不调用模型）
	_, err = a.CreateChapterWithOverride("九州大陆，灵气复苏。", "", "秦昭夜探听风阁", 2, "", "", 20, 0, false)
	if err == nil {
		t.Fatal("缺计划必须拒绝生成")
	}
	for _, want := range []string{"先补章节计划", "第2章写前预检未通过", "章节计划"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("拒绝错误应含 %q: %s", want, err.Error())
		}
	}
	if kinds, _ := stub.snapshot(); len(kinds) != 0 {
		t.Fatalf("硬闸拒绝时不得调用模型，实际调用: %v", kinds)
	}
	t.Logf("② CreateChapterWithOverride(allowOverride=false) 被拒 → %v；模型调用 0 次", err)

	// ③ AI 草案（不落盘）
	proposal, err := a.NovelChapterPlanPropose(2)
	if err != nil {
		t.Fatalf("草案生成失败: %v", err)
	}
	if proposal.PlotSummary == "" || len(proposal.CharacterFocus) == 0 ||
		proposal.EmotionalTone == "" || proposal.NarrativeGoal == "" ||
		proposal.ConflictType == "" || proposal.EndingType == "" {
		t.Fatalf("七字段应齐备: %+v", *proposal)
	}
	if len(proposal.KeyEvents) < types.PlanMinKeyEvents || len(proposal.KeyEvents) > 4 {
		t.Fatalf("key_events 应为 2-4 条: %v", proposal.KeyEvents)
	}
	if got, gerr := a.NovelChapterPlanGet(2); gerr != nil || got != nil {
		t.Fatalf("草案不得落盘（审批制）: plan=%+v err=%v", got, gerr)
	}
	kinds, prompts := stub.snapshot()
	if len(kinds) != 1 || kinds[0] != "plan" {
		t.Fatalf("第 1 次调用应为计划草案: %v", kinds)
	}
	for _, want := range []string{"第二章 交易", "秦昭查清灭门旧案"} {
		if !strings.Contains(prompts[0], want) {
			t.Fatalf("草案输入段应含 %q（本章节点 / 故事主线）:\n%s", want, prompts[0])
		}
	}
	t.Logf("③ Propose(2) → key_events=%v ending_type=%s emotional_tone=%s（未落盘）",
		proposal.KeyEvents, proposal.EndingType, proposal.EmotionalTone)

	// ④ 作者审批落盘
	planJSONBytes, merr := json.Marshal(proposal)
	if merr != nil {
		t.Fatalf("序列化草案: %v", merr)
	}
	if err := a.NovelChapterPlanSave(2, string(planJSONBytes)); err != nil {
		t.Fatalf("审批落盘失败: %v", err)
	}
	stored, err := a.NovelChapterPlanGet(2)
	if err != nil || stored == nil {
		t.Fatalf("落盘后应可读回: plan=%+v err=%v", stored, err)
	}
	if len(stored.KeyEvents) != len(proposal.KeyEvents) || stored.EndingType != proposal.EndingType {
		t.Fatalf("往返不一致:\n got=%+v\nwant=%+v", *stored, *proposal)
	}
	t.Logf("④ Save(2) 审批落盘 → Get 往返一致（%d 条关键事件，含 plans.json 落盘）", len(stored.KeyEvents))

	// ⑤ 预检放行
	rep, err = a.NovelChapterGatePrecheck(2)
	if err != nil {
		t.Fatalf("预检不应报错: %v", err)
	}
	if !rep.Allowed || rep.Blocking || !rep.HasPlan {
		t.Fatalf("计划齐备应放行: %+v", *rep)
	}
	t.Logf("⑤ Precheck(2) → allowed=%v blocking=%v hasPlan=%v", rep.Allowed, rep.Blocking, rep.HasPlan)

	// ⑥ 带计划生成 → 落章
	res, err := a.CreateChapterWithOverride("九州大陆，灵气复苏。", "", "秦昭夜探听风阁", 2, "", "", 20, 0, false)
	if err != nil {
		t.Fatalf("计划齐备应放行生成: %v", err)
	}
	if res["streaming"] != true || res["chapterNum"] != 2 {
		t.Fatalf("生成返回负载不对: %v", res)
	}
	waitGensDone(t, a)
	body, err := pm.ReadChapter(2)
	if err != nil {
		t.Fatalf("读章节: %v", err)
	}
	if !strings.Contains(body, "玄铁剑") {
		t.Fatalf("章节正文应落盘: %q", body)
	}
	kinds, prompts = stub.snapshot()
	if len(kinds) != 2 || kinds[1] != "chapter" {
		t.Fatalf("第 2 次调用应为章节正文: %v", kinds)
	}
	if !strings.Contains(prompts[1], "本章计划（本章的意图契约，必须完成）") {
		t.Fatalf("生成 prompt 应注入本章计划区段:\n%s", prompts[1])
	}
	t.Logf("⑥ CreateChapterWithOverride(allowOverride=false) 通过 → 落章 chapters/002.md（%d 字），生成 prompt 含「本章计划」区段",
		len([]rune(body)))

	// ⑦ 写后偏差回写：analysis-v2 夹具（结尾/情绪不符 + 第 2 条事件未达成）
	if err := pm.UpsertAnalysisV2(planLoopAnalysisFixture()); err != nil {
		t.Fatalf("写 analysis-v2 夹具: %v", err)
	}
	// 跨章重复事件的取证夹具：把第 1 章计划直接写进 plans.json（绕开审批闸），
	// 模拟「存量/手改 plans.json」——Save 侧对同一情形会拒绝（见 TestNovelChapterPlanSaveMatrix
	// 的「跨章重复被点名拒绝」）。这样偏差报告的四类判据可以在同一夹具上全部出现。
	pf, err := pm.ReadChapterPlans()
	if err != nil {
		t.Fatalf("读计划表: %v", err)
	}
	pf.Plans[strconv.Itoa(1)] = types.ChapterPlan{
		SubIndex: 1, Title: "第一章 夜行",
		PlotSummary:    "秦昭潜入听风阁，救下守夜的小厮。",
		KeyEvents:      []string{proposal.KeyEvents[0], "沈砚旧誓再提"},
		CharacterFocus: []string{"秦昭"}, EmotionalTone: "压抑",
		NarrativeGoal: "建立旧案悬念", ConflictType: "人与环境",
		EndingType: string(types.EndingSuspense),
	}
	if err := pm.WriteChapterPlans(pf); err != nil {
		t.Fatalf("注入第 1 章计划: %v", err)
	}

	dev, err := a.NovelChapterPlanDeviation(2)
	if err != nil {
		t.Fatalf("偏差回写失败: %v", err)
	}
	if dev == nil || !dev.HasPlan || !dev.Analyzed {
		t.Fatalf("报告应标记 HasPlan/Analyzed: %+v", dev)
	}
	// ① 未达成事件（第 2 条计划事件在 analysis 载荷中不出现）
	if len(dev.MissingEvents) != 1 || dev.MissingEvents[0] != proposal.KeyEvents[1] {
		t.Fatalf("应只有第 2 条事件未达成: %v（计划 %v）", dev.MissingEvents, proposal.KeyEvents)
	}
	// ② 结尾类型不符（计划悬念 / 实际情感收尾）
	if !dev.EndingMismatch || dev.PlannedEnding != string(types.EndingSuspense) ||
		dev.ActualEnding != string(types.EndingEmotional) {
		t.Fatalf("结尾判据不对: mismatch=%v planned=%q actual=%q", dev.EndingMismatch, dev.PlannedEnding, dev.ActualEnding)
	}
	// ③ 情绪漂移（计划紧张递进 / 实际欢快）
	if !dev.EmotionDrift || dev.PlannedEmotion != "紧张递进" || dev.ActualEmotion != "欢快" {
		t.Fatalf("情绪判据不对: drift=%v planned=%q actual=%q", dev.EmotionDrift, dev.PlannedEmotion, dev.ActualEmotion)
	}
	// ④ 跨章重复（与注入的第 1 章计划重复）
	if len(dev.DuplicateEvents) == 0 || !strings.Contains(strings.Join(dev.DuplicateEvents, "|"), proposal.KeyEvents[0]) {
		t.Fatalf("重复事件判据不对: %v", dev.DuplicateEvents)
	}
	for _, want := range []string{"1/2 关键事件达成", "结尾类型不符（计划：悬念 / 实际：情感收尾）", "情绪漂移", "重复"} {
		if !strings.Contains(dev.Summary, want) {
			t.Fatalf("summary 应含 %q: %s", want, dev.Summary)
		}
	}
	if !strings.Contains(dev.NextSuggestion, "下一章计划建议") || !strings.Contains(dev.NextSuggestion, proposal.KeyEvents[1]) {
		t.Fatalf("nextSuggestion 应给下一章计划建议并点名未达成事件:\n%s", dev.NextSuggestion)
	}
	if saved, ok, rerr := pm.ReadPlanDeviation(2); rerr != nil || !ok || saved == nil || saved.Summary != dev.Summary {
		t.Fatalf("偏差应落盘且与返回一致: ok=%v saved=%+v err=%v", ok, saved, rerr)
	}
	t.Logf("⑦ Deviation(2) → missingEvents=%v | endingMismatch=%v（计划 %s / 实际 %s）| emotionDrift=%v（计划 %s / 实际 %s）| duplicateEvents=%v",
		dev.MissingEvents, dev.EndingMismatch, dev.PlannedEnding, dev.ActualEnding,
		dev.EmotionDrift, dev.PlannedEmotion, dev.ActualEmotion, dev.DuplicateEvents)
	t.Logf("   summary = %s", dev.Summary)
	t.Logf("   nextSuggestion:\n%s", dev.NextSuggestion)
	t.Logf("证据链完整：缺计划被拦（0 次模型调用）→ 草案 → 审批 → 生成（1 次正文调用）→ 偏差四判据 + 落盘")
}
