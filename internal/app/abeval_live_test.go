package app

// A/B 评测线（2026-11-08 众神殿办公板块子代理裁决收尾刀7，pan_1794121254456
// 建议的最后一环）：同题双臂比较 **A+spill**（根代理直做）与 **B-readonly-task**
// （探索/取证经 readonly 子代理委派，主对话只编排成文）——裁决要求用 10–20 个
// 真实办公任务比较漏项、返工、证据完整率、墙钟与审批次数，达标后探索型子代
// 理才常开。
//
// 门控：GAEA_AB_EVAL=1 才跑（真引擎真 token、分钟级；CI/缺省恒 skip，同
// GAEA_OCR_LIVE 先例）。样本：内建 3 个种子（多表汇总/条款提取/图表报告），
// GAEA_AB_EVAL_SAMPLES=<json> 覆盖（同 schema，样本自备 10–20 个）。
// 引擎：GAEA_AB_EVAL_PROVIDER=<name> 选 [[providers]] 条目，缺省取首个有
// model 的条目（走 boot.NewProvider=生产同源装配）。
// 产出：GAEA_AB_EVAL_OUT（缺省 .tmp/abeval/<时间戳>/）——每样本每臂
// record.json（工具轨迹+用量+清单核对+快照增量）+ summary.md 汇总表。
//
// 指标口径（确定性层，裁决五指标中的三样直接测、两样给素材）：
//   墙钟=臂耗时；审批次数=Gate.Check 调用数（计数闸）；
//   返工信号=同一目标路径被写工具命中 >1 次（编辑/重写轮次）；
//   漏项=交付清单逐项在工作区产物文本中命中与否（子串口径，近似——
//   record.json 带全部产物与轨迹供评测席 LLM/人工复核，不替代判断）；
//   证据完整率=Journal 证据卡数 vs 产物文件数（record 给原始两数）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/gaea/gaea/internal/ai"
	appconfig "github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/boot"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
	builtin "github.com/gaea/gaea/internal/gaea/tool/builtin"
)

type abFixture struct {
	Kind    string  `json:"kind"` // xlsx | text
	Path    string  `json:"path"` // 工作区相对
	Sheet   string  `json:"sheet,omitempty"`
	Rows    [][]any `json:"rows,omitempty"`
	Content string  `json:"content,omitempty"`
}

type abSample struct {
	ID        string      `json:"id"`
	Prompt    string      `json:"prompt"`
	Fixtures  []abFixture `json:"fixtures"`
	Checklist []string    `json:"checklist"`
}

// abArmBDiscipline 是 B 臂的处理面（两臂工具面完全一致，差异只在编排纪律——
// 与裁决「A+spill vs A+readonly task」同形：同能力不同路由）。
const abArmBDiscipline = "\n\n[执行纪律] 探索、读取材料、数据汇总、图表绘制等子步骤，优先用 task 工具派发子代理完成（只读子任务传 readonly=true；会写文件的子任务声明 write_paths），主对话只做拆解、编排与最终交付物成文；子代理答复只需要点。"

