package app

// character_relation_project_test.go — v4.454 小说项目角色「与主角的关系」：
// 范围矩阵（全部/剩余全部/个人）+ 无主角指路 + 名册消费 + 项目级字段两路保护
// （AI 补全不丢、库→项目同步不丢）。打桩方式对齐 plot_branch_handler_test.go。

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"context"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/character"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

type projRelationEnv struct {
	a        *App
	pm       *project.Manager
	lib      *characterlib.Store
	calls    *int32
	mu       sync.Mutex
	lastUser string
}

func (e *projRelationEnv) userPrompt() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastUser
}

func newProjRelationEnv(t *testing.T, replies ...string) *projRelationEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	var calls int32
	env := &projRelationEnv{calls: &calls}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := int(atomic.AddInt32(&calls, 1)) - 1
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
				env.mu.Lock()
				env.lastUser = m.Content
				env.mu.Unlock()
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
	// characterGenerate（AI 补全链）走 characterlib 功能绑定，缺省会撞真实登录态
	cfg.SetFeatureModel("characterlib", "herdsman", "test-model")

	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}
	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	a.ctx = ctx
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "关系测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	// GenerateProjectCharacterFill 走 characterAgent 保存链
	a.characterAgent = character.New(nil, pm, cfg, nil)
	lib := characterlib.NewStore(filepath.Join(t.TempDir(), "charlib"))
	if lib == nil {
		t.Fatalf("角色库初始化失败")
	}
	t.Cleanup(func() { _ = lib.Close() })
	a.charLib = lib
	env.a = a
	env.pm = pm
	env.lib = lib
	return env
}

// seedProjCast 标准四角色：主角林晚 / 沈砚（无关系）/ 苏烈（已有「宿敌」）/ 龙套阿七。
func seedProjCast(t *testing.T, pm *project.Manager) {
	t.Helper()
	cf := &types.CharacterFile{Characters: []types.Character{
		{ID: "p1", Name: "林晚", RoleType: "protagonist", Status: "Alive", Personality: "冷静自持，外冷内热"},
		{ID: "p2", Name: "沈砚", RoleType: "supporting", Status: "Alive"},
		{ID: "p3", Name: "苏烈", RoleType: "antagonist", Status: "Alive", ProtagonistRelation: "宿敌"},
		{ID: "p4", Name: "阿七", RoleType: "minor", Status: "Alive"},
	}}
	if err := pm.WriteCharacters(cf); err != nil {
		t.Fatalf("种子角色: %v", err)
	}
}

const projRelationReply = `{"relation":"结拜兄妹"}`

// TestGenerateProjectProtagonistRelations_NoProtagonist 无主角卡诚实指路（0 模型调用）。
func TestGenerateProjectProtagonistRelations_NoProtagonist(t *testing.T) {
	env := newProjRelationEnv(t, projRelationReply)
	if err := env.pm.WriteCharacters(&types.CharacterFile{Characters: []types.Character{
		{ID: "p2", Name: "沈砚", RoleType: "supporting", Status: "Alive"},
	}}); err != nil {
		t.Fatalf("种子角色: %v", err)
	}
	if _, err := env.a.GenerateProjectProtagonistRelations("missing", ""); err == nil ||
		!strings.Contains(err.Error(), "未找到主角") {
		t.Fatalf("应报未找到主角指路错误: %v", err)
	}
	if n := atomic.LoadInt32(env.calls); n != 0 {
		t.Fatalf("模型调用应为 0 次: %d", n)
	}
}

// TestGenerateProjectProtagonistRelations_Matrix 范围矩阵与只写本书纪律。
func TestGenerateProjectProtagonistRelations_Matrix(t *testing.T) {
	t.Run("missing 剩余全部：只补沈砚与阿七，苏烈已有关系不动", func(t *testing.T) {
		env := newProjRelationEnv(t, projRelationReply)
		seedProjCast(t, env.pm)
		res, err := env.a.GenerateProjectProtagonistRelations("missing", "")
		if err != nil {
			t.Fatalf("missing 失败: %v", err)
		}
		if res["updated"] != 2 || res["total"] != 2 {
			t.Fatalf("missing 应更新 2 位: %+v", res)
		}
		if !strings.Contains(env.userPrompt(), "主角设定") || !strings.Contains(env.userPrompt(), "宿敌") {
			t.Fatalf("prompt 应携带主角锚点与已用关系避重:\n%s", env.userPrompt())
		}
		cf, _ := env.pm.ReadCharacters()
		byName := map[string]types.Character{}
		for _, c := range cf.Characters {
			byName[c.Name] = c
		}
		if byName["沈砚"].ProtagonistRelation != "结拜兄妹" || byName["阿七"].ProtagonistRelation != "结拜兄妹" {
			t.Fatalf("沈砚/阿七应写入: %+v %+v", byName["沈砚"], byName["阿七"])
		}
		if byName["苏烈"].ProtagonistRelation != "宿敌" {
			t.Fatalf("苏烈已有关系不得覆盖: %+v", byName["苏烈"])
		}
		if byName["林晚"].ProtagonistRelation != "" {
			t.Fatalf("主角本人不参与: %+v", byName["林晚"])
		}
	})

	t.Run("all 全部：除主角外覆盖重写（3 位）", func(t *testing.T) {
		env := newProjRelationEnv(t, projRelationReply)
		seedProjCast(t, env.pm)
		res, err := env.a.GenerateProjectProtagonistRelations("all", "")
		if err != nil {
			t.Fatalf("all 失败: %v", err)
		}
		if res["updated"] != 3 {
			t.Fatalf("all 应更新 3 位: %+v", res)
		}
		if n := atomic.LoadInt32(env.calls); n != 3 {
			t.Fatalf("应恰好调用三次模型: %d", n)
		}
	})

	t.Run("one 个人：点名/主角本人友好报错/未知角色/未知范围", func(t *testing.T) {
		env := newProjRelationEnv(t, projRelationReply)
		seedProjCast(t, env.pm)
		res, err := env.a.GenerateProjectProtagonistRelations("one", "阿七")
		if err != nil {
			t.Fatalf("one 失败: %v", err)
		}
		if res["updated"] != 1 {
			t.Fatalf("one 应更新 1 位: %+v", res)
		}
		if _, err := env.a.GenerateProjectProtagonistRelations("one", "林晚"); err == nil ||
			!strings.Contains(err.Error(), "本人即主角") {
			t.Fatalf("主角本人应友好报错: %v", err)
		}
		if _, err := env.a.GenerateProjectProtagonistRelations("one", "不存在"); err == nil ||
			!strings.Contains(err.Error(), "未找到角色") {
			t.Fatalf("未知角色应报错: %v", err)
		}
		if _, err := env.a.GenerateProjectProtagonistRelations("whatever", ""); err == nil ||
			!strings.Contains(err.Error(), "未知范围") {
			t.Fatalf("未知范围应报错: %v", err)
		}
	})
}

// TestProjectProtagonistRelationConsumption 生成结果注入章节生成角色名册。
func TestProjectProtagonistRelationConsumption(t *testing.T) {
	env := newProjRelationEnv(t, projRelationReply)
	seedProjCast(t, env.pm)
	if _, err := env.a.GenerateProjectProtagonistRelations("missing", ""); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	summary := env.a.buildCharacterSummary(env.pm)
	if !strings.Contains(summary, "与主角：结拜兄妹") {
		t.Fatalf("名册应携带与主角关系:\n%s", summary)
	}
	// 主角本人与未生成者不渲染该段
	if strings.Contains(summary, "林晚：主角·与主角") {
		t.Fatalf("主角本人不应有名册关系段:\n%s", summary)
	}
}

// TestProjectRelationPreservedOnFill AI 补全（库投影链路）不得清掉项目级关系字段。
func TestProjectRelationPreservedOnFill(t *testing.T) {
	env := newProjRelationEnv(t, `{"name":"苏烈","figure":"高大","behavior_rules":"言出必行","emotion_logic":"记仇但讲理"}`)
	seedProjCast(t, env.pm)
	cf, _ := env.pm.ReadCharacters()
	var su types.Character
	for _, c := range cf.Characters {
		if c.Name == "苏烈" {
			su = c
		}
	}
	out, err := env.a.GenerateProjectCharacterFill(string(mustJSONProj(t, su)))
	if err != nil {
		t.Fatalf("补全失败: %v", err)
	}
	if !strings.Contains(out, "宿敌") {
		t.Fatalf("AI 补全应保留项目级关系字段: %s", out)
	}
}

// TestProjectRelationPreservedOnSync 库→项目同步物化不得清掉项目级关系字段。
func TestProjectRelationPreservedOnSync(t *testing.T) {
	env := newProjRelationEnv(t, projRelationReply)
	seedProjCast(t, env.pm)
	// 全部项目角色入库并关联（同步的防误清护栏要求）
	libChars := []characterlib.Character{
		{ID: "p1", Name: "林晚", RoleType: "protagonist", Status: "Alive"},
		{ID: "p2", Name: "沈砚", RoleType: "supporting", Status: "Alive"},
		{ID: "p3", Name: "苏烈", RoleType: "antagonist", Status: "Alive"},
		{ID: "p4", Name: "阿七", RoleType: "minor", Status: "Alive"},
	}
	for i := range libChars {
		if err := env.lib.Upsert(&libChars[i]); err != nil {
			t.Fatalf("库种子: %v", err)
		}
		if err := env.lib.Associate(env.pm.Dir, libChars[i].ID, "", "", ""); err != nil {
			t.Fatalf("关联: %v", err)
		}
	}
	if err := env.a.CharacterSyncProject(); err != nil {
		t.Fatalf("同步失败: %v", err)
	}
	cf, _ := env.pm.ReadCharacters()
	for _, c := range cf.Characters {
		if c.Name == "苏烈" && c.ProtagonistRelation != "宿敌" {
			t.Fatalf("同步应保留项目级关系字段: %+v", c)
		}
	}
}

// mustJSONProj 测试辅助（mustJSON 与 image_handler_test 重名，改名避让）。
func mustJSONProj(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
