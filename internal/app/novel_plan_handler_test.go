package app

// ── 章节计划闭环 线C 测试（规格 docs/gaea-longform-novel-system-2026-09.md §7.2/§7.4）──
//
// 覆盖：
//   - Save 拒绝矩阵（缺目标 / 关键事件 <2 / 跨章重复点名 / 非法 JSON）+ 合法落盘
//     往返 + 整表读-改-写不伤其它章；
//   - Get 缺失=正常态（nil,nil）与章号守卫；
//   - Propose：假 LLM（httptest SSE，打桩手法对齐 consistency_deep_handler_test.go）
//     断言七字段解析、key_events 数量约束、模板槽位含已有 key_events 列表，
//     以及「解析失败 / 本地校验不过 / 无引擎 / 无大纲节点 → 如实报错，不发半成品」；
//   - Deviation 四类判据各一例 + 无计划 / 未分析两个正常态 + 落盘与落盘失败尽力语义；
//   - 归一化与命中阈值（纯函数）、planOthers 确定性；
//   - Precheck 委托（缺计划 → Blocking）。

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
	"sync/atomic"
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

// newPlanTestProject 在临时目录创建一个真实小说项目（计划读写走线A 的真实实现）。
func newPlanTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "计划测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	return pm
}

// newPlanTestApp 只有项目、无 AI client / 无模板引擎的最小 App：
// Get/Save/Deviation/Precheck 全链都不需要模型。
func newPlanTestApp(t *testing.T) (*App, *project.Manager) {
	t.Helper()
	a := &App{core: &core{}}
	a.writingState = &writingState{core: a.core, app: a}
	pm := newPlanTestProject(t)
	a.setPM(pm)
	return a, pm
}

// planValidFixture 合法计划（六项齐备 + 2 条关键事件）。
func planValidFixture(chapterNum int) types.ChapterPlan {
	return types.ChapterPlan{
		SubIndex:       chapterNum,
		Title:          fmt.Sprintf("第%d章", chapterNum),
		PlotSummary:    "秦昭夜探听风阁，与沈砚决裂，玄铁剑沉入井底。",
		KeyEvents:      []string{"秦昭夜探风阁", "玄铁剑沉入井底"},
		CharacterFocus: []string{"秦昭"},
		EmotionalTone:  "紧张递进",
		NarrativeGoal:  "把暗线冲突推到明面",
		ConflictType:   "人与人",
		EndingType:     string(types.EndingSuspense),
	}
}

// planMustJSON 计划 → 绑定面 planJSON。
func planMustJSON(t *testing.T, p types.ChapterPlan) string {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal 计划: %v", err)
	}
	return string(b)
}

// planSaveMust 审批落盘一份合法计划（夹具用）。
func planSaveMust(t *testing.T, a *App, chapterNum int, p types.ChapterPlan) {
	t.Helper()
	if err := a.NovelChapterPlanSave(chapterNum, planMustJSON(t, p)); err != nil {
		t.Fatalf("落盘第%d章计划: %v", chapterNum, err)
	}
}

// planRawSeed 直接写计划表（绕过审批闸）：模拟「计划表已由旧版本/AI 批量写入」
// 的既有状态——偏差回写必须能对没有经过 Save 闸的计划做判定。
func planRawSeed(t *testing.T, pm *project.Manager, plans map[int]types.ChapterPlan) {
	t.Helper()
	pf := &types.ChapterPlanFile{Version: 1, Plans: map[string]types.ChapterPlan{}}
	for num, p := range plans {
		pf.Plans[strconv.Itoa(num)] = p
	}
	if err := pm.WriteChapterPlans(pf); err != nil {
		t.Fatalf("写计划表: %v", err)
	}
}

// planWantErrContains 断言报错且点名给定字串。
func planWantErrContains(t *testing.T, err error, subs ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("期望报错（应含 %v），得到 nil", subs)
	}
	for _, s := range subs {
		if !strings.Contains(err.Error(), s) {
			t.Fatalf("错误信息 %q 应包含 %q", err.Error(), s)
		}
	}
}

// planAnalysisFixture 构造 analysis-v2 载荷（hookType=="" 表示无结尾钩子=字段缺省）。
func planAnalysisFixture(chapterNum int, points []string, summary, hookType, emotion string) types.ChapterAnalysisResult {
	res := types.AnalysisResultV2{
		Summary:      summary,
		EmotionalArc: types.EmotionalArc{PrimaryEmotion: emotion},
	}
	for _, p := range points {
		res.PlotPoints = append(res.PlotPoints, types.PlotPoint{Content: p})
	}
	if hookType != "" {
		res.Hooks = []types.Hook{{Type: hookType, Content: "结尾钩子", Position: "结尾"}}
	}
	return types.ChapterAnalysisResult{ChapterNum: chapterNum, Result: res}
}

