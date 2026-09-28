package app

// ── 刀1 线B 测试：写前硬闸（planPrecheck）与生成注入 ──────────────
//
// 覆盖（规格 docs/gaea-longform-novel-system-2026-09.md §7.1-3 / §7.3）：
//   - planPrecheck：缺计划阻断 / 齐备放行 / 缺失字段中文标签 / 大纲问题转换
//     （Code 前缀 outline_）/ 跨章重复点名 / 损坏文件如实报错 / 无项目报错；
//   - CreateChapter 硬闸：缺计划拒绝（错误文本含「先补章节计划」与缺失项、且
//     不建节点、不调用模型）；CreateChapterWithOverride(true) 放行；
//   - 注入：齐备计划时 prompt 含「本章计划」区段与「大纲要点」区段（真实模板 +
//     真实请求体捕获）；无 KeyPoints/Emotion 时不渲染空区段；
//   - 渲染纯函数：空计划/空节点返回 ""，超长按预算截断。

import (
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
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// ── 夹具 ──────────────────────────────────────────────────────

// newChapterGateProject 真实项目（计划/大纲读写走线A 的真实落盘实现）。
func newChapterGateProject(t *testing.T) *project.Manager {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "硬闸测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	t.Cleanup(func() { _ = pm.Close() })
	return pm
}

// newChapterGateBareApp 只有项目、无 AI client 的最小 App（planPrecheck 不需模型）。
func newChapterGateBareApp(t *testing.T) (*App, *project.Manager) {
	t.Helper()
	a := &App{core: &core{}}
	a.writingState = &writingState{core: a.core, app: a}
	pm := newChapterGateProject(t)
	a.setPM(pm)
	return a, pm
}

// newChapterGateLLMApp 带 mock LLM（httptest SSE，捕获请求体）与真实模板引擎的
// 生成 App：打桩手法对齐 create_chapter_prompt_test.go。模型回复固定为「第一章正文」
// （不含 ---CHAPTER_SUMMARY---，即"模型漏输出摘要"态）。
func newChapterGateLLMApp(t *testing.T) (*App, *project.Manager, chan []byte) {
	t.Helper()
	return newChapterGateLLMAppReply(t, "第一章正文")
}

// newChapterGateLLMAppReply 同 newChapterGateLLMApp，但模型回复内容可指定
// （用于验证摘要写回护栏：有/无 ---CHAPTER_SUMMARY--- 两种态）。
func newChapterGateLLMAppReply(t *testing.T, reply string) (*App, *project.Manager, chan []byte) {
	t.Helper()
	requests := make(chan []byte, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body, err := io.ReadAll(r.Body); err == nil {
			select {
			case requests <- body:
			default:
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n"+
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"+
			"data: [DONE]\n\n", strconv.Quote(reply))
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

	pm := newChapterGateProject(t)
	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.ctx = context.Background()
	a.setPM(pm)
	return a, pm, requests
}

// gatePlan 齐备计划（六项判据全满足）。
func gatePlan(chapterNum int) types.ChapterPlan {
	return types.ChapterPlan{
		SubIndex:       chapterNum,
		Title:          fmt.Sprintf("第%d章 夜探", chapterNum),
		PlotSummary:    "秦昭夜探听风阁，与沈砚决裂。",
		KeyEvents:      []string{"秦昭夜探风阁", "玄铁剑沉入井底"},
		CharacterFocus: []string{"秦昭", "沈砚"},
		EmotionalTone:  "紧张递进",
		NarrativeGoal:  "把暗线冲突推到明面",
		ConflictType:   "人与人",
		EndingType:     string(types.EndingSuspense),
		EstimatedWords: 3000,
	}
}

// gateSeedPlan 落盘一批计划（chapterNum → 计划）。
func gateSeedPlan(t *testing.T, pm *project.Manager, plans map[int]types.ChapterPlan) {
	t.Helper()
	f := &types.ChapterPlanFile{Version: 1, Plans: map[string]types.ChapterPlan{}}
	for num, p := range plans {
		f.Plans[strconv.Itoa(num)] = p
	}
	if err := pm.WriteChapterPlans(f); err != nil {
		t.Fatalf("写章节计划: %v", err)
	}
}

// gateSeedOutline 落盘大纲节点。
func gateSeedOutline(t *testing.T, pm *project.Manager, nodes ...types.OutlineNode) {
	t.Helper()
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}
}

// gateOutlineNode 一个「标题+摘要齐备」的大纲主线节点（KeyPoints/Emotion 按需给）。
func gateOutlineNode(num int, keyPoints []string, emotion string) types.OutlineNode {
	return types.OutlineNode{
		ID:         fmt.Sprintf("n_%d", num),
		Title:      fmt.Sprintf("第%d章 夜探", num),
		Summary:    "秦昭夜里潜入听风阁，撞见沈砚与黑衣人交易。",
		KeyPoints:  keyPoints,
		Emotion:    emotion,
		OrderIndex: num,
		Status:     types.OutlineWriting,
	}
}

// gateCapturePrompt 从捕获的请求体取 user/system prompt。
func gateCapturePrompt(t *testing.T, requests chan []byte, timeout time.Duration) (user, system string) {
	t.Helper()
	select {
	case body := <-requests:
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("解析请求体失败: %v", err)
		}
		for _, m := range req.Messages {
			switch m.Role {
			case "user":
				user = m.Content
			case "system":
				system = m.Content
			}
		}
		return user, system
	case <-time.After(timeout):
		t.Fatalf("未捕获到 AI 请求")
		return "", ""
	}
}

// ── planPrecheck ───────────────────────────────────────────────

// TestPlanPrecheck_MissingPlanBlocks 缺计划：S1 阻断、Allowed=false、给出中文缺失项，
// 且**不报错**（缺失是判定结果，不是故障）。
func TestPlanPrecheck_MissingPlanBlocks(t *testing.T) {
	a, _ := newChapterGateBareApp(t)
	rep, err := a.planPrecheck(1)
	if err != nil {
		t.Fatalf("缺计划不应报错: %v", err)
	}
	if rep.ChapterNum != 1 || rep.HasPlan {
		t.Fatalf("缺计划应 HasPlan=false: %+v", *rep)
	}
	if !rep.Blocking || rep.Allowed {
		t.Fatalf("缺计划应 Blocking=true / Allowed=false: %+v", *rep)
	}
	if len(rep.PlanProblems) != 1 || rep.PlanProblems[0].Severity != "S1" {
		t.Fatalf("缺计划应恰好一条 S1: %+v", rep.PlanProblems)
	}
	if !gateHasLabel(rep.Missing, "章节计划") {
		t.Fatalf("Missing 应含「章节计划」: %+v", rep.Missing)
	}
}

// TestPlanPrecheck_CompletePlanAllows 齐备计划 + 齐备大纲节点 → 放行，零问题。
func TestPlanPrecheck_CompletePlanAllows(t *testing.T) {
	a, pm := newChapterGateBareApp(t)
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
	gateSeedOutline(t, pm, gateOutlineNode(1, []string{"拾起玄铁剑"}, "沉重"))

	rep, err := a.planPrecheck(1)
	if err != nil {
		t.Fatalf("预检报错: %v", err)
	}
	if !rep.HasPlan || !rep.Allowed || rep.Blocking {
		t.Fatalf("齐备计划应放行: %+v", *rep)
	}
	if len(rep.PlanProblems) != 0 || len(rep.OutlineIssues) != 0 {
		t.Fatalf("齐备应零问题: 计划 %+v / 大纲 %+v", rep.PlanProblems, rep.OutlineIssues)
	}
	if len(rep.Missing) != 0 {
		t.Fatalf("齐备时 Missing 应为空: %+v", rep.Missing)
	}
}

// TestPlanPrecheck_MissingFieldLabels 缺字段：Missing 给出中文标签，字段缺项仍然阻断。
func TestPlanPrecheck_MissingFieldLabels(t *testing.T) {
	a, pm := newChapterGateBareApp(t)
	p := gatePlan(2)
	p.ConflictType = ""
	p.EndingType = ""
	p.EmotionalTone = ""
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{2: p})

	rep, err := a.planPrecheck(2)
	if err != nil {
		t.Fatalf("预检报错: %v", err)
	}
	if !rep.HasPlan || !rep.Blocking || rep.Allowed {
		t.Fatalf("缺冲突/结尾应阻断: %+v", *rep)
	}
	for _, want := range []string{"冲突类型", "结尾类型", "情绪基调"} {
		if !gateHasLabel(rep.Missing, want) {
			t.Fatalf("Missing 应含 %q: %+v", want, rep.Missing)
		}
	}
}

