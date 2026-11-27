package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// 2026-11-27 AI 反馈 #5：单行超长内容（一行 26KB 的 JSON）行分页读不全——
// byte_offset / max_bytes 按字节严格分页、不采样，多字节字符不被切成半个字。
func TestReadFileByteWindow(t *testing.T) {
	dir := t.TempDir()
	// 一行里混着多字节字符：窗口边界落在汉字中间才算数。
	unit := "abc-def-ghi-jkl-mno-pqr-stu-vwx-yz0-12345-数据项-"
	var sb strings.Builder
	for sb.Len() < 40*1024 {
		sb.WriteString(unit)
	}
	want := "[" + sb.String() + "]"
	path := filepath.Join(dir, "one-line.json")
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	r := readFile{workDir: dir}
	read := func(args string) string {
		t.Helper()
		out, err := r.Execute(context.Background(), json.RawMessage(args))
		if err != nil {
			t.Fatalf("read_file %s 失败: %v", args, err)
		}
		return out
	}

	// 只给 max_bytes：从 0 起读一个窗口，尾部给续读游标。
	out := read(`{"path":"one-line.json","max_bytes":1000}`)
	head := strings.SplitN(out, "\n[bytes ", 2)[0]
	if len(head) > 1000 || len(head) < 990 || !strings.HasPrefix(want, head) {
		t.Fatalf("首个窗口应是文件头的约 1000 字节（对齐字符边界）：%d 字节", len(head))
	}

	// 顺着游标拼回去：与原文逐字节一致。
	var got strings.Builder
	got.WriteString(head)
	for i := 0; i < 80; i++ {
		idx := strings.Index(out, "pass byte_offset=")
		if idx < 0 {
			break // 没有续读游标 = 已读到 EOF
		}
		rest := out[idx+len("pass byte_offset="):]
		if sp := strings.IndexAny(rest, " "); sp > 0 {
			rest = rest[:sp]
		}
		next, err := strconv.Atoi(rest)
		if err != nil {
			t.Fatalf("续读游标应为整数：%q", rest)
		}
		out = read(`{"path":"one-line.json","byte_offset":` + strconv.Itoa(next) + `,"max_bytes":8000}`)
		if idx := strings.Index(out, "\n[bytes "); idx >= 0 {
			got.WriteString(out[:idx])
		} else {
			got.WriteString(out)
		}
	}
	if got.String() != want {
		t.Fatalf("按游标拼回应与原文一致：得到 %d 字节 / 原文 %d 字节", got.Len(), len(want))
	}

	// 上限：max_bytes 再大也只给 24K（压在输出层 32KiB 压缩线内）。
	out = read(`{"path":"one-line.json","byte_offset":0,"max_bytes":999999}`)
	if n := len(strings.SplitN(out, "\n[bytes ", 2)[0]); n > readFileByteWindowMax || n < readFileByteWindowMax-4 {
		t.Fatalf("窗口应封顶 %d 字节，得到 %d", readFileByteWindowMax, n)
	}
	// 越界与互斥。
	if out := read(`{"path":"one-line.json","byte_offset":99999999}`); !strings.Contains(out, "past EOF") {
		t.Fatalf("越界应如实报告：%q", out)
	}
	if _, err := r.Execute(context.Background(), json.RawMessage(`{"path":"one-line.json","byte_offset":10,"offset":3}`)); err == nil || !strings.Contains(err.Error(), "互斥") {
		t.Fatalf("字节窗口与行分页应互斥：%v", err)
	}
	if _, err := r.Execute(context.Background(), json.RawMessage(`{"path":"one-line.json","byte_offset":-1}`)); err == nil {
		t.Fatal("负 byte_offset 应报错")
	}
}
