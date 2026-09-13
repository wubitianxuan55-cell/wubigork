package bookimport

import (
	"strconv"
	"strings"
	"testing"
)

// ── 篇幅路由（oh-story T4）──

func TestRouteTier(t *testing.T) {
	cases := []struct {
		words, chapters int
		want            Tier
		note            string
	}{
		{0, 0, TierShort, "零章=短篇"},
		{10000, 3, TierShort, "字数与章数均低于短篇线"},
		{29999, 100, TierShort, "字数贴着短篇上界内侧"},
		{30000, 100, TierMid, "字数到上界即中篇"},
		{500000, 100, TierMid, "中间地带"},
		{799999, 299, TierMid, "长篇双线内侧"},
		{800000, 100, TierLong, "字数过线即长篇"},
		{500000, 300, TierLong, "章数过线即长篇"},
		{900000, 299, TierLong, "长篇信号优先于短篇章数"},
		{100, 2, TierShort, "少量短章=短篇"},
	}
	for _, c := range cases {
		if got := RouteTier(c.words, c.chapters); got != c.want {
			t.Errorf("RouteTier(%d,%d)=%s want %s（%s）", c.words, c.chapters, got, c.want, c.note)
		}
	}
}

func TestSegmentSize(t *testing.T) {
	if SegmentSize(TierShort) != 0 {
		t.Fatal("短篇=章级细纲不聚合")
	}
	if SegmentSize(TierMid) != 10 || SegmentSize(TierLong) != 30 {
		t.Fatal("中/长篇聚合粒度应为 10/30")
	}
}

func TestAggregateSkeleton(t *testing.T) {
	items := make([]OutlineStructure, 0, 25)
	for i := 1; i <= 25; i++ {
		items = append(items, OutlineStructure{ChapterNumber: i, Title: "标题" + strconv.Itoa(i), Summary: "第" + strconv.Itoa(i) + "章概要"})
	}
	segs := AggregateSkeleton(items, 10)
	if len(segs) != 3 {
		t.Fatalf("25 章按 10 聚合应为 3 段: %d", len(segs))
	}
	if segs[0].ChapterFrom != 1 || segs[0].ChapterTo != 10 || segs[0].Title != "第1-10章" {
		t.Fatalf("首段跨度不符: %+v", segs[0])
	}
	if segs[2].ChapterFrom != 21 || segs[2].ChapterTo != 25 || segs[2].Title != "第21-25章" {
		t.Fatalf("尾段余数不符: %+v", segs[2])
	}
	if !strings.Contains(segs[0].Summary, "标题10") || len([]rune(segs[0].Summary)) > 200+2 {
		t.Fatalf("摘要应由章标题拼接并截断: %q", segs[0].Summary)
	}

	if AggregateSkeleton(items, 0) != nil {
		t.Fatal("segSize=0（章级）不应聚合")
	}
	if AggregateSkeleton(nil, 10) != nil {
		t.Fatal("空条目不应聚合")
	}
}

