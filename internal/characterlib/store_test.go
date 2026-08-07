package characterlib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/whisper"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "gaea-charlib-test-"+t.Name())
	_ = os.RemoveAll(dir)
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatalf("Store 初始化失败")
	}
	t.Cleanup(func() {
		_ = s.Close()
		_ = os.RemoveAll(dir)
	})
	return s
}

func TestEnsureBuiltins_Idempotent(t *testing.T) {
	s := newTestStore(t)
	presets := []whisper.PersonalityPreset{
		{ID: "gaea", Label: "gaea", Gender: "female", Dims: whisper.PersonalityDims{T: 80, I: 50, S: 30, O: 70, R: 50}, VoiceGuide: "大地"},
		{ID: "tsundere", Label: "傲娇", Gender: "female", Dims: whisper.PersonalityDims{T: 30, I: 50, S: 70, O: 40, R: 50}, VoiceGuide: "嘴硬心软"},
	}
	if err := s.EnsureBuiltins(presets); err != nil {
		t.Fatalf("首次种子化失败: %v", err)
	}
	// 幂等：二次执行不报错、不重复
	if err := s.EnsureBuiltins(presets); err != nil {
		t.Fatalf("二次种子化失败: %v", err)
	}
	c, err := s.Get("gaea")
	if err != nil || c == nil {
		t.Fatalf("Get(gaea) = %v, %v", c, err)
	}
	if c.Kind != KindBuiltin || !c.ChatEnabled || c.Dims.T != 80 {
		t.Fatalf("内置角色字段不符: %+v", c)
	}
	items, total, err := s.List("", KindBuiltin, false, 50, 0)
	if err != nil || total != 2 || len(items) != 2 {
		t.Fatalf("List(builtin) = %d/%d, %v", len(items), total, err)
	}
}

func TestUpsertGetList_SearchFilterPagination(t *testing.T) {
	s := newTestStore(t)
	base := []Character{
		{ID: "a", Name: "林晚", Kind: KindCustom, Gender: "female", Tags: []string{"女主", "剑修"}, ChatEnabled: true, VoiceGuide: "清冷"},
		{ID: "b", Name: "顾长风", Kind: KindCustom, Gender: "male", Tags: []string{"男主"}, ChatEnabled: true, VoiceGuide: "沉稳"},
		{ID: "c", Name: "苏小小", Kind: KindCustom, Gender: "female", ChatEnabled: false, RoleType: "supporting"},
	}
	for _, c := range base {
		if err := s.Upsert(&c); err != nil {
			t.Fatalf("Upsert(%s) 失败: %v", c.ID, err)
		}
	}
	// 搜索名称
	items, total, err := s.List("林", "", false, 50, 0)
	if err != nil || total != 1 || len(items) != 1 || items[0].ID != "a" {
		t.Fatalf("搜索「林」= %d/%d %v", len(items), total, err)
	}
	// 搜索标签
	items, total, _ = s.List("剑修", "", false, 50, 0)
	if total != 1 {
		t.Fatalf("标签搜索 = %d", total)
	}
	// 类型过滤
	items, total, _ = s.List("", KindCustom, false, 50, 0)
	if total != 3 {
		t.Fatalf("custom 过滤 = %d", total)
	}
	// 仅聊天角色
	items, total, _ = s.List("", "", true, 50, 0)
	if total != 2 {
		t.Fatalf("chatOnly = %d", total)
	}
	// 分页
	_, total, _ = s.List("", "", false, 2, 0)
	if total != 3 {
		t.Fatalf("total 应为 3: %d", total)
	}
	page1, _, _ := s.List("", "", false, 2, 0)
	page2, _, _ := s.List("", "", false, 2, 2)
	if len(page1) != 2 || len(page2) != 1 {
		t.Fatalf("分页异常: p1=%d p2=%d", len(page1), len(page2))
	}
}

func TestDelete_BuiltinSoftCustomHard(t *testing.T) {
	s := newTestStore(t)
	_ = s.EnsureBuiltins([]whisper.PersonalityPreset{{ID: "gaea", Label: "gaea", Gender: "female"}})
	if err := s.Delete("gaea"); err != nil {
		t.Fatalf("删除内置失败: %v", err)
	}
	c, _ := s.Get("gaea")
	if c == nil || !c.Hidden {
		t.Fatalf("内置角色应软隐藏: %+v", c)
	}
	_, total, _ := s.List("", "", false, 50, 0)
	if total != 0 {
		t.Fatalf("软隐藏后列表应为空: %d", total)
	}

	custom := Character{ID: "x", Name: "X", Kind: KindCustom}
	_ = s.Upsert(&custom)
	_ = s.Associate("proj1", "x", "protagonist", "黑化", "Alive")
	if err := s.Delete("x"); err != nil {
		t.Fatalf("删除自定义失败: %v", err)
	}
	c, _ = s.Get("x")
	if c != nil {
		t.Fatalf("自定义角色应硬删: %+v", c)
	}
	projects, err := s.ProjectIDsForCharacter("x")
	if err != nil || len(projects) != 0 {
		t.Fatalf("关联应级联清理: %v %v", projects, err)
	}
}

