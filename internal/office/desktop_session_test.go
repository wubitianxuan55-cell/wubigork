package office

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveModesRoundTrip 落盘 → 读回一致性（原子写路径与 LoadModes 闭环）。
func TestSaveModesRoundTrip(t *testing.T) {
	root := t.TempDir()
	modes := map[string]bool{"s1": true, "s2": false}
	if err := SaveModes(root, modes); err != nil {
		t.Fatalf("SaveModes: %v", err)
	}
	got := LoadModes(root)
	if !got["s1"] || got["s2"] {
		t.Errorf("LoadModes = %v, want s1=true s2=false", got)
	}
}

// TestSaveModesNoTempLeftover 原子写收尾干净：写成功后目录里不应残留
// 临时文件（临时文件 + rename 失败会留 .tmp，正常路径必须清干净）。
func TestSaveModesNoTempLeftover(t *testing.T) {
	root := t.TempDir()
	if err := SaveModes(root, map[string]bool{"a": true}); err != nil {
		t.Fatalf("SaveModes: %v", err)
	}
	if err := SaveModes(root, map[string]bool{"a": true, "b": true}); err != nil {
		t.Fatalf("SaveModes 二次写: %v", err)
	}
	dir := filepath.Join(root, "office")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("残留临时文件: %s", e.Name())
		}
	}
}

// TestSetPersistedMode 设/取/清闭环。
func TestSetPersistedMode(t *testing.T) {
	root := t.TempDir()
	if err := SetPersistedMode(root, "sx", true); err != nil {
		t.Fatalf("SetPersistedMode(true): %v", err)
	}
	if !GetPersistedMode(root, "sx") {
		t.Error("GetPersistedMode(sx) = false, want true")
	}
	if err := SetPersistedMode(root, "sx", false); err != nil {
		t.Fatalf("SetPersistedMode(false): %v", err)
	}
	if GetPersistedMode(root, "sx") {
		t.Error("GetPersistedMode(sx) = true after disable, want false")
	}
	if err := ClearAllPersistedModes(root); err != nil {
		t.Fatalf("ClearAllPersistedModes: %v", err)
	}
	if len(LoadModes(root)) != 0 {
		t.Errorf("LoadModes 清空后 = %v, want empty", LoadModes(root))
	}
}

// TestLoadModesMissingFile 文件不存在（首次运行）→ 空 map 不崩。
func TestLoadModesMissingFile(t *testing.T) {
	modes := LoadModes(t.TempDir())
	if len(modes) != 0 {
		t.Errorf("LoadModes 缺文件 = %v, want empty", modes)
	}
}
