package proposal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	officedb "github.com/gaea/gaea/internal/office/db"
)

// mockAI 简单 AI 客户端 mock — 按调用返回预设文本
type mockAI struct {
	replies map[string]string // 按 userMsg 前缀匹配
	def     string            // 默认回复
}

func (m *mockAI) ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error) {
	for prefix, r := range m.replies {
		if strings.Contains(userMsg, prefix) {
			return r, nil
		}
	}
	return m.def, nil
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newServiceAt(t, t.TempDir(), &mockAI{def: "mock"})
}

// newServiceAt 在指定目录创建服务并注册数据库关闭清理
func newServiceAt(t *testing.T, dir string, ai AIClient) *Service {
	t.Helper()
	svc := NewService(dir, ai)
	t.Cleanup(func() { _ = officedb.CloseDatabase(filepath.Join(dir, "office")) })
	return svc
}

// ─── CRUD ────────────────────────────────────────────────────

func TestCreateAndGet(t *testing.T) {
	s := newTestService(t)
	p, err := s.Create("土壤修复方案", "soil-remediation-bid", "某场地污染修复", "环保工程")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == "" {
		t.Fatal("ID 为空")
	}
	if p.Status != "draft" {
		t.Errorf("Status = %q, want draft", p.Status)
	}
	if p.Title != "土壤修复方案" {
		t.Errorf("Title = %q", p.Title)
	}

	got, err := s.Get(p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != p.Title || got.Category != "环保工程" {
		t.Errorf("Get 返回不一致: %+v", got)
	}
}

func TestCreateAttachesToDefaultProject(t *testing.T) {
	s := newTestService(t)
	p, err := s.Create("方案", "blank", "", "其他")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ProjectID != "default" {
		t.Errorf("ProjectID = %q, want default", p.ProjectID)
	}
	proj, err := s.store.GetProject("default")
	if err != nil || proj.Name != "未归档项目" {
		t.Fatalf("默认项目异常: %v %+v", err, proj)
	}
}

func TestCreate_WithSections(t *testing.T) {
	s := newTestService(t)
	p, err := s.Create("方案", "soil-remediation-bid", "需求", "环保工程")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Sections) == 0 {
		t.Error("模板应有默认章节")
	}
	// 章节应有 ID 和 ProposalID
	for _, sec := range p.Sections {
		if sec.ID == "" {
			t.Error("章节 ID 为空")
		}
		if sec.ProposalID != p.ID {
			t.Errorf("章节 ProposalID = %q, want %q", sec.ProposalID, p.ID)
		}
		if sec.Status != "pending" {
			t.Errorf("章节 Status = %q, want pending", sec.Status)
		}
	}
}

func TestList(t *testing.T) {
	s := newTestService(t)
	s.Create("A", "soil-remediation-bid", "", "")
	s.Create("B", "soil-remediation-bid", "", "")
	items, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("List 数量 = %d, want 2", len(items))
	}
}

func TestGet_Missing(t *testing.T) {
	s := newTestService(t)
	if _, err := s.Get("nope"); err == nil {
		t.Error("不存在的方案应报错")
	}
}

func TestUpdate_Persists(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("旧标题", "soil-remediation-bid", "", "")
	p.Title = "新标题"
	if err := s.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := s.Get(p.ID)
	if got.Title != "新标题" {
		t.Errorf("标题 = %q, want 新标题", got.Title)
	}
}

