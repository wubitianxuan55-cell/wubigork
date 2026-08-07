package app

import (
	"testing"

	"github.com/gaea/gaea/internal/office/proposal"
)

// 验收场景：轻语（右脑）记住甲方偏好 → 跨脑检索在方案（左脑）写作时可命中。
func TestBrainAcceptanceRightToLeftScenario(t *testing.T) {
	rightFake := &fakeRightBrain{rows: map[string]string{}}
	leftBrain := &leftBrain{src: &fakeLeftSource{ps: []proposal.Proposal{
		{ID: "p1", Title: "土壤修复标书", Category: "环保工程", Requirements: "报价策略见甲方偏好"},
	}}}
	bs := &BrainStore{main: &fakeMainBrain{rows: map[string]string{}}, left: leftBrain, right: rightFake}

	if err := bs.Write(BrainRight, "甲方A", "偏好", "保守报价"); err != nil {
		t.Fatal(err)
	}
	hits, err := bs.Search("甲方A 报价", BrainRight)
	if err != nil || len(hits) == 0 {
		t.Fatalf("右脑命中失败: %+v %v", hits, err)
	}
	// 同一查询可命中两脑（右脑偏好 + 左脑标书）。
	all, err := bs.Search("报价")
	if err != nil || len(all) == 0 {
		t.Fatalf("跨脑检索失败: %+v %v", all, err)
	}
	brains := map[string]bool{}
	for _, h := range all {
		brains[h.Brain] = true
	}
	if !brains[BrainRight] || !brains[BrainLeft] {
		t.Fatalf("应命中右脑与左脑: %+v", brains)
	}
}