// TestPlanPrecheck_OutlineIssuesConverted 大纲节点契约转换为 PlanProblem：
// Code 带 outline_ 前缀、Severity 沿用、缺摘要（S2）参与阻断。
func TestPlanPrecheck_OutlineIssuesConverted(t *testing.T) {
	a, pm := newChapterGateBareApp(t)
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{3: gatePlan(3)})
	node := gateOutlineNode(3, nil, "")
	node.Summary = "" // 大纲摘要缺失 → S2 阻断
	gateSeedOutline(t, pm, node)

	rep, err := a.planPrecheck(3)
	if err != nil {
		t.Fatalf("预检报错: %v", err)
	}
	if len(rep.OutlineIssues) == 0 {
		t.Fatalf("应报大纲问题: %+v", *rep)
	}
	found := false
	for _, is := range rep.OutlineIssues {
		if !strings.HasPrefix(is.Code, "outline_") {
			t.Fatalf("大纲问题 Code 必须带 outline_ 前缀: %+v", is)
		}
		if is.Code == "outline_summary_empty" {
			found = true
			if is.Severity != "S2" || strings.TrimSpace(is.Message) == "" {
				t.Fatalf("Severity/Message 应沿用既有判据: %+v", is)
			}
		}
	}
	if !found {
		t.Fatalf("应报 outline_summary_empty: %+v", rep.OutlineIssues)
	}
	if !rep.Blocking || rep.Allowed {
		t.Fatalf("大纲 S2 应阻断: %+v", *rep)
	}
	if !gateHasLabel(rep.Missing, "章节摘要（谁·在哪·发生什么）") {
		t.Fatalf("Missing 应含章节摘要标签: %+v", rep.Missing)
	}
}

