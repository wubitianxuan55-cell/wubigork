package pantheon

import (
	"os"
	"strings"
	"testing"
)

// 2026-11-03 五刀(实弹 pan_1793681319746 病例驱动)验收基线：
//  1. 插话段只在真有用户插话时注入（治 r5~r8 四席复读「插话仍只有常驻规则」）；
//  2. 主席每轮收尾出小结（真分歧/裁决/下轮必答），全文分发为下一轮议程；
//  3. 共识判据=核心方向无阻塞分歧，遗留台账不阻塞；
//  4. 汇总提炼式禁流水账；
//  5. 转述纪律（贡献 ID+短原话+三级证据标注）。

func TestUserNoteInstructionOnlyWhenRealNote(t *testing.T) {
	plain := discussionPrompt(StartRequest{Topic: "无插话"}, AgentStatus{ID: "beta", Name: "BETA"},
		[]Contribution{{ID: "c1", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: "观点"}}, 2, 1, 2, nil, nil)
	if strings.Contains(plain, "【用户插话】") {
		t.Fatalf("no real note must not inject the block:\n%s", plain)
	}
	withNote := discussionPrompt(StartRequest{Topic: "有插话"}, AgentStatus{ID: "beta", Name: "BETA"},
		[]Contribution{
			{ID: "c1", Round: 1, AgentID: "alpha", AgentName: "ALPHA", Content: "观点"},
			{ID: "u1", Phase: PhaseUserNote, Round: 1, AgentID: "user", AgentName: "用户", Content: "聚焦成本"},
		}, 2, 1, 2, nil, nil)
	if !strings.Contains(withNote, "【用户插话】") || !strings.Contains(withNote, "聚焦成本") {
		t.Fatalf("real note must inject the block and dispatch in full:\n%s", withNote)
	}
}

func TestModerationRunsBetweenRounds(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "小结机制", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitForStatus(t, m, session.ID, StatusCompleted)
	var moderations []Contribution
	for _, c := range done.Contributions {
		if c.Phase == PhaseModeration {
			moderations = append(moderations, c)
		}
	}
	// 3 轮：r1、r2 后各一次小结（末轮不做）。
	if len(moderations) != 2 {
		t.Fatalf("moderation contributions = %d, want 2:\n%+v", len(moderations), done.Contributions)
	}
	// 小结作为下一轮议程分发：r2 的席位 prompt 必须含 r1 小结全文标记。
	sawAgenda := false
	for _, call := range runner.calls {
		if strings.Contains(call.Stdin, "主席轮间小结 · 第 1 轮收尾") {
			sawAgenda = true
		}
	}
	if !sawAgenda {
		t.Fatal("round-2 prompts must carry round-1 moderation as agenda")
	}
}

func TestConsensusInstructionCarriesLedger(t *testing.T) {
	prompt := discussionPrompt(StartRequest{Topic: "台账", UntilConsensus: true},
		AgentStatus{ID: "beta", Name: "BETA"}, nil, 2, 1, 2, nil, nil)
	for _, want := range []string{"遗留台账", "不是分歧，是台账", "共识状态：一致"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("consensus instruction misses %q", want)
		}
	}
}

func TestSynthesisDemandsDistillation(t *testing.T) {
	prompt := synthesisPrompt(StartRequest{Topic: "提炼"}, AgentStatus{Name: "Grok"}, nil, nil, nil)
	for _, want := range []string{"禁止流水账", "核心结论最多 3 条", "被淘汰的方案", "验证路径", "偏航"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("synthesis misses %q", want)
		}
	}
}

func TestDiscussionPromptCarriesAntiDistortionRules(t *testing.T) {
	prompt := discussionPrompt(StartRequest{Topic: "转述"}, AgentStatus{ID: "beta", Name: "BETA"}, nil, 2, 1, 2, nil, nil)
	for _, want := range []string{"转述纪律", "贡献 ID", "【工具核验】", "【未核验】"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("anti-distortion rules miss %q", want)
		}
	}
	if !strings.Contains(collaborationRules(), "质量阶梯") {
		t.Fatal("quality ladder missing from collaboration rules")
	}
}

// 终止也生成草稿（2026-11-03 用户拍板）：已发表论证不随终止丢弃。
func TestCancelledSessionKeepsDraftResult(t *testing.T) {
	runner := consensusRunner("仍有分歧")
	runner.outputs["beta"] = func(commandSpec) (string, error) {
		return "", nil // 模拟一席空转
	}
	m := newTestManager(t, runner, fakeAdapter("alpha"), fakeAdapter("beta"))
	defer m.Close()
	session, err := m.Start(StartRequest{
		Topic: "终止草稿", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, session.ID, StatusCompleted)
	cancelled, err := m.Start(StartRequest{
		Topic: "终止草稿2", AgentIDs: []string{"alpha", "beta"}, ChairmanAgentID: "alpha",
		Rounds: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	m.Cancel(cancelled.ID)
	done := waitForStatus(t, m, cancelled.ID, StatusCancelled)
	if done.ResultPath == "" {
		t.Fatal("cancelled session must keep a draft resultPath")
	}
	path, err := m.ResultPath(cancelled.ID)
	if err != nil {
		t.Fatalf("draft result must be openable: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "未收口草稿") {
		t.Fatalf("draft header missing:\n%.200s", data)
	}
}
