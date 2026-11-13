//go:build ignore

// office_com_smoke.go — dsh-office COM 档真机冒烟（波1）：编译 helper →
// status → 新建 scratch docx → 批量写入 → 保存 → 渲染 → 读取 → 关闭。
// 全程只操作本脚本自己在系统临时目录新建的文档（上游 office-smoke 同纪律），
// 走查完即清。运行：go run scripts/office_com_smoke.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/office/comhelper"
	"github.com/gaea/gaea/internal/office/omml"
)

func must(err error, what string) {
	if err != nil {
		fmt.Printf("❌ %s: %v\n", what, err)
		os.Exit(1)
	}
	fmt.Printf("✅ %s\n", what)
}

func main() {
	comhelper.Logger = func(m string) { fmt.Printf("  [helper] %s\n", m) }
	ctx := context.Background()

	exe, err := comhelper.EnsureHelperExe()
	must(err, "编译 helper（csc.exe，首跑约 1s）")
	fmt.Printf("   exe=%s\n", exe)

	c := comhelper.NewClient()
	defer c.Dispose()

	raw, err := c.Call(ctx, "status", map[string]any{"suite": "auto"}, 30*time.Second)
	must(err, "status")
	var apps []map[string]any
	_ = json.Unmarshal(raw, &apps)
	for _, a := range apps {
		fmt.Printf("   %-5s suite=%v installed=%v running=%v docs=%d\n",
			a["app"], a["suite"], a["installed"], a["running"], len(a["documents"].([]any)))
	}

	scratch := filepath.Join(os.TempDir(), fmt.Sprintf("gaea-office-smoke-%d.docx", time.Now().UnixMilli()))
	base := map[string]any{"suite": "auto", "app": "word", "path": scratch}

	raw, err = c.Call(ctx, "open", merge(base, map[string]any{"show": true, "follow": true, "typing": true, "card": false, "silent": false}), 80*time.Second)
	must(err, fmt.Sprintf("open scratch（窗口应短暂打开）: %s", scratch))
	fmt.Printf("   open 回执: %.160s\n", string(raw))

	raw, err = c.Call(ctx, "edit", merge(map[string]any{"suite": "auto", "app": "word", "doc": scratch, "follow": true, "typing": true, "card": false, "silent": false},
		map[string]any{"ops": []map[string]any{
			{"op": "insert_paragraphs", "items": []map[string]any{
				{"text": "gaea office COM 档冒烟"},
				{"text": "这一段由 COM 实时写入 Word/WPS 窗口，格式由应用自己排版。"},
			}},
		}}), 270*time.Second)
	must(err, "edit 批量写入")
	fmt.Printf("   edit 回执: %.200s\n", string(raw))

	// 波2 math 腿实弹：omml 预转公式（模拟工具层 FormulasOf→math map），
	// 第二段 edit 写入带 $...$ 公式的文本。suite=wps 强制走 WPS Writer
	//（helper 的 WpsMath 消费 math map；MS Word 档走 Word 自带解析不用 map）。
	opsAny := []any{map[string]any{"op": "insert_paragraphs", "items": []any{
		map[string]any{"text": "质能方程 $E=mc^2$ 与矩阵 $$\\begin{pmatrix} a & b \\\\ c & d \\end{pmatrix}$$ 由 omml 预转。"},
	}}}
	math := omml.FormulasOf(opsAny, nil)
	if len(math) == 0 {
		fmt.Println("❌ omml.FormulasOf 未产出 math map")
		os.Exit(1)
	}
	fmt.Printf("   math map %d 项: %v\n", len(math), keysOf(math))
	raw, err = c.Call(ctx, "edit", merge(map[string]any{"suite": "wps", "app": "word", "doc": scratch, "follow": true, "typing": true, "card": false, "silent": false},
		map[string]any{"ops": opsAny, "math": math}), 270*time.Second)
	must(err, "edit 公式段（WPS math 腿）")
	fmt.Printf("   edit 回执: %.200s\n", string(raw))

	raw, err = c.Call(ctx, "save", merge(base, map[string]any{"doc": scratch}), 110*time.Second)
	must(err, "save 落盘")
	fmt.Printf("   save 回执: %s\n", string(raw))

	png := filepath.Join(os.TempDir(), fmt.Sprintf("gaea-office-smoke-%d.png", time.Now().UnixMilli()))
	raw, err = c.Call(ctx, "render", merge(base, map[string]any{"doc": scratch, "page": 1, "out": png, "width": 800}), 50*time.Second)
	must(err, "render 第 1 页")
	fmt.Printf("   render 回执: %s\n", string(raw))

	raw, err = c.Call(ctx, "read", merge(base, map[string]any{"doc": scratch}), 50*time.Second)
	must(err, "read 读取")
	fmt.Printf("   read 回执: %.200s\n", string(raw))

	raw, err = c.Call(ctx, "close", merge(base, map[string]any{"doc": scratch, "save": false}), 50*time.Second)
	must(err, "close（窗口应关闭）")

	fmt.Println("冒烟全链通过（open→edit→save→render→read→close）。")
}

func merge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func keysOf(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