// TestPlanPrecheck_CrossChapterDuplicateBlocks 跨章事件重复（含全半角/空白差异）：
// 点名「与第 N 章重复」并阻断。
func TestPlanPrecheck_CrossChapterDuplicateBlocks(t *testing.T) {
	a, pm := newChapterGateBareApp(t)
	first := gatePlan(1)
	first.KeyEvents = []string{"雨夜夺符！", "拜师被拒"} // 全角标点写法
	second := gatePlan(2)
	second.KeyEvents = []string{" 雨夜 夺符!", "另一件事"} // 全角→半角 + 空白差异
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: first, 2: second})

	rep, err := a.planPrecheck(2)
	if err != nil {
		t.Fatalf("预检报错: %v", err)
	}
	if !rep.Blocking {
		t.Fatalf("跨章重复应阻断: %+v", *rep)
	}
	dup := ""
	for _, p := range rep.PlanProblems {
		if p.Code == types.PlanProblemEventDuplicated {
			dup = p.Message
			if p.Severity != "S1" {
				t.Fatalf("跨章重复应为 S1: %+v", p)
			}
		}
	}
	if !strings.Contains(dup, "与第1章重复：雨夜 夺符!") {
		t.Fatalf("应点名重复章号与事件原文: %q", dup)
	}
}

// TestPlanPrecheck_CorruptPlansFileErrors 损坏文件：如实报错，绝不覆盖、绝不假装没有计划。
func TestPlanPrecheck_CorruptPlansFileErrors(t *testing.T) {
	a, pm := newChapterGateBareApp(t)
	path := filepath.Join(pm.Dir, "chapters", "plans.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建 chapters 目录: %v", err)
	}
	corrupt := []byte("{这不是 JSON")
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatalf("写损坏计划文件: %v", err)
	}

	rep, err := a.planPrecheck(1)
	if err == nil {
		t.Fatalf("损坏计划文件应报错，实际返回 %+v", rep)
	}
	if !strings.Contains(err.Error(), "章节计划") {
		t.Fatalf("错误文本应说明是章节计划问题: %v", err)
	}
	after, rerr := os.ReadFile(path)
	if rerr != nil || string(after) != string(corrupt) {
		t.Fatalf("损坏文件不得被覆盖: %v / %q", rerr, string(after))
	}
}

