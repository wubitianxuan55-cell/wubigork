package whisper

import (
	"errors"
	"testing"
)

// ─── T6-5.1: memory_self_editor.go ──────────────────────────────

func TestBatchResolve_NoValidPairs(t *testing.T) {
	e := NewMemorySelfEditor()
	// 同 ID 配对被过滤
	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "x"}, Existing: &MemoryFact{ID: "x"}},
	}
	fs := NewFactStore()
	got := e.BatchResolve(pairs, fs, llmStub{reply: "{}"})
	if len(got) != 0 {
		t.Errorf("无效配对应无结果, got %v", got)
	}
	// consolidated 层新事实也被过滤
	pairs2 := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1", FactLayer: "consolidated"}, Existing: &MemoryFact{ID: "e1"}},
	}
	if got := e.BatchResolve(pairs2, fs, llmStub{reply: "{}"}); len(got) != 0 {
		t.Errorf("consolidated 新事实应被过滤, got %v", got)
	}
}

func TestBatchResolve_SinglePairKeepNew(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Domain: "user", Subcategory: "TASTES", Subject: "猫", Summary: "讨厌猫", Weight: 1})
	fs.Add(MemoryFact{ID: "e2", Domain: "user", Subcategory: "TASTES", Subject: "狗", Summary: "喜欢狗", Weight: 1})
	raw := "{\"judgment\":\"conflict\",\"action\":\"keep_new\",\"reason\":\"新事实更可信\"}"

	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1", FactLayer: "raw", Domain: "user", Subcategory: "TASTES", Subject: "猫", Summary: "喜欢猫", Weight: 2},
			Existing: &existing.MemoryFact},
	}
	results := e.BatchResolve(pairs, fs, llmStub{reply: raw})
	if len(results) != 1 || results[0] == "" {
		t.Fatalf("应产生 1 条结果, got %v", results)
	}
	if fs.Get("e1") != nil && fs.Get("e1").IsActive() {
		t.Error("keep_new 应退役旧事实 e1")
	}
	if len(e.GetEditLog()) != 1 || e.GetEditLog()[0].Action != "retire_old_conflict" {
		t.Errorf("编辑日志应记录 retire_old_conflict, got %+v", e.GetEditLog())
	}
}

func TestBatchResolve_BatchPath(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing1 := fs.Add(MemoryFact{ID: "e1", Domain: "user", Subcategory: "TASTES", Subject: "猫", Summary: "讨厌猫", Weight: 1})
	existing2 := fs.Add(MemoryFact{ID: "e2", Domain: "user", Subcategory: "FOOD", Subject: "辣", Summary: "不吃辣", Weight: 1})
	inner := "{\"pairs\":[{\"pair_idx\":1,\"judgment\":\"conflict\",\"action\":\"keep_new\",\"reason\":\"r1\"},{\"pair_idx\":2,\"judgment\":\"reinforce\",\"action\":\"merge\",\"reason\":\"r2\"}]}"

	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1", FactLayer: "raw", Domain: "user", Subcategory: "TASTES", Subject: "猫", Summary: "喜欢猫", Weight: 2}, Existing: &existing1.MemoryFact},
		{NewFact: &MemoryFact{ID: "n2", FactLayer: "raw", Domain: "user", Subcategory: "FOOD", Subject: "辣", Summary: "超爱吃辣", Weight: 3}, Existing: &existing2.MemoryFact},
	}
	results := e.BatchResolve(pairs, fs, llmStub{reply: inner})
	if len(results) != 2 {
		t.Fatalf("批量应产生 2 条结果, got %v", results)
	}
	if fs.Get("e1").IsActive() {
		t.Error("pair1 conflict/keep_new 应退役 e1")
	}
	if fs.Get("e2").Weight < 3.3 {
		t.Errorf("pair2 reinforce 应提升 e2 weight, got %f", fs.Get("e2").Weight)
	}
	if fs.Get("n2") != nil && fs.Get("n2").IsActive() {
		t.Error("pair2 reinforce 应退役 n2")
	}
}

func TestApplyResolution_Reinforce(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Summary: "短摘要", Weight: 1.0})
	fs.Add(MemoryFact{ID: "n1", Summary: "更长的详细摘要内容", Weight: 2.0})
	check := ContradictionCheck{Judgment: "reinforce", Action: "merge", Reason: "互补"}
	result := e.applyResolution(check, &fs.Get("n1").MemoryFact, &existing.MemoryFact, fs)
	if result == "" || result != "强化并合并：互补" {
		t.Errorf("reinforce 结果错误: %q", result)
	}
	got := fs.Get("e1")
	if got.Weight != 2.0+SelfEditReinforceWeightBoost {
		t.Errorf("weight 应提升到 %v, got %v", 2.0+SelfEditReinforceWeightBoost, got.Weight)
	}
	if got.Summary != "更长的详细摘要内容" {
		t.Errorf("summary 应取更长者, got %q", got.Summary)
	}
	if fs.Get("n1").IsActive() {
		t.Error("reinforce 应退役新事实")
	}
}

