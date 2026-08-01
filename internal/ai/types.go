package ai

import "encoding/json"

// ChatMessage OpenAI 兼容消息格式
type ChatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []ChatToolCall `json:"tool_calls,omitempty"`   // assistant：模型请求的工具调用
	ToolCallID string         `json:"tool_call_id,omitempty"` // tool：回传的工具调用 ID
	Name       string         `json:"name,omitempty"`         // tool：工具名（部分 API 需要）
}

// ChatToolCall OpenAI 兼容工具调用（assistant 消息内）。
type ChatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function ChatToolFunction `json:"function,omitempty"`
}

// ChatToolFunction 工具调用的函数名与参数（JSON 字符串）。
type ChatToolFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ChatToolFunctionSpec 单个工具的 function 定义（OpenAI 兼容 tools 元素内部）。
type ChatToolFunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatToolSchema OpenAI 兼容工具定义（tools 数组元素）。
// OpenAI 规范要求 {"type":"function","function":{...}}——缺失 type 字段
// 会被 DeepSeek/Grok 等兼容 API 以 400 拒绝（"tools[0]: missing field `type`"）。
type ChatToolSchema struct {
	Type     string               `json:"type"` // 固定 "function"
	Function ChatToolFunctionSpec `json:"function"`
}

// ChatToolCallDelta 流式 tool_calls 增量分片，按 Index 拼装。
type ChatToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

// ChatRequest Chat Completions 请求
type ChatRequest struct {
	Model            string           `json:"model"`
	EngineID         string           `json:"-"` // 功能级引擎覆盖（空=全局激活引擎），不序列化
	Messages         []ChatMessage    `json:"messages"`
	MaxTokens        int              `json:"max_tokens,omitempty"`
	Temperature      float64          `json:"temperature,omitempty"`
	Stream           bool             `json:"stream"`
	Tools            []ChatToolSchema `json:"tools,omitempty"`             // 工具定义（agent 工具循环用）
	ReasoningEffort  string           `json:"reasoning_effort,omitempty"`  // Grok: "low" / "high" — 控制推理深度
	TopP             float64          `json:"top_p,omitempty"`             // nucleus sampling
	FrequencyPenalty float64          `json:"frequency_penalty,omitempty"` // -2.0..2.0，抑制重复
	PresencePenalty  float64          `json:"presence_penalty,omitempty"`  // -2.0..2.0，鼓励新话题
}

// ChatSimpleOptions ChatSimpleStream 的可选参数覆盖
type ChatSimpleOptions struct {
	EngineID        string  // 功能级引擎覆盖（空=全局激活引擎）
	Temperature     float64 // 覆盖默认 temperature（0 表示使用默认值）
	MaxTokens       int     // 覆盖默认 max_tokens（0 表示使用默认值）
	ReasoningEffort string  // 推理深度（"" 表示不开启推理）
	TopP            float64 // nucleus sampling（0 表示不发送）
	TimeoutMinutes  int     // 超时分钟数（0 表示使用默认 5 分钟）
}

// ChatChoice 单个候选
type ChatChoice struct {
	Index   int         `json:"index"`
	Delta   ChatDelta   `json:"delta,omitempty"`
	Message ChatMessage `json:"message,omitempty"`
}

// ChatDelta SSE 流式 delta
type ChatDelta struct {
	Role         string              `json:"role,omitempty"`
	Content      string              `json:"content,omitempty"`
	ToolCalls    []ChatToolCallDelta `json:"tool_calls,omitempty"` // 工具调用分片（按 Index 拼装）
	FinishReason string              `json:"finish_reason,omitempty"`
}

// ChatResponse Chat Completions 响应（非流式）
type ChatResponse struct {
	ID      string       `json:"id,omitempty"`
	Model   string       `json:"model,omitempty"`
	Choices []ChatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// SSEChunk 流式响应的一帧
type SSEChunk struct {
	Content   string         `json:"content"` // delta 文本
	Done      bool           `json:"done"`    // 是否结束
	Error     string         `json:"error,omitempty"`
	ToolCalls []ChatToolCall `json:"tool_calls,omitempty"` // 完整工具调用（finish_reason=tool_calls 时携带）
}

// ModelsResponse /v1/models
type ModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// ── 图片生成 ───────────────────────────────────────────────

// ImageGenerationRequest POST /v1/images/generations
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Negative       string `json:"negative,omitempty"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"` // "url" 或 "b64_json"
	Seed           int    `json:"seed,omitempty"`
	Lora           string `json:"lora,omitempty"` // LoRA 文件名（逗号分隔多个）
}

// ImageData 单张图片结果
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// ImageGenerationResponse /v1/images/generations 响应
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}
