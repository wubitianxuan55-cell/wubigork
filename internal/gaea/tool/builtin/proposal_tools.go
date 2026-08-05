// Package builtin — 办公 agent 方案编写工具（经 proposal 全局服务）
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/proposal"
)

func init() {
	tool.RegisterBuiltin(proposalList{})
	tool.RegisterBuiltin(proposalWrite{})
	tool.RegisterBuiltin(proposalExport{})
}

// proposalList 列出全部方案
type proposalList struct{}

func (proposalList) Name() string { return "proposal_list" }
func (proposalList) Description() string {
	return "列出全部投标方案（ID/标题/阶段/状态），供选择方案继续操作。"
}
func (proposalList) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
}
func (proposalList) ReadOnly() bool                 { return true }
func (proposalList) CompactDescription() string     { return "列出全部投标方案" }
func (proposalList) CompactSchema() json.RawMessage { return json.RawMessage(`{}`) }

func (proposalList) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	svc := proposal.GlobalService()
	if svc == nil {
		return "", fmt.Errorf("方案服务未启用")
	}
	list, err := svc.List()
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return tool.WrapText("暂无方案。"), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "共 %d 个方案：\n\n", len(list))
	for _, p := range list {
		fmt.Fprintf(&b, "- %s | %s | 阶段=%s | %s\n", p.ID, p.Title, orStage(p.Stage), p.Status)
	}
	return tool.WrapText(b.String()), nil
}

// proposalWrite 写入方案内容：section_id 为空更新需求，非空更新章节
type proposalWrite struct{}

func (proposalWrite) Name() string { return "proposal_write" }
func (proposalWrite) Description() string {
	return "写入投标方案内容：proposal_id + section_id + content；section_id 为空时更新方案需求描述。"
}
func (proposalWrite) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "proposal_id":{"type":"string"},
  "section_id":{"type":"string"},
  "content":{"type":"string"}
},
"required":["proposal_id","content"]
}`)
}
func (proposalWrite) ReadOnly() bool                 { return false }
func (proposalWrite) CompactDescription() string     { return "写入方案需求或章节内容" }
func (proposalWrite) CompactSchema() json.RawMessage { return json.RawMessage(`{}`) }

func (proposalWrite) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ProposalID string `json:"proposal_id"`
		SectionID  string `json:"section_id"`
		Content    string `json:"content"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.ProposalID == "" || p.Content == "" {
		return "", fmt.Errorf("proposal_id 与 content 必填")
	}
	svc := proposal.GlobalService()
	if svc == nil {
		return "", fmt.Errorf("方案服务未启用")
	}
	if p.SectionID == "" {
		cur, err := svc.Get(p.ProposalID)
		if err != nil {
			return "", err
		}
		cur.Requirements = p.Content
		if err := svc.Update(cur); err != nil {
			return "", err
		}
	} else {
		if _, err := svc.UpdateSection(p.ProposalID, p.SectionID, "", p.Content); err != nil {
			return "", err
		}
	}
	return tool.WrapText(fmt.Sprintf("✅ 已成功写入方案 %s（%s）", p.ProposalID, orEmpty(p.SectionID, "需求描述"))), nil
}

// proposalExport 导出方案 Markdown
type proposalExport struct{}

func (proposalExport) Name() string { return "proposal_export" }
func (proposalExport) Description() string {
	return "导出投标方案为 Markdown 文件，返回文件路径。"
}
func (proposalExport) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{"proposal_id":{"type":"string"}},
"required":["proposal_id"]
}`)
}
func (proposalExport) ReadOnly() bool                 { return false }
func (proposalExport) CompactDescription() string     { return "导出方案 Markdown" }
func (proposalExport) CompactSchema() json.RawMessage { return json.RawMessage(`{}`) }

func (proposalExport) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ProposalID string `json:"proposal_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if p.ProposalID == "" {
		return "", fmt.Errorf("proposal_id 必填")
	}
	svc := proposal.GlobalService()
	if svc == nil {
		return "", fmt.Errorf("方案服务未启用")
	}
	path, err := svc.ExportMarkdown(p.ProposalID)
	if err != nil {
		return "", err
	}
	return tool.WrapText(fmt.Sprintf("✅ 已导出方案 Markdown：%s", path)), nil
}

func orStage(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func orEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
