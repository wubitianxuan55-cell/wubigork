package session

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStateRoundTrip 验证 SaveState → LoadState 往返一致（含中文摘要）。
func TestStateRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.state.json")
	st := SessionState{Running: true, Summary: "正在输出表格", UpdatedAt: 12345}
	if err := SaveState(p, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := LoadState(p)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if !got.Running || got.Summary != st.Summary || got.UpdatedAt != st.UpdatedAt {
		t.Fatalf("LoadState = %+v, want %+v", got, st)
	}
}

// TestLoadStateMissing 验证文件不存在 → 零值 state 且无错误。
func TestLoadStateMissing(t *testing.T) {
	st, err := LoadState(filepath.Join(t.TempDir(), "nope.state.json"))
	if err != nil {
		t.Fatalf("LoadState on missing file: %v", err)
	}
	if st.Running || st.Summary != "" || st.UpdatedAt != 0 {
		t.Fatalf("LoadState on missing file = %+v, want zero", st)
	}
}

// TestLoadStateCorrupt 验证损坏的 state 文件按零值处理，不阻塞调用方。
func TestLoadStateCorrupt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.state.json")
	if err := os.WriteFile(p, []byte("{{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := LoadState(p)
	if err != nil {
		t.Fatalf("LoadState on corrupt file: %v", err)
	}
	if st.Running {
		t.Fatalf("corrupt state parsed as running=true: %+v", st)
	}
}

// TestClearState 验证清除删除文件，且对不存在的文件静默成功。
func TestClearState(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.state.json")
	if err := SaveState(p, SessionState{Running: true}); err != nil {
		t.Fatal(err)
	}
	if err := ClearState(p); err != nil {
		t.Fatalf("ClearState: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("state 文件清除后仍存在")
	}
	if err := ClearState(p); err != nil {
		t.Fatalf("ClearState on missing file: %v", err)
	}
}

// TestStatePath 验证 sidecar 路径规则与空路径防护。
func TestStatePath(t *testing.T) {
	if got := StatePath(`C:\x\sessions\a.jsonl`); got != `C:\x\sessions\a.jsonl.state.json` {
		t.Fatalf("StatePath = %q, want sibling .state.json", got)
	}
	if got := StatePath(""); got != "" {
		t.Fatalf("StatePath(\"\") = %q, want empty", got)
	}
}
