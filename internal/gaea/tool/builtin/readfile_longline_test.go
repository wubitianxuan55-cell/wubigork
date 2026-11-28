package builtin

// readfile_longline_test.go — 2026-11-28 黄甲反馈回归钉：转换产物里内嵌的 base64
// 图片让单行超过 1MB（bufio.Scanner 上限），此前整次读取报一句 token too long，连
// 窗口之后那一行超长都读不了；同一份素材 md 连撞 4 次，模型只能自己写脚本剥图。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLongLineFile 写一份 60 行的 md：第 31 行是约 1.2MB 的内嵌图片，其余是短行。
func writeLongLineFile(t *testing.T) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	var b strings.Builder
	for i := 1; i <= 60; i++ {
		if i == 31 {
			b.WriteString("![图](data:image/png;base64," + strings.Repeat("QUJD", 300000) + ")\n")
			continue
		}
		fmt.Fprintf(&b, "line%d\n", i)
	}
	path = filepath.Join(dir, "long.md")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

// 窗口在超长行之前：窗口内的行照常返回，并说明下一行读不动、怎么读。
func TestReadFileLongLineRightAfterWindow(t *testing.T) {
	dir, _ := writeLongLineFile(t)
	out, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"long.md","offset":10,"limit":20}`))
	if err != nil {
		t.Fatalf("超长行在窗口之后不该让整次读取失败：%v", err)
	}
	for _, want := range []string{"11→line11", "30→line30", "pass offset=30 to continue", "[第 31 行单行超过 1MB 未读取", "byte_offset/max_bytes", "format_convert"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执应含 %q", want)
		}
	}
	if strings.Contains(out, "line32") {
		t.Errorf("超长行之后的行不该被读出（翻页时再报）")
	}
}

// 超长行落在窗口里：读到的行照常返回，缺的那一行点名。
func TestReadFileLongLineInsideWindow(t *testing.T) {
	dir, _ := writeLongLineFile(t)
	out, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"long.md","offset":25,"limit":10}`))
	if err != nil {
		t.Fatalf("超长行在窗口里不该丢掉已读到的行：%v", err)
	}
	for _, want := range []string{"26→line26", "30→line30", "[第 31 行单行超过 1MB 未读取"} {
		if !strings.Contains(out, want) {
			t.Errorf("回执应含 %q", want)
		}
	}
}

// 超长行就是窗口的第一行：报错，但给出能照着做的两条路。
func TestReadFileLongLineAtWindowStartGivesGuidance(t *testing.T) {
	dir, _ := writeLongLineFile(t)
	_, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"long.md","offset":30,"limit":10}`))
	if err == nil {
		t.Fatal("窗口里一行都读不到应报错")
	}
	for _, want := range []string{"单行超过 1MB", "byte_offset", "format_convert"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("报错应含 %q：%v", want, err)
		}
	}
	// 字节窗口照常可读超长行所在位置。
	out, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"long.md","byte_offset":0,"max_bytes":12}`))
	if err != nil || !strings.Contains(out, "line1") {
		t.Fatalf("字节窗口应能读：out=%q err=%v", out, err)
	}
}

// 要读全文件的形态（尾部窗口、section）读不动超长行：报错并指路。
func TestReadFileLongLineWholeFileModesGiveGuidance(t *testing.T) {
	dir, _ := writeLongLineFile(t)
	_, err := (readFile{workDir: dir}).Execute(context.Background(), json.RawMessage(`{"path":"long.md","offset":-5}`))
	if err == nil || !strings.Contains(err.Error(), "byte_offset") {
		t.Fatalf("尾部窗口遇超长行应报错指路：%v", err)
	}
}
