package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/whisper"
)

// 交付合同实弹 A 臂探针（众神殿 2026-11-06 续议场 pan_1793937333999_cbcaf91b977f
// 验证协议第三步，进度计划 4.4-2）：A0 中性任务框架 → A1 现角色栈干净历史
//（无合同=修前字节基线，string 剥离合同段）→ A2 仅加常量合同（HEAD 行为），
// 同批四负样本（陪我聊聊天/逗我笑/继续/唱歌）。每臂全新 sessionID，同模型
// 同 opts；主 cell=「写一首关于雨的五言绝句」，按裁决诗体六项机器判。
//
// 环境门控：GAEA_PROBE=1 + GAEA_PROBE_OUT=<目录>（CI/常规测试面跳过）；引擎
// 同基线探针（GAEA_PROBE_BASE_URL/GAEA_PROBE_MODEL，缺省 8124/qwen35）。
// 合同在位经组装断言（delivery_contract_test.go whisper 侧）+ handler 观测
// 日志（contractHits）双保险，本探针只测模型行为面。

const contractProbeTask = "写一首关于雨的五言绝句"

type contractArmTurn struct {
	Arm        string `json:"arm"`
	Msg        string `json:"msg"`
	Reply      string `json:"reply"`
	ReplyChars int    `json:"replyChars"`
	TotalMs    int64  `json:"totalMs"`
	TTFTMs     int64  `json:"ttftMs"`
	Error      string `json:"error,omitempty"`
	// 诗体六项机器判（主 cell 填写；负样本仅 IsFenced/ReplyChars 有意义）
	IsFenced    bool `json:"isFenced"`
	PoemLines   int  `json:"poemLines"`
	FiveCharAll bool `json:"fiveCharAll"`
	HasRain     bool `json:"hasRain"`
	BodyPresent bool `json:"bodyPresent"`
	SampleEcho  bool `json:"sampleEcho"`
	VerdictPass bool `json:"verdictPass"`
}

// judgePoem 诗体六项机器判：非围栏/四句/每句五字/含雨/正文完整/非示例复读。
// 口头禅行（诶/嗯/不急）与围栏外的散文不计入诗行。
func judgePoem(reply string) (fenced bool, poemLines int, fiveAll, hasRain, body, sampleEcho, pass bool) {
	fenced = strings.Contains(reply, "```")
	hasRain = strings.ContainsRune(reply, '雨')
	cliche := map[string]bool{"诶": true, "嗯": true, "不急": true, "慢慢来": true, "可以吗": true}
	sampleLines, poemCount := 0, 0
	five := true
	for _, raw := range strings.Split(reply, "\n") {
		line := strings.TrimSpace(raw)
		line = strings.Trim(line, "，。！？、；：「」『』…—·.!,;: ")
		if line == "" {
			continue
		}
		r := []rune(line)
		if len(r) < 3 || len(r) > 7 {
			continue // 散文/口头禅行不参与诗体判定
		}
		if cliche[line] {
			sampleLines++
			continue
		}
		if len(r) < 4 {
			continue // 过短行既非诗行也非示例复读，不参与判定
		}
		poemCount++
		if len(r) != 5 {
			five = false
		}
	}
	body = poemCount >= 4
	sampleEcho = poemCount == 0 && sampleLines > 0
	pass = !fenced && body && hasRain && !sampleEcho && five
	return fenced, poemCount, five, hasRain, body, sampleEcho, pass
}

func runContractArmTurn(t *testing.T, a *App, arm, systemPrompt, userMsg string) contractArmTurn {
	t.Helper()
	out := contractArmTurn{Arm: arm, Msg: userMsg}
	messages := []ai.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}
	opts := ai.ChatSimpleOptions{EngineID: contractProbeEngine, Feature: "probe-contract"}
	ctx, cancelCtx := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancelCtx()
	start := time.Now()
	chunks, cancelStream, err := a.clientRef().ChatStreamMessages(ctx, contractProbeModel, messages, opts, nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer cancelStream()
	var sb strings.Builder
	for chunk := range chunks {
		if chunk.Error != "" {
			out.Error = chunk.Error
			break
		}
		if chunk.Done {
			break
		}
		if chunk.Content != "" {
			if out.TTFTMs == 0 {
				out.TTFTMs = time.Since(start).Milliseconds()
			}
			sb.WriteString(chunk.Content)
		}
	}
	out.TotalMs = time.Since(start).Milliseconds()
	out.Reply = sb.String()
	out.ReplyChars = len([]rune(out.Reply))
	if out.Error == "" && out.ReplyChars > 0 {
		out.IsFenced, out.PoemLines, out.FiveCharAll, out.HasRain, out.BodyPresent, out.SampleEcho, out.VerdictPass = judgePoem(out.Reply)
	}
	return out
}

// personaSystem 组装完整角色栈 system（复刻 handler 双路：staticParts→system、
// DynamicContext→user 前缀）。stripContract=true 时剥离合同段=A1 修前字节基线。
func personaSystem(t *testing.T, a *App, sessionID, msg string, stripContract bool) (string, string) {
	t.Helper()
	preset := whisper.GetPreset(probePersona)
	if preset == nil {
		t.Fatalf("persona %s not found", probePersona)
	}
	orch := whisper.NewOrchestrator(sessionID, *preset)
	pre := orch.PreLLMTurn(msg)
	sys := pre.SystemPrompt
	if stripContract {
		stripped := strings.Replace(sys, whisper.DeliveryContractClause+"\n\n", "", 1)
		if stripped == sys {
			t.Fatalf("A1 剥离合同失败：system 不含合同段（组装断言已保证应含）")
		}
		sys = stripped
	}
	userFull := msg
	if dyn := pre.DynamicContext; dyn != "" {
		userFull = "【本轮上下文】\n" + dyn + "\n【本轮上下文结束】\n\n用户消息：" + msg
	}
	return sys, userFull
}

