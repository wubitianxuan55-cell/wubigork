package control

// 刀4（codex 蒸馏差距档案）验收：策略快照写读 round-trip / 恢复按存档覆盖 /
// 无快照回 boot 基线（旧会话现状等价）/ 新会话不泄漏存档策略。

import (
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/permission"
)

func newPolicyTestController(t *testing.T, dir string) *Controller {
	t.Helper()
	cfgPolicy := permission.New("ask", nil, nil, nil)
	c := New(Options{
		Policy:           cfgPolicy,
		ApprovalMode:     "",
		DenyNetworkTools: true, // boot 基线：断网开
		SessionDir:       dir,
		Label:            "pol",
	})
	if c.denyNetworkTools != true || c.approvalMode != "" || c.policy.Mode != permission.Ask {
		t.Fatalf("boot 基线字段初始化不符: net=%v mode=%q perm=%v", c.denyNetworkTools, c.approvalMode, c.policy.Mode)
	}
	return c
}

func TestPolicySnapshotRoundTripAndApply(t *testing.T) {
	dir := t.TempDir()
	c := newPolicyTestController(t, dir)
	path := filepath.Join(dir, "s1.jsonl")

	// 存档形状：会话创建于「网络开 + allow 级别」的全局，恢复环境却不同。
	c.mu.Lock()
	c.approvalMode = "on-request"
	c.denyNetworkTools = false
	c.policy.Mode = permission.Allow
	c.mu.Unlock()
	c.writeSessionPolicySnapshot(path)

	meta, ok, err := session.LoadBranchMeta(path)
	if err != nil || !ok || meta.PolicySnapshot == nil {
		t.Fatalf("meta 快照缺失: ok=%v err=%v", ok, err)
	}
	if !meta.PolicySnapshot.SandboxNetwork || meta.PolicySnapshot.ApprovalMode != "on-request" || meta.PolicySnapshot.PermissionMode != "allow" {
		t.Fatalf("快照内容不符: %+v", meta.PolicySnapshot)
	}

	// 当前环境切回「断网 + ask」后恢复该会话：按存档覆盖。
	c.mu.Lock()
	c.approvalMode = ""
	c.denyNetworkTools = true
	c.policy.Mode = permission.Ask
	c.applySessionPolicySnapshotLocked(*meta.PolicySnapshot)
	if c.denyNetworkTools || c.approvalMode != "on-request" || c.policy.Mode != permission.Allow || c.permLevel != "auto" {
		t.Fatalf("恢复未按存档覆盖: net=%v mode=%q perm=%v level=%q", c.denyNetworkTools, c.approvalMode, c.policy.Mode, c.permLevel)
	}
	c.mu.Unlock()

	// 新会话语义：回 boot 基线，存档策略不泄漏。
	c.mu.Lock()
	c.resetSessionPolicyToBootLocked()
	if !c.denyNetworkTools || c.approvalMode != "" || c.policy.Mode != permission.Ask || c.permLevel != "ask" {
		t.Fatalf("回基线不符: net=%v mode=%q perm=%v level=%q", c.denyNetworkTools, c.approvalMode, c.policy.Mode, c.permLevel)
	}
	c.mu.Unlock()
}

func TestPolicySnapshotLegacySession(t *testing.T) {
	// 旧会话 meta 无快照：恢复走 boot 基线（现状等价），不报错。
	dir := t.TempDir()
	c := newPolicyTestController(t, dir)
	path := filepath.Join(dir, "legacy.jsonl")
	c.mu.Lock()
	c.approvalMode = "on-request" // 模拟上一个会话遗留的策略态
	c.denyNetworkTools = false
	c.mu.Unlock()

	c.resetSessionAuthorityPolicyAfterResume(path)

	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.denyNetworkTools || c.approvalMode != "" {
		t.Fatalf("旧会话恢复应回 boot 基线: net=%v mode=%q", c.denyNetworkTools, c.approvalMode)
	}
	// 幂等回写：meta 现在带上了 boot 基线形状的快照。
	meta, ok, err := session.LoadBranchMeta(path)
	if err != nil || !ok || meta.PolicySnapshot == nil {
		t.Fatalf("回写快照缺失: ok=%v err=%v", ok, err)
	}
	if meta.PolicySnapshot.SandboxNetwork {
		t.Fatalf("回写快照应等于 boot 基线（断网开）: %+v", meta.PolicySnapshot)
	}
}