func TestImportProjectCharacters_IdempotentAndDedupe(t *testing.T) {
	s := newTestStore(t)
	chars := []types.Character{
		{ID: "ch_1", Name: "林晚", RoleType: "protagonist", Arc: "崛起", Status: "Alive", Background: "孤儿"},
		{ID: "ch_2", Name: "顾长风", RoleType: "antagonist", Arc: "堕落", Status: "Alive"},
	}
	n, err := s.ImportProjectCharacters("projA", chars)
	if err != nil || n != 2 {
		t.Fatalf("首次导入 = %d, %v", n, err)
	}
	// 幂等重复导入
	n, err = s.ImportProjectCharacters("projA", chars)
	if err != nil || n != 2 {
		t.Fatalf("重复导入 = %d, %v", n, err)
	}
	// 名称去重：另一项目同名角色合并到同一 ID
	other := []types.Character{{ID: "ch_99", Name: "林晚", RoleType: "supporting", Arc: "回归"}}
	n, err = s.ImportProjectCharacters("projB", other)
	if err != nil || n != 1 {
		t.Fatalf("跨项目导入 = %d, %v", n, err)
	}
	c, _ := s.Get("ch_1")
	if c == nil {
		t.Fatalf("应按名称合并到 ch_1")
	}
	if c.Background != "孤儿" {
		t.Fatalf("合并应保留已有小说字段: %+v", c)
	}
	projects, _ := s.ProjectIDsForCharacter("ch_1")
	if len(projects) != 2 {
		t.Fatalf("同一角色应被两个项目引用: %v", projects)
	}
}

func TestProjectCharactersForNovel_MergesPerProjectState(t *testing.T) {
	s := newTestStore(t)
	chars := []types.Character{
		{ID: "ch_1", Name: "林晚", RoleType: "protagonist", Arc: "崛起", Status: "Alive", Personality: "清冷"},
	}
	_, _ = s.ImportProjectCharacters("projA", chars)
	// 项目 A 内弧线推进
	_ = s.Associate("projA", "ch_1", "protagonist", "黑化", "Alive")
	out, err := s.ProjectCharactersForNovel("projA")
	if err != nil || len(out) != 1 {
		t.Fatalf("物化失败: %v %v", out, err)
	}
	if out[0].Arc != "黑化" {
		t.Fatalf("应取项目内弧线状态: %+v", out[0])
	}
	if out[0].Personality != "清冷" {
		t.Fatalf("应保留全局人格字段: %+v", out[0])
	}
	// 未引用项目的角色不在物化结果中
	empty, err := s.ProjectCharactersForNovel("projB")
	if err != nil || len(empty) != 0 {
		t.Fatalf("projB 应无角色: %v %v", empty, err)
	}
}

func TestEnsureAssistants_MirrorsChatConfig(t *testing.T) {
	s := newTestStore(t)
	presets := []whisper.PersonalityPreset{
		{ID: "gaea", Label: "gaea", Gender: "female", Dims: whisper.PersonalityDims{T: 85, I: 55, S: 20, O: 80, R: 50}, VoiceGuide: "大地"},
	}
	asts := []assistant.Assistant{
		{ID: "ast_1", Name: "峨嵋", PersonalityID: "lib_01", VoiceGuide: "仙气", Gender: "female", Dims: whisper.PersonalityDims{T: 60, I: 40, S: 50, O: 60, R: 40}, Enabled: true},
		{ID: "gaea", Name: "gaea", PersonalityID: "gaea", Enabled: true},
	}
	if err := s.EnsureBuiltins(presets); err != nil {
		t.Fatalf("种子化失败: %v", err)
	}
	if err := s.EnsureAssistants(asts, presets); err != nil {
		t.Fatalf("助手同步失败: %v", err)
	}
	c, _ := s.Get("lib_01")
	if c == nil || c.Kind != KindAssistant || !c.ChatEnabled || c.VoiceGuide != "仙气" || c.AssistantID != "ast_1" {
		t.Fatalf("助手角色镜像不符: %+v", c)
	}
	// 无自定义字段的助手回退预设
	c2, _ := s.Get("gaea")
	if c2 == nil || c2.Dims.T != 85 || c2.VoiceGuide != "大地" {
		t.Fatalf("gaea 助手应回退预设字段: %+v", c2)
	}
	// 助手可聊天角色出现在聊天列表
	items := s.ListChatEnabled()
	if len(items) != 2 {
		t.Fatalf("聊天列表应为 2（gaea + lib_01）: %d", len(items))
	}
}

func TestToPreset_CarriesChatFields(t *testing.T) {
	c := &Character{
		ID: "lib_01", Name: "林晚", Gender: "female",
		Dims:          whisper.PersonalityDims{T: 40, I: 60, S: 70, O: 50, R: 30},
		VoiceGuide:    "清冷剑修",
		BehaviorRules: "说话简短",
		EmotionLogic:  "外冷内热",
		Tags:          []string{"女主"},
	}
	p := c.ToPreset()
	if p == nil || p.ID != "lib_01" || p.Dims.T != 40 {
		t.Fatalf("ToPreset 转换错误: %+v", p)
	}
	if !strings.Contains(p.VoiceGuide, "行为规则：说话简短") || !strings.Contains(p.VoiceGuide, "情感逻辑：外冷内热") {
		t.Fatalf("行为规则/情感逻辑未注入 voiceGuide: %s", p.VoiceGuide)
	}
}
