// engine_models.go 模型列表域：OpenAI 兼容响应结构、模型列表拉取（fetchModels，
// 含各引擎鉴权/过滤/内置模型补充/目录元数据 enrich）、模型能力分类
// （ClassifyModelKind/ByName，llm/tts/stt/ocr/rerank/embedding/image 单一来源）、
// GLM Key 校验 ping（glmPing/zhipuErrorMessage）与地址/OpenCode 兼容性判定。
package modelengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ── OpenAI 兼容响应结构 ────────────────────────────────────

type modelsListResponse struct {
	Data []struct {
		ID      string `json:"id"`
		OwnedBy string `json:"owned_by"`
		Status  string `json:"status"`
		// Unsloth Studio OpenAI 兼容目录扩展（/v1/models）：当前是否已加载。
		// Studio 固定后端 8888/v1 会同时列出「已加载模型」与「仅缓存/未加载
		// 条目」（如下载一半的 GGUF）；loaded=false 的条目 gaea 无法调用，
		// 刷新时直接过滤掉，避免把不可用模型带回默认模型/功能绑定候选。
		Loaded *bool `json:"loaded,omitempty"`
		// Studio 某些条目会下发展示名（如 HF 缓存半成品）；已加载的
		// ollama-manifest 别名没有 display_name，由 modelHubDisplayName
		// 从别名解析出友好名（tinyrick/…:Q6_K_P）。
		DisplayName string `json:"display_name,omitempty"`
	} `json:"data"`
}

