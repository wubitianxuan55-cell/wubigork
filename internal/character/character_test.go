package character

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// capturingClient 记录 ChatSimpleStreamWithOptions 参数，其余接口方法返回桩。
type capturingClient struct {
	model, engine, system, user string
}

func (c *capturingClient) ChatSimpleStreamWithOptions(_ context.Context, model, systemPrompt, userMsg string, opts ai.ChatSimpleOptions) (string, error) {
	c.model = model
	c.engine = opts.EngineID
	c.system = systemPrompt
	c.user = userMsg
	return `{"characters": []}`, nil
}

func (c *capturingClient) ChatStream(context.Context, *ai.ChatRequest) (<-chan ai.SSEChunk, error) {
	return nil, nil
}

func (c *capturingClient) ChatSimpleStream(context.Context, string, string, string) (string, error) {
	return "", nil
}

func (c *capturingClient) GenerateImage(context.Context, *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	return nil, nil
}

// newTestAgent 构造角色 Agent：临时项目 + 临时模板引擎 + 记录型 client。
func newTestAgent(t *testing.T) (*Agent, *capturingClient, *project.Manager) {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试小说", "奇幻", "默认", "")
	if err != nil {
		t.Fatalf("project.Create: %v", err)
	}
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{
			ID: "mc", Name: "林晚", RoleType: "protagonist", Status: "Alive",
			Personality: "冷静", Appearance: "黑发",
		}},
		Organizations: []types.Organization{},
		Relationships: []types.Relationship{},
	}); err != nil {
		t.Fatalf("WriteCharacters: %v", err)
	}

	// 最小 character-agent 模板（真实 RTCO 加载器）
	tplDir := t.TempDir()
	tpl := map[string]any{
		"name":   "character-agent",
		"system": "你是角色创作助手",
		"task":   "创作或修改角色",
		"input_sections": map[string]any{
			"user_request":        map[string]any{"priority": "P0", "label": "用户请求"},
			"existing_characters": map[string]any{"priority": "P1", "label": "现有角色"},
		},
		"output":      map[string]any{"format": "json", "description": "角色 JSON"},
		"constraints": map[string]any{"must": []string{"输出 JSON"}, "forbidden": []string{}},
	}
	data, _ := json.Marshal(tpl)
	if err := os.WriteFile(filepath.Join(tplDir, "character-agent.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	detailTpl := map[string]any{
		"name":   "character-detail",
		"system": "你是角色编辑助手",
		"task":   "针对单个角色对话",
		"input_sections": map[string]any{
			"target_character": map[string]any{"priority": "P0", "label": "目标角色"},
			"worldview":        map[string]any{"priority": "P1", "label": "世界观"},
			"user_request":     map[string]any{"priority": "P2", "label": "用户请求"},
		},
		"output":      map[string]any{"format": "json", "description": "角色 JSON"},
		"constraints": map[string]any{"must": []string{"输出 JSON"}, "forbidden": []string{}},
	}
	detailData, _ := json.Marshal(detailTpl)
	if err := os.WriteFile(filepath.Join(tplDir, "character-detail.json"), detailData, 0644); err != nil {
		t.Fatal(err)
	}

	client := &capturingClient{}
	cfg := &config.Config{Model: "grok-4.20"}
	return New(client, pm, cfg, prompt.NewEngine(tplDir)), client, pm
}

// ── E02：角色注入 prompt 不得夹带剧照 base64 ────────────────

func TestCharacterChat_StripsPortraitBase64(t *testing.T) {
	agent, client, pm := newTestAgent(t)
	// 给角色挂上 base64 剧照（ComfyUI 生成后存为 data URL）
	cf, _ := pm.ReadCharacters()
	cf.Characters[0].PortraitURL = "data:image/png;base64,SENTINEL_B64_9f8e7d6c5b4a3210"
	if err := pm.WriteCharacters(cf); err != nil {
		t.Fatal(err)
	}

	if _, err := agent.Chat(context.Background(), "完善这个角色"); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	promptText := client.system + "\n" + client.user
	if strings.Contains(promptText, "SENTINEL_B64_9f8e7d6c5b4a3210") {
		t.Fatal("E02 回归：prompt 夹带剧照 base64")
	}
	if strings.Contains(promptText, "data:image") {
		t.Fatal("E02 回归：prompt 含 data URL 图片")
	}
	if strings.Contains(promptText, "portrait_url") {
		t.Fatal("E02 回归：prompt 含 portrait_url 字段")
	}
	if !strings.Contains(promptText, "林晚") {
		t.Error("角色名应保留在 prompt 中")
	}
}

func TestCharacterChatDetail_StripsPortraitBase64(t *testing.T) {
	agent, client, pm := newTestAgent(t)
	cf, _ := pm.ReadCharacters()
	cf.Characters[0].PortraitURL = "data:image/png;base64,DETAIL_SENTINEL_1234567890"
	if err := pm.WriteCharacters(cf); err != nil {
		t.Fatal(err)
	}

	if _, err := agent.ChatCharacterDetail(context.Background(), "mc", "描述外貌"); err != nil {
		t.Fatalf("ChatCharacterDetail: %v", err)
	}
	promptText := client.system + "\n" + client.user
	if strings.Contains(promptText, "DETAIL_SENTINEL_1234567890") || strings.Contains(promptText, "data:image") {
		t.Fatal("E02 回归：角色详情 prompt 夹带剧照 base64")
	}
}

