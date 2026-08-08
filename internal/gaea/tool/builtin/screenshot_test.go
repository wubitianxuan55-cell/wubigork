package builtin

import (
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScreenCaptureToolMeta(t *testing.T) {
	tool := screenCapture{}
	if tool.Name() != "screen_capture" || strings.TrimSpace(tool.Description()) == "" {
		t.Fatalf("工具元信息异常: %s", tool.Name())
	}
	if !json.Valid(tool.Schema()) {
		t.Fatalf("Schema 非法: %s", string(tool.Schema()))
	}
	if !json.Valid(tool.CompactSchema()) {
		t.Fatalf("CompactSchema 非法: %s", string(tool.CompactSchema()))
	}
}

func TestCropImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	src.Set(10, 10, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	got := cropImage(src, 0, 0, 20, 20)
	if b := got.Bounds(); b.Dx() != 20 || b.Dy() != 20 {
		t.Fatalf("裁剪尺寸 = %dx%d, want 20x20", b.Dx(), b.Dy())
	}
	if got.At(10, 10).(color.RGBA).R != 255 {
		t.Error("裁剪后像素内容不正确")
	}
	// 越界裁剪应钳制到源图范围
	clamped := cropImage(src, 90, 90, 100, 100)
	if b := clamped.Bounds(); b.Dx() != 10 || b.Dy() != 10 {
		t.Fatalf("越界裁剪尺寸 = %dx%d, want 10x10", b.Dx(), b.Dy())
	}
}

func TestSaveScreenshot(t *testing.T) {
	t.Chdir(t.TempDir())
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	rel, err := saveScreenshot("", img)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, ".gaea/uploads/screenshot-") || !strings.HasSuffix(rel, ".png") {
		t.Fatalf("路径 = %q", rel)
	}
	if _, err := os.Stat(filepath.FromSlash(rel)); err != nil {
		t.Fatalf("文件不存在: %v", err)
	}
}

// TestSaveScreenshot_Workspace 绑定工作区目录时写入该目录并返回相对路径。
func TestSaveScreenshot_Workspace(t *testing.T) {
	t.Chdir(t.TempDir())
	ws := filepath.Join(t.TempDir(), "workspace")
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	rel, err := saveScreenshot(ws, img)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(rel, ".gaea/") == false {
		t.Fatalf("路径应为工作区相对路径: %q", rel)
	}
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(rel))); err != nil {
		t.Fatalf("工作区文件不存在: %v", err)
	}
	// 进程 cwd 下不应生成（证明写入了工作区）
	if _, err := os.Stat(filepath.FromSlash(rel)); err == nil {
		t.Error("文件不应落在进程 cwd")
	}
}

func TestVisionToolMeta(t *testing.T) {
	tool := visionTool{}
	if tool.Name() != "vision" || strings.TrimSpace(tool.Description()) == "" {
		t.Fatalf("工具元信息异常: %s", tool.Name())
	}
	if !json.Valid(tool.Schema()) {
		t.Fatalf("Schema 非法: %s", string(tool.Schema()))
	}
	if tool.ReadOnly() != true {
		t.Error("vision 应为只读工具")
	}
}
