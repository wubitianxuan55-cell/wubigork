package util

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

// ── ExtractJSON ────────────────────────────────────────────────────────

func TestExtractJSON_NormalCodeBlock(t *testing.T) {
	input := "以下是结果：\n```json\n{\"name\": \"test\", \"value\": 42}\n```\n"
	got := ExtractJSON(input)
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(got), &obj); err != nil {
		t.Fatalf("结果不是合法 JSON: %s\n原始: %q", err, got)
	}
	if obj["name"] != "test" {
		t.Errorf("name = %v, 期望 test", obj["name"])
	}
}

func TestExtractJSON_MultipleObjects(t *testing.T) {
	input := `前文{ "a": 1 }中间{ "b": 2 }后文`
	got := ExtractJSON(input)
	// 实现取第一个 { 到最后一个 }
	if got != `{ "a": 1 }中间{ "b": 2 }` {
		t.Errorf("期望从首个 { 到末个 }, 得到 %q", got)
	}
}

func TestExtractJSON_NestedBraces(t *testing.T) {
	input := `{"outer": {"inner": "value"}, "arr": [1,2,3]}`
	got := ExtractJSON(input)
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(got), &obj); err != nil {
		t.Fatalf("结果不是合法 JSON: %s\n原始: %q", err, got)
	}
	if obj["outer"] == nil {
		t.Error("outer 字段缺失")
	}
}

func TestExtractJSON_NoBraces(t *testing.T) {
	input := "这是一段纯文本，没有花括号"
	got := ExtractJSON(input)
	if got != input {
		t.Errorf("无花括号时应返回原串, 得到 %q", got)
	}
}

func TestExtractJSON_EmptyString(t *testing.T) {
	input := ""
	got := ExtractJSON(input)
	if got != "" {
		t.Errorf("空字符串应返回空, 得到 %q", got)
	}
}

func TestExtractJSON_OnlyOpeningBrace(t *testing.T) {
	input := "这是不完整的 { 只有左花括号"
	got := ExtractJSON(input)
	if got != input {
		t.Errorf("无匹配花括号时应返回原串, 得到 %q", got)
	}
}

// ── Truncate ───────────────────────────────────────────────────────────

func TestTruncate_ShortString(t *testing.T) {
	input := "hello"
	got := Truncate(input, 10)
	if got != input {
		t.Errorf("短于 maxLen 时应原样返回, 得到 %q", got)
	}
}

func TestTruncate_LongString(t *testing.T) {
	input := "hello world truncate test"
	got := Truncate(input, 5)
	expected := "hello..."
	if got != expected {
		t.Errorf("期望 %q, 得到 %q", expected, got)
	}
}

func TestTruncate_ChineseRunes(t *testing.T) {
	input := "你好世界测试截断"
	got := Truncate(input, 4)
	expected := "你好世界..."
	if got != expected {
		t.Errorf("期望 %q, 得到 %q", expected, got)
	}
}

func TestTruncate_MaxLenZero(t *testing.T) {
	input := "anything"
	got := Truncate(input, 0)
	expected := "..."
	if got != expected {
		t.Errorf("maxLen=0 时应返回 ..., 得到 %q", got)
	}
}

// ── SafeGo ─────────────────────────────────────────────────────────────

func TestSafeGo_NormalFunction(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		close(done)
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SafeGo goroutine 未在 1s 内完成")
	}
}

func TestSafeGo_PanicRecover(t *testing.T) {
	done := make(chan struct{})
	var ranPanic atomic.Bool
	SafeGo(func() {
		defer close(done)
		ranPanic.Store(true)
		panic("测试 panic")
	})
	select {
	case <-done:
		if !ranPanic.Load() {
			t.Fatal("panic 函数未执行")
		}
	case <-time.After(time.Second):
		t.Fatal("SafeGo goroutine 未在 1s 内完成")
	}
}

// ── MustMarshal / MustMarshalCompact ───────────────────────────────────

type testMarshalObj struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestMustMarshal_NormalStruct(t *testing.T) {
	obj := testMarshalObj{Name: "test", Value: 42}
	data := MustMarshal(obj)
	var decoded testMarshalObj
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("MustMarshal 结果不可反序列化: %s", err)
	}
	if decoded.Name != "test" || decoded.Value != 42 {
		t.Errorf("往返结果不一致: %+v", decoded)
	}
}

func TestMustMarshalCompact_NormalStruct(t *testing.T) {
	data := MustMarshalCompact(testMarshalObj{Name: "test"})
	if len(data) == 0 {
		t.Fatal("MustMarshalCompact 返回空数据")
	}
}

func TestMustMarshal_PanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("期望 panic，但没有")
		}
	}()
	_ = MustMarshal(make(chan int))
}

func TestMustMarshalCompact_PanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("期望 panic，但没有")
		}
	}()
	_ = MustMarshalCompact(make(chan int))
}

// ── ParseMarkedSections ────────────────────────────────────────────────

