package builtin

// D4 写锁 sidecar 测试（dsh-ops bind-before-publish 蒸馏，2026-11-10）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool"
)

func lockWorkspace(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func TestFileLockMutualExclusionAndContention(t *testing.T) {
	ws := lockWorkspace(t)
	target := filepath.Join(ws, "a.txt")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 手动持有锁 → WithFileLock 退避后报 LOCK_CONTENTION（可重试+零效应）。
	sidecar := lockSidecarPath(target)
	if err := os.WriteFile(sidecar, []byte("holder-token-xxxxxxxxxxxx"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 锁龄新鲜+token 合法 → 正常争用路径。
	err := WithFileLock(target, func() error { return nil })
	if err == nil {
		t.Fatal("锁被占应争用失败")
	}
	f, ok := tool.AsFailure(err)
	if !ok || f.Subcode != "LOCK_CONTENTION" || !f.Retryable || f.EffectState != tool.EffectNone {
		t.Fatalf("争用语义错: %+v", f)
	}
	// 释放后可正常拿锁。
	if err := os.Remove(sidecar); err != nil {
		t.Fatal(err)
	}
	ran := false
	if err := WithFileLock(target, func() error { ran = true; return nil }); err != nil {
		t.Fatalf("空闲锁应可得: %v", err)
	}
	if !ran {
		t.Fatal("fn 应被执行")
	}
	// WithFileLock 结束后锁哨兵已清。
	if _, serr := os.Stat(sidecar); !os.IsNotExist(serr) {
		t.Fatal("锁哨兵应在释放后清除")
	}
}

func TestFileLockFnErrorStillReleases(t *testing.T) {
	ws := lockWorkspace(t)
	target := filepath.Join(ws, "b.txt")
	sentinel := &tool.Failure{Code: tool.CodeExecError, Err: errStr("boom")}
	err := WithFileLock(target, func() error { return sentinel })
	if err != sentinel {
		t.Fatalf("fn 错误应原样透传: %v", err)
	}
	if _, serr := os.Stat(lockSidecarPath(target)); !os.IsNotExist(serr) {
		t.Fatal("fn 失败锁也必须释放（defer）")
	}
}

func TestFileLockBindVerify(t *testing.T) {
	ws := lockWorkspace(t)
	target := filepath.Join(ws, "c.txt")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	bind, err := bindFileLockTarget(target)
	if err != nil || !bind.existed {
		t.Fatalf("bind 应记录存在目标: %v", err)
	}
	// 未变 → verify 过。
	if err := bind.verify(target); err != nil {
		t.Fatalf("未变应通过: %v", err)
	}
	// 替换（删+重建=inode 漂移）→ conflict。
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("v2-external"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = bind.verify(target)
	if f, ok := tool.AsFailure(err); !ok || f.Code != tool.CodeConflict {
		t.Fatalf("inode 漂移应 conflict: %v", err)
	}
	// absent 快照：目标被外部创建 → conflict；仍 absent → 过。
	absent := filepath.Join(ws, "new.txt")
	ab, err := bindFileLockTarget(absent)
	if err != nil || ab.existed {
		t.Fatal("absent bind 应为空快照")
	}
	if err := ab.verify(absent); err != nil {
		t.Fatalf("仍 absent 应通过: %v", err)
	}
	if err := os.WriteFile(absent, []byte("appeared"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ab.verify(absent); err == nil {
		t.Fatal("absent→出现应 conflict")
	}
	// 存在快照被删 → conflict（带删除指引）。
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := bind.verify(target); err == nil {
		t.Fatal("存在→删除应 conflict")
	}
}

func TestFileLockStaleReclaim(t *testing.T) {
	ws := lockWorkspace(t)
	target := filepath.Join(ws, "d.txt")
	if err := os.WriteFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	sidecar := lockSidecarPath(target)
	// 死持有者：高龄+token 畸形 → 回收后可得锁。
	if err := os.WriteFile(sidecar, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(sidecar, old, old); err != nil {
		t.Fatal(err)
	}
	if err := WithFileLock(target, func() error { return nil }); err != nil {
		t.Fatalf("高龄畸形锁应被回收: %v", err)
	}
	// 高龄但 token 合法（活持有者年龄恰好大）→ 不回收，争用失败。
	if err := os.WriteFile(sidecar, []byte(strings.Repeat("a", 24)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(sidecar, old, old); err != nil {
		t.Fatal(err)
	}
	if err := WithFileLock(target, func() error { return nil }); err == nil {
		t.Fatal("合法 token 的高龄锁不得按年龄回收（持有者可能还活着）")
	}
}

func TestFileLockReleaseTokenGuard(t *testing.T) {
	ws := lockWorkspace(t)
	target := filepath.Join(ws, "e.txt")
	// 手工放一把别人的锁，release 不得删它。
	sidecar := lockSidecarPath(target)
	if err := os.WriteFile(sidecar, []byte("someone-else-token"), 0o644); err != nil {
		t.Fatal(err)
	}
	releaseFileLock(target)
	if _, serr := os.Stat(sidecar); serr != nil {
		t.Fatal("别人的锁不得被 release 误删")
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }
