// Package whisper — wave_endpoint.go
// 100% 对齐 ackem chat/waveEndpoint.ts
// 端点选择：Wave0 可走本地模型，Wave1+ 走主 API

package whisper

// WaveEndpoint 端点配置
type WaveEndpoint struct {
	Provider  string            `json:"provider"`  // "openai" | "anthropic"
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Model     string            `json:"model"`
	MaxTokens int               `json:"maxTokens"`
	IsLocal   bool              `json:"isLocal"`
}

// ProbeLocalChatResult 本地端点探测结果
type ProbeLocalChatResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs,omitempty"`
	Model     string `json:"model,omitempty"`
	Error     string `json:"error,omitempty"`
}

// LocalChatConfig 本地聊天配置（从 settings 抽取）
type LocalChatConfig struct {
	LocalChatEnabled   bool
	LocalChatBaseURL   string
	LocalChatModel     string
	LocalChatMaxTokens int
	LocalChatAPIKey    string
}

// SelectWaveEndpoint 选择端点：Wave0 可走本地；Wave1+ 走主 API
// 100% 对齐 ackem chat/waveEndpoint.ts selectWaveEndpoint
// 注意：wubigrok 用模型中心统一管理，此处保留端点选择逻辑
// 供上层决定使用本地/云端模型
func SelectWaveEndpoint(
	waveIndex int,
	mainProvider string,     // "openai" | "anthropic"
	mainModel string,
	mainMaxTokens int,
	localConfig LocalChatConfig,
) WaveEndpoint {
	// Wave0 优先本地
	if waveIndex == 0 && localConfigured(localConfig) {
		base := localConfig.LocalChatBaseURL
		if base == "" {
			base = localConfig.LocalChatBaseURL
		}
		maxTokens := localConfig.LocalChatMaxTokens
		if maxTokens <= 0 {
			maxTokens = 80
		}
		return WaveEndpoint{
			Provider:  "openai",
			URL:       resolveLocalCompletionsURL(base),
			Headers:   buildLocalHeaders(localConfig.LocalChatAPIKey),
			Model:     localConfig.LocalChatModel,
			MaxTokens: maxTokens,
			IsLocal:   true,
		}
	}

	// 主 API
	provider := mainProvider
	if provider != "anthropic" {
		provider = "openai"
	}
	maxTokens := mainMaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	return WaveEndpoint{
		Provider:  provider,
		Model:     mainModel,
		MaxTokens: maxTokens,
		IsLocal:   false,
	}
}

// ─── 辅助函数 ──────────────────────────────────────────────────

func localConfigured(cfg LocalChatConfig) bool {
	return cfg.LocalChatEnabled &&
		cfg.LocalChatBaseURL != "" &&
		cfg.LocalChatModel != ""
}

func resolveLocalCompletionsURL(baseURL string) string {
	// 去掉尾部斜杠
	for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	// 如果已含 /chat/completions 则直接返回
	if len(baseURL) > 16 && baseURL[len(baseURL)-16:] == "/chat/completions" {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func buildLocalHeaders(apiKey string) map[string]string {
	h := map[string]string{
		"Content-Type": "application/json",
	}
	if apiKey != "" {
		h["Authorization"] = "Bearer " + apiKey
	}
	return h
}
