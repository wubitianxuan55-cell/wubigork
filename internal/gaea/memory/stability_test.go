package memory

import (
	"math"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

func approxF(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// StabilityDays：基线 30d，1.5^cites 增长，封顶 240d（×8）。
func TestStabilityDays(t *testing.T) {
	if !approxF(StabilityDays(0), 30) || !approxF(StabilityDays(-3), 30) {
		t.Fatalf("零/负引用应恒基线 30：0→%v -3→%v", StabilityDays(0), StabilityDays(-3))
	}
	if !approxF(StabilityDays(1), 45) {
		t.Fatalf("1 cite=%v want 45", StabilityDays(1))
	}
	if !approxF(StabilityDays(2), 67.5) {
		t.Fatalf("2 cites=%v want 67.5", StabilityDays(2))
	}
	// 封顶：1.5^5=7.59 < 8（227.8d），1.5^6=11.39 > 8 → 恒 240。
	if !approxF(StabilityDays(5), 30*7.59375) {
		t.Fatalf("5 cites=%v want 227.8125", StabilityDays(5))
	}
	if !approxF(StabilityDays(6), 240) || !approxF(StabilityDays(100), 240) {
		t.Fatalf("封顶失效：6→%v 100→%v", StabilityDays(6), StabilityDays(100))
	}
}

// RecallDecayFactor：pinned/无时间戳免疫；稳定性拉长半衰期（引用多的更慢衰减）。
func TestRecallDecayFactor(t *testing.T) {
	now := time.Now()
	immortal := Memory{Name: "p", Pinned: true, UpdatedAt: now.Add(-800 * 24 * time.Hour)}
	if RecallDecayFactor(immortal, 0, now) != 1.0 {
		t.Fatal("pinned 应免疫衰减")
	}
	noTime := Memory{Name: "nt"}
	if RecallDecayFactor(noTime, 0, now) != 1.0 {
		t.Fatal("无时间戳=无衰减证据，应满分")
	}
	old := now.Add(-100 * 24 * time.Hour)
	uncited := Memory{Name: "u", UpdatedAt: old}
	cited := Memory{Name: "c", UpdatedAt: old}
	du, dc := RecallDecayFactor(uncited, 0, now), RecallDecayFactor(cited, 5, now)
	if !approxF(du, math.Exp2(-100/30.0)) {
		t.Fatalf("未引用=%v", du)
	}
	if dc <= du {
		t.Fatalf("引用 5 次的衰减应更慢：cited=%v uncited=%v", dc, du)
	}
	if dc < 0.01 {
		t.Fatalf("下限失效：%v", dc)
	}
	// 未来时间戳=诚实不造数，满分。
	future := Memory{Name: "f", UpdatedAt: now.Add(24 * time.Hour)}
	if RecallDecayFactor(future, 0, now) != 1.0 {
		t.Fatal("未来时间戳应满分")
	}
}

// 融合的 Decay 乘子：新鲜度进 final；零值=未提供按 1（直构兼容）；pinned 豁免。
func TestFuseRecallArmsDecayFactor(t *testing.T) {
	in := []RecallFuseInput{
		{Name: "fresh", Lexical: 1.0, Decay: 1.0},
		{Name: "stale", Lexical: 1.0, Decay: 0.25},
		{Name: "legacy", Lexical: 1.0}, // 零值=未提供 → 按不衰减
		{Name: "pinned-stale", Lexical: 1.0, Decay: 0.25, Pinned: true},
	}
	out := FuseRecallArms(in)
	byName := map[string]RecallFusedHit{}
	for _, h := range out {
		byName[h.Name] = h
	}
	if !approxF(byName["fresh"].Final, RecallWeightLexical) {
		t.Fatalf("fresh=%v", byName["fresh"].Final)
	}
	if !approxF(byName["stale"].Final, RecallWeightLexical*0.25) {
		t.Fatalf("stale=%v", byName["stale"].Final)
	}
	if !approxF(byName["legacy"].Final, RecallWeightLexical) {
		t.Fatalf("legacy 零值 Decay 应按 1：%v", byName["legacy"].Final)
	}
	if !approxF(byName["pinned-stale"].Final, RecallWeightLexical+RecallWeightPinned) {
		t.Fatalf("pinned 应豁免 decay：%v", byName["pinned-stale"].Final)
	}
}

// CiteCounts：op=cite 聚合；无事件=空 map；file 后端=nil。
func TestStoreCiteCounts(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(Memory{Name: "cited", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{Name: "plain", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := s.AppendFeedbackEvent(Event{At: time.Now().UnixMilli(), Op: OpCite, Name: "cited"}); err != nil {
			t.Fatal(err)
		}
	}
	counts := s.CiteCounts()
	if counts["cited"] != 3 || counts["plain"] != 0 {
		t.Fatalf("counts=%v want cited=3", counts)
	}
	// file 后端诚实 nil。
	fileStore := Store{Dir: t.TempDir()}
	if fileStore.CiteCounts() != nil {
		t.Fatal("file 后端 CiteCounts 应 nil")
	}
}
