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
	Name        string `json:"name"`
	MaxScore    string `json:"maxScore"`
	Requirement string `json:"requirement"`
}

type BidSummary struct {
	TechScoring     []ScoringItem     `json:"techScoring"`
	KeyRequirements []string          `json:"keyRequirements"`
	Duration        string            `json:"duration"`
	RedLines        []string          `json:"redLines"`
	Overview        string            `json:"overview"`
	Extra           map[string]string `json:"extra"`
	RawMarkdown     string            `json:"rawMarkdown"` // 合并后的 Markdown 全文（AI解析用）
	RawFiles        []FileDoc         `json:"rawFiles"`    // 多文件列表（上传的原始文件+转换结果）
	RawText         string            `json:"rawText"`     // 兼容旧版
}

type FileDoc struct {
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
	Children   []ProposalSection `json:"children,omitempty"` // 前端树形展示用
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
