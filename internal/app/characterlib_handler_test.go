package app

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/character"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
	wdb "github.com/gaea/gaea/internal/whisper/db"
)

// newCharacterLibTestApp 构造角色库测试 App（不依赖 LLM）。
func newCharacterLibTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	root := filepath.Join(t.TempDir(), "data")

	c := &core{
		cfg:       &config.Config{},
		engineMgr: modelengine.NewManager("", ""),
		chatStore: chat.NewStore(filepath.Join(root, "chat")),
		charLib:   characterlib.NewStore(filepath.Join(root, "characterlib")),
	}
	t.Cleanup(func() { _ = c.chatStore.Close() })
	t.Cleanup(func() { _ = c.charLib.Close() })
	t.Cleanup(func() { _ = wdb.CloseDatabase(root) })

	a := &App{core: c}
	a.writingState = &writingState{core: c, app: a, mu: sync.RWMutex{}}
	a.whisperState = &whisperState{core: c, app: a, whisperDataRoot: root}
	a.assistantMgr = assistant.NewEmpty(root)
	return a
}

func TestCharacterSave_CreatesChatCharacterAndAssistant(t *testing.T) {
	a := newCharacterLibTestApp(t)
	payload := `{
		"id": "lib_01", "name": "林晚", "kind": "custom", "gender": "female",
		"roleType": "protagonist", "background": "剑修", "arc": "崛起",
		"chatEnabled": true, "voiceGuide": "清冷剑修",
		"dims": {"T":40,"I":60,"S":70,"O":50,"R":30},
		"behaviorRules": "说话简短", "emotionLogic": "外冷内热"
	}`
	c, err := a.CharacterSave(payload)
	if err != nil {
		t.Fatalf("CharacterSave: %v", err)
	}
	if c.ID != "lib_01" || c.AssistantID == "" {
		t.Fatalf("保存结果异常: %+v", c)
	}
	// assistant 通道已创建
	ast := a.assistantMgr.Get(c.AssistantID)
	if ast == nil || !ast.Enabled || ast.VoiceGuide != "清冷剑修" {
		t.Fatalf("assistant 通道未同步: %+v", ast)
	}
	// 聊天桥接：getOrCreateOrch 用库内角色生成人格
	orch := a.getOrCreateOrch("lib_01")
	if orch == nil || orch.Preset.ID != "lib_01" || orch.Preset.Dims.T != 40 {
		t.Fatalf("聊天桥接失败: %+v", orch)
	}
	if orch.Preset.Label != "林晚" || orch.Preset.VoiceGuide == "" {
		t.Fatalf("人格字段未来自角色库: %+v", orch.Preset)
	}
	// 统一人格列表包含库内可聊天角色
	presets := a.WhisperGetPersonalities()
	found := false
	for _, p := range presets {
		if p.ID == "lib_01" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("聊天人格列表缺少库内角色")
	}
}

