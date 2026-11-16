package chat

import (
	"strconv"
	"path/filepath"
	"strings"
	"testing"
)

// A 批分支树单测（chat-demo branches.ts 语义对齐，docs/gaea-chat-branch-tree-design-2026-11.md）。

func TestParseBT_MergeExtraBT(t *testing.T) {
	// 缺失/非法 → not ok
	if _, ok := ParseBT(""); ok {
		t.Fatal("空 extra 不应解析出 bt")
	}
	if _, ok := ParseBT(`{"answered_by":1}`); ok {
		t.Fatal("无 bt 键不应解析出 bt")
	}
	if _, ok := ParseBT(`{"bt":"oops"`); ok {
		t.Fatal("非法 JSON 不应解析出 bt")
	}
	// merge 保留既有键
	merged, err := MergeExtraBT(`{"answered_by":{"engine":"x"},"reasoning":"abc"}`, BT{P: 7, C: 0})
	if err != nil {
		t.Fatalf("MergeExtraBT: %v", err)
	}
	bt, ok := ParseBT(merged)
	if !ok || bt.P != 7 {
		t.Fatalf("merge 后 bt = %+v ok=%v", bt, ok)
	}
	if !strings.Contains(merged, `"answered_by"`) || !strings.Contains(merged, `"reasoning"`) {
		t.Fatalf("merge 丢键: %s", merged)
	}
	// 覆盖旧 bt
	merged2, _ := MergeExtraBT(merged, BT{P: 7, C: 9})
	if bt2, _ := ParseBT(merged2); bt2.C != 9 {
		t.Fatalf("二次 merge 应覆盖 c: %+v", bt2)
	}
}

func TestVisiblePath_LinearPassthrough(t *testing.T) {
	msgs := []Message{
		{ID: 1, Role: "user", Content: "a"},
		{ID: 2, Role: "assistant", Content: "b"},
	}
	got := VisiblePath(msgs)
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("线形话题应直通: %+v", got)
	}
}

func treeFixture() []Message {
	// 问题1 → 回答1a（选中）/回答1b；回答1a 下 → 问题2 → 回答2
	// bt: 1{p0,c=1a} 1a{p1,c=2} 1b{p1,c0} 2{p1a,c0}...
	// 回答1b 是隐藏分支（1 的 c 指向 1a）。
	return []Message{
		{ID: 1, Role: "user", Content: "q1", Extra: `{"bt":{"p":0,"c":10}}`},
		{ID: 10, Role: "assistant", Content: "a1a", Extra: `{"bt":{"p":1,"c":2}}`},
		{ID: 11, Role: "assistant", Content: "a1b", Extra: `{"bt":{"p":1,"c":0}}`},
		{ID: 2, Role: "user", Content: "q2", Extra: `{"bt":{"p":10,"c":20}}`},
		{ID: 20, Role: "assistant", Content: "a2", Extra: `{"bt":{"p":2,"c":0}}`},
	}
}

func TestVisiblePath_TreeWalk(t *testing.T) {
	path := VisiblePath(treeFixture())
	var ids []int64
	for _, m := range path {
		ids = append(ids, m.ID)
	}
	want := []int64{1, 10, 2, 20}
	if len(ids) != len(want) {
		t.Fatalf("路径 = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("路径 = %v, want %v", ids, want)
		}
	}
}

func TestVisiblePath_SwitchSelection(t *testing.T) {
	msgs := treeFixture()
	// 把 1 的选中子切到 11（隐藏分支）
	msgs[0].Extra = `{"bt":{"p":0,"c":11}}`
	path := VisiblePath(msgs)
	var ids []int64
	for _, m := range path {
		ids = append(ids, m.ID)
	}
	// 切到 1b 后：1 → 11（后续 2/20 挂在被隐藏的 10 下，不进路径）
	want := []int64{1, 11}
	if len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] {
		t.Fatalf("切换后路径 = %v, want %v", ids, want)
	}
}

