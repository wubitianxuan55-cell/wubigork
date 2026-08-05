// Package proposal — AI 生成服务
package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	officedb "github.com/gaea/gaea/internal/office/db"
	"github.com/google/uuid"
)

// AIClient AI 调用接口（解耦，由 app 层注入）
type AIClient interface {
	ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error)
}

// Service 方案服务（业务逻辑 + AI）
type Service struct {
	store *Store
	ai    AIClient
}

// NewService 创建服务实例（打开 office.db，执行旧数据迁移）
func NewService(dataRoot string, ai AIClient) *Service {
	officeDir := filepath.Join(dataRoot, "office")
	dbs := officedb.GetDatabase(officeDir)
	if dbs == nil {
		return &Service{store: nil, ai: ai}
	}
	st := NewStore(dbs, officeDir)
	if _, err := MigrateLegacyJSON(st); err != nil {
		log.Printf("[proposal] 旧数据迁移失败: %v", err)
	}
	return &Service{
		store: st,
		ai:    ai,
	}
}

// Store 暴露存储层
func (s *Service) Store() *Store { return s.store }

// ─── 模板 ────────────────────────────────────────────────

// ListTemplates 列出所有模板
func (s *Service) ListTemplates() []Template {
	return DefaultTemplates
}

// ─── 项目 ────────────────────────────────────────────────

// CreateProject 新建项目
func (s *Service) CreateProject(name, category, client string) (*Project, error) {
	if s.store == nil {
		return nil, fmt.Errorf("方案存储不可用")
	}
	return s.store.CreateProject(name, category, client)
}

// ListProjects 列出全部项目
func (s *Service) ListProjects() ([]Project, error) {
	if s.store == nil {
		return nil, fmt.Errorf("方案存储不可用")
	}
	return s.store.ListProjects()
}

// DeleteProject 删除项目
func (s *Service) DeleteProject(id string) error {
	if s.store == nil {
		return fmt.Errorf("方案存储不可用")
	}
	return s.store.DeleteProject(id)
}

// ─── CRUD ────────────────────────────────────────────────

// Create 创建方案（projectID 为空时挂到「未归档项目」）
func (s *Service) Create(title, templateID, requirements, category string, projectID ...string) (*Proposal, error) {
	if s.store == nil {
		return nil, fmt.Errorf("方案存储不可用")
	}
	pid := ""
	if len(projectID) > 0 {
		pid = projectID[0]
	}
	if pid == "" {
		proj, err := s.store.EnsureDefaultProject()
		if err != nil {
			return nil, err
		}
		pid = proj.ID
	}
	tmpl := GetTemplate(templateID)
	if tmpl == nil {
		tmpl = &DefaultTemplates[len(DefaultTemplates)-1]
	}
	if title == "" {
		title = "未命名方案"
	}
	sections := SectionsFromTemplate(tmpl)
	return s.store.Create(title, tmpl.ID, requirements, category, pid, sections)
}

// List 列出所有方案
func (s *Service) List() ([]Proposal, error) {
	return s.store.List()
}

// Get 获取方案
func (s *Service) Get(id string) (*Proposal, error) {
	return s.store.Get(id)
}

// Update 更新方案
func (s *Service) Update(p *Proposal) error {
	return s.store.Update(p)
}

// Delete 删除方案
func (s *Service) Delete(id string) error {
	return s.store.Delete(id)
}

