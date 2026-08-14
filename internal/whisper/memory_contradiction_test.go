package whisper

import (
	"errors"
	"testing"
)

// ─── T6-5.1: memory_contradiction.go ────────────────────────────

func TestParseContradictionResult_ValidConflict(t *testing.T) {
	raw := "{\"judgment\":\"conflict\",\"action\":\"keep_new\",\"reason\":\"用户已改变\"}"

	got := parseContradictionResult(raw, "fact-old-1")
	if got == nil {
		t.Fatal("合法 JSON 应解析出结果")
	}
	if got.Judgment != "conflict" || got.Action != "keep_new" || got.Reason != "用户已改变" {
		t.Errorf("解析字段错误: %+v", got)
	}
	if got.ConflictingFactID != "fact-old-1" {
		t.Errorf("conflict 应带旧事实 ID, got %q", got.ConflictingFactID)
	}
}

func TestParseContradictionResult_ReinforceNoID(t *testing.T) {
	raw := "{\"judgment\":\"reinforce\",\"action\":\"merge\",\"reason\":\"互补\"}"

	got := parseContradictionResult(raw, "fact-old-2")
	if got == nil || got.Judgment != "reinforce" || got.Action != "merge" {
		t.Fatalf("解析 reinforce 失败: %+v", got)
	}
	if got.ConflictingFactID != "" {
		t.Errorf("reinforce 不应带冲突 ID, got %q", got.ConflictingFactID)
	}
}

func TestParseContradictionResult_InvalidJudgmentNormalized(t *testing.T) {
	raw := "{\"judgment\":\"banana\",\"action\":\"flag\",\"reason\":\"x\"}"

	got := parseContradictionResult(raw, "f3")
	if got == nil || got.Judgment != "unrelated" {
		t.Errorf("非法 judgment 应规范化为 unrelated, got %+v", got)
	}
}

func TestParseContradictionResult_InvalidActionNormalized(t *testing.T) {
	raw := "{\"judgment\":\"conflict\",\"action\":\"banana\",\"reason\":\"x\"}"

	got := parseContradictionResult(raw, "f4")
	if got == nil || got.Action != "keep_new" {
		t.Errorf("非法 action 应规范化为 keep_new, got %+v", got)
	}
}

func TestParseContradictionResult_NoJSONReturnsNil(t *testing.T) {
	if got := parseContradictionResult("这是纯文本", "f5"); got != nil {
		t.Errorf("无 JSON 应返回 nil, got %+v", got)
	}
}

func TestParseContradictionResult_EmbeddedJSON(t *testing.T) {
	inner := "{\"judgment\":\"conflict\",\"action\":\"keep_old\",\"reason\":\"旧事实更可靠\"}"

	raw := "模型思考：..." + inner + "... 结论如上"
	got := parseContradictionResult(raw, "f6")
	if got == nil || got.Judgment != "conflict" || got.Action != "keep_old" {
		t.Errorf("内嵌 JSON 应被提取解析, got %+v", got)
	}
}

func TestParseBatchContradictionResult(t *testing.T) {
	inner := "{\"pairs\":[{\"pair_idx\":1,\"judgment\":\"conflict\",\"action\":\"keep_new\",\"reason\":\"r1\"},{\"pair_idx\":3,\"judgment\":\"reinforce\",\"action\":\"merge\",\"reason\":\"r3\"}]}"

	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1"}, Existing: &MemoryFact{ID: "e1"}},
		{NewFact: &MemoryFact{ID: "n2"}, Existing: &MemoryFact{ID: "e2"}},
		{NewFact: &MemoryFact{ID: "n3"}, Existing: &MemoryFact{ID: "e3"}},
	}
	results := parseBatchContradictionResult(inner, pairs)
	if len(results) != 3 {
		t.Fatalf("结果数应为 3, got %d", len(results))
	}
	if results[0] == nil || results[0].Judgment != "conflict" || results[0].ConflictingFactID != "e1" {
		t.Errorf("pair 1 应解析为 conflict 且带 e1, got %+v", results[0])
	}
	if results[1] != nil {
		t.Errorf("pair 2 无对应结果应为 nil, got %+v", results[1])
	}
	if results[2] == nil || results[2].Judgment != "reinforce" {
		t.Errorf("pair 3 应解析为 reinforce, got %+v", results[2])
	}
}

