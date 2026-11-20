// Package builtin provides Tianxuan's compile-time built-in tools. Each tool
// self-registers via init(); main blank-imports this package to wire them in.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/feedback"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(aiFeedback{}) }

// aiFeedback AI 反馈工具（2026-11-20 用户立项：模型在使用过程中遇到困难、
// 发现不足后主动提出反馈意见，记忆面板反馈区收集展示，可导出 md 改进软件）。
//
// 语义边界：这不是记忆写入（不进 dream 确认队列——反馈不是事实候选，用户
// 立项的机制本身就是要收集模型侧意见）；落点 <userDir>/ai-feedback.jsonl
// （gaea 自管数据区，非用户工作区文件，不触发发布边界）。
type aiFeedback struct{}

func (aiFeedback) Name() string { return "ai_feedback" }

func (aiFeedback) Description() string {
	return "向开发者提交一条产品反馈（当你在任务中遇到真实困难时主动用，不要等用户要求）：工具反复试错不顺手、行为与预期不符、缺少关键能力、工作流有摩擦——一条反馈讲一件事。反馈进入记忆面板「AI 反馈」区，用户可查看/筛选/导出成 md 用于改进软件。不用于提问（提问直接问用户）、不用于报任务进度、不要为凑数提交空泛意见。"
}

func (aiFeedback) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "title":{"type":"string","description":"一句话概括问题（≤80 字，具体到工具/场景）"},
  "detail":{"type":"string","description":"具体描述：当时在做什么、工具返回了什么、为什么不顺、期望是什么。有真实回执/数字/路径就引用"},
  "category":{"type":"string","enum":["tool-friction","bug","missing-capability","workflow","other"],"description":"tool-friction=工具不好用；bug=行为不对；missing-capability=缺能力；workflow=流程摩擦；other"},
  "severity":{"type":"string","enum":["low","medium","high"],"description":"low=偶发摩擦；medium=明显浪费时间；high=阻塞交付或产生返工"},
  "board":{"type":"string","enum":["chat","office","wenxue"],"description":"当前所在板块（自报）"}
},
"required":["title","detail"]
}`)
}

// ReadOnly 反馈写 gaea 自管数据区（<userDir>/ai-feedback.jsonl），不是用户
// 工作区写面——权限按只读放行（与 evidence journal 落 .gaea/ 同语义）。
func (aiFeedback) ReadOnly() bool { return true }

// SpaceTag 自声明 shared：聊天/办公/文学三板块都装配（image_gen shared 先例；
// SpaceShared 为本包 spacetags.go 定义的通用标签）。
func (aiFeedback) SpaceTag() string { return SpaceShared }

// feedbackUserDir seam（包级变量以便测试定向临时目录）。
var feedbackUserDir = config.MemoryUserDir

func (aiFeedback) CompactDescription() string {
	return compactDesc["ai_feedback"]
}
func (aiFeedback) CompactSchema() json.RawMessage { return compactSchema["ai_feedback"] }

func (aiFeedback) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Title    string `json:"title"`
		Detail   string `json:"detail"`
		Category string `json:"category"`
		Severity string `json:"severity"`
		Board    string `json:"board"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("ai_feedback 参数解析失败: %w", err)
	}
	p.Title = strings.TrimSpace(p.Title)
	p.Detail = strings.TrimSpace(p.Detail)
	if p.Title == "" {
		return "", fmt.Errorf("title is required：一句话概括问题（具体到工具/场景）")
	}
	if p.Detail == "" {
		return "", fmt.Errorf("detail is required：描述当时在做什么、工具返回了什么、期望是什么")
	}
	if p.Category != "" && !(p.Category == "tool-friction" || p.Category == "bug" || p.Category == "missing-capability" || p.Category == "workflow" || p.Category == "other") {
		return "", fmt.Errorf("category %q 不在枚举内（schema 列出合法值），修正后重试", p.Category)
	}
	if p.Severity != "" && !(p.Severity == "low" || p.Severity == "medium" || p.Severity == "high") {
		return "", fmt.Errorf("severity %q 不在枚举内（low/medium/high），修正后重试", p.Severity)
	}
	if p.Board != "" && !(p.Board == "chat" || p.Board == "office" || p.Board == "wenxue") {
		return "", fmt.Errorf("board %q 不在枚举内（chat/office/wenxue），修正后重试", p.Board)
	}
	it, err := feedback.Append(feedbackUserDir(), feedback.Item{
		Title: p.Title, Detail: p.Detail,
		Category: p.Category, Severity: p.Severity, Board: p.Board,
	})
	if err != nil {
		return "", fmt.Errorf("反馈入库失败: %w", err)
	}
	return fmt.Sprintf("反馈已记录（%s）。条目进入记忆面板「AI 反馈」区，用户可查看/导出。感谢——具体到工具与场景的反馈最有用。", it.ID), nil
}
