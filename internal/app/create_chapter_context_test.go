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

// newContextTestProject 创建章节上下文增强测试用项目目录
func newContextTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "上下文增强测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	return pm
}

// TestChapterContextInjectsForeshadowsAndWorldview 未回收伏笔与世界观点必须进入
// 增强区段：planted/hinted 注入，revealed 过滤，空维度跳过。
func TestChapterContextInjectsForeshadowsAndWorldview(t *testing.T) {
	pm := newContextTestProject(t)

	longDesc := strings.Repeat("谜", 300) // 超长描述应被截断
	if err := pm.WriteForeshadows(&types.ForeshadowFile{Items: []types.Foreshadow{
		{ID: "f1", Category: "plot", Description: "主角背上有一道古剑胎记", PlantedIn: "001.md", Status: types.ForeshadowPlanted, IsLongTerm: true},
		{ID: "f2", Category: "character", Description: "酒馆老板总在打听商队路线", PlantedIn: "002.md", Status: types.ForeshadowHinted},
		{ID: "f3", Category: "world", Description: "北境冰湖下封印的旧神", PlantedIn: "001.md", RevealedIn: "003.md", Status: types.ForeshadowRevealed},
		{ID: "f4", Category: "plot", Description: longDesc, PlantedIn: "002.md", Status: types.ForeshadowPlanted},
	}}); err != nil {
		t.Fatalf("写入伏笔: %v", err)
	}
	if err := pm.WriteWorldviewFile(&types.WorldviewFile{Sections: []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: "灵气复苏三百年，九州大陆王朝更迭频繁", Order: 1},
		{ID: "geography", Title: "地理风貌", Content: "", Order: 2}, // 空维度应跳过
		{ID: "rules", Title: "规则体系", Content: "修士以灵石为基，剑修一脉以心御剑", Order: 4},
	}}); err != nil {
		t.Fatalf("写入世界观: %v", err)
	}

	got := buildChapterContextSections(pm)
	if got == "" {
		t.Fatalf("有伏笔和世界观数据时增强区段不应为空")
	}
	for _, want := range []string{
		"## 未回收伏笔（创作约束）",
		"不得与之矛盾",
		"古剑胎记",
		"已埋设",
		"（长线）",
		"已暗示·进行中",
		"## 世界观要点",
		"- 时代背景：灵气复苏三百年",
		"- 规则体系：修士以灵石为基",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("增强区段缺少 %q，实际为:\n%s", want, got)
		}
	}
	if strings.Contains(got, "旧神") {
		t.Errorf("已回收（revealed）伏笔不应注入:\n%s", got)
	}
	if strings.Contains(got, "- 地理风貌：") {
		t.Errorf("空维度不应注入:\n%s", got)
	}
	if !strings.Contains(got, "...") {
		t.Errorf("超长伏笔描述应被截断（缺省略号）:\n%s", got)
	}
}

