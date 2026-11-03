package app

import (
	"testing"

	"github.com/gaea/gaea/internal/pantheon"
)

// SSE 事件面必须携带 round：前端按「席位:轮次」去重直播残影与定稿，丢 round
// 会让每个席位的发言渲染两遍（2026-10-31 实弹「上面一条下面一条」根因）。
func TestPantheonEventPayloadCarriesRound(t *testing.T) {
	payload := pantheonEventPayload(pantheon.Event{
		Seq: -7, SessionID: "pan_x", Phase: "discussion", Round: 2,
		AgentID: "grok", Kind: pantheon.EventAgentDelta, Text: "增量", At: 123,
	})
	if payload["round"] != 2 {
		t.Fatalf("SSE 事件面缺 round: %+v", payload)
	}
	if payload["sessionId"] != "pan_x" || payload["agentId"] != "grok" || payload["kind"] != pantheon.EventAgentDelta {
		t.Fatalf("SSE 事件面基础字段缺失: %+v", payload)
	}
}
