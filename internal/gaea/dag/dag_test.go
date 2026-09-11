package dag

import (
	"os"
	"path/filepath"
	"testing"
)

func chainNodes() []Node {
	return []Node{
		{ID: "read", Title: "读报表", Prompt: "读取三份月度报表"},
		{ID: "pivot", Title: "透视汇总", Prompt: "透视汇总", DependsOn: []string{"read"}},
		{ID: "report", Title: "出报告", Prompt: "嵌图表出 Word", DependsOn: []string{"pivot"}},
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		goal    string
		nodes   []Node
		wantErr bool
	}{
		{"正常链", "出月报", chainNodes(), false},
		{"空 goal", "  ", chainNodes(), true},
		{"空节点", "出月报", nil, true},
		{"id 重复", "出月报", []Node{{ID: "a"}, {ID: "a"}}, true},
		{"缺 title", "出月报", []Node{{ID: "a", Prompt: "p"}}, true},
		{"缺 prompt", "出月报", []Node{{ID: "a", Title: "t"}}, true},
		{"悬空依赖", "出月报", []Node{{ID: "a", Title: "t", Prompt: "p", DependsOn: []string{"ghost"}}}, true},
		{"成环", "出月报", []Node{
			{ID: "a", Title: "t", Prompt: "p", DependsOn: []string{"b"}},
			{ID: "b", Title: "t", Prompt: "p", DependsOn: []string{"a"}},
		}, true},
	}
	for _, tc := range cases {
		err := Validate(tc.goal, tc.nodes)
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: Validate err=%v wantErr=%v", tc.name, err, tc.wantErr)
		}
	}
}

func TestOrderWaves(t *testing.T) {
	// 线性链 → 三波
	waves, err := Order(chainNodes(), nil)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	if len(waves) != 3 || waves[0][0].ID != "read" || waves[1][0].ID != "pivot" || waves[2][0].ID != "report" {
		t.Fatalf("线性链分波错误: %+v", waves)
	}
	// 菱形 → b/c 同波（波内按 id 排序=确定性）
	diamond := []Node{
		{ID: "a", Title: "t", Prompt: "p"},
		{ID: "c", Title: "t", Prompt: "p", DependsOn: []string{"a"}},
		{ID: "b", Title: "t", Prompt: "p", DependsOn: []string{"a"}},
		{ID: "d", Title: "t", Prompt: "p", DependsOn: []string{"b", "c"}},
	}
	waves, err = Order(diamond, nil)
	if err != nil {
		t.Fatalf("Order diamond: %v", err)
	}
	if len(waves) != 3 || len(waves[1]) != 2 || waves[1][0].ID != "b" || waves[1][1].ID != "c" {
		t.Fatalf("菱形分波错误: %+v", waves)
	}
	// 子集选择：只重跑 d → 单波单节点
	waves, err = Order(diamond, map[string]bool{"d": true})
	if err != nil {
		t.Fatalf("Order subset: %v", err)
	}
	if len(waves) != 1 || len(waves[0]) != 1 || waves[0][0].ID != "d" {
		t.Fatalf("子集分波错误: %+v", waves)
	}
}

func TestDerived(t *testing.T) {
	chain := chainNodes()
	if d := Derived(Run{Nodes: chain}); d != DerivedDraft {
		t.Errorf("全新链应为 draft，得 %s", d)
	}
	running := chainNodes()
	running[1].Status = StatusRunning
	running[1].RunCount = 1
	if d := Derived(Run{Nodes: running}); d != DerivedRunning {
		t.Errorf("有 running 应为 running，得 %s", d)
	}
	failed := chainNodes()
	failed[0].Status = StatusFailed
	failed[0].RunCount = 1
	failed[2].Status = StatusSkipped
	if d := Derived(Run{Nodes: failed}); d != DerivedFailed {
		t.Errorf("有 failed 应为 failed，得 %s", d)
	}
	ready := chainNodes()
	for i := range ready {
		ready[i].Status = StatusDone
		ready[i].RunCount = 1
	}
	if d := Derived(Run{Nodes: ready}); d != DerivedReady {
		t.Errorf("全 done 应为 ready，得 %s", d)
	}
	ready[2].Status = StatusAccepted
	if d := Derived(Run{Nodes: ready}); d != DerivedReady {
		t.Errorf("部分 accepted 仍应 ready，得 %s", d)
	}
	ready[0].Status = StatusAccepted
	ready[1].Status = StatusAccepted
	if d := Derived(Run{Nodes: ready}); d != DerivedAccepted {
		t.Errorf("全 accepted 应为 accepted，得 %s", d)
	}
}

func TestMarkInterrupted(t *testing.T) {
	r := Run{Nodes: chainNodes()}
	r.Nodes[1].Status = StatusRunning
	if !MarkInterrupted(&r, "应用重启中断") {
		t.Fatal("有 running 节点应报告变更")
	}
	if r.Nodes[1].Status != StatusFailed || r.Nodes[1].Error == "" {
		t.Fatalf("中断清扫结果错误: %+v", r.Nodes[1])
	}
	if MarkInterrupted(&r, "x") {
		t.Fatal("无 running 节点不应报告变更")
	}
}