func TestDelete(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("待删", "soil-remediation-bid", "", "")
	if err := s.Delete(p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(p.ID); err == nil {
		t.Error("删除后 Get 应报错")
	}
}

func TestPersistence_AcrossServiceInstances(t *testing.T) {
	// 模拟重启：新 Service 指向同一目录，数据应保留
	dir := t.TempDir()
	s1 := newServiceAt(t, dir, &mockAI{def: "mock"})
	p, _ := s1.Create("持久化方案", "soil-remediation-bid", "", "")

	s2 := newServiceAt(t, dir, &mockAI{def: "mock"})
	got, err := s2.Get(p.ID)
	if err != nil {
		t.Fatalf("重启后 Get: %v", err)
	}
	if got.Title != "持久化方案" {
		t.Errorf("重启后标题 = %q", got.Title)
	}
}

// ─── 章节操作 ────────────────────────────────────────────────

func TestUpdateSection_TopLevel(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	if len(p.Sections) == 0 {
		t.Fatal("无章节可更新")
	}
	target := p.Sections[0]
	updated, err := s.UpdateSection(p.ID, target.ID, "新章名", "新内容")
	if err != nil {
		t.Fatalf("UpdateSection: %v", err)
	}
	// 在更新后的方案中找对应章节
	found := false
	for _, sec := range updated.Sections {
		if sec.ID == target.ID {
			found = true
			if sec.Title != "新章名" || sec.Content != "新内容" {
				t.Errorf("章节未更新: %+v", sec)
			}
			if sec.Status != "completed" {
				t.Errorf("Status = %q, want completed", sec.Status)
			}
		}
	}
	if !found {
		t.Error("更新后的方案找不到目标章节")
	}
}

func TestUpdateSection_ChildRecursive(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	// 找到第一个有子章节的（大纲生成后才有子章节，这里手动构造）
	// 直接验证递归查找：添加一个手动构造的深层章节
	sec := p.Sections[0]
	childID := "child-1"
	sec.Children = append(sec.Children, ProposalSection{ID: childID, Title: "子节", ProposalID: p.ID, Status: "pending"})
	p.Sections[0] = sec
	s.Update(p)

	updated, err := s.UpdateSection(p.ID, childID, "子节新名", "子节内容")
	if err != nil {
		t.Fatalf("UpdateSection 子章节: %v", err)
	}
	// 递归查找子章节
	var check func(sections []ProposalSection) bool
	check = func(sections []ProposalSection) bool {
		for _, sc := range sections {
			if sc.ID == childID {
				return sc.Title == "子节新名" && sc.Content == "子节内容"
			}
			if check(sc.Children) {
				return true
			}
		}
		return false
	}
	if !check(updated.Sections) {
		t.Error("子章节未更新（递归查找失败）")
	}
}

func TestUpdateSection_MissingSection(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	if _, err := s.UpdateSection(p.ID, "no-such-id", "t", "c"); err == nil {
		t.Error("不存在的章节应报错")
	}
}

// ─── 原始文件操作 ────────────────────────────────────────────

func TestSaveRawText_And_Remove(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")

	// 创建临时文件作为原始文件
	tmp := filepath.Join(t.TempDir(), "招标文件.txt")
	if err := os.WriteFile(tmp, []byte("招标文件内容"), 0644); err != nil {
		t.Fatal(err)
	}

	updated, err := s.SaveRawText(p.ID, tmp)
	if err != nil {
		t.Fatalf("SaveRawText: %v", err)
	}
	if updated.BidSummary == nil || len(updated.BidSummary.RawFiles) != 1 {
		t.Fatalf("BidSummary.RawFiles 未添加: %+v", updated.BidSummary)
	}
	if updated.BidSummary.RawFiles[0].Name != "招标文件.txt" {
		t.Errorf("文件名 = %q", updated.BidSummary.RawFiles[0].Name)
	}

	// 移除
	removed, err := s.RemoveRawFile(p.ID, 0)
	if err != nil {
		t.Fatalf("RemoveRawFile: %v", err)
	}
	if removed.BidSummary == nil || len(removed.BidSummary.RawFiles) != 0 {
		t.Error("RawFiles 未移除")
	}
}

func TestRemoveRawFile_OutOfRange(t *testing.T) {
	s := newTestService(t)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	if _, err := s.RemoveRawFile(p.ID, 5); err == nil {
		t.Error("越界索引应报错")
	}
}

// ─── AI 路径：GenerateOutline / GenerateSection ──────────────

const outlineJSON = `{"title":"土壤修复施工方案","sections":[
  {"title":"第1章 工程概况","level":1,"children":[
    {"title":"1.1 项目背景","level":2,"children":[
      {"title":"1.1.1 场地现状","level":3}
    ]}
  ]},
  {"title":"第2章 修复工艺","level":1}
]}`

func TestGenerateOutline(t *testing.T) {
	dir := t.TempDir()
	ai := &mockAI{replies: map[string]string{"需求描述": outlineJSON}}
	s := newServiceAt(t, dir, ai)
	p, _ := s.Create("方案", "soil-remediation-bid", "污染场地修复", "环保工程")

	got, err := s.GenerateOutline(context.Background(), p.ID, "污染场地修复", OutlineStrategyReference, 150000)
	if err != nil {
		t.Fatalf("GenerateOutline: %v", err)
	}
	// 验证大纲写入：2 章 + 1 节 + 1 小节
	if len(got.Sections) != 2 {
		t.Fatalf("章数 = %d, want 2", len(got.Sections))
	}
	if got.Sections[0].Title != "第1章 工程概况" {
		t.Errorf("第1章标题 = %q", got.Sections[0].Title)
	}
	if len(got.Sections[0].Children) != 1 {
		t.Fatalf("第1章子节数 = %d, want 1", len(got.Sections[0].Children))
	}
	child := got.Sections[0].Children[0]
	if child.Title != "1.1 项目背景" || len(child.Children) != 1 {
		t.Errorf("子节解析错误: %+v", child)
	}
	if child.Children[0].Title != "1.1.1 场地现状" {
		t.Errorf("小节标题 = %q", child.Children[0].Title)
	}
	// 大纲标题应更新到方案
	if got.Title != "土壤修复施工方案" {
		t.Errorf("方案标题 = %q, want 土壤修复施工方案", got.Title)
	}
	// 持久化验证
	persisted, _ := s.Get(p.ID)
	if persisted.Title != "土壤修复施工方案" || len(persisted.Sections) != 2 {
		t.Error("大纲未持久化")
	}
}

func TestGenerateOutline_AINil(t *testing.T) {
	s := newServiceAt(t, t.TempDir(), nil)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	if _, err := s.GenerateOutline(context.Background(), p.ID, "", OutlineStrategyReference, 150000); err == nil {
		t.Error("AI 为 nil 应报错")
	}
}

func TestGenerateOutline_BadJSON(t *testing.T) {
	ai := &mockAI{def: "不是JSON内容"}
	s := newServiceAt(t, t.TempDir(), ai)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	// 非法 JSON 不应崩溃，应报错或返回原方案
	_, err := s.GenerateOutline(context.Background(), p.ID, "需求", OutlineStrategyReference, 150000)
	if err == nil {
		// 某些实现可能容忍，检查方案未被破坏
		if got, _ := s.Get(p.ID); got.Title != "方案" {
			t.Error("方案被非法 JSON 破坏")
		}
	}
}

func TestGenerateSection(t *testing.T) {
	ai := &mockAI{
		replies: map[string]string{"需求描述": outlineJSON},
		def:     "这是生成的章节内容，字数足够长。" + strings.Repeat("内容", 50),
	}
	s := newServiceAt(t, t.TempDir(), ai)
	p, _ := s.Create("方案", "soil-remediation-bid", "", "")
	// 生成大纲后生成章节
	p, err := s.GenerateOutline(context.Background(), p.ID, "需求", OutlineStrategyReference, 150000)
	if err != nil {
		t.Fatalf("GenerateOutline: %v", err)
	}
	if len(p.Sections) == 0 {
		t.Fatal("大纲为空")
	}
	targetID := p.Sections[0].ID
	got, err := s.GenerateSection(context.Background(), p.ID, targetID, "详细撰写")
	if err != nil {
		t.Fatalf("GenerateSection: %v", err)
	}
	// 章节应有内容且状态 completed
	for _, sec := range got.Sections {
		if sec.ID == targetID {
			if sec.Content == "" {
				t.Error("章节内容为空")
			}
			if sec.Status != "completed" {
				t.Errorf("Status = %q, want completed", sec.Status)
			}
		}
	}
}

func TestParseBidFileV2_WithSources(t *testing.T) {
	ai := &mockAI{replies: map[string]string{
		"招标文件": `{
  "overview": "本项目为污染场地修复",
  "overviewQuote": "本项目为污染场地修复工程",
  "duration": "90 日历天",
  "durationQuote": "工期要求：90 日历天",
  "qualification": [{"name":"环保工程专业承包资质","content":"需具备环保工程专业承包三级及以上","quote":"环保工程专业承包三级及以上"}],
  "techScoring": [{"name":"施工方案","maxScore":"20","requirement":"方案完整合理","quote":"施工方案 20 分"}],
  "keyRequirements": ["项目经理须常驻现场"],
  "redLines": [{"name":"废标条款","content":"投标文件未按要求签字盖章","quote":"未按要求签字盖章"}],
  "format": [{"name":"装订要求","content":"A4 双面打印","quote":"A4 双面打印"}],
  "darkRules": [{"name":"暗标要求","content":"不得出现单位名称","quote":"不得出现单位名称"}]
}`,
	}}
	s := newServiceAt(t, t.TempDir(), ai)
	p, _ := s.Create("方案", "blank", "", "其他")
	// 注入一个已转换的招标文件
	fileID, err := s.store.AddFile(p.ID, "tender", "招标文件.pdf", "x.pdf", 100)
	if err != nil {
		t.Fatal(err)
	}
	p.BidSummary = &BidSummary{
		RawFiles: []FileDoc{{
			FileID: fileID, Name: "招标文件.pdf", Path: "x.pdf",
			Markdown: "一、项目概况\n本项目为污染场地修复工程。\n二、工期要求\n工期要求：90 日历天。\n三、资质要求\n需具备环保工程专业承包三级及以上资质。\n四、评分标准\n施工方案 20 分。\n五、其他\n投标文件未按要求签字盖章作废标处理；A4 双面打印；暗标不得出现单位名称。",
		}},
		RawMarkdown: "一、项目概况\n本项目为污染场地修复工程。",
	}
	if err := s.store.Update(p); err != nil {
		t.Fatal(err)
	}

	got, err := s.ParseBidFile(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ParseBidFile: %v", err)
	}
	if got.BidSummary.ParseStatus != "done" {
		t.Errorf("ParseStatus = %q", got.BidSummary.ParseStatus)
	}
	if len(got.BidSummary.Qualification) != 1 || len(got.BidSummary.Qualification[0].Sources) == 0 {
		t.Fatalf("资质来源缺失: %+v", got.BidSummary.Qualification)
	}
	src := got.BidSummary.Qualification[0].Sources[0]
	if src.Snippet == "" || src.Confidence <= 0 {
		t.Errorf("来源异常: %+v", src)
	}
	if got.BidSummary.TechScoring[0].Sources[0].Page != 0 {
		t.Errorf("页码异常: %+v", got.BidSummary.TechScoring[0].Sources)
	}
	rows, err := s.store.ListParseResults(p.ID)
	if err != nil || len(rows) == 0 {
		t.Fatalf("parse_results 未落库: %v %+v", err, rows)
	}
}

// ─── 模板 ────────────────────────────────────────────────────

func TestListTemplates(t *testing.T) {
	s := newTestService(t)
	templates := s.ListTemplates()
	if len(templates) == 0 {
		t.Fatal("模板列表为空")
	}
	for _, tmpl := range templates {
		if tmpl.ID == "" || tmpl.Name == "" {
			t.Errorf("模板字段缺失: %+v", tmpl)
		}
		// blank 模板是空模板特例，其余模板应有预定义章节
		if tmpl.ID != "blank" && len(tmpl.Sections) == 0 {
			t.Errorf("模板 %s 无章节", tmpl.ID)
		}
	}
}
