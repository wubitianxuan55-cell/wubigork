package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/feedback"
)

// AI 反馈工具（2026-11-20 立项）：遇困难主动提反馈→记忆面板反馈区→导出 md。

func TestAIFeedback_Execute(t *testing.T) {
	dir := t.TempDir()
	old := feedbackUserDir
	feedbackUserDir = func() string { return dir }
	defer func() { feedbackUserDir = old }()

	out, err := aiFeedback{}.Execute(context.Background(), json.RawMessage(
		`{"title":"read_file 采样危险","detail":"核对数字时采样片段被误读为连续","category":"tool-friction","severity":"high","board":"office"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "反馈已记录") {
		t.Fatalf("回执应确认入库: %s", out)
	}
	items := feedback.List(dir)
	if len(items) != 1 || items[0].Category != "tool-friction" || items[0].Board != "office" {
		t.Fatalf("入库条目不符: %+v", items)
	}
	if _, err := os.Stat(filepath.Join(dir, "ai-feedback.jsonl")); err != nil {
		t.Fatalf("库文件应存在: %v", err)
	}
}

func TestAIFeedback_Validation(t *testing.T) {
	dir := t.TempDir()
	old := feedbackUserDir
	feedbackUserDir = func() string { return dir }
	defer func() { feedbackUserDir = old }()
	exec := func(args string) error {
		_, err := aiFeedback{}.Execute(context.Background(), json.RawMessage(args))
		return err
	}
	cases := []struct {
		name, args, want string
	}{
		{"空 title", `{"title":"","detail":"x"}`, "title"},
		{"空 detail", `{"title":"x","detail":""}`, "detail"},
		{"坏分类", `{"title":"x","detail":"y","category":"noise"}`, "category"},
		{"坏严重度", `{"title":"x","detail":"y","severity":"critical"}`, "severity"},
		{"坏板块", `{"title":"x","detail":"y","board":"dsh"}`, "board"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := exec(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("应拒绝（含 %q）: %v", tc.want, err)
			}
		})
	}
	if got := feedback.List(dir); len(got) != 0 {
		t.Fatalf("拒绝的调用不得入库: %+v", got)
	}
}

// 契约面：shared 装配（三板块）+只读权限（写 gaea 自管数据区非用户文件）。
func TestAIFeedback_SpaceAndContract(t *testing.T) {
	var v aiFeedback
	if v.SpaceTag() != "shared" {
		t.Fatalf("SpaceTag=%q, want shared（三板块装配）", v.SpaceTag())
	}
	if !v.ReadOnly() {
		t.Fatal("写 gaea 自管数据区按只读放行（journal 同语义）")
	}
	if !strings.Contains(v.Description(), "不要等用户要求") {
		t.Fatal("描述必须鼓励主动反馈")
	}
}
