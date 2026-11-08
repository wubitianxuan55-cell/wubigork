package builtin

// 快照闭锁测试（2026-11-08 众神殿办公板块子代理裁决刀5，风险条：「正向白名
// 单工具可能产生 WritePaths 未申报的临时文件或目录；实现席对每个白名单工具
// 做临时工作区文件系统快照闭包测试」）。每个 containment 白名单写工具在节点
// 作用域策略（WriteScopePolicy）下验证两腿：
//   - 声明内写：成功落盘，且工作区快照的增/改全部落在声明根内（闭包成立）；
//   - 声明外写：checkToolAccess 拒绝（"blocked: path access denied: …declared
//     write paths…"），快照零变化。
// bash 无 WritePathResolver（写面不可包含），由 agent 包 restrictContainable
// Registry 从并行节点注册表排除，不在本表。

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		files[rel] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	return files
}

func mustArgsJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// driveWriter 用单轮 runner 驱动一次工具调用（策略在 checkToolAccess 生效），
// 返回工具结果文本。
func driveWriter(t *testing.T, ws string, tt tool.Tool, args string) string {
	t.Helper()
	reg := tool.NewRegistry()
	reg.Add(tt)
	prov := &closureProvider{args: args}
	var results []event.Event
	sink := event.FuncSink(func(e event.Event) { results = append(results, e) })
	r := agent.New(prov, reg, agent.NewSession(""), agent.Options{
		MaxSteps:     1,
		AccessPolicy: scopedPolicy(t, ws),
	}, sink)
	_, _ = r.Run(context.Background(), "go") // 单轮步数上限错误无关紧要，看工具结果
	var out strings.Builder
	for _, e := range results {
		if e.Kind == event.ToolResult {
			out.WriteString(e.Tool.Output + e.Tool.Err)
		}
	}
	return out.String()
}

func scopedPolicy(t *testing.T, ws string) tool.AccessPolicy {
	t.Helper()
	declared := filepath.Join(ws, "out")
	if err := os.MkdirAll(declared, 0o755); err != nil {
		t.Fatal(err)
	}
	return tool.NewWriteScopePolicy(tool.NewPathAccessPolicy(ws, nil, nil, nil), []string{declared})
}

// closureProvider 第一回合发一次工具调用，收尾即止。
type closureProvider struct {
	args   string
	gotReq provider.Request
}

func (p *closureProvider) Name() string { return "closure" }

