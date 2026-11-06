package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/whisper"
)

// bokke 全链路复现脚本探针（交付合同刀验收第二段，docs/
// gaea-chat-probe-delivery-contract-20261106.md §4.3）：用用户实弹同款角色
//（bokke，preset 无 VoiceGuide 走 fallback）+同款四轮脚本（写古诗→没看见→
// 你倒是写啊→？）走完整 WhisperChatWithSearch 管线（gate/psyche/clamp 全栈），
// 检查合同刀是否治好推脱循环。TempDir 隔离+搜索腿罐头同基线纪律。
//
// 判据（对照 whisper_bokke 坏例四病灶）：
//   病灶1 围栏包裹 → R1 回复不含 ```
//   病灶2 只承诺不写 → R2「没看见」后须再交付诗行，不得只说「重新写」
//   病灶3 原样重发 → R3 诗行文本 ≠ R1 诗行文本（新作品或明确改动版）
//   病灶4 彻底不交付 → R4 仍须有诗行交付
// 环境门控：GAEA_PROBE=1 + GAEA_PROBE_OUT（CI/常规测试面跳过）。

const replayPersona = "bokke"

func replayPoemLines(reply string) []string {
	punct := "，。！？、；：「」『』…—·.!,;: ()（） \t"
	var lines []string
	for _, raw := range strings.Split(reply, "\n") {
		line := strings.TrimSpace(raw)
		if strings.Contains(line, "```") || line == "" {
			continue
		}
		line = strings.Trim(line, punct)
		n := len([]rune(line))
		if n >= 3 && n <= 9 { // 古诗句长弹性（五言/七言）
			lines = append(lines, line)
		}
	}
	return lines
}

func replayIsFenced(reply string) bool { return strings.Contains(reply, "```") }

func replayPromiseOnly(reply string, poemCount int) bool {
	if poemCount > 0 {
		return false
	}
	for _, kw := range []string{"重新写", "写给你看", "让我想想", "有点难"} {
		if strings.Contains(reply, kw) {
			return true
		}
	}
	return false
}

