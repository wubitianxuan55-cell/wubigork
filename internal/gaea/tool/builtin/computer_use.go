// Package builtin provides Tianxuan's compile-time built-in tools.

//go:build windows

package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(computerUse{}) }

type computerUse struct{}

func (computerUse) Name() string { return "computer_use" }

func (computerUse) Description() string {
	return "计算机操作工具：捕获屏幕截图、模拟鼠标点击和键盘输入。推荐与具备视觉能力的模型配合使用。"
}

func (computerUse) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "action":{"type":"string","description":"操作: screenshot|click|type|scroll"},
  "x":{"type":"integer"},
  "y":{"type":"integer"},
  "text":{"type":"string"},
  "output":{"type":"string","description":"截图保存路径"},
  "button":{"type":"string"},
  "direction":{"type":"string"},
  "amount":{"type":"integer"}
},
"required":["action"]
}`)
}

func (computerUse) ReadOnly() bool { return false }

func (computerUse) CompactDescription() string { return "计算机操作: 截图/点击/输入/滚动" }
func (computerUse) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string"}},"required":["action"]}`)
}

func (c computerUse) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Action    string `json:"action"`
		X         int    `json:"x,omitempty"`
		Y         int    `json:"y,omitempty"`
		Text      string `json:"text,omitempty"`
		Output    string `json:"output,omitempty"`
		Button    string `json:"button,omitempty"`
		Direction string `json:"direction,omitempty"`
		Amount    int    `json:"amount,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}

	switch strings.ToLower(p.Action) {
	case "screenshot":
		return doScreenshot(p.Output)
	case "click":
		return doClick(p.X, p.Y, p.Button)
	case "type":
		return doTypeText(p.Text)
	case "scroll":
		return doScroll(p.Direction, p.Amount)
	default:
		return "", fmt.Errorf("unknown action: %s", p.Action)
	}
}

func doScreenshot(output string) (string, error) {
	if output == "" {
		output = "screenshot_" + strconv.FormatInt(time.Now().Unix(), 10) + ".png"
	}
	if !filepath.IsAbs(output) {
		wd, _ := os.Getwd()
		output = filepath.Join(wd, output)
	}

	ps := fmt.Sprintf(
		`Add-Type -AssemblyName System.Windows.Forms;`+
			`$b=[Windows.Forms.Screen]::PrimaryScreen.Bounds;`+
			`$bm=New-Object Drawing.Bitmap $b.Width,$b.Height;`+
			`$g=[Drawing.Graphics]::FromImage($bm);`+
			`$g.CopyFromScreen($b.X,$b.Y,0,0,$b.Size);$g.Dispose();`+
			`$bm.Save('%s',[Drawing.Imaging.ImageFormat]::Png);$bm.Dispose()`,
		output)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("截图失败: %w", err)
	}
	if _, err := os.Stat(output); err != nil {
		return "", fmt.Errorf("截图未保存: %w", err)
	}
	return fmt.Sprintf("截图已保存到: %s", output), nil
}

func doClick(x, y int, button string) (string, error) {
	ps := fmt.Sprintf(
		`[Windows.Forms.Cursor]::Position=New-Object Drawing.Point(%d,%d);`+
			`[Windows.Forms.SendKeys]::SendWait('{Click}')`, x, y)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("点击失败: %w", err)
	}
	return fmt.Sprintf("已点击 (%d, %d)", x, y), nil
}

func doTypeText(text string) (string, error) {
	ps := fmt.Sprintf(`$null=[Windows.Forms.SendKeys]::SendWait('%s')`, text)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("输入失败: %w", err)
	}
	return fmt.Sprintf("已输入 %d 个字符", len(text)), nil
}

func doScroll(direction string, amount int) (string, error) {
	if amount <= 0 {
		amount = 3
	}
	key := "{DOWN}"
	if strings.EqualFold(direction, "up") {
		key = "{UP}"
	}
	ps := fmt.Sprintf(
		`$w=New-Object -ComObject WScript.Shell;`+
			`for($i=0;$i-lt%d;$i++){$null=$w.SendKeys('%s')}`,
		amount, key)

	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("滚动失败: %w", err)
	}
	return fmt.Sprintf("已滚动 %s %d 次", direction, amount), nil
}
