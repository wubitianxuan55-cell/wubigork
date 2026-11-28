package builtin

// bash_relocation_test.go — 2026-11-28 黄甲反馈回归钉：工作区内 mv 目录（目录里
// 有既有 Office 件）不再被发布边界整体隔离，回执说明哪些件是随迁。此前实测 7 批
// 拦截、主控逐件 move_file 取回 120 次。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashPublishRelocationWithinWorkspacePasses(t *testing.T) {
	ws := publishBashWorkdir(t)
	logs := filepath.Join(ws, "logs")
	impl := filepath.Join(ws, "03_impl")
	for _, d := range []string{logs, impl} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"SG-01.docx", "SG-02.docx"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(logs, n), []byte("log"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b := bash{workDir: ws}

	// bash/pwsh 两形态都成立：目标父目录已存在，mv 即改名落位。
	out, err := b.Execute(context.Background(), []byte(`{"command":"mv logs 03_impl/logs"}`))
	if err != nil {
		t.Fatalf("工作区内目录移动不该被拦：%v\n%s", err, out)
	}
	for _, want := range []string{
		"发布边界：2 件既有 Office 件随工作区内移动换了位置",
		"logs/SG-01.docx → 03_impl/logs/SG-01.docx",
		"logs/SG-02.docx → 03_impl/logs/SG-02.docx",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("回执应含 %q：\n%s", want, out)
		}
	}
	for _, n := range names {
		if _, serr := os.Stat(filepath.Join(impl, "logs", n)); serr != nil {
			t.Errorf("%s 应留在新位置：%v", n, serr)
		}
	}
	if q, _ := filepath.Glob(filepath.Join(ws, ".gaea", "publish-quarantine", "*", "logs", "*")); len(q) != 0 {
		t.Errorf("随迁件不该进隔离区：%v", q)
	}
}
