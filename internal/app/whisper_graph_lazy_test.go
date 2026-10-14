package app

import (
	"testing"

	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/whisper"
	whisperdb "github.com/gaea/gaea/internal/whisper/db"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 复走 2026-10-14（v4.479.1 真机壳 + CDP 9333 沙箱）实证的残留缺口回归钉：
// GaeaWhisperGraphSubgraph 此前只读已装载会话表——fresh boot 下本 run 未开过
// 轻语聊天，图谱面板直查恒空（批 50 修 json 键名后的另一半空图根因，真机
// 「用户」实体 29 条边在库仍返回 0 节点）。修后首查懒装载 hermes.db，与
// 因果解释/先开口两兄弟（getOrCreateOrch）语义一致。
func TestWhisperGraphSubgraphLazyRestore(t *testing.T) {
	root := t.TempDir()
	if _, err := whisperdb.GetDatabase(root); err != nil {
		t.Fatalf("whisper db unavailable: %v", err)
	}
	t.Cleanup(func() { _ = whisperdb.CloseDatabase(root) })

	const sessionID = "whisper_gaea"
	// whisperSessions 是包级表：先清可能被前序测试遗留的 whisper_gaea 陈旧
	// 会话（其 dataRoot 指向别家 temp 库，懒装载会拿到空图假红），测试后再清
	// 自己的，双向防串。
	delete(whisperSessions, sessionID)
	t.Cleanup(func() { delete(whisperSessions, sessionID) })

	if err := repos.InsertFact(root, whisper.MemoryFact{
		ID: "fact_lazy_graph_1", Domain: "偏好", Subject: "用户",
		Summary: "用户喜欢安静的音乐", Status: "active", SourceSessionID: sessionID,
	}); err != nil {
		t.Fatalf("seed fact: %v", err)
	}
	if err := repos.ReplaceTriplesInDB(root, []whisper.Triple{{
		ID: "kg_lazy_graph_1", Subject: "用户", Predicate: "likes", Object: "安静的音乐",
		Confidence: 0.9, SourceFactIDs: []string{"fact_lazy_graph_1"},
	}}); err != nil {
		t.Fatalf("seed triples: %v", err)
	}

	a := &App{core: &core{cfg: &config.Config{}}}
	a.whisperState = &whisperState{
		core: a.core, app: a, whisperDataRoot: root,
		assistantMgr: assistant.NewEmpty(t.TempDir()),
	}

	sub, err := a.GaeaWhisperGraphSubgraph("gaea", "用户", 1)
	if err != nil {
		t.Fatalf("查询报错: %v", err)
	}
	if len(sub.Nodes) == 0 || len(sub.Edges) == 0 {
		t.Fatalf("懒装载后应出数据: nodes=%d edges=%d", len(sub.Nodes), len(sub.Edges))
	}
	foundCenter, foundEdge := false, false
	for _, n := range sub.Nodes {
		if n.ID == "用户" {
			foundCenter = true
		}
	}
	for _, e := range sub.Edges {
		if e.From == "用户" && e.Type == "likes" && e.To == "安静的音乐" {
			foundEdge = true
		}
	}
	if !foundCenter {
		t.Fatalf("应含中心节点 用户: %+v", sub.Nodes)
	}
	if !foundEdge {
		t.Fatalf("应含边 用户-likes→安静的音乐: %+v", sub.Edges)
	}

	// 不存在实体：空子图且非 nil（面板 ?? [] 消费的契约口径）
	miss, err := a.GaeaWhisperGraphSubgraph("gaea", "不存在的实体", 1)
	if err != nil || miss.Nodes == nil || miss.Edges == nil {
		t.Fatalf("不存在实体应返回非 nil 空子图: err=%v nodes=%v edges=%v", err, miss.Nodes, miss.Edges)
	}
}
