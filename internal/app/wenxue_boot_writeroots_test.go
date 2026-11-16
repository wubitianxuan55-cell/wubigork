package app

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"

	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// 2026-11-16 图文 write_file 越界 bug 根修的行为锁：config.toml 的
// sandbox.workspace_root（办公板块 persistWorkspaceLocked 持久化的全局共享
// 值，本机实锤=D:\新工作\2630合并沧州）不得劫持图文引擎的写入边界——
// WriteRootsAt 让它优先于 Cwd，图文会话写小说目录被拒。boot.Options.WriteRoots
// 显式接管后，write_file 只锚小说工作空间。
func TestWenxueBootWriteRootsOverride(t *testing.T) {
	novel := t.TempDir()  // 图文小说工作空间（引擎 Cwd）
	office := t.TempDir() // 办公板块持久化的 workspace_root（劫持源）

	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) {
		cfg := gaeaConfig.Default()
		cfg.DefaultModel = "gaea"
		cfg.Providers = []gaeaConfig.ProviderEntry{{Name: "gaea", Kind: "wubigrok", Model: "", ContextWindow: 1_000_000}}
		cfg.Tools.Enabled = nil
		cfg.Sandbox.Bash = "off"
		// 复刻本机 config.toml 的劫持形态：workspace_root=办公持久化目录。
		cfg.Sandbox.WorkspaceRoot = office
		return cfg, nil
	})
	t.Cleanup(func() { gaeaConfig.SetLoader(nil) })
	// wubigrok provider 走 bridge：测试环境注入内存桩客户端（gaea_projects_test 同款）。
	bridge.SetClient(ai.NewClient(config.Load()))

	ctrl, err := gaeaBoot.Build(context.Background(), gaeaBoot.Options{
		// Model 用 Default() 里注入的 gaea 档（Kind=wubigrok，与办公测试同款
		// 装配）；写边界行为与模型档无关。
		Model:      "gaea",
		RequireKey: false,
		Sink:       event.FuncSink(func(event.Event) {}),
		MaxSteps:   0,
		SessionDir: t.TempDir(),
		Stderr:     io.Discard,
		Cwd:        novel,
		WriteRoots: []string{novel},
	})
	if err != nil {
		t.Fatalf("图文引擎构建失败: %v", err)
	}
	defer ctrl.Close()

	var writer tool.Tool
	for _, tl := range ctrl.Tools() {
		if tl.Name() == "write_file" {
			writer = tl
			break
		}
	}
	if writer == nil {
		t.Fatal("write_file 未装配")
	}

	args := func(path string) json.RawMessage {
		raw, err := json.Marshal(map[string]string{"path": path, "content": "正文"})
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}

	// 写小说工作空间（图文本职）：必须成功。
	if _, err := writer.Execute(context.Background(), args(filepath.Join(novel, "章节.md"))); err != nil {
		t.Fatalf("写小说工作空间不得被拒: %v", err)
	}
	// 写办公 workspace_root（劫持形态）：必须被拒，且边界文案指向小说目录。
	if _, err := writer.Execute(context.Background(), args(filepath.Join(office, "越界.md"))); err == nil {
		t.Fatal("写入办公 workspace_root 必须被拒")
	} else if !strings.Contains(err.Error(), novel) {
		t.Fatalf("越界文案必须指向接管后的写入根（小说目录）: %v", err)
	}
}