func TestCharacterImportAssociateSyncProject(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novelA")
	pm, err := project.Create(dir, "测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	a.characterAgent = character.New(nil, pm, a.cfg, nil)

	cf := &types.CharacterFile{
		Characters: []types.Character{
			{ID: "ch_1", Name: "林晚", RoleType: "protagonist", Arc: "崛起", Status: "Alive", Background: "孤儿"},
		},
	}
	if err := pm.WriteCharacters(cf); err != nil {
		t.Fatalf("写角色文件: %v", err)
	}

	n, err := a.CharacterImportProject()
	if err != nil || n != 1 {
		t.Fatalf("导入项目 = %d, %v", n, err)
	}
	// 同一角色可被第二个项目引用
	dir2 := filepath.Join(t.TempDir(), "novelB")
	pm2, _ := project.Create(dir2, "第二本", "都市", "", "")
	a.setPM(pm2)
	a.characterAgent = character.New(nil, pm2, a.cfg, nil)
	if err := a.CharacterAssociate("ch_1", "supporting"); err != nil {
		t.Fatalf("关联到第二项目: %v", err)
	}
	projects, err := a.charLib.ProjectIDsForCharacter("ch_1")
	if err != nil || len(projects) != 2 {
		t.Fatalf("角色应被两个项目引用: %v %v", projects, err)
	}

	// 物化回项目：项目 B 的 characters.json 只含引用角色
	if err := a.CharacterSyncProject(); err != nil {
		t.Fatalf("物化到项目: %v", err)
	}
	cf2, err := pm2.ReadCharacters()
	if err != nil || len(cf2.Characters) != 1 || cf2.Characters[0].Name != "林晚" {
		t.Fatalf("项目 B 物化结果异常: %v %v", cf2, err)
	}
	// 项目 B 内覆盖弧线状态后，物化取覆盖值
	_ = a.charLib.Associate(dir2, "ch_1", "supporting", "黑化", "Alive")
	_ = a.CharacterSyncProject()
	cf2, _ = pm2.ReadCharacters()
	if cf2.Characters[0].Arc != "黑化" {
		t.Fatalf("项目内弧线覆盖未生效: %+v", cf2.Characters[0])
	}
}

// TestCharacterAssociateTo_PicksSpecificNovel 回归：角色库「加入项目」时
// 应能选择任意小说，而不是只能加入当前打开的项目。
func TestCharacterAssociateTo_PicksSpecificNovel(t *testing.T) {
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()

	// 当前打开的项目 A
	dirA := filepath.Join(a.cfg.NovelsDir, "novelA")
	pmA, err := project.Create(dirA, "小说A", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目A: %v", err)
	}
	a.setPM(pmA)
	a.characterAgent = character.New(nil, pmA, a.cfg, nil)

	// 未打开的另一个项目 B
	dirB := filepath.Join(a.cfg.NovelsDir, "novelB")
	if _, err := project.Create(dirB, "小说B", "都市", "", ""); err != nil {
		t.Fatalf("创建项目B: %v", err)
	}

	c, err := a.CharacterSave(`{"id":"ch_1","name":"林晚","roleType":"protagonist","chatEnabled":false}`)
	if err != nil {
		t.Fatalf("保存角色: %v", err)
	}

	// 指定加入 B（不改变当前项目 A）
	if err := a.CharacterAssociateTo(dirB, c.ID, "supporting"); err != nil {
		t.Fatalf("加入指定小说: %v", err)
	}
	projects, err := a.charLib.ProjectIDsForCharacter(c.ID)
	if err != nil || len(projects) != 1 || projects[0] != dirB {
		t.Fatalf("应只加入小说B: %v %v", projects, err)
	}
	// 角色必须物化进小说 B 的 characters.json，小说角色面板才可见
	pmB, err := project.Open(dirB)
	if err != nil {
		t.Fatalf("打开小说B: %v", err)
	}
	cfB, err := pmB.ReadCharacters()
	if err != nil || len(cfB.Characters) != 1 || cfB.Characters[0].ID != c.ID {
		t.Fatalf("角色应物化进小说B characters.json: %+v %v", cfB, err)
	}
	if a.getPM().Dir != dirA {
		t.Fatalf("当前打开的项目不应被改变: %v", a.getPM().Dir)
	}

	// 非法目录 / 无效项目应被拒绝
	if err := a.CharacterAssociateTo(filepath.Join(t.TempDir(), "outside"), c.ID, ""); err == nil {
		t.Fatalf("书架外的目录应被拒绝")
	}
	if err := a.CharacterAssociateTo(filepath.Join(a.cfg.NovelsDir, "not-a-novel"), c.ID, ""); err == nil {
		t.Fatalf("无 project.json 的目录应被拒绝")
	}
}

// TestGetCharacters_HealsOrphanLibraryRefs 回归：旧版本从角色库「加入项目」
// 只写关联表、未物化 characters.json，打开小说角色面板（GetCharacters）时
// 应自动按 ID 合入，保证角色立即可见。
func TestGetCharacters_HealsOrphanLibraryRefs(t *testing.T) {
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()
	dir := filepath.Join(a.cfg.NovelsDir, "novelA")
	pm, err := project.Create(dir, "测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	a.characterAgent = character.New(nil, pm, a.cfg, nil)

	c, err := a.CharacterSave(`{"id":"ch_1","name":"林晚","roleType":"protagonist","chatEnabled":false}`)
	if err != nil {
		t.Fatalf("保存角色: %v", err)
	}
	// 模拟旧版「加入项目」：只写关联表，characters.json 保持为空
	if err := a.charLib.Associate(dir, c.ID, "protagonist", "", "Alive"); err != nil {
		t.Fatalf("关联角色: %v", err)
	}
	cf0, err := pm.ReadCharacters()
	if err != nil || len(cf0.Characters) != 0 {
		t.Fatalf("前置：characters.json 应为空: %+v %v", cf0, err)
	}

	data := a.GetCharacters()
	chars, ok := data["characters"].([]types.Character)
	if !ok || len(chars) != 1 || chars[0].ID != c.ID {
		t.Fatalf("GetCharacters 应包含角色库关联角色: %+v", data)
	}
	cf1, err := pm.ReadCharacters()
	if err != nil || len(cf1.Characters) != 1 || cf1.Characters[0].Name != "林晚" {
		t.Fatalf("characters.json 应已写入物化副本: %+v %v", cf1, err)
	}
}

func TestCharacterSave_JSONRoundTrip(t *testing.T) {
	a := newCharacterLibTestApp(t)
	c, err := a.CharacterSave(`{"name":"测试角色","chatEnabled":false,"roleType":"supporting"}`)
	if err != nil {
		t.Fatalf("CharacterSave: %v", err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out, err := a.CharacterGet(c.ID)
	if err != nil {
		t.Fatalf("CharacterGet: %v", err)
	}
	got, ok := out["character"].(*characterlib.Character)
	if !ok || got == nil || got.ID != c.ID || got.Name != "测试角色" {
		t.Fatalf("往返不一致: %s %+v", string(b), out["character"])
	}
}

func TestNovelSaveDoesNotPolluteLibrary(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novel")
	pm, _ := project.Create(dir, "测试", "玄幻", "", "")
	a.setPM(pm)
	a.characterAgent = character.New(nil, pm, a.cfg, nil)

	// 库内先有丰富设定
	if _, err := a.CharacterSave(`{"id":"ch_1","name":"林晚","background":"库内丰富背景","arc":"崛起"}`); err != nil {
		t.Fatalf("库内保存: %v", err)
	}
	// 小说页保存同名同 ID 的薄记录
	if err := a.SaveCharacter(`{"id":"ch_1","name":"林晚","background":"项目薄记录","arc":"黑化"}`); err != nil {
		t.Fatalf("小说页保存: %v", err)
	}
	c, err := a.charLib.Get("ch_1")
	if err != nil || c == nil {
		t.Fatalf("读取库内角色: %v %v", c, err)
	}
	if c.Background != "库内丰富背景" || c.Arc != "崛起" {
		t.Fatalf("小说页反向污染了全局角色库: %+v", c)
	}
}

func TestCharacterSyncProject_RefusesLegacyChars(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novel")
	pm, _ := project.Create(dir, "测试", "玄幻", "", "")
	a.setPM(pm)
	a.characterAgent = character.New(nil, pm, a.cfg, nil)
	// 项目里有未入库角色（旧数据）
	cf := &types.CharacterFile{Characters: []types.Character{{ID: "ch_9", Name: "未入库角色"}}}
	_ = pm.WriteCharacters(cf)
	if err := a.CharacterSyncProject(); err == nil {
		t.Fatalf("存在未入库角色时同步应被拒绝（防止覆盖丢数据）")
	}
}

// TestSaveCharactersBatchStaysInProject 回归：章节捕获的新角色批量保存后
// 只进项目 characters.json，不自动进全局角色库；入库必须由用户手动迁移。
func TestSaveCharactersBatchStaysInProject(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	a.characterAgent = character.New(nil, pm, a.cfg, nil)

	res, err := a.SaveCharactersBatch(`["阿九","老乞丐"]`)
	if err != nil {
		t.Fatalf("SaveCharactersBatch: %v", err)
	}
	created, ok := res["created"].([]types.Character)
	if !ok || len(created) != 2 {
		t.Fatalf("created = %#v, want 2 个角色", res["created"])
	}

	// 1. 项目 characters.json 已写入
	cf, err := pm.ReadCharacters()
	if err != nil || len(cf.Characters) != 2 {
		t.Fatalf("项目 characters.json = %d 个, want 2 (err=%v)", len(cf.Characters), err)
	}

	// 2. 未自动入库：全局库查不到，项目引用为空
	for _, ch := range created {
		lib, err := a.charLib.Get(ch.ID)
		if err != nil {
			t.Fatalf("查询全局库失败: %v", err)
		}
		if lib != nil {
			t.Fatalf("角色 %s(%s) 不应自动进入全局角色库（必须手动迁移）: %+v", ch.Name, ch.ID, lib)
		}
	}
	refs, err := a.charLib.ListByProject(dir)
	if err != nil || len(refs) != 0 {
		t.Fatalf("未手动迁移前项目引用应为空, got %d (err=%v)", len(refs), err)
	}

	// 3. 手动「一次性迁移」后：全局库可查 + 项目建立引用
	n, err := a.CharacterImportProject()
	if err != nil || n != 2 {
		t.Fatalf("手动迁移 = %d, want 2 (err=%v)", n, err)
	}
	for _, ch := range created {
		lib, err := a.charLib.Get(ch.ID)
		if err != nil || lib == nil {
			t.Fatalf("手动迁移后角色 %s(%s) 仍不在全局库: lib=%v err=%v", ch.Name, ch.ID, lib, err)
		}
		if lib.Name != ch.Name || lib.RoleType != "supporting" {
			t.Errorf("库内角色字段异常: %+v", lib)
		}
	}
	refs, err = a.charLib.ListByProject(dir)
	if err != nil || len(refs) != 2 {
		t.Fatalf("手动迁移后项目引用 = %d 个, want 2 (err=%v)", len(refs), err)
	}
}

// TestMergeCharacters 回归：合并同一人的两个角色卡——
// 保留角色、填充空缺字段、关系与组织引用重定向并去重、被合并角色移除。
func TestMergeCharacters(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novel")
	pm, _ := project.Create(dir, "测试", "玄幻", "", "")
	a.setPM(pm)
	a.characterAgent = character.New(nil, pm, a.cfg, nil)

	cf := &types.CharacterFile{
		Characters: []types.Character{
			{ID: "ch_keep", Name: "阿九", RoleType: "protagonist", Gender: "male", Personality: "冷静", Status: "Alive"},
			{ID: "ch_dup", Name: "九公子", RoleType: "supporting", Appearance: "白衣", Background: "江湖游侠", Status: "Alive"},
		},
		Organizations: []types.Organization{
			{ID: "org_1", Name: "丐帮", Members: []string{"ch_keep", "ch_dup", "ch_dup"}},
		},
		Relationships: []types.Relationship{
			{FromID: "ch_dup", ToID: "ch_keep", RelationType: "rival", Description: "自环", Intimacy: 0},
			{FromID: "ch_keep", ToID: "ch_dup", RelationType: "friend", Description: "重复", Intimacy: 10},
			{FromID: "ch_keep", ToID: "ch_dup", RelationType: "friend", Description: "重复", Intimacy: 10},
			{FromID: "ch_dup", ToID: "ch_other", RelationType: "mentor", Description: "师父", Intimacy: 20},
		},
	}
	if err := pm.WriteCharacters(cf); err != nil {
		t.Fatalf("写角色文件: %v", err)
	}

	res, err := a.MergeCharacters("ch_keep", "ch_dup")
	if err != nil {
		t.Fatalf("MergeCharacters: %v", err)
	}
	chars, _ := res["characters"].([]types.Character)
	if len(chars) != 1 {
		t.Fatalf("合并后角色数 = %d, want 1", len(chars))
	}
	kept := chars[0]
	if kept.Name != "阿九" {
		t.Errorf("应保留主角色名, got %q", kept.Name)
	}
	if kept.Appearance != "白衣" || kept.Background != "江湖游侠" {
		t.Errorf("空缺字段未从被合并角色补充: %+v", kept)
	}

	rels, _ := res["relationships"].([]types.Relationship)
	if len(rels) != 1 {
		t.Fatalf("合并后关系数 = %d, want 1（同名自环删除、重复去重、其余重定向）: %+v", len(rels), rels)
	}
	for _, r := range rels {
		if r.FromID == "ch_dup" || r.ToID == "ch_dup" {
			t.Errorf("关系仍指向被合并角色: %+v", r)
		}
	}

	orgs, _ := res["organizations"].([]types.Organization)
	if len(orgs[0].Members) != 1 || orgs[0].Members[0] != "ch_keep" {
		t.Errorf("组织成员未重定向/去重: %+v", orgs[0].Members)
	}

	// 文件已落盘
	cf2, err := pm.ReadCharacters()
	if err != nil || len(cf2.Characters) != 1 || cf2.Characters[0].ID != "ch_keep" {
		t.Fatalf("文件未同步: %+v err=%v", cf2, err)
	}
}

func TestCharacterDrawRandom_ReturnsLibraryCharacters(t *testing.T) {
	a := newCharacterLibTestApp(t)
	_, _ = a.CharacterSave(`{"id":"lib_a","name":"林晚","gender":"female","chatEnabled":true}`)
	_, _ = a.CharacterSave(`{"id":"lib_b","name":"顾长风","gender":"male","chatEnabled":true}`)
	_, _ = a.CharacterSave(`{"id":"lib_c","name":"苏小小","gender":"female","chatEnabled":false}`)
	items := a.CharacterDrawRandom(10, "", "", false)
	if len(items) != 3 {
		t.Fatalf("抽卡应返回全部 3 个: %d", len(items))
	}
	items = a.CharacterDrawRandom(10, "female", "", false)
	if len(items) != 2 {
		t.Fatalf("女性抽卡应返回 2 个: %d", len(items))
	}
	items = a.CharacterDrawRandom(10, "", "", true)
	if len(items) != 2 {
		t.Fatalf("可聊天抽卡应返回 2 个: %d", len(items))
	}
}
