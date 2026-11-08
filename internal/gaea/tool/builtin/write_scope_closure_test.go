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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

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
		// guard 可选环境门（如 python+matplotlib 依赖，缺即 skip）
		guard func(t *testing.T)
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
		{
			// format_convert：xlsx→md 纯 Go 路径（excelize 造最小工作簿）；
			// 读面走 ReadTarget（源在工作区内=父策略放行），写面=声明 output。
			name: "format_convert",
			mk:   func(roots []string) tool.Tool { return formatConvert{} },
			seed: func(ws, declared string) {
				writeFixtureXlsx(t, filepath.Join(ws, "data.xlsx"))
			},
			seedOut: func(ws, declared string) {},
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"path": filepath.Join(ws, "data.xlsx"), "output": filepath.Join(declared, "out.md")})
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]string{"path": filepath.Join(ws, "data.xlsx"), "output": filepath.Join(ws, "escaped.md")})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, err := os.ReadFile(filepath.Join(declared, "out.md"))
				if err != nil || len(b) == 0 {
					t.Errorf("声明内 format_convert 应落盘: %v", err)
				}
			},
		},
		{
			// chart_gen：python+matplotlib 渲染（缺宿主依赖即 skip，同既有冒烟配方）。
			name:  "chart_gen",
			mk:    func(roots []string) tool.Tool { return chartGen{} },
			guard: requireClosurePython,
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{"labels": []string{"一", "二"}, "values": []float64{3, 5}, "chart_type": "bar", "output": filepath.Join(declared, "chart.png")})
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{"labels": []string{"一", "二"}, "values": []float64{3, 5}, "chart_type": "bar", "output": filepath.Join(ws, "escaped.png")})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, err := os.ReadFile(filepath.Join(declared, "chart.png"))
				if err != nil || len(b) < 100 {
					t.Errorf("声明内 chart_gen 应落盘 PNG: %v len=%d", err, len(b))
				}
			},
		},
		{
			// diagram_gen：同 chart_gen 的宿主依赖门。
			name:  "diagram_gen",
			mk:    func(roots []string) tool.Tool { return diagramGen{} },
			guard: requireClosurePython,
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{
					"nodes":  []map[string]any{{"id": "a", "label": "读数"}, {"id": "b", "label": "出图"}},
					"edges":  []map[string]any{{"from": "a", "to": "b"}},
					"output": filepath.Join(declared, "diagram.png"),
				})
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{
					"nodes":  []map[string]any{{"id": "a", "label": "读数"}, {"id": "b", "label": "出图"}},
					"edges":  []map[string]any{{"from": "a", "to": "b"}},
					"output": filepath.Join(ws, "escaped.png"),
				})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				b, err := os.ReadFile(filepath.Join(declared, "diagram.png"))
				if err != nil || len(b) < 100 {
					t.Errorf("声明内 diagram_gen 应落盘 PNG: %v len=%d", err, len(b))
				}
			},
		},
		{
			// schedule_apply：显式 path 即写目标（resolver 语义）；纯 Go。
			name: "schedule_apply",
			mk:   func(roots []string) tool.Tool { return scheduleApply{} },
			inScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{
					"project": map[string]any{"name": "t", "startDate": "2026-11-09", "tasks": []map[string]any{{"id": "1", "name": "a", "duration": 1, "level": 1}}},
					"path":    filepath.Join(declared, "plan.json"),
					"summary": "闭锁测试",
				})
			},
			outScope: func(ws, declared string) string {
				return mustArgsJSON(t, map[string]any{
					"project": map[string]any{"name": "t", "startDate": "2026-11-09", "tasks": []map[string]any{{"id": "1", "name": "a", "duration": 1, "level": 1}}},
					"path":    filepath.Join(ws, "plan.json"),
					"summary": "闭锁测试",
				})
			},
			verifyIn: func(t *testing.T, ws, declared string) {
				if _, err := os.Stat(filepath.Join(declared, "plan.json")); err != nil {
					t.Errorf("声明内 schedule_apply 应落盘: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.guard != nil {
				tc.guard(t)
			}
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
			if t.Failed() {
				t.Logf("工具结果: %s", out)
			}
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

// requireClosurePython 与 diagram_gen_test 的 requireDiagramPython 同配方：
// 宿主缺 python+matplotlib 即 skip（渲染型工具的声明外拒绝由策略层保证，
// 与渲染依赖无关——跳过只省机器时间，不放过任何越权路径）。
func requireClosurePython(t *testing.T) {
	t.Helper()
	for _, py := range []string{"python", "python3"} {
		cmd := exec.Command(py, "-c", "import matplotlib")
		if err := cmd.Run(); err == nil {
			return
		}
	}
	t.Skip("宿主缺 python+matplotlib（渲染依赖），跳过")
}

// writeFixtureXlsx 用 excelize 造最小工作簿（闭锁测试的转换源）。
func writeFixtureXlsx(t *testing.T, path string) {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	cell := func(col string, row int, v any) {
		if err := f.SetCellValue("Sheet1", col+strconv.Itoa(row), v); err != nil {
			t.Fatal(err)
		}
	}
	cell("A1", 1, "部门")
	cell("B1", 1, "金额")
	cell("A2", 2, "销售")
	cell("B2", 2, 120)
	cell("A3", 3, "研发")
	cell("B3", 3, 80)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
}
