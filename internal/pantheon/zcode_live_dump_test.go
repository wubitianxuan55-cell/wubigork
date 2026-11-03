package pantheon

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLiveZCodeStreamDump 复现实弹空结果(2026-11-03):走 gaea 完整链路
// (adapter.build + RunStreamWithTools),落盘 gaea 眼里的完整 stdout,
// 对比旁路 delta 计数与 parseAgentOutput 结果。opt-in。
func TestLiveZCodeStreamDump(t *testing.T) {
	if os.Getenv("PANTHEON_LIVE_SMOKE") == "" {
		t.Skip("set PANTHEON_LIVE_SMOKE=1")
	}
	adapters := buildAdapters(nil, nil, os.Getenv("DEEPSEEK_API_KEY"))
	adapter := adapters["zcode"]
	probeCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	status := adapter.probe(probeCtx)
	cancel()
	if !status.Available {
		t.Skipf("zcode unavailable: %s", status.Reason)
	}

	workDir := os.Getenv("PANTHEON_LIVE_WORKDIR")
	if workDir == "" {
		workDir = `C:\AI\wubigrok`
	}
	// 实弹同款完整开场 prompt(zcode 第 3 位/4 席,35 万行仓库)。
	req := StartRequest{
		Topic:     "前端测试策略选型:Vitest 与 Jest 在 35 万行桌面应用仓库的取舍",
		Workspace: workDir,
		AgentIDs:  []string{"grok", "codex", "zcode", "dsh"},
	}
	prompt := openingPrompt(req, AgentStatus{ID: "zcode", Name: "ZCode"}, nil, 3, 4, nil)

	spec, err := adapter.build(prompt, workDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, cancelRun := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancelRun()

	var deltas []string
	var toolCalls int
	start := time.Now()
	raw, err := (osCommandRunner{}).RunStreamWithTools(runCtx, spec,
		func(chunk string) { deltas = append(deltas, chunk) },
		func(trace ToolTrace) { toolCalls++ },
	)
	elapsed := time.Since(start)

	dumpDir := filepath.Join(os.TempDir(), "zcode-live-dump")
	_ = os.MkdirAll(dumpDir, 0o755)
	rawPath := filepath.Join(dumpDir, "raw.jsonl")
	_ = os.WriteFile(rawPath, []byte(raw), 0o644)

	t.Logf("elapsed=%s err=%v toolCalls=%d deltaCount=%d rawBytes=%d rawPath=%s",
		elapsed.Round(time.Second), err, toolCalls, len(deltas), len(raw), rawPath)
	deltaTotal := 0
	for _, d := range deltas {
		deltaTotal += len(d)
	}
	t.Logf("deltaTotalRunes=%d", len([]rune(strings.Join(deltas, ""))))
	parsed := parseAgentOutput(raw)
	t.Logf("parseAgentOutput runes=%d head=%q", len([]rune(parsed)), trimRunes(parsed, 80))
	if len([]rune(parsed)) < 50 {
		t.Fatalf("EMPTY RESULT reproduced: parsed=%d runes, see %s", len([]rune(parsed)), rawPath)
	}
}