// abSeedSamples 三个种子样本：典型办公形状（多源汇总/条款提取/图表报告），
// 输入由夹具生成器在运行时落盘（xlsx 用 excelize 造，text 直写）。
func abSeedSamples() []abSample {
	return []abSample{
		{
			ID: "three-xlsx-rollup",
			Prompt: "工作区有 sales.xlsx / cost.xlsx / expense.xlsx 三份部门数据（B 列为金额）。" +
				"汇总三份报表，生成 report/summary.md：逐部门金额小节 + 全部部门合计总额 + 哪个部门金额最高。",
			Fixtures: []abFixture{
				{Kind: "xlsx", Path: "sales.xlsx", Rows: [][]any{{"项目", "金额"}, {"华东", 120}, {"华南", 95}}},
				{Kind: "xlsx", Path: "cost.xlsx", Rows: [][]any{{"项目", "金额"}, {"材料", 210}, {"人工", 140}}},
				{Kind: "xlsx", Path: "expense.xlsx", Rows: [][]any{{"项目", "金额"}, {"差旅", 46}, {"办公", 23}}},
			},
			Checklist: []string{"report/summary.md", "华东", "材料", "差旅", "634"},
		},
		{
			ID: "contract-clauses",
			Prompt: "工作区有 contract.txt（一份服务合同全文）。提取关键条款，生成 report/clauses.md：" +
				"甲方、乙方、合同金额、服务期限、违约责任、争议管辖六项，每项一节并引用原文数字。",
			Fixtures: []abFixture{
				{Kind: "text", Path: "contract.txt", Content: strings.Join([]string{
					"服务合同",
					"甲方：杭州云帆科技有限公司",
					"乙方：北京智算软件有限公司",
					"第一条 服务内容：乙方为甲方提供数据中台建设与运维服务。",
					"第二条 合同金额：本合同总金额为人民币 1,280,000 元（含税）。",
					"第三条 服务期限：自 2026 年 12 月 1 日起至 2027 年 11 月 30 日止，共 12 个月。",
					"第四条 违约责任：任何一方违约应向守约方支付合同金额 10% 的违约金。",
					"第五条 争议解决：因本合同引起的争议提交杭州仲裁委员会仲裁。",
				}, "\n")},
			},
			Checklist: []string{"report/clauses.md", "云帆", "智算", "1,280,000", "12 个月", "10%", "杭州仲裁"},
		},
		{
			ID: "chart-report",
			Prompt: "工作区有 quarter.xlsx（四个季度营收）。用图表工具生成 report/revenue.png 柱状图，" +
				"并写 report/conclusion.md：说明哪个季度营收最高及其金额，引用数据支撑结论。",
			Fixtures: []abFixture{
				{Kind: "xlsx", Path: "quarter.xlsx", Rows: [][]any{{"季度", "营收（万元）"}, {"Q1", 320}, {"Q2", 415}, {"Q3", 388}, {"Q4", 466}}},
			},
			Checklist: []string{"report/revenue.png", "report/conclusion.md", "Q4", "466"},
		},
	}
}

// abEvalProvider 生产同源装配（boot.NewProvider）：选 GAEA_AB_EVAL_PROVIDER
// 点名的条目，缺省取首个有 model 的条目。
func abEvalProvider(t *testing.T) provider.LLMProvider {
	t.Helper()
	cfg, err := config.Load()
	if err != nil || cfg == nil || len(cfg.Providers) == 0 {
		t.Fatalf("config.Load 无可用 [[providers]]: %v", err)
	}
	selector := os.Getenv("GAEA_AB_EVAL_PROVIDER")
	var entry *config.ProviderEntry
	for i := range cfg.Providers {
		p := &cfg.Providers[i]
		if selector != "" {
			if p.Name == selector {
				entry = p
				break
			}
			continue
		}
		// wubigrok 桥条目 model 为空（引擎运行时解析），Kind 非空即可用。
		if entry == nil && (p.Model != "" || len(p.Models) > 0 || p.Kind != "") {
			entry = p
		}
	}
	if entry == nil {
		t.Fatalf("未找到可用的 [[providers]] 条目（selector=%q）", selector)
	}
	// wubigrok 桥需要 ai client + 办公功能级模型绑定（生产=GaeaInit 注入；
	// 评测复刻同源装配——不注入 SetFeature 时桥按全局活跃引擎解析，办公板块
	// 绑定 grok 而活跃引擎是别的模型时就会 404）。ai client 吃 legacy app 级
	// 配置 internal/config.Load，引擎条目吃内核 internal/gaea/config.Load。
	appCfg := appconfig.Load()
	bridge.SetClient(ai.NewClient(appCfg))
	if eng, model := appCfg.GetFeatureModel("gaea"); eng != "" {
		bridge.SetFeature(eng, model)
		t.Logf("办公模型绑定已注入: engine=%s model=%s", eng, model)
	}
	prov, err := boot.NewProvider(entry)
	if err != nil {
		t.Fatalf("boot.NewProvider(%s): %v", entry.Name, err)
	}
	t.Logf("A/B 评测引擎: %s (%s)", entry.Name, entry.Model)
	return prov
}

