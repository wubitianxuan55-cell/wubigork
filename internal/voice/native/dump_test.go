//go:build windows && amd64 && cgo

// dump_test.go — 调试探针：GAEA_LOGMEL_DUMP=<path> 时把 Go 特征 dump 成 JSON 供 python 对拍。
package native

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDumpLogMelForDiff(t *testing.T) {
	out := os.Getenv("GAEA_LOGMEL_DUMP")
	if out == "" {
		t.Skip("GAEA_LOGMEL_DUMP 未设置")
	}
	name := os.Getenv("GAEA_LOGMEL_DUMP_NAME")
	if name == "" {
		name = "eot_complete"
	}
	pcm := loadGoldenWav(t, name)
	feats, err := WhisperLogMel(pcm)
	if err != nil {
		t.Fatalf("WhisperLogMel: %v", err)
	}
	grid := make([][]float32, whisperNumMels)
	for m := 0; m < whisperNumMels; m++ {
		grid[m] = feats[m*whisperNumFrames : (m+1)*whisperNumFrames]
	}
	raw, err := json.Marshal(grid)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(out, raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Logf("dumped to %s", out)
}