// ── Save 拒绝矩阵 + 往返 ──────────────────────────────────────

// TestNovelChapterPlanSaveMatrix Save 拒绝矩阵 + 合法落盘往返 + 整表读-改-写。
func TestNovelChapterPlanSaveMatrix(t *testing.T) {
	a, pm := newPlanTestApp(t)

	// 先落一章合法计划：验证后续写入不会伤到其它章（整表读-改-写）。
	seed := planValidFixture(1)
	planSaveMust(t, a, 1, seed)

	t.Run("缺目标被拒", func(t *testing.T) {
		p := planValidFixture(3)
		p.NarrativeGoal = ""
		err := a.NovelChapterPlanSave(3, planMustJSON(t, p))
		planWantErrContains(t, err, types.PlanProblemMissingGoal)
		if got, gerr := a.NovelChapterPlanGet(3); gerr != nil || got != nil {
			t.Fatalf("被拒的计划不得落盘: plan=%+v err=%v", got, gerr)
		}
	})

	t.Run("关键事件不足被拒", func(t *testing.T) {
		p := planValidFixture(3)
		p.KeyEvents = []string{"只有一条事件"}
		err := a.NovelChapterPlanSave(3, planMustJSON(t, p))
		planWantErrContains(t, err, types.PlanProblemMissingKeyEvents)
		if got, _ := a.NovelChapterPlanGet(3); got != nil {
			t.Fatalf("被拒的计划不得落盘: %+v", got)
		}
	})

	t.Run("跨章重复被点名拒绝", func(t *testing.T) {
		p := planValidFixture(3)
		// 与第 1 章种子计划的关键事件「秦昭夜探风阁」重复
		p.KeyEvents = []string{seed.KeyEvents[0], "沈砚交出听风阁令牌"}
		err := a.NovelChapterPlanSave(3, planMustJSON(t, p))
		planWantErrContains(t, err, types.PlanProblemEventDuplicated, seed.KeyEvents[0])
		if got, _ := a.NovelChapterPlanGet(3); got != nil {
			t.Fatalf("被拒的计划不得落盘: %+v", got)
		}
	})

	t.Run("非法JSON被拒", func(t *testing.T) {
		err := a.NovelChapterPlanSave(3, "{不是 JSON")
		planWantErrContains(t, err, "解析失败")
	})

	t.Run("空内容被拒", func(t *testing.T) {
		err := a.NovelChapterPlanSave(3, "   ")
		planWantErrContains(t, err, "计划内容为空")
	})

	t.Run("合法计划落盘往返", func(t *testing.T) {
		p := planValidFixture(3)
		p.PlotSummary = "秦昭在听风阁顶摊牌，玄铁剑沉井。"
		p.KeyEvents = []string{"秦昭听风阁摊牌", "沈砚交出听风阁令牌"}
		if err := a.NovelChapterPlanSave(3, planMustJSON(t, p)); err != nil {
			t.Fatalf("合法计划应落盘: %v", err)
		}
		got, err := a.NovelChapterPlanGet(3)
		if err != nil || got == nil {
			t.Fatalf("往返读取失败: plan=%+v err=%v", got, err)
		}
		if got.PlotSummary != p.PlotSummary || got.EndingType != p.EndingType ||
			got.EmotionalTone != p.EmotionalTone || len(got.KeyEvents) != len(p.KeyEvents) {
			t.Fatalf("往返不一致:\n got=%+v\nwant=%+v", *got, p)
		}
		if got.SubIndex != 3 {
			t.Fatalf("SubIndex 应补齐为章号: %d", got.SubIndex)
		}

		// 其它章原样保留（整表读-改-写）
		other, err := a.NovelChapterPlanGet(1)
		if err != nil || other == nil {
			t.Fatalf("第1章计划被写坏: %+v err=%v", other, err)
		}
		if other.PlotSummary != seed.PlotSummary || len(other.KeyEvents) != len(seed.KeyEvents) {
			t.Fatalf("第1章计划应保持不变:\n got=%+v\nwant=%+v", *other, seed)
		}

		// 落盘可见性：plans.json 由线A 落盘，重新读项目同名文件仍可解析
		plans, err := pm.ReadChapterPlans()
		if err != nil || plans == nil {
			t.Fatalf("读计划表: %v", err)
		}
		if _, ok := plans.Plans["3"]; !ok {
			t.Fatalf("plans.json 应有 3 号键: %+v", plans.Plans)
		}
	})
}