// ── E03：未绑定时不得强制全局 cfg.Model ────────────────────

func TestCharacterChat_UnboundNovelResolvesByEngine(t *testing.T) {
	agent, client, _ := newTestAgent(t)
	// 未绑定 novel 功能模型，cfg.Model 是 xAI 陈旧默认
	if _, err := agent.Chat(context.Background(), "完善角色"); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if client.model != "" {
		t.Errorf("E03 回归：未绑定时 model = %q，应留空让客户端按引擎解析（避免把全局模型发给非 xAI 引擎）", client.model)
	}
	if client.engine != "" {
		t.Errorf("engine = %q, want 空（全局活跃引擎）", client.engine)
	}
}

func TestCharacterChat_BoundNovelUsesBinding(t *testing.T) {
	agent, client, _ := newTestAgent(t)
	agent.cfg.SetFeatureModel("novel", "herdsman", "qwen3-8b")
	if _, err := agent.Chat(context.Background(), "完善角色"); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if client.model != "qwen3-8b" || client.engine != "herdsman" {
		t.Errorf("绑定后 model/engine = (%q,%q), want (qwen3-8b,herdsman)", client.model, client.engine)
	}
}

// TestSetRemoveCharacterCareer 职业绑定矩阵（t5 第四刀 §7.6）：
// 主职业设置/替换（手工意图明确，与差分器拒绝替换语义有意不同）/
// 副职业上限 2/同名幂等更新/阶段钳 ≥1/移除/角色不存在。
func TestSetRemoveCharacterCareer(t *testing.T) {
	a, _, pm := newTestAgent(t)
	must := func(err error, what string) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	}

	// 设主职业：阶段 <1 钳 1
	must(a.SetCharacterCareer("mc", `{"is_main":true,"career_name":"剑修","stage":0}`), "设主职业")
	cf, _ := pm.ReadCharacters()
	if cf.Characters[0].MainCareerID != "剑修" || cf.Characters[0].MainCareerStage != 1 {
		t.Fatalf("主职业未落/未钳: %+v", cf.Characters[0])
	}

	// 替换主职业（手工 UI 允许）
	must(a.SetCharacterCareer("mc", `{"is_main":true,"career_name":"刀客","stage":3}`), "替换主职业")
	cf, _ = pm.ReadCharacters()
	if cf.Characters[0].MainCareerID != "刀客" || cf.Characters[0].MainCareerStage != 3 {
		t.Fatalf("主职业替换未落: %+v", cf.Characters[0])
	}

	// 副职业：加两个 + 同名幂等更新
	must(a.SetCharacterCareer("mc", `{"is_main":false,"career_name":"炼丹师","stage":2}`), "副职业1")
	must(a.SetCharacterCareer("mc", `{"is_main":false,"career_name":"阵法师","stage":1}`), "副职业2")
	must(a.SetCharacterCareer("mc", `{"is_main":false,"career_name":"炼丹师","stage":4}`), "副职业同名更新")
	cf, _ = pm.ReadCharacters()
	subs := cf.Characters[0].SubCareers
	if len(subs) != 2 {
		t.Fatalf("副职业应 2 个: %+v", subs)
	}
	if subs[0].CareerID != "炼丹师" || subs[0].Stage != 4 || subs[0].CareerName != "炼丹师" {
		t.Fatalf("同名更新未幂等: %+v", subs[0])
	}

	// 上限拒绝
	if err := a.SetCharacterCareer("mc", `{"is_main":false,"career_name":"御兽师","stage":1}`); err == nil {
		t.Fatal("第三个副职业应被上限拒绝")
	}

	// 空名称拒绝
	if err := a.SetCharacterCareer("mc", `{"is_main":true,"career_name":"  ","stage":1}`); err == nil {
		t.Fatal("空职业名应拒绝")
	}

	// 移除：主职业 + 副职业 + 不存在副职业报错
	must(a.RemoveCharacterCareer("mc", `{"is_main":false,"career_name":"阵法师"}`), "移除副职业")
	must(a.RemoveCharacterCareer("mc", `{"is_main":true}`), "移除主职业")
	cf, _ = pm.ReadCharacters()
	if cf.Characters[0].MainCareerID != "" || len(cf.Characters[0].SubCareers) != 1 {
		t.Fatalf("移除未落: %+v %+v", cf.Characters[0].MainCareerID, cf.Characters[0].SubCareers)
	}
	if err := a.RemoveCharacterCareer("mc", `{"is_main":false,"career_name":"不存在"}`); err == nil {
		t.Fatal("移除不存在的副职业应报错")
	}

	// 角色不存在
	if err := a.SetCharacterCareer("ghost", `{"is_main":true,"career_name":"剑修","stage":1}`); err == nil {
		t.Fatal("不存在的角色应报错")
	}
}