// TestChapterContextSkipsWhenEmptyOrBroken 无数据/文件损坏时必须静默降级：
// 不注入空区段，不报错，不 panic。
func TestChapterContextSkipsWhenEmptyOrBroken(t *testing.T) {
	pm := newContextTestProject(t)

	// 新建项目：foreshadows.json 为空、worldview.json 六维度全空 → 不注入任何区段
	if got := buildChapterContextSections(pm); got != "" {
		t.Errorf("空项目不应注入增强区段，got:\n%s", got)
	}

	// foreshadows.json / worldview.json 损坏 → 仍静默跳过
	if err := os.WriteFile(filepath.Join(pm.Dir, "foreshadows.json"), []byte("{broken json"), 0644); err != nil {
		t.Fatalf("写损坏伏笔文件: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pm.Dir, "worldview.json"), []byte("not json at all"), 0644); err != nil {
		t.Fatalf("写损坏世界观文件: %v", err)
	}
	if got := buildChapterContextSections(pm); got != "" {
		t.Errorf("文件损坏时应静默跳过，got:\n%s", got)
	}
}

// TestChapterContextLegacyWorldviewMarkdown 旧 worldview.md（无 worldview.json）
// 必须兜底注入，不报错。
func TestChapterContextLegacyWorldviewMarkdown(t *testing.T) {
	pm := newContextTestProject(t)
	if err := os.Remove(filepath.Join(pm.Dir, "worldview.json")); err != nil {
		t.Fatalf("删除 worldview.json: %v", err)
	}
	legacy := "天下九州，以剑为尊，凡人皆可开脉。"
	if err := os.WriteFile(filepath.Join(pm.Dir, "worldview.md"), []byte(legacy), 0644); err != nil {
		t.Fatalf("写旧版 worldview.md: %v", err)
	}
	got := buildChapterContextSections(pm)
	if !strings.Contains(got, "## 世界观要点") {
		t.Errorf("旧 md 世界观应注入世界观要点区段:\n%s", got)
	}
	if !strings.Contains(got, "天下九州，以剑为尊") {
		t.Errorf("旧 md 世界观内容丢失:\n%s", got)
	}
}

// TestChapterContextBudgetTruncation 总预算与分段截断必须生效：
// 合计 ≤ ctxBudgetTotal；伏笔条数封顶；超长内容被截断。
func TestChapterContextBudgetTruncation(t *testing.T) {
	// joinWithBudget：两段各 5000 rune → 合计必须压回 4000 内
	joined := joinWithBudget(ctxBudgetTotal,
		strings.Repeat("甲", 5000), strings.Repeat("乙", 5000))
	if n := len([]rune(joined)); n > ctxBudgetTotal {
		t.Errorf("joinWithBudget 超预算: got %d rune, want <= %d", n, ctxBudgetTotal)
	}
	if !strings.Contains(joined, "...") {
		t.Errorf("joinWithBudget 截断应留省略号:\n%s", joined[:80])
	}

	// 伏笔：条数封顶 ctxForeshadowMaxItems，整体不超过 ctxForeshadowBudget
	pm := newContextTestProject(t)
	items := make([]types.Foreshadow, 0, 40)
	for i := 0; i < 40; i++ {
		items = append(items, types.Foreshadow{
			ID: "f", Category: "plot", Description: strings.Repeat("伏", 200),
			PlantedIn: "001.md", Status: types.ForeshadowPlanted,
		})
	}
	if err := pm.WriteForeshadows(&types.ForeshadowFile{Items: items}); err != nil {
		t.Fatalf("写入伏笔: %v", err)
	}
	fs := buildForeshadowSection(pm)
	if n := strings.Count(fs, "- ["); n > ctxForeshadowMaxItems {
		t.Errorf("伏笔条数未封顶: got %d, want <= %d", n, ctxForeshadowMaxItems)
	}
	if n := len([]rune(fs)); n > ctxForeshadowBudget {
		t.Errorf("伏笔区超预算: got %d rune, want <= %d", n, ctxForeshadowBudget)
	}

	// 世界观：单维度 5000 rune → 整区不超过 ctxWorldviewBudget
	if err := pm.WriteWorldviewFile(&types.WorldviewFile{Sections: []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: strings.Repeat("史", 5000), Order: 1},
	}}); err != nil {
		t.Fatalf("写入世界观: %v", err)
	}
	ws := buildWorldviewSection(pm)
	if n := len([]rune(ws)); n > ctxWorldviewBudget {
		t.Errorf("世界观区超预算: got %d rune, want <= %d", n, ctxWorldviewBudget)
	}

	// 合计：两项同时超大 → buildChapterContextSections 不超过总预算
	if n := len([]rune(buildChapterContextSections(pm))); n > ctxBudgetTotal {
		t.Errorf("增强区段合计超总预算: got %d rune, want <= %d", n, ctxBudgetTotal)
	}
}

