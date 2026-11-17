package agent

import (
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// TestPersistedParentRef 归一规则：持久 sa_ ref 才作父；ephemeral id 无 meta
// 侧车可指、mt_ 是单轮本地模型调用，一律落空按主会话派生降级。
func TestPersistedParentRef(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"sa_20261117_100000_0000000001_ab", "sa_20261117_100000_0000000001_ab"},
		{"sa_ephemeral_123_1", ""},
		{"mt_20261117_100000_0000000001_ab", ""},
		{"call_1", ""},
	}
	for _, tc := range cases {
		if got := PersistedParentRef(tc.in); got != tc.want {
			t.Fatalf("PersistedParentRef(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSubagentMetaParentRefRoundTrip 血缘盖章落盘与一次写定语义：
// MarkRunning 落 parentRef/spawnCallId；重复写（不同 run 值）不覆盖既有值；
// continue_from 装载的 run 不盖章（血缘属于原 spawn）。
func TestSubagentMetaParentRefRoundTrip(t *testing.T) {
	store := NewSubagentStore(t.TempDir())

	run, err := store.PrepareFresh("sys", spaces.SpaceWork)
	if err != nil {
		t.Fatalf("PrepareFresh: %v", err)
	}
	run.ParentRef = "sa_parent"
	run.SpawnCallID = "call_parent"
	if err := store.MarkRunning(run); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	// 落一份可续跑的 transcript（空会话终态只写 meta，PrepareContinue 会因
	// transcript 缺失报错）。
	run.Session.Add(provider.Message{Role: provider.RoleUser, Content: "调研任务"})
	meta, err := store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("loadMeta: %v", err)
	}
	if meta.ParentRef != "sa_parent" || meta.SpawnCallID != "call_parent" {
		t.Fatalf("stamped meta wrong: parentRef=%q spawnCallId=%q", meta.ParentRef, meta.SpawnCallID)
	}

	// 一次写定：进度快照/终态写带上不同的 run 值也不覆盖历史盖章。
	rerun := *run
	rerun.ParentRef = "sa_other"
	rerun.SpawnCallID = "call_other"
	if err := store.SaveCompleted(&rerun); err != nil {
		t.Fatalf("SaveCompleted: %v", err)
	}
	meta, err = store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("reload meta: %v", err)
	}
	if meta.ParentRef != "sa_parent" || meta.SpawnCallID != "call_parent" {
		t.Fatalf("one-time-write violated after terminal save: %+v", meta)
	}
	if meta.Status != SubagentCompleted {
		t.Fatalf("status = %q, want completed", meta.Status)
	}

	// 续跑路径：PrepareContinue 的 run 不带血缘（新出发不改变原 spawn 血缘），
	// meta 既有盖章原样保留。
	continued, err := store.PrepareContinue(run.Ref, spaces.SpaceWork)
	if err != nil {
		t.Fatalf("PrepareContinue: %v", err)
	}
	defer continued.Release()
	if continued.ParentRef != "" || continued.SpawnCallID != "" {
		t.Fatalf("continue run must not carry fresh lineage, got %q/%q", continued.ParentRef, continued.SpawnCallID)
	}
	if err := store.MarkRunning(continued); err != nil {
		t.Fatalf("continue MarkRunning: %v", err)
	}
	meta, err = store.loadMeta(run.Ref)
	if err != nil {
		t.Fatalf("reload meta after continue: %v", err)
	}
	if meta.ParentRef != "sa_parent" || meta.SpawnCallID != "call_parent" {
		t.Fatalf("continue must preserve original lineage: %+v", meta)
	}
}
