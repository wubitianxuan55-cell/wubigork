package app

// wx_file_handler_test.go — 微信入站文件自持 + 内容提取（v4.41）：复制自持/
// 上限防线/截断/不支持格式诚实文案/提取失败降级/清理策略/构造接线。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newWxFileHandlerTestStore 直构测试仓库（临时目录 + 小上限可选）。
func newWxFileHandlerTestStore(t *testing.T, maxFile int64) (*wxFileStore, string) {
	t.Helper()
	root := t.TempDir()
	s := &wxFileStore{dir: filepath.Join(root, "wx_files"), maxFile: maxFile}
	if maxFile <= 0 {
		s.maxFile = wxFileMaxBytes
	}
	return s, root
}

// 复制自持：txt 源 → wx_files/ 下 FNV 哈希名 + .txt 扩展名 + 内容原样；注入行
// 带文件名头与提取正文。
func TestWxFileStore_PersistAndInjectTxt(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	src := filepath.Join(t.TempDir(), "笔记.txt")
	if err := os.WriteFile(src, []byte("第一行\n第二行"), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	got := s.Ingest(src, "笔记.txt", 28, "d-1")
	if !strings.HasPrefix(got, "[用户发来文件 笔记.txt（") {
		t.Fatalf("注入行头不符：%q", got)
	}
	if !strings.Contains(got, "]\n第一行\n第二行") {
		t.Fatalf("注入行应含提取正文：%q", got)
	}
	ents, err := os.ReadDir(s.dir)
	if err != nil || len(ents) != 1 {
		t.Fatalf("wx_files 应恰有 1 个自持副本: %v %+v", err, ents)
	}
	name := ents[0].Name()
	if !strings.HasPrefix(name, "wx-file-") || !strings.HasSuffix(name, ".txt") {
		t.Fatalf("自持名应为 wx-file-<hash>-<ns>.txt：%q", name)
	}
	data, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil || string(data) != "第一行\n第二行" {
		t.Fatalf("自持副本内容应原样: %q %v", data, err)
	}
}

// 50MiB 二次防线：超上限的源文件拒绝复制并诚实告知（自持目录不落任何文件）。
func TestWxFileStore_OverLimitRefused(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 100) // 上限 100 字节
	src := filepath.Join(t.TempDir(), "big.bin")
	if err := os.WriteFile(src, make([]byte, 201), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	got := s.Ingest(src, "big.bin", 201, "d-2")
	if !strings.Contains(got, "收到文件 big.bin") || !strings.Contains(got, "超出") {
		t.Fatalf("超限应诚实告知：%q", got)
	}
	if ents, _ := os.ReadDir(s.dir); len(ents) != 0 {
		t.Fatalf("超限不应落盘: %+v", ents)
	}
}

// 复制失败（源文件消失）诚实降级，不 panic、不返回空串。
func TestWxFileStore_CopyFailHonest(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	got := s.Ingest(filepath.Join(t.TempDir(), "gone.bin"), "gone.bin", 10, "d-3")
	if got == "" || !strings.Contains(got, "保存失败") {
		t.Fatalf("复制失败应诚实降级：%q", got)
	}
}

// 截断：>6000 字符的提取内容截到 6000 字 + 「…(内容过长已截断)」标注。
func TestWxFileStore_Truncation(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	src := filepath.Join(t.TempDir(), "long.txt")
	long := strings.Repeat("字", 7000)
	if err := os.WriteFile(src, []byte(long), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	got := s.Ingest(src, "long.txt", int64(len(long)), "d-4")
	if !strings.Contains(got, "…(内容过长已截断)") {
		t.Fatalf("超限应标注截断：%q…", got[:80])
	}
	// 头 6000 字保留、第 7000 字不出现。
	if !strings.Contains(got, strings.Repeat("字", 6000)) {
		t.Fatalf("应保留前 6000 字")
	}
	if strings.Contains(got, strings.Repeat("字", 6001)) {
		t.Fatalf("第 6001 字起应被截断")
	}
}

// 不支持的格式：诚实文案 + 自持路径（可在桌面端打开）。
func TestWxFileStore_UnsupportedFormatHonest(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	src := filepath.Join(t.TempDir(), "model.zip")
	if err := os.WriteFile(src, []byte("PK\x03\x04fake"), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	got := s.Ingest(src, "model.zip", 9, "d-5")
	if !strings.Contains(got, "该格式暂不支持内容读取") {
		t.Fatalf("不支持格式应诚实告知：%q", got)
	}
	if !strings.Contains(got, "wx_files") || !strings.Contains(got, ".zip") {
		t.Fatalf("应带自持路径（桌面端可打开）：%q", got)
	}
}

// 提取失败（空 docx：markitdown 与内置 zip 解析器双双报错）：诚实降级带自持
// 路径，不 panic。（用空文件而非伪 docx——markitdown 在场的机器会把伪 docx
// 当纯文本「成功」转换，错误路径不确定；空文件两条链路都必错。）
func TestWxFileStore_ExtractFailHonest(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	src := filepath.Join(t.TempDir(), "broken.docx")
	if err := os.WriteFile(src, nil, 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	got := s.Ingest(src, "broken.docx", 0, "d-6")
	if !strings.Contains(got, "[用户发来文件 broken.docx") || !strings.Contains(got, "内容读取失败") {
		t.Fatalf("提取失败应诚实降级：%q", got)
	}
	if !strings.Contains(got, "wx_files") {
		t.Fatalf("降级文案应带自持路径：%q", got)
	}
}

// injectLine 直测：自持副本被外力删除（提取读不到文件）→ 同样诚实降级。
func TestWxFileStore_InjectLineMissingFile(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	got := s.injectLine(filepath.Join(s.dir, "missing.docx"), "missing.docx", 15)
	if !strings.Contains(got, "内容读取失败") || !strings.Contains(got, "missing.docx") {
		t.Fatalf("副本缺失应诚实降级：%q", got)
	}
}

// 清理策略：文件数超 50 时按修改时间从旧到新删，刚收到的文件永远保留。
func TestWxFileStore_QuotaDeletesOldest(t *testing.T) {
	s, _ := newWxFileHandlerTestStore(t, 0)
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	base := time.Now().Add(-time.Hour)
	for i := 0; i < 55; i++ {
		p := filepath.Join(s.dir, wxFileSelfName(fmt.Sprintf("old%d.bin", i), base.Add(time.Duration(i)*time.Minute)))
		if err := os.WriteFile(p, []byte{byte(i)}, 0o644); err != nil {
			t.Fatalf("写旧文件: %v", err)
		}
		if err := os.Chtimes(p, base.Add(time.Duration(i)*time.Minute), base.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("设 mtime: %v", err)
		}
	}
	src := filepath.Join(t.TempDir(), "new.txt")
	if err := os.WriteFile(src, []byte("fresh"), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	if got := s.Ingest(src, "new.txt", 5, "d-7"); !strings.Contains(got, "fresh") {
		t.Fatalf("新文件应正常注入：%q", got)
	}
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		t.Fatalf("读目录: %v", err)
	}
	if len(ents) != wxFileMaxCount {
		t.Fatalf("清理后应恰剩 %d 个文件，got %d", wxFileMaxCount, len(ents))
	}
	var haveOld0, haveOld54, haveFresh bool
	for _, e := range ents {
		data, _ := os.ReadFile(filepath.Join(s.dir, e.Name()))
		switch {
		case len(data) == 1 && data[0] == 0:
			haveOld0 = true
		case len(data) == 1 && data[0] == 54:
			haveOld54 = true
		case string(data) == "fresh":
			haveFresh = true
		}
	}
	if haveOld0 {
		t.Fatalf("最旧文件应被清理")
	}
	if !haveOld54 || !haveFresh {
		t.Fatalf("较新的旧文件与新副本都应保留（old54=%v fresh=%v）", haveOld54, haveFresh)
	}
}

// 扩展名安全化：无扩展名/花式扩展名回退 .bin，大写归一小写。
func TestWxFileSafeExt(t *testing.T) {
	cases := map[string]string{
		"a.txt":       ".txt",
		"报 表.DOCX":    ".docx",
		"noext":       ".bin",
		"weird.a1b2":  ".a1b2",
		"evil.xx/yy":  ".bin", // 末段无扩展名 → 回退 .bin
		"dot.x.y":     ".y",
		"emptyext.":   ".bin",
		"长扩展名.abcdef": ".abcdef",
	}
	for in, want := range cases {
		if got := wxFileSafeExt(in); got != want {
			t.Errorf("wxFileSafeExt(%q) = %q, want %q", in, got, want)
		}
	}
}

// 大小文案口径：KB 整数 / MB 一位小数 / B 字节。
func TestFormatWxFileSize(t *testing.T) {
	cases := map[int64]string{
		3:         "3 B",
		424 << 10: "424 KB",
		5 << 20:   "5.0 MB",
	}
	for in, want := range cases {
		if got := formatWxFileSize(in); got != want {
			t.Errorf("formatWxFileSize(%d) = %q, want %q", in, got, want)
		}
	}
}

// 构造接线：newWxFileHandler 返回的处理器落盘到 dataRoot/wx_files。
func TestNewWxFileHandler_Wiring(t *testing.T) {
	root := t.TempDir()
	h := newWxFileHandler(root)
	if h == nil {
		t.Fatalf("应返回非 nil 处理器")
	}
	src := filepath.Join(t.TempDir(), "hi.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("写源文件: %v", err)
	}
	got := h(src, "hi.txt", 5, "d-8")
	if !strings.Contains(got, "hello") {
		t.Fatalf("处理器应返回注入行：%q", got)
	}
	if ents, _ := os.ReadDir(filepath.Join(root, "wx_files")); len(ents) != 1 {
		t.Fatalf("应自持到 dataRoot/wx_files: %+v", ents)
	}
}
