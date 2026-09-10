package memory

import (
	"reflect"
	"testing"
	"time"
)

// Pin/Unpin 闭环（5.3）：切换落列 + pin/unpin 事件留痕 + 未命中报错。
func TestPinUnpinLifecycle(t *testing.T) {
	gdb := graphTestDB(t)
	store := SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	if _, err := store.Save(Memory{Name: "keep-me", Body: "b"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.Pin("keep-me"); err != nil {
		t.Fatalf("pin: %v", err)
	}
	got, ok := store.Get("keep-me")
	if !ok || !got.Pinned {
		t.Fatalf("pin 后 Pinned 应为 true: %+v", got)
	}
	// Save 覆盖（remember 工具重写正文）不得丢固化态（ON CONFLICT 不触 pinned 列）。
	if _, err := store.Save(Memory{Name: "keep-me", Body: "改写后的正文"}); err != nil {
		t.Fatalf("resave: %v", err)
	}
	got, _ = store.Get("keep-me")
	if !got.Pinned {
		t.Fatal("save 覆盖后固化态丢失")
	}
	if err := store.Unpin("keep-me"); err != nil {
		t.Fatalf("unpin: %v", err)
	}
	got, _ = store.Get("keep-me")
	if got.Pinned {
		t.Fatal("unpin 后仍固化")
	}
	// 事件留痕：pin 与 unpin 各一条，顺序正确。
	events, _ := (&EventLog{DB: gdb}).LoadEvents()
	var ops []string
	for _, e := range events {
		if e.Name == "keep-me" && (e.Op == OpPin || e.Op == OpUnpin) {
			ops = append(ops, e.Op)
		}
	}
	if !reflect.DeepEqual(ops, []string{OpPin, OpUnpin}) {
		t.Fatalf("pin 事件留痕不符: %v", ops)
	}
	// 未命中：明确报错（不存在）。
	if err := store.Pin("absent"); err == nil {
		t.Fatal("pin 不存在记忆应报错")
	}
	// 已归档不可 pin（固化只作用于活跃集合）。
	if _, err := store.Save(Memory{Name: "gone"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Archive("gone"); err != nil {
		t.Fatal(err)
	}
	if err := store.Pin("gone"); err == nil {
		t.Fatal("pin 已归档记忆应报错")
	}
}

// 固化豁免保留期硬删：CleanupArchived 跳过 pinned=1（「90 天一刀切」的替代）。
func TestPinnedExemptFromCleanup(t *testing.T) {
	gdb := graphTestDB(t)
	dir := t.TempDir()
	store := SQLiteStoreFor(gdb, dir, dir)
	for _, n := range []string{"plain", "guarded"} {
		if _, err := store.Save(Memory{Name: n, Description: n}); err != nil {
			t.Fatal(err)
		}
	}
	// guarded 活跃时先固化再归档（正交：手动归档可作用于固化条，pinned 保留）。
	if err := store.Pin("guarded"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Archive("guarded"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Archive("plain"); err != nil {
		t.Fatal(err)
	}
	// 两行都回拨到 100 天前（超过默认 90 天保留期）。
	old := time.Now().UTC().Add(-100 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := gdb.Exec(`UPDATE facts SET updated_at=? WHERE name IN ('plain','guarded')`, old); err != nil {
		t.Fatal(err)
	}
	removed, err := store.CleanupArchived(time.Now().Add(-ArchivedRetention))
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if len(removed) != 1 || removed[0].Name != "plain" {
		t.Fatalf("只应清掉 plain（固化豁免）: %+v", removed)
	}
}

// 衰减评分纯函数：新鲜满分 / 半衰期减半 / 长期闲置下限 0.01 / 固化恒 1 /
// 未来时间戳与零值时间戳不造数。
func TestDecayScore(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	mk := func(daysAgo float64, used bool) Memory {
		m := Memory{Name: "x"}
		if used {
			m.LastUsedAt = now.Add(-time.Duration(daysAgo * 24 * float64(time.Hour)))
		}
		return m
	}
	if s := DecayScore(mk(0, true), now); s <= 0.99 || s > 1.0 {
		t.Fatalf("刚触达应接近满分: %v", s)
	}
	if s := DecayScore(mk(30, true), now); s < 0.49 || s > 0.51 {
		t.Fatalf("30 天闲置应约 0.5: %v", s)
	}
	if s := DecayScore(mk(90, true), now); s < 0.12 || s > 0.13 {
		t.Fatalf("90 天闲置应约 0.125: %v", s)
	}
	if s := DecayScore(mk(3650, true), now); s != 0.01 {
		t.Fatalf("长期闲置应到下限 0.01: %v", s)
	}
	pinned := mk(3650, true)
	pinned.Pinned = true
	if s := DecayScore(pinned, now); s != 1.0 {
		t.Fatalf("固化恒满分: %v", s)
	}
	if s := DecayScore(mk(0, false), now); s != 1.0 {
		t.Fatalf("无触达证据按满分（诚实不造数）: %v", s)
	}
}

// 三态分类：固化 > 归档外活跃集合的 衰减/活跃 二分；阈值缺省 60 天。
func TestLifecycleOf(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	pinned := Memory{Name: "p", Pinned: true}
	if LifecycleOf(pinned, now, 0) != LifecyclePinned {
		t.Fatal("固化应最优先")
	}
	fresh := Memory{Name: "f", LastUsedAt: now.Add(-10 * 24 * time.Hour)}
	if LifecycleOf(fresh, now, 0) != LifecycleActive {
		t.Fatal("10 天未用应活跃")
	}
	stale := Memory{Name: "s", LastUsedAt: now.Add(-90 * 24 * time.Hour)}
	if LifecycleOf(stale, now, 0) != LifecycleDecaying {
		t.Fatal("90 天未用应衰减")
	}
	noStamp := Memory{Name: "n"}
	if LifecycleOf(noStamp, now, 0) != LifecycleActive {
		t.Fatal("无时间戳（存量行）应按活跃处理")
	}
}

// 晨报/预载排序固化加权：固化条先于更近的非固化条（排序输出确定）。
func TestMorningSortPinnedFirst(t *testing.T) {
	mems := []Memory{
		{Name: "recent", UpdatedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)},
		{Name: "pinned-old", Pinned: true, UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	morningRecencySort(mems)
	if mems[0].Name != "pinned-old" {
		t.Fatalf("固化条应排在最前: %s", mems[0].Name)
	}
}
