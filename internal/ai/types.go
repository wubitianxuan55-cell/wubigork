package ai

// ChatMessage OpenAI 兼容消息格式
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest Chat Completions 请求
type ChatRequest struct {
	Model            string        `json:"model"`
	Messages         []ChatMessage `json:"messages"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	Temperature      float64       `json:"temperature,omitempty"`
	Stream           bool          `json:"stream"`
	ReasoningEffort  string        `json:"reasoning_effort,omitempty"`  // Grok: "low" / "high" — 控制推理深度
	TopP             float64       `json:"top_p,omitempty"`             // nucleus sampling
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"` // -2.0..2.0，抑制重复
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`  // -2.0..2.0，鼓励新话题
}

// ChatSimpleOptions ChatSimpleStream 的可选参数覆盖
type ChatSimpleOptions struct {
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
	Role         string `json:"role,omitempty"`
	Content      string `json:"content,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
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
	Content string `json:"content"` // delta 文本
	Done    bool   `json:"done"`    // 是否结束
	Error   string `json:"error,omitempty"`
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