func TestApplyResolution_ConflictKeepNew(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Summary: "旧", Weight: 1})
	fs.Add(MemoryFact{ID: "n1", Summary: "新", Weight: 1})
	check := ContradictionCheck{Judgment: "conflict", Action: "keep_new", Reason: "用户变了"}
	result := e.applyResolution(check, &fs.Get("n1").MemoryFact, &existing.MemoryFact, fs)
	if result != "退役旧事实（冲突，保留新）：用户变了" {
		t.Errorf("keep_new 结果错误: %q", result)
	}
	if fs.Get("e1").IsActive() {
		t.Error("e1 应被退役")
	}
	if !fs.Get("n1").IsActive() {
		t.Error("n1 应保留")
	}
}

func TestApplyResolution_ConflictKeepOld(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Summary: "旧", Weight: 1})
	fs.Add(MemoryFact{ID: "n1", Summary: "新", Weight: 1})
	check := ContradictionCheck{Judgment: "conflict", Action: "keep_old", Reason: "旧更可靠"}
	result := e.applyResolution(check, &fs.Get("n1").MemoryFact, &existing.MemoryFact, fs)
	if result != "退役新事实（冲突，保留旧）：旧更可靠" {
		t.Errorf("keep_old 结果错误: %q", result)
	}
	if !fs.Get("e1").IsActive() || fs.Get("n1").IsActive() {
		t.Error("应退役新事实、保留旧事实")
	}
}

func TestApplyResolution_ConflictMerge(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Summary: "旧摘要", Weight: 1.0})
	fs.Add(MemoryFact{ID: "n1", Summary: "新摘要内容更完整", Weight: 2.0})
	check := ContradictionCheck{Judgment: "conflict", Action: "merge", Reason: "合并"}
	result := e.applyResolution(check, &fs.Get("n1").MemoryFact, &existing.MemoryFact, fs)
	if result != "合并冲突事实：合并" {
		t.Errorf("merge 结果错误: %q", result)
	}
	got := fs.Get("e1")
	if got.Summary != "新摘要内容更完整" || got.Weight != 2.0 {
		t.Errorf("合并后 e1 应取新摘要与新 weight: %+v", got)
	}
	if fs.Get("n1").IsActive() {
		t.Error("合并后新事实应退役")
	}
}

func TestApplyResolution_ConflictFlag(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Summary: "旧", Weight: 1})
	fs.Add(MemoryFact{ID: "n1", Summary: "新", Weight: 1})
	check := ContradictionCheck{Judgment: "conflict", Action: "flag", Reason: "不确定"}
	result := e.applyResolution(check, &fs.Get("n1").MemoryFact, &existing.MemoryFact, fs)
	if result != "标记为需人工确认：不确定" {
		t.Errorf("flag 结果错误: %q", result)
	}
	if !fs.Get("e1").IsActive() || !fs.Get("n1").IsActive() {
		t.Error("flag 不应退役任何事实")
	}
}

func TestApplyResolution_UnrelatedNoop(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Summary: "旧", Weight: 1})
	fs.Add(MemoryFact{ID: "n1", Summary: "新", Weight: 1})
	check := ContradictionCheck{Judgment: "unrelated", Action: "keep_new", Reason: "无关"}
	if result := e.applyResolution(check, &fs.Get("n1").MemoryFact, &existing.MemoryFact, fs); result != "" {
		t.Errorf("unrelated 应无操作, got %q", result)
	}
	if len(e.GetEditLog()) != 0 {
		t.Errorf("unrelated 不应记日志, got %+v", e.GetEditLog())
	}
}

func TestBatchResolve_LLMErrorNoPanic(t *testing.T) {
	e := NewMemorySelfEditor()
	fs := NewFactStore()
	existing := fs.Add(MemoryFact{ID: "e1", Domain: "user", Subcategory: "TASTES", Subject: "猫", Summary: "讨厌猫", Weight: 1})
	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1", FactLayer: "raw", Domain: "user", Subcategory: "TASTES", Subject: "猫", Summary: "喜欢猫", Weight: 2}, Existing: &existing.MemoryFact},
	}
	results := e.BatchResolve(pairs, fs, llmStub{err: errors.New("down")})
	if len(results) != 0 {
		t.Errorf("LLM 失败应无结果, got %v", results)
	}
}

func TestEditLog_GetAndClear(t *testing.T) {
	e := NewMemorySelfEditor()
	e.log("flag", "n1", "e1", "测试")
	log := e.GetEditLog()
	if len(log) != 1 || log[0].Action != "flag" || log[0].TargetFactID != "n1" {
		t.Fatalf("日志内容错误: %+v", log)
	}
	// GetEditLog 返回副本
	log[0].Action = "mutated"
	if e.GetEditLog()[0].Action != "flag" {
		t.Error("GetEditLog 应返回副本")
	}
	e.ClearLog()
	if len(e.GetEditLog()) != 0 {
		t.Error("ClearLog 后应为空")
	}
}

func TestEditLog_TrimsOverMax(t *testing.T) {
	e := NewMemorySelfEditor()
	for i := 0; i < SelfEditLogMax+30; i++ {
		e.log("flag", "t", "r", "x")
	}
	got := e.GetEditLog()
	if len(got) > SelfEditLogMax {
		t.Errorf("日志长度不应超过上限 %d, got %d", SelfEditLogMax, len(got))
	}
	if len(got) == 0 || got[len(got)-1].Reason != "x" {
		t.Errorf("裁剪后应保留最新记录: %+v", got)
	}
}
