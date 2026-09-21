package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ─── 刀1: 执行序批次（Reasonix v1.38 execute_batch 蒸馏）─────────────────────

// 同批次「先读 A 再改 A」是模型声明的顺序依赖——资源键统一后必须拆批保序，
// 不再落进同一并行批产生读写竞态。
func TestPartitionReadThenEditSameFileOrdered(t *testing.T) {
	calls := []provider.ToolCall{
		{ID: "1", Name: "read_file", Arguments: `{"path":"a.txt"}`},
		{ID: "2", Name: "edit_file", Arguments: `{"path":"a.txt","old_string":"x","new_string":"y"}`},
	}
	batches := partitionToolCalls(nil, calls)
	if len(batches) != 2 {
		t.Fatalf("read→edit same file must split into ordered batches, got %+v", batches)
	}
	if batches[0].start != 0 || batches[1].start != 1 {
		t.Fatalf("order must be preserved: %+v", batches)
	}
}

// 跨路径读与写无共享资源——共存并行收益保留。
func TestPartitionReadEditDifferentFilesCoexist(t *testing.T) {
	calls := []provider.ToolCall{
		{ID: "1", Name: "read_file", Arguments: `{"path":"a.txt"}`},
		{ID: "2", Name: "edit_file", Arguments: `{"path":"b.txt","old_string":"x","new_string":"y"}`},
	}
	batches := partitionToolCalls(nil, calls)
	if len(batches) != 1 || !batches[0].parallel {
		t.Fatalf("different-path read+write should coexist in one parallel batch, got %+v", batches)
	}
}

// 写后读同路径同样保序（写完成前读不得开跑）。
func TestPartitionWriteThenReadSameFileOrdered(t *testing.T) {
	calls := []provider.ToolCall{
		{ID: "1", Name: "write_file", Arguments: `{"path":"a.txt","content":"x"}`},
		{ID: "2", Name: "read_file", Arguments: `{"path":"a.txt"}`},
	}
	batches := partitionToolCalls(nil, calls)
	if len(batches) != 2 {
		t.Fatalf("write→read same file must split into ordered batches, got %+v", batches)
	}
}

// ─── 刀2: Live file observations（Reasonix fileops/observation.go 蒸馏）─────

func TestFileObservationStaleDetection(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(p, []byte("v1 content"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &AgentRunner{}
	a.observeFile(p)

	if msg := a.checkFileStale(p); msg != "" {
		t.Fatalf("unchanged file must pass, got %q", msg)
	}

	// 外部修改 → 版本不一致 → 拦
	if err := os.WriteFile(p, []byte("externally changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	msg := a.checkFileStale(p)
	if msg == "" || !strings.Contains(msg, "stale version") {
		t.Fatalf("external modification must be flagged stale, got %q", msg)
	}

	// 自身写入刷新观察后放行（后续编辑站在自己刚写的内容上）
	a.observeFile(p)
	if msg := a.checkFileStale(p); msg != "" {
		t.Fatalf("self-refresh must clear staleness, got %q", msg)
	}
}

func TestFileObservationDeletedAndNeverObserved(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "gone.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &AgentRunner{}
	a.observeFile(p)
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	msg := a.checkFileStale(p)
	if msg == "" || !strings.Contains(msg, "no longer exists") {
		t.Fatalf("deleted observed file must be flagged, got %q", msg)
	}

	// 从未观察过的路径不拦（无过期依赖）
	other := filepath.Join(dir, "never-seen.txt")
	if msg := a.checkFileStale(other); msg != "" {
		t.Fatalf("never-observed path must pass, got %q", msg)
	}
}
