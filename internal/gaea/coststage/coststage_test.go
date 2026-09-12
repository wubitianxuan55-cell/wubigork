package coststage

import (
	"encoding/json"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
)

// newStore 用临时目录打开 Hephaestus.db 并构建 store(测试后自动关闭)。
func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	return Open(gdb)
}

func TestStoreUnavailable(t *testing.T) {
	s := Open(nil)
	if s.Available() {
		t.Fatal("nil db 应不可用")
	}
	if err := s.SaveStage(StageValue{}); err == nil {
		t.Fatal("不可用 store 保存应报错")
	}
	if got := s.ListStages("p"); got != nil {
		t.Fatalf("不可用 store 列表应为 nil,got %+v", got)
	}
	if err := s.DeleteStage("p", StageEstimate); err == nil {
		t.Fatal("不可用 store 删除应报错")
	}
}

func TestSaveStageUpsert(t *testing.T) {
	s := newStore(t)

	// 新建
	if err := s.SaveStage(StageValue{ProjectID: "p1", Stage: StageEstimate, Amount: 1000, Date: "2026-01-01", Note: "初版"}); err != nil {
		t.Fatal(err)
	}
	got := s.ListStages("p1")
	if len(got) != 1 || got[0].Stage != StageEstimate || got[0].Amount != 1000 || got[0].Date != "2026-01-01" {
		t.Fatalf("首次保存 = %+v", got)
	}
	if got[0].CreatedAt.IsZero() || got[0].UpdatedAt.IsZero() {
		t.Fatalf("时间戳缺失 = %+v", got[0])
	}

	// 同项目同阶段重复保存:UPSERT 更新金额/备注,不新增行
	if err := s.SaveStage(StageValue{ProjectID: "p1", Stage: StageEstimate, Amount: 1100, Note: "修订"}); err != nil {
		t.Fatal(err)
	}
	got = s.ListStages("p1")
	if len(got) != 1 {
		t.Fatalf("UPSERT 后行数 = %d, want 1", len(got))
	}
	if got[0].Amount != 1100 || got[0].Note != "修订" {
		t.Fatalf("UPSERT 未更新字段 = %+v", got[0])
	}

	// 其他项目互不影响
	if got := s.ListStages("other"); len(got) != 0 {
		t.Fatalf("其他项目不应有数据 = %+v", got)
	}
}

func TestSaveStageValidation(t *testing.T) {
	s := newStore(t)
	cases := []StageValue{
		{ProjectID: "p", Stage: "设计概算", Amount: 1},       // 非五算阶段
		{ProjectID: "p", Stage: "", Amount: 1},           // 空阶段
		{ProjectID: "p", Stage: "估算 ", Amount: 1},        // 带空白
		{ProjectID: "", Stage: StageEstimate, Amount: 1}, // 空项目 id
	}
	for i, c := range cases {
		if err := s.SaveStage(c); err == nil {
			t.Fatalf("case %d 应报错:%+v", i, c)
		}
	}
}

func TestListStagesOrderAndSkip(t *testing.T) {
	s := newStore(t)

	// 乱序录入:预算/估算/决算(概算、结算未录入)
	_ = s.SaveStage(StageValue{ProjectID: "p2", Stage: StageBudget, Amount: 115})
	_ = s.SaveStage(StageValue{ProjectID: "p2", Stage: StageEstimate, Amount: 100})
	_ = s.SaveStage(StageValue{ProjectID: "p2", Stage: StageFinal, Amount: 160})
	got := s.ListStages("p2")
	if len(got) != 3 {
		t.Fatalf("行数 = %d, want 3", len(got))
	}
	want := []string{StageEstimate, StageBudget, StageFinal}
	for i, st := range want {
		if got[i].Stage != st {
			t.Fatalf("第 %d 行 stage = %q, want %q(全量=%+v)", i, got[i].Stage, st, got)
		}
	}

	// 全量录入时严格按 StageOrder
	for _, st := range []string{StageFinal, StageEstimate, StageDesign, StageSettlement, StageBudget} {
		_ = s.SaveStage(StageValue{ProjectID: "p3", Stage: st, Amount: 100})
	}
	full := s.ListStages("p3")
	if len(full) != 5 {
		t.Fatalf("全量行数 = %d, want 5", len(full))
	}
	for i, st := range StageOrder {
		if full[i].Stage != st {
			t.Fatalf("顺序 = %+v, want %v", full, StageOrder)
		}
	}

	// 未录入项目返回空
	if got := s.ListStages("nobody"); len(got) != 0 {
		t.Fatalf("未录入项目应为空 = %+v", got)
	}
}

func TestDeleteStage(t *testing.T) {
	s := newStore(t)
	_ = s.SaveStage(StageValue{ProjectID: "p4", Stage: StageEstimate, Amount: 100})
	_ = s.SaveStage(StageValue{ProjectID: "p4", Stage: StageBudget, Amount: 115})
	if err := s.DeleteStage("p4", StageBudget); err != nil {
		t.Fatal(err)
	}
	got := s.ListStages("p4")
	if len(got) != 1 || got[0].Stage != StageEstimate {
		t.Fatalf("删除后 = %+v", got)
	}
	// 删除不存在的阶段:无操作不报错
	if err := s.DeleteStage("p4", StageFinal); err != nil {
		t.Fatalf("删除不存在阶段应无操作: %v", err)
	}
	// 非法阶段名报错
	if err := s.DeleteStage("p4", "乱写"); err == nil {
		t.Fatal("非法 stage 删除应报错")
	}
}

func TestOpenIdempotent(t *testing.T) {
	// 重复 Open 幂等(CREATE TABLE IF NOT EXISTS),二次打开的 store 可正常读写
	s1 := newStore(t)
	s2 := Open(s1.db)
	if !s2.Available() {
		t.Fatal("二次 Open 应可用")
	}
	if err := s2.SaveStage(StageValue{ProjectID: "p5", Stage: StageEstimate, Amount: 1}); err != nil {
		t.Fatal(err)
	}
	if got := s1.ListStages("p5"); len(got) != 1 {
		t.Fatalf("两 store 应共享同一表 = %+v", got)
	}
}

// TestWireShapeCamelCase 线上 JSON 形状回归锁（v4.269，同 costproject）。
func TestWireShapeCamelCase(t *testing.T) {
	b, err := json.Marshal(StageValue{ID: 1, ProjectID: "p", Stage: "投资估算", Amount: 100})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"id", "projectId", "stage", "amount"} {
		if _, ok := m[k]; !ok {
			t.Errorf("StageValue 缺 camelCase 键 %q: %v", k, m)
		}
	}
	cb, _ := json.Marshal(CompareRow{Stage: "概算", HasPrev: true})
	var cm map[string]any
	_ = json.Unmarshal(cb, &cm)
	for _, k := range []string{"stage", "hasValue", "prevStage", "chainDiff", "baseDiffPct"} {
		if _, ok := cm[k]; !ok {
			t.Errorf("CompareRow 缺 camelCase 键 %q", k)
		}
	}
	db2, _ := json.Marshal(Deviation{FromStage: "估算", Level: "正常"})
	var dm map[string]any
	_ = json.Unmarshal(db2, &dm)
	for _, k := range []string{"fromStage", "toStage", "direction", "level", "suggestion"} {
		if _, ok := dm[k]; !ok {
			t.Errorf("Deviation 缺 camelCase 键 %q", k)
		}
	}
}