// fetchModels 从引擎获取模型列表
// fetchModels 从引擎获取模型列表
func (m *Manager) fetchModels(ctx context.Context, engine *EngineConfig) ([]ModelInfo, error) {
	if engine.Type == EngineGLM {
		// 智谱无 /models 端点：刷新直接返回官方静态目录（零 HTTP）
		return m.glmCatalogModels(), nil
	}
	if engine.Type == EngineModelHub {
		// Unsloth Studio：OpenAI /v1/models 只暴露「当前已加载」模型，完整
		// 模型清单要再读 Studio 内部 /api/hub/local（同一把 sk- Key 可访问）。
		// 合并后两个 Ollama 迁移模型都能被 gaea 识别（运行/停止状态分开）。
		return m.fetchModelHubModels(ctx, engine)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")
	if !validBaseURL(baseURL) {
		return nil, fmt.Errorf("引擎地址无效：需要 http:// 或 https:// 前缀，请在模型中心修正")
	}
	url := baseURL + "/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// xAI / DeepSeek / GLM / OpenCode Go / Model Hub 需要认证（Key 未配置时
	// 不带 Authorization 头，服务端会回 401 由下方给出对应引擎提示）
	if engine.Type == EngineXAI && m.xaiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.xaiKey)
	} else if engine.Type == EngineDeepseek && m.deepseekKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.deepseekKey)
	} else if engine.Type == EngineGLM && m.glmKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.glmKey)
	} else if engine.Type == EngineOpencodeGo && m.opencodeKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.opencodeKey)
	} else if engine.Type == EngineOpencodeZen && m.opencodeZenKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.opencodeZenKey)
	} else if engine.Type == EngineModelHub && m.modelhubKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.modelhubKey)
	} else if engine.Type == EngineCustom {
		// 自定义引擎：Key 在内存 customKeys（KeyStore 同源），空 Key 不带
		// Authorization 头（兼容无鉴权的本地 OpenAI 兼容服务）。
		if key := m.CustomEngineKey(engine.ID); key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 401 {
			if engine.Type == EngineXAI {
				return nil, fmt.Errorf("HTTP 401: 未登录 xAI，请先点击「登录 xAI」获取授权")
			} else if engine.Type == EngineDeepseek {
				return nil, fmt.Errorf("HTTP 401: DeepSeek API Key 无效或未配置，请在设置中配置")
			} else if engine.Type == EngineGLM {
				return nil, fmt.Errorf("HTTP 401: GLM API Key 无效或未配置，请在模型中心配置（open.bigmodel.cn 获取）")
			} else if engine.Type == EngineOpencodeGo {
				return nil, fmt.Errorf("HTTP 401: OpenCode Go API Key 无效或未配置，请先在模型中心配置（opencode.ai 订阅获取）")
			} else if engine.Type == EngineOpencodeZen {
				return nil, fmt.Errorf("HTTP 401: OpenCode Zen API Key 无效或未配置，请先在模型中心配置（opencode.ai/auth 获取）")
			} else if engine.Type == EngineModelHub {
				return nil, fmt.Errorf("HTTP 401: Model Hub API Key 无效或未配置，请先在模型中心保存 Unsloth 生成的 Key（sk-unsloth- 开头，Unsloth 设置 → API 创建）")
			} else if engine.Type == EngineCustom {
				return nil, fmt.Errorf("HTTP 401: 自定义引擎 API Key 无效或未配置，请在模型中心自定义引擎卡片修正")
			}
		}
		return nil, fmt.Errorf("HTTP %d: 模型列表获取失败", resp.StatusCode)
	}

	var result modelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析模型列表失败: %w", err)
	}

	models := make([]ModelInfo, len(result.Data))
	for i, d := range result.Data {
		// OpenCode 目录端点分布与 Go 不同（如 grok 在 Zen 走 /responses、
		// minimax 在 Zen 走 chat/completions），按引擎分别过滤，
		// 只保留当前聊天客户端支持的 OpenAI 兼容 /chat/completions 模型。
		if engine.Type == EngineOpencodeGo && !opencodeGoCompatible(d.ID) {
			continue
		}
		if engine.Type == EngineOpencodeZen && !opencodeZenCompatible(d.ID) {
			continue
		}
		models[i] = ModelInfo{
			ID:      d.ID,
			OwnedBy: d.OwnedBy,
			Status:  d.Status,
			Kind:    ClassifyModelKind(engine.Type, d.ID),
		}
	}

	// 过滤后可能有空洞（continue 跳过），压缩数组
	if engine.Type == EngineOpencodeGo || engine.Type == EngineOpencodeZen {
		filtered := make([]ModelInfo, 0, len(models))
		for _, m := range models {
			if m.ID != "" {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	// xAI 引擎补充内置语音模型 grok-tts（TTS API 不返回在 /v1/models 列表中）
	if engine.Type == EngineXAI {
		found := false
		for _, mdl := range models {
			if mdl.ID == "grok-tts" {
				found = true
				break
			}
		}
		if !found {
			models = append(models, ModelInfo{ID: "grok-tts", OwnedBy: "xai", Kind: "tts"})
		}
	}

	// C 刀目录通用化：动态模型列表按通用目录补充徽标元数据（只填空字段，
	// 不覆盖引擎返回值，随 saveState 持久化）。GLM 分支已在函数头提前返回
	// 静态目录、不走此公共出口（无双重 enrich）；opencode-go/custom 不在
	// 目录内，enrich 原样返回（零行为变化）。
	models = enrichCatalogMeta(string(engine.Type), models)

	return models, nil
}

// ClassifyModelKind 按引擎类型与模型名分类（llm/tts/stt/ocr/rerank/embedding/image）。
// 3.0 Step 3d：模型能力关键词分类的单一来源——语音（voice_handler.go:isSTTModel）、
// OCR（gaea_ocr.go:pickHerdsmanModel）等消费点委托到本函数，不再各自维护关键词表。
// 分类下沉到后端后，前端不再需要按名称猜测；行为与旧前端启发式保持一致，避免行为跳变。
// EngineCustom（A 刀自定义 OpenAI 兼容服务商）无厂商特型规则，与多数类型一样
// 直接落通用关键词表、默认 llm——刻意不加类型分支（避免死代码），由测试锚定该行为。
func ClassifyModelKind(engineType EngineType, modelID string) string {
	l := strings.ToLower(modelID)
	if engineType == EngineCosyVoice ||
		strings.Contains(l, "tts") || strings.Contains(l, "voice") ||
		strings.Contains(l, "edge") || strings.Contains(l, "speech") ||
		strings.Contains(l, "voxcpm") {
		return "tts"
	}
	if strings.Contains(l, "sherpa") || strings.Contains(l, "whisper") ||
		strings.Contains(l, "zipformer") || strings.Contains(l, "asr") ||
		strings.Contains(l, "funasr") {
		return "stt"
	}
	if strings.Contains(l, "paddleocr") || strings.Contains(l, "ocr") ||
		strings.Contains(l, "mineru") {
		return "ocr"
	}
	if strings.Contains(l, "rerank") {
		return "rerank"
	}
	if strings.Contains(l, "embedding") || strings.Contains(l, "bge-m3") ||
		strings.Contains(l, "bge") {
		return "embedding"
	}
	// GLM 官方静态目录优先判型：glm-image 系为生图，其余 glm-* 均为对话/视觉
	// 理解模型（官方目录锚定）——通用 turbo 关键词曾把 glm-5-turbo 误判为生图
	// （v4.10.0 回归），故 GLM 引擎先走本块再落通用关键词表。
	if engineType == EngineGLM && strings.HasPrefix(l, "glm-") {
		if strings.HasPrefix(l, "glm-image") {
			return "image"
		}
		return "llm"
	}
	if strings.Contains(l, "image") || strings.Contains(l, "zimage") ||
		strings.Contains(l, "flux") || strings.Contains(l, "cogview") ||
		strings.Contains(l, "turbo") ||
		strings.Contains(l, "sd") || strings.Contains(l, "dalle") ||
		strings.Contains(l, "krea") {
		return "image"
	}
	return "llm"
}

// ClassifyModelByName 只按模型名关键词分类（不依赖引擎类型），供语音/OCR 侧
// 按模型 ID 单参数判断能力（如 isSTTModel / pickHerdsmanModel 委托）。
// 与 ClassifyModelKind 的引擎无关部分保持同一关键词表，避免双源漂移。
func ClassifyModelByName(modelID string) string {
	return ClassifyModelKind("", modelID)
}

// glmPing 用最小 chat 请求验证 Key 有效性——智谱没有模型列表端点可供鉴权
// 探测，官方鉴权口径 = Authorization: Bearer <API Key>（docs.bigmodel.cn
// 「HTTP API 调用」）。错误体官方形态 {"error":{"code","message"}}，原样透出。
func (m *Manager) glmPing(ctx context.Context, engine *EngineConfig) error {
	m.mu.RLock()
	key := m.glmKey
	m.mu.RUnlock()
	if key == "" {
		return fmt.Errorf("GLM API Key 未配置，请在模型中心 GLM 卡片保存 Key（open.bigmodel.cn 获取）")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")+"/chat/completions",
		strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"ping"}],"max_tokens":1}`, engine.DefaultModel)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if msg := zhipuErrorMessage(body); msg != "" {
		return fmt.Errorf("GLM Key 校验失败（HTTP %d）：%s", resp.StatusCode, msg)
	}
	return fmt.Errorf("GLM Key 校验失败：HTTP %d", resp.StatusCode)
}

// zhipuErrorMessage 解析智谱错误体 {"error":{"code","message"}}（官方形态，
// 真机实测 {"code":"500","message":"内部错误"} 等）。
func zhipuErrorMessage(body []byte) string {
	var e struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return ""
}

// validBaseURL 引擎地址必须带 http(s) scheme——防御把 API Key 等非地址内容
// 粘进地址框（v4.9.1 真机实测：GLM 卡片曾对云端引擎露出地址框，Key 被存成
// base_url 后每个请求都报 unsupported protocol scheme ""）。
func validBaseURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// opencodeGoCompatible 判断 OpenCode Go 模型是否走 OpenAI /chat/completions 端点。
// 参考 https://dev.opencode.ai/docs/go 端点分类表。
func opencodeGoCompatible(modelID string) bool {
	id := strings.ToLower(modelID)
	if strings.HasPrefix(id, "qwen3") || strings.HasPrefix(id, "minimax") || id == "gpt-5.6-luna" {
		return false
	}
	return true
}

// opencodeZenCompatible 判断 OpenCode Zen 模型是否走 OpenAI /chat/completions 端点。
// 参考 https://dev.opencode.ai/docs/zen 端点分类表：
//   - gpt-* / grok-* → /responses；claude-* / qwen3* → /messages；gemini-* → 专用端点
//   - deepseek-* / minimax-* / glm-* / kimi-* / mimo-* / 免费模型等 → /chat/completions
func opencodeZenCompatible(modelID string) bool {
	id := strings.ToLower(modelID)
	for _, prefix := range []string{"gpt-", "claude-", "gemini-", "qwen3", "grok-"} {
		if strings.HasPrefix(id, prefix) {
			return false
		}
	}
	return true
}