// TestNovelChapterPlanGetMissingAndGuards 缺失=正常态；非法章号=报错。
func TestNovelChapterPlanGetMissingAndGuards(t *testing.T) {
	a, _ := newPlanTestApp(t)

	got, err := a.NovelChapterPlanGet(7)
	if err != nil {
		t.Fatalf("未制定计划应返回 nil,nil（正常态），得到 err=%v", err)
	}
	if got != nil {
		t.Fatalf("未制定计划应返回 nil，得到 %+v", got)
	}

	for _, num := range []int{0, -1} {
		if _, err := a.NovelChapterPlanGet(num); err == nil {
			t.Fatalf("章号 %d 应报错", num)
		}
		if err := a.NovelChapterPlanSave(num, planMustJSON(t, planValidFixture(1))); err == nil {
			t.Fatalf("章号 %d 保存应报错", num)
		}
		if _, err := a.NovelChapterPlanDeviation(num); err == nil {
			t.Fatalf("章号 %d 偏差应报错", num)
		}
	}
}

// ── Propose（假 LLM）──────────────────────────────────────────

// planStubEnv Propose 打桩环境：SSE 假 LLM + 记录最后一次 user prompt。
type planStubEnv struct {
	a     *App
	pm    *project.Manager
	calls *int32

	mu   sync.Mutex
	user string
}

func (e *planStubEnv) setUserPrompt(s string) {
	e.mu.Lock()
	e.user = s
	e.mu.Unlock()
}

func (e *planStubEnv) userPrompt() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.user
}

// newPlanStubEnv replies 按调用次序回放（超出后重复最后一条）。
func newPlanStubEnv(t *testing.T, replies ...string) *planStubEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	env := &planStubEnv{calls: new(int32)}

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := int(atomic.AddInt32(env.calls, 1)) - 1
		if idx >= len(replies) {
			idx = len(replies) - 1
		}
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &req)
		for _, m := range req.Messages {
			if m.Role == "user" {
				env.setUserPrompt(m.Content)
			}
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
	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	// 模板引擎读磁盘 prompts/（含新增的 chapter-plan.json），与运行期同源。
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	a.ctx = ctx
	pm := newPlanTestProject(t)
	a.setPM(pm)

	env.a = a
	env.pm = pm
	return env
}

// planSeedProposeInputs 铺 Propose 的输入：大纲（含 StoryThread）+ 他已有关键事件 + 本章分析。
func planSeedProposeInputs(t *testing.T, env *planStubEnv) {
	t.Helper()

	node := types.OutlineNode{
		ID: "n_3", Title: "第三章 摊牌", OrderIndex: 3, Status: types.OutlinePlanned,
		Summary: "秦昭在听风阁与沈砚摊牌，玄铁剑易主。",
		KeyPoints: []string{
			"秦昭亮出听风阁令牌",
			"沈砚承认旧案",
		},
		Emotion:    "紧张递进",
		Characters: []string{"秦昭", "沈砚"},
	}
	of := &types.OutlineFile{
		StoryThread: "主线：秦昭查清灭门旧案，玄铁剑是唯一物证",
		Nodes: []types.OutlineNode{
			{ID: "n_1", Title: "第一章 夜行", OrderIndex: 1, Summary: "秦昭潜入听风阁。"},
			{ID: "n_2", Title: "第二章 旧誓", OrderIndex: 2, Summary: "沈砚断指明誓。"},
			node,
		},
	}
	if err := env.pm.WriteOutlines(of); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	// 其它章已有关键事件（existing_key_events 槽位来源）
	planSaveMust(t, env.a, 1, types.ChapterPlan{
		SubIndex: 1, Title: "第一章 夜行", PlotSummary: "秦昭潜入听风阁。",
		KeyEvents:      []string{"秦昭潜入听风阁", "沈砚断指明誓"},
		CharacterFocus: []string{"秦昭"}, EmotionalTone: "压抑",
		NarrativeGoal: "建立旧案悬念", ConflictType: "人与环境", EndingType: string(types.EndingSuspense),
	})

	// 本章既有分析载荷（revision 场景：analysis-v2 摘要进 chapter_analysis 槽位）
	if err := env.pm.UpsertAnalysisV2(planAnalysisFixture(3,
		[]string{"秦昭亮出令牌，沈砚沉默。"}, "本章末以沈砚的沉默收束。", "悬念", "紧张")); err != nil {
		t.Fatalf("写分析载荷: %v", err)
	}
}

