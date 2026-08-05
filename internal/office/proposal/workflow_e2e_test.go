package proposal

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/knowledge"
)

// TestFullWorkflowE2E 完整工作流端到端：项目→招标文件→解析→大纲→生成→批量→装配→检查→导出→归档→参考
func TestFullWorkflowE2E(t *testing.T) {
	ctx := context.Background()
	kst, err := knowledge.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	const sectionBody = "这是一段用于重复率检测的章节正文内容，修复工期为 90 日历天，采用固化稳定化工艺，执行 GB 36600-2018 标准要求，包含技术描述与工艺参数说明，长度超过四十个字符。"
	ai := &mockAI{
		replies: map[string]string{
			"请解析以下招标文件": `{
  "totalWords": 120000,
  "overview": "某污染地块土壤修复",
  "overviewQuote": "某污染地块土壤修复工程",
  "duration": "90 日历天",
  "durationQuote": "工期要求：90 日历天",
  "qualification": [{"name":"环保工程资质","content":"环保工程专业承包三级及以上","quote":"环保工程专业承包三级及以上"}],
  "techScoring": [{"name":"施工方案","maxScore":"20","requirement":"方案完整合理","quote":"施工方案 20 分"}],
  "keyRequirements": ["项目经理须常驻现场"],
  "redLines": [{"name":"废标条款","content":"投标文件未按要求签字盖章作废标处理","quote":"未按要求签字盖章"}],
  "format": [{"name":"装订","content":"A4 双面打印","quote":"A4 双面打印"}],
  "darkRules": [{"name":"暗标","content":"不得出现单位名称、不得加粗","quote":"不得出现单位名称"}]
}`,
			"总字数目标": `{"title":"某污染地块修复投标方案","sections":[
  {"title":"第一章 项目概况","level":1,"children":[
    {"title":"1.1 项目背景","level":2},
    {"title":"1.2 修复目标","level":2}
  ]},
  {"title":"第二章 技术路线","level":1,"children":[
    {"title":"2.1 工艺比选","level":2},
    {"title":"2.2 施工组织","level":2}
  ]}
]}`,
			"【1.1 项目背景】": `[
  {"name":"施工方案","maxScore":"20","covered":"full","suggestion":"覆盖完整"}
]`,
		},
		def: sectionBody,
	}
	s := newServiceAt(t, t.TempDir(), ai)
	s.SetKnowledgeStoreForTest(kst)

	// ── 1. 项目 + 方案 ────────────────────────────────────
	proj, err := s.store.EnsureDefaultProject()
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Create("某污染地块修复投标方案", "soil-remediation-bid", "污染场地修复", "环保工程", proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.ProjectID != proj.ID {
		t.Fatalf("方案未挂项目: %+v", p)
	}
	t.Logf("1. 方案创建 OK：%s（项目 %s）", p.Title, proj.ID)

	// ── 2. 招标文件（已转换 Markdown）───────────────────────
	tenderMD := "一、项目概况\n某污染地块土壤修复工程。\n二、工期要求\n工期要求：90 日历天。\n三、资质要求\n环保工程专业承包三级及以上。\n四、评分标准\n施工方案 20 分。\n五、其他\n投标文件未按要求签字盖章作废标处理；A4 双面打印；暗标不得出现单位名称。"
	fileID, err := s.store.AddFile(p.ID, "tender", "招标文件.txt", "tender.txt", len(tenderMD))
	if err != nil {
		t.Fatal(err)
	}
	p.BidSummary = &BidSummary{RawFiles: []FileDoc{{FileID: fileID, Name: "招标文件.txt", Markdown: tenderMD}}}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}
	t.Log("2. 招标文件登记 OK")

	// ── 3. 招标解析 ───────────────────────────────────────
	parsed, err := s.ParseBidFile(ctx, p.ID)
	if err != nil {
		t.Fatalf("ParseBidFile: %v", err)
	}
	bs := parsed.BidSummary
	if bs.ParseStatus != "done" || bs.TotalWords != 120000 || len(bs.TechScoring) != 1 || len(bs.Qualification) != 1 {
		t.Fatalf("解析结果异常: %+v", bs)
	}
	rows, err := s.store.ListParseResults(p.ID)
	if err != nil || len(rows) == 0 {
		t.Fatalf("parse_results 未落库: %v %+v", err, rows)
	}
	if bs.TechScoring[0].Sources[0].Snippet == "" {
		t.Fatalf("评分项来源缺失: %+v", bs.TechScoring[0])
	}
	t.Logf("3. 招标解析 OK：totalWords=%d、评分项=%d、parse_results=%d 行、来源=%q", bs.TotalWords, len(bs.TechScoring), len(rows), bs.TechScoring[0].Sources[0].Snippet)

	// ── 4. 大纲生成（按评标办法 + 字数取招标要求）────────────
	outlined, err := s.GenerateOutline(ctx, p.ID, "污染场地修复", OutlineStrategyScoring, 0)
	if err != nil {
		t.Fatalf("GenerateOutline: %v", err)
	}
	if len(outlined.Sections) != 2 || len(outlined.Sections[0].Children) != 2 {
		t.Fatalf("大纲树异常: %+v", outlined.Sections)
	}
	sum := 0
	for _, sec := range flattenSections(outlined.Sections) {
		if len(sec.Children) == 0 {
			sum += sec.WordTarget
		}
	}
	if sum != 120000 {
		t.Fatalf("字数预算合计 = %d, want 120000", sum)
	}
	t.Logf("4. 大纲生成 OK：2 章 4 节，叶子字数合计=%d", sum)

	// ── 5. 单章流式生成 ───────────────────────────────────
	targetID := outlined.Sections[0].Children[0].ID
	gen1, err := s.GenerateSection(ctx, p.ID, targetID, "")
	if err != nil {
		t.Fatalf("GenerateSection: %v", err)
	}
	sec := findSectionByID(gen1.Sections, targetID)
	if sec == nil || sec.Status != "completed" || sec.Content == "" {
		t.Fatalf("单章生成异常: %+v", sec)
	}
	t.Logf("5. 单章生成 OK：%s（%d 字）", sec.Title, sec.Words)

	// ── 6. 批量生成剩余（含断点续写：跳过已完成）─────────────
	var events []string
	if err := s.RunBatch(ctx, p.ID, func(cur, total int, sid, status string, words int) {
		events = append(events, sid+":"+status)
	}); err != nil {
		t.Fatalf("RunBatch: %v", err)
	}
	final, _ := s.store.Get(p.ID)
	completed := 0
	for _, sec := range flattenSections(final.Sections) {
		if sec.Status == "completed" {
			completed++
		}
	}
	if completed != 4 {
		t.Fatalf("批量后完成章节 = %d, want 4", completed)
	}
	if len(events) != 4 {
		t.Fatalf("批量事件 = %d, want 4: %+v", len(events), events)
	}
	t.Logf("6. 批量生成 OK：4/4 完成（事件=%v）", events)

	// ── 7. 合并装配 ───────────────────────────────────────
	md := Assemble(final)
	for _, want := range []string{"第1章", "第2章", "1.1", "2.2"} {
		if !strings.Contains(md, want) {
			t.Fatalf("装配缺少 %q", want)
		}
	}
	t.Log("7. 合并装配 OK：第1章/1.1/2.2 编号正确")

	// ── 8. 全面检查 ───────────────────────────────────────
	_ = s.store.SaveProjectFacts(proj.ID, map[string]string{"工期": "90 日历天", "业主单位": "某区生态环境局"})
	_, items, err := s.CheckAll(ctx, p.ID)
	if err != nil {
		t.Fatalf("CheckAll: %v", err)
	}
	rules := map[string]string{}
	for _, it := range items {
		rules[it.Rule] = it.Status
	}
	for _, want := range []string{"评分覆盖检查", "废标条款响应", "数据一致性检查", "重复率检测", "规范引用检查"} {
		if rules[want] == "" {
			t.Fatalf("检查缺少规则 %s（实际 %v）", want, rules)
		}
	}
	if rules["重复率检测"] != "fail" {
		t.Fatalf("重复率应 fail（章节内容相同），实际 %s", rules["重复率检测"])
	}
	t.Logf("8. 全面检查 OK：%d 项，规则=%v", len(items), rules)

	// ── 9. 导出 MD ────────────────────────────────────────
	exportPath, err := s.ExportMarkdown(p.ID)
	if err != nil {
		t.Fatalf("ExportMarkdown: %v", err)
	}
	data, err := os.ReadFile(exportPath)
	if err != nil || !strings.Contains(string(data), "第1章") {
		t.Fatalf("导出文件异常: %s %v", exportPath, err)
	}
	t.Logf("9. 导出 OK：%s", exportPath)

	// ── 10. 归档到记忆中枢 + 同类参考注入 ──────────────────
	name, err := s.ArchiveProposal(p.ID)
	if err != nil {
		t.Fatalf("ArchiveProposal: %v", err)
	}
	entry, err := kst.Get(name)
	if err != nil || entry.Title == "" {
		t.Fatalf("归档条目缺失: %v %+v", err, entry)
	}
	ref := s.legacyRefFor("soil-remediation-bid", "环保工程")
	if ref == "" || !strings.Contains(ref, "某污染地块修复投标方案") {
		t.Fatalf("同类参考异常: %s", ref)
	}
	sc, err := s.SectionContext(ctx, p.ID, targetID)
	if err != nil || !strings.Contains(sc.UserPrompt, "【历史方案参考") {
		t.Fatalf("SectionContext 未注入历史参考: %v", err)
	}
	t.Logf("10. 归档 OK：%s（同类参考已注入）", name)

	t.Log("✅ 完整工作流 E2E 通过：建项目→招标文件→解析→大纲→生成→批量→装配→检查→导出→归档→参考")
}