func TestVisiblePath_PartialWindow(t *testing.T) {
	// 分页窗口截断：父（1）不在集合，10/2/20 可走，11 无引用为叶子但更旧——
	// 叶子回溯应从 20 走到 10（父 1 缺失截断）。
	msgs := []Message{
		{ID: 10, Role: "assistant", Content: "a1a", Extra: `{"bt":{"p":1,"c":2}}`},
		{ID: 11, Role: "assistant", Content: "a1b", Extra: `{"bt":{"p":1,"c":0}}`},
		{ID: 2, Role: "user", Content: "q2", Extra: `{"bt":{"p":10,"c":20}}`},
		{ID: 20, Role: "assistant", Content: "a2", Extra: `{"bt":{"p":2,"c":0}}`},
	}
	path := VisiblePath(msgs)
	var ids []int64
	for _, m := range path {
		ids = append(ids, m.ID)
	}
	want := []int64{10, 2, 20}
	if len(ids) != len(want) {
		t.Fatalf("窗口截断路径 = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("窗口截断路径 = %v, want %v", ids, want)
		}
	}
}

func TestStore_BranchLifecycle(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "chat"))
	t.Cleanup(func() { _ = s.Close() })
	if err := s.CreateTopic("t1", "分支", "plain"); err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	// 线形两轮
	u1, _ := s.AppendMessage("t1", "user", "q1", "")
	a1, _ := s.AppendMessage("t1", "assistant", "a1", `{"answered_by":{"engine":"e"}}`)
	u2, _ := s.AppendMessage("t1", "user", "q2", "")
	_, _ = s.AppendMessage("t1", "assistant", "a2", "")

	if !BTNeeded(mustList(t, s, "t1")) {
		t.Fatal("未规范化话题应需要规范化")
	}
	if err := s.NormalizeTopicBranches("t1"); err != nil {
		t.Fatalf("NormalizeTopicBranches: %v", err)
	}
	if BTNeeded(mustList(t, s, "t1")) {
		t.Fatal("规范化后不应再需要规范化")
	}
	// 规范化保留既有 extra 键
	a1Now, err := s.GetMessage(a1.ID)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if !strings.Contains(a1Now.Extra, `"answered_by"`) {
		t.Fatalf("规范化丢 extra 键: %s", a1Now.Extra)
	}
	if bt, _ := ParseBT(a1Now.Extra); bt.P != u1.ID || bt.C != u2.ID {
		t.Fatalf("规范化线性链错误: %+v", bt)
	}
	// 幂等
	if err := s.NormalizeTopicBranches("t1"); err != nil {
		t.Fatalf("二次 Normalize: %v", err)
	}

	// 重新生成：a1 下挂 a1b 并选中（真实流程：assistant 行落库时已带 bt.p=问题）
	a1b, err := s.AppendMessage("t1", "assistant", "a1b", `{"bt":{"p":`+i64str(a1.ID)+`}}`)
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if err := s.SetSelectedChild(a1.ID, a1b.ID); err != nil {
		t.Fatalf("SetSelectedChild: %v", err)
	}
	path := VisiblePath(mustList(t, s, "t1"))
	// 切到 a1b：u1 → a1 → a1b（u2/a2 挂在 a1 的旧选中子下被隐藏）
	if len(path) != 3 || path[0].ID != u1.ID || path[1].ID != a1.ID || path[2].ID != a1b.ID {
		t.Fatalf("切换后路径应 [u1 a1 a1b]: %+v", path)
	}
	// 越枝改写被拒：用 u2（父是 a1）冒充 a1 的子——合法；换不同话题的消息非法
	if err := s.CreateTopic("t2", "别的话题", "plain"); err != nil {
		t.Fatalf("CreateTopic t2: %v", err)
	}
	ox, _ := s.AppendMessage("t2", "user", "别的话题", `{"bt":{"p":0,"c":0}}`)
	if err := s.SetSelectedChild(a1.ID, ox.ID); err == nil {
		t.Fatal("跨话题 SetSelectedChild 应报错")
	}
	// 父子链不成立被拒：u2 的父是 a1，但把 u2 的父行改成 a1b 后再选 a1→u2 应失败
	if err := s.SetSelectedChild(a1b.ID, u2.ID); err == nil {
		t.Fatal("父子链不成立的 SetSelectedChild 应报错")
	}
}

func i64str(v int64) string {
	return strconv.FormatInt(v, 10)
}

func mustList(t *testing.T, s *Store, topicID string) []Message {
	t.Helper()
	msgs, err := s.ListMessages(topicID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	return msgs
}
