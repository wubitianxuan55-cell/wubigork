package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestImageGenToolMeta 校验生图工具的元信息与 Schema。
func TestImageGenToolMeta(t *testing.T) {
	tool := imageGenTool{}
	if tool.Name() != "image_gen" || strings.TrimSpace(tool.Description()) == "" {
		t.Fatalf("工具元信息异常: %s", tool.Name())
	}
	if !json.Valid(tool.Schema()) || !json.Valid(tool.CompactSchema()) {
		t.Fatal("Schema 非法")
	}
	if tool.ReadOnly() {
		t.Error("image_gen 不应为只读工具")
	}
}

// TestSaveGenImage 校验 data URL 落盘（png/jpg/纯 base64）。
func TestSaveGenImage(t *testing.T) {
	t.Chdir(t.TempDir())
	cases := []struct {
		dataURL string
		ext     string
	}{
		{"data:image/png;base64,aGVsbG8=", ".png"},
		{"data:image/jpeg;base64,aGVsbG8=", ".jpg"},
		{"aGVsbG8=", ".png"}, // 无 data: 前缀的裸 base64
	}
	for _, c := range cases {
		rel, err := saveGenImage(".", c.dataURL)
		if err != nil {
			t.Fatalf("saveGenImage(%s): %v", c.dataURL[:20], err)
		}
		if !strings.HasSuffix(rel, c.ext) {
			t.Errorf("扩展名 = %s, want %s", rel, c.ext)
		}
		if _, err := os.Stat(filepath.FromSlash(rel)); err != nil {
			t.Errorf("文件不存在: %v", err)
		}
	}
}

// TestSaveGenImage_BadData 非法 base64 应报错。
func TestSaveGenImage_BadData(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := saveGenImage(".", "data:image/png;base64,!!!"); err == nil {
		t.Fatal("期望解码失败，实际成功")
	}
}