// TestPlanPrecheck_ErrorsOnNoProjectOrBadChapterNum 无项目 / 章号无效 → error。
func TestPlanPrecheck_ErrorsOnNoProjectOrBadChapterNum(t *testing.T) {
	bare := &App{core: &core{}}
	bare.writingState = &writingState{core: bare.core, app: bare}
	if _, err := bare.planPrecheck(1); err == nil {
		t.Fatal("无项目应报错")
	}
	a, _ := newChapterGateBareApp(t)
	if _, err := a.planPrecheck(0); err == nil {
		t.Fatal("章号无效应报错")
	}
}

// gateHasLabel 缺失标签列表包含某标签。
func gateHasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

// ── CreateChapter 硬闸 ─────────────────────────────────────────

// TestCreateChapterGate_RefusesWithoutPlan 缺计划 → 拒绝生成：错误文本含「先补章节计划」
// 与缺失项；且不发生模型调用、不创建大纲节点（硬闸在副作用之前）。
func TestCreateChapterGate_RefusesWithoutPlan(t *testing.T) {
	a, pm, requests := newChapterGateLLMApp(t)

	_, err := a.CreateChapter("大陆设定", "", "雨夜夺符", 1, "", "", 1200, 0)
	if err == nil {
		t.Fatal("缺计划应拒绝生成")
	}
	msg := err.Error()
	for _, want := range []string{"先补章节计划", "第1章写前预检未通过", "章节计划", "本章尚未制定计划"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("错误文本应含 %q: %s", want, msg)
		}
	}
	select {
	case body := <-requests:
		t.Fatalf("硬闸拒绝时不得调用模型，却收到请求（%d 字节）", len(body))
	case <-time.After(200 * time.Millisecond):
	}
	of, rerr := pm.ReadOutlines()
	if rerr == nil && len(of.Nodes) != 0 {
		t.Fatalf("硬闸拒绝时不得创建章节节点: %+v", of.Nodes)
	}
}

// TestCreateChapterGate_OverrideAllows 显式覆盖（AllowOverride）放行：生成启动、
// 模型被调用、节点建立。
func TestCreateChapterGate_OverrideAllows(t *testing.T) {
	a, pm, requests := newChapterGateLLMApp(t)

	res, err := a.CreateChapterWithOverride("大陆设定", "", "雨夜夺符", 1, "", "", 1200, 0, true)
	if err != nil {
		t.Fatalf("显式覆盖应放行: %v", err)
	}
	if res["streaming"] != true || res["chapterNum"] != 1 {
		t.Fatalf("返回负载不对: %v", res)
	}
	gateCapturePrompt(t, requests, 5*time.Second)
	waitGensDone(t, a)
	of, rerr := pm.ReadOutlines()
	if rerr != nil || len(of.Nodes) == 0 {
		t.Fatalf("放行后应建立章节节点: %v / %+v", rerr, of)
	}
}

// TestCreateChapterGate_InjectsPlanAndOutlinePoints 齐备计划 → prompt 同时含
// 「本章计划」区段（六项字段）与「大纲要点」区段（KeyPoints/Emotion，此前零进入
// prompt），系统提示含新的 must 约束。
func TestCreateChapterGate_InjectsPlanAndOutlinePoints(t *testing.T) {
	a, pm, requests := newChapterGateLLMApp(t)
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
	gateSeedOutline(t, pm, gateOutlineNode(1, []string{"拾起玄铁剑", "黑衣人是沈砚"}, "沉重"))

	if _, err := a.CreateChapter("大陆设定", "", "雨夜夺符", 1, "", "", 1200, 0); err != nil {
		t.Fatalf("齐备计划应放行生成: %v", err)
	}
	user, system := gateCapturePrompt(t, requests, 5*time.Second)

	for _, want := range []string{
		"## 本章计划（本章的意图契约，必须完成）",
		"叙事目标：把暗线冲突推到明面",
		"关键事件（逐条必须落地，不得遗漏或跳过）：",
		"- 秦昭夜探风阁",
		"- 玄铁剑沉入井底",
		"冲突类型：人与人",
		"结尾类型：悬念",
		"情绪基调：紧张递进",
		"角色焦点：秦昭、沈砚",
		"## 本章大纲要点（细纲口径，逐条落地）",
		"必须落地的要点：",
		"- 拾起玄铁剑",
		"- 黑衣人是沈砚",
		"情感基调：沉重",
	} {
		if !strings.Contains(user, want) {
			t.Fatalf("user prompt 缺少 %q:\n%s", want, user)
		}
	}
	if !strings.Contains(system, "必须完成「本章计划」列出的全部关键事件") {
		t.Fatalf("system prompt 缺少模板新约束:\n%s", system)
	}
	waitGensDone(t, a)
}

