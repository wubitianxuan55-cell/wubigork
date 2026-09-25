package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

// sin_insight_test.go — 原罪看板数据面（SinTrajectory/SinContextView/
// SinContextNodeDetail）折叠契约：落库消息 + extra 工具轨迹 → 与办公同构
// 的 Trajectory/ContextTimeline；估算口径诚实标注（Estimated、Window=0）。

// seedSinInsight 铺一个两轮故事：轮1 纯文本；轮2 带工具轨迹与插图回写。
func seedSinInsight(t *testing.T) (*App, string) {
	t.Helper()
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("折叠测试")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tools, _ := json.Marshal([]sinToolTrace{
		{ID: "c1", Name: "sin_notes", Args: `{"action":"write"}`, Output: "已记录便签", ElapsedMS: 5},
		{ID: "c2", Name: "web_search", Args: `{"query":"x"}`, Output: "结果若干", Error: "超时", ElapsedMS: 1200},
	})
	seed := []struct {
		role    string
		content string
		extra   string
	}{
		{"user", "写一个开场", ""},
		{"assistant", "夜色落下来。", `{"reasoning":"先铺环境"}`},
		{"user", "继续，查一下背景", ""},
		{"assistant", "她查到了旧档案。", `{"reasoning":"补背景","tools":` + string(tools) + `,"illustrations":{"0":"a.png"}}`},
	}
	for _, m := range seed {
		if _, err := a.chatStore.AppendMessage(topic.ID, m.role, m.content, m.extra); err != nil {
			t.Fatalf("AppendMessage(%s): %v", m.role, err)
		}
	}
	return a, topic.ID
}

func TestSinTrajectoryFold(t *testing.T) {
	a, id := seedSinInsight(t)
	tr, err := a.SinTrajectory(id)
	if err != nil {
		t.Fatalf("SinTrajectory: %v", err)
	}
	if !tr.Ok || len(tr.Turns) != 2 {
		t.Fatalf("turns = %d, want 2（两轮 user 各起一轮）", len(tr.Turns))
	}
	t1 := tr.Turns[0]
	if len(t1.Records) != 2 || t1.Records[0].Kind != "user" || t1.Records[1].Kind != "assistant" {
		t.Fatalf("轮1记录 = %+v, want [user assistant]", t1.Records)
	}
	if t1.Records[1].Assistant == nil || t1.Records[1].Assistant.Reasoning != "先铺环境" {
		t.Fatalf("轮1推理未还原: %+v", t1.Records[1].Assistant)
	}
	t2 := tr.Turns[1]
	// 轮2：user + 2 工具 + assistant = 4 条；工具在回复前（真实时序）。
	if len(t2.Records) != 4 {
		t.Fatalf("轮2记录 = %d 条, want 4", len(t2.Records))
	}
	if t2.Records[1].Tool == nil || t2.Records[1].Tool.Name != "sin_notes" || t2.Records[1].Tool.Status != "ok" {
		t.Fatalf("工具1未还原: %+v", t2.Records[1].Tool)
	}
	if t2.Records[2].Tool == nil || t2.Records[2].Tool.Status != "error" || t2.Records[2].Tool.Err != "超时" {
		t.Fatalf("工具2错误态未还原: %+v", t2.Records[2].Tool)
	}
	if t2.End == nil || t2.End.Ts <= 0 {
		t.Errorf("轮2 未闭合: End=%+v", t2.End)
	}
	// 同秒落库（chat.store 秒级时间戳）时耗时诚实为 0：仅当 assistant 晚于
	// user 至少 1 秒才非零——不伪造精度。
	if t2.End != nil && t2.End.Ts > t2.StartedAt && t2.DurationMs <= 0 {
		t.Errorf("assistant 晚于 user 但耗时 = %d", t2.DurationMs)
	}
}

