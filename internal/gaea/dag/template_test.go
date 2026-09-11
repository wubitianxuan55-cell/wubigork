package dag

// 模板库测试：FromRun 剥运行痕迹/Instantiate 全新草稿/Store 往返+校验 fail-closed
// +删除不存在显式报错+列表倒序。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tplChain() Run {
	return Run{
		ID:   "dag_src",
		Goal: "出月度报告",
		Nodes: []Node{
			{ID: "read", Title: "读报表", Prompt: "读取三份月度 xlsx", Status: StatusDone,
				Ref: "sa_read", Outputs: []string{"docs/read.xlsx"}, RunCount: 2, SteerCount: 1},
			{ID: "report", Title: "出报告", Prompt: "嵌图表出报告 docx", DependsOn: []string{"read"},
				Status: StatusAccepted, RunCount: 1, AcceptedAt: "2026-09-11T09:00:00+08:00"},
		},
	}
}

func TestTemplateFromRunStripsRuntime(t *testing.T) {
	tpl, err := FromRun(tplChain(), "月度经营报告")
	if err != nil {
		t.Fatalf("FromRun: %v", err)
	}
	if !strings.HasPrefix(tpl.ID, "dagtpl_") || tpl.Name != "月度经营报告" || tpl.SourceRunID != "dag_src" {
		t.Fatalf("模板头错误: %+v", tpl)
	}
	for _, n := range tpl.Nodes {
		if n.Status != StatusPending || n.Ref != "" || n.RunCount != 0 || n.SteerCount != 0 ||
			len(n.Outputs) != 0 || n.Error != "" || n.AcceptedAt != "" {
			t.Fatalf("运行痕迹未剥净: %+v", n)
		}
	}
	if tpl.Nodes[1].DependsOn == nil || tpl.Nodes[1].DependsOn[0] != "read" {
		t.Fatalf("依赖形状丢失: %+v", tpl.Nodes[1])
	}
}

// 空名回退 goal 截断；超长名显式报错不静默截断。
func TestTemplateNameRules(t *testing.T) {
	tpl, err := FromRun(tplChain(), "  ")
	if err != nil {
		t.Fatalf("空名应回退 goal: %v", err)
	}
	if tpl.Name != "出月度报告" {
		t.Fatalf("空名回退错误: %q", tpl.Name)
	}
	long := strings.Repeat("长", NameMaxRune+1)
	if _, err := FromRun(tplChain(), long); err == nil {
		t.Fatal("超长名应报错")
	}
}

func TestTemplateInstantiateFresh(t *testing.T) {
	tpl, _ := FromRun(tplChain(), "月度报告")
	run := Instantiate(tpl)
	if run.ID == tplChain().ID || run.ID == "" {
		t.Fatalf("run id 应全新: %q", run.ID)
	}
	if run.Goal != "出月度报告" || len(run.Nodes) != 2 {
		t.Fatalf("goal/节点数丢失: %+v", run)
	}
	if Derived(run) != DerivedDraft {
		t.Fatalf("重建应是草稿: %s", Derived(run))
	}
	for _, n := range run.Nodes {
		if n.Status != StatusPending || n.Ref != "" || n.RunCount != 0 {
			t.Fatalf("重建节点应全新: %+v", n)
		}
	}
}

func TestTemplateStoreRoundtrip(t *testing.T) {
	s := NewTemplateStore(filepath.Join(t.TempDir(), "templates"))
	if ts, err := s.List(); err != nil || len(ts) != 0 {
		t.Fatalf("空目录 List: err=%v n=%d", err, len(ts))
	} else if ts == nil {
		// 目录不存在必须返回非 nil 空集：nil slice 序列化成 JSON null，会把
		// 前端非空断言的消费方（DagPanel tpls.length）炸掉（v4.234 真机走查实锤）。
		t.Fatalf("目录不存在 List 应返回非 nil 空集")
	}
	tpl, err := FromRun(tplChain(), "月度报告")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save(tpl); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get(tpl.ID)
	if err != nil || got.Name != "月度报告" || len(got.Nodes) != 2 {
		t.Fatalf("Get 往返: err=%v %+v", err, got)
	}
	// 坏形状 fail-closed：成环模板拒入库。
	bad := tpl
	bad.ID = NewTemplateID()
	bad.Nodes[0].DependsOn = []string{"report"}
	if err := s.Save(bad); err == nil {
		t.Fatal("成环模板应拒入库")
	}
	if _, err := s.Get(bad.ID); err == nil {
		t.Fatal("被拒模板不应落盘")
	}
	// 删除闭环；再删显式报错。
	if err := s.Delete(tpl.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(tpl.ID); err == nil {
		t.Fatal("删除后 Get 应报错")
	}
	if err := s.Delete(tpl.ID); err == nil {
		t.Fatal("删除不存在模板应报错")
	}
	// 穿越拒绝（safeID 同款）。
	if err := s.Delete("../evil"); err == nil {
		t.Fatal("路径穿越应拒绝")
	}
}

// 列表按创建时间倒序；run 档 List 不会把 templates 子目录当 run（目录跳过）。
func TestTemplateListOrderAndIsolation(t *testing.T) {
	root := t.TempDir()
	rs := NewStore(root)
	ts := NewTemplateStore(filepath.Join(root, "templates"))

	old := tplChain()
	old.ID = "dag_old"
	if err := rs.Save(old); err != nil {
		t.Fatal(err)
	}
	names := []struct {
		nm, at string
	}{{"甲", "2026-09-01T00:00:00+08:00"}, {"乙", "2026-09-02T00:00:00+08:00"}}
	for _, it := range names {
		tpl, err := FromRun(tplChain(), it.nm)
		if err != nil {
			t.Fatal(err)
		}
		tpl.CreatedAt = it.at // 显式钉时刻（RFC3339 秒级，同秒落盘无法分先后）
		if err := ts.Save(tpl); err != nil {
			t.Fatal(err)
		}
	}
	tpls, err := ts.List()
	if err != nil || len(tpls) != 2 {
		t.Fatalf("List: err=%v n=%d", err, len(tpls))
	}
	if tpls[0].Name != "乙" || tpls[1].Name != "甲" {
		t.Fatalf("应按创建时间倒序: %+v", tpls)
	}
	// run 档 List 只见顶层：templates 子目录与其内容不混进 run 列表。
	runs, err := rs.List()
	if err != nil || len(runs) != 1 {
		t.Fatalf("run List 应只有 1 条: err=%v n=%d", err, len(runs))
	}
	if _, err := os.Stat(filepath.Join(root, "templates")); err != nil {
		t.Fatalf("模板目录应在: %v", err)
	}
}
