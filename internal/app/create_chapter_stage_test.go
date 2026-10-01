package app

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// TestIsStageOpener 阶段开篇判定：第 21/41/… 章为阶段开篇；第 1 章是全书开篇
// 走 create-chapter-first，不算阶段开篇。
func TestIsStageOpener(t *testing.T) {
	cases := map[int]bool{
		0: false, 1: false, 2: false, 20: false,
		21: true, 40: false, 41: true, 61: true,
	}
	for num, want := range cases {
		if got := isStageOpener(num); got != want {
			t.Errorf("isStageOpener(%d) = %v, want %v", num, got, want)
		}
	}
}

// TestStageRange 阶段开篇章对应的上一阶段区间。
func TestStageRange(t *testing.T) {
	if s, e := stageRange(21); s != 1 || e != 20 {
		t.Errorf("stageRange(21) = (%d,%d), want (1,20)", s, e)
	}
	if s, e := stageRange(41); s != 21 || e != 40 {
		t.Errorf("stageRange(41) = (%d,%d), want (21,40)", s, e)
	}
}

// TestStageRecapSection_NonStageEmpty 非阶段开篇章零注入。
func TestStageRecapSection_NonStageEmpty(t *testing.T) {
	pm := newStatesTestProject(t)
	a := &writingState{core: &core{}}
	for _, num := range []int{1, 5, 20, 22, 40} {
		if got := a.stageRecapSection(pm, num); got != "" {
			t.Errorf("第%d章不应注入阶段总结，got:\n%s", num, got)
		}
	}
}

// TestStageRecapSection_FallbackDigest 阶段开篇章：client 为 nil 时 LLM 合成
// 不可用，回落逐章摘要清单——注入段仍完整（总结头 + 逐章行 + 开篇纪律）。
func TestStageRecapSection_FallbackDigest(t *testing.T) {
	pm := newStatesTestProject(t)
	nodes := make([]types.OutlineNode, 0, 20)
	for i := 1; i <= 20; i++ {
		nodes = append(nodes, types.OutlineNode{
			ID:         fmt.Sprintf("ch-%02d", i),
			Title:      "第" + strconv.Itoa(i) + "章",
			OrderIndex: i,
			Summary:    "第" + strconv.Itoa(i) + "章剧情摘要",
		})
	}
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	a := &writingState{core: &core{}} // client==nil → 回落路径
	got := a.stageRecapSection(pm, 21)
	if got == "" {
		t.Fatal("第21章应注入阶段总结段")
	}
	for _, want := range []string{"【上一阶段总结（第1~20章）】", "第1章：第1章剧情摘要", "第20章：第20章剧情摘要", "【新阶段开篇（第21章）纪律】"} {
		if !strings.Contains(got, want) {
			t.Errorf("注入段缺 %q:\n%s", want, got)
		}
	}
}

// TestStageRecapSection_NoMaterial 素材为空（无大纲无摘要）时零注入——不假装
// 能总结不存在的剧情。
func TestStageRecapSection_NoMaterial(t *testing.T) {
	pm := newStatesTestProject(t)
	a := &writingState{core: &core{}}
	if got := a.stageRecapSection(pm, 21); got != "" {
		t.Errorf("无素材应返回空串，got:\n%s", got)
	}
}
