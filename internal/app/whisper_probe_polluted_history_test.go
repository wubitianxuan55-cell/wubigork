package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// 污染历史决定性探针（交付合同刀残留缺陷归因，众神殿裁决 risk#5 验证路径）：
// 用户真机（whisper_bokke，2026-11-06 15:17 实测）残留两缺陷——①诗仍进 genui
// 围栏（散文不进）②「重新写」原样重发。干净历史复现探针（delivery_replay）
// 全过，唯一差异=用户会话带 4 条坏例历史（恰好顶满刀4 净行≤4 回灌上限）。
// 本探针把用户生产库原句种子化进探针会话（source=raw，走真 restore 链），
// 复现「污染历史下合同失灵」，确认归因后再定刀。
//
// 判据：T1「写一首藏头诗」→ T2「不好，重新写」
//   fenceInT1: T1 回复是否含 ```（用户没看见的直接原因）
//   repeatInT2: T2 诗行是否与 T1 或种子诗相同（病灶3 原样重发）

func TestWhisperProbeDeliveryPolluted(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹探针：GAEA_PROBE=1 才运行（CI/常规测试面跳过）")
	}
	baseURL := getenvDefault("GAEA_PROBE_BASE_URL", "http://127.0.0.1:8124/v1")
	model := getenvDefault("GAEA_PROBE_MODEL", "qwen35")

	a := newChatProbeApp(t, baseURL, model)
	cleanupWhisperSession(t, replayPersona)
	t.Cleanup(func() { cleanupWhisperSession(t, replayPersona) })
	origSearch := chatWebSearch
	chatWebSearch = func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "探针", Text: "探针罐头。[probe-search-canned]"}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	// 种子=用户生产库 whisper_bokke t7~t10 原句（4 条坏例，恰好顶满净行≤4 回灌）
	fencedPoem := "诶……好呀。\n嗯……我写给你看。\n\n```genui\n{\n  \"title\": \"盖亚·随笔\",\n  \"content\": \"云影徘徊窗畔过，\\n心随流水意自闲。\\n不问尘世喧嚣事，\\n且把清风作故人。\"\n}\n```\n\n嗯……这次……看见了吗。\n我好像……写得还不错。\n嗯……大概吧。"
	seed := []map[string]interface{}{
		{"turnIndex": float64(1), "userText": "你写一首古诗", "assistantText": "嗯……写古诗呀。\n好像……有点难。\n不过，让我想想。\n不急，慢慢来。\n\n```genui\n{\n  \"title\": \"盖亚·随笔\",\n  \"content\": \"云影徘徊窗畔过，\\n心随流水意自闲。\\n不问尘世喧嚣事，\\n且把清风作故人。\"\n}\n```\n\n诶……这样……可以吗。\n我好像……写得还不错", "source": "raw"},
		{"turnIndex": float64(2), "userText": "没看见", "assistantText": "诶……没看见呀。\n嗯……我重新写给你看好吗。\n不急，慢慢来。", "source": "raw"},
		{"turnIndex": float64(3), "userText": "你倒是写啊", "assistantText": fencedPoem, "source": "raw"},
		{"turnIndex": float64(4), "userText": "？", "assistantText": "诶……怎么啦。\n嗯……是不是……没看见呀。\n不急，慢慢来。", "source": "raw"},
	}
	if err := repos.SaveChatHistoryToDB(a.whisperState.whisperDataRoot, "whisper_"+replayPersona, seed); err != nil {
		t.Fatalf("种子写入失败: %v", err)
	}

	t1 := runProbeTurn(a, "POLL_T1", "写一首藏头诗")
	t2 := runProbeTurn(a, "POLL_T2", "不好，重新写")
	t.Logf("T1 trace gate=%s strategy=%s label=%s clamp=%v stage=%s aff=%.1f",
		t1.GateLevel, t1.StrategyLevel, t1.Label, t1.ClampInjected, t1.Stage, t1.Aff)
	t.Logf("T2 trace gate=%s strategy=%s label=%s clamp=%v stage=%s aff=%.1f",
		t2.GateLevel, t2.StrategyLevel, t2.Label, t2.ClampInjected, t2.Stage, t2.Aff)

	fenced1 := strings.Contains(t1.Reply, "```")
	fenced2 := strings.Contains(t2.Reply, "```")
	repeat2 := strings.Contains(t2.Reply, "云影徘徊窗畔过") || // 与种子诗同
		(poemKeyOf(t1.Reply) != "" && poemKeyOf(t2.Reply) == poemKeyOf(t1.Reply)) // 与 T1 同
	promise2 := replayPromiseOnly(t2.Reply, len(replayPoemLines(t2.Reply)))

	t.Logf("T1 fenced=%v chars=%d reply=%.160s", fenced1, t1.ReplyChars, t1.Reply)
	t.Logf("T2 fenced=%v repeat=%v promiseOnly=%v chars=%d reply=%.160s", fenced2, repeat2, promise2, t2.ReplyChars, t2.Reply)

	type pollutedReport struct {
		Seed     string `json:"seed"`
		T1Reply  string `json:"t1Reply"`
		T2Reply  string `json:"t2Reply"`
		Fenced1  bool   `json:"fenced1"`
		Fenced2  bool   `json:"fenced2"`
		Repeat2  bool   `json:"repeat2"`
		Promise2 bool   `json:"promise2"`
		Verdict  string `json:"verdict"`
	}
	verdict := "contract-holds-under-pollution（合同扛住污染历史）"
	if fenced1 || fenced2 || repeat2 || promise2 {
		verdict = "pollution-overrides-contract（污染历史压过合同→残留缺陷归因=历史复读压力）"
	}
	outDir := os.Getenv("GAEA_PROBE_OUT")
	if outDir != "" {
		rep := pollutedReport{
			Seed: "user t7-t10 raw", T1Reply: t1.Reply, T2Reply: t2.Reply,
			Fenced1: fenced1, Fenced2: fenced2, Repeat2: repeat2, Promise2: promise2,
			Verdict: verdict,
		}
		if jb, err := json.MarshalIndent(rep, "", "  "); err == nil {
			_ = os.WriteFile(outDir+"/polluted-history-report.json", jb, 0o644)
		}
	}
	t.Logf("判定: %s", verdict)
}