// planProposeReply 七字段假回复（3 条关键事件，与已有事件不重复）。
func planProposeReply(chapterNum int) string {
	plan := types.ChapterPlan{
		SubIndex:       chapterNum,
		Title:          "第三章 摊牌",
		PlotSummary:    "秦昭亮出听风阁令牌，沈砚承认旧案，两人约定三日后对质。",
		KeyEvents:      []string{"秦昭亮出听风阁令牌", "沈砚承认旧案", "两人约定三日后对质"},
		CharacterFocus: []string{"秦昭", "沈砚"},
		EmotionalTone:  "紧张递进",
		NarrativeGoal:  "把暗线冲突推到明面",
		ConflictType:   "人与人",
		EndingType:     string(types.EndingSuspense),
	}
	b, _ := json.Marshal(plan)
	return string(b)
}

// TestNovelChapterPlanPropose_ParsesSevenFieldsAndSlots Propose 正常链：
// 七字段解析、key_events 数量约束、模板槽位装配（已有 key_events / 主线 / 分析摘要）。
func TestNovelChapterPlanPropose_ParsesSevenFieldsAndSlots(t *testing.T) {
	env := newPlanStubEnv(t, planProposeReply(3))
	planSeedProposeInputs(t, env)

	plan, err := env.a.NovelChapterPlanPropose(3, "")
	if err != nil {
		t.Fatalf("Propose 失败: %v", err)
	}
	if plan.SubIndex != 3 {
		t.Fatalf("SubIndex 应为章号: %d", plan.SubIndex)
	}

	// 七字段解析
	if !strings.Contains(plan.PlotSummary, "听风阁令牌") {
		t.Fatalf("PlotSummary 未解析: %q", plan.PlotSummary)
	}
	if len(plan.KeyEvents) < types.PlanMinKeyEvents || len(plan.KeyEvents) > 4 {
		t.Fatalf("key_events 应为 2-4 条: %v", plan.KeyEvents)
	}
	if len(plan.CharacterFocus) < 1 {
		t.Fatalf("character_focus 至少 1 人: %v", plan.CharacterFocus)
	}
	if plan.EmotionalTone != "紧张递进" || plan.NarrativeGoal == "" ||
		plan.ConflictType != "人与人" || plan.EndingType != string(types.EndingSuspense) {
		t.Fatalf("七字段不完整: %+v", *plan)
	}

	// 模板槽位装配：已有 key_events 列表作为输入段传入（规格 §7.2 Propose）
	promptText := env.userPrompt()
	for _, want := range []string{
		"秦昭潜入听风阁",     // 其它章已有关键事件（去重输入）
		"沈砚断指明誓",      // 同上
		"秦昭查清灭门旧案",    // OutlineFile.StoryThread（线A 修复写入路径后的主线）
		"第三章 摊牌",      // 本章大纲节点
		"听风阁令牌",       // 章节分析载荷摘要（chapter_analysis 槽位）
		"已有章节的关键事件",   // 模板槽位标签（证明走的是 chapter-plan 模板）
		"本章阶段位置与计划任务", // v4.452.0：位置段（10 章一阶段起承转合）
		"阶段之「起」",      // 第3章=阶段内第3章 → 起（布局段）
		"沈砚承认旧案",      // 大纲 KeyPoints
	} {
		if !strings.Contains(promptText, want) {
			t.Fatalf("Propose user prompt 应包含 %q，实际:\n%s", want, promptText)
		}
	}
	if n := atomic.LoadInt32(env.calls); n != 1 {
		t.Fatalf("应恰好调用一次模型: %d", n)
	}

	// 草案不落盘（审批制）
	if got, err := env.a.NovelChapterPlanGet(3); err != nil || got != nil {
		t.Fatalf("Propose 不得落盘: plan=%+v err=%v", got, err)
	}
}

// TestNovelChapterPlanPropose_RejectsBadReply 解析失败 / 本地校验不过 / 无引擎 /
// 缺大纲节点 → 如实报错，绝不返回半成品或规则兜底计划。
func TestNovelChapterPlanPropose_RejectsBadReply(t *testing.T) {
	t.Run("解析失败附原始返回前缀", func(t *testing.T) {
		env := newPlanStubEnv(t, "不是 JSON，模型跑偏了")
		planSeedProposeInputs(t, env)
		_, err := env.a.NovelChapterPlanPropose(3, "")
		planWantErrContains(t, err, "解析失败", "原始返回前", "不是 JSON")
	})

	t.Run("关键事件不足不发半成品", func(t *testing.T) {
		p := planValidFixture(3)
		p.KeyEvents = []string{"只给了一条"}
		b, _ := json.Marshal(p)
		env := newPlanStubEnv(t, string(b))
		planSeedProposeInputs(t, env)
		_, err := env.a.NovelChapterPlanPropose(3, "")
		planWantErrContains(t, err, types.PlanProblemMissingKeyEvents, "不返回半成品")
	})

	t.Run("模型客户端不可用如实报错", func(t *testing.T) {
		a := &App{core: &core{}}
		a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts")}
		a.setPM(newPlanTestProject(t))
		_, err := a.NovelChapterPlanPropose(3, "")
		planWantErrContains(t, err, "AI 客户端未就绪")
	})

	t.Run("缺大纲节点如实报错", func(t *testing.T) {
		env := newPlanStubEnv(t, planProposeReply(9))
		_, err := env.a.NovelChapterPlanPropose(9, "")
		planWantErrContains(t, err, "未找到第 9 章的大纲节点")
	})
}