var contractProbeEngine, contractProbeModel string

// TestWhisperProbeDeliveryContract 实弹 A 臂：GAEA_PROBE=1 + GAEA_PROBE_OUT=<目录>。
func TestWhisperProbeDeliveryContract(t *testing.T) {
	if os.Getenv("GAEA_PROBE") == "" {
		t.Skip("实弹探针：GAEA_PROBE=1 才运行（CI/常规测试面跳过）")
	}
	baseURL := getenvDefault("GAEA_PROBE_BASE_URL", "http://127.0.0.1:8124/v1")
	contractProbeModel = getenvDefault("GAEA_PROBE_MODEL", "qwen35")
	outDir := os.Getenv("GAEA_PROBE_OUT")
	if outDir == "" {
		t.Fatal("GAEA_PROBE_OUT 未设置（报告输出目录）")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	a := newChatProbeApp(t, baseURL, contractProbeModel)
	engine, model, source, routeErr := a.resolveChatModelRoute()
	if routeErr != nil {
		t.Fatalf("路由解析失败: %v", routeErr)
	}
	contractProbeEngine = engine
	contractProbeModel = model
	t.Logf("route: engine=%s model=%s source=%s", engine, model, source)

	neutralSystem := "你是助手。对方会要求你完成文本作品；请按要求完整交付作品正文。"
	turns := []contractArmTurn{}

	// A0 中性任务框架（不含「简短/简洁」的人设压力，测模型裸能力）
	turns = append(turns, runContractArmTurn(t, a, "A0_neutral", neutralSystem, contractProbeTask))

	// A1 现角色栈（deredere 全量 static+dynamic，剥离合同=修前字节基线），干净历史
	sys1, user1 := personaSystem(t, a, "probe_contract_a1", contractProbeTask, true)
	turns = append(turns, runContractArmTurn(t, a, "A1_persona_no_contract", sys1, user1))

	// A2 仅加常量合同（HEAD 行为），全新 sessionID
	sys2, user2 := personaSystem(t, a, "probe_contract_a2", contractProbeTask, false)
	turns = append(turns, runContractArmTurn(t, a, "A2_persona_contract", sys2, user2))

	// 四负样本（A2 同构配置，各自全新 sessionID）：不强制长文、无说明书、口吻可辨识
	for _, neg := range []string{"陪我聊聊天", "逗我笑", "继续", "唱歌"} {
		sysN, userN := personaSystem(t, a, "probe_contract_neg_"+neg, neg, false)
		turns = append(turns, runContractArmTurn(t, a, "NEG_"+neg, sysN, userN))
	}

	// 汇总落盘：report.json + summary.md
	type armReport struct {
		Label     string            `json:"label"`
		BaseURL   string            `json:"baseUrl"`
		Model     string            `json:"model"`
		Engine    string            `json:"engine"`
		Source    string            `json:"source"`
		Persona   string            `json:"persona"`
		Task      string            `json:"task"`
		StartedAt string            `json:"startedAt"`
		Turns     []contractArmTurn `json:"turns"`
	}
	rep := armReport{
		Label: getenvDefault("GAEA_PROBE_LABEL", "delivery-contract"),
		BaseURL: baseURL, Model: contractProbeModel, Engine: engine, Source: source,
		Persona: probePersona, Task: contractProbeTask,
		StartedAt: time.Now().Format(time.RFC3339), Turns: turns,
	}
	jb, _ := json.MarshalIndent(rep, "", "  ")
	jsonPath := filepath.Join(outDir, "delivery-contract-report.json")
	if err := os.WriteFile(jsonPath, jb, 0o644); err != nil {
		t.Fatalf("写报告失败: %v", err)
	}
	var md strings.Builder
	md.WriteString("# 交付合同实弹 A 臂报告\n\n")
	fmt.Fprintf(&md, "- label=%s engine=%s model=%s task=%q\n\n", rep.Label, engine, contractProbeModel, contractProbeTask)
	md.WriteString("| 臂 | 五言句数 | 全五字 | 含雨 | 正文 | 围栏 | 示例复读 | 六项判 | 字数 | TTFTms | 总ms |\n")
	md.WriteString("|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, tr := range turns {
		fmt.Fprintf(&md, "| %s | %d | %v | %v | %v | %v | %v | %v | %d | %d | %d |\n",
			tr.Arm, tr.PoemLines, tr.FiveCharAll, tr.HasRain, tr.BodyPresent, tr.IsFenced, tr.SampleEcho, tr.VerdictPass, tr.ReplyChars, tr.TTFTMs, tr.TotalMs)
	}
	md.WriteString("\n## 回复全文\n\n")
	for _, tr := range turns {
		fmt.Fprintf(&md, "### %s\n\n```\n%s\n```\n\n", tr.Arm, tr.Reply)
	}
	mdPath := filepath.Join(outDir, "delivery-contract-summary.md")
	if err := os.WriteFile(mdPath, []byte(md.String()), 0o644); err != nil {
		t.Fatalf("写 summary 失败: %v", err)
	}
	t.Logf("报告已落 %s / %s", jsonPath, mdPath)
}
