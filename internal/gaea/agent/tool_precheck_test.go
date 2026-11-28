package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/cache"
	fileenc "github.com/gaea/gaea/internal/gaea/fileutil/encoding"
)

func TestPrecheckEditFileOldStringFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "test.go"))
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	a := &AgentRunner{}
	msg := a.precheckEditFile(nil, []byte(`{"path":"`+path+`","old_string":"func main()","new_string":"func run()"}`))
	if msg != "" {
		t.Fatalf("old_string found, should pass: %s", msg)
	}
}

func TestPrecheckEditFileOldStringNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "test.go"))
	content := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"
	os.WriteFile(path, []byte(content), 0644)

	a := &AgentRunner{}
	msg := a.precheckEditFile(nil, []byte(`{"path":"`+path+`","old_string":"func missing()","new_string":"func run()"}`))
	if msg == "" {
		t.Fatal("old_string not in file, should block")
	}
	if !strings.Contains(msg, "precheck blocked") {
		t.Fatalf("msg should say precheck blocked: %s", msg)
	}
}

func TestPrecheckEditFileMissing(t *testing.T) {
	a := &AgentRunner{}
	msg := a.precheckEditFile(nil, []byte(`{"path":"/nonexistent/file.go","old_string":"x","new_string":"y"}`))
	if msg != "" {
		t.Fatalf("missing file should be silent (let real Execute handle): %s", msg)
	}
}

func TestPrecheckEditFileEmptyArgs(t *testing.T) {
	a := &AgentRunner{}
	if msg := a.precheckEditFile(nil, []byte(`{}`)); msg != "" {
		t.Fatal("empty args should be silent")
	}
	if msg := a.precheckEditFile(nil, []byte(`{"path":"","old_string":"x"}`)); msg != "" {
		t.Fatal("empty path should be silent")
	}
	if msg := a.precheckEditFile(nil, []byte(`{"path":"/tmp/x","old_string":""}`)); msg != "" {
		t.Fatal("empty old_string should be silent")
	}
}

// 2026-11-28 黄甲反馈：executeOne 在默认整读后把 read_file 的渲染输出（行号槽
// +尾标）缓存进 (path, 0, 0)；预检此前拿这个槽比对，多行 old_string 必然 not
// found（子代理改 ch04 连拦 3 次，锚点确在原文里）。预检必须比对磁盘原文。
func TestPrecheckEditFileComparesDiskNotRenderedCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "ch04.md"))
	content := "# 4 全过程环境管理措施\n\n## 4.1 环境管理措施\n\n### 4.1.1 环境管理组织\n\n项目部建立以项目负责人为第一责任人的环境保护管理组织：项目负责人负责审批环境管理方案。\n"
	os.WriteFile(path, []byte(content), 0644)

	// 与 executeOne 同款：默认整读（offset/limit 均未传）后缓存的是渲染输出。
	a := &AgentRunner{tc: cache.New(0)}
	rendered := " 1→# 4 全过程环境管理措施\n 2→\n 3→## 4.1 环境管理措施\n 4→\n 5→### 4.1.1 环境管理组织\n" +
		" 6→\n 7→项目部建立以项目负责人为第一责任人的环境保护管理组织：项目负责人负责审批环境管理方案。\n\n[revision sha256:abc]\n"
	a.tc.Set(path, 0, 0, rendered)
	if _, ok := a.tc.Get(path, 0, 0); !ok {
		t.Fatal("cache setup failed")
	}

	editArgs := func(old string) []byte {
		b, err := json.Marshal(map[string]string{"path": path, "old_string": old, "new_string": "x"})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	multi := "### 4.1.1 环境管理组织\n\n项目部建立以项目负责人为第一责任人的环境保护管理组织"
	if msg := a.precheckEditFile(nil, editArgs(multi)); msg != "" {
		t.Fatalf("多行锚点在磁盘原文里，渲染缓存不得致拦：%s", msg)
	}
	// 真不在磁盘上的锚点照拦。
	absent := "### 4.1.9 不存在的节\n\n正文"
	msg := a.precheckEditFile(nil, editArgs(absent))
	if msg == "" || !strings.Contains(msg, "precheck blocked") {
		t.Fatalf("锚点不在文件里应拦：%q", msg)
	}
	if strings.Contains(msg, "1→") {
		t.Fatalf("拦截提示里的现行文本应是磁盘原文，不该带行号槽：%s", msg)
	}
	// 预检不得回写缓存：(path, 0, 0) 仍是 read_file 的渲染输出。
	got, ok := a.tc.Get(path, 0, 0)
	if !ok || got != rendered {
		t.Fatalf("预检改写了 read_file 缓存槽：ok=%v got=%q", ok, got)
	}
}

// 解码口径与 edit_file 一致：GB18030 文件里的中文锚点此前按原始字节比，
// 必 not found 而被误拦。
func TestPrecheckEditFileDecodesNonUTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.ToSlash(filepath.Join(dir, "gbk.md"))
	os.WriteFile(path, fileenc.Encode("# 概述\n\n本章说明施工组织设计的主要内容。\n", fileenc.GB18030), 0644)

	a := &AgentRunner{}
	if msg := a.precheckEditFile(nil, []byte(`{"path":"`+path+`","old_string":"本章说明施工组织设计的主要内容。","new_string":"x"}`)); msg != "" {
		t.Fatalf("GB18030 文件里的中文锚点应能找到：%s", msg)
	}
}

func TestPrecheckToolDispatch(t *testing.T) {
	a := &AgentRunner{}
	if msg := a.precheckTool("unknown_tool", nil); msg != "" {
		t.Fatal("unknown tool should be silent")
	}
	if msg := a.precheckTool("read_file", []byte(`{"path":"/tmp/x"}`)); msg != "" {
		t.Fatal("read_file should be silent")
	}
	a.precheckTool("edit_file", []byte(`{"path":"/tmp/x","old_string":"y","new_string":"z"}`))
}
