package characterlib

import (
	"net/http"
	"net/http/httptest"
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

func TestImportProjectCharacters_IdempotentAndOneWay(t *testing.T) {
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

	// 单向约束：已存在的库内角色绝不被项目数据覆盖
	if err := s.Upsert(&Character{ID: "ch_1", Name: "林晚", Kind: KindCustom, Background: "库内丰富背景", Arc: "崛起"}); err != nil {
		t.Fatalf("预置库内角色失败: %v", err)
	}
	reimport := []types.Character{{ID: "ch_1", Name: "林晚", RoleType: "supporting", Arc: "黑化", Background: "项目薄记录"}}
	n, err = s.ImportProjectCharacters("projA", reimport)
	if err != nil || n != 1 {
		t.Fatalf("再次导入 = %d, %v", n, err)
	}
	c, _ := s.Get("ch_1")
	if c.Background != "库内丰富背景" || c.Arc != "崛起" {
		t.Fatalf("项目导入污染了库内角色: %+v", c)
	}
	// 关联仍建立（项目可引用已存在的库内角色）
	refs, _ := s.ProjectIDsForCharacter("ch_1")
	if len(refs) != 1 || refs[0] != "projA" {
		t.Fatalf("关联异常: %v", refs)
	}
	// 另一项目同名角色：不合并（避免 ID 重映射破坏项目内关系引用）
	other := []types.Character{{ID: "ch_99", Name: "林晚", RoleType: "supporting"}}
	n, err = s.ImportProjectCharacters("projB", other)
	if err != nil || n != 1 {
		t.Fatalf("跨项目导入 = %d, %v", n, err)
	}
	c99, _ := s.Get("ch_99")
	if c99 == nil || c99.Name != "林晚" {
		t.Fatalf("应以项目 ID 新建独立角色: %+v", c99)
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

// TestEnsureAssistants_GuardSkipsCustomKeepsMirror 镜像守卫：custom 角色
// 被 personalityId 指向时字段原样保留；builtin 命中覆写与未命中新建影子行
// 的既有镜像行为不回归。
func TestEnsureAssistants_GuardSkipsCustomKeepsMirror(t *testing.T) {
	s := newTestStore(t)
	presets := []whisper.PersonalityPreset{
		{ID: "gaea", Label: "gaea", Gender: "female", VoiceGuide: "大地"},
	}
	if err := s.EnsureBuiltins(presets); err != nil {
		t.Fatalf("种子化失败: %v", err)
	}
	// 用户自建角色（v4.48 可被选为助手人格，personalityId 即角色 id）
	if err := s.Upsert(&Character{ID: "lib_01", Name: "林晚", Kind: KindCustom, Gender: "female", Tags: []string{"女主"}, VoiceGuide: "清冷", ChatEnabled: true}); err != nil {
		t.Fatalf("预置 custom 角色失败: %v", err)
	}
	asts := []assistant.Assistant{
		{ID: "ast_custom", Name: "峨嵋", PersonalityID: "lib_01", VoiceGuide: "仙气", Enabled: true},
		{ID: "ast_builtin", Name: "小鸢", PersonalityID: "gaea", VoiceGuide: "太空", Enabled: true},
		{ID: "ast_new", Name: "阿幻", PersonalityID: "lib_new", Enabled: true},
	}
	if err := s.EnsureAssistants(asts, presets); err != nil {
		t.Fatalf("助手同步失败: %v", err)
	}
	// ① custom 角色绝不被镜像覆写：Kind/名字/人格字段原样，无 AssistantID
	c, err := s.Get("lib_01")
	if err != nil || c == nil {
		t.Fatalf("Get(lib_01) = %v, %v", c, err)
	}
	if c.Kind != KindCustom || c.Name != "林晚" || c.AssistantID != "" || c.VoiceGuide != "清冷" || !c.ChatEnabled {
		t.Fatalf("custom 角色被助手镜像覆写: %+v", c)
	}
	// ② builtin 命中：镜像覆写行为保持
	b, _ := s.Get("gaea")
	if b == nil || b.Kind != KindAssistant || b.Name != "小鸢" || b.AssistantID != "ast_builtin" {
		t.Fatalf("builtin 命中镜像行为回归: %+v", b)
	}
	// ② 未命中：影子行照建
	n, _ := s.Get("lib_new")
	if n == nil || n.Kind != KindAssistant || n.Name != "阿幻" || n.AssistantID != "ast_new" {
		t.Fatalf("未命中应新建影子行: %+v", n)
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

func TestDrawRandom_FiltersAndLimit(t *testing.T) {
	s := newTestStore(t)
	base := []Character{
		{ID: "a", Name: "A", Kind: KindCustom, Gender: "female", ChatEnabled: true},
		{ID: "b", Name: "B", Kind: KindCustom, Gender: "male", ChatEnabled: true},
		{ID: "c", Name: "C", Kind: KindCustom, Gender: "female", ChatEnabled: false},
		{ID: "d", Name: "D", Kind: KindCustom, Gender: "male", ChatEnabled: true},
	}
	for _, c := range base {
		_ = s.Upsert(&c)
	}
	// 仅女性
	items, err := s.DrawRandom(10, "female", "", false)
	if err != nil {
		t.Fatalf("DrawRandom: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("女性角色应抽到 2 个: %d", len(items))
	}
	for _, c := range items {
		if c.Gender != "female" {
			t.Fatalf("抽到性别不符角色: %+v", c)
		}
	}
	// 仅可聊天
	items, _ = s.DrawRandom(10, "", "", true)
	if len(items) != 3 {
		t.Fatalf("可聊天角色应抽到 3 个: %d", len(items))
	}
	// 数量上限
	items, _ = s.DrawRandom(99, "", "", false)
	if len(items) != 4 {
		t.Fatalf("抽卡数量不应超过库内总数: %d", len(items))
	}
}

// TestCapPortraitURL 超大内联头像不随响应返回（防止撑爆 Wails IPC 卡死）。
func TestCapPortraitURL(t *testing.T) {
	if got := capPortraitURL("https://imgen.x.ai/xai-image/abc"); got == "" {
		t.Fatal("远程 URL 不应被截断")
	}
	small := "data:image/png;base64," + strings.Repeat("a", 100)
	if got := capPortraitURL(small); got != small {
		t.Fatal("小内联头像不应被截断")
	}
	huge := "data:image/png;base64," + strings.Repeat("b", maxPortraitDataURL)
	if got := capPortraitURL(huge); got != "" {
		t.Fatalf("超大内联头像应置空, got len=%d", len(got))
	}
}

// TestUpsertSavesPortraitToFile data URL 剧照保存时单独落盘为文件，
// 库里只存文件路径（防巨型 base64 撑爆 IPC）。
func TestUpsertSavesPortraitToFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("store 不可用")
	}
	defer s.Close()

	// 1x1 红色 PNG
	portrait := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	c := &Character{ID: "c1", Name: "测试", Kind: KindCustom, PortraitURL: portrait}
	if err := s.Upsert(c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("c1")
	if err != nil || got == nil {
		t.Fatalf("Get: %v %v", err, got)
	}
	if got.PortraitURL == portrait {
		t.Fatal("data URL 应被落盘为文件路径")
	}
	if !strings.HasPrefix(got.PortraitURL, dir) {
		t.Fatalf("应返回文件路径, got %q", got.PortraitURL)
	}
	if _, err := os.Stat(got.PortraitURL); err != nil {
		t.Fatalf("剧照文件不存在: %v", err)
	}
}

// TestMigratePortraitsToFiles 既有巨型 base64 剧照启动迁移为文件。
func TestMigratePortraitsToFiles(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("store 不可用")
	}
	defer s.Close()

	huge := "data:image/png;base64," + strings.Repeat("A", 400*1024)
	if err := s.Upsert(&Character{ID: "h1", Name: "巨图", Kind: KindCustom, PortraitURL: huge}); err != nil {
		t.Fatal(err)
	}
	// Upsert 已落盘 → 直接回写一个 data URL 模拟历史遗留行
	if _, err := s.db.Exec(`UPDATE characters SET portrait_url=? WHERE id='h1'`, huge); err != nil {
		t.Fatal(err)
	}
	n := s.MigratePortraitsToFiles()
	if n != 1 {
		t.Fatalf("迁移数 = %d, want 1", n)
	}
	got, _ := s.Get("h1")
	if got == nil || got.PortraitURL == huge {
		t.Fatalf("迁移后应指向文件: %+v", got)
	}
	if _, err := os.Stat(got.PortraitURL); err != nil {
		t.Fatalf("迁移文件不存在: %v", err)
	}
}

// TestUpsertLocalizesRemotePortrait 远程剧照（xAI 临时图等）保存时下载到本地。
func TestUpsertLocalizesRemotePortrait(t *testing.T) {
	body := []byte("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("store 不可用")
	}
	defer s.Close()

	remote := srv.URL + "/portrait.png"
	c := &Character{ID: "r1", Name: "远程图", Kind: KindCustom, PortraitURL: remote}
	if err := s.Upsert(c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("r1")
	if err != nil || got == nil {
		t.Fatalf("Get: %v %v", err, got)
	}
	if got.PortraitURL == remote {
		t.Fatal("远程 URL 应被下载为本地文件路径")
	}
	if _, err := os.Stat(got.PortraitURL); err != nil {
		t.Fatalf("下载的剧照文件不存在: %v", err)
	}
}

// TestMigrateRemotePortraits 历史遗留的远程剧照 URL 启动时迁移为本地文件。
func TestMigrateRemotePortraits(t *testing.T) {
	body := []byte("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	s := NewStore(dir)
	if s == nil || s.db == nil {
		t.Fatal("store 不可用")
	}
	defer s.Close()

	// 直写 DB 模拟历史遗留行（绕过 Upsert 的下载逻辑）
	remote := srv.URL + "/legacy.png"
	if _, err := s.db.Exec(
		`INSERT INTO characters (id, name, kind, portrait_url, created_at, updated_at) VALUES ('l1', '遗留', 'custom', ?, 'x', 'x')`,
		remote); err != nil {
		t.Fatal(err)
	}
	n := s.MigrateRemotePortraits()
	if n != 1 {
		t.Fatalf("迁移数 = %d, want 1", n)
	}
	got, _ := s.Get("l1")
	if got == nil || got.PortraitURL == remote {
		t.Fatalf("迁移后应指向本地文件: %+v", got)
	}
	if _, err := os.Stat(got.PortraitURL); err != nil {
		t.Fatalf("迁移文件不存在: %v", err)
	}
}
