package boot

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// 2026-11-16 图文 write_file 越界根修的组合语义锁（boot_roots）：
//   - 默认形态复刻劫持源：cfg.Sandbox.WorkspaceRoot（办公板块持久化的全局
//     共享值）经 WriteRootsAt/ReadRootsAt 渗入根集合；
//   - Options.WriteRoots/ReadRoots 非空即整体接管，配置值不再渗入；
//   - 接管后的根合成 PathAccessPolicy：写与读都只认小说工作空间。
func TestResolveRootsTakeover(t *testing.T) {
	novel := t.TempDir()
	office := t.TempDir()
	cfg := config.Default()
	cfg.Sandbox.WorkspaceRoot = office

	// 默认形态：办公 workspace_root 渗入写与读（劫持源如实呈现——图文引擎
	// 不接管就是这副样子）。
	writeRoots, readRoots := resolveRoots(cfg, novel, Options{})
	if !containsRoot(writeRoots, office) {
		t.Fatalf("默认形态写入根必须含配置 workspace_root（劫持源）: %v", writeRoots)
	}
	if !containsRoot(readRoots, office) {
		t.Fatalf("默认形态读根必须含配置 workspace_root（读泄漏源）: %v", readRoots)
	}

	// 接管形态：配置值不再渗入，写读边界都只锚小说工作空间。
	writeRoots, readRoots = resolveRoots(cfg, novel, Options{
		WriteRoots: []string{novel},
		ReadRoots:  []string{novel},
	})
	if containsRoot(writeRoots, office) || containsRoot(readRoots, office) {
		t.Fatalf("接管后配置 workspace_root 不得渗入: write=%v read=%v", writeRoots, readRoots)
	}
	if !containsRoot(writeRoots, novel) || !containsRoot(readRoots, novel) {
		t.Fatalf("接管后边界必须锚小说工作空间: write=%v read=%v", writeRoots, readRoots)
	}
}

// 接管后的根合成真实策略类型：写与读都只认小说工作空间（读边界用
// NewPathAccessPolicy 的真实组合——workspace 恒可读、readRoots 只是额外
// 边界，图文传 [novel] 即无额外边界）。
func TestTakeoverPolicyComposition(t *testing.T) {
	novel := t.TempDir()
	office := t.TempDir()
	cfg := config.Default()
	cfg.Sandbox.WorkspaceRoot = office
	writeRoots, readRoots := resolveRoots(cfg, novel, Options{
		WriteRoots: []string{novel},
		ReadRoots:  []string{novel},
	})
	policy := tool.NewPathAccessPolicy(novel, readRoots, writeRoots, nil)

	if err := policy.CheckWrite(context.Background(), filepath.Join(novel, "章节.md")); err != nil {
		t.Fatalf("写小说目录不得被拒: %v", err)
	}
	if err := policy.CheckWrite(context.Background(), filepath.Join(office, "越界.md")); err == nil {
		t.Fatal("写办公 workspace_root 必须被拒")
	}
	if err := policy.CheckRead(context.Background(), filepath.Join(office, "造价台账.md")); err == nil {
		t.Fatal("读办公 workspace_root 必须被拒（读边界接管）")
	}
	if err := policy.CheckRead(context.Background(), filepath.Join(novel, "章节.md")); err != nil {
		t.Fatalf("读小说目录不得被拒: %v", err)
	}
}

func containsRoot(roots []string, target string) bool {
	for _, r := range roots {
		if strings.EqualFold(filepath.Clean(r), filepath.Clean(target)) {
			return true
		}
	}
	return false
}