func TestSinContextViewEstimate(t *testing.T) {
	a, id := seedSinInsight(t)
	tl, err := a.SinContextView(id)
	if err != nil {
		t.Fatalf("SinContextView: %v", err)
	}
	if len(tl.Requests) != 2 {
		t.Fatalf("requests = %d, want 2（每轮一根柱）", len(tl.Requests))
	}
	for _, r := range tl.Requests {
		if !r.Estimated {
			t.Errorf("seq=%d 请求未标 Estimated（sin 无 usage，必须诚实标注）", r.Seq)
		}
		if r.Category.System <= 0 {
			t.Errorf("seq=%d system 估算缺失", r.Seq)
		}
	}
	// 累计语义：轮2 各类 ≥ 轮1。
	r1, r2 := tl.Requests[0].Category, tl.Requests[1].Category
	if r2.User < r1.User || r2.Assistant < r1.Assistant || r2.Tool < r1.Tool {
		t.Fatalf("请求构成非累计: %+v vs %+v", r1, r2)
	}
	if r2.Tool <= r1.Tool {
		t.Fatalf("轮2工具输出未入账: tool %d → %d", r1.Tool, r2.Tool)
	}
	// current = 最后一根柱；窗口未知=0；files/events 恒空。
	if tl.Current != r2 {
		t.Fatalf("current ≠ 最后一根柱")
	}
	if tl.Window != 0 {
		t.Errorf("window = %d, want 0（前端显示「—」不伪造 0%%）", tl.Window)
	}
	if len(tl.Files) != 0 || len(tl.Events) != 0 {
		t.Errorf("sin 无文件活动/事件: files=%d events=%d", len(tl.Files), len(tl.Events))
	}
	if tl.Stats.Turns != 2 || tl.Stats.ToolCalls != 2 || tl.Stats.Images != 1 {
		t.Fatalf("stats = %+v, want turns=2 toolCalls=2 images=1", tl.Stats)
	}
	// 工具耗时排行（timing.tools）与 delta 锚点。
	if tl.Timing == nil || len(tl.Timing.Tools) != 2 || tl.Timing.Tools[0].Name != "web_search" {
		t.Fatalf("timing 排行 = %+v, want web_search（1200ms）居首", tl.Timing)
	}
	last := tl.Requests[1]
	if last.Delta == nil || last.Delta.First {
		t.Fatalf("轮2 delta 缺失或误标 first")
	}
	if last.BriefUserSeq == 0 || last.BriefRespSeq == 0 {
		t.Errorf("brief 锚点缺失: user=%d resp=%d", last.BriefUserSeq, last.BriefRespSeq)
	}
}

func TestSinContextNodeDetail(t *testing.T) {
	a, id := seedSinInsight(t)
	tl, err := a.SinContextView(id)
	if err != nil {
		t.Fatalf("SinContextView: %v", err)
	}
	// user / assistant / tool 三类节点详情；system/注入不可展开。
	seen := map[string]bool{}
	for _, n := range tl.Nodes {
		d, err := a.SinContextNodeDetail(id, n.Seq)
		if n.Cat == "system" || n.Cat == "inject" {
			if err == nil {
				t.Errorf("%s 节点不应可展开（与办公同口径）", n.Cat)
			}
			continue
		}
		if err != nil {
			t.Fatalf("节点 seq=%d(%s) 详情: %v", n.Seq, n.Cat, err)
		}
		if d.Seq != n.Seq {
			t.Errorf("详情 seq=%d, want %d", d.Seq, n.Seq)
		}
		seen[d.Kind] = true
		if d.Kind == "tool_result" && d.Tool == "" {
			t.Errorf("tool_result 缺工具名")
		}
	}
	if !seen["user_message"] || !seen["assistant_message"] || !seen["tool_result"] {
		t.Fatalf("详情 kind 覆盖不全: %v", seen)
	}
	// 全文断言：user 节点正文=消息原文。
	d, err := a.SinContextNodeDetail(id, sinNodeSeq(3, 0))
	if err != nil || d.Text != "继续，查一下背景" {
		t.Fatalf("user 全文 = %q err=%v", d.Text, err)
	}
	// 未知 seq 诚实报错。
	if _, err := a.SinContextNodeDetail(id, 99999); err == nil {
		t.Error("未知 seq 应报错")
	}
}

// TestSinInsightEmptyTopic 空故事：空快照（ok=true）不报错；非法话题报错。
func TestSinInsightEmptyTopic(t *testing.T) {
	a := newSinTestApp(t)
	topic, err := a.SinTopicCreate("")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tr, err := a.SinTrajectory(topic.ID)
	if err != nil || !tr.Ok || len(tr.Turns) != 0 {
		t.Fatalf("空故事轨迹 = %+v err=%v, want 空快照", tr, err)
	}
	tl, err := a.SinContextView(topic.ID)
	if err != nil || !tl.Ok || len(tl.Requests) != 0 {
		t.Fatalf("空故事上下文 = %+v err=%v, want 空快照", tl, err)
	}
	if _, err := a.SinContextView("chat_not_exists"); err == nil {
		t.Error("非 sin 话题应被守卫拒绝")
	}
}

// 编译期口径哨兵：估算/常量与注释承诺一致（漂移即编译失败）。
var (
	_ = contextview.EmptyTimeline
	_ = trajectory.EmptyTrajectory
	_ = strings.TrimSpace
)
