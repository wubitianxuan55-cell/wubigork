package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
)

// 2026-11-21 反馈 P0-1 根修回归钉：拦截文案是多行的（文件清单/隔离位置/取回
// 方式/合规出口），而 agent 收口层信封 error 字段只取 err 首行、detail 字段取
// 工具 result——此前 bash 错误路径 result 恒空串，模型只看到「已隔离：」断头句
// +空 detail（实测 37 批次全盲）。文案必须同时作为 result 返回。
func TestBash_PublishViolationNoteRidesInResult(t *testing.T) {
	shell := sandbox.ResolveShell()
	dir := t.TempDir()
	b := bash{shell: shell, workDir: dir}
	args, _ := json.Marshal(map[string]any{
		"command": "echo payload > probe.docx",
	})
	out, err := b.Execute(context.Background(), args)
	if err == nil {
		t.Skipf("发布边界未拦截（shell=%v 工作区写未生效），跳过传输断言", shell.Kind)
	}
	// result 携带全文：清单行+隔离位置+取回方式都不得被首行截断丢掉。
	for _, want := range []string{"发布边界拦截", "probe.docx", "隔离位置", "取回"} {
		if !strings.Contains(out, want) {
			t.Errorf("拦截文案应经 result 传输（信封 detail），缺 %q：\n%s", want, out)
		}
	}
	if !strings.HasPrefix(err.Error(), "发布边界拦截") {
		t.Errorf("err 首行（信封 error 字段）应为拦截标题，实得: %v", err)
	}
	// 越界件确已被隔离出正式工作区。
	if _, serr := os.Stat(filepath.Join(dir, "probe.docx")); serr == nil {
		t.Error("越界 durable 件应已移出正式工作区")
	}
}
