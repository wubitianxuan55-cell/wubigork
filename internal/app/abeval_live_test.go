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
	"archive/zip"
	"bytes"
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
	Kind    string  `json:"kind"` // xlsx | text | docx | docx_revs | pptx（P2 质量矩阵扩 docx/pptx，hermetic 造件）
	Path    string  `json:"path"` // 工作区相对
	Sheet   string  `json:"sheet,omitempty"`
	Rows    [][]any `json:"rows,omitempty"`
	Content string  `json:"content,omitempty"` // docx/pptx=逐行段落文本
}

type abSample struct {
	ID        string      `json:"id"`
	Prompt    string      `json:"prompt"`
	Fixtures  []abFixture `json:"fixtures"`
	Checklist []string    `json:"checklist"`
	// Negative=true=已知缺陷件样本（checklist 故意含源材料缺失项）：漏项为
	// 预期，汇总表单独标记，不计入正例通过率分母（P2 缺陷件标注，裁决口径
	// 「真实件+缺陷件成对」的评测面形态）。
	Negative bool `json:"negative,omitempty"`
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
		// ── P2 质量矩阵种子（2026-11-10 众神殿裁决 action-5：种子先行 3 格）──
		{
			ID: "docx-term-edit",
			Prompt: "工作区有 contract.docx（服务合同）。用 docx_apply 把「共 12 个月」改为「共 24 个月」" +
				"（修订制写入），然后生成 report/change-log.md：记录修改前的期限条款与修改后的期限条款。",
			Fixtures: []abFixture{
				{Kind: "docx", Path: "contract.docx", Content: strings.Join([]string{
					"服务合同",
					"甲方：杭州云帆科技有限公司",
					"乙方：北京智算软件有限公司",
					"第三条 服务期限：自 2026 年 12 月 1 日起至 2027 年 11 月 30 日止，共 12 个月。",
					"第四条 合同金额：人民币 1,280,000 元（含税）。",
				}, "\n")},
			},
			Checklist: []string{"report/change-log.md", "24 个月", "12 个月"},
		},
		{
			ID: "docx-accept-revs",
			Prompt: "工作区有 reviewed.docx（已带 gaea AI 修订：期限 30 天改 60 天）。用 docx_accept 接受全部修订，" +
				"然后生成 report/accept-note.md：说明接受后的期限。",
			Fixtures: []abFixture{
				{Kind: "docx_revs", Path: "reviewed.docx"},
			},
			Checklist: []string{"report/accept-note.md", "60 天"},
		},
		{
			ID: "pptx-budget-edit",
			Prompt: "工作区有 deck.pptx（两页：预算页与交付计划页）。用 pptx_apply 把第 1 页的" +
				"「预算总额：120 万元」替换为「预算总额：180 万元」，然后生成 report/deck-note.md 说明改动。",
			Fixtures: []abFixture{
				{Kind: "pptx", Path: "deck.pptx", Content: strings.Join([]string{
					"预算总额：120 万元",
					"交付计划：2027 年 Q1 交付",
				}, "\n")},
			},
			Checklist: []string{"report/deck-note.md", "180 万元"},
		},
		// 已知缺陷件样本（Negative）：源材料缺「违约责任」条目，清单仍要求它——
		// 漏项为预期，验证漏项指标能真报缺陷而不是误判模型。
		{
			ID: "clauses-negative-missing-liability",
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
					"第五条 争议解决：因本合同引起的争议提交杭州仲裁委员会仲裁。",
				}, "\n")},
			},
			Checklist: []string{"report/clauses.md", "云帆", "1,280,000", "违约责任", "杭州仲裁"},
			Negative:  true,
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
		// P2 质量矩阵（2026-11-10 众神殿裁决 action-5）：只读渲染进臂。
		"doc_render": true,
	}
	for _, tl := range tool.Builtins() {
		if allow[tl.Name()] {
			reg.Add(tl)
		}
	}
	// P1a office 工具族（app ExtraTools 装配面，非 builtin）：三件套适配器与
	// 生成入口此前不在臂内，矩阵交付格根本跑不了（zcode-r3 缺口②最硬条）。
	// 均实现 WritePathResolver，AccessPolicy(ws) 逐调用包含校验与
	// ConfineWriters 同界。
	for _, tl := range gaeaExtraTools(&App{}) {
		switch tl.Name() {
		case "office_generate", "docx_apply", "docx_accept", "pptx_apply", "xlsx_apply":
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
		case "docx", "docx_revs":
			var raw []byte
			if fx.Kind == "docx_revs" {
				raw = abBuildDocxRevs(t)
			} else {
				raw = abBuildDocx(t, strings.Split(fx.Content, "\n"))
			}
			if err := os.WriteFile(full, raw, 0o644); err != nil {
				t.Fatal(err)
			}
		case "pptx":
			if err := os.WriteFile(full, abBuildPptx(t, strings.Split(fx.Content, "\n")), 0o644); err != nil {
				t.Fatal(err)
			}
		default: // text/csv 等
			if err := os.WriteFile(full, []byte(fx.Content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// ── P2 hermetic 造件（纯 Go zip，形状对齐 office 包测试夹具；不依赖 python/
// soffice，CI 可复算）───────────────────────────────────────────────────

const abDocxDocTmpl = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>%s
  </w:body>
</w:document>`

func abZipEntries(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// abBuildDocx 造最小 docx：每行一个 w:p 段落（纯文本 run）。
func abBuildDocx(t *testing.T, paragraphs []string) []byte {
	var body strings.Builder
	for _, p := range paragraphs {
		body.WriteString("<w:p><w:r><w:t>" + p + "</w:t></w:r></w:p>")
	}
	return abZipEntries(t, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": fmt.Sprintf(abDocxDocTmpl, body.String()),
	})
}

// abBuildDocxRevs 造带 gaea AI 修订的最小 docx（w:del 30 天 + w:ins 60 天），
// 供 docx_accept 样本「接受全部修订」。
func abBuildDocxRevs(t *testing.T) []byte {
	body := `<w:p><w:r><w:t>合同期限为 </w:t></w:r>` +
		`<w:del w:id="1" w:author="gaea AI" w:date="2026-11-10T00:00:00Z"><w:r><w:delText>30 天</w:delText></w:r></w:del>` +
		`<w:ins w:id="2" w:author="gaea AI" w:date="2026-11-10T00:00:00Z"><w:r><w:t>60 天</w:t></w:r></w:ins>` +
		`<w:r><w:t>。</w:t></w:r></w:p>`
	return abZipEntries(t, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`,
		"word/document.xml": fmt.Sprintf(abDocxDocTmpl, body),
	})
}

// abBuildPptx 造最小 pptx：每行文本一个 slide（单 run 段落），页数随行数。
func abBuildPptx(t *testing.T, slideTexts []string) []byte {
	var sldIds strings.Builder
	for i := range slideTexts {
		fmt.Fprintf(&sldIds, `<p:sldId id="%d" r:id="rId%d"/>`, 257+i, i+2)
	}
	var overrides, relEntries strings.Builder
	relEntries.WriteString(`<Relationship Id="rId1" Type="http://x/slideMaster" Target="slideMasters/slideMaster1.xml"/>`)
	for i := range slideTexts {
		fmt.Fprintf(&overrides, "\n  "+`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i+1)
		fmt.Fprintf(&relEntries, "\n"+`<Relationship Id="rId%d" Type="http://x/slide" Target="slides/slide%d.xml"/>`, i+2, i+1)
	}
	entries := map[string]string{
		"[Content_Types].xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>%s
</Types>`, overrides.String()),
		"ppt/_rels/presentation.xml.rels": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
%s
</Relationships>`, relEntries.String()),
		"ppt/theme/theme1.xml": "<theme/>",
		"ppt/presentation.xml": fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<p:sldIdLst>%s</p:sldIdLst>
</p:presentation>`, sldIds.String()),
	}
	for i, txt := range slideTexts {
		entries[fmt.Sprintf("ppt/slides/slide%d.xml", i+1)] = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
<p:cSld><p:spTree>
<p:sp><p:txBody><a:bodyPr/><a:p><a:r><a:rPr lang="zh-CN"/><a:t>%s</a:t></a:r></a:p></p:txBody></p:sp>
</p:spTree></p:cSld>
</p:sld>`, txt)
	}
	return abZipEntries(t, entries)
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
	// P1a office 工具族（P2 质量矩阵）：结构化写面同样计返工信号。
	"office_generate": true, "docx_apply": true, "docx_accept": true,
	"pptx_apply": true, "xlsx_apply": true,
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
	for _, k := range []string{"path", "destination", "output", "output_path"} {
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

	// P2：office 工具族经 gaeaCwd() 解析声明路径与落 Journal——臂工作区须写进
	// App 单例配置，与 AccessPolicy 同根（isolateWorkspaceTo 同款，测毕恢复）。
	ga.mu.Lock()
	oldCfg := ga.cfg
	ga.cfg = &config.Config{Workspace: ws}
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.cfg = oldCfg
		ga.mu.Unlock()
	})

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
		negativeTag := ""
		if s.Negative {
			negativeTag = " ⚠缺陷件（漏项为预期）"
		}
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
			rows = append(rows, fmt.Sprintf("| %s%s | %s | %.0f | %d/%d | %d/%d | %d | %d | %d |",
				rec.Sample, negativeTag, rec.Arm, rec.WallSeconds, rec.PromptTokens, rec.CompletionTokens,
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