func TestParseMarkedSections_Normal(t *testing.T) {
	input := `前文---WORLD---
{"name": "test"}
---END_WORLD---后文`
	sections, err := ParseMarkedSections(input, "---WORLD---", "---END_WORLD---")
	if err != nil {
		t.Fatalf("ParseMarkedSections 返回错误: %s", err)
	}
	if len(sections) != 1 {
		t.Fatalf("期望 1 个区段, 得到 %d", len(sections))
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(sections[0].RawJSON), &obj); err != nil {
		t.Fatalf("区段 JSON 不可解析: %s", err)
	}
	if obj["name"] != "test" {
		t.Errorf("name = %v, 期望 test", obj["name"])
	}
}

func TestParseMarkedSections_Multiple(t *testing.T) {
	input := `---A---
{"id": 1}
---END_A---
中间文字
---A---
{"id": 2}
---END_A---`
	sections, err := ParseMarkedSections(input, "---A---", "---END_A---")
	if err != nil {
		t.Fatalf("ParseMarkedSections 返回错误: %s", err)
	}
	if len(sections) != 2 {
		t.Fatalf("期望 2 个区段, 得到 %d", len(sections))
	}
}

func TestParseMarkedSections_EmptySectionsSkipped(t *testing.T) {
	input := `---A---

---END_A---
---A---
{"valid": true}
---END_A---`
	sections, err := ParseMarkedSections(input, "---A---", "---END_A---")
	if err != nil {
		t.Fatalf("ParseMarkedSections 返回错误: %s", err)
	}
	if len(sections) != 1 {
		t.Fatalf("期望 1 个非空区段, 得到 %d", len(sections))
	}
}

func TestParseMarkedSections_NoMarkerReturnsError(t *testing.T) {
	input := "纯文本没有标记"
	_, err := ParseMarkedSections(input, "---A---", "---END_A---")
	if err == nil {
		t.Fatal("无标记时应返回 error")
	}
}

// ── EstimateTokens ─────────────────────────────────────────────────────

func TestEstimateTokens_Chinese(t *testing.T) {
	count := EstimateTokens("你好世界")
	if count <= 0 {
		t.Errorf("中文 token 数应 > 0, 得到 %d", count)
	}
	if count != 6 {
		t.Errorf("'你好世界' 期望 6 tokens, 得到 %d", count)
	}
}

func TestEstimateTokens_English(t *testing.T) {
	count := EstimateTokens("hello world test")
	if count <= 0 {
		t.Errorf("英文 token 数应 > 0, 得到 %d", count)
	}
}

func TestEstimateTokens_Mixed(t *testing.T) {
	count := EstimateTokens("你好 hello 世界 world")
	if count <= 0 {
		t.Errorf("混合文本 token 数应 > 0, 得到 %d", count)
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	count := EstimateTokens("")
	if count != 0 {
		t.Errorf("空字符串期望 0, 得到 %d", count)
	}
}

// ── Max / Min ──────────────────────────────────────────────────────────

func TestMax_Positive(t *testing.T) {
	if got := Max(3, 5); got != 5 {
		t.Errorf("Max(3,5) = %d, 期望 5", got)
	}
	if got := Max(5, 3); got != 5 {
		t.Errorf("Max(5,3) = %d, 期望 5", got)
	}
}

func TestMax_Negative(t *testing.T) {
	if got := Max(-3, -1); got != -1 {
		t.Errorf("Max(-3,-1) = %d, 期望 -1", got)
	}
}

func TestMax_Equal(t *testing.T) {
	if got := Max(7, 7); got != 7 {
		t.Errorf("Max(7,7) = %d, 期望 7", got)
	}
}

func TestMin_Positive(t *testing.T) {
	if got := Min(3, 5); got != 3 {
		t.Errorf("Min(3,5) = %d, 期望 3", got)
	}
	if got := Min(5, 3); got != 3 {
		t.Errorf("Min(5,3) = %d, 期望 3", got)
	}
}

func TestMin_Negative(t *testing.T) {
	if got := Min(-3, -1); got != -3 {
		t.Errorf("Min(-3,-1) = %d, 期望 -3", got)
	}
}

func TestMin_Equal(t *testing.T) {
	if got := Min(7, 7); got != 7 {
		t.Errorf("Min(7,7) = %d, 期望 7", got)
	}
}

// ── TruncateRef ────────────────────────────────────────────────────────

func TestTruncateRef_UnderLimit(t *testing.T) {
	input := "短文本"
	result, truncated := TruncateRef(input)
	if truncated {
		t.Error("短于 RefLimit 不应截断")
	}
	if result != input {
		t.Errorf("返回值应与输入一致, 得到 %q", result)
	}
}

func TestTruncateRef_OverLimit(t *testing.T) {
	input := make([]rune, RefLimit+100)
	for i := range input {
		input[i] = 'a'
	}
	str := string(input)
	result, truncated := TruncateRef(str)
	if !truncated {
		t.Error("超过 RefLimit 应截断")
	}
	resultRunes := []rune(result)
	if len(resultRunes) > RefLimit+50 {
		t.Errorf("截断后长度应 ~RefLimit, 得到 %d", len(resultRunes))
	}
	if len(result) <= RefLimit {
		t.Error("截断后应包含提示文字")
	}
}
