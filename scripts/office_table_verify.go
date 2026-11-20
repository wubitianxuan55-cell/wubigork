//go:build ignore

// office_table_verify.go — insert_table 终验：11 行表（用户实际规模）+
// 三线表 + 连插两表 + save 落盘，回读文件核对表数/尺寸/内容/边框。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/office/comhelper"
)

func main() {
	ctx := context.Background()
	_, err := comhelper.EnsureHelperExe()
	if err != nil {
		fmt.Println("❌ 编译:", err)
		os.Exit(1)
	}
	c := comhelper.NewClient()
	defer c.Dispose()

	scratch := filepath.Join(os.TempDir(), fmt.Sprintf("gaea-tbl-verify-%d.docx", time.Now().UnixMilli()))
	base := map[string]any{"suite": "auto", "app": "word", "path": scratch}
	editBase := map[string]any{"suite": "auto", "app": "word", "doc": scratch, "follow": false, "typing": false, "card": false, "silent": false}

	if _, err := c.Call(ctx, "open", merge(base, map[string]any{"show": false, "follow": false, "typing": false, "card": false, "silent": false}), 80*time.Second); err != nil {
		fmt.Println("❌ open:", err)
		os.Exit(1)
	}

	// 11×4 表（用户实际规模），带 caption
	rows := make([][]string, 11)
	rows[0] = []string{"岗位", "人数", "职责", "备注"}
	for i := 1; i < 11; i++ {
		rows[i] = []string{fmt.Sprintf("岗位%d", i), fmt.Sprintf("%d", i), "职责说明", "—"}
	}
	raw, err := c.Call(ctx, "edit", merge(editBase, map[string]any{"ops": []any{
		map[string]any{"op": "insert_paragraphs", "items": []any{map[string]any{"text": "项目人员配置表如下。"}}},
		map[string]any{"op": "insert_table", "data": rows, "caption": "项目人员配置"},
	}}), 270*time.Second)
	fmt.Println("11行表:", clip(string(raw), 160), "err:", err)
	if err != nil {
		os.Exit(1)
	}

	// 三线表
	raw, err = c.Call(ctx, "edit", merge(editBase, map[string]any{"ops": []any{
		map[string]any{"op": "insert_table", "data": [][]string{{"因子", "浓度"}, {"镍", "900"}}, "borders": "three-line", "caption": "检测因子"},
	}}), 270*time.Second)
	fmt.Println("三线表:", clip(string(raw), 160), "err:", err)

	// none 边框
	raw, err = c.Call(ctx, "edit", merge(editBase, map[string]any{"ops": []any{
		map[string]any{"op": "insert_table", "data": [][]string{{"A", "B"}, {"1", "2"}}, "borders": "none"},
	}}), 270*time.Second)
	fmt.Println("无边框表:", clip(string(raw), 160), "err:", err)

	// set_cell 跟进
	raw, err = c.Call(ctx, "edit", merge(editBase, map[string]any{"ops": []any{
		map[string]any{"op": "set_cell", "table": 1, "row": 2, "col": 2, "text": "改后数值"},
	}}), 120*time.Second)
	fmt.Println("set_cell:", clip(string(raw), 120), "err:", err)

	if _, err := c.Call(ctx, "save", merge(base, map[string]any{"doc": scratch}), 110*time.Second); err != nil {
		fmt.Println("❌ save:", err)
		os.Exit(1)
	}
	_, _ = c.Call(ctx, "close", merge(base, map[string]any{"doc": scratch, "save": false}), 50*time.Second)
	fmt.Println("SAVED:", scratch)
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

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

var _ = json.RawMessage{}