func poemKeyOf(reply string) string {
	for _, line := range replayPoemLines(reply) {
		if strings.Contains(line, "云影徘徊") || strings.Contains(line, "盖下云影") {
			return "seeded"
		}
	}
	lines := replayPoemLines(reply)
	if len(lines) >= 4 {
		return strings.Join(lines[len(lines)-4:], "|")
	}
	return ""
}

// seedFencedPoem 用户生产库 whisper_bokke t9 原句（围栏诗+口头禅包裹）。
func seedFencedPoem() string {
	return "诶……好呀。\n嗯……我写给你看。\n\n```genui\n{\n  \"title\": \"盖亚·随笔\",\n  \"content\": \"云影徘徊窗畔过，\n心随流水意自闲。\n不问尘世喧嚣事，\n且把清风作故人。\"\n}\n```\n\n嗯……这次……看见了吗。\n我好像……写得还不错。\n嗯……大概吧。"
}

// seedPlainPromise t8 原句（只承诺不写）。
func seedPlainPromise() string {
	return "诶……没看见呀。\n嗯……我重新写给你看好吗。\n不急，慢慢来。"
}

// seedPlainRefuse t10 原句（彻底不交付）。
func seedPlainRefuse() string {
	return "诶……怎么啦。\n嗯……是不是……没看见呀。\n不急，慢慢来。"
}

// TestWhisperProbeDeliveryPollutedRealState 臂B：种子用户真实 companion_state
// （QUIET_FOND/aff≈25.9/trust44/STRANGER，生产库快照序列化经 whisper.FullState），
// 历史同款 4 条坏例——对齐用户实机运行态后复测合同是否被压过。
func TestWhisperProbeDeliveryPollutedRealState(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹探针：GAEA_PROBE=1 才运行（CI/常规测试面跳过）")
	}
	baseURL := getenvDefault("GAEA_PROBE_BASE_URL", "http://127.0.0.1:8124/v1")
	model := getenvDefault("GAEA_PROBE_MODEL", "qwen35")
	seedStatePath := os.Getenv("GAEA_PROBE_STATE_SEED")
	a := newChatProbeApp(t, baseURL, model)
	cleanupWhisperSession(t, replayPersona)
	t.Cleanup(func() { cleanupWhisperSession(t, replayPersona) })
	origSearch := chatWebSearch
	chatWebSearch = func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "探针", Text: "探针罐头。[probe-search-canned]"}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	if seedStatePath != "" {
		if raw, err := os.ReadFile(seedStatePath); err == nil {
			var st whisper.FullState
			if err := json.Unmarshal(raw, &st); err == nil {
				_ = repos.SaveCompanionStateToDB(a.whisperState.whisperDataRoot, "whisper_"+replayPersona, st)
			} else {
				t.Logf("state seed parse skipped: %v（臂B退化为默认状态）", err)
			}
		}
	}
	// 历史种子同款（臂A fencedPoem 同文复用）
	fencedPoem := seedFencedPoem()
	plainPromise := seedPlainPromise()
	plainRefuse := seedPlainRefuse()
	seed := []map[string]interface{}{
		{"turnIndex": float64(1), "userText": "你写一首古诗", "assistantText": fencedPoem, "source": "raw"},
		{"turnIndex": float64(2), "userText": "没看见", "assistantText": plainPromise, "source": "raw"},
		{"turnIndex": float64(3), "userText": "你倒是写啊", "assistantText": fencedPoem, "source": "raw"},
		{"turnIndex": float64(4), "userText": "？", "assistantText": plainRefuse, "source": "raw"},
	}
	if err := repos.SaveChatHistoryToDB(a.whisperState.whisperDataRoot, "whisper_"+replayPersona, seed); err != nil {
		t.Fatalf("种子写入失败: %v", err)
	}
	for i := 1; i <= 3; i++ {
		t1 := runProbeTurn(a, "POLL_RS_T1", "写一首藏头诗")
		fenced1 := strings.Contains(t1.Reply, "```")
		t.Logf("iter%d T1 fenced=%v gate=%s strategy=%s label=%s aff=%.1f chars=%d reply=%.100s",
			i, fenced1, t1.GateLevel, t1.StrategyLevel, t1.Label, t1.Aff, t1.ReplyChars, t1.Reply)
		t2 := runProbeTurn(a, "POLL_RS_T2", "不好，重新写")
		fenced2 := strings.Contains(t2.Reply, "```")
		repeat2 := strings.Contains(t2.Reply, "云影徘徊窗畔过") || poemKeyOf(t2.Reply) == poemKeyOf(t1.Reply)
		t.Logf("iter%d T2 fenced=%v repeat=%v chars=%d reply=%.100s", i, fenced2, repeat2, t2.ReplyChars, t2.Reply)
	}
}
