// Package proposal — AI 生成服务
package proposal

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	officedb "github.com/gaea/gaea/internal/office/db"
)

// AIClient AI 调用接口（解耦，由 app 层注入）
type AIClient interface {
	ChatSimpleStream(ctx context.Context, model, systemPrompt, userMsg string) (string, error)
}

// Service 方案服务（业务逻辑 + AI）
type Service struct {
	store *Store
	ai    AIClient
	kb    *knowledge.Store
	ocr   OCRProvider
}

// NewService 创建服务实例（打开 office.db，执行旧数据迁移）
func NewService(dataRoot string, ai AIClient) *Service {
	officeDir := filepath.Join(dataRoot, "office")
	dbs := officedb.GetDatabase(officeDir)
	if dbs == nil {
		return &Service{store: nil, ai: ai}
	}
	st := NewStore(dbs, officeDir)
	if err := st.SeedTemplates(); err != nil {
		log.Printf("[proposal] 模板初始化失败: %v", err)
	}
	if err := st.SeedDarkRules(); err != nil {
		log.Printf("[proposal] 暗标规则初始化失败: %v", err)
	}
	if _, err := st.EnsureDefaultProject(); err != nil {
		log.Printf("[proposal] 初始化默认项目失败: %v", err)
	}
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
	if s.store != nil {
		if list, err := s.store.ListTemplatesDB(); err == nil && len(list) > 0 {
			return list
		}
	}
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

// GetProjectFacts 读取项目事实
func (s *Service) GetProjectFacts(projectID string) (map[string]string, error) {
	if s.store == nil {
		return nil, fmt.Errorf("方案存储不可用")
	}
	return s.store.GetProjectFacts(projectID)
}

// SaveProjectFacts 保存项目事实
func (s *Service) SaveProjectFacts(projectID string, facts map[string]string) error {
	if s.store == nil {
		return fmt.Errorf("方案存储不可用")
	}
	return s.store.SaveProjectFacts(projectID, facts)
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
// GenerateSection AI 撰写章节内容
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

// ParseBidFile 执行结构化解析管线（v2）
func (s *Service) ParseBidFile(ctx context.Context, proposalID string) (*Proposal, error) {
	return s.parseBidFile(ctx, proposalID, nil)
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

// CheckAll 运行全部校验规则，返回统一检查报告
func (s *Service) CheckAll(ctx context.Context, proposalID string) (*Proposal, []CheckItem, error) {
	if s.store == nil {
		return nil, nil, fmt.Errorf("方案存储不可用")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return nil, nil, err
	}
	items := RunChecks(ctx, p, s.defaultRules())
	summary := &CheckSummary{Total: len(items), UpdatedAt: now()}
	for _, it := range items {
		if it.Status == "fail" || it.Status == "error" {
			summary.Failed++
		} else if it.Status == "warn" {
			summary.Warnings++
		}
	}
	p.CheckSummary = summary
	ensureReviewChecklist(p)
	p.advanceStage(StageCheck)
	p.UpdatedAt = now()
	if err := s.store.Update(p); err != nil {
		return nil, nil, err
	}
	return p, items, nil
}

// ensureReviewChecklist 为空时补默认单人复核清单
func ensureReviewChecklist(p *Proposal) {
	if len(p.ReviewChecklist) > 0 {
		return
	}
	p.ReviewChecklist = []ReviewItem{
		{ID: "r1", Label: "废标条款逐条核对"},
		{ID: "r2", Label: "工期要求一致"},
		{ID: "r3", Label: "评分标准覆盖"},
		{ID: "r4", Label: "暗标格式合规"},
		{ID: "r5", Label: "规范引用准确"},
		{ID: "r6", Label: "签字盖章复核"},
	}
}

// CheckCoverage 语义覆盖检查（兼容旧绑定，内部走覆盖规则）
func (s *Service) CheckCoverage(ctx context.Context, proposalID string) (*Proposal, []CoverageResult, error) {
	p, items, err := s.CheckAll(ctx, proposalID)
	if err != nil {
		return nil, nil, err
	}
	var out []CoverageResult
	for _, it := range items {
		if it.Rule != "评分覆盖检查" {
			continue
		}
		out = append(out, CoverageResult{Name: it.Message, Covered: it.Status, Suggestion: it.Evidence})
	}
	return p, out, nil
}

// CoverageResult 覆盖检查结果
type CoverageResult struct {
	Name       string `json:"name"`
	MaxScore   string `json:"maxScore"`
	Covered    string `json:"covered"`
	Score      string `json:"score"`
	Suggestion string `json:"suggestion"`
}
