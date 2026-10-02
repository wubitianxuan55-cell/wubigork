package style

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gaea/gaea/internal/project"
)

// 本文件锁定 StyleProfile 读写收口（审计 IN1-05）后的两条行为红线：
// 落点必须是 <dir>/.gaea/style-profile.json（逐字节同 JSON）；
// 旧品牌目录 <dir>/.wubigork/style-profile.json 必须仍然可读（回退
// 分支收口在 project.Manager.ReadStyleProfileFile 一处，删它必红）。

func roundTripProfile() *Profile {
	return &Profile{
		Name:        "冷峻克制",
		Description: "短句为主，情绪内敛",
		Traits: map[string]string{
			"narrative_voice": "冷峻克制",
			"pacing":          "张弛有度",
			"chinese_style":   "古风韵味",
		},
		RawMarkdown: "## 写作风格: 冷峻克制\n",
	}
}

// TestSaveLoadProfileRoundTripAtGaeaDir Save/Load 往返：落点必须是
// .gaea/style-profile.json，磁盘字节与 MarshalIndent（两空格）逐字节一致，
// 读回字段全等。若落点被改错一级（如落到项目根），本测试必红。
func TestSaveLoadProfileRoundTripAtGaeaDir(t *testing.T) {
	dir := t.TempDir()
	pm := &project.Manager{Dir: dir}
	profile := roundTripProfile()

	if err := SaveProfile(pm, profile); err != nil {
		t.Fatalf("SaveProfile 失败: %v", err)
	}

	// 落点：必须是 .gaea/style-profile.json（不是项目根、不是别的一级）。
	wantPath := filepath.Join(dir, ".gaea", "style-profile.json")
	raw, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("落点 %s 应存在（落点被改错一级会在此红）: %v", wantPath, err)
	}
	wantBytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent 失败: %v", err)
	}
	if string(raw) != string(wantBytes) {
		t.Fatalf("磁盘字节应与 MarshalIndent 输出逐字节一致:\n got: %s\nwant: %s", raw, wantBytes)
	}

	got, err := LoadProfile(pm)
	if err != nil {
		t.Fatalf("LoadProfile 失败: %v", err)
	}
	if !reflect.DeepEqual(got, profile) {
		t.Fatalf("读回应与写入档案全等:\n got: %+v\nwant: %+v", got, profile)
	}
}

// TestLoadProfileFallsBackToLegacyBrandDir 旧品牌目录可读：仅存在
// .wubigork/style-profile.json（无 .gaea）时 LoadProfile 必须经回退读到；
// 且回退不影响写侧——SaveProfile 仍落 .gaea/，旧文件原样不动。
// project 侧回退分支被删则本测试必红。
func TestLoadProfileFallsBackToLegacyBrandDir(t *testing.T) {
	dir := t.TempDir()
	pm := &project.Manager{Dir: dir}
	legacy := roundTripProfile()

	legacyPath := filepath.Join(dir, ".wubigork", "style-profile.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		t.Fatalf("建旧品牌目录失败: %v", err)
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent 失败: %v", err)
	}
	if err := os.WriteFile(legacyPath, data, 0644); err != nil {
		t.Fatalf("写旧品牌档案失败: %v", err)
	}

	got, err := LoadProfile(pm)
	if err != nil {
		t.Fatalf("旧品牌目录（.wubigork）应可读，回退分支被删会在此红: %v", err)
	}
	if !reflect.DeepEqual(got, legacy) {
		t.Fatalf("旧品牌档案读回应全等:\n got: %+v\nwant: %+v", got, legacy)
	}

	// 写侧不回退：SaveProfile 固定落 .gaea/，.wubigork 旧文件不动。
	fresh := *legacy
	fresh.Name = "新写档案"
	if err := SaveProfile(pm, &fresh); err != nil {
		t.Fatalf("SaveProfile 失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gaea", "style-profile.json")); err != nil {
		t.Fatalf("SaveProfile 应落 .gaea/style-profile.json: %v", err)
	}
	afterLegacy, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatalf("旧品牌文件不应被写侧触碰: %v", err)
	}
	if string(afterLegacy) != string(data) {
		t.Fatalf("旧品牌文件应原样不动")
	}
	if got, err := LoadProfile(pm); err != nil || got.Name != "新写档案" {
		t.Fatalf("新写后应优先读 .gaea 新档: got=%+v err=%v", got, err)
	}
}

// TestLoadProfileMissingErrors 无任何档案时必须如实报错（消费方
// buildStyle 靠 err 返回空串），且回退分支只对 os.IsNotExist 放行。
func TestLoadProfileMissingErrors(t *testing.T) {
	pm := &project.Manager{Dir: t.TempDir()}
	if _, err := LoadProfile(pm); err == nil {
		t.Fatalf("无档案时 LoadProfile 应报错")
	}
}
