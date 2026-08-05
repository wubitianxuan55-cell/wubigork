package boot

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/netclient"
)

// TestAddBuiltins_WorkspaceDir 验证 addBuiltins 传入工作空间目录（dir 非空）时，
// 基础工具（read_file 等）的相对路径解析绑定到该目录，而非进程 cwd。
// 对应修复：办公 agent 工具操作目录必须跟随工作空间（gaeaCwd），
// 否则界面显示 bangong 而工具在进程启动目录 C:\AI\wubigrok 下读写。
func TestAddBuiltins_WorkspaceDir(t *testing.T) {
	// 进程 cwd 固定在临时目录 A；工作空间在临时目录 B
	procCwd := t.TempDir()
	wsDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(procCwd); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	// 目标文件只存在于工作空间，不存在于进程 cwd
	if err := os.WriteFile(filepath.Join(wsDir, "hello.txt"), []byte("hi-from-workspace"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tool.NewRegistry()
	addBuiltins(reg, wsDir, nil, []string{wsDir}, sandbox.Spec{}, netclient.ProxySpec{}, io.Discard)

	readFile, ok := reg.Get("read_file")
	if !ok {
		t.Fatal("read_file 未注册")
	}
	out, err := readFile.Execute(context.Background(), json.RawMessage(`{"path":"hello.txt"}`))
	if err != nil {
		t.Fatalf("read_file 相对路径应解析到工作空间目录: %v", err)
	}
	if !strings.Contains(out, "hi-from-workspace") {
		t.Errorf("read_file 输出 = %q, want 包含 hi-from-workspace（工具应绑定工作空间目录）", out)
	}

	// 进程 cwd 下不应有 hello.txt（证明未退回进程 cwd）
	if _, err := os.Stat(filepath.Join(procCwd, "hello.txt")); err == nil {
		t.Error("hello.txt 不应出现在进程 cwd——工具必须绑定工作空间目录")
	}
}

// TestAddBuiltins_NoDir 验证 dir 为空（CLI/默认）时保持原行为：
// 相对路径基于进程 cwd。
func TestAddBuiltins_NoDir(t *testing.T) {
	procCwd := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(procCwd); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	if err := os.WriteFile(filepath.Join(procCwd, "local.txt"), []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tool.NewRegistry()
	addBuiltins(reg, "", nil, nil, sandbox.Spec{}, netclient.ProxySpec{}, io.Discard)

	readFile, ok := reg.Get("read_file")
	if !ok {
		t.Fatal("read_file 未注册")
	}
	if _, err := readFile.Execute(context.Background(), json.RawMessage(`{"path":"local.txt"}`)); err != nil {
		t.Errorf("dir 为空时 read_file 应基于进程 cwd 解析: %v", err)
	}
}
