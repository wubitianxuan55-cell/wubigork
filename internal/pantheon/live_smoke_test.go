package pantheon

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveAgentAdapters is opt-in because it invokes the user's authenticated
// local agent harnesses and may consume plan/API quota. It sends no project data.
func TestLiveAgentAdapters(t *testing.T) {
	if os.Getenv("PANTHEON_LIVE_SMOKE") == "" {
		t.Skip("set PANTHEON_LIVE_SMOKE=1 to call installed agent harnesses")
	}
	adapters := buildAdapters(nil, nil, os.Getenv("DEEPSEEK_API_KEY"))
	only := strings.TrimSpace(os.Getenv("PANTHEON_LIVE_AGENTS"))
	workDir := strings.TrimSpace(os.Getenv("PANTHEON_LIVE_WORKDIR"))
	if workDir == "" {
		// 共享 workDir 必须绑外层 t：子测试的 TempDir 在该子测试结束即被清理，
		// 后续席位 chdir 立炸（PANTHEON_LIVE_SMOKE 实弹逮住）
		workDir = t.TempDir()
	}
	for _, id := range []string{"codex", "zcode", "dsh"} {
		if only != "" && !strings.Contains(","+only+",", ","+id+",") {
			continue
		}
		t.Run(id, func(t *testing.T) {
			adapter := adapters[id]
			probeCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			status := adapter.probe(probeCtx)
			cancel()
			if !status.Available {
				t.Fatalf("%s unavailable: %s", id, status.Reason)
			}
			if workDir == "" {
				workDir = t.TempDir()
			}
			spec, err := adapter.build("Reply with exactly READY and do not use tools.", workDir, nil)
			if err != nil {
				t.Fatal(err)
			}
			runCtx, cancelRun := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancelRun()
			raw, err := (osCommandRunner{}).Run(runCtx, spec)
			if err != nil {
				t.Fatalf("%s run failed: %v\nraw=%s", id, err, raw)
			}
			text := strings.TrimSpace(parseAgentOutput(raw))
			if text != "READY" {
				t.Fatalf("%s returned %q, raw=%s", id, text, raw)
			}
		})
	}
}