// TestNovelChapterPlanPropose_Direction v4.450.0 三点打通：direction（分支意向/
// 剧情要求）进编译输入；无节点时以 direction 兜底——分支→计划不再被「先有节点
// 还是先有计划」卡死。
func TestNovelChapterPlanPropose_Direction(t *testing.T) {
	const wantDirection = "分支意向：沈砚黑化线——旧案真凶是师尊"

	t.Run("有节点：direction 进编译输入", func(t *testing.T) {
		env := newPlanStubEnv(t, planProposeReply(3))
		planSeedProposeInputs(t, env)
		plan, err := env.a.NovelChapterPlanPropose(3, wantDirection)
		if err != nil {
			t.Fatalf("Propose 失败: %v", err)
		}
		if !strings.Contains(env.userPrompt(), wantDirection) {
			t.Fatalf("user prompt 应包含创作方向，实际:\n%s", env.userPrompt())
		}
		if plan == nil || plan.SubIndex != 3 {
			t.Fatalf("返回计划异常: %+v", plan)
		}
	})

	t.Run("无节点+有 direction：兜底编译成功且不落盘", func(t *testing.T) {
		env := newPlanStubEnv(t, planProposeReply(9))
		plan, err := env.a.NovelChapterPlanPropose(9, wantDirection)
		if err != nil {
			t.Fatalf("direction 应兜底无节点提案: %v", err)
		}
		if !strings.Contains(env.userPrompt(), wantDirection) {
			t.Fatalf("user prompt 应包含创作方向，实际:\n%s", env.userPrompt())
		}
		// 无节点：chapter_outline 空槽位不渲染（大纲节点文本不得出现）
		if strings.Contains(env.userPrompt(), "本章大纲节点") {
			t.Fatalf("无节点时不应渲染大纲节点区段，实际:\n%s", env.userPrompt())
		}
		if plan.SubIndex != 9 {
			t.Fatalf("SubIndex 应为章号: %d", plan.SubIndex)
		}
		if !strings.Contains(env.userPrompt(), "阶段之「转」") {
			t.Fatalf("第9章计划应携带转教义，实际:%s", env.userPrompt())
		}
		if got, err := env.a.NovelChapterPlanGet(9); err != nil || got != nil {
			t.Fatalf("Propose 不得落盘: plan=%+v err=%v", got, err)
		}
	})
}

// ── Deviation 四类判据 + 两个正常态 ──────────────────────────

