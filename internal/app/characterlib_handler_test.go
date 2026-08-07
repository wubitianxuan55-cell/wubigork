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
