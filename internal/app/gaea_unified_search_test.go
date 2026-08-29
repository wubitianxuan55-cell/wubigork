package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// newUnifiedSearchEnv 搭建统一检索测试环境：临时用户目录 + mock embedding 服务
// + 临时 Hephaestus.db；成本/办公/知识/向量索引全部注入隔离存储，避免触碰真实库。
func newUnifiedSearchEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"data":[{"id":"bge-m3"}]}`))
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]map[string]any, 0, len(req.Input))
		for i, s := range req.Input {
			vec := []float32{0, 0}
			if strings.Contains(s, "锤") || strings.Contains(s, "桩") {
				vec[0] = 1
			}
			if strings.Contains(s, "水泥") {
				vec[1] = 1
			}
			data = append(data, map[string]any{"index": i, "embedding": vec})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(srv.Close)

	gdb := db.GetDatabase(home)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() {
		db.CloseDatabase(home)
		SetAppSemanticStoreForTest(nil)
		SetAppEmbedderForTest(nil)
		knowledge.SetStoreForTest(nil)
		ResetOfficeStoreForTest()
		ResetCostStoreForTest()
	})
	SetAppSemanticStoreForTest(semantic.Open(gdb))
	SetAppEmbedderForTest(retrieval.NewEmbedder(srv.URL, "bge-m3"))
	SetOfficeStoreForTest(memory.SQLiteStoreFor(gdb, home, home))
	SetCostStoreForTest(cost.Open(gdb))

	isoStore, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	knowledge.SetStoreForTest(isoStore)
}

// TestGaeaUnifiedSearch_Combined 组合结果正确性：一次调用同时返回
// 关键词全文命中（工作区文件）与跨库语义命中（知识/成本/办公三类）。
func TestGaeaUnifiedSearch_Combined(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/方案.md", []byte("振动锤选型要点：需匹配地质条件。"), 0o644); err != nil {
		t.Fatal(err)
	}
	newUnifiedSearchEnv(t)

	// 工程知识库条目（语义跨库命中：knowledge）
	ks, err := knowledge.Global().Store()
	if err != nil {
		t.Fatal(err)
	}
	if err := ks.Save(knowledge.Entry{
		Name: "pile-guide", Title: "桩基施工要点", Category: knowledge.CatCase,
		Body: "振动锤选型需匹配地质，桩基施工工艺详见规范。",
	}); err != nil {
		t.Fatal(err)
	}

	// 显式填充嵌入的 *core：embedding 未配置时 localSearchEmbedder 会走到
	// resolveHerdsmanSearchModel（读 a.engineMgr），零值 App 的 core 为 nil 会崩溃。
	a := &App{core: &core{}}
	// 成本条目（语义跨库命中：cost）
	if err := a.hubCostStore().Save(cost.Entry{
		Name: "pile-rent", Title: "振动锤租赁台班", Category: "机械", Unit: "台班", Price: 6500,
	}); err != nil {
		t.Fatal(err)
	}
	// 办公记忆（语义跨库命中：office）
	if _, err := a.hubOfficeStore().Save(memory.Memory{
		Name: "office-pile", Title: "打桩机械台账", Description: "振动锤与静压桩机配置", Body: "记录现场机械调度。",
	}); err != nil {
		t.Fatal(err)
	}

	view, err := a.GaeaUnifiedSearch("振动锤", 10)
	if err != nil {
		t.Fatalf("GaeaUnifiedSearch: %v", err)
	}

	// 关键词段：命中工作区文件 docs/方案.md
	kwHit := false
	for _, h := range view.Keyword {
		if h.Path == "docs/方案.md" {
			kwHit = true
			if !strings.Contains(h.Snippet, "振动锤") {
				t.Errorf("关键词片段应含查询词: %q", h.Snippet)
			}
		}
	}
	if !kwHit {
		t.Fatalf("关键词未命中 docs/方案.md: %+v", view.Keyword)
	}

	// 语义段：跨库命中 knowledge/cost/office 三种
	wantSem := map[string]bool{
		"knowledge/pile-guide": false,
		"cost/pile-rent":       false,
		"office/office-pile":   false,
	}
	for _, h := range view.Semantic {
		key := h.Kind + "/" + h.Name
		if _, ok := wantSem[key]; ok {
			wantSem[key] = true
		}
	}
	for key, found := range wantSem {
		if !found {
			t.Errorf("语义未命中 %s: %+v", key, view.Semantic)
		}
	}

	// 记忆统一层扩展：brain 组在测试环境（未 initBrain，a.brain==nil）应为
	// 空数组且不报错；files 组结构完整（本环境无预建文件向量，命中可空但数组非 nil）。
	if view.Brain == nil {
		t.Fatal("brain 应为空数组（非 nil），保证 JSON 序列化为 []")
	}
	if len(view.Brain) != 0 {
		t.Errorf("未装配三脑时 brain 应为空: %+v", view.Brain)
	}
	if view.Files == nil {
		t.Fatal("files 应为数组（非 nil），保证 JSON 序列化为 []")
	}
}

// TestGaeaUnifiedSearch_EmptyQuery 空 query 返回空视图（四组均为空数组），不报错。
func TestGaeaUnifiedSearch_EmptyQuery(t *testing.T) {
	// 显式填充嵌入的 *core：embedding 未配置时 localSearchEmbedder 会走到
	// resolveHerdsmanSearchModel（读 a.engineMgr），零值 App 的 core 为 nil 会崩溃。
	a := &App{core: &core{}}
	view, err := a.GaeaUnifiedSearch("", 10)
	if err != nil {
		t.Fatalf("空 query 不应报错: %v", err)
	}
	if len(view.Keyword) != 0 || len(view.Semantic) != 0 || len(view.Brain) != 0 || len(view.Files) != 0 {
		t.Fatalf("空 query 应返回空视图: %+v", view)
	}
	if view.Keyword == nil || view.Semantic == nil || view.Brain == nil || view.Files == nil {
		t.Fatalf("空 query 也应返回空数组（非 nil，JSON 序列化为 []）: %+v", view)
	}
	view2, err := a.GaeaUnifiedSearch("   ", 10)
	if err != nil {
		t.Fatalf("空白 query 不应报错: %v", err)
	}
	if len(view2.Keyword) != 0 || len(view2.Semantic) != 0 || len(view2.Brain) != 0 || len(view2.Files) != 0 {
		t.Fatalf("空白 query 应返回空视图: %+v", view2)
	}
}

// TestGaeaUnifiedSearch_BrainNil 三脑未装配（a.brain==nil）时 brain 组为空数组
// 且不报错——hub 搜索降级为 keyword/semantic/files 照常。
func TestGaeaUnifiedSearch_BrainNil(t *testing.T) {
	t.Chdir(t.TempDir())
	newUnifiedSearchEnv(t)
	a := &App{core: &core{}}
	view, err := a.GaeaUnifiedSearch("振动锤", 10)
	if err != nil {
		t.Fatalf("brain nil 不应报错: %v", err)
	}
	if view.Brain == nil {
		t.Fatal("brain 应为空数组（非 nil）")
	}
	if len(view.Brain) != 0 {
		t.Fatalf("brain nil 时命中应为空: %+v", view.Brain)
	}
}

// TestGaeaUnifiedSearch_EmbeddingUnavailable embedding 不可用降级：
// semantic 为空数组而 keyword 照常返回（与现有降级行为一致）。
func TestGaeaUnifiedSearch_EmbeddingUnavailable(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/成本测算.md", []byte("本项目成本测算总金额为 100 万元。"), 0o644); err != nil {
		t.Fatal(err)
	}
	newUnifiedSearchEnv(t)
	// 模拟 embedding 未配置/引擎未启动
	SetAppEmbedderForTest(nil)

	// 显式填充嵌入的 *core：embedding 未配置时 localSearchEmbedder 会走到
	// resolveHerdsmanSearchModel（读 a.engineMgr），零值 App 的 core 为 nil 会崩溃。
	a := &App{core: &core{}}
	view, err := a.GaeaUnifiedSearch("成本", 10)
	if err != nil {
		t.Fatalf("embedding 不可用不应报错: %v", err)
	}
	if view.Semantic == nil {
		t.Fatal("semantic 应为空数组（非 nil），保证 JSON 序列化为 []")
	}
	if len(view.Semantic) != 0 {
		t.Fatalf("embedding 不可用时 semantic 应为空: %+v", view.Semantic)
	}
	kwHit := false
	for _, h := range view.Keyword {
		if h.Path == "docs/成本测算.md" {
			kwHit = true
		}
	}
	if !kwHit {
		t.Fatalf("embedding 不可用时关键词仍应命中: %+v", view.Keyword)
	}
}

// scopeHitByName / scopeBrainHit 辅助：检索结果按名称/脑归类。
func scopeSemNames(hits []SemanticHitView) map[string]bool {
	out := map[string]bool{}
	for _, h := range hits {
		out[h.Kind+"/"+h.Name] = true
	}
	return out
}

func scopeBrainHits(hits []Hit) map[string]bool {
	out := map[string]bool{}
	for _, h := range hits {
		out[h.Brain+"|"+h.Entity] = true
	}
	return out
}

// scopeFakeBrainAdapter 测试注入的三脑适配器（右脑/主脑命中可控）。
type scopeFakeBrainAdapter struct {
	hits []Hit
}

func (f *scopeFakeBrainAdapter) Read(string) ([]Fact, error)                { return nil, nil }
func (f *scopeFakeBrainAdapter) Write(string, string, string) error         { return nil }
func (f *scopeFakeBrainAdapter) Search(string) ([]Hit, error)               { return f.hits, nil }

// S1.2 B 读端隔离器：GaeaUnifiedSearch scope 过滤——工位搜索不见乐园记忆
//（semantic office 按空间回查、cost/knowledge/file 恒 work、brain 右脑 play
// 专属、keyword 共享工作区面不过滤）；scope 缺省 "" = 全部（旧行为）。
func TestGaeaUnifiedSearchScopeIsolation(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("docs/方案.md", []byte("振动锤选型要点：需匹配地质条件。"), 0o644); err != nil {
		t.Fatal(err)
	}
	newUnifiedSearchEnv(t)

	a := &App{core: &core{}}
	// 成本条目（恒 work 语义源）
	if err := a.hubCostStore().Save(cost.Entry{
		Name: "pile-rent", Title: "振动锤租赁台班", Category: "机械", Unit: "台班", Price: 6500,
	}); err != nil {
		t.Fatal(err)
	}
	// 办公 facts：work + play 各一条（同库混存两空间）
	if _, err := a.hubOfficeStore().Save(memory.Memory{
		Name: "office-pile", Title: "打桩机械台账", Description: "振动锤与静压桩机配置", Body: "记录现场机械调度。",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.hubOfficeStore().Save(memory.Memory{
		Name: "office-play", Space: "play", Title: "乐园设备", Description: "振动锤游戏机配置", Body: "游戏厅设备记录。",
	}); err != nil {
		t.Fatal(err)
	}
	// 三脑：主脑共享命中 + 右脑（whisper=play 专属）命中；左脑走真实 facts 源
	a.brain = &BrainStore{
		main:  &scopeFakeBrainAdapter{hits: []Hit{{Brain: BrainMain, Entity: "共享画像", Text: "振动锤相关画像"}}},
		left:  &leftBrain{src: &officeFactLeftSource{store: a.hubOfficeStore()}},
		right: &scopeFakeBrainAdapter{hits: []Hit{{Brain: BrainRight, Entity: "轻语记忆", Text: "振动锤轻语"}}},
	}
	t.Cleanup(func() { a.brain = nil })

	// ── scope=work：不见 play ──────────────────────────────────
	work, err := a.GaeaUnifiedSearch("振动锤", 10, "work")
	if err != nil {
		t.Fatalf("work scope: %v", err)
	}
	semWork := scopeSemNames(work.Semantic)
	if !semWork["cost/pile-rent"] || !semWork["office/office-pile"] {
		t.Fatalf("work scope 语义应含 cost+office-work: %v", semWork)
	}
	if semWork["office/office-play"] {
		t.Fatalf("work scope 不得出现 play 记忆（隔离红线）: %v", semWork)
	}
	brWork := scopeBrainHits(work.Brain)
	if _, ok := brWork["brain.right|轻语记忆"]; ok {
		t.Fatalf("work scope 不得出现 brain.right（whisper=play 专属）: %v", brWork)
	}
	if _, ok := brWork["brain.left|打桩机械台账"]; !ok {
		t.Fatalf("work scope 左脑应见 work 事实: %v", brWork)
	}
	if _, ok := brWork["brain.left|乐园设备"]; ok {
		t.Fatalf("work scope 左脑不得见 play 事实: %v", brWork)
	}
	if _, ok := brWork["brain.main|共享画像"]; !ok {
		t.Fatalf("主脑共享面 work 可见: %v", brWork)
	}
	// keyword 共享工作区面：work scope 照常命中
	kwWork := false
	for _, h := range work.Keyword {
		if h.Path == "docs/方案.md" {
			kwWork = true
		}
	}
	if !kwWork {
		t.Fatalf("keyword 共享面 work 应命中: %+v", work.Keyword)
	}

	// ── scope=play：office 只见 play 侧；cost/knowledge/file 不过滤（共享
	// 源、无 play 侧数据可漏，锚点 5）────────────────────────────
	play, err := a.GaeaUnifiedSearch("振动锤", 10, "play")
	if err != nil {
		t.Fatalf("play scope: %v", err)
	}
	semPlay := scopeSemNames(play.Semantic)
	if !semPlay["office/office-play"] {
		t.Fatalf("play scope 语义应含 play 记忆: %v", semPlay)
	}
	if !semPlay["cost/pile-rent"] {
		t.Fatalf("play scope cost/knowledge/file 不过滤（共享源）: %v", semPlay)
	}
	if semPlay["office/office-pile"] {
		t.Fatalf("play scope 不得出现 work 侧 office 记忆: %v", semPlay)
	}
	brPlay := scopeBrainHits(play.Brain)
	if _, ok := brPlay["brain.right|轻语记忆"]; !ok {
		t.Fatalf("play scope 应见 brain.right（play 专属）: %v", brPlay)
	}
	if _, ok := brPlay["brain.left|乐园设备"]; !ok {
		t.Fatalf("play scope 左脑应见 play 事实: %v", brPlay)
	}
	if _, ok := brPlay["brain.left|打桩机械台账"]; ok {
		t.Fatalf("play scope 左脑不得见 work 事实: %v", brPlay)
	}

	// ── scope 缺省 ""：全部（旧行为）──────────────────────────
	all, err := a.GaeaUnifiedSearch("振动锤", 10)
	if err != nil {
		t.Fatalf("缺省 scope: %v", err)
	}
	semAll := scopeSemNames(all.Semantic)
	if !semAll["office/office-pile"] || !semAll["office/office-play"] || !semAll["cost/pile-rent"] {
		t.Fatalf("缺省 scope 应全量（旧行为）: %v", semAll)
	}
	// 三脑全搜：主脑 1 + 左脑（work+play 事实各 1）+ 右脑 1 = 4
	if len(all.Brain) != 4 {
		t.Fatalf("缺省 scope 三脑全搜: %+v", all.Brain)
	}

	// ── 非法 scope：回退 ""（全部/旧行为）──────────────────────
	bogus, err := a.GaeaUnifiedSearch("振动锤", 10, "bogus")
	if err != nil {
		t.Fatalf("非法 scope 不应报错: %v", err)
	}
	if len(bogus.Semantic) != len(all.Semantic) || len(bogus.Brain) != 4 {
		t.Fatalf("非法 scope 应等价缺省: %+v", bogus)
	}
}
