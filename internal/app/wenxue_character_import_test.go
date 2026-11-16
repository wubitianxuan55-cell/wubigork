package app

// wenxue_character_import_test.go — 角色库导入刀单测（2026-11-16）。
// 覆盖：全字段映射六节落卡（与前端 cardFromLibraryChar 同源契约）、来源 id
// 记档、overwrite 显式闸、按名/按 id 精确导入、多同名/无命中错误、list 回执
// 库内同名命中。charLib 走真实 Store（temp 目录），GAEA_TUWEN_ROOT 锚临时
// 工作空间——hermetic，零引擎依赖。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// newWenxueImportTestApp 角色库导入测试台：charLib + 临时图文工作空间。
// charlib db 目录走 os.TempDir 自管清理（绕开 t.TempDir 对 db 文件的严格
// RemoveAll——Windows 下 db.Close 后句柄释放有延迟，框架清理会误报；同
// characterlib/store_test.go newTestStore 先例）。
func newWenxueImportTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	libDir := filepath.Join(os.TempDir(), "gaea-wenxue-charlib-import-"+t.Name())
	_ = os.RemoveAll(libDir)
	c := &core{
		cfg:       &config.Config{},
		engineMgr: modelengine.NewManager("", ""),
		charLib:   characterlib.NewStore(libDir),
	}
	if c.charLib == nil {
		t.Fatalf("characterlib Store 初始化失败")
	}
	t.Cleanup(func() {
		_ = c.charLib.Close()
		// 句柄释放有延迟：重试 RemoveAll 数次（cf. newTestStore 先例）
		for i := 0; i < 10; i++ {
			if os.RemoveAll(libDir) == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	})
	return &App{core: c}
}

func seedLibChar(t *testing.T, a *App, id, name string, mutate func(*characterlib.Character)) {
	t.Helper()
	c := &characterlib.Character{
		ID: id, Name: name, Kind: characterlib.KindCustom,
		Gender: "女", Age: "24", Tags: []string{"侦探", "冷静"},
		RoleType:    "protagonist",
		Personality: "外冷内热，观察力惊人",
		Background:  "前刑警，因冤案离职，如今开侦探社。",
		Appearance:  "黑色长发，左眉有疤",
		Figure:      "高挑清瘦",
		Motivation:  "查明当年冤案真相",
		Arc:         "从独行侠到学会信任搭档",
		Status:      "Alive",
		Notes:       "常年失眠，靠浓咖啡续命",
		DialogueSamples: []string{
			"证据不会说谎，人才会。",
			"案子结了，人还得往前走。",
		},
	}
	if mutate != nil {
		mutate(c)
	}
	if err := a.charLib.Upsert(c); err != nil {
		t.Fatalf("seed 角色库 %s: %v", id, err)
	}
}

func TestWenxueCharacterImport_MapsFullFieldsToSixSections(t *testing.T) {
	a := newWenxueImportTestApp(t)
	seedLibChar(t, a, "ch_linwan", "林晚", nil)

	out, err := wenxueCharacterImportTool(a).Execute(t.Context(), json.RawMessage(`{"name":"林晚"}`))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !strings.Contains(out, `"source_id":"ch_linwan"`) {
		t.Fatalf("回执缺 source_id: %s", out)
	}

	root := wenxueWorkspace()
	raw, err := os.ReadFile(filepath.Join(root, "角色", "林晚.md"))
	if err != nil {
		t.Fatalf("读卡: %v", err)
	}
	card := string(raw)
	for _, want := range []string{
		"# 林晚",
		"## 设定", "定位：主角 · 女 · 24", "标签：侦探、冷静",
		"前刑警，因冤案离职，如今开侦探社。", "动机：查明当年冤案真相",
		"来源：角色库（id ch_linwan",
		"## 外貌", "黑色长发，左眉有疤", "高挑清瘦",
		"## 性格", "外冷内热，观察力惊人", "弧线：从独行侠到学会信任搭档",
		"## 口头禅", "- 证据不会说谎，人才会。", "- 案子结了，人还得往前走。",
		"## 关系", "（留待书中发展）",
		"## 当前状态", "状态：存活", "常年失眠，靠浓咖啡续命",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("卡面缺 %q", want)
		}
	}
	// 节序与前端映射同源（设定→外貌→性格→口头禅→关系→当前状态）
	order := []string{"## 设定", "## 外貌", "## 性格", "## 口头禅", "## 关系", "## 当前状态"}
	last := -1
	for _, h := range order {
		idx := strings.Index(card, h)
		if idx < 0 || idx < last {
			t.Fatalf("节序不符（%q 位置异常）", h)
		}
		last = idx
	}
}