// TestCreateChapterGate_NoKeyPointsNoEmptySection 大纲节点无 KeyPoints/Emotion →
// 不渲染空区段（prompt 不出现「本章大纲要点」标题）；计划区段照常出现。
func TestCreateChapterGate_NoKeyPointsNoEmptySection(t *testing.T) {
	a, pm, requests := newChapterGateLLMApp(t)
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
	gateSeedOutline(t, pm, gateOutlineNode(1, nil, ""))

	if _, err := a.CreateChapter("大陆设定", "", "雨夜夺符", 1, "", "", 1200, 0); err != nil {
		t.Fatalf("齐备计划应放行生成: %v", err)
	}
	user, _ := gateCapturePrompt(t, requests, 5*time.Second)
	if !strings.Contains(user, "## 本章计划（本章的意图契约，必须完成）") {
		t.Fatalf("计划区段应渲染:\n%s", user)
	}
	if strings.Contains(user, "本章大纲要点") || strings.Contains(user, "必须落地的要点") {
		t.Fatalf("无 KeyPoints/Emotion 时不得渲染空区段:\n%s", user)
	}
	waitGensDone(t, a)
}

// ── 生成收尾写回护栏（摘要非空才覆盖）─────────────────────────────

// TestCreateChapterGate_EmptySummaryKeepsNodeSummary 护栏：模型漏输出
// ---CHAPTER_SUMMARY--- 时，节点既有 Summary 必须保持不变（非空才覆盖），且随后
// 同章预检不因 outline_summary_empty 被阻断——否则硬闸会把显示问题放大成阻断。
func TestCreateChapterGate_EmptySummaryKeepsNodeSummary(t *testing.T) {
	// 模型回复不含 ---CHAPTER_SUMMARY---（summary 解析为空）
	a, pm, requests := newChapterGateLLMApp(t)
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
	node := gateOutlineNode(1, []string{"拾起玄铁剑"}, "沉重")
	node.Summary = "原有摘要：秦昭潜入听风阁，撞见沈砚交易。"
	gateSeedOutline(t, pm, node)

	if _, err := a.CreateChapterWithOverride("大陆设定", "", "雨夜夺符", 1, "", "", 1, 0, false); err != nil {
		t.Fatalf("齐备计划应放行生成: %v", err)
	}
	gateCapturePrompt(t, requests, 5*time.Second)
	waitGensDone(t, a)

	of, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	got := findOutlineNodeByNum(of.Nodes, 1)
	if got == nil {
		t.Fatalf("应保留第 1 章节点: %+v", of.Nodes)
	}
	if got.Summary != node.Summary {
		t.Fatalf("漏输出摘要时节点既有 Summary 不得被清空:\n got=%q\nwant=%q", got.Summary, node.Summary)
	}
	if got.Status != types.OutlineDone {
		t.Fatalf("正文已落盘，节点状态仍应推进到 done: %v", got.Status)
	}

	rep, err := a.planPrecheck(1)
	if err != nil {
		t.Fatalf("同章预检不应报错: %v", err)
	}
	for _, is := range rep.OutlineIssues {
		if is.Code == "outline_summary_empty" {
			t.Fatalf("既有摘要被保留后不应再报 outline_summary_empty: %+v", rep.OutlineIssues)
		}
	}
	if !rep.Allowed || rep.Blocking {
		t.Fatalf("摘要未被清空时同章不应被阻断: %+v", *rep)
	}
}

