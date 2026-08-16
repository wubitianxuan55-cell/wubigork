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
	Model              string             `json:"model"`
	EngineID           string             `json:"-"` // 功能级引擎覆盖（空=全局激活引擎），不序列化
	Messages           []ChatMessage      `json:"messages"`
	MaxTokens          int                `json:"max_tokens,omitempty"`
	Temperature        float64            `json:"temperature,omitempty"`
	Stream             bool               `json:"stream"`
	Tools              []ChatToolSchema   `json:"tools,omitempty"`                // 工具定义（agent 工具循环用）
	ReasoningEffort    string             `json:"reasoning_effort,omitempty"`     // Grok: "low" / "high" — 控制推理深度
	EnableThinking     *bool              `json:"enable_thinking,omitempty"`      // Qwen3 等本地模型：开启思考模式（输出 reasoning_content）
	ChatTemplateKwargs map[string]any     `json:"chat_template_kwargs,omitempty"` // llama.cpp 系服务端模板参数（如 enable_thinking）
	TopP               float64            `json:"top_p,omitempty"`                // nucleus sampling
	FrequencyPenalty   float64            `json:"frequency_penalty,omitempty"`    // -2.0..2.0，抑制重复
	PresencePenalty    float64            `json:"presence_penalty,omitempty"`     // -2.0..2.0，鼓励新话题
	StreamOptions      *ChatStreamOptions `json:"stream_options,omitempty"`       // 流式用量上报（include_usage）
	skipIncludeUsage   bool               `json:"-"`                              // 内部：重试时不再请求 include_usage
}

// ChatStreamOptions OpenAI 兼容流式附加选项。
type ChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ChatSimpleOptions ChatSimpleStream 的可选参数覆盖
type ChatSimpleOptions struct {
	EngineID        string  // 功能级引擎覆盖（空=全局激活引擎）
	Temperature     float64 // 覆盖默认 temperature（0 表示使用默认值）
	MaxTokens       int     // 覆盖默认 max_tokens（0 表示使用默认值）
	ReasoningEffort string  // 推理深度（"" 表示不开启推理）
	EnableThinking  bool    // 本地 Qwen3 系模型：开启思考模式（输出 reasoning_content）
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
	Role             string              `json:"role,omitempty"`
	Content          string              `json:"content,omitempty"`
	ReasoningContent string              `json:"reasoning_content,omitempty"` // 思考模式下的推理分片（Qwen3/DeepSeek 系）
	ToolCalls        []ChatToolCallDelta `json:"tool_calls,omitempty"`        // 工具调用分片（按 Index 拼装）
	FinishReason     string              `json:"finish_reason,omitempty"`
}

// ChatResponse Chat Completions 响应（非流式）
type ChatResponse struct {
	ID      string       `json:"id,omitempty"`
	Model   string       `json:"model,omitempty"`
	Choices []ChatChoice `json:"choices"`
	Usage   *ChatUsage   `json:"usage,omitempty"`
	Error   *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ChatUsage OpenAI 兼容 usage 字段（流式最后一块通常也携带）。
// 缓存拆分兼容两种形状：DeepSeek 顶层 prompt_cache_{hit,miss}_tokens，
// 以及 OpenAI/MiMo 标准 prompt_tokens_details.cached_tokens。
type ChatUsage struct {
	PromptTokens          int64 `json:"prompt_tokens,omitempty"`
	CompletionTokens      int64 `json:"completion_tokens,omitempty"`
	TotalTokens           int64 `json:"total_tokens,omitempty"`
	PromptCacheHitTokens  int64 `json:"prompt_cache_hit_tokens,omitempty"`  // DeepSeek 风格
	PromptCacheMissTokens int64 `json:"prompt_cache_miss_tokens,omitempty"` // DeepSeek 风格
	PromptTokensDetails   *struct {
		CachedTokens int64 `json:"cached_tokens,omitempty"` // OpenAI/MiMo 风格
	} `json:"prompt_tokens_details,omitempty"`
}

// CacheHitTokens 返回缓存的 prompt token 数（兼容两种形状）。
func (u *ChatUsage) CacheHitTokens() int64 {
	if u == nil {
		return 0
	}
	if u.PromptCacheHitTokens > 0 {
		return u.PromptCacheHitTokens
	}
	if u.PromptTokensDetails != nil {
		return u.PromptTokensDetails.CachedTokens
	}
	return 0
}

// CacheMissTokens 返回未命中的 prompt token 数；服务端未拆分时按
// prompt - cache_hit 推算（下限 0），保证统计面板不出现负数。
func (u *ChatUsage) CacheMissTokens() int64 {
	if u == nil {
		return 0
	}
	if u.PromptCacheMissTokens > 0 {
		return u.PromptCacheMissTokens
	}
	miss := u.PromptTokens - u.CacheHitTokens()
	if miss < 0 {
		return 0
	}
	return miss
}

// SSEChunk 流式响应的一帧
type SSEChunk struct {
	Content   string         `json:"content"`             // delta 文本
	Reasoning string         `json:"reasoning,omitempty"` // 思考模式下的推理 delta
	Done      bool           `json:"done"`                // 是否结束
	Error     string         `json:"error,omitempty"`
	ToolCalls []ChatToolCall `json:"tool_calls,omitempty"` // 完整工具调用（finish_reason=tool_calls 时携带）
	Usage     *ChatUsage     `json:"usage,omitempty"`      // 流结束块携带的用量（若有）
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
	Model            string                                  `json:"model"`
	Prompt           string                                  `json:"prompt"`
	Negative         string                                  `json:"negative,omitempty"`
	N                int                                     `json:"n,omitempty"`
	Size             string                                  `json:"size,omitempty"`
	ResponseFormat   string                                  `json:"response_format,omitempty"` // "url" 或 "b64_json"
	Seed             int                                     `json:"seed,omitempty"`
	Lora             string                                  `json:"lora,omitempty"`       // LoRA 文件名（逗号分隔多个）
	Mode             string                                  `json:"mode,omitempty"`       // txt2img | img2img | t2v
	InitImage        string                                  `json:"init_image,omitempty"` // img2img 参考图（base64 data URL）
	Denoise          float64                                 `json:"denoise,omitempty"`    // img2img 重绘幅度 0-1
	Frames           int                                     `json:"frames,omitempty"`     // t2v 帧数
	FPS              int                                     `json:"fps,omitempty"`        // t2v 帧率
	// ProgressCallback 生成进度回调（status/elapsedSeconds/percent(-1=未知)/node 当前节点 class_type）。
	// 仅 ComfyUI 后端会带上真实 percent 与 node；其余后端 percent 恒为 -1。
	ProgressCallback func(status string, elapsedSeconds int, percent int, node string) `json:"-"`
}

// PortraitStylePrefix 角色剧照统一前置风格提示词（写实摄影风）。
// 角色库与小说角色的剧照生成共用，保证出图风格一致。
const PortraitStylePrefix = "写实摄影风格，自然光线，超高分辨率，逼真质感，细节丰富，8K，专业拍摄，"

// ImageData 单张图片结果
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Kind          string `json:"kind,omitempty"` // image | video
}

// ImageGenerationResponse /v1/images/generations 响应
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}
