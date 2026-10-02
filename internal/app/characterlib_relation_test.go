package app

// characterlib_relation_test.go — v4.453「与主角的关系」AI 随机生成链路：
// 批量范围矩阵（全部/剩余全部/个人）+ 主角锚点段 + 无主角指路。
// 打桩方式对齐 plot_branch_handler_test.go：SaveEngine 注册 herdsman 引擎指向
// httptest server，SSE 分片回放并捕获 user prompt。

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

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/prompt"
)

type relationEnv struct {
	a        *App
	lib      *characterlib.Store
	calls    *int32
	mu       sync.Mutex
	lastUser string
}

func (e *relationEnv) userPrompt() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastUser
}

func newRelationEnv(t *testing.T, reply string) *relationEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	var calls int32
	env := &relationEnv{calls: &calls}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
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

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"
	// characterGenerate 走 characterlib 功能绑定（未配置回退全局引擎会撞真实登录态）
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
	lib := characterlib.NewStore(filepath.Join(t.TempDir(), "charlib"))
	if lib == nil {
		t.Fatalf("角色库初始化失败")
	}
	t.Cleanup(func() { _ = lib.Close() })
	a.charLib = lib
	env.a = a
	env.lib = lib
	return env
}

// mustUpsert 测试种子角色。
func mustUpsert(t *testing.T, lib *characterlib.Store, c characterlib.Character) {
	t.Helper()
	if c.CreatedAt == "" {
		c.CreatedAt = "2026-10-02T00:00:00Z"
	}
	c.UpdatedAt = c.CreatedAt
	if err := lib.Upsert(&c); err != nil {
		t.Fatalf("种子角色 %s: %v", c.Name, err)
	}
}

// seedRelationCast 标准四角色：主角林晚 / 沈砚（无关系）/ 苏烈（已有「宿敌」）/ 助手小笺。
func seedRelationCast(t *testing.T, lib *characterlib.Store) {
	t.Helper()
	mustUpsert(t, lib, characterlib.Character{
		ID: "c_proto", Name: "林晚", RoleType: "protagonist", Personality: "冷静自持，外冷内热",
	})
	mustUpsert(t, lib, characterlib.Character{ID: "c_a", Name: "沈砚"})
	mustUpsert(t, lib, characterlib.Character{ID: "c_b", Name: "苏烈", ProtagonistRelation: "宿敌"})
	mustUpsert(t, lib, characterlib.Character{ID: "c_as", Name: "小笺", Kind: characterlib.KindAssistant})
}

const relationReply = `{"protagonist_relation":"结拜兄妹"}`

// TestCharacterGenerateProtagonistRelations_NoProtagonist 无主角卡诚实报错指路
// （模型调用 0 次）。
func TestCharacterGenerateProtagonistRelations_NoProtagonist(t *testing.T) {
	env := newRelationEnv(t, relationReply)
	_, err := env.a.CharacterGenerateProtagonistRelations("missing", "")
	if err == nil || !strings.Contains(err.Error(), "未找到主角") {
		t.Fatalf("应报未找到主角指路错误: %v", err)
	}
	if n := atomic.LoadInt32(env.calls); n != 0 {
		t.Fatalf("模型调用应为 0 次: %d", n)
	}
}

