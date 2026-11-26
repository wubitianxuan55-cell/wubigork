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

// 2026-11-26 造价反馈（百锦路 #1）：账本只追加、没有「已取回」态，文件被
// move 走后批次仍显示 quarantined，模型照账本去取 source not found。每件要按
// 盘面如实报还在不在，落点要给到件。
func TestQuarantineListReportsOnDiskPresence(t *testing.T) {
	dir := t.TempDir()
	q := filepath.Join(dir, ".gaea", "publish-quarantine")
	stamp := "20261126-130035.696875400"
	file := filepath.Join(q, stamp, ".gaea", "exports", "六表工料法.xlsx")
	os.MkdirAll(filepath.Dir(file), 0o755)
	os.WriteFile(file, []byte("x"), 0o644)
	ledger := `{"stamp":"` + stamp + `","event":"final","status":"quarantined","source":"foreground","entries":[{"rel":".gaea/exports/六表工料法.xlsx","kind":"modified"}]}`
	os.WriteFile(filepath.Join(q, "ledger.jsonl"), []byte(ledger), 0o644)

	out, err := quarantineList{workDir: dir}.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "在隔离区：是") {
		t.Errorf("文件还在时应报「是」：%s", out)
	}
	if !strings.Contains(out, "每件完整路径 = 落点/原相对路径") {
		t.Errorf("落点应说明文件在批次目录的镜像子路径下：%s", out)
	}
	if strings.Contains(out, "工作区外暂存位") {
		t.Errorf("不得再建议移到工作区外取回：%s", out)
	}

	// 取走后：账本没变，盘面变了，按盘面报。
	os.Remove(file)
	out, err = quarantineList{workDir: dir}.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "在隔离区：否") || strings.Contains(out, "在隔离区：是") {
		t.Errorf("取走后应报「否」：%s", out)
	}
}
