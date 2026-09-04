package app

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/contextview"
)

// resolveNodeImages 单测（2.5b 后半）：绝对引用直接解码；相对引用按 cwd
// 解析；缺失文件诚实 Exists=false；不可解码文件 Exists=true 但尺寸留零。
func TestResolveNodeImages(t *testing.T) {
	dir := t.TempDir()
	// 100×60 PNG：⌈100/28⌉=4 × ⌈60/28⌉=3 = 12 tokens（标准档不缩放）。
	img := image.NewRGBA(image.Rect(0, 0, 100, 60))
	img.Set(0, 0, color.White)
	abs := filepath.Join(dir, "real.png")
	f, err := os.Create(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	// 不可解码的「图片」：svg 是文本，DecodeConfig 失败。
	svg := filepath.Join(dir, "vec.svg")
	if err := os.WriteFile(svg, []byte("<svg xmlns='x'/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := contextview.NodeDetail{
		Kind:      "user_message",
		ImageRefs: []string{abs, "missing.png", "rel.png", svg},
	}
	resolveNodeImages(&d, dir)

	if len(d.Images) != 4 {
		t.Fatalf("images = %d, want 4", len(d.Images))
	}
	real := d.Images[0]
	if !real.Exists || real.Width != 100 || real.Height != 60 {
		t.Fatalf("real.png 解析错误: %+v", real)
	}
	if real.StdTokens != 12 || real.HighTokens != 12 {
		t.Fatalf("real.png tokens = %d/%d, want 12/12", real.StdTokens, real.HighTokens)
	}
	if real.ScaledW != 100 || real.ScaledH != 60 {
		t.Fatalf("real.png 缩放尺寸 = %dx%d, want 100x60", real.ScaledW, real.ScaledH)
	}
	if d.Images[1].Exists {
		t.Fatalf("missing.png 应诚实 Exists=false: %+v", d.Images[1])
	}
	// 相对引用：Path 按 cwd 解析为绝对路径，RefCwd 记录依据。
	rel := d.Images[2]
	if rel.Exists {
		t.Fatalf("rel.png 文件不存在应 Exists=false: %+v", rel)
	}
	if rel.Path != filepath.Join(dir, "rel.png") || rel.RefCwd != dir {
		t.Fatalf("rel.png 路径解析错误: %+v", rel)
	}
	// svg：文件存在但尺寸未知（诚实留零）。
	if !d.Images[3].Exists || d.Images[3].Width != 0 || d.Images[3].StdTokens != 0 {
		t.Fatalf("svg 应存在但尺寸/token 留零: %+v", d.Images[3])
	}
}
