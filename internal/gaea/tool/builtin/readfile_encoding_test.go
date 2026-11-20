package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadFileLargeUTF8ChineseNoMojibake（2026-11-20 实测回归）：>256KB 的
// 严格 UTF-8 中文文件，检测采样截断点（readFileDetectSample=256KB）恰切在
// 多字节汉字中间时，旧实现 utf8.Valid 判非 → Detect 级联误落 GB18030 →
// 整档按 GBK 解码成静默乱码（「对象、范围与边界」→「瀵硅薄銆佽寖鍥翠笌杈圭晫」，
// 解码还「成功」）。修复=采样尾回退到 rune 边界再判型。
func TestReadFileLargeUTF8ChineseNoMojibake(t *testing.T) {
	// 构造：ASCII 前缀 2 字节 + 大量 3 字节汉字，令 256KB 边界落在汉字中间
	//（(262144-2) % 3 = 1 → 边界切中 rune 第 2 字节）。
	var body strings.Builder
	body.WriteString("ab")
	unit := "对象、范围与边界" // 8 个 rune、24 字节，全部 3 字节
	for body.Len() <= readFileDetectSample+1024 {
		body.WriteString(unit)
		body.WriteString("\n")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "tech-plan.md")
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (readFile{workDir: dir}).Execute(context.Background(),
		json.RawMessage(`{"path":"tech-plan.md","limit":5,"line_numbers":false}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 修复后：严格 UTF-8 直读，首行原文保留；修复前：整档 GBK 乱码
	//（首行 "ab" + 汉字 → "ab瀵硅薄..."）。
	if !strings.HasPrefix(out, "ab对象") {
		t.Errorf("大 UTF-8 中文文件被误判编码出乱码，got head %q", clip(out, 40))
	}
	if strings.Contains(out, "瀵硅薄") {
		t.Errorf("检出 GBK 乱码特征串「瀵硅薄」——采样截断修复未生效")
	}
}

// TestReadFileLargeGB18030StillDecoded：回归钉——真 GBK 文件（含中文、无
// BOM）仍然走 GB18030 解码，rune 边界回退不误伤真 GBK 判定。
func TestReadFileLargeGB18030StillDecoded(t *testing.T) {
	// "对象、范围与边界" 的 GB18030 字节，重复到 >256KB，行尾 \n（ASCII）。
	gbkLine := []byte{0xB6, 0xD4, 0xCF, 0xF3, 0xA1, 0xA2, 0xB7, 0xB6, 0xCE, 0xA7, 0xD3, 0xEB, 0xB1, 0xDF, 0xBD, 0xE7}
	var buf []byte
	for len(buf) <= readFileDetectSample+1024 {
		buf = append(buf, gbkLine...)
		buf = append(buf, '\n')
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "gbk.md")
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := (readFile{workDir: dir}).Execute(context.Background(),
		json.RawMessage(`{"path":"gbk.md","limit":3,"line_numbers":false}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "对象、范围与边界") {
		t.Errorf("真 GB18030 文件未被解码，got head %q", clip(out, 40))
	}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