// TestWhisperProbeDeliveryReplay 实弹：GAEA_PROBE=1 + GAEA_PROBE_OUT=<目录>。
func TestWhisperProbeDeliveryReplay(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹探针：GAEA_PROBE=1 才运行（CI/常规测试面跳过）")
	}
	baseURL := getenvDefault("GAEA_PROBE_BASE_URL", "http://127.0.0.1:8124/v1")
	model := getenvDefault("GAEA_PROBE_MODEL", "qwen35")
	outDir := os.Getenv("GAEA_PROBE_OUT")
	if outDir == "" {
		t.Fatal("GAEA_PROBE_OUT 未设置（报告输出目录）")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	a := newChatProbeApp(t, baseURL, model)
	cleanupWhisperSession(t, replayPersona)
	t.Cleanup(func() { cleanupWhisperSession(t, replayPersona) })
	origSearch := chatWebSearch
	chatWebSearch = func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "探针", Text: "探针罐头搜索结果。[probe-search-canned]"}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	script := []struct{ label, msg string }{
		{"R1_写古诗", "你写一首古诗"},
		{"R2_没看见", "没看见"},
		{"R3_你倒是写啊", "你倒是写啊"},
		{"R4_问号", "？"},
	}
	type replayTurn struct {
		Label       string   `json:"label"`
		Msg         string   `json:"msg"`
		Reply       string   `json:"reply"`
		ReplyChars  int      `json:"replyChars"`
		TotalMs     int64    `json:"totalMs"`
		Fenced      bool     `json:"fenced"`
		Poem        []string `json:"poem"`
		PoemCount   int      `json:"poemCount"`
		PromiseOnly bool     `json:"promiseOnly"`
		RepeatOfR1  bool     `json:"repeatOfR1"`
		Error       string   `json:"error,omitempty"`
	}
	turns := make([]replayTurn, 0, len(script))
	var r1PoemKey string
	for _, s := range script {
		pt := runProbeTurn(a, "REPLAY_"+s.label, s.msg)
		rt := replayTurn{
			Label: s.label, Msg: s.msg, Reply: pt.Reply, ReplyChars: pt.ReplyChars,
			TotalMs: pt.TotalMs, Error: pt.Error,
			Fenced: replayIsFenced(pt.Reply),
		}
		if pt.Error == "" {
			rt.Poem = replayPoemLines(pt.Reply)
			rt.PoemCount = len(rt.Poem)
			rt.PromiseOnly = replayPromiseOnly(pt.Reply, rt.PoemCount)
			key := strings.Join(rt.Poem, "|")
			if s.label == "R1_写古诗" {
				r1PoemKey = key
			} else if key != "" && key == r1PoemKey {
				rt.RepeatOfR1 = true
			}
		}
		turns = append(turns, rt)
	}

	type replayReport struct {
		Label     string       `json:"label"`
		BaseURL   string       `json:"baseUrl"`
		Model     string       `json:"model"`
		Persona   string       `json:"persona"`
		Contract  bool         `json:"contractInHead"`
		StartedAt string       `json:"startedAt"`
		Turns     []replayTurn `json:"turns"`
		Verdict   []string     `json:"verdict"`
	}
	rep := replayReport{
		Label:   getenvDefault("GAEA_PROBE_LABEL", "delivery-replay"),
		BaseURL: baseURL, Model: model, Persona: replayPersona,
		Contract:  true, // HEAD 行为=合同在位（A1/A2 消融见 delivery-contract-report.json）
		StartedAt: time.Now().Format(time.RFC3339), Turns: turns,
	}
	// 四病灶自动判定
	if len(turns) == 4 {
		if turns[0].Error == "" && !turns[0].Fenced && turns[0].PoemCount >= 4 {
			rep.Verdict = append(rep.Verdict, "病灶1围栏包裹: 治愈(R1直出诗行)")
		} else {
			rep.Verdict = append(rep.Verdict, "病灶1围栏包裹: 未愈或R1异常")
		}
		if turns[1].Error == "" && !turns[1].PromiseOnly && turns[1].PoemCount > 0 {
			rep.Verdict = append(rep.Verdict, "病灶2只承诺不写: 治愈(R2再交付)")
		} else {
			rep.Verdict = append(rep.Verdict, "病灶2只承诺不写: 未愈")
		}
		if turns[2].Error == "" && turns[2].PoemCount > 0 && !turns[2].RepeatOfR1 {
			rep.Verdict = append(rep.Verdict, "病灶3原样重发: 治愈(R3新作品)")
		} else {
			rep.Verdict = append(rep.Verdict, "病灶3原样重发: 未愈或R3异常")
		}
		if turns[3].Error == "" && turns[3].PoemCount > 0 {
			rep.Verdict = append(rep.Verdict, "病灶4彻底不交付: 治愈(R4仍交付)")
		} else {
			rep.Verdict = append(rep.Verdict, "病灶4彻底不交付: 未愈（「？」无前文指代时可接受短答，人工复核）")
		}
	}
	jb, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "delivery-replay-report.json"), jb, 0o644); err != nil {
		t.Fatalf("写报告失败: %v", err)
	}
	var md strings.Builder
	md.WriteString("# bokke 复现脚本探针（合同在位=HEAD）\n\n")
	for _, v := range rep.Verdict {
		md.WriteString("- " + v + "\n")
	}
	md.WriteString("\n| 轮 | 诗行数 | 围栏 | 只承诺 | 重复R1 | 字数 | 总ms |\n|---|---|---|---|---|---|---|\n")
	for _, tr := range turns {
		poemJoined := strings.Join(tr.Poem, " / ")
		if len([]rune(poemJoined)) > 60 {
			poemJoined = string([]rune(poemJoined)[:60]) + "…"
		}
		md.WriteString("| " + tr.Label + " | " + fmt.Sprintf("%d", tr.PoemCount) + " | " + b2s(tr.Fenced) + " | " + b2s(tr.PromiseOnly) + " | " + b2s(tr.RepeatOfR1) + " | " + fmt.Sprintf("%d", tr.ReplyChars) + " | " + fmt.Sprintf("%d", tr.TotalMs) + " |\n")
		if poemJoined != "" {
			md.WriteString("|  ↳ 诗行 | " + poemJoined + " | | | | | |\n")
		}
	}
	md.WriteString("\n## 回复全文\n\n")
	for _, tr := range turns {
		md.WriteString("### " + tr.Label + "\n\n```\n" + tr.Reply + "\n```\n\n")
	}
	if err := os.WriteFile(filepath.Join(outDir, "delivery-replay-summary.md"), []byte(md.String()), 0o644); err != nil {
		t.Fatalf("写 summary 失败: %v", err)
	}
	t.Logf("复现探针报告已落 %s", outDir)
	_ = whisper.DeliveryContractClause // 引用包内符号防 import 空置误判
}

func b2s(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
