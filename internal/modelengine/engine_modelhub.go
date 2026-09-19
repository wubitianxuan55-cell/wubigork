// engine_modelhub.go Unsloth Studio Model Hub 域（本地引擎）：模型清单合并拉取
// （fetchModelHubModels：/v1/models 已加载集合 + /api/hub/local 完整本地清单）、
// 模型加载控制（StartModelHubModel / ModelHubModelLoaded 幂等守护）与
// ollama-manifest 别名 → 友好展示名解析（modelHubDisplayName/cleanModelHubDisplayName）。
package modelengine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/netclient"
)

// ── Unsloth Studio 响应结构 ────────────────────────────────

// studioHubLocalModel Studio /api/hub/local 单条模型（Ollama 迁移模型与
// HF 缓存均在其中；OpenAI /v1/models 只列「已加载」）。gaea 用它与 /v1/models
// 合并：Ollama 迁移的两个模型（tinyrick/aratan）无论当前是否加载都能识别。
type studioHubLocalModel struct {
	ID           string `json:"id"` // ollama-manifest:… 引用（与 /v1/models id 一致）
	DisplayName  string `json:"display_name"`
	Source       string `json:"source"`
	ModelFormat  string `json:"model_format"`
	Partial      bool   `json:"partial"`
	Capabilities struct {
		CanChat bool `json:"can_chat"`
	} `json:"capabilities"`
}

type studioHubLocalResponse struct {
	Models []studioHubLocalModel `json:"models"`
}

// fetchModelHubModels 拉取 Unsloth Studio 的模型清单并合并：
//  1. OpenAI 兼容 /v1/models → 「当前已加载」模型集合（loaded=true）；
//  2. Studio 内部 /api/hub/local（同一 sk- Key 可访问）→ 完整本地模型，
//     其中 source=ollama 且可聊天的条目（tinyrick/aratan）即使未加载也列出，
//     状态标记为 stopped——让 gaea 一次看到「Ollama 迁来的两个模型」。
//     /api/hub/local 请求失败时降级为只列已加载模型（保证主链路不受影响）。
func (m *Manager) fetchModelHubModels(ctx context.Context, engine *EngineConfig) ([]ModelInfo, error) {
	base := strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")
	if !validBaseURL(base) {
		return nil, fmt.Errorf("引擎地址无效：需要 http:// 或 https:// 前缀，请在模型中心修正")
	}
	m.mu.RLock()
	key := m.modelhubKey
	m.mu.RUnlock()

	loadJSON := func(url string, out any) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := m.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("请求失败: %w", err)
		}
		defer netclient.DrainAndClose(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("HTTP 401: Model Hub API Key 无效或未配置，请先在模型中心保存 Unsloth 生成的 Key（sk-unsloth- 开头，Unsloth 设置 → API 创建）")
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP %d: 模型列表获取失败", resp.StatusCode)
		}
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("解析模型列表失败: %w", err)
		}
		return nil
	}

	// 1) 已加载集合（/v1/models，loaded=true；缺失 loaded 字段保守放行）
	var list modelsListResponse
	if err := loadJSON(base+"/models", &list); err != nil {
		return nil, err
	}
	running := map[string]ModelInfo{}
	for _, d := range list.Data {
		if d.Loaded != nil && !*d.Loaded {
			continue
		}
		running[d.ID] = ModelInfo{
			ID:      d.ID,
			OwnedBy: d.OwnedBy,
			Status:  "running",
			Name:    modelHubDisplayName(d.ID, d.DisplayName),
			Kind:    ClassifyModelKind(engine.Type, d.ID),
		}
	}

	// 2) 完整本地清单（best-effort）
	hubBase := strings.TrimSuffix(base, "/v1")
	merged := make(map[string]ModelInfo, len(running)+2)
	var hub studioHubLocalResponse
	if err := loadJSON(hubBase+"/api/hub/local", &hub); err != nil {
		slog.Warn("Model Hub 本地清单获取失败（降级为只列已加载模型）", "error", err)
	} else {
		for _, item := range hub.Models {
			// Ollama 迁移模型（可聊天 GGUF、未半成品）即使未加载也带回；
			// HF 缓存条目只在已加载时由 running 集合兜底。
			if item.Source != "ollama" || item.ModelFormat != "gguf" ||
				item.Partial || !item.Capabilities.CanChat || item.ID == "" {
				continue
			}
			status := "stopped"
			if _, ok := running[item.ID]; ok {
				status = "running"
			}
			merged[item.ID] = ModelInfo{
				ID:      item.ID,
				OwnedBy: "unsloth-studio",
				Status:  status,
				Name:    cleanModelHubDisplayName(item.DisplayName),
				Kind:    ClassifyModelKind(engine.Type, item.ID),
			}
		}
	}
	// 已加载但未出现在清单过滤范围里的模型（如用户加载的 HF GGUF）兜底补回
	for id, mdl := range running {
		if _, ok := merged[id]; !ok {
			merged[id] = mdl
		}
	}

	models := make([]ModelInfo, 0, len(merged))
	for _, mdl := range merged {
		models = append(models, mdl)
	}
	sort.SliceStable(models, func(i, j int) bool {
		if (models[i].Status == "running") != (models[j].Status == "running") {
			return models[i].Status == "running" // 运行中的排前面（默认模型拾取优先）
		}
		return models[i].ID < models[j].ID
	})
	return models, nil
}

