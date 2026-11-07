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

// TestLiveDSHSeatSessionReuse 是「缓存命中」两刀的实弹验收（同样 opt-in）：
// 同一席位第二次调用带 --session-id 续接，引擎必须认同一会话，且第二轮
// prompt 绝大部分来自缓存。2026-11-06 本机实测 96%~97%。
func TestLiveDSHSeatSessionReuse(t *testing.T) {
	if os.Getenv("PANTHEON_LIVE_SMOKE") == "" {
		t.Skip("set PANTHEON_LIVE_SMOKE=1 to call installed agent harnesses")
	}
	adapter := buildAdapters(nil, nil, os.Getenv("DEEPSEEK_API_KEY"))["dsh"]
	// 凭据跟着「工作目录」走：DSH headless 会读 cwd 下的 .env（本仓 .env 带
	// DEEPSEEK_API_KEY）或环境变量。临时目录没有凭据，实弹必须在真实工作区跑。
	workDir := strings.TrimSpace(os.Getenv("PANTHEON_LIVE_WORKDIR"))
	if workDir == "" {
		workDir = t.TempDir()
	}
	run := func(prompt, resumeID string) SeatUsage {
		t.Helper()
		spec, err := adapter.commandSpec(prompt, workDir, nil, resumeID)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		raw, err := (osCommandRunner{}).Run(ctx, spec)
		if strings.Contains(raw, "MISSING_CREDENTIAL") {
			t.Skipf("本机没有可用的 DeepSeek 凭据（在 %s 下跑才能读到 .env）：%v", workDir, err)
		}
		if err != nil {
			t.Fatalf("dsh run failed: %v\nraw=%s", err, raw)
		}
		return parseSeatRunStats(raw)
	}

	first := run("Reply with exactly READY and do not use tools.", "")
	if first.EngineSessionID == "" {
		t.Fatalf("首轮没有拿到引擎会话 ID：%+v", first)
	}
	second := run("Reply with exactly DONE and do not use tools.", first.EngineSessionID)
	if second.EngineSessionID != first.EngineSessionID {
		t.Fatalf("续接换了会话：first=%s second=%s", first.EngineSessionID, second.EngineSessionID)
	}
	if hit := second.CacheHitPercent(); hit < 50 {
		t.Fatalf("第二轮缓存命中率 %.1f%%，续接未生效（账目 %+v）", hit, second)
	}
	t.Logf("会话 %s：第二轮缓存命中 %.1f%%（%d/%d prompt token）",
		second.EngineSessionID, second.CacheHitPercent(),
		second.CacheReadTokens, second.CacheReadTokens+second.InputTokens)
}

// TestLiveZCodeSeatSessionReuse 是 ZCode 的同类实弹：续接 sess_… 后应只剩
// 一次模型请求、缓存绝大部分命中，且不再重新探索。2026-11-06 实测 10.7s /
// 99.2%（33856/34133）。
func TestLiveZCodeSeatSessionReuse(t *testing.T) {
	if os.Getenv("PANTHEON_LIVE_SMOKE") == "" {
		t.Skip("set PANTHEON_LIVE_SMOKE=1 to call installed agent harnesses")
	}
	adapter := buildAdapters(nil, nil, "")["zcode"]
	workDir := strings.TrimSpace(os.Getenv("PANTHEON_LIVE_WORKDIR"))
	if workDir == "" {
		workDir = t.TempDir()
	}
	run := func(prompt, resumeID string) SeatUsage {
		t.Helper()
		spec, err := adapter.commandSpec(prompt, workDir, nil, resumeID)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		raw, err := (osCommandRunner{}).Run(ctx, spec)
		if err != nil {
			t.Fatalf("zcode run failed: %v\nraw=%s", err, raw)
		}
		return parseSeatRunStats(raw)
	}

	first := run("Reply with exactly READY. Do not use any tools.", "")
	if first.EngineSessionID == "" {
		t.Fatalf("首轮没有拿到 ZCode 会话 ID：%+v", first)
	}
	second := run("Reply with exactly DONE. Do not use any tools.", first.EngineSessionID)
	if second.EngineSessionID != first.EngineSessionID {
		t.Fatalf("续接换了会话：first=%s second=%s", first.EngineSessionID, second.EngineSessionID)
	}
	if hit := second.CacheHitPercent(); hit < 50 {
		t.Fatalf("第二轮缓存命中率 %.1f%%，续接未生效（账目 %+v）", hit, second)
	}
	t.Logf("会话 %s：第二轮缓存命中 %.1f%%（%d/%d prompt token，%d 次模型请求）",
		second.EngineSessionID, second.CacheHitPercent(),
		second.CacheReadTokens, second.CacheReadTokens+second.InputTokens, second.Steps)
}