func TestParseBatchContradictionResult_BadJSON(t *testing.T) {
	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1"}, Existing: &MemoryFact{ID: "e1"}},
	}
	results := parseBatchContradictionResult("无 JSON", pairs)
	if len(results) != 1 || results[0] != nil {
		t.Errorf("坏 JSON 应返回全 nil 结果, got %v", results)
	}
}

func TestParseBatchContradictionResult_NormalizesInvalid(t *testing.T) {
	inner := "{\"pairs\":[{\"pair_idx\":1,\"judgment\":\"banana\",\"action\":\"nope\",\"reason\":\"x\"},{\"pair_idx\":2,\"judgment\":\"conflict\",\"action\":\"merge\",\"reason\":\"y\"}]}"
	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1"}, Existing: &MemoryFact{ID: "e1"}},
		{NewFact: &MemoryFact{ID: "n2"}, Existing: &MemoryFact{ID: "e2"}},
	}
	results := parseBatchContradictionResult(inner, pairs)
	if results[0] == nil || results[0].Judgment != "unrelated" || results[0].Action != "flag" {
		t.Errorf("非法 judgment/action 应规范化: %+v", results[0])
	}
	if results[1] == nil || results[1].Judgment != "conflict" || results[1].ConflictingFactID != "e2" {
		t.Errorf("合法项应正常解析: %+v", results[1])
	}
}

func TestContradictionDetector_Check(t *testing.T) {
	cd := NewContradictionDetector()
	raw := "{\"judgment\":\"conflict\",\"action\":\"keep_new\",\"reason\":\"冲突\"}"

	got, err := cd.Check(
		&MemoryFact{ID: "n", Subcategory: "TASTES", Subject: "猫", Summary: "喜欢猫"},
		&MemoryFact{ID: "e", Subcategory: "TASTES", Subject: "猫", Summary: "讨厌猫"},
		llmStub{reply: raw},
	)
	if err != nil {
		t.Fatalf("Check 不应报错: %v", err)
	}
	if got == nil || got.Judgment != "conflict" || got.ConflictingFactID != "e" {
		t.Errorf("Check 结果错误: %+v", got)
	}
}

func TestContradictionDetector_CheckLLMError(t *testing.T) {
	cd := NewContradictionDetector()
	_, err := cd.Check(
		&MemoryFact{ID: "n", Subcategory: "TASTES", Subject: "猫", Summary: "喜欢猫"},
		&MemoryFact{ID: "e", Subcategory: "TASTES", Subject: "猫", Summary: "讨厌猫"},
		llmStub{err: errors.New("llm down")},
	)
	if err == nil {
		t.Fatal("LLM 失败应返回错误")
	}
}

func TestContradictionDetector_CheckBatchEmpty(t *testing.T) {
	cd := NewContradictionDetector()
	if got := cd.CheckBatch(nil, llmStub{reply: "{}"}); got != nil {
		t.Errorf("空批次应返回 nil, got %v", got)
	}
}

func TestContradictionDetector_CheckBatchLLMError(t *testing.T) {
	cd := NewContradictionDetector()
	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1"}, Existing: &MemoryFact{ID: "e1"}},
	}
	got := cd.CheckBatch(pairs, llmStub{err: errors.New("down")})
	if len(got) != 1 || got[0] != nil {
		t.Errorf("LLM 失败时批量应返回 len=1 的全 nil, got %v", got)
	}
}

func TestContradictionDetector_CheckBatchSuccess(t *testing.T) {
	cd := NewContradictionDetector()
	inner := "{\"pairs\":[{\"pair_idx\":1,\"judgment\":\"unrelated\",\"action\":\"keep_new\",\"reason\":\"无关\"}]}"

	pairs := []ContradictionPair{
		{NewFact: &MemoryFact{ID: "n1"}, Existing: &MemoryFact{ID: "e1"}},
	}
	got := cd.CheckBatch(pairs, llmStub{reply: inner})
	if len(got) != 1 || got[0] == nil || got[0].Judgment != "unrelated" {
		t.Errorf("批量成功路径错误: %v", got)
	}
}
