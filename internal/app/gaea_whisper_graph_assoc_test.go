package app

import (
	"testing"

	"github.com/gaea/gaea/internal/whisper"
)

// TestGaeaWhisperGraphSubgraph_IncludesEventChainAssociation 图谱子图并入
// event_chain 关联边（数据已有、此前只在索引里不可见——审计 §C 补口）。
func TestGaeaWhisperGraphSubgraph_IncludesEventChainAssociation(t *testing.T) {
	a := newChatServiceTestApp(t)
	orch := a.whisperState.getOrCreateOrch("pidAssoc")
	cleanupWhisperSession(t, "pidAssoc")

	// KG 三元组：让「工作」成为可查询的中心实体
	orch.KG.AddTriple(whisper.Triple{Subject: "工作", Predicate: "压力", Object: "大", Confidence: 0.8})
	// 两条事实 + event_chain 关联（工作 → 睡眠）
	fWork := orch.FactStore.Add(whisper.MemoryFact{ID: "fWork", Subject: "工作", Summary: "工作很忙"})
	fSleep := orch.FactStore.Add(whisper.MemoryFact{ID: "fSleep", Subject: "睡眠", Summary: "睡眠变差"})
	orch.AssocIndex.Add(whisper.Association{
		FactIDA: fWork.ID, FactIDB: fSleep.ID,
		AssociationType: "event_chain", Strength: 0.5,
	})

	sub, err := a.whisperState.GaeaWhisperGraphSubgraph("pidAssoc", "工作", 1)
	if err != nil {
		t.Fatalf("GaeaWhisperGraphSubgraph: %v", err)
	}

	found := false
	for _, e := range sub.Edges {
		if e.From == "工作" && e.To == "睡眠" && e.Type == "因果" {
			found = true
			if e.Weight != 0.5 {
				t.Errorf("关联边权重应为 strength 0.5, got %v", e.Weight)
			}
		}
	}
	if !found {
		t.Fatalf("子图应含 event_chain 因果边（工作→睡眠）, edges=%+v", sub.Edges)
	}
	hasSleepNode := false
	for _, n := range sub.Nodes {
		if n.ID == "睡眠" {
			hasSleepNode = true
		}
	}
	if !hasSleepNode {
		t.Errorf("子图应含「睡眠」节点, nodes=%+v", sub.Nodes)
	}
}

// TestGaeaWhisperGraphSubgraph_AssociationNotConnectedSkipped 与查询实体不连通的
// 关联不入子图（保持以查询实体为中心）。
func TestGaeaWhisperGraphSubgraph_AssociationNotConnectedSkipped(t *testing.T) {
	a := newChatServiceTestApp(t)
	orch := a.whisperState.getOrCreateOrch("pidAssoc2")
	cleanupWhisperSession(t, "pidAssoc2")

	orch.KG.AddTriple(whisper.Triple{Subject: "工作", Predicate: "压力", Object: "大", Confidence: 0.8})
	fA := orch.FactStore.Add(whisper.MemoryFact{ID: "fIsolatedA", Subject: "远方", Summary: "远方的事"})
	fB := orch.FactStore.Add(whisper.MemoryFact{ID: "fIsolatedB", Subject: "孤立", Summary: "孤立的事"})
	orch.AssocIndex.Add(whisper.Association{
		FactIDA: fA.ID, FactIDB: fB.ID,
		AssociationType: "event_chain", Strength: 0.5,
	})

	sub, err := a.whisperState.GaeaWhisperGraphSubgraph("pidAssoc2", "工作", 1)
	if err != nil {
		t.Fatalf("GaeaWhisperGraphSubgraph: %v", err)
	}
	for _, e := range sub.Edges {
		if e.Type == "因果" {
			t.Fatalf("不连通关联不应入子图: %+v", sub.Edges)
		}
	}
}