// cleanModelHubDisplayName 去掉 Studio /api/hub/local 展示名里的规格后缀
// （如 "…:Q6_K_P (27.3B Q6_K)" → "…:Q6_K_P"），保持模型卡名称干净。
func cleanModelHubDisplayName(display string) string {
	if i := strings.Index(display, " ("); i > 0 {
		return display[:i]
	}
	return display
}

// StartModelHubModel 让 Unsloth Studio 加载指定模型（modelID 为
// ollama-manifest:… 引用）。调用后 Studio 会把当前加载模型切换为该模型，
// gaea 前端随后刷新模型列表即可看到状态变化。返回 nil 表示已加载成功。
func (m *Manager) StartModelHubModel(ctx context.Context, modelID string) error {
	m.mu.RLock()
	engine, ok := m.engines["modelhub"]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("引擎 modelhub 不存在")
	}
	if !engine.Enabled {
		return fmt.Errorf("Model Hub 引擎未启用")
	}
	base := strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")
	if !validBaseURL(base) {
		return fmt.Errorf("引擎地址无效：需要 http:// 或 https:// 前缀")
	}
	m.mu.RLock()
	key := m.modelhubKey
	m.mu.RUnlock()
	if key == "" {
		return fmt.Errorf("Model Hub API Key 未配置，请先在模型中心保存 Unsloth 生成的 Key（sk-unsloth- 开头）")
	}
	hubBase := strings.TrimSuffix(base, "/v1")
	payload, err := json.Marshal(map[string]any{
		"model_path":   modelID,
		"load_in_4bit": false,
		"force_reload": false,
	})
	if err != nil {
		return fmt.Errorf("序列化加载请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hubBase+"/api/inference/load", strings.NewReader(string(payload)))
	if err != nil {
		return fmt.Errorf("创建加载请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("加载请求失败: %w", err)
	}
	defer netclient.DrainAndClose(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("HTTP 401: Model Hub API Key 无效，请在模型中心重新保存（Unsloth 设置 → API 创建）")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("Studio 加载模型失败（HTTP %d）: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil // 成功响应但无可解析体（某些版本直接 204/空）→ 视为成功
	}
	if out.Error != "" {
		return fmt.Errorf("Studio 加载模型失败: %s", out.Error)
	}
	if out.Status != "" && out.Status != "loaded" && out.Status != "loading" {
		return fmt.Errorf("Studio 加载模型未就绪（status=%s）", out.Status)
	}
	slog.Info("Model Hub 模型已加载", "model", modelID)
	return nil
}

// ModelHubModelLoaded 轻量探测目标模型是否已在 Studio 加载（MH4 预热的幂等
// 守护：force_reload 语义官方未文档化（蒸馏规划 §四-2），预热前先查 /v1/models
// 已含目标即跳过，从不依赖重复 load 的行为）。
// 返回 (false, nil) = 可达但目标未加载；(false, err) = Studio 不可达/鉴权失败。
func (m *Manager) ModelHubModelLoaded(ctx context.Context, modelID string) (bool, error) {
	m.mu.RLock()
	engine, ok := m.engines["modelhub"]
	key := m.modelhubKey
	m.mu.RUnlock()
	if !ok {
		return false, fmt.Errorf("引擎 modelhub 不存在")
	}
	if !engine.Enabled {
		return false, fmt.Errorf("Model Hub 引擎未启用")
	}
	base := strings.TrimRight(strings.TrimSpace(engine.BaseURL), "/")
	if !validBaseURL(base) {
		return false, fmt.Errorf("引擎地址无效：需要 http:// 或 https:// 前缀")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", nil)
	if err != nil {
		return false, fmt.Errorf("创建探测请求失败: %w", err)
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("Studio 不可达: %w", err)
	}
	defer netclient.DrainAndClose(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return false, fmt.Errorf("HTTP 401: Model Hub API Key 无效")
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Studio 探测失败（HTTP %d）", resp.StatusCode)
	}
	var list modelsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return false, fmt.Errorf("解析已加载模型列表失败: %w", err)
	}
	for _, d := range list.Data {
		if d.ID == modelID && (d.Loaded == nil || *d.Loaded) {
			return true, nil
		}
	}
	return false, nil
}

// modelHubDisplayName 生成 Model Hub（Unsloth Studio）模型的展示名。
// Studio /v1/models 已加载模型只给 opaque 的 ollama-manifest:<URL 编码路径>
// 别名（形如 C:\Users\…\manifests\registry.ollama.ai\tinyrick\<模型>\Q6_K_P），
// 不适合直接展示；这里解析出 Ollama 同款 repo 名（tinyrick/<模型>:Q6_K_P）。
// 服务端显式下发 display_name 时优先使用（如 HF 缓存条目的友好名）。
func modelHubDisplayName(id, display string) string {
	if strings.TrimSpace(display) != "" {
		return display
	}
	if !strings.HasPrefix(id, "ollama-manifest:") {
		return id
	}
	decoded, err := url.QueryUnescape(strings.TrimPrefix(id, "ollama-manifest:"))
	if err != nil {
		return id
	}
	norm := strings.ReplaceAll(decoded, "\\", "/")
	idx := strings.Index(norm, "manifests/")
	if idx < 0 {
		return id
	}
	parts := strings.Split(norm[idx+len("manifests/"):], "/")
	// 期望布局 manifests/<host>/<namespace>/<model>/<tag>（≥4 段）。
	if len(parts) < 4 {
		return id
	}
	repo := strings.Join(parts[1:len(parts)-1], "/")
	return repo + ":" + parts[len(parts)-1]
}
