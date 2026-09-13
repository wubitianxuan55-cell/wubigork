package types

import "testing"

func boolPtr(b bool) *bool { return &b }

func fWithStatus(s ForeshadowStatus) Foreshadow { return Foreshadow{Status: s} }

// TestUrgencyLevel spec §10.1：target-cur = -1,0,1,2,3,5 → 3,2,2,2,1,1（remind=5）。
func TestUrgencyLevel(t *testing.T) {
	cases := []struct {
		target, cur, want int
	}{
		{-1, 0, 0}, // 无计划回收章不猜值
		{5, 6, 3},  // 已超期 1 章
		{5, 5, 2},  // 本章必须回收
		{6, 5, 2},  // 还剩 1 章
		{7, 5, 2},  // 还剩 2 章（≤ 阈值 2）
		{8, 5, 1},  // 还剩 3 章（≤ remind 5）
		{10, 5, 1}, // 还剩 5 章（== remind）
		{11, 5, 0}, // 还剩 6 章（> remind）
	}
	for _, c := range cases {
		f := fWithStatus(ForeshadowPlanted)
		if c.target == 0 {
			f.TargetResolveIn = ""
		} else {
			f.TargetResolveIn = chapterFileOfNum(c.target)
		}
		if got := UrgencyLevel(f, c.cur); got != c.want {
			t.Errorf("UrgencyLevel(target=%d, cur=%d) = %d, want %d", c.target, c.cur, got, c.want)
		}
	}
}

// TestUrgencyLevel_StatusMatrix D15：partially_resolved/hinted 有压力；
// revealed/resolved 别名/abandoned 恒 0；自定义 remind 生效。
func TestUrgencyLevel_StatusMatrix(t *testing.T) {
	// 部分回收仍有回收压力（D15 修正：MuMu 把它判 0）
	f := fWithStatus(ForeshadowPartial)
	f.TargetResolveIn = "005.md"
	if got := UrgencyLevel(f, 5); got != 2 {
		t.Errorf("partially_resolved 本章未回收应为 2, got %d", got)
	}
	// hinted 视同已埋入（gaea 原生态）
	f.Status = ForeshadowHinted
	if got := UrgencyLevel(f, 5); got != 2 {
		t.Errorf("hinted 应与 planted 同口径, got %d", got)
	}
	// 终态恒 0
	for _, s := range []ForeshadowStatus{ForeshadowRevealed, ForeshadowResolved, ForeshadowAbandoned} {
		if got := UrgencyLevel(fWithStatus(s), 5); got != 0 {
			t.Errorf("status=%s 应为 0, got %d", s, got)
		}
	}
	// 自定义 remind 窗口
	f = fWithStatus(ForeshadowPlanted)
	f.TargetResolveIn = "010.md"
	f.RemindBeforeChapters = 20
	if got := UrgencyLevel(f, 5); got != 1 {
		t.Errorf("remind=20 时还剩 5 章应为 1, got %d", got)
	}
	// remind 非法回退缺省 5（10-5=5 剩余 → 1）
	f.RemindBeforeChapters = -3
	if got := UrgencyLevel(f, 5); got != 1 {
		t.Errorf("remind 非法应回退 5, got %d", got)
	}
}

// TestClassifyResolve 四值判定全覆盖，含无计划 → no_plan。
func TestClassifyResolve(t *testing.T) {
	mk := func(target string) Foreshadow {
		f := fWithStatus(ForeshadowPlanted)
		f.TargetResolveIn = target
		return f
	}
	cases := []struct {
		f    Foreshadow
		cur  int
		want ResolveStatus
	}{
		{mk("005.md"), 5, ResolveMustNow},
		{mk("004.md"), 5, ResolveOverdue},
		{mk("006.md"), 5, ResolveNotYet},
		{mk(""), 5, ResolveNoPlan},
		{mk("abc.md"), 5, ResolveNoPlan}, // 文件名无章号 = 无计划
	}
	for i, c := range cases {
		if got := ClassifyResolve(c.f, c.cur); got != c.want {
			t.Errorf("case %d: ClassifyResolve = %q, want %q", i, got, c.want)
		}
	}
}

