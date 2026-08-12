package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

// TestMigrateLegacyDataRoot 旧 ResourceDir/whisper_data → 新 DataRoot/whisper_data：
// 幂等迁移，目标已有数据时不动。
func TestMigrateLegacyDataRoot(t *testing.T) {
	legacyRoot := t.TempDir()
	legacyData := filepath.Join(legacyRoot, "whisper_data")
	if err := os.MkdirAll(filepath.Join(legacyData, "chat"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(legacyData, "engines.json"), []byte(`{"x":1}`), 0o644)
	_ = os.WriteFile(filepath.Join(legacyData, "chat", "a.jsonl"), []byte("line1"), 0o644)
	_ = os.WriteFile(filepath.Join(legacyData, "model_stats.json"), []byte(`{"total":5}`), 0o644)

	newRoot := filepath.Join(t.TempDir(), "data", "whisper_data")
	a := &App{core: &core{cfg: &config.Config{ResourceDir: legacyRoot}}}
	a.whisperState = &whisperState{whisperDataRoot: newRoot}

	a.migrateLegacyDataRoot()

	for _, want := range []string{"engines.json", "model_stats.json", filepath.Join("chat", "a.jsonl")} {
		if _, err := os.Stat(filepath.Join(newRoot, want)); err != nil {
			t.Errorf("迁移后缺少 %s: %v", want, err)
		}
	}

	// 幂等：二次迁移不覆盖已有目标内容
	_ = os.WriteFile(filepath.Join(newRoot, "new.json"), []byte("new"), 0o644)
	a.migrateLegacyDataRoot()
	if data, err := os.ReadFile(filepath.Join(newRoot, "new.json")); err != nil || string(data) != "new" {
		t.Errorf("二次迁移破坏了已有文件: %v %q", err, string(data))
	}
}