// TestNovelChapterPlanDeviationCriteria 四类判据（未达成 / 结尾不符 / 情绪漂移 /
// 跨章重复）与字段缺省（无法判定不报漂移）。
func TestNovelChapterPlanDeviationCriteria(t *testing.T) {
	type tc struct {
		name     string
		plan     types.ChapterPlan
		others   map[int]types.ChapterPlan
		analysis *types.ChapterAnalysisResult
		check    func(t *testing.T, dev *types.PlanDeviation)
	}

	allHit := planAnalysisFixture(2,
		[]string{"秦昭夜探风阁，撞见守夜人。", "玄铁剑沉入井底，水花四溅。"},
		"本章以井底的水声收束。", "悬念", "紧张递进")
	oneMiss := planAnalysisFixture(2,
		[]string{"秦昭夜探风阁，撞见守夜人。"},
		"", "悬念", "紧张递进")
	endingOff := planAnalysisFixture(2,
		[]string{"秦昭夜探风阁，撞见守夜人。", "玄铁剑沉入井底，水花四溅。"},
		"", "情感", "紧张递进")
	emotionOff := planAnalysisFixture(2,
		[]string{"秦昭夜探风阁，撞见守夜人。", "玄铁剑沉入井底，水花四溅。"},
		"", "悬念", "欢快")
	fieldsMissing := planAnalysisFixture(2,
		[]string{"秦昭夜探风阁，撞见守夜人。", "玄铁剑沉入井底，水花四溅。"},
		"", "", "")
	dupOther := planValidFixture(1)
	dupOther.KeyEvents = []string{"秦昭夜探风阁", "沈砚旧誓再提"}
	dupOther.PlotSummary = "另一章。"

	cases := []tc{
		{
			name: "全部达成", plan: planValidFixture(2), analysis: &allHit,
			check: func(t *testing.T, dev *types.PlanDeviation) {
				if len(dev.MissingEvents) != 0 {
					t.Fatalf("不应有未达成事件: %v", dev.MissingEvents)
				}
				if dev.EndingMismatch || dev.EmotionDrift || len(dev.DuplicateEvents) != 0 {
					t.Fatalf("不应有偏差: %+v", *dev)
				}
				if dev.Summary != "2/2 关键事件达成" {
					t.Fatalf("Summary=%q", dev.Summary)
				}
				if !strings.Contains(dev.NextSuggestion, "下一章计划建议") {
					t.Fatalf("应有下一章建议: %q", dev.NextSuggestion)
				}
			},
		},
		{
			name: "① 关键事件未达成", plan: planValidFixture(2), analysis: &oneMiss,
			check: func(t *testing.T, dev *types.PlanDeviation) {
				if len(dev.MissingEvents) != 1 || dev.MissingEvents[0] != "玄铁剑沉入井底" {
					t.Fatalf("未达成事件判定错: %v", dev.MissingEvents)
				}
				if !strings.Contains(dev.Summary, "1/2 关键事件达成") {
					t.Fatalf("Summary=%q", dev.Summary)
				}
				if !strings.Contains(dev.NextSuggestion, "玄铁剑沉入井底") {
					t.Fatalf("下一章建议应点名未达成事件: %q", dev.NextSuggestion)
				}
			},
		},
		{
			name: "② 结尾类型不符", plan: planValidFixture(2), analysis: &endingOff,
			check: func(t *testing.T, dev *types.PlanDeviation) {
				if !dev.EndingMismatch {
					t.Fatalf("应判结尾不符: %+v", *dev)
				}
				if dev.PlannedEnding != string(types.EndingSuspense) || dev.ActualEnding != string(types.EndingEmotional) {
					t.Fatalf("结尾类型映射错: planned=%q actual=%q", dev.PlannedEnding, dev.ActualEnding)
				}
				if !strings.Contains(dev.Summary, "结尾类型不符（计划：悬念 / 实际：情感收尾）") {
					t.Fatalf("Summary=%q", dev.Summary)
				}
			},
		},
		{
			name: "③ 情绪漂移", plan: planValidFixture(2), analysis: &emotionOff,
			check: func(t *testing.T, dev *types.PlanDeviation) {
				if !dev.EmotionDrift {
					t.Fatalf("应判情绪漂移: %+v", *dev)
				}
				if dev.PlannedEmotion != "紧张递进" || dev.ActualEmotion != "欢快" {
					t.Fatalf("情绪字段错: planned=%q actual=%q", dev.PlannedEmotion, dev.ActualEmotion)
				}
				if !strings.Contains(dev.Summary, "情绪漂移") {
					t.Fatalf("Summary=%q", dev.Summary)
				}
			},
		},
		{
			name: "④ 跨章重复事件", plan: planValidFixture(2),
			others: map[int]types.ChapterPlan{1: dupOther}, analysis: &allHit,
			check: func(t *testing.T, dev *types.PlanDeviation) {
				if len(dev.DuplicateEvents) == 0 {
					t.Fatalf("应点名重复事件: %+v", *dev)
				}
				if joined := strings.Join(dev.DuplicateEvents, "|"); !strings.Contains(joined, "秦昭夜探风阁") {
					t.Fatalf("重复事件应点名原文: %v", dev.DuplicateEvents)
				}
				if !strings.Contains(dev.Summary, "重复") {
					t.Fatalf("Summary=%q", dev.Summary)
				}
			},
		},
		{
			name: "字段缺省=无法判定不报漂移", plan: planValidFixture(2), analysis: &fieldsMissing,
			check: func(t *testing.T, dev *types.PlanDeviation) {
				if dev.EndingMismatch || dev.EmotionDrift {
					t.Fatalf("缺省字段不得判漂移: %+v", *dev)
				}
				if dev.ActualEnding != "" || dev.ActualEmotion != "" {
					t.Fatalf("缺省应留空: ending=%q emotion=%q", dev.ActualEnding, dev.ActualEmotion)
				}
				if len(dev.MissingEvents) != 0 {
					t.Fatalf("事件仍应命中: %v", dev.MissingEvents)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, pm := newPlanTestApp(t)
			seed := map[int]types.ChapterPlan{2: c.plan}
			for num, p := range c.others {
				seed[num] = p
			}
			planRawSeed(t, pm, seed)
			if c.analysis != nil {
				if err := pm.UpsertAnalysisV2(*c.analysis); err != nil {
					t.Fatalf("写分析载荷: %v", err)
				}
			}

			dev, err := a.NovelChapterPlanDeviation(2)
			if err != nil {
				t.Fatalf("Deviation 失败: %v", err)
			}
			if !dev.HasPlan || !dev.Analyzed {
				t.Fatalf("报告应标记 HasPlan/Analyzed: %+v", *dev)
			}
			c.check(t, dev)

			// 尽力落盘：分析已完成 → 偏差应可读回
			stored, ok, err := pm.ReadPlanDeviation(2)
			if err != nil || !ok || stored == nil {
				t.Fatalf("偏差应落盘: ok=%v stored=%+v err=%v", ok, stored, err)
			}
			if stored.Summary != dev.Summary {
				t.Fatalf("落盘偏差与返回不一致: %q vs %q", stored.Summary, dev.Summary)
			}
		})
	}
}

// TestNovelChapterPlanDeviationNormalStates 无计划 / 未分析两个正常态（不报错）。
func TestNovelChapterPlanDeviationNormalStates(t *testing.T) {
	t.Run("无计划", func(t *testing.T) {
		a, pm := newPlanTestApp(t)
		dev, err := a.NovelChapterPlanDeviation(4)
		if err != nil {
			t.Fatalf("无计划是正常态，不应报错: %v", err)
		}
		if dev == nil || dev.HasPlan || dev.Analyzed {
			t.Fatalf("应为 {HasPlan:false}: %+v", dev)
		}
		if dev.ChapterNum != 4 || dev.Summary == "" {
			t.Fatalf("正常态应带章号与中文概要: %+v", *dev)
		}
		if !strings.Contains(dev.NextSuggestion, "尚未制定计划") {
			t.Fatalf("无计划时应给补计划建议: %q", dev.NextSuggestion)
		}
		if _, ok, _ := pm.ReadPlanDeviation(4); ok {
			t.Fatal("无计划态不应落盘偏差报告")
		}
	})

	t.Run("未分析", func(t *testing.T) {
		a, pm := newPlanTestApp(t)
		planSaveMust(t, a, 5, planValidFixture(5))
		dev, err := a.NovelChapterPlanDeviation(5)
		if err != nil {
			t.Fatalf("未分析是正常态，不应报错: %v", err)
		}
		if dev == nil || !dev.HasPlan || dev.Analyzed {
			t.Fatalf("应为 {HasPlan:true,Analyzed:false}: %+v", dev)
		}
		if !strings.Contains(dev.Summary, "尚未分析") {
			t.Fatalf("Summary=%q", dev.Summary)
		}
		if !strings.Contains(dev.NextSuggestion, "分析本章") {
			t.Fatalf("未分析时应提示先分析: %q", dev.NextSuggestion)
		}
		if _, ok, _ := pm.ReadPlanDeviation(5); ok {
			t.Fatal("未分析态不应落盘偏差报告")
		}
	})
}

// TestNovelChapterPlanDeviationPersistBestEffort 落盘失败**不影响返回**：
// 用同名普通文件占位 deviation 落盘目录（规格 §7.4 口径 analysis/plan-deviation/），
// 让 WritePlanDeviation 失败——报告仍须完整返回。
//
// 若线A 的目录口径与本测试的占位不符，占位不生效，本用例退化为「正常落盘」，
// 断言（返回完整）依然成立。
func TestNovelChapterPlanDeviationPersistBestEffort(t *testing.T) {
	a, pm := newPlanTestApp(t)
	planSaveMust(t, a, 6, planValidFixture(6))
	if err := pm.UpsertAnalysisV2(planAnalysisFixture(6,
		[]string{"秦昭夜探风阁。", "玄铁剑沉入井底。"}, "", "悬念", "紧张递进")); err != nil {
		t.Fatalf("写分析载荷: %v", err)
	}

	blocked := filepath.Join(pm.Dir, "analysis", "plan-deviation")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err == nil {
		if werr := os.WriteFile(blocked, []byte("占位：让偏差落盘失败"), 0o644); werr == nil {
			t.Logf("已用普通文件占位 %s（偏差落盘应失败）", blocked)
		}
	}

	dev, err := a.NovelChapterPlanDeviation(6)
	if err != nil {
		t.Fatalf("落盘失败不应影响返回: %v", err)
	}
	if dev == nil || !dev.Analyzed || dev.Summary == "" {
		t.Fatalf("报告应完整返回: %+v", dev)
	}
	if len(dev.MissingEvents) != 0 {
		t.Fatalf("本章两条事件均命中: %v", dev.MissingEvents)
	}
}

// ── 纯函数：归一化、命中阈值、others 组装 ────────────────────

// TestPlanEventHitNormalize 归一化（空白/全半角/大小写）与 2-gram 覆盖率阈值。
func TestPlanEventHitNormalize(t *testing.T) {
	cases := []struct {
		name  string
		event string
		hay   string
		want  bool
	}{
		{"整体包含", "秦昭夜探风阁", "本章写秦昭夜探风阁，撞见守夜人。", true},
		{"空白与全角空格差异", "秦昭 夜探　风阁", "秦昭夜探风阁", true},
		{"大小写与全角字母", "ＡＢＣ行动", "abc行动开始了", true},
		{"虚词插入仍命中", "林晚夺得玄铁剑", "林晚夺得了玄铁剑，剑身嗡鸣", true},
		{"无关文本不命中", "林晚夺得玄铁剑", "沈砚在酒肆里喝了一夜的酒", false},
		{"短事件只做包含判定", "下雪", "那天没有下雪", true},
		{"短事件无关不命中", "下雪", "天气炎热", false},
		{"空池不命中", "秦昭夜探风阁", "", false},
		{"空事件不命中", "   ", "秦昭夜探风阁", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := planEventHit(c.event, c.hay); got != c.want {
				t.Fatalf("planEventHit(%q,%q)=%v want %v", c.event, c.hay, got, c.want)
			}
		})
	}

	// 覆盖率阈值本身（6 个二元组）：2/6=0.33 < 0.6 未命中；4/6=0.67 ≥ 0.6 命中。
	partial := "秦昭夜探风阁张三李四"
	if planEventHit(partial, "秦昭夜探别处") {
		t.Fatal("覆盖率 2/6 应低于阈值 0.6，判未命中")
	}
	if !planEventHit(partial, "秦昭夜探风阁张三") {
		t.Fatal("覆盖率 4/6 应达到阈值 0.6，判命中")
	}
}