// TestCreateChapterGate_SummaryWrittenWhenPresent 反向回归：模型给出摘要时，
// 节点摘要照常被覆盖为新摘要（护栏只挡空值，不挡正常写回）。
func TestCreateChapterGate_SummaryWrittenWhenPresent(t *testing.T) {
	a, pm, requests := newChapterGateLLMAppReply(t, "第一章正文。\n---CHAPTER_SUMMARY---\n关键事件：雨夜夺符，内鬼现身。")
	gateSeedPlan(t, pm, map[int]types.ChapterPlan{1: gatePlan(1)})
	node := gateOutlineNode(1, []string{"拾起玄铁剑"}, "沉重")
	node.Summary = "旧摘要"
	gateSeedOutline(t, pm, node)

	if _, err := a.CreateChapterWithOverride("大陆设定", "", "雨夜夺符", 1, "", "", 1, 0, false); err != nil {
		t.Fatalf("齐备计划应放行生成: %v", err)
	}
	gateCapturePrompt(t, requests, 5*time.Second)
	waitGensDone(t, a)

	of, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	got := findOutlineNodeByNum(of.Nodes, 1)
	if got == nil {
		t.Fatalf("应保留第 1 章节点: %+v", of.Nodes)
	}
	if !strings.Contains(got.Summary, "雨夜夺符") {
		t.Fatalf("模型给出摘要时应写回节点: %q", got.Summary)
	}
	if got.Status != types.OutlineDone {
		t.Fatalf("节点状态应为 done: %v", got.Status)
	}
}

// ── 渲染纯函数 ─────────────────────────────────────────────────

// TestBuildChapterPlanSection_EmptyAndBudget 空计划/空字段 → ""；六项齐备 →
// 六行中文标签；超长按 ctxChapterPlanBudget 截断。
func TestBuildChapterPlanSection_EmptyAndBudget(t *testing.T) {
	if got := buildChapterPlanSection(nil); got != "" {
		t.Fatalf("nil 计划应返回空串: %q", got)
	}
	if got := buildChapterPlanSection(&types.ChapterPlan{}); got != "" {
		t.Fatalf("六项全空应返回空串（不渲染空区段）: %q", got)
	}

	p := gatePlan(4)
	got := buildChapterPlanSection(&p)
	for _, want := range []string{"叙事目标：", "关键事件", "冲突类型：", "结尾类型：", "情绪基调：", "角色焦点："} {
		if !strings.Contains(got, want) {
			t.Fatalf("计划区段缺少 %q:\n%s", want, got)
		}
	}

	long := gatePlan(5)
	long.NarrativeGoal = strings.Repeat("目", 4000)
	long.KeyEvents = []string{strings.Repeat("事", 4000)}
	long.ConflictType = strings.Repeat("冲", 4000)
	long.CharacterFocus = []string{strings.Repeat("人", 4000)}
	if n := len([]rune(buildChapterPlanSection(&long))); n > ctxChapterPlanBudget {
		t.Fatalf("计划区段超预算: got %d, want <= %d", n, ctxChapterPlanBudget)
	}
}

// TestBuildPlanOutlinePointsSection_Empty 无节点/无 KeyPoints 与 Emotion → ""；
// 仅有 Emotion 时只渲染基调行。
func TestBuildPlanOutlinePointsSection_Empty(t *testing.T) {
	if got := buildOutlinePointsSection(nil); got != "" {
		t.Fatalf("nil 节点应返回空串: %q", got)
	}
	empty := gateOutlineNode(1, nil, "")
	if got := buildOutlinePointsSection(&empty); got != "" {
		t.Fatalf("无 KeyPoints/Emotion 应返回空串: %q", got)
	}
	onlyEmotion := gateOutlineNode(1, nil, "紧张")
	got := buildOutlinePointsSection(&onlyEmotion)
	if !strings.Contains(got, "情感基调：紧张") || strings.Contains(got, "必须落地的要点") {
		t.Fatalf("仅有 Emotion 时应只渲染基调行:\n%s", got)
	}
	long := gateOutlineNode(1, []string{strings.Repeat("点", 4000)}, strings.Repeat("基", 4000))
	if n := len([]rune(buildOutlinePointsSection(&long))); n > ctxOutlinePointsBudget {
		t.Fatalf("大纲要点区段超预算: got %d, want <= %d", n, ctxOutlinePointsBudget)
	}
}
