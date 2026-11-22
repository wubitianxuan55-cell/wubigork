package builtin

// quarantine_list 回归钉（P1-12 建议④）：按 stamp 去重（一行=一次状态变更，
// 行数≠批次数——实测 74 行 37 批次被误记「70 条」）；status 过滤；空态诚实。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"encoding/json"
)

func TestQuarantineListDedupsByStamp(t *testing.T) {
	dir := t.TempDir()
	q := filepath.Join(dir, ".gaea", "publish-quarantine")
	os.MkdirAll(q, 0o755)
	// 造两批次账本：每批 begin+final 两行（begin 行 status=pending）。
	ledger := `{"stamp":"20261121-100000.000000000","event":"begin","status":"pending","source":"foreground","entries":[{"rel":"a.docx","kind":"created"}]}
{"stamp":"20261121-100000.000000000","event":"final","status":"quarantined","source":"foreground","entries":[{"rel":"a.docx","kind":"created"}]}
{"stamp":"20261121-110000.000000000","event":"begin","status":"pending","source":"background","entries":[{"rel":"b/整册.docx","kind":"created"}]}
{"stamp":"20261121-110000.000000000","event":"final","status":"partial","source":"background","entries":[{"rel":"b/整册.docx","kind":"created"}]}
`
	if err := os.WriteFile(filepath.Join(q, "ledger.jsonl"), []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := quarantineList{workDir: dir}.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// 去重：两批次、终态取最后一条（quarantined/partial），不得出现 pending 面。
	for _, want := range []string{"2 批次", "quarantined", "partial", "a.docx", "b/整册.docx", "background"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执缺 %q：%s", want, out)
		}
	}
	if strings.Count(out, "pending]") != 0 {
		t.Errorf("同 stamp 应取终态行，pending 不得出现：%s", out)
	}
	// status 过滤。
	out2, err := quarantineList{workDir: dir}.Execute(context.Background(), json.RawMessage(`{"status":"quarantined"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "1 批次") || strings.Contains(out2, "partial") {
		t.Errorf("status 过滤应只留 1 批：%s", out2)
	}
	// 空态诚实。
	out3, _ := quarantineList{workDir: t.TempDir()}.Execute(context.Background(), nil)
	if !strings.Contains(out3, "no quarantine") {
		t.Errorf("空态应诚实：%s", out3)
	}
}