func TestStoreRoundTripAndSafety(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	r := Run{ID: "dag_20260911_120000", Goal: "出月报", CreatedAt: "2026-09-11T12:00:00+08:00", Nodes: chainNodes()}
	if err := s.Save(r); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get(r.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Goal != "出月报" || len(got.Nodes) != 3 || got.UpdatedAt == "" {
		t.Fatalf("往返不一致: %+v", got)
	}
	list, err := s.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v len=%d", err, len(list))
	}
	// SaveNew：同 id 冲突追加后缀，绝不覆盖
	r2 := Run{ID: r.ID, Goal: "另一条", Nodes: chainNodes()}
	if err := s.SaveNew(&r2); err != nil {
		t.Fatalf("SaveNew: %v", err)
	}
	if r2.ID == r.ID {
		t.Fatal("SaveNew 应改写冲突 id")
	}
	list, _ = s.List()
	if len(list) != 2 {
		t.Fatalf("SaveNew 后应两条，得 %d", len(list))
	}
	// 路径穿越拒绝
	for _, bad := range []string{"../evil", "a/b", "a\\b", "", ".json"} {
		if _, err := s.Get(bad); err == nil {
			t.Errorf("非法 id %q 应被拒绝", bad)
		}
	}
	if _, err := s.Get("不存在"); err == nil {
		t.Error("不存在的 run 应报错（fail-closed）")
	}
	// List 只认 .json，跳过 .tmp
	if err := os.WriteFile(filepath.Join(dir, "dag_tmp.json.tmp"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, _ = s.List()
	if len(list) != 2 {
		t.Fatalf("tmp 文件不应入列，得 %d", len(list))
	}
}

// ApplyEdit 增量改图（v4.222）：形状未变保留状态/变更回 pending/移除/撤销验收
// 诚实计数；运行中与坏形状 fail-closed。
func TestApplyEditReconcile(t *testing.T) {
	r := Run{
		ID:   "dag_e",
		Goal: "出月度报告",
		Nodes: []Node{
			{ID: "read", Title: "读报表", Prompt: "读取三份月度 xlsx", Status: StatusAccepted,
				Ref: "sa_read", Outputs: []string{"docs/read.xlsx"}, RunCount: 1},
			{ID: "report", Title: "出报告", Prompt: "嵌图表出报告 docx", DependsOn: []string{"read"},
				Status: StatusDone, RunCount: 1},
		},
	}

	// 改 report 指令 + 新增 pivot + 移除无 → read 原样保留（验收不动）。
	newNodes := []Node{
		{ID: "read", Title: "读报表", Prompt: "读取三份月度 xlsx", DependsOn: []string{}},
		{ID: "report", Title: "出报告", Prompt: "改用季度口径出报告", DependsOn: []string{"read"}},
	}
	out, rep, err := ApplyEdit(r, "出月度报告", newNodes)
	if err != nil {
		t.Fatalf("ApplyEdit: %v", err)
	}
	if rep.Kept != 1 || rep.Updated != 1 || rep.Removed != 0 || rep.RevokedAcc != 0 {
		t.Fatalf("调和计数错误: %+v", rep)
	}
	if n := out.Nodes[0]; n.Status != StatusAccepted || len(n.Outputs) != 1 {
		t.Fatalf("未变节点应保留状态与产物: %+v", n)
	}
	if n := out.Nodes[1]; n.Status != StatusPending || n.RunCount != 0 || n.Ref != "" {
		t.Fatalf("变更节点应回 pending 全新: %+v", n)
	}

	// 改已验收节点 = 撤销验收，计数透出。
	acc := Run{ID: "dag_e2", Goal: "g", Nodes: []Node{
		{ID: "a", Title: "t", Prompt: "旧指令", Status: StatusAccepted, RunCount: 1},
	}}
	out, rep, err = ApplyEdit(acc, "g", []Node{{ID: "a", Title: "t", Prompt: "新指令"}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.RevokedAcc != 1 || out.Nodes[0].Status != StatusPending {
		t.Fatalf("撤销验收语义错误: rep=%+v node=%+v", rep, out.Nodes[0])
	}

	// 守卫：运行中拒绝；成环拒绝；移除仍被依赖的节点=悬空依赖拒绝。
	running := Run{ID: "dag_e3", Goal: "g", Nodes: []Node{{ID: "a", Title: "t", Prompt: "p", Status: StatusRunning}}}
	if _, _, err := ApplyEdit(running, "g", []Node{{ID: "a", Title: "t", Prompt: "p2"}}); err == nil {
		t.Fatal("运行中应拒绝改图")
	}
	if _, _, err := ApplyEdit(r, "g", []Node{
		{ID: "a", Title: "t", Prompt: "p", DependsOn: []string{"b"}},
		{ID: "b", Title: "t", Prompt: "p", DependsOn: []string{"a"}},
	}); err == nil {
		t.Fatal("新图成环应拒绝")
	}
	if _, _, err := ApplyEdit(r, "g", []Node{
		{ID: "report", Title: "出报告", Prompt: "p", DependsOn: []string{"read"}},
	}); err == nil {
		t.Fatal("移除仍被依赖的 read 应悬空拒绝")
	}
}
