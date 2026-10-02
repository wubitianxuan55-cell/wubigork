package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	"strings"
)

// ── 打桩：本地 OpenAI 兼容 mock（httptest），按调用次序回放 replies 并计数 ──
// 打桩方式对齐 create_chapter_prompt_test.go：SaveEngine 注册 herdsman 引擎
// 指向 httptest server，SSE 分片回放 delta.content。

// plotBranchTestEnv 剧情分支链路测试最小环境：可计数假 LLM + 临时小说项目。
type plotBranchTestEnv struct {
	a     *App
	pm    *project.Manager
	calls *int32 // AI 请求计数

	mu   sync.Mutex
	user string // 最近一次请求的 user prompt（位置注入断言用）
}

// userPrompt 返回最近一次请求的 user prompt。
func (e *plotBranchTestEnv) userPrompt() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.user
}

// newPlotBranchTestEnv replies 按调用次序回放；超出后重复最后一条。
func newPlotBranchTestEnv(t *testing.T, replies []string) *plotBranchTestEnv {
	t.Helper()

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	var calls int32
	env := &plotBranchTestEnv{calls: &calls}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := int(atomic.AddInt32(&calls, 1)) - 1
		if idx >= len(replies) {
			idx = len(replies) - 1
		}
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if raw, err := io.ReadAll(r.Body); err == nil {
			_ = json.Unmarshal(raw, &req)
			for _, m := range req.Messages {
				if m.Role == "user" {
					env.mu.Lock()
					env.user = m.Content
					env.mu.Unlock()
				}
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
	pm, err := project.Create(dir, "测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	a.ctx = ctx
	a.setPM(pm)

	env.a = a
	env.pm = pm
	return env
}

// seedOutlineNode 写入一个可供分支应用的大纲节点
func (e *plotBranchTestEnv) seedOutlineNode(t *testing.T, id string) {
	t.Helper()
	of := &types.OutlineFile{Nodes: []types.OutlineNode{{
		ID: id, Title: "第1章", OrderIndex: 1, Status: types.OutlinePlanned,
	}}}
	if err := e.pm.WriteOutlines(of); err != nil {
		t.Fatalf("写大纲: %v", err)
	}
}

// branchReply 构造一段含 3 条分支的 AI 回复，prefix 用于区分不同次生成的内容
func branchReply(prefix string) string {
	branches := make([]map[string]interface{}, 0, 3)
	for i := 1; i <= 3; i++ {
		branches = append(branches, map[string]interface{}{
			"id":                  fmt.Sprintf("%d", i),
			"title":               fmt.Sprintf("%s钩子%d", prefix, i),
			"summary":             fmt.Sprintf("%s摘要%d：具体场景开头", prefix, i),
			"characters_involved": []string{"林晚", fmt.Sprintf("%s新角%d", prefix, i)},
			"core_conflict":       fmt.Sprintf("%s冲突%d", prefix, i),
			"foreshadow_impact":   fmt.Sprintf("%s伏笔%d", prefix, i),
			"tone":                "紧张",
		})
	}
	b, err := json.Marshal(map[string]interface{}{"branches": branches})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestApplyBranchUsesStoredBranchesWithoutRegenerating 核心闭环：brainstorm 持久化后，
// apply 必须命中存储的第 index 条分支（与用户预览一致），且完全不再二次调用 AI。
func TestApplyBranchUsesStoredBranchesWithoutRegenerating(t *testing.T) {
	// 第 1 次调用回「初版」，若误触发重生成则会回「重生成」（内容不同，可断言区分）
	env := newPlotBranchTestEnv(t, []string{branchReply("初版"), branchReply("重生成")})
	env.seedOutlineNode(t, "node-1")

	if _, err := env.a.BrainstormBranches("node-1"); err != nil {
		t.Fatalf("BrainstormBranches: %v", err)
	}
	if got := atomic.LoadInt32(env.calls); got != 1 {
		t.Fatalf("brainstorm 应恰好调用 1 次 AI，实际 %d 次", got)
	}
	if _, err := os.Stat(filepath.Join(env.pm.Dir, "branches.json")); err != nil {
		t.Fatalf("brainstorm 结果未持久化到 branches.json: %v", err)
	}

	// 归零计数：apply 阶段任何 AI 调用都会被捕获
	atomic.StoreInt32(env.calls, 0)
	res, err := env.a.ApplyBranch("node-1", 1, "")
	if err != nil {
		t.Fatalf("ApplyBranch: %v", err)
	}
	if got := atomic.LoadInt32(env.calls); got != 0 {
		t.Fatalf("apply 命中存储后不得再调 AI，实际调了 %d 次", got)
	}
	if from, _ := res["applied_from"].(string); from != "stored" {
		t.Fatalf("applied_from = %v，期望 stored", res["applied_from"])
	}
	branch, ok := res["branch"].(PlotBranch)
	if !ok {
		t.Fatalf("返回缺少 branch 字段: %v", res)
	}
	// 必须是「初版」第 2 条（index=1），而非重生成的「重生成」第 2 条
	if branch.Summary != "初版摘要2：具体场景开头" {
		t.Fatalf("应用的分支与预览不一致：summary=%q", branch.Summary)
	}
	if branch.CoreConflict != "初版冲突2" || branch.ForeshadowImpact != "初版伏笔2" {
		t.Fatalf("KeyPoints 来源分支错误: %q / %q", branch.CoreConflict, branch.ForeshadowImpact)
	}

	// 大纲节点已写入存储的分支内容
	of, err := env.pm.ReadOutlines()
	if err != nil {
		t.Fatalf("回读大纲: %v", err)
	}
	node := of.Nodes[0]
	if node.Summary != "初版摘要2：具体场景开头" {
		t.Fatalf("大纲节点 summary 未落盘: %q", node.Summary)
	}
	if node.Emotion != "紧张" {
		t.Fatalf("大纲节点 emotion = %q", node.Emotion)
	}
	// 分支角色已幂等物化进 characters.json（ApplyBranch 内联同步）
	cf, err := env.pm.ReadCharacters()
	if err != nil {
		t.Fatalf("回读角色: %v", err)
	}
	if len(cf.Characters) != 2 {
		t.Fatalf("角色同步应建 2 张卡，实际 %d", len(cf.Characters))
	}
}

// TestApplyBranchRegeneratesWhenNoStoredBranches 旧数据兼容：无持久化（branches.json
// 缺失）时回退现场重生成（恰好 1 次 AI），并在返回的分支说明里注明。
func TestApplyBranchRegeneratesWhenNoStoredBranches(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{branchReply("回退")})
	env.seedOutlineNode(t, "node-1")

	res, err := env.a.ApplyBranch("node-1", 0, "")
	if err != nil {
		t.Fatalf("ApplyBranch: %v", err)
	}
	if got := atomic.LoadInt32(env.calls); got != 1 {
		t.Fatalf("回退路径应恰好调用 1 次 AI，实际 %d 次", got)
	}
	if from, _ := res["applied_from"].(string); from != "regenerated" {
		t.Fatalf("applied_from = %v，期望 regenerated", res["applied_from"])
	}
	note, _ := res["branch_note"].(string)
	if note == "" {
		t.Fatalf("回退路径必须在 branch_note 中注明")
	}
	branch, ok := res["branch"].(PlotBranch)
	if !ok || branch.Summary != "回退摘要1：具体场景开头" {
		t.Fatalf("回退应用的分支内容错误: %v", res["branch"])
	}
}

// TestApplyBranchSyncCharactersIdempotent 角色同步幂等（ApplyBranch 后触发的
// syncCharactersFromOutline）：跑两遍不重复建卡、不覆盖既有角色字段、
// 组织/关系原样保留、不调用 AI。
func TestApplyBranchSyncCharactersIdempotent(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{branchReply("占位")})

	cf := &types.CharacterFile{
		Characters: []types.Character{
			{ID: "ch_1", Name: "林晚", RoleType: "protagonist", Personality: "清冷", Status: "Alive"},
		},
		Organizations: []types.Organization{
			{ID: "org_1", Name: "青云宗", Members: []string{"ch_1"}},
		},
		Relationships: []types.Relationship{
			{FromID: "ch_1", ToID: "ch_2", RelationType: "rival", Intimacy: -10},
		},
	}
	if err := env.pm.WriteCharacters(cf); err != nil {
		t.Fatalf("写角色文件: %v", err)
	}

	// 林晚=按名字命中跳过；ch_1=按 ID 命中跳过；带空格新名字需 trim 后物化
	node := &types.OutlineNode{
		ID:         "node-1",
		Title:      "第2章",
		Characters: []string{"林晚", "ch_1", "  苏烈  ", "新角色甲"},
	}
	for run := 1; run <= 2; run++ {
		env.a.syncCharactersFromOutline(node)

		got, err := env.pm.ReadCharacters()
		if err != nil {
			t.Fatalf("第 %d 遍回读角色: %v", run, err)
		}
		if len(got.Characters) != 3 {
			t.Fatalf("第 %d 遍：应 3 个角色（不重复建卡），实际 %d", run, len(got.Characters))
		}
		byName := map[string]types.Character{}
		for _, ch := range got.Characters {
			byName[ch.Name] = ch
		}
		lin, ok := byName["林晚"]
		if !ok || lin.ID != "ch_1" || lin.Personality != "清冷" || lin.RoleType != "protagonist" {
			t.Fatalf("第 %d 遍：既有角色被覆盖或丢失: %+v", run, lin)
		}
		su, ok := byName["苏烈"]
		if !ok || su.Status != "Alive" || su.ID == "" {
			t.Fatalf("第 %d 遍：新角色「苏烈」物化失败: %+v", run, su)
		}
		if _, ok := byName["新角色甲"]; !ok {
			t.Fatalf("第 %d 遍：新角色「新角色甲」物化失败", run)
		}
		if len(got.Organizations) != 1 || got.Organizations[0].Name != "青云宗" || len(got.Organizations[0].Members) != 1 {
			t.Fatalf("第 %d 遍：组织字段未保留: %+v", run, got.Organizations)
		}
		if len(got.Relationships) != 1 || got.Relationships[0].RelationType != "rival" {
			t.Fatalf("第 %d 遍：关系字段未保留: %+v", run, got.Relationships)
		}
	}
	if got := atomic.LoadInt32(env.calls); got != 0 {
		t.Fatalf("角色同步不得调用 AI，实际 %d 次", got)
	}
}

// ── v4.451.0 分支剧情 50 字压缩：分支只是方向，细节由章节计划承担 ──

// TestClampBranchPitch 超预算分支剧情的确定性收口：句界优先，无句读硬截。
func TestClampBranchPitch(t *testing.T) {
	short := "雨夜断桥，林晚截杀叛徒，代价是暴露身份。"
	if got := clampBranchPitch(short); got != short {
		t.Fatalf("预算内应原样保留: %q", got)
	}

	exact := strings.Repeat("雪", 50)
	if got := clampBranchPitch(exact); got != exact {
		t.Fatalf("恰满预算应原样: %q", got)
	}

	// 超长：预算内最后一个句读符后截断（完整句子，不切断语意）
	long := "雨夜断桥，林晚截杀叛徒。" + strings.Repeat("混战中玄铁剑坠河。", 4) + "她必须捞回。"
	got := clampBranchPitch(long)
	want := "雨夜断桥，林晚截杀叛徒。" + strings.Repeat("混战中玄铁剑坠河。", 4)
	if got != want {
		t.Fatalf("应句界截断: got=%q want=%q", got, want)
	}
	if strings.HasSuffix(got, "她必须捞回。") {
		t.Fatalf("预算外的尾句应被丢弃: %q", got)
	}

	// 无句读：硬截 49 字 + 省略号，总长仍守 50 字预算
	noPunct := strings.Repeat("雨", 60)
	if got := clampBranchPitch(noPunct); got != strings.Repeat("雨", 49)+"…" {
		t.Fatalf("应硬截+省略号且总长不超预算: %q", got)
	}
}

// TestParsePlotBranchReplyClampsPitch 解析收口：回复里超长的 summary 统一压到 50 字内。
func TestParsePlotBranchReplyClampsPitch(t *testing.T) {
	reply, err := json.Marshal(map[string]interface{}{"branches": []map[string]interface{}{{
		"id": "1", "title": "钩子", "summary": "雨夜断桥，林晚截杀叛徒。" + strings.Repeat("长", 80),
	}}})
	if err != nil {
		t.Fatalf("构造回复: %v", err)
	}
	branches, err := parsePlotBranchReply(string(reply))
	if err != nil {
		t.Fatalf("解析: %v", err)
	}
	got := branches[0].Summary
	if n := len([]rune(got)); n > 50 {
		t.Fatalf("summary 应压到 50 字内，实际 %d 字: %q", n, got)
	}
	if want := "雨夜断桥，林晚截杀叛徒。"; got != want {
		t.Fatalf("预算内有句读应句界截断: got=%q want=%q", got, want)
	}
}

// ── v4.452.0 位置感知接通：QuickBrainstormBranches 携带目标章号 → 起承转合教义进 prompt ──

// TestQuickBrainstormBranchesPosition 向导主链路的位置注入：章号>0 按阶段位置
// 切换教义（第8章=转）；章号 0 保持旧兜底（有前文=常规推进）。
func TestQuickBrainstormBranchesPosition(t *testing.T) {
	env := newPlotBranchTestEnv(t, []string{branchReply("A")})
	if _, err := env.a.QuickBrainstormBranches("设定", "前文摘要", "[]", 8); err != nil {
		t.Fatalf("构思失败: %v", err)
	}
	prompt := env.userPrompt()
	if !strings.Contains(prompt, "阶段之「转」") || !strings.Contains(prompt, "第1~10章") {
		t.Fatalf("第8章构思应携带转教义（第1~10章），实际:%s", prompt)
	}

	env0 := newPlotBranchTestEnv(t, []string{branchReply("B")})
	if _, err := env0.a.QuickBrainstormBranches("设定", "前文摘要", "[]", 0); err != nil {
		t.Fatalf("构思失败: %v", err)
	}
	if !strings.Contains(env0.userPrompt(), "常规推进") {
		t.Fatalf("章号未知应回落常规推进，实际:%s", env0.userPrompt())
	}
}