func abEvalSamples(t *testing.T) []abSample {
	t.Helper()
	if p := os.Getenv("GAEA_AB_EVAL_SAMPLES"); p != "" {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读样本清单: %v", err)
		}
		var samples []abSample
		if err := json.Unmarshal(b, &samples); err != nil {
			t.Fatalf("样本清单 JSON: %v", err)
		}
		return samples
	}
	return abSeedSamples()
}

// abGateCounters 按键计审批次数（键=臂:工具名）。
type abGateCounters struct {
	mu     sync.Mutex
	counts map[string]int
}

func (c *abGateCounters) add(name string) {
	c.mu.Lock()
	c.counts[name]++
	c.mu.Unlock()
}

type abGate struct {
	counts *abGateCounters
	arm    string
}

func (g *abGate) Check(_ context.Context, toolName string, _ json.RawMessage, _ bool) (bool, string, error) {
	g.counts.add(g.arm + ":" + toolName)
	return true, "", nil
}

// abEvalRegistry 臂工具面（两臂逐字节一致）：读面三件+格式/图表+泄洪取回，
// 写面五件（confine 到本臂工作区）。task 不在此处挂——由 abRunArm 先建注册表
// 再建 task（parentReg 指回本注册表）后补挂。bash/浏览器/记忆/调度等一律不进
// （评测只对齐办公文件任务面，且不碰用户真实数据）。
func abEvalRegistry(ws string) *tool.Registry {
	reg := tool.NewRegistry()
	allow := map[string]bool{
		"read_file": true, "ls": true, "grep": true,
		"format_convert": true, "chart_gen": true, "diagram_gen": true,
		"read_spill": true,
	}
	for _, tl := range tool.Builtins() {
		if allow[tl.Name()] {
			reg.Add(tl)
		}
	}
	for _, tl := range builtin.ConfineWriters([]string{ws}) {
		reg.Add(tl)
	}
	return reg
}

