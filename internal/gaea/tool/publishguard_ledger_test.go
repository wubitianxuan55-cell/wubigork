package tool

// D6 清理债务账本测试（dsh-ops 蒸馏 2026-11-10 第二波）：先记账后放句柄、
// 只删账本背书的过期 stamp、账本外名字整目录跳过、失败留账。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func d6Guard(t *testing.T) (*PublishGuard, string) {
	t.Helper()
	ws := t.TempDir()
	q := filepath.Join(ws, ".gaea", "publish-quarantine")
	return NewPublishGuard([]string{ws}, q, nil).WithSource("test", "echo x > out.docx"), q
}

func d6Violations(t *testing.T, g *PublishGuard, ws, rel, content string) []PublishViolation {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(ws, filepath.FromSlash(rel))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	vs, err := g.Scan()
	if err != nil || len(vs) == 0 {
		t.Fatalf("应检出越界: %v %v", vs, err)
	}
	return vs
}

func d6LedgerRecords(t *testing.T, q string) map[string]quarantineRecord {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(q, quarantineLedgerFile))
	if err != nil {
		t.Fatal(err)
	}
	recs := map[string]quarantineRecord{}
	for _, line := range strings.Split(string(raw), "\n") {
		if line == "" {
			continue
		}
		var r quarantineRecord
		if json.Unmarshal([]byte(line), &r) == nil {
			recs[r.Status] = r
		}
	}
	return recs
}

func TestQuarantineLedgerPendingThenFinal(t *testing.T) {
	g, q := d6Guard(t)
	ws := g.roots[0]
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	vs := d6Violations(t, g, ws, "report.docx", "payload")
	if _, err := g.Quarantine(vs); err != nil {
		t.Fatal(err)
	}
	recs := d6LedgerRecords(t, q)
	if _, ok := recs["pending"]; !ok {
		t.Fatal("pending 意图记录必须先于移动落账")
	}
	fin, ok := recs["quarantined"]
	if !ok {
		t.Fatal("终态记录缺失")
	}
	if fin.Source != "test" || fin.Command == "" || len(fin.Entries) != 1 || fin.Entries[0].RelPath != "report.docx" {
		t.Fatalf("账本归属证据不全: %+v", fin)
	}
	dest := g.LastQuarantineDest()
	if !strings.HasPrefix(dest, q) {
		t.Fatalf("目的地回显错: %q", dest)
	}
	// 隔离件在账本声明的位置。
	if _, err := os.Stat(filepath.Join(dest, "report.docx")); err != nil {
		t.Fatalf("隔离件应在目的地: %v", err)
	}
}

func TestSweepQuarantineRules(t *testing.T) {
	g, q := d6Guard(t)
	ws := g.roots[0]
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Quarantine(d6Violations(t, g, ws, "old/report.docx", "old")); err != nil {
		t.Fatal(err)
	}
	stamp := g.LastQuarantineDest()

	// 未到期：不删。
	if s, f, err := SweepQuarantine(q, time.Hour); err != nil || s != 0 || f != 0 {
		t.Fatalf("保留期内不得清扫: %d %d %v", s, f, err)
	}
	// 到期：账本背书，删 stamp 目录。
	if s, f, err := SweepQuarantine(q, -time.Hour); err != nil || s != 1 || f != 1 {
		t.Fatalf("到期应清扫: %d %d %v", s, f, err)
	}
	if _, err := os.Stat(stamp); !os.IsNotExist(err) {
		t.Fatal("stamp 目录应被删除")
	}
	recs := d6LedgerRecords(t, q)
	if _, ok := recs["swept"]; !ok {
		t.Fatal("清扫完成应记 swept")
	}

	// 账本外名字：新隔离到期后，往 stamp 里塞账本外文件 → 整目录跳过。
	g2, _ := d6Guard(t)
	g2.quarantine = q
	g2.roots = []string{ws}
	// 手造一个「账本背书」stamp+账本外文件。
	led := beginQuarantineRecord(q, "test", "cmd", []PublishViolation{{Path: "x", RelPath: "extra/rogue.bin", Root: ws, Kind: "created"}})
	led.finish([]string{"x"}, nil)
	stamp2 := filepath.Join(q, led.rec.Stamp)
	if err := os.MkdirAll(filepath.Join(stamp2, "extra"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stamp2, "extra", "rogue.bin"), []byte("rogue"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stamp2, "UNEXPECTED.md"), []byte("not in ledger"), 0o644); err != nil {
		t.Fatal(err)
	}
	if s, _, err := SweepQuarantine(q, -time.Hour); err != nil || s != 0 {
		t.Fatalf("账本外名字应整目录跳过: %d %v", s, err)
	}
	if _, err := os.Stat(filepath.Join(stamp2, "UNEXPECTED.md")); err != nil {
		t.Fatal("账本外文件不得被删")
	}
}

func TestSweepSkipsOwnerlessStamps(t *testing.T) {
	// 无账本的存量 stamp：一律不动（手动处理）。
	q := t.TempDir()
	stamp := filepath.Join(q, "20250101-000000.000000000")
	if err := os.MkdirAll(stamp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stamp, "legacy.docx"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if s, f, err := SweepQuarantine(q, -time.Hour); err != nil || s != 0 || f != 0 {
		t.Fatalf("无账本不得动: %d %d %v", s, f, err)
	}
	if _, err := os.Stat(filepath.Join(stamp, "legacy.docx")); err != nil {
		t.Fatal("存量件不得被删")
	}
}
