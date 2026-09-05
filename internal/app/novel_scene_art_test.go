package app

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/project"
)

// TestSaveSceneIllustrationToPlayExportsB64 T1 章节配图落盘：b64 结果 → play exports 文件。
func TestSaveSceneIllustrationToPlayExportsB64(t *testing.T) {
	dir := t.TempDir()
	pm := &project.Manager{Dir: dir}
	resp := &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{
			B64JSON: base64.StdEncoding.EncodeToString([]byte("fake-png")),
		}},
	}
	out, err := saveSceneIllustrationToPlayExports(pm, 3, resp)
	if err != nil {
		t.Fatalf("save scene illustration: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(out), ".gaea/play/exports/scene-3-") {
		t.Fatalf("落点错误: %s", out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("配图文件未落盘: %v", err)
	}
}

// TestSaveSceneIllustrationToPlayExportsNoData 无图结果 → 诚实报错。
func TestSaveSceneIllustrationToPlayExportsNoData(t *testing.T) {
	pm := &project.Manager{Dir: t.TempDir()}
	if _, err := saveSceneIllustrationToPlayExports(pm, 1, &ai.ImageGenerationResponse{}); err == nil {
		t.Fatalf("空结果应报错")
	}
}