func abMaterialize(t *testing.T, ws string, fixtures []abFixture) {
	t.Helper()
	for _, fx := range fixtures {
		full := filepath.Join(ws, filepath.FromSlash(fx.Path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		switch fx.Kind {
		case "xlsx":
			f := excelize.NewFile()
			sheet := fx.Sheet
			if sheet == "" {
				sheet = "Sheet1"
			}
			for r, row := range fx.Rows {
				for c, v := range row {
					cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
					if err := f.SetCellValue(sheet, cell, v); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := f.SaveAs(full); err != nil {
				t.Fatal(err)
			}
			f.Close()
		default: // text/csv 等
			if err := os.WriteFile(full, []byte(fx.Content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

type abToolCall struct {
	Tool string `json:"tool"`
	Args string `json:"args"`
}

type abArmRecord struct {
	Sample           string          `json:"sample"`
	Arm              string          `json:"arm"`
	WallSeconds      float64         `json:"wallSeconds"`
	PromptTokens     int             `json:"promptTokens"`
	CompletionTokens int             `json:"completionTokens"`
	GateAsks         map[string]int  `json:"gateAsks"`
	ToolCalls        []abToolCall    `json:"toolCalls"`
	WriteHits        map[string]int  `json:"writeHits"` // 目标路径→写命中次数（>1=返工信号）
	ReworkSignals    []string        `json:"reworkSignals"`
	JournalCards     int             `json:"journalCards"`
	OutputFiles      []string        `json:"outputFiles"`
	Checklist        map[string]bool `json:"checklist"`
	ChecklistHit     int             `json:"checklistHit"`
	FinalText        string          `json:"finalText"`
	RunError         string          `json:"runError,omitempty"`
}

var abWriterTools = map[string]bool{
	"write_file": true, "edit_file": true, "multi_edit": true, "edit_lines": true,
	"move_file": true, "chart_gen": true, "diagram_gen": true, "schedule_apply": true,
}

// abArgTarget 从工具参数里提写目标（path/source/destination/output 任一）。
func abArgTarget(name, args string) string {
	if !abWriterTools[name] {
		return ""
	}
	var m map[string]any
	if json.Unmarshal([]byte(args), &m) != nil {
		return ""
	}
	for _, k := range []string{"path", "destination", "output"} {
		if v, ok := m[k].(string); ok && v != "" {
			return name + ":" + v
		}
	}
	return ""
}

func abRunArm(t *testing.T, prov provider.LLMProvider, s abSample, arm, discipline string) abArmRecord {
	t.Helper()
	ws := t.TempDir()
	abMaterialize(t, ws, s.Fixtures)
	before := abSnapshot(t, ws)

	counts := &abGateCounters{counts: map[string]int{}}
	gate := &abGate{counts: counts, arm: arm}
	reg := abEvalRegistry(ws)
	taskTool := agent.NewTaskTool(prov, nil, reg, 40, 0, 0.2, "", "", gate).
		WithAccessPolicy(tool.NewPathAccessPolicy(ws, nil, nil, nil))
	taskTool.SetSpill(true)
	reg.Add(taskTool)

	var mu sync.Mutex
	var rec abArmRecord
	rec.Sample, rec.Arm = s.ID, arm
	rec.WriteHits = map[string]int{}
	rec.Checklist = map[string]bool{}
	sink := event.FuncSink(func(e event.Event) {
		switch e.Kind {
		case event.ToolDispatch:
			mu.Lock()
			rec.ToolCalls = append(rec.ToolCalls, abToolCall{Tool: e.Tool.Name, Args: e.Tool.Args})
			mu.Unlock()
		case event.Usage:
			mu.Lock()
			if e.Usage != nil {
				rec.PromptTokens += e.Usage.PromptTokens
				rec.CompletionTokens += e.Usage.CompletionTokens
			}
			mu.Unlock()
		}
	})

	sess := agent.NewSession("You are a meticulous office file assistant working in a workspace directory. Complete the task with the provided tools; deliver files as requested.")
	if !spaces.Valid(sess.Space()) {
		sess.SetSpace(spaces.SpaceWork)
	}
	runner := agent.New(prov, reg, sess, agent.Options{
		MaxSteps:     48,
		Temperature:  0.2,
		Gate:         gate,
		AccessPolicy: tool.NewPathAccessPolicy(ws, nil, nil, nil),
		JournalDir:   filepath.Join(ws, ".gaea", "work", "journal"),
		Spill:        true,
	}, sink)

	timeout := 15 * time.Minute
	if v := os.Getenv("GAEA_AB_EVAL_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = time.Duration(n) * time.Minute
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	turn, runErr := runner.Run(ctx, s.Prompt+discipline)
	rec.WallSeconds = time.Since(start).Seconds()
	if runErr != nil {
		rec.RunError = runErr.Error()
	}
	rec.FinalText = turn.Summary
	if turn.Success {
		// FilesCreated/FilesModified 是 runner 官方产物清单，与快照增量互为佐证。
		rec.OutputFiles = append(rec.OutputFiles, turn.FilesCreated...)
		rec.OutputFiles = append(rec.OutputFiles, turn.FilesModified...)
	}
	rec.GateAsks = counts.counts // 跑完再取（run 期间仍在被写）

	// 返工信号：写命中 >1 的目标路径。
	for k, v := range rec.WriteHits {
		if v > 1 {
			rec.ReworkSignals = append(rec.ReworkSignals, k)
		}
	}
	// 证据卡数（Journal 落卡）。
	rec.JournalCards = abCountFiles(t, filepath.Join(ws, ".gaea", "work", "journal"))
	// 快照增量=产物文件。
	after := abSnapshot(t, ws)
	for rel := range after {
		if before[rel] != after[rel] {
			rec.OutputFiles = append(rec.OutputFiles, rel)
		}
	}
	// 清单核对：全部产物文本拼合后子串命中（大小写不敏感）。
	var corpus strings.Builder
	for rel, content := range after {
		if before[rel] == content {
			continue // 只看新/改产物
		}
		if isTextual(rel) {
			corpus.WriteString(rel)
			corpus.WriteString("\n")
			if len(content) < 1<<20 {
				corpus.WriteString(content)
				corpus.WriteString("\n")
			}
		} else {
			corpus.WriteString(rel)
			corpus.WriteString("\n")
		}
	}
	corpus.WriteString(rec.FinalText)
	// 归一化：Windows rel 路径是反斜杠，清单/产物统一正斜杠再比对。
	hay := strings.ToLower(strings.ReplaceAll(corpus.String(), "\\", "/"))
	for _, item := range s.Checklist {
		hit := strings.Contains(hay, strings.ToLower(strings.ReplaceAll(item, "\\", "/")))
		rec.Checklist[item] = hit
		if hit {
			rec.ChecklistHit++
		}
	}
	return rec
}

func abSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		if strings.Contains(rel, ".gaea"+string(os.PathSeparator)+"work"+string(os.PathSeparator)+"journal") {
			return nil // 证据卡不算产物
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		files[rel] = string(b)
		return nil
	})
	return files
}

func abCountFiles(t *testing.T, root string) int {
	t.Helper()
	n := 0
	filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func isTextual(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".md", ".txt", ".json", ".csv", ".html", ".svg":
		return true
	}
	return false
}

func TestABEvalLive(t *testing.T) {
	if os.Getenv("GAEA_AB_EVAL") == "" {
		t.Skip("设 GAEA_AB_EVAL=1 跑 A/B 实弹评测（真引擎真 token，分钟级）")
	}
	prov := abEvalProvider(t)
	samples := abEvalSamples(t)
	outDir := os.Getenv("GAEA_AB_EVAL_OUT")
	if outDir == "" {
		outDir = filepath.Join("..", "..", ".tmp", "abeval", time.Now().Format("20060102_150405"))
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var rows []string
	rows = append(rows, "| 样本 | 臂 | 墙钟(s) | tokens(P/C) | 清单命中 | 返工信号 | 证据卡 | 审批次数 |",
		"|---|---|---|---|---|---|---|---|")
	for _, s := range samples {
		for _, arm := range []struct {
			name, discipline string
		}{{"A-spill", ""}, {"B-readonly-task", abArmBDiscipline}} {
			rec := abRunArm(t, prov, s, arm.name, arm.discipline)
			// WriteHits 从 ToolCalls 现算（sink 里只留轨迹，判定集中在收尾）。
			rec.WriteHits = map[string]int{}
			for _, c := range rec.ToolCalls {
				if tgt := abArgTarget(c.Tool, c.Args); tgt != "" {
					rec.WriteHits[tgt]++
				}
			}
			rec.ReworkSignals = nil
			for k, v := range rec.WriteHits {
				if v > 1 {
					rec.ReworkSignals = append(rec.ReworkSignals, k)
				}
			}
			b, _ := json.MarshalIndent(rec, "", "  ")
			rp := filepath.Join(outDir, fmt.Sprintf("%s_%s.json", rec.Sample, rec.Arm))
			if err := os.WriteFile(rp, b, 0o644); err != nil {
				t.Fatal(err)
			}
			rows = append(rows, fmt.Sprintf("| %s | %s | %.0f | %d/%d | %d/%d | %d | %d | %d |",
				rec.Sample, rec.Arm, rec.WallSeconds, rec.PromptTokens, rec.CompletionTokens,
				rec.ChecklistHit, len(s.Checklist), len(rec.ReworkSignals), rec.JournalCards,
				len(rec.GateAsks)))
			if rec.RunError != "" {
				rows = append(rows, fmt.Sprintf("| ^ | ^ 失败 | %s |", truncateOneLine(rec.RunError, 120)))
			}
			t.Logf("%s/%s: 墙钟 %.0fs 清单 %d/%d", rec.Sample, rec.Arm, rec.WallSeconds, rec.ChecklistHit, len(s.Checklist))
		}
	}
	sm := "# A/B 评测汇总\n\n" + strings.Join(rows, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(outDir, "summary.md"), []byte(sm), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("汇总已写 %s", filepath.Join(outDir, "summary.md"))
}

func truncateOneLine(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