// TestChapterContextCharacterCardEnhanced 角色卡增强：性格不再截 20 字，
// 补身份/目标/关系要点；无角色时保持原降级文案。
func TestChapterContextCharacterCardEnhanced(t *testing.T) {
	pm := newContextTestProject(t)
	a := &writingState{}

	// 无角色：保持原降级文案
	if got := a.buildCharacterSummary(pm); got != "（暂无角色设定）" {
		t.Errorf("无角色时应返回（暂无角色设定），got: %q", got)
	}

	personality := strings.Repeat("冷", 50) // 50 rune：旧实现截到 20，新上限 60 不应截断
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Personality: personality,
				Background: "没落剑宗唯一传人", Motivation: "重铸剑心，为师复仇", Status: "Alive"},
			{ID: "c2", Name: "沈青", RoleType: "antagonist", Personality: "阴鸷多疑", Status: "Alive"},
		},
		Relationships: []types.Relationship{
			{FromID: "c1", ToID: "c2", RelationType: "enemy", Description: "夺剑之仇不共戴天"},
		},
	}); err != nil {
		t.Fatalf("写入角色: %v", err)
	}

	got := a.buildCharacterSummary(pm)
	for _, want := range []string{
		"- 林晚：主角·" + personality, // 性格 50 rune 完整保留（回归：原 20 字截断）
		"身份：没落剑宗唯一传人",
		"目标：重铸剑心，为师复仇",
		"关系：与沈青为敌对（夺剑之仇不共戴天）",
		"·Alive",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("角色摘要缺少 %q，实际为:\n%s", want, got)
		}
	}

	// 性格超上限（100 rune）时按 60 截断
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist",
				Personality: strings.Repeat("冷", 100), Status: "Alive"},
		},
	}); err != nil {
		t.Fatalf("写入角色: %v", err)
	}
	got = a.buildCharacterSummary(pm)
	if strings.Contains(got, strings.Repeat("冷", 100)) {
		t.Errorf("性格超上限应被截断:\n%s", got)
	}
	if !strings.Contains(got, strings.Repeat("冷", 60)) {
		t.Errorf("性格截断后应保留前 60 rune:\n%s", got)
	}
}

// TestCreateChapterPromptIncludesEnhancedContext 端到端：CreateChapter 发给 AI 的
// user prompt 必须包含未回收伏笔、世界观要点与增强后的角色卡字段。
func TestCreateChapterPromptIncludesEnhancedContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	// 捕获第一个请求体（初始生成 prompt，非续写 prompt）
	var mu sync.Mutex
	var firstBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		if firstBody == nil {
			firstBody = body
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"第一章正文"},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`)
	}))
	defer srv.Close()

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

	pm := newContextTestProject(t)
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Personality: "清冷坚韧",
				Background: "没落剑宗唯一传人", Motivation: "重铸剑心，为师复仇", Status: "Alive"},
			{ID: "c2", Name: "沈青", RoleType: "antagonist", Personality: "阴鸷多疑", Status: "Alive"},
		},
		Relationships: []types.Relationship{
			{FromID: "c1", ToID: "c2", RelationType: "enemy", Description: "夺剑之仇不共戴天"},
		},
	}); err != nil {
		t.Fatalf("写入角色: %v", err)
	}
	if err := pm.WriteForeshadows(&types.ForeshadowFile{Items: []types.Foreshadow{
		{ID: "f1", Category: "plot", Description: "主角背上有一道古剑胎记", PlantedIn: "001.md", Status: types.ForeshadowPlanted},
	}}); err != nil {
		t.Fatalf("写入伏笔: %v", err)
	}
	if err := pm.WriteWorldviewFile(&types.WorldviewFile{Sections: []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: "灵气复苏三百年，九州大陆王朝更迭频繁", Order: 1},
	}}); err != nil {
		t.Fatalf("写入世界观: %v", err)
	}

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.setPM(pm)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	a.ctx = ctx

	if _, err := a.CreateChapter("九州大陆，灵气复苏。", "", "主角踏上旅途", 1, "", "", 1200, 0); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	var body []byte
	for {
		mu.Lock()
		body = firstBody
		mu.Unlock()
		if body != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("未捕获到 AI 请求")
		}
		time.Sleep(20 * time.Millisecond)
	}

	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("解析请求体失败: %v", err)
	}
	var userPrompt string
	for _, m := range req.Messages {
		if m.Role == "user" {
			userPrompt = m.Content
		}
	}
	if userPrompt == "" {
		t.Fatalf("请求中缺少 user 消息")
	}
	for _, want := range []string{
		"小说设定",           // 既有锚点不破坏
		"## 未回收伏笔（创作约束）", // 伏笔注入
		"古剑胎记",
		"## 世界观要点", // 世界观注入
		"- 时代背景：灵气复苏三百年",
		"目标：重铸剑心，为师复仇", // 角色卡增强
		"与沈青为敌对",
	} {
		if !strings.Contains(userPrompt, want) {
			t.Errorf("user prompt 缺少 %q，实际为:\n%s", want, userPrompt)
		}
	}
}
