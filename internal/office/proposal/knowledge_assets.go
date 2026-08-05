// Package proposal — 记忆中枢知识资产接入（规范/素材/历史方案）
package proposal

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/specdata"
)

// SetKnowledgeStoreForTest 覆盖知识库存储（测试隔离）
func (s *Service) SetKnowledgeStoreForTest(st *knowledge.Store) { s.kb = st }

// knowledgeStore 返回可用知识库存储；未注入时走全局单例，失败返回 nil
func (s *Service) knowledgeStore() *knowledge.Store {
	if s.kb != nil {
		return s.kb
	}
	st, err := func() (st *knowledge.Store, err error) {
		defer func() {
			if r := recover(); r != nil {
				st, err = nil, fmt.Errorf("知识库不可用: %v", r)
			}
		}()
		return knowledge.Global().Store()
	}()
	if err != nil || st == nil {
		return nil
	}
	return st
}

// EnsureSpecAssets 把内置规范索引与土壤修复技术知识幂等写入记忆中枢
func (s *Service) EnsureSpecAssets() error {
	st := s.knowledgeStore()
	if st == nil {
		return fmt.Errorf("知识库不可用")
	}
	for _, e := range specdata.SpecLibrary() {
		name := "spec-" + slugSpec(e.Code+"-"+e.Clause)
		body := e.Content
		if e.Explanation != "" {
			body += "\n\n💡 解释：" + e.Explanation
		}
		entry := knowledge.Entry{
			Name: name, Title: fmt.Sprintf("%s %s %s", e.Code, e.Clause, e.Title),
			Category: knowledge.CatStandard, Tags: []string{e.Category},
			Status: "已发布", Source: e.Code, Body: body,
		}
		if err := st.Save(entry); err != nil {
			return err
		}
	}
	// 土壤修复通用技术知识
	soil := knowledge.Entry{
		Name: "soil-remediation-tech", Title: "土壤修复常用技术与规范标准",
		Category: knowledge.CatExperience, Tags: []string{"土壤修复", "技术比选"},
		Status: "已发布", Source: "gaea 内置", Body: SoilRemediationKB,
	}
	return st.Save(soil)
}

func slugSpec(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// SpecRef 规范检索结果
type SpecRef struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	Code  string `json:"code"`
	Body  string `json:"body"`
}

// SearchSpecs 在记忆中枢检索规范条文（Top 8）
func (s *Service) SearchSpecs(query string) []SpecRef {
	st := s.knowledgeStore()
	if st == nil || strings.TrimSpace(query) == "" {
		return nil
	}
	tokens := strings.Fields(query)
	var candidates []knowledge.Entry
	for _, e := range st.ReadAll() {
		if e.Category != knowledge.CatStandard {
			continue
		}
		if specScore(e, tokens) > 0 {
			candidates = append(candidates, e)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return specScore(candidates[i], tokens) > specScore(candidates[j], tokens)
	})
	var out []SpecRef
	for _, e := range candidates {
		if len(out) >= 8 {
			break
		}
		out = append(out, SpecRef{Name: e.Name, Title: e.Title, Code: e.Source, Body: e.Body})
	}
	return out
}

// specScore 按查询令牌重排：标题命中 3 分/次，正文命中 1 分/次
func specScore(e knowledge.Entry, tokens []string) int {
	score := 0
	for _, tok := range tokens {
		score += strings.Count(e.Title, tok) * 3
		score += strings.Count(e.Body, tok)
	}
	return score
}

// officeAssetsCategory 素材库在记忆中枢中的分类
const officeAssetsCategory = "素材库"

// AssetRef 素材条目
type AssetRef struct {
	Name  string   `json:"name"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Body  string   `json:"body"`
}

// AddAsset 新增素材（业绩/人员/设备/常用段落）
func (s *Service) AddAsset(title string, tags []string, body string) error {
	st := s.knowledgeStore()
	if st == nil {
		return fmt.Errorf("知识库不可用")
	}
	if title == "" || body == "" {
		return fmt.Errorf("title 与 body 不能为空")
	}
	return st.Save(knowledge.Entry{
		Name:     fmt.Sprintf("asset-%s-%d", slugSpec(title), time.Now().UnixNano()),
		Title:    title,
		Category: officeAssetsCategory,
		Tags:     tags,
		Status:   "已发布",
		Body:     body,
	})
}

// ListAssets 列出全部素材
func (s *Service) ListAssets() []AssetRef {
	st := s.knowledgeStore()
	if st == nil {
		return nil
	}
	var out []AssetRef
	for _, e := range st.ReadAll() {
		if e.Category != officeAssetsCategory {
			continue
		}
		out = append(out, AssetRef{Name: e.Name, Title: e.Title, Tags: e.Tags, Body: e.Body})
	}
	return out
}

// SearchAssets 检索素材（query 与 tag 可空）
func (s *Service) SearchAssets(query, tag string) []AssetRef {
	st := s.knowledgeStore()
	if st == nil {
		return nil
	}
	filter := knowledge.Filter{Category: officeAssetsCategory, Tag: tag}
	var out []AssetRef
	for _, e := range knowledge.Search(st, query, filter) {
		out = append(out, AssetRef{Name: e.Name, Title: e.Title, Tags: e.Tags, Body: e.Body})
	}
	return out
}

// RemoveAsset 删除素材
func (s *Service) RemoveAsset(name string) error {
	st := s.knowledgeStore()
	if st == nil {
		return fmt.Errorf("知识库不可用")
	}
	return st.Delete(name)
}

// ArchiveProposal 把方案装配全文归档到记忆中枢（幂等覆盖），返回条目名
func (s *Service) ArchiveProposal(proposalID string) (string, error) {
	st := s.knowledgeStore()
	if st == nil {
		return "", fmt.Errorf("知识库不可用")
	}
	p, err := s.store.Get(proposalID)
	if err != nil {
		return "", err
	}
	name := "proposal-" + p.ID
	entry := knowledge.Entry{
		Name:     name,
		Title:    "【历史方案】" + p.Title,
		Category: knowledge.CatDesign,
		Tags:     []string{"legacy-proposal", p.Category, p.Template},
		Status:   "已发布",
		Source:   p.Category,
		Body:     Assemble(p),
	}
	if err := st.Save(entry); err != nil {
		return "", err
	}
	return name, nil
}

// SearchLegacyProposals 检索历史方案
func (s *Service) SearchLegacyProposals(query string) []AssetRef {
	st := s.knowledgeStore()
	if st == nil {
		return nil
	}
	var out []AssetRef
	for _, e := range knowledge.Search(st, query, knowledge.Filter{Category: knowledge.CatDesign, Tag: "legacy-proposal"}) {
		out = append(out, AssetRef{Name: e.Name, Title: e.Title, Tags: e.Tags, Body: e.Body})
	}
	return out
}

// legacyRefFor 返回同模板/同分类历史方案的参考摘要（最多 600 字）
func (s *Service) legacyRefFor(template, category string) string {
	st := s.knowledgeStore()
	if st == nil {
		return ""
	}
	for _, e := range st.ReadAll() {
		if e.Category != knowledge.CatDesign || !hasTag(e.Tags, "legacy-proposal") {
			continue
		}
		if template != "" && hasTag(e.Tags, template) {
			return truncate(e.Body, 600)
		}
		if category != "" && hasTag(e.Tags, category) {
			return truncate(e.Body, 600)
		}
	}
	return ""
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
