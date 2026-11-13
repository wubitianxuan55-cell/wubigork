package builtin

// parse_error_style_test.go — 参数错误三要素守卫（灵犀 advantages #12 蒸馏，
// docs/gaea-lingxi-advantages-2026-11.md）：
// 高频工具的参数解析失败必须以「<工具名> 参数解析失败: <底层错误>」呈现——
// 工具名让模型知道该重读哪份 schema，字段路径与期望类型由底层 json 错误携带，
// 合法值在 schema 里。坏参在触碰任何 IO/会话态之前就返回，本表全 hermetic。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseErrorsCarryToolName(t *testing.T) {
	cases := []struct {
		tool string
		run  func() error
	}{
		{"read_file", func() error { _, err := (readFile{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"write_file", func() error { _, err := (writeFile{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"edit_file", func() error { _, err := (editFile{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"multi_edit", func() error { _, err := (multiEdit{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"edit_lines", func() error { _, err := (editLines{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"replace", func() error { _, err := (replaceTool{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"grep", func() error { _, err := (grepTool{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"ls", func() error { _, err := (listDir{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"bash", func() error { _, err := (bash{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
		{"read_skill", func() error { _, err := (readSkill{}).Execute(context.Background(), json.RawMessage(`{bad`)); return err }},
	}
	for _, tc := range cases {
		err := tc.run()
		if err == nil {
			t.Fatalf("%s：坏参数应报错", tc.tool)
		}
		if !strings.Contains(err.Error(), tc.tool) || !strings.Contains(err.Error(), "参数解析失败") {
			t.Fatalf("%s 解析错误缺工具名或固定短语（错误三要素守卫），实际：%v", tc.tool, err)
		}
	}
}
