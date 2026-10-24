package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── 刀 A2（Reasonix 2.x compact_user_turns 蒸馏）测试：digest 后用户轮的
// 逐字保留预算——最老优先 / 固定比率估价 / 丢弃显形 / 确定性。──────────

func userTurnRegion() []provider.Message {
	// 摘要之后（policyStart 之后）的一段：用户轮与工具工作交错。
	return []provider.Message{
		{Role: provider.RoleUser, Content: summaryTagOpen + "\nold digest\n" + summaryTagClose},
		{Role: provider.RoleUser, Content: "第一条用户约束 " + strings.Repeat("a", 200)},
		{Role: provider.RoleAssistant, Content: "working " + strings.Repeat("w", 400)},
		{Role: provider.RoleTool, Name: "bash", Content: "output " + strings.Repeat("o", 400)},
		{Role: provider.RoleUser, Content: "第二条用户约束 " + strings.Repeat("b", 200)},
		{Role: provider.RoleAssistant, Content: "more work " + strings.Repeat("w", 400)},
	}
}

func TestKeepUserTurnsRetainsMidSessionTurns(t *testing.T) {
	a := testAgentC(20000, 0.8, 5) // 预算=1000 tok，固定比率 0.25 → ~4000 字符额度
	kept, fold, ret := a.partitionFold(userTurnRegion())
	if ret.Kept != 2 || ret.Dropped != 0 {
		t.Fatalf("两条用户轮都应在预算内保留: kept=%d dropped=%d", ret.Kept, ret.Dropped)
	}
	for _, m := range kept {
		if m.Role == provider.RoleUser && strings.Contains(m.Content, "第一条用户约束") {
			return // 找到了
		}
	}
	t.Fatal("用户约束轮应进 kept 区")
	if len(fold) == 0 {
		t.Fatal("assistant/tool 工作应仍被折叠")
	}
}

func TestKeepUserTurnsOldestFirstWhenTight(t *testing.T) {
	// 预算只够一条：最老优先（老轮活过更多次折叠）。
	a := testAgentC(2000, 0.8, 5) // 预算=100 tok → ~400 字符
	_, _, ret := a.partitionFold(userTurnRegion())
	if ret.Kept != 1 || ret.Dropped != 1 {
		t.Fatalf("紧预算应保一丢一: kept=%d dropped=%d", ret.Kept, ret.Dropped)
	}
	kept, _, _ := a.partitionFold(userTurnRegion())
	for _, m := range kept {
		if m.Role == provider.RoleUser && strings.HasPrefix(m.Content, "第二条") {
			t.Fatal("最老优先：应保留第一条而非第二条")
		}
	}
}

func TestKeepUserTurnsOversizedTurnDroppedAndSurfaced(t *testing.T) {
	a := testAgentC(20000, 0.8, 5)
	sink := &collectorSink{}
	a.sink = sink
	s := NewSession("system prompt")
	s.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("u", 20000)}) // 超预算大轮
	for i := 0; i < 10; i++ {
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "work " + strings.Repeat("w", 600)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "bash", Content: "out " + strings.Repeat("o", 600)})
	}
	a.session = s
	a.compaction.LastPrompt = 20000 // 触发压缩
	a.maybeCompact(context.Background(), nil)
	found := false
	for _, e := range sink.events {
		if e.Kind == event.Notice && strings.Contains(e.Text, "[[keep]]") {
			found = true
		}
	}
	if !found {
		t.Fatal("大用户轮被丢必须上屏并告知 [[keep]] 逃逸通道")
	}
}

func TestKeepUserTurnsFixedRatioNotCalibrated(t *testing.T) {
	// 估价用固定比率（0.25），不随 lastUsage 校准比率漂移——同一轮这次留、
	// 下次不能折（确定性要求，上游 fixedTokenEstimate 注释口径）。
	a := testAgentC(20000, 0.8, 5)
	a.lastUsage.Store(&provider.Usage{PromptTokens: 999999}) // 极端校准比率
	_, _, ret := a.partitionFold(userTurnRegion())
	if ret.Kept != 2 || ret.Dropped != 0 {
		t.Fatalf("固定比率下保留判定应与无 usage 时一致: kept=%d dropped=%d", ret.Kept, ret.Dropped)
	}
}

func TestPartitionFoldDeterministic(t *testing.T) {
	a := testAgentC(20000, 0.8, 5)
	region := userTurnRegion()
	k1, f1, r1 := a.partitionFold(region)
	k2, f2, r2 := a.partitionFold(region)
	if r1 != r2 || len(k1) != len(k2) || len(f1) != len(f2) {
		t.Fatal("同输入两次 partitionFold 应完全一致")
	}
	for i := range k1 {
		if k1[i].Content != k2[i].Content {
			t.Fatalf("kept[%d] 内容漂移", i)
		}
	}
}

func TestKeepUserTurnsMarkedDoesNotConsumeBudget(t *testing.T) {
	// [[keep]] 标记轮由 keep policy 持有（自己的预算），不占用户轮预算。
	a := testAgentC(2000, 0.8, 5) // 紧预算
	region := []provider.Message{
		{Role: provider.RoleUser, Content: "[[keep]] 标记轮 " + strings.Repeat("m", 200)},
		{Role: provider.RoleUser, Content: "普通轮 " + strings.Repeat("n", 200)},
	}
	keep := keepIndexes(region, KeepUserMarked)
	ret := a.keepUserTurns(region, keep)
	if !keep[0] {
		t.Fatal("标记轮应被 keep policy 持有")
	}
	if ret.Dropped != 0 {
		t.Fatalf("标记轮不应占用户轮预算（普通轮应仍有额度）: dropped=%d", ret.Dropped)
	}
}
