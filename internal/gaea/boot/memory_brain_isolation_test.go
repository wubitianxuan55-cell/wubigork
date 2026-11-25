package boot

// memory_brain_isolation_test.go — 记忆左右脑隔离测试（2026-11-16，图文/办公
// 分脑刀的验收）：Options.MemoryUserDir 缝非空时，remember 落盘进 scope 目录的
// 独立 DB（docs/画像/auto-memory 全随目录），另一脑不可见；双脑互不串。
// hermetic：脑目录全部显式传 t.TempDir() 子目录——测试从不触真实
// config.MemoryUserDir()（默认脑）。mock provider 同 boot_test 纪律。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/testutil"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// brainTestKindSeq 本文件独立单调序号（testkind_test.go 在 boot_test 外部包，
// 不可见；provider 注册表全局唯一性由前缀+序号自行保证）。
var brainTestKindSeq atomic.Int64

func brainTestKind(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, brainTestKindSeq.Add(1))
}

func brainTestBuild(t *testing.T, brainDir, projDir string) map[string]tool.Tool {
	t.Helper()

	kind := brainTestKind("test-mock-brain")
	provider.Register(kind, func(cfg provider.Config) (provider.Provider, error) {
		return testutil.NewMock("mock"), nil
	})
	cfg := config.Default()
	cfg.DefaultModel = "mock"
	cfg.Providers = []config.ProviderEntry{{
		Name: "mock", Kind: kind, Model: "grok-3", ContextWindow: 1_000_000,
	}}
	cfg.Tools.Enabled = nil
	cfg.Sandbox.Bash = "off"
	cfg.Memory.Enabled = true
	config.SetLoader(func() (*config.Config, error) { return cfg, nil })
	t.Cleanup(func() { config.SetLoader(nil) })

	ctrl, err := Build(context.Background(), Options{
		Model:         "mock",
		RequireKey:    false,
		Sink:          event.FuncSink(func(event.Event) {}),
		Stderr:        io.Discard,
		SessionDir:    filepath.Join(projDir, ".gaea", "sessions"),
		Cwd:           projDir,
		MemoryUserDir: brainDir,
	})
	if err != nil {
		t.Fatalf("Build(brain=%s): %v", brainDir, err)
	}
	t.Cleanup(ctrl.Close)
	byName := make(map[string]tool.Tool)
	for _, tl := range ctrl.Tools() {
		byName[tl.Name()] = tl
	}
	return byName
}

func brainExec(t *testing.T, tl tool.Tool, args string) (string, error) {
	t.Helper()
	return tl.Execute(context.Background(), json.RawMessage(args))
}

func TestBuildMemoryBrainIsolation(t *testing.T) {
	home := t.TempDir() // 隔离 os.UserConfigDir 影响面（本测试两脑都显式传目录）
	brainA := filepath.Join(home, "brain-a")
	brainB := filepath.Join(home, "brain-b")
	projA := filepath.Join(home, "novel-a")
	projB := filepath.Join(home, "office-b")
	for _, d := range []string{projA, projB} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	facesA := brainTestBuild(t, brainA, projA)
	facesB := brainTestBuild(t, brainB, projB)
	// TempDir 清理前关池（GetDatabase 按目录单例持句柄，Windows 锁文件）。
	t.Cleanup(func() {
		_ = db.CloseDatabase(brainA)
		_ = db.CloseDatabase(brainB)
	})

	rememberA, ok := facesA["remember"]
	if !ok {
		t.Fatal("脑 A 装配缺 remember")
	}
	if _, err := brainExec(t, rememberA, `{"name":"brain-isolation-probe","description":"左右脑隔离探针","body":"只应出现在脑 A。","type":"project"}`); err != nil {
		t.Fatalf("脑 A remember 落盘: %v", err)
	}

	// 脑 A 的 DB 落在 scope 目录（DatabasePath=<userDir>/Hephaestus.db）。
	if _, err := os.Stat(filepath.Join(brainA, "Hephaestus.db")); err != nil {
		t.Fatalf("脑 A 目录应落独立 Hephaestus.db: %v", err)
	}
	if _, err := os.Stat(filepath.Join(brainB, "Hephaestus.db")); err != nil {
		t.Fatalf("脑 B 目录应落独立 Hephaestus.db: %v", err)
	}

	// 脑 A 读得到探针（独立 DB 回读）。
	gdbA := db.GetDatabase(brainA)
	memA := memory.Load(memory.Options{CWD: projA, UserDir: brainA, DB: gdbA})
	if _, ok := memA.Store.Get("brain-isolation-probe"); !ok {
		t.Fatal("脑 A 应回读探针事实")
	}

	// 脑 B 读不到：工具面 + 独立 DB 回读 + 索引，三向断言（memory_get 已于
	// 2026-11-25 并入 memory_search 的按名读取，按名读走各脑自己的 Store 视图）。
	rememberB, ok := facesB["remember"]
	if !ok {
		t.Fatal("脑 B 装配缺 remember")
	}
	if _, ok := facesB["memory_search"]; !ok {
		t.Fatal("脑 B 装配缺 memory_search")
	}
	gdbB := db.GetDatabase(brainB)
	memB := memory.Load(memory.Options{CWD: projB, UserDir: brainB, DB: gdbB})
	if m, ok := memB.Store.Get("brain-isolation-probe"); ok {
		t.Fatalf("脑 B 不应回读脑 A 的事实（独立 DB）：%+v", m)
	}
	if strings.Contains(memB.Index, "brain-isolation-probe") {
		t.Fatal("脑 B 索引不应含脑 A 探针")
	}
	// 脑 B 自己写入走自己的 DB（脑 B 可写可读，证明缝是分脑不是断写）。
	if _, err := brainExec(t, rememberB, `{"name":"brain-b-own","description":"脑 B 自有事实","body":"脑 B 独立落盘。","type":"project"}`); err != nil {
		t.Fatalf("脑 B remember 落盘: %v", err)
	}
	if m, ok := memory.Load(memory.Options{CWD: projB, UserDir: brainB, DB: gdbB}).Store.Get("brain-b-own"); !ok || !strings.Contains(m.Body, "脑 B 独立落盘") {
		t.Fatalf("脑 B 应回读自有事实: %+v ok=%v", m, ok)
	}
	// 脑 A 仍看不到脑 B 的自有事实（互不串的双向确认）。
	if _, ok := memA.Store.Get("brain-b-own"); ok {
		t.Fatal("脑 A 不应看到脑 B 的自有事实（独立 DB）")
	}
}