// TestForeshadowLayerOf 分层矩阵：四层 + 兜底层 + 远期/终态/开关抑制。
func TestForeshadowLayerOf(t *testing.T) {
	mk := func(status ForeshadowStatus, target string) Foreshadow {
		f := fWithStatus(status)
		f.TargetResolveIn = target
		f.PlantedIn = "003.md"
		return f
	}
	cases := []struct {
		name string
		f    Foreshadow
		cur  int
		want ForeshadowLayer
	}{
		{"本章必须回收", mk(ForeshadowPlanted, "005.md"), 5, ForeshadowLayerMust},
		{"超期", mk(ForeshadowPlanted, "004.md"), 5, ForeshadowLayerOverdue},
		{"近期参考", mk(ForeshadowPlanted, "008.md"), 5, ForeshadowLayerNear},
		{"远期不注入", mk(ForeshadowPlanted, "011.md"), 5, ForeshadowLayerNone},
		{"无计划兜底", mk(ForeshadowPlanted, ""), 5, ForeshadowLayerNoPlan},
		{"hinted 视同埋入", mk(ForeshadowHinted, "005.md"), 5, ForeshadowLayerMust},
		{"partial 视同埋入", mk(ForeshadowPartial, "004.md"), 5, ForeshadowLayerOverdue},
		{"已回收不注入", mk(ForeshadowRevealed, "005.md"), 5, ForeshadowLayerNone},
		{"resolved 别名不注入", mk(ForeshadowResolved, "005.md"), 5, ForeshadowLayerNone},
		{"废弃不注入", mk(ForeshadowAbandoned, "005.md"), 5, ForeshadowLayerNone},
		{"pending 本章计划埋入", mk(ForeshadowPending, ""), 3, ForeshadowLayerPlant},
		{"pending 非本章不注入", mk(ForeshadowPending, ""), 5, ForeshadowLayerNone},
	}
	for _, c := range cases {
		if got := ForeshadowLayerOf(c.f, c.cur, ForeshadowLookaheadDefault); got != c.want {
			t.Errorf("%s: ForeshadowLayerOf = %v, want %v", c.name, got, c.want)
		}
	}
	// include_in_context=false 显式排除
	f := mk(ForeshadowPlanted, "005.md")
	f.IncludeInContext = boolPtr(false)
	if got := ForeshadowLayerOf(f, 5, ForeshadowLookaheadDefault); got != ForeshadowLayerNone {
		t.Errorf("include_in_context=false 应不注入, got %v", got)
	}
	// auto_remind=false 只抑制 L3，不抑制 L1/L2（D8）
	f = mk(ForeshadowPlanted, "008.md")
	f.AutoRemind = boolPtr(false)
	if got := ForeshadowLayerOf(f, 5, ForeshadowLookaheadDefault); got != ForeshadowLayerNone {
		t.Errorf("auto_remind=false 应抑制近期参考层, got %v", got)
	}
	f.TargetResolveIn = "005.md"
	if got := ForeshadowLayerOf(f, 5, ForeshadowLookaheadDefault); got != ForeshadowLayerMust {
		t.Errorf("auto_remind=false 不得抑制必须回收层, got %v", got)
	}
	// lookahead 非法回退缺省 5：target=cur+5 → Near；cur+6 → None
	f = mk(ForeshadowPlanted, "010.md")
	if got := ForeshadowLayerOf(f, 5, -1); got != ForeshadowLayerNear {
		t.Errorf("lookahead 非法应回退 5（cur+5 近期）, got %v", got)
	}
	f.TargetResolveIn = "011.md"
	if got := ForeshadowLayerOf(f, 5, 0); got != ForeshadowLayerNone {
		t.Errorf("lookahead 非法回退 5（cur+6 应远期）, got %v", got)
	}
	// lookahead 上限钳制 20：cur+21 远期不注入
	f.TargetResolveIn = "026.md"
	if got := ForeshadowLayerOf(f, 5, 99); got != ForeshadowLayerNone {
		t.Errorf("lookahead 应钳制 20（cur+21 远期）, got %v", got)
	}
}

// TestForeshadowLayerPriority 预算消耗顺序唯一且单调。
func TestForeshadowLayerPriority(t *testing.T) {
	order := []ForeshadowLayer{
		ForeshadowLayerMust, ForeshadowLayerOverdue, ForeshadowLayerNear,
		ForeshadowLayerPlant, ForeshadowLayerNoPlan,
	}
	for i := 1; i < len(order); i++ {
		if ForeshadowLayerPriority(order[i-1]) >= ForeshadowLayerPriority(order[i]) {
			t.Errorf("层优先级应严格递增: %v vs %v", order[i-1], order[i])
		}
	}
}

// chapterFileOfNum 测试助手：章号 → "005.md" 形态文件名。
func chapterFileOfNum(n int) string {
	s := "000"
	if n > 0 {
		s = ""
		for x := n; x > 0; x /= 10 {
			s = string(rune('0'+x%10)) + s
		}
		for len(s) < 3 {
			s = "0" + s
		}
	}
	return s + ".md"
}
