package app

// ── 原罪工具集 v1 注册表（v4.262 刀）──
//
// 边界（硬隔离口径不变）：只开原罪域内工具——联网搜索、网页抓取、角色卡查询、
// 故事便签、故事大纲；**不接**办公工作区读写、命令执行、记忆面。
//
// 注册表形态与项目其它 seam 同源（image_backend / search engine / MCP）：
// 实现文件在 init() 自注册，重复注册即 panic（把接线错误拦在编译期）；
// 本轮可用工具按 sinToolOrder 固定顺序输出（与注册顺序解耦，提示词稳定）。

import (
	"context"
	"encoding/json"

	"github.com/gaea/gaea/internal/ai"
)

// 工具名常量（提示词、事件、前端标签、测试单点引用）。
const (
	sinToolWebSearch  = "web_search"
	sinToolWebFetch   = "web_fetch"
	sinToolCast       = "sin_cast"
	sinToolNotes      = "sin_notes"
	sinToolOutline    = "sin_outline"
	sinToolExport     = "sin_export"
	sinToolIllustrate = "sin_illustrate"
)

// sinToolOrder 工具在提示词与事件里的稳定顺序。
// 顺序 = 模型选工具时的心理优先级：先「把现实细节查准」（联网），再原罪域内的
// 底稿（便签/大纲/角色卡），最后「收尾动作」（导出）。
var sinToolOrder = []string{sinToolWebSearch, sinToolWebFetch, sinToolCast, sinToolNotes, sinToolOutline, sinToolIllustrate, sinToolExport}

// sinTool 单个原罪工具：声明（name/description/schema）+ 执行。
type sinTool interface {
	Name() string
	Description() string
	// Schema 参数的 JSON Schema（OpenAI 兼容 function.parameters）。
	Schema() json.RawMessage
	// ReadOnly 无副作用（提示词与前端徽标共用；写类工具必须如实返回 false）。
	ReadOnly() bool
	// Execute 解析模型生成的原始 JSON 参数，返回喂回模型的结果文本。
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// sinToolContext 工具实例化入参：话题域是原罪自有存储与角色卡查询的作用域。
// app 供角色卡/存储等需要 App 能力的工具使用（其余工具可忽略）。
type sinToolContext struct {
	app     *App
	topicID string
}

// sinToolFactory 按 kind 构建工具实例。
type sinToolFactory func(c sinToolContext) sinTool

var sinToolRegistry = map[string]sinToolFactory{}

// registerSinTool 注册工具 kind；空名/空工厂/重复注册 panic（编译期接线错误）。
func registerSinTool(kind string, f sinToolFactory) {
	if kind == "" || f == nil {
		panic("app: sin tool kind/factory must not be empty")
	}
	if _, dup := sinToolRegistry[kind]; dup {
		panic("app: duplicate sin tool kind " + kind)
	}
	sinToolRegistry[kind] = f
}

// sinToolSet 本轮可用工具（按 sinToolOrder；未注册的 kind 跳过——分批接线时仍可运行）。
func (a *App) sinToolSet(topicID string) []sinTool {
	c := sinToolContext{app: a, topicID: topicID}
	out := make([]sinTool, 0, len(sinToolOrder))
	for _, kind := range sinToolOrder {
		if f, ok := sinToolRegistry[kind]; ok {
			out = append(out, f(c))
		}
	}
	return out
}

// sinToolSchemas 把工具集转成 OpenAI 兼容 tools 定义（type=function 固定——
// 缺 type 会被 Grok/DeepSeek 等以 400 拒绝）。
func sinToolSchemas(tools []sinTool) []ai.ChatToolSchema {
	if len(tools) == 0 {
		return nil
	}
	out := make([]ai.ChatToolSchema, 0, len(tools))
	for _, t := range tools {
		out = append(out, ai.ChatToolSchema{
			Type: "function",
			Function: ai.ChatToolFunctionSpec{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			},
		})
	}
	return out
}

// sinToolTrace 单条工具调用轨迹：done.tools 与消息 extra.tools 同一形态，
// 前端单点解析（历史还原 = 流式渲染同一条形状）。
type sinToolTrace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Args      string `json:"args,omitempty"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms,omitempty"`
	ReadOnly  bool   `json:"read_only,omitempty"`
	// Artifacts 工具产物（sin_illustrate 图片；result 帧与 extra.tools 同形，
	// 前端过程卡据此渲染缩略图；落库后回写 extra.illustrations）。
	Artifacts []sinToolArtifact `json:"artifacts,omitempty"`
}

// sinToolByName 按名查找本轮工具；未知工具返回 nil——调用方如实把
// 「未知工具」回给模型（不静默吞、不猜）。
func sinToolByName(tools []sinTool, name string) sinTool {
	for _, t := range tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}