func (p *closureProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.gotReq = req
	ch := make(chan provider.Chunk, 3)
	ch <- provider.Chunk{Type: provider.ChunkToolCallStart, ToolCall: &provider.ToolCall{ID: "c1", Name: "closure-tool"}}
	ch <- provider.Chunk{Type: provider.ChunkToolCall, ToolCall: &provider.ToolCall{ID: "c1", Name: p.gotReq.Tools[0].Name, Arguments: p.args}}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func (p *closureProvider) Chat(ctx context.Context, req provider.Request) (*provider.Completion, error) {
	return provider.ChatFromStream(ctx, p, req)
}

func TestWriterToolsClosureWithinDeclaredScope(t *testing.T) {
	cases := []struct {
		name string
		mk   func(roots []string) tool.Tool
		// seed 在声明内准备输入（编辑类需要既有文件）
		seed func(ws, declared string)
		// seedOut 在声明外准备被拒调用的目标文件（快照前落位，不算工具增量）
		seedOut func(ws, declared string)
		// inScope/outScope 返回工具参数（路径取自 ws/declared 实值）
		inScope  func(ws, declared string) string
		outScope func(ws, declared string) string
		// verifyIn 在声明内成功后校验落盘效果
		verifyIn func(t *testing.T, ws, declared string)
	}{
		{
			name: "write_file",
			mk:   func(roots []string) tool.Tool { return writeFile{roots: roots} },
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"path": filepath.Join(declared, "a.txt"), "content": "hi"})
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"path": filepath.Join(ws, "b.txt"), "content": "hi"})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, err := os.ReadFile(filepath.Join(declared, "a.txt"))
				if err != nil || string(b) != "hi" {
					t.Errorf("声明内 write_file 应落盘: %v %q", err, b)
				}
			},
		},
		{
			name: "edit_file",
			mk:   func(roots []string) tool.Tool { return editFile{roots: roots} },
			seed: func(ws, declared string) {
				os.WriteFile(filepath.Join(declared, "a.txt"), []byte("hello"), 0o644)
			},
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"path": filepath.Join(declared, "a.txt"), "old_string": "hello", "new_string": "world"})
			},
			seedOut: func(ws, declared string) {
				os.WriteFile(filepath.Join(ws, "seed.txt"), []byte("hello"), 0o644)
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"path": filepath.Join(ws, "seed.txt"), "old_string": "hello", "new_string": "world"})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, _ := os.ReadFile(filepath.Join(declared, "a.txt"))
				if string(b) != "world" {
					t.Errorf("声明内 edit_file 应生效: %q", b)
				}
			},
		},
		{
			name: "multi_edit",
			mk:   func(roots []string) tool.Tool { return multiEdit{roots: roots} },
			seed: func(ws, declared string) {
				os.WriteFile(filepath.Join(declared, "a.txt"), []byte("one two"), 0o644)
			},
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{"path": filepath.Join(declared, "a.txt"), "edits": []map[string]string{{"old_string": "one", "new_string": "1"}, {"old_string": "two", "new_string": "2"}}})
			},
			seedOut: func(ws, declared string) {
				os.WriteFile(filepath.Join(ws, "seed.txt"), []byte("one"), 0o644)
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{"path": filepath.Join(ws, "seed.txt"), "edits": []map[string]string{{"old_string": "one", "new_string": "1"}}})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, _ := os.ReadFile(filepath.Join(declared, "a.txt"))
				if string(b) != "1 2" {
					t.Errorf("声明内 multi_edit 应生效: %q", b)
				}
			},
		},
		{
			name: "edit_lines",
			mk:   func(roots []string) tool.Tool { return editLines{roots: roots} },
			seed: func(ws, declared string) {
				os.WriteFile(filepath.Join(declared, "a.txt"), []byte("l1\nl2\nl3"), 0o644)
			},
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{"path": filepath.Join(declared, "a.txt"), "start_line": 1, "end_line": 1, "new_content": "L1"})
			},
			seedOut: func(ws, declared string) {
				os.WriteFile(filepath.Join(ws, "seed.txt"), []byte("l1\nl2"), 0o644)
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{"path": filepath.Join(ws, "seed.txt"), "start_line": 1, "end_line": 1, "new_content": "L1"})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, _ := os.ReadFile(filepath.Join(declared, "a.txt"))
				if !strings.HasPrefix(string(b), "L1") {
					t.Errorf("声明内 edit_lines 应生效: %q", b)
				}
			},
		},
		{
			name: "move_file",
			mk:   func(roots []string) tool.Tool { return moveFile{roots: roots} },
			seed: func(ws, declared string) {
				os.WriteFile(filepath.Join(declared, "src.txt"), []byte("mv"), 0o644)
			},
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"source": filepath.Join(declared, "src.txt"), "destination": filepath.Join(declared, "dst.txt")})
			},
			outScope: func(ws, declared string) string {
				os.WriteFile(filepath.Join(declared, "src.txt"), []byte("mv"), 0o644)
				return mustArgsJSON(t, map[string]string{"source": filepath.Join(declared, "src.txt"), "destination": filepath.Join(ws, "escaped.txt")})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				if _, err := os.Stat(filepath.Join(declared, "dst.txt")); err != nil {
					t.Errorf("声明内 move_file 应落盘: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// ── 声明内腿：成功 + 快照增量全在声明根内 ──
			ws := t.TempDir()
			declared := filepath.Join(ws, "out")
			if err := os.MkdirAll(declared, 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.seed != nil {
				tc.seed(ws, declared)
			}
			before := snapshotTree(t, ws)
			roots := []string{ws}
			out := driveWriter(t, ws, tc.mk(roots), tc.inScope(ws, declared))
			if strings.Contains(out, "path access denied") || strings.Contains(out, "blocked") {
				t.Fatalf("声明内写不应被拒: %q", out)
			}
			tc.verifyIn(t, ws, declared)
			after := snapshotTree(t, ws)
			for rel := range after {
				if before[rel] == after[rel] {
					continue
				}
				abs := filepath.Join(ws, rel)
				if !strings.HasPrefix(strings.ToLower(abs), strings.ToLower(declared+string(os.PathSeparator))) && abs != declared {
					t.Errorf("快照增量越出声明根: %s", rel)
				}
			}
			for rel := range before {
				if _, ok := after[rel]; !ok && !strings.HasPrefix(strings.ToLower(filepath.Join(ws, rel)), strings.ToLower(declared+string(os.PathSeparator))) {
					t.Errorf("声明外文件被删改: %s", rel)
				}
			}

			// ── 声明外腿：拒绝 + 快照零变化 ──
			ws2 := t.TempDir()
			declared2 := filepath.Join(ws2, "out")
			if err := os.MkdirAll(declared2, 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.seed != nil {
				tc.seed(ws2, declared2)
			}
			if tc.seedOut != nil {
				tc.seedOut(ws2, declared2)
			}
			before2 := snapshotTree(t, ws2)
			out2 := driveWriter(t, ws2, tc.mk([]string{ws2}), tc.outScope(ws2, declared2))
			if !strings.Contains(out2, "declared write paths") {
				t.Fatalf("声明外写应收到节点作用域拒绝: %q", out2)
			}
			after2 := snapshotTree(t, ws2)
			if len(before2) != len(after2) {
				t.Errorf("声明外拒绝后快照应零变化: before=%d after=%d", len(before2), len(after2))
			}
			for rel, v := range before2 {
				if after2[rel] != v {
					t.Errorf("声明外拒绝后文件被改动: %s", rel)
				}
			}
		})
	}
}