// UpdateSection 更新单个章节内容
func (s *Service) UpdateSection(proposalID, sectionID, title, content string) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	found := false
	for _, sec := range flattenSections(p.Sections) {
		if sec.ID == sectionID {
			found = true
			if title != "" {
				sec.Title = title
			}
			sec.Content = content
			sec.Status = "completed"
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// ─── AI 生成 ─────────────────────────────────────────────

// GenerateOutline AI 生成方案大纲
func (s *Service) GenerateOutline(ctx context.Context, proposalID, requirements string) (*Proposal, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}

	systemPrompt := enrichSoilPrompt(p.Template, `你是一位专业方案撰写顾问。根据用户需求和招标要求，生成三级结构化方案大纲。

返回纯 JSON，格式为：
{"title":"方案标题","sections":[
  {"title":"第1章 章节名","level":1,"children":[
    {"title":"1.1 节名","level":2,"children":[
      {"title":"1.1.1 小节名","level":3},
      {"title":"1.1.2 小节名","level":3}
    ]},
    {"title":"1.2 节名","level":2}
  ]},
  {"title":"第2章 章节名","level":1,"children":[...]}
]}

要求：
- 第1级（章）：5-10章，逻辑递进
- 第2级（节）：每章2-5节
- 第3级（小节）：每节1-3个小节（可选）
- 标题简洁有力
- 如果提供了评分标准，大纲必须覆盖所有评分项`)

	userMsg := fmt.Sprintf("模板类型：%s\n需求描述：%s", p.Template, requirements)
	if p.BidSummary != nil {
		if len(p.BidSummary.TechScoring) > 0 {
			userMsg += "\n\n【评分标准（大纲必须覆盖）】"
			for _, item := range p.BidSummary.TechScoring {
				userMsg += fmt.Sprintf("\n- %s（%s分）：%s", item.Name, item.MaxScore, item.Requirement)
			}
		}
		if len(p.BidSummary.KeyRequirements) > 0 {
			userMsg += "\n\n【核心要求】"
			for _, req := range p.BidSummary.KeyRequirements {
				userMsg += "\n- " + req
			}
		}
		if p.BidSummary.Overview != "" {
			userMsg += "\n\n【项目概况】" + p.BidSummary.Overview
		}
	}
	reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %w", err)
	}

	// 解析 AI 返回的 JSON（支持树形三级大纲）
	reply = extractJSON(reply)
	var outline struct {
		Title    string `json:"title"`
		Sections []struct {
			Title    string `json:"title"`
			Level    int    `json:"level"`
			Children []struct {
				Title    string `json:"title"`
				Level    int    `json:"level"`
				Children []struct {
					Title string `json:"title"`
					Level int    `json:"level"`
				} `json:"children"`
			} `json:"children"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(reply), &outline); err != nil {
		return nil, fmt.Errorf("解析 AI 输出失败: %w\n原始输出: %s", err, truncate(reply, 500))
	}

	p.Title = outline.Title
	p.Requirements = requirements
	p.Status = "writing"

	// 构建三级章节树
	var newSections []ProposalSection
	idx := 0
	for _, ch := range outline.Sections {
		chSec := ProposalSection{
			ID: uuid.New().String(), ProposalID: proposalID,
			Index: idx, Level: ch.Level, Title: ch.Title, Status: "pending",
		}
		if ch.Level == 0 {
			chSec.Level = 1
		}
		idx++
		for _, sec := range ch.Children {
			secLevel := sec.Level
			if secLevel == 0 {
				secLevel = 2
			}
			subSec := ProposalSection{
				ID: uuid.New().String(), ProposalID: proposalID,
				ParentID: chSec.ID, Index: idx, Level: secLevel,
				Title: sec.Title, Status: "pending",
			}
			idx++
			for _, sub := range sec.Children {
				subLevel := sub.Level
				if subLevel == 0 {
					subLevel = 3
				}
				subSec.Children = append(subSec.Children, ProposalSection{
					ID: uuid.New().String(), ProposalID: proposalID,
					ParentID: subSec.ID, Index: idx, Level: subLevel,
					Title: sub.Title, Status: "pending",
				})
				idx++
			}
			chSec.Children = append(chSec.Children, subSec)
		}
		newSections = append(newSections, chSec)
	}
	p.Sections = newSections
	p.UpdatedAt = now()

	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// GenerateSection AI 撰写章节内容
func (s *Service) GenerateSection(ctx context.Context, proposalID, sectionID, instruction string) (*Proposal, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}

	// 找到目标章节（支持任意层级）
	var targetSec *ProposalSection
	var prevContent string
	for _, sec := range flattenSections(p.Sections) {
		if sec.ID == sectionID {
			targetSec = sec
		} else if targetSec == nil && sec.Content != "" {
			prevContent = sec.Content
		}
	}
	if targetSec == nil {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}

	// 构建上下文：收集已完成章节内容
	var contextParts []string
	contextParts = append(contextParts, fmt.Sprintf("方案标题：%s", p.Title))
	contextParts = append(contextParts, fmt.Sprintf("方案类型：%s", p.Template))
	contextParts = append(contextParts, fmt.Sprintf("需求描述：%s", p.Requirements))

	// 加入大纲（含全部层级与完成标记）
	contextParts = append(contextParts, "方案大纲：")
	var walk func(ss []ProposalSection, depth int)
	walk = func(ss []ProposalSection, depth int) {
		for _, sec := range ss {
			marker := ""
			if sec.Status == "completed" {
				marker = " ✓"
			}
			contextParts = append(contextParts, fmt.Sprintf("%s%d. %s%s", strings.Repeat("  ", depth), sec.Index+1, sec.Title, marker))
			walk(sec.Children, depth+1)
		}
	}
	walk(p.Sections, 0)

	// 加入前一章内容（如有）
	if prevContent != "" {
		contextParts = append(contextParts, "\n前一章节内容参考：\n"+truncate(prevContent, 1500))
	}

	// Layer 1: 注入投标要点（招标文件解析结果）
	if p.BidSummary != nil {
		if len(p.BidSummary.TechScoring) > 0 {
			contextParts = append(contextParts, "\n【招标评分标准】")
			for _, item := range p.BidSummary.TechScoring {
				contextParts = append(contextParts, fmt.Sprintf("- %s（%s分）：%s", item.Name, item.MaxScore, item.Requirement))
			}
		}
		if len(p.BidSummary.KeyRequirements) > 0 {
			contextParts = append(contextParts, "\n【核心要求】")
			for _, req := range p.BidSummary.KeyRequirements {
				contextParts = append(contextParts, "- "+req)
			}
		}
		if p.BidSummary.Duration != "" {
			contextParts = append(contextParts, "\n【工期】"+p.BidSummary.Duration)
		}
		if p.BidSummary.Overview != "" {
			contextParts = append(contextParts, "\n【项目概况】"+p.BidSummary.Overview)
		}
	}

	systemPrompt := enrichSoilPrompt(p.Template, fmt.Sprintf(`你是一位专业的环保工程投标方案撰写专家，精通土壤修复领域。现在撰写投标技术方案中的「%s」章节。
要求：
- 语言专业、条理清晰，符合投标文件规范
- 使用 Markdown 格式
- 字数 500-1500 字（核心章节应更详细）
- 紧扣本章标题和招标评分标准
- 尽可能引用场地污染数据，体现实地调研深度
- 如果上下文中有评分标准，本章应尽量覆盖相关评分项
- 对于技术描述，引用 HJ 25.4 等规范标准

直接输出章节正文，不需要标题。`, targetSec.Title))

	reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, strings.Join(contextParts, "\n"))
	if err != nil {
		return nil, fmt.Errorf("AI 生成失败: %w", err)
	}

	targetSec.Content = reply
	targetSec.Status = "completed"
	p.UpdatedAt = now()

	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Polish 润色/加工章节内容
func (s *Service) Polish(ctx context.Context, proposalID, sectionID, content, operation string) (*Proposal, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}

	var targetSec *ProposalSection
	for _, sec := range flattenSections(p.Sections) {
		if sec.ID == sectionID {
			targetSec = sec
			break
		}
	}
	if targetSec == nil {
		return nil, fmt.Errorf("章节未找到: %s", sectionID)
	}
	if content == "" {
		content = targetSec.Content
	}

	var actionDesc string
	switch operation {
	case "polish":
		actionDesc = "润色优化以下内容，使其更专业流畅"
	case "expand":
		actionDesc = "扩写以下内容，增加细节和深度，扩充到原来的1.5-2倍"
	case "shorten":
		actionDesc = "精简以下内容，保留核心要点，压缩到原来的一半"
	case "summarize":
		actionDesc = "总结以下内容的要点，用简洁的列表形式"
	default:
		actionDesc = "根据以下要求优化内容：" + operation
	}

	systemPrompt := fmt.Sprintf(`你是一位专业文档编辑。请%s。
直接输出处理后的内容，不需要额外说明。`, actionDesc)

	reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, content)
	if err != nil {
		return nil, fmt.Errorf("AI 处理失败: %w", err)
	}

	targetSec.Content = reply
	p.UpdatedAt = now()

	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// ─── 投标方案增强 ──────────────────────────────────────────

// ParseBidFile 基于已转换的 Markdown 招标文件，AI 提取结构化摘要
func (s *Service) ParseBidFile(ctx context.Context, proposalID string) (*Proposal, error) {
	if s.ai == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}

	rawMD := ""
	if p.BidSummary != nil {
		rawMD = p.BidSummary.RawMarkdown
	}
	if rawMD == "" {
		return nil, fmt.Errorf("请先导入招标文件（选择 PDF/Word → 自动转换为 Markdown），再点击 AI 解析")
	}

	systemPrompt := `你是一位专业的招投标专家。请基于以下招标文件（已转为 Markdown 格式），提取所有对投标方案编写有影响的关键信息。

返回纯 JSON：
{
  "techScoring": [{"name":"评分项","maxScore":"分值","requirement":"具体要求"}],
  "keyRequirements": ["核心要求"],
  "duration": "工期要求",
  "redLines": ["废标条款/否决项"],
  "overview": "项目概况（200字以内）",
  "extra": {"分类":"内容"}
}

要求：
- 不要遗漏任何影响投标的关键信息
- extra 字段自行命名分类，灵活发挥
- 不存在的类别填空数组或空字符串即可`

	const chunkSize = 15000
	rawRunes := []rune(rawMD)

	if len(rawRunes) <= chunkSize {
		userMsg := "请解析以下招标文件（已转为 Markdown 格式）：\n\n" + rawMD
		reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, userMsg)
		if err != nil {
			return nil, fmt.Errorf("AI 解析失败: %w", err)
		}
		return s.applyBidSummary(p, reply)
	}

	// 大文件：分块提取
	var allResults []string
	for start := 0; start < len(rawRunes); start += chunkSize {
		end := start + chunkSize
		if end > len(rawRunes) {
			end = len(rawRunes)
		}
		chunk := string(rawRunes[start:end])
		chunkPrompt := fmt.Sprintf("请解析以下招标文件（第%d-%d字，共%d字）：\n\n%s\n\n请提取本段中的所有关键信息，返回同样格式的JSON。",
			start+1, end, len(rawRunes), chunk)
		reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, chunkPrompt)
		if err != nil {
			continue
		}
		allResults = append(allResults, extractJSON(reply))
	}

	if len(allResults) == 1 {
		return s.applyBidSummary(p, allResults[0])
	}
	mergePrompt := fmt.Sprintf("以下是招标文件分段解析的%d个结果，请合并去重：\n\n%s\n\n返回合并后的完整 JSON。",
		len(allResults), strings.Join(allResults, "\n---\n"))
	reply, err := s.ai.ChatSimpleStream(ctx, "", systemPrompt, mergePrompt)
	if err != nil {
		return nil, fmt.Errorf("AI 汇总失败: %w", err)
	}
	return s.applyBidSummary(p, reply)
}
func (s *Service) applyBidSummary(p *Proposal, reply string) (*Proposal, error) {
	reply = extractJSON(reply)
	var bs BidSummary
	if err := json.Unmarshal([]byte(reply), &bs); err != nil {
		return nil, fmt.Errorf("解析 AI 输出失败: %w\n原始输出: %s", err, truncate(reply, 500))
	}
	bs.RawMarkdown = p.BidSummary.RawMarkdown // 保留已转换的 Markdown
	p.BidSummary = &bs
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// SaveRawText 追加文件：转换→存入RawFiles→合并RawMarkdown
func (s *Service) SaveRawText(proposalID, filePath string) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}

	markdown, err := ConvertToMarkdown(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件转换失败: %w", err)
	}

	if p.BidSummary == nil {
		p.BidSummary = &BidSummary{}
	}

	// 追加文件
	info, _ := os.Stat(filePath)
	size := 0
	if info != nil {
		size = int(info.Size())
	}
	p.BidSummary.RawFiles = append(p.BidSummary.RawFiles, FileDoc{
		Name: filepath.Base(filePath), Markdown: markdown, Size: size,
	})

	// 合并所有文件的 Markdown
	var merged strings.Builder
	merged.WriteString(fmt.Sprintf("# 招标文件汇编\n\n> 共 %d 个文件\n\n", len(p.BidSummary.RawFiles)))
	for i, f := range p.BidSummary.RawFiles {
		merged.WriteString(fmt.Sprintf("## 文件 %d：%s\n\n", i+1, f.Name))
		merged.WriteString(f.Markdown)
		merged.WriteString("\n\n---\n\n")
	}
	p.BidSummary.RawMarkdown = merged.String()
	p.BidSummary.RawText = merged.String()
	p.UpdatedAt = now()

	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// RemoveRawFile 删除已上传的文件
func (s *Service) RemoveRawFile(proposalID string, index int) (*Proposal, error) {
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, err
	}
	if p.BidSummary == nil || index < 0 || index >= len(p.BidSummary.RawFiles) {
		return nil, fmt.Errorf("原始文件索引越界: %d", index)
	}
	if f := p.BidSummary.RawFiles[index]; f.Path != "" {
		_ = os.Remove(f.Path)
	}
	p.BidSummary.RawFiles = append(p.BidSummary.RawFiles[:index], p.BidSummary.RawFiles[index+1:]...)

	// 重新合并
	var merged strings.Builder
	merged.WriteString(fmt.Sprintf("# 招标文件汇编\n\n> 共 %d 个文件\n\n", len(p.BidSummary.RawFiles)))
	for i, f := range p.BidSummary.RawFiles {
		merged.WriteString(fmt.Sprintf("## 文件 %d：%s\n\n", i+1, f.Name))
		merged.WriteString(f.Markdown)
		merged.WriteString("\n\n---\n\n")
	}
	p.BidSummary.RawMarkdown = merged.String()
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) CheckCoverage(ctx context.Context, proposalID string) (*Proposal, []CoverageResult, error) {
	if s.ai == nil {
		return nil, nil, fmt.Errorf("AI 客户端未初始化")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, nil, err
	}
	if p.BidSummary == nil || len(p.BidSummary.TechScoring) == 0 {
		return nil, nil, fmt.Errorf("未解析招标文件，请先上传并解析招标文件")
	}

	// 收集所有章节内容
	var allContent strings.Builder
	for _, sec := range flattenSections(p.Sections) {
		if sec.Content != "" {
			allContent.WriteString(fmt.Sprintf("【%s】\n%s\n\n", sec.Title, sec.Content))
		}
	}
	var scoringList strings.Builder
	for _, item := range p.BidSummary.TechScoring {
		scoringList.WriteString(fmt.Sprintf("- %s（%s分）：%s\n", item.Name, item.MaxScore, item.Requirement))
	}
	sp := fmt.Sprintf("你是环保工程投标评审专家。对照评分标准检查方案：\n%s\n返回 JSON: [{\"name\":\"\",\"maxScore\":\"\",\"covered\":\"full|partial|none\",\"score\":\"\",\"suggestion\":\"\"}]", scoringList.String())
	reply, err := s.ai.ChatSimpleStream(ctx, "", sp, allContent.String())
	if err != nil {
		return nil, nil, fmt.Errorf("AI 检查失败: %w", err)
	}
	reply = extractJSON(reply)
	var results []CoverageResult
	if err := json.Unmarshal([]byte(reply), &results); err != nil {
		return nil, nil, fmt.Errorf("解析失败: %w", err)
	}
	return p, results, nil
}

// CoverageResult 覆盖检查结果
type CoverageResult struct {
	Name       string `json:"name"`
	MaxScore   string `json:"maxScore"`
	Covered    string `json:"covered"`
	Score      string `json:"score"`
	Suggestion string `json:"suggestion"`
}