func TestWenxueCharacterImport_OverwriteGateAndByID(t *testing.T) {
	a := newWenxueImportTestApp(t)
	seedLibChar(t, a, "ch_linwan", "林晚", nil)

	tool := wenxueCharacterImportTool(a)
	// 首导 + 已存在拒
	if _, err := tool.Execute(t.Context(), json.RawMessage(`{"name":"林晚"}`)); err != nil {
		t.Fatalf("首导: %v", err)
	}
	if _, err := tool.Execute(t.Context(), json.RawMessage(`{"name":"林晚"}`)); err == nil {
		t.Fatal("overwrite=false 对已存在卡应拒")
	}
	// 库侧档案更新后按 id 重导入（整卡刷新语义）
	seedLibChar(t, a, "ch_linwan", "林晚", func(c *characterlib.Character) {
		c.Background = "冤案平反后重返警队。"
	})
	if _, err := tool.Execute(t.Context(), json.RawMessage(`{"id":"ch_linwan","overwrite":true}`)); err != nil {
		t.Fatalf("id 重导入: %v", err)
	}
	root := wenxueWorkspace()
	raw, err := os.ReadFile(filepath.Join(root, "角色", "林晚.md"))
	if err != nil {
		t.Fatalf("读卡: %v", err)
	}
	if !strings.Contains(string(raw), "冤案平反后重返警队。") {
		t.Fatal("重导入后库侧更新应落卡")
	}
}

func TestWenxueCharacterImport_LookupErrors(t *testing.T) {
	a := newWenxueImportTestApp(t)
	seedLibChar(t, a, "ch_linwan", "林晚", nil)
	seedLibChar(t, a, "ch_linwan2", "林晚", func(c *characterlib.Character) { c.Name = "林晚" })
	seedLibChar(t, a, "ch_mo", "莫雨", nil)
	tool := wenxueCharacterImportTool(a)

	t.Run("多同名要求 id", func(t *testing.T) {
		_, err := tool.Execute(t.Context(), json.RawMessage(`{"name":"林晚"}`))
		if err == nil || !strings.Contains(err.Error(), "请传 id") {
			t.Fatalf("err = %v, want 多同名提示", err)
		}
	})
	t.Run("无命中给近似提示", func(t *testing.T) {
		_, err := tool.Execute(t.Context(), json.RawMessage(`{"name":"林"}`))
		if err == nil || !strings.Contains(err.Error(), "近似命中") {
			t.Fatalf("err = %v, want 近似命中提示", err)
		}
	})
	t.Run("完全无命中", func(t *testing.T) {
		_, err := tool.Execute(t.Context(), json.RawMessage(`{"name":"不存在的人"}`))
		if err == nil || !strings.Contains(err.Error(), "角色库无") {
			t.Fatalf("err = %v, want 无命中提示", err)
		}
	})
	t.Run("name/id 双空拒", func(t *testing.T) {
		if _, err := tool.Execute(t.Context(), json.RawMessage(`{}`)); err == nil {
			t.Fatal("双空应拒")
		}
	})
	t.Run("无库 App 拒", func(t *testing.T) {
		bare := &App{core: a.core}
		bare.charLib = nil
		if _, err := wenxueCharacterImportTool(bare).Execute(t.Context(), json.RawMessage(`{"name":"林晚"}`)); err == nil || !strings.Contains(err.Error(), "角色库未初始化") {
			t.Fatalf("err = %v, want 角色库未初始化", err)
		}
	})
}

func TestWenxueCharacterImport_EmptyFieldsFallback(t *testing.T) {
	a := newWenxueImportTestApp(t)
	seedLibChar(t, a, "ch_min", "阿默", func(c *characterlib.Character) {
		c.Gender, c.Age, c.Tags = "", "", nil
		c.RoleType, c.Personality, c.Background, c.Appearance = "", "", "", ""
		c.Figure, c.Motivation, c.Arc, c.Status, c.Notes = "", "", "", "", ""
		c.DialogueSamples = nil
	})
	if _, err := wenxueCharacterImportTool(a).Execute(t.Context(), json.RawMessage(`{"name":"阿默"}`)); err != nil {
		t.Fatalf("import: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(wenxueWorkspace(), "角色", "阿默.md"))
	if err != nil {
		t.Fatalf("读卡: %v", err)
	}
	card := string(raw)
	// 空字段节落「（待补）」占位，与前端映射同源；来源行恒在
	for _, want := range []string{"## 设定", "（待补）", "（留待书中发展）", "来源：角色库（id ch_min"} {
		if !strings.Contains(card, want) {
			t.Errorf("卡面缺 %q", want)
		}
	}
}

func TestWenxueCharacterList_ReportsCharlibHits(t *testing.T) {
	a := newWenxueImportTestApp(t)
	seedLibChar(t, a, "ch_linwan", "林晚", nil)
	seedLibChar(t, a, "ch_mo", "莫雨", nil)

	// 工作空间只建了林晚一张卡
	if err := os.MkdirAll(filepath.Join(wenxueWorkspace(), "角色"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wenxueWorkspace(), "角色", "林晚.md"), []byte("# 林晚\n\n## 设定\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := wenxueCharacterListTool(a).Execute(t.Context(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var res struct {
		CharlibHits []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"charlib_hits"`
		CharlibHint string `json:"charlib_hint"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("回执非 JSON: %v (%s)", err, out)
	}
	if len(res.CharlibHits) != 1 || res.CharlibHits[0].ID != "ch_linwan" {
		t.Fatalf("hits = %+v, want 仅 ch_linwan（同名命中）", res.CharlibHits)
	}
	if res.CharlibHint == "" {
		t.Error("命中时应带导入提示")
	}
}

func TestWenxueCharacterList_NilCharlibStillWorks(t *testing.T) {
	_ = wenxueToolsTestRoot(t)
	out, err := wenxueCharacterListTool(nil).Execute(t.Context(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if strings.Contains(out, "charlib_hits") {
		t.Fatalf("无库不应带 hits 字段: %s", out)
	}
}
