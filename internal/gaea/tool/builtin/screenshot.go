package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/screen"
)

func init() { tool.RegisterBuiltin(screenCapture{}) }

// screenCapture 截图工具：让智能体像 Codex 一样自行捕获屏幕。
// 默认截全屏，可选 region 截取局部；保存 PNG 到工作区 .gaea/uploads/。
// workDir 非空时（桌面端）保存到工作区目录，空时（CLI）相对进程 cwd。
type screenCapture struct{ workDir string }

func (screenCapture) Name() string { return "screen_capture" }

func (screenCapture) Description() string {
	return "捕获屏幕（截图）。默认捕获整个屏幕；可用 region 参数指定像素区域（x/y/width/height，原点为屏幕左上角）只截取局部。结果保存为 PNG 并返回文件路径，之后可调用 vision 工具识别图片内容。"
}

func (screenCapture) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "region":{"type":"object","description":"可选：截取屏幕局部区域（像素坐标）","properties":{"x":{"type":"integer","description":"区域左上角 X（默认0）"},"y":{"type":"integer","description":"区域左上角 Y（默认0）"},"width":{"type":"integer","minimum":1,"description":"区域宽度"},"height":{"type":"integer","minimum":1,"description":"区域高度"}}}
},
"required":[]
}`)
}

func (screenCapture) ReadOnly() bool { return false }

func (screenCapture) CompactDescription() string     { return compactDesc["screen_capture"] }
func (screenCapture) CompactSchema() json.RawMessage { return compactSchema["screen_capture"] }

func (c screenCapture) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Region *struct {
			X int `json:"x"`
			Y int `json:"y"`
			W int `json:"width"`
			H int `json:"height"`
		} `json:"region"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &p); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
	}
	img, err := screen.Capture()
	if err != nil {
		return "", err
	}
	if p.Region != nil && p.Region.W > 0 && p.Region.H > 0 {
		img = cropImage(img, p.Region.X, p.Region.Y, p.Region.W, p.Region.H)
	}
	rel, err := saveScreenshot(c.workDir, img)
	if err != nil {
		return "", err
	}
	b := img.Bounds()
	return fmt.Sprintf("截图已保存：[%s](%s)（尺寸 %dx%d）", filepath.Base(rel), rel, b.Dx(), b.Dy()), nil
}

// cropImage 按区域裁剪图片（越界部分自动裁剪到屏幕范围内）。
func cropImage(src image.Image, x, y, w, h int) image.Image {
	b := src.Bounds()
	x0 := clamp(x, b.Min.X, b.Max.X)
	y0 := clamp(y, b.Min.Y, b.Max.Y)
	x1 := clamp(x+w, b.Min.X, b.Max.X)
	y1 := clamp(y+h, b.Min.Y, b.Max.Y)
	if x1 <= x0 || y1 <= y0 {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	for row := y0; row < y1; row++ {
		for col := x0; col < x1; col++ {
			dst.Set(col-x0, row-y0, src.At(col, row))
		}
	}
	return dst
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// saveScreenshot 把截图保存到工作区 .gaea/uploads/，
// 返回相对工作区（或进程 cwd）的路径。
func saveScreenshot(workDir string, img image.Image) (string, error) {
	dir := filepath.Join(workDir, ".gaea", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("screenshot-%d.png", time.Now().UnixNano())
	rel := filepath.ToSlash(filepath.Join(".gaea", "uploads", name))
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", err
	}
	return strings.TrimPrefix(rel, "./"), nil
}
