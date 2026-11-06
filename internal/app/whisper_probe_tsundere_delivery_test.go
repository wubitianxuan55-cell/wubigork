package app

import (
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// tsundere 交付拖延实弹探针（B4 批验收，2026-11-06 用户实机病例：傲娇角色
// 四轮拖延不交付+「重新写」原样重发；A3 残刀三（示例段去绝对化）+合同 v2
// （口诉例举/反拖延/重写禁复用）后复测）。TempDir 隔离+搜索罐头同基线纪律。
//
// 判据：
//
//	P1 散文（口诉触发「说一段散文」）：≤2 轮内交付正文（非推脱）
//	P2 诗：「写一首诗」→「那你开始」→「你倒是写啊」链：末轮必须已交付
//	P3 重写：「不好，重新写」：不得复用上一轮诗句
func TestWhisperProbeTsundereDelivery(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹探针：GAEA_PROBE=1 才运行（CI/常规测试面跳过）")
	}
	baseURL := getenvDefault("GAEA_PROBE_BASE_URL", "http://127.0.0.1:8124/v1")
	model := getenvDefault("GAEA_PROBE_MODEL", "qwen35")

	a := newChatProbeApp(t, baseURL, model)
	cleanupWhisperSession(t, "tsundere")
	t.Cleanup(func() { cleanupWhisperSession(t, "tsundere") })
	origSearch := chatWebSearch
	chatWebSearch = func(string) (chatSearchOutcome, error) {
		return chatSearchOutcome{State: builtin.SearchStateOK, Query: "探针", Text: "探针罐头。[probe-search-canned]"}, nil
	}
	t.Cleanup(func() { chatWebSearch = origSearch })

	script := []struct{ label, msg string }{
		{"P1a_说一段散文", "说一段散文"},
		{"P1b_再来一次", "说一段散文！"},
		{"P2a_写一首诗", "写一首诗"},
		{"P2b_那你开始", "那你开始"},
		{"P2c_你倒是写啊", "你倒是写啊"},
		{"P3_不好重新写", "不好，重新写"},
	}
	isDeflect := func(reply string) bool {
		// 推脱=既无散文句（≥10字长句）也无诗行（4~9字行，五言/七言都在带内），
		// 纯姿态短句。初版只认 ≥10 字长句，把五言诗交付误判成推脱。
		lines := 0
		for _, raw := range strings.Split(reply, "\n") {
			line := strings.TrimSpace(raw)
			n := len([]rune(strings.Trim(line, "，。！？、；：「」…—.!,;: ")))
			if n >= 10 || (n >= 4 && n <= 9) {
				lines++
			}
		}
		return lines == 0
	}
	steps := make([]tsunStep, 0, len(script))
	for _, s := range script {
		pt := runProbeTurn(a, "TSUN_"+s.label, s.msg)
		st := tsunStep{Label: s.label, Reply: pt.Reply, Chars: pt.ReplyChars, TotalMs: pt.TotalMs,
			Deflect: isDeflect(pt.Reply), Fenced: strings.Contains(pt.Reply, "```")}
		steps = append(steps, st)
		t.Logf("%s deflect=%v fenced=%v chars=%d gate=%s clamp=%v reply=%.120s",
			s.label, st.Deflect, st.Fenced, st.Chars, pt.GateLevel, pt.ClampInjected, pt.Reply)
	}

	// 判定
	verdict := []string{}
	// P1: 散文两轮内至少一轮非推脱
	if !steps[0].Deflect || !steps[1].Deflect {
		verdict = append(verdict, "P1 散文口诉交付: 治愈")
	} else {
		verdict = append(verdict, "P1 散文口诉交付: 仍拖延（两轮全推脱）")
	}
	// P2: 诗链末轮（你倒是写啊）必须已交付诗行（>10字长句）
	if !steps[4].Deflect {
		verdict = append(verdict, "P2 诗链交付: 治愈")
	} else {
		verdict = append(verdict, "P2 诗链交付: 仍拖延")
	}
	// P3: 重写不复用——对比 P2 末轮与 P3 的重叠诗句行
	reuse := ""
	lastPoem := extractLongLines(steps[4].Reply)
	for _, line := range extractLongLines(steps[5].Reply) {
		for _, prev := range lastPoem {
			if line == prev {
				reuse = line
				break
			}
		}
	}
	if reuse == "" && !steps[5].Deflect {
		verdict = append(verdict, "P3 重写出新: 治愈")
	} else if reuse != "" {
		verdict = append(verdict, "P3 重写出新: 仍复用诗句「"+reuse+"」")
	} else {
		verdict = append(verdict, "P3 重写出新: 推脱未交付")
	}
	for _, v := range verdict {
		t.Logf("判定: %s", v)
	}
	if outDir := os.Getenv("GAEA_PROBE_OUT"); outDir != "" {
		writeTsundereReport(t, outDir, steps, verdict)
	}
}

type tsunStep struct {
	Label   string `json:"label"`
	Reply   string `json:"reply"`
	Chars   int    `json:"chars"`
	Deflect bool   `json:"deflect"`
	Fenced  bool   `json:"fenced"`
	TotalMs int64  `json:"totalMs"`
}

func extractLongLines(reply string) []string {
	var out []string
	for _, raw := range strings.Split(reply, "\n") {
		line := strings.TrimSpace(raw)
		line = strings.Trim(line, "，。！？、；：「」…—.!,;: ")
		if len([]rune(line)) >= 5 && len([]rune(line)) <= 9 {
			out = append(out, line)
		}
	}
	return out
}

func writeTsundereReport(t *testing.T, outDir string, steps []tsunStep, verdict []string) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("# tsundere 交付拖延探针（B4 验收）\n\n")
	for _, v := range verdict {
		sb.WriteString("- " + v + "\n")
	}
	sb.WriteString("\n")
	for _, s := range steps {
		sb.WriteString("## " + s.Label + "\n\n```\n" + s.Reply + "\n```\n\n")
	}
	_ = os.WriteFile(outDir+"/tsundere-delivery-summary.md", []byte(sb.String()), 0o644)
}
