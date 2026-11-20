package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

// 实测反馈⑦：覆盖旧文件时 write_file 一直都在拍写前快照（writefile.go
// StageBaseline），但回执只报 wrote N bytes——用户以为「文件没有快照」。
// 回执必须显式报 baseline 路径与保留份数；新文件（无旧件）回执保持原样。
func TestWriteFileReceiptReportsBaseline(t *testing.T) {
	ws, w, _ := casWorkspace(t)
	led := evidence.NewChangeLedger()
	led.SetBaselineDir(filepath.Join(ws, ".gaea", "work", "rollback"))
	ctx := evidence.WithChanges(context.Background(), led)

	// 新文件：无 baseline 行
	out, err := w.Execute(ctx, casArgs(t, map[string]any{"path": "new.md", "content": "v0"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "baseline") {
		t.Fatalf("新文件不应有 baseline 行: %s", out)
	}

	// 覆盖：回执两行——wrote + baseline 路径与保留份数
	out, err = w.Execute(ctx, casArgs(t, map[string]any{"path": "new.md", "content": "v1"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "baseline: ") || !strings.Contains(out, ".before") {
		t.Fatalf("覆盖回执缺 baseline 路径: %s", out)
	}
	// 报出的快照真实存在（诚实回执）
	line := out[strings.Index(out, "baseline: ")+len("baseline: "):]
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	if i := strings.IndexAny(line, "（("); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if _, err := os.Stat(line); err != nil {
		t.Fatalf("回执报的快照路径不存在 %q: %v", line, err)
	}
}