// TestCharacterGenerateProtagonistRelations_Matrix 范围矩阵：missing 只补空白 /
// all 覆盖除主角与助手外全部 / one 点名单个 / 主角本人友好报错 / 未知范围拒绝。
func TestCharacterGenerateProtagonistRelations_Matrix(t *testing.T) {
	t.Run("missing 剩余全部：只补空白的沈砚，已有关系的苏烈与助手不动", func(t *testing.T) {
		env := newRelationEnv(t, relationReply)
		seedRelationCast(t, env.lib)
		res, err := env.a.CharacterGenerateProtagonistRelations("missing", "")
		if err != nil {
			t.Fatalf("missing 生成失败: %v", err)
		}
		if res["updated"] != 1 || res["total"] != 1 {
			t.Fatalf("missing 应只更新 1 位: %+v", res)
		}
		if !strings.Contains(env.userPrompt(), "主角=「林晚」") || !strings.Contains(env.userPrompt(), "已用关系：宿敌") {
			t.Fatalf("prompt 应携带主角锚点与已用关系避重:\n%s", env.userPrompt())
		}
		sy, _ := env.lib.Get("c_a")
		if sy == nil || sy.ProtagonistRelation != "结拜兄妹" {
			t.Fatalf("沈砚关系应写入: %+v", sy)
		}
		su, _ := env.lib.Get("c_b")
		if su == nil || su.ProtagonistRelation != "宿敌" {
			t.Fatalf("苏烈已有关系不得被覆盖: %+v", su)
		}
		if n := atomic.LoadInt32(env.calls); n != 1 {
			t.Fatalf("应恰好调用一次模型: %d", n)
		}
	})

	t.Run("all 全部：除主角与助手外覆盖重写（2 位）", func(t *testing.T) {
		env := newRelationEnv(t, relationReply)
		seedRelationCast(t, env.lib)
		res, err := env.a.CharacterGenerateProtagonistRelations("all", "")
		if err != nil {
			t.Fatalf("all 生成失败: %v", err)
		}
		if res["updated"] != 2 {
			t.Fatalf("all 应更新 2 位（沈砚+苏烈）: %+v", res)
		}
		su, _ := env.lib.Get("c_b")
		if su == nil || su.ProtagonistRelation != "结拜兄妹" {
			t.Fatalf("all 模式苏烈应被覆盖: %+v", su)
		}
		proto, _ := env.lib.Get("c_proto")
		if proto.ProtagonistRelation != "" {
			t.Fatalf("主角本人不参与: %+v", proto)
		}
		as, _ := env.lib.Get("c_as")
		if as.ProtagonistRelation != "" {
			t.Fatalf("助手人格不参与: %+v", as)
		}
		if n := atomic.LoadInt32(env.calls); n != 2 {
			t.Fatalf("应恰好调用两次模型: %d", n)
		}
	})

	t.Run("one 个人：点名单个；主角本人友好报错；未知角色报错", func(t *testing.T) {
		env := newRelationEnv(t, relationReply)
		seedRelationCast(t, env.lib)
		res, err := env.a.CharacterGenerateProtagonistRelations("one", "苏烈")
		if err != nil {
			t.Fatalf("one 生成失败: %v", err)
		}
		if res["updated"] != 1 {
			t.Fatalf("one 应更新 1 位: %+v", res)
		}
		if _, err := env.a.CharacterGenerateProtagonistRelations("one", "林晚"); err == nil ||
			!strings.Contains(err.Error(), "本人即主角") {
			t.Fatalf("主角本人应友好报错: %v", err)
		}
		if _, err := env.a.CharacterGenerateProtagonistRelations("one", "不存在"); err == nil ||
			!strings.Contains(err.Error(), "未找到角色") {
			t.Fatalf("未知角色应报错: %v", err)
		}
		if _, err := env.a.CharacterGenerateProtagonistRelations("whatever", ""); err == nil ||
			!strings.Contains(err.Error(), "未知范围") {
			t.Fatalf("未知范围应报错: %v", err)
		}
	})
}

// TestProtagonistRelationBlock 主角锚点段的三态：有主角=锚+避重；本人=空串指令；
// 无主角=虚构指路。
func TestProtagonistRelationBlock(t *testing.T) {
	env := newRelationEnv(t, relationReply)
	seedRelationCast(t, env.a.charLib)

	if got := env.a.protagonistRelationBlock("沈砚"); !strings.Contains(got, "主角=「林晚」") ||
		!strings.Contains(got, "已用关系：宿敌") {
		t.Fatalf("他者应携带主角锚点+避重: %s", got)
	}
	if got := env.a.protagonistRelationBlock("林晚"); !strings.Contains(got, "输出空字符串") {
		t.Fatalf("主角本人应输出空串指令: %s", got)
	}

	env2 := newRelationEnv(t, relationReply) // 无种子 = 无主角
	if got := env2.a.protagonistRelationBlock("沈砚"); !strings.Contains(got, "暂无主角") {
		t.Fatalf("无主角应走虚构指路: %s", got)
	}
}
