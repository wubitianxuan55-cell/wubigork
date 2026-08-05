// Package proposal — 方案编写模块数据类型
package proposal

import "time"

// Proposal 方案文档
type Proposal struct {
	ID           string            `json:"id"`
	ProjectID    string            `json:"projectId"`
	Title        string            `json:"title"`
	Category     string            `json:"category"` // 分类（环保工程/市政工程/水利工程/其他）
	Template     string            `json:"template"`
	Requirements string            `json:"requirements"`
	BidSummary   *BidSummary       `json:"bidSummary,omitempty"`
	Status       string            `json:"status"`
	Version      int               `json:"version"`
	Sections     []ProposalSection `json:"sections"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}
type ScoringItem struct {
	Name        string      `json:"name"`
	MaxScore    string      `json:"maxScore"`
	Requirement string      `json:"requirement"`
	Sources     []SourceRef `json:"sources,omitempty"`
}

type BidSummary struct {
	TechScoring     []ScoringItem     `json:"techScoring"`
	KeyRequirements []string          `json:"keyRequirements"`
	Duration        string            `json:"duration"`
	RedLines        []string          `json:"redLines"`
	Overview        string            `json:"overview"`
	Extra           map[string]string `json:"extra"`
	RawMarkdown     string            `json:"rawMarkdown"`             // 合并后的 Markdown 全文（AI解析用）
	RawFiles        []FileDoc         `json:"rawFiles"`                // 多文件列表（上传的原始文件+转换结果）
	RawText         string            `json:"rawText"`                 // 兼容旧版
	Qualification   []BidItem         `json:"qualification,omitempty"` // 资质要求
	Format          []BidItem         `json:"format,omitempty"`        // 格式要求
	DarkRules       []BidItem         `json:"darkRules,omitempty"`     // 暗标要求
	RedLineItems    []BidItem         `json:"redLineItems,omitempty"`  // 废标条款（带来源）
	OverviewSources []SourceRef       `json:"overviewSources,omitempty"`
	DurationSources []SourceRef       `json:"durationSources,omitempty"`
	ParseStatus     string            `json:"parseStatus,omitempty"` // none|done|partial
	TotalWords      int               `json:"totalWords,omitempty"`  // 招标文件要求的正文字数（0=未要求）
}

type FileDoc struct {
	FileID   string `json:"fileId"`   // files 表 ID（旧数据为空时用 file-<index>）
	Name     string `json:"name"`     // 文件名
	Path     string `json:"path"`     // 文件路径（供后端转换用）
	Markdown string `json:"markdown"` // 转换后的 Markdown（空=未转换）
	Size     int    `json:"size"`     // 原始大小（字节）
}

type ProposalSection struct {
	ID         string            `json:"id"`
	ProposalID string            `json:"proposalId"`
	ParentID   string            `json:"parentId"` // 父节点ID（空=顶级章）
	Index      int               `json:"index"`
	Level      int               `json:"level"` // 1=章 2=节 3=小节
	Title      string            `json:"title"`
	Content    string            `json:"content"`
	Status     string            `json:"status"` // pending | writing | completed
	Sources    string            `json:"sources,omitempty"`
	WordTarget int               `json:"wordTarget,omitempty"` // 字数目标（叶子节点）
	Words      int               `json:"words,omitempty"`      // 当前字数
	Children   []ProposalSection `json:"children,omitempty"`   // 前端树形展示用
}

// Template 方案模板
type Template struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Sections    []string `json:"sections"`
}

// GenerateOutlineInput AI 大纲生成输入
type GenerateOutlineInput struct {
	ProposalID   string `json:"proposalId"`
	Requirements string `json:"requirements"`
	Template     string `json:"template"`
}

// GenerateSectionInput AI 章节生成输入
type GenerateSectionInput struct {
	ProposalID  string `json:"proposalId"`
	SectionID   string `json:"sectionId"`
	Instruction string `json:"instruction"`
}

// PolishInput 润色输入
type PolishInput struct {
	ProposalID  string `json:"proposalId"`
	SectionID   string `json:"sectionId"`
	Content     string `json:"content"`
	Instruction string `json:"instruction"` // polish | expand | shorten | summarize
}

func now() string { return time.Now().Format("2006-01-02 15:04:05") }

// Project 投标项目/标段
type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Client    string `json:"client"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ParseResultItem 解析结果行（parse_results 表）
type ParseResultItem struct {
	FileID     string  `json:"fileId"`
	Field      string  `json:"field"`
	Value      string  `json:"value"`
	Page       int     `json:"page"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Snippet    string  `json:"snippet"`
	Confidence float64 `json:"confidence"`
}

// SourceRef 来源引用（文件 + 页码 + Markdown 偏移 + 原文摘录）
type SourceRef struct {
	FileID     string  `json:"fileId"`
	FileName   string  `json:"fileName"`
	Page       int     `json:"page"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Snippet    string  `json:"snippet"`
	Confidence float64 `json:"confidence"`
}

// BidItem 带来源的要求项（资质/格式/暗标/废标）
type BidItem struct {
	Name    string      `json:"name"`
	Content string      `json:"content"`
	Sources []SourceRef `json:"sources"`
}

// 大纲生成策略
const (
	OutlineStrategyScoring   = "scoring"   // 严格按评标办法
	OutlineStrategyFormat    = "format"    // 严格按投标文件格式要求
	OutlineStrategyReference = "reference" // 参考评标办法及格式
)

// FallbackTotalWords 兜底总字数：招标文件未提取到要求且用户未设置时使用
const FallbackTotalWords = 100000
