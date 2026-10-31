package pantheon

import (
	"strings"
	"testing"
)

// 2026-10-31 圆桌裁决第二刀验收：对照实验可表达——路径标签入档、
// 单轮放行、独立开场不注入同轮前序；用户指定顺序与首位轮换保留。

func TestNormalizeProtocol(t *testing.T) {
	// 2026-10-31 用户拍板：独立开场升为默认（首轮各抒己见，避免首位框定范围）。
	if got, err := NormalizeProtocol(""); err != nil || got != ProtocolIndependent {
		t.Fatalf("empty protocol = (%q,%v), want independent default", got, err)
	}
	if got, err := NormalizeProtocol(" council "); err != nil || got != ProtocolCouncil {
		t.Fatalf("council = (%q,%v), want council", got, err)
	}
	if got, err := NormalizeProtocol("chatroom"); err == nil {
		t.Fatalf("unknown protocol %q accepted as %q", "chatroom", got)
	}
}

func TestIndependentOpeningInjectsNoSameRoundContent(t *testing.T) {
	runner := defaultFakeRunner()
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()

	session, err := m.Start(StartRequest{
		Topic: "独立开场基线", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolIndependent, Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.Request.Protocol != ProtocolIndependent {
		t.Fatalf("session protocol = %q, want independent", done.Request.Protocol)
	}
	betaRound1, betaRound2 := 0, 0
	for _, call := range runner.calls {
		if !strings.Contains(call.Stdin, "「BETA」") {
			continue
		}
		switch {
		case strings.Contains(call.Stdin, "第 1 轮开场陈述"):
			betaRound1++
			if strings.Contains(call.Stdin, "先做只读验证") {
				t.Fatalf("independent opening leaked alpha's same-round statement:\n%s", call.Stdin)
			}
			if !strings.Contains(call.Stdin, "目前还没有已经发表的讨论记录") {
				t.Fatalf("independent opening must treat every seat as first speaker:\n%s", call.Stdin)
			}
		case strings.Contains(call.Stdin, "第 2 轮补充论证"):
			betaRound2++
			if !strings.Contains(call.Stdin, "先做只读验证") {
				t.Fatalf("round 2 must see round-1 history again:\n%s", call.Stdin)
			}
		}
	}
	if betaRound1 != 1 || betaRound2 != 1 {
		t.Fatalf("beta calls round1=%d round2=%d, want 1/1", betaRound1, betaRound2)
	}
	// 用户指定顺序与首位轮换保留：第 1 轮 alpha 先、第 2 轮 beta 先。
	firstRoundAlpha := -1
	secondRoundBeta := -1
	for i, call := range runner.calls {
		if firstRoundAlpha < 0 && strings.Contains(call.Stdin, "「ALPHA」") && strings.Contains(call.Stdin, "第 1 轮开场陈述") {
			firstRoundAlpha = i
		}
		if strings.Contains(call.Stdin, "「BETA」") && strings.Contains(call.Stdin, "第 2 轮补充论证") {
			secondRoundBeta = i
			break
		}
	}
	if firstRoundAlpha < 0 || secondRoundBeta < 0 || secondRoundBeta < firstRoundAlpha {
		t.Fatalf("user order/rotation broken: alpha first call at %d, beta round-2 call at %d", firstRoundAlpha, secondRoundBeta)
	}
}

func TestSingleRoundCouncilRuns(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "单轮对照", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolCouncil, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if done.CurrentRound != 1 {
		t.Fatalf("single round must stay single, got %d", done.CurrentRound)
	}
	if len(done.Contributions) != 3 { // 2 席 × 1 轮 + 汇总
		t.Fatalf("contributions = %d, want 3", len(done.Contributions))
	}
	if done.Request.Rounds != 1 {
		t.Fatalf("request rounds normalized to %d, want 1", done.Request.Rounds)
	}
}

func TestIndependentProtocolAllowsSingleSeat(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "单席深答", AgentIDs: []string{"alpha"},
		Protocol: ProtocolIndependent, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	if len(done.Contributions) != 2 { // 单席 × 1 轮 + 汇总
		t.Fatalf("contributions = %d, want 2", len(done.Contributions))
	}
	warned := false
	for _, warning := range done.Warnings {
		if strings.Contains(warning, "少于两个") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("single-seat session must warn about solo summary: %v", done.Warnings)
	}
}

func TestCouncilStillRequiresTwoSeats(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"))
	defer m.Close()
	if _, err := m.Start(StartRequest{
		Topic: "圆桌单席", AgentIDs: []string{"alpha"}, Protocol: ProtocolCouncil, Rounds: 1,
	}); err == nil || !strings.Contains(err.Error(), "至少需要选择两个 Agent") {
		t.Fatalf("council single seat must stay blocked, got err=%v", err)
	}
	// 默认（独立开场）下单席放行，兜住默认翻转不回退。
	if _, err := m.Start(StartRequest{
		Topic: "默认单席", AgentIDs: []string{"alpha"}, Rounds: 1,
	}); err != nil {
		t.Fatalf("default (independent) single seat must be allowed, got err=%v", err)
	}
}

func TestProtocolRecordedAndExported(t *testing.T) {
	m := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "路径入档", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolIndependent, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	export, err := m.Export(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"议事路径：独立开场", "费用未采集", "耗时仅可同席自比"} {
		if !strings.Contains(export, want) {
			t.Fatalf("export misses %q:\n%s", want, export)
		}
	}
	if done.ResultPath == "" {
		t.Fatal("completed session missing resultPath")
	}
	// 圆桌接力腿：显式 council 仍可用，标签落导出。
	council := newTestManager(t, defaultFakeRunner(), fakeAdapter("alpha"), fakeAdapter("beta"))
	defer council.Close()
	councilSession, err := council.Start(StartRequest{
		Topic: "路径入档默认", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Protocol: ProtocolCouncil, Rounds: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, council, councilSession.ID, StatusCompleted)
	councilExport, err := council.Export(councilSession.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(councilExport, "议事路径：圆桌接力") {
		t.Fatalf("council export misses default path label:\n%s", councilExport)
	}
	if councilSession.Request.Protocol != ProtocolCouncil {
		// 兼容旧调用方：请求未带协议时 Start 返回前应归一为 council。
		t.Fatalf("council protocol not normalized, got %q", councilSession.Request.Protocol)
	}
}
