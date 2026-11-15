package app

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/assistant"
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
	a.mediaState = &mediaState{core: c, app: a}
	a.projectState = &projectState{core: c, app: a, mu: sync.RWMutex{}}
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
	// 聊天桥接：getOrCreateOrch 用库内角色生成人格（进程级会话缓存用完即删，
	// 固定 ID 在 -count 多次运行下不串扰）
	orch := a.getOrCreateOrch("lib_01")
	cleanupWhisperSession(t, "lib_01")
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

	cf := &types.CharacterFile{
		Characters: []types.Character{
			{ID: "ch_1", Name: "林晚", RoleType: "protagonist", Arc: "崛起", Status: "Alive", Background: "孤儿"},
		},
	}
	if err := pm.WriteCharacters(cf); err != nil {
		t.Fatalf("写角色文件: %v", err)
	}

	report, err := a.CharacterImportProject("")
	if err != nil || report["imported"] != 1 {
		t.Fatalf("导入项目 = %v, %v", report, err)
	}
	// 同一角色可被第二个项目引用
	dir2 := filepath.Join(t.TempDir(), "novelB")
	pm2, _ := project.Create(dir2, "第二本", "都市", "", "")
	a.setPM(pm2)
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

func TestCharacterSyncProject_RefusesLegacyChars(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novel")
	pm, _ := project.Create(dir, "测试", "玄幻", "", "")
	a.setPM(pm)
	// 项目里有未入库角色（旧数据）
	cf := &types.CharacterFile{Characters: []types.Character{{ID: "ch_9", Name: "未入库角色"}}}
	_ = pm.WriteCharacters(cf)
	if err := a.CharacterSyncProject(); err == nil {
		t.Fatalf("存在未入库角色时同步应被拒绝（防止覆盖丢数据）")
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

// TestCharacterSetProgressions 阶段八 8.2：状态演进整表替换/校验拒收/清空；
// 并钉 Sync 投影对 Progressions 的 v4.454 同款回带保护。
func TestCharacterSetProgressions(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novelP")
	pm, err := project.Create(dir, "演进测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{ID: "ch_1", Name: "沈青", RoleType: "supporting", Status: "Alive"}},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}

	// 校验拒收：缺章号 / 缺文本 / 坏 JSON
	if err := a.CharacterSetProgressions("ch_1", `[{"chapter":0,"text":"x"}]`); err == nil {
		t.Fatalf("缺章号应拒收")
	}
	if err := a.CharacterSetProgressions("ch_1", `[{"chapter":5,"text":"  "}]`); err == nil {
		t.Fatalf("缺文本应拒收")
	}
	if err := a.CharacterSetProgressions("ch_1", "not-json"); err == nil {
		t.Fatalf("坏 JSON 应拒收")
	}
	if err := a.CharacterSetProgressions("ghost", `[{"chapter":5,"text":"x"}]`); err == nil || !strings.Contains(err.Error(), "未找到角色") {
		t.Fatalf("未知角色应报未找到: %v", err)
	}

	// 整表替换落库
	if err := a.CharacterSetProgressions("ch_1", `[{"chapter":5,"field":"personality","text":"黑化阴沉","note":"卷一结尾"},{"chapter":8,"text":"左臂被斩断"}]`); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	cf, err := pm.ReadCharacters()
	if err != nil || len(cf.Characters) != 1 || len(cf.Characters[0].Progressions) != 2 {
		t.Fatalf("演进未落库: %v %+v", err, cf)
	}
	if cf.Characters[0].Progressions[0].Note != "卷一结尾" {
		t.Fatalf("备注未落库: %+v", cf.Characters[0].Progressions[0])
	}
	// 空数组=清空
	if err := a.CharacterSetProgressions("ch_1", `[]`); err != nil {
		t.Fatalf("清空失败: %v", err)
	}
	cf, _ = pm.ReadCharacters()
	if len(cf.Characters[0].Progressions) != 0 {
		t.Fatalf("空数组应清空: %+v", cf.Characters[0].Progressions)
	}
}

// TestSyncProject_CarriesInjectFields 阶段八 8.1/8.2：库投影同步不丢项目级
// 注入字段（模式/别名/排除词/状态演进，v4.454 主角关系同款回带）。
func TestSyncProject_CarriesInjectFields(t *testing.T) {
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "novelS")
	pm, err := project.Create(dir, "同步保护", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{
			ID: "ch_1", Name: "沈青", RoleType: "supporting", Status: "Alive",
			InjectMode:   "always",
			Aliases:      []string{"青儿"},
			ExcludeHits:  []string{"王爷"},
			Progressions: []types.CharacterProgression{{Chapter: 5, Field: "status", Text: "Missing"}},
		}},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	// 无库引用角色 → Sync 会拒（防误清）；改为直接造已关联场景：
	// 先导入入库建立引用，再改项目副本注入配置，最后同步验证回带。
	if _, err := a.CharacterImportProject(""); err != nil {
		t.Fatalf("导入: %v", err)
	}
	// 导入后项目副本被库投影覆盖（此时库内无注入配置）——重写项目副本带上注入配置
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{
			ID: "ch_1", Name: "沈青", RoleType: "supporting", Status: "Alive",
			InjectMode:   "always",
			Aliases:      []string{"青儿"},
			ExcludeHits:  []string{"王爷"},
			Progressions: []types.CharacterProgression{{Chapter: 5, Field: "status", Text: "Missing"}},
		}},
	}); err != nil {
		t.Fatalf("重写角色: %v", err)
	}
	if err := a.CharacterSyncProject(); err != nil {
		t.Fatalf("同步: %v", err)
	}
	cf, err := pm.ReadCharacters()
	if err != nil || len(cf.Characters) != 1 {
		t.Fatalf("读回失败: %v", err)
	}
	ch := cf.Characters[0]
	if ch.InjectMode != "always" || len(ch.Aliases) != 1 || len(ch.ExcludeHits) != 1 || len(ch.Progressions) != 1 {
		t.Fatalf("同步后注入字段应原样回带: %+v", ch)
	}
}