// TestPlanOthersAndKeyEventsText planOthers 章号升序、跳过本章与非章号键。
func TestPlanOthersAndKeyEventsText(t *testing.T) {
	pf := &types.ChapterPlanFile{Version: 1, Plans: map[string]types.ChapterPlan{
		"3":   {KeyEvents: []string{"三章事件"}},
		"1":   {KeyEvents: []string{"一章事件"}},
		"2":   {KeyEvents: []string{"二章事件"}},
		"abc": {KeyEvents: []string{"脏数据"}},
	}}
	others := planOthers(pf, 2)
	if len(others) != 2 {
		t.Fatalf("应取除本章外的 2 条: %+v", others)
	}
	if others[0].KeyEvents[0] != "一章事件" || others[1].KeyEvents[0] != "三章事件" {
		t.Fatalf("应章号升序：%+v", others)
	}
	text := planExistingKeyEventsText(others)
	if !strings.Contains(text, "一章事件") || !strings.Contains(text, "三章事件") ||
		strings.Contains(text, "二章事件") || strings.Contains(text, "脏数据") {
		t.Fatalf("已有事件文本错: %q", text)
	}
	if planExistingKeyEventsText(nil) != "" {
		t.Fatal("无其它计划时输入段应为空（模板空槽位不渲染）")
	}
}

// ── Precheck 委托 ─────────────────────────────────────────────

// TestNovelChapterPlanGatePrecheckDelegates 缺计划 → Blocking（委托线B planPrecheck）。
func TestNovelChapterPlanGatePrecheckDelegates(t *testing.T) {
	a, _ := newPlanTestApp(t)
	rep, err := a.NovelChapterGatePrecheck(1)
	if err != nil {
		t.Fatalf("预检不应报错（缺计划是判定结果，不是故障）: %v", err)
	}
	if rep == nil {
		t.Fatal("预检应返回报告")
	}
	if rep.ChapterNum != 1 || rep.HasPlan {
		t.Fatalf("缺计划应 HasPlan=false: %+v", *rep)
	}
	if !rep.Blocking || rep.Allowed {
		t.Fatalf("缺计划应 Blocking=true / Allowed=false: %+v", *rep)
	}
	if len(rep.Missing) == 0 {
		t.Fatalf("缺计划应给出缺失项: %+v", *rep)
	}
}
