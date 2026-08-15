// Package retrieval 成本库等条目的本地语义精排：调用 Herdsman（或任意
// OpenAI 兼容服务）的 /v1/rerank 对粗召回候选做 cross-encoder 重排。
// 纯本地推理，不消耗云端 API token；服务不可用时调用方回退 SQL/BM25 结果。
package retrieval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ScoredDoc 是精排结果：原文档索引 + 相关性得分（越高越相关）。
type ScoredDoc struct {
	Index   int     `json:"index"`
	Score   float64 `json:"score"`
	Content string  `json:"content"`
}

// Reranker 是本地 rerank 客户端（OpenAI 兼容 POST /v1/rerank）。
type Reranker struct {
	BaseURL string
	Model   string
	client  *http.Client

	mu        sync.Mutex
	checkedAt time.Time
	available bool
}

// New 构造 rerank 客户端；baseURL 如 "http://localhost:8080/v1"。
func New(baseURL, model string) *Reranker {
	if model == "" {
		model = "bge-reranker-v2-m3"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.Contains(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return &Reranker{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Available 探测模型是否已安装/可加载（结果缓存 60 秒）。
func (r *Reranker) Available(ctx context.Context) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Since(r.checkedAt) < 60*time.Second {
		return r.available
	}
	r.checkedAt = time.Now()
	r.available = r.checkModel(ctx)
	return r.available
}

func (r *Reranker) checkModel(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.BaseURL+"/models", nil)
	if err != nil {
		return false
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return false
	}
	for _, m := range out.Data {
		if m.ID == r.Model {
			return true
		}
	}
	return false
}

// Rerank 对 docs 按与 query 的相关性精排，返回按得分降序的 topN 结果。
// 得分可能为负（cross-encoder logits），仅用于排序。
func (r *Reranker) Rerank(ctx context.Context, query string, docs []string, topN int) ([]ScoredDoc, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	if topN <= 0 || topN > len(docs) {
		topN = len(docs)
	}
	body, err := json.Marshal(map[string]any{
		"model":     r.Model,
		"query":     query,
		"documents": docs,
		"top_n":     topN,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.BaseURL+"/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rerank 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("rerank HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Results []struct {
			Index int     `json:"index"`
			Score float64 `json:"relevance_score"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("rerank 响应解析失败: %w", err)
	}
	res := make([]ScoredDoc, 0, len(out.Results))
	for _, r0 := range out.Results {
		if r0.Index >= 0 && r0.Index < len(docs) {
			res = append(res, ScoredDoc{Index: r0.Index, Score: r0.Score, Content: docs[r0.Index]})
		}
	}
	return res, nil
}

// ── RerankProvider seam（3.0 Step 3d #3：rerank 环境变量绑死 → 注册表 + config）──
// 范式见 internal/gaea/provider/provider.go 与 internal/ai/image_backend.go 的
// Register/New/Kinds。消费者（cost_tools）只依赖 RerankProvider 接口与
// config 驱动的 kind；切换 rerank 后端只改配置、代码零改动。

// RerankProvider 本地 rerank 能力接口（OpenAI 兼容 POST /v1/rerank）。
type RerankProvider interface {
	// Available 探测模型是否已安装/可加载（结果可缓存）。
	Available(ctx context.Context) bool
	// Rerank 对 docs 按与 query 的相关性精排，返回按得分降序的 topN 结果。
	Rerank(ctx context.Context, query string, docs []string, topN int) ([]ScoredDoc, error)
}

// RerankerKindOpenAI OpenAI 兼容 rerank 后端 kind（覆盖 Herdsman/Ollama
// 等 /v1/rerank 兼容服务）。
const RerankerKindOpenAI = "openai"

// RerankerConfig 是 rerank 后端实例配置（注册表 New 入参）。
type RerankerConfig struct {
	BaseURL string // API 地址（如 "http://localhost:8080"；自动补 /v1 后缀）
	Model   string // 模型名（空 = "bge-reranker-v2-m3"）
}

// RerankerFactory 按实例配置构建 rerank 后端（kind → 实例）。
type RerankerFactory func(cfg RerankerConfig) (RerankProvider, error)

// rerankerRegistry kind → 工厂注册表。各实现 init() 自注册；互斥注册，
// 重复即 panic（编译期接线错误）。
var rerankerRegistry = map[string]RerankerFactory{}

func init() {
	RegisterReranker(RerankerKindOpenAI, func(cfg RerankerConfig) (RerankProvider, error) {
		return New(cfg.BaseURL, cfg.Model), nil
	})
}

// RegisterReranker 注册 rerank 后端 kind（如 "openai"）。供各实现 init()
// 自注册；kind 为空或重复注册直接 panic。
func RegisterReranker(kind string, factory RerankerFactory) {
	if kind == "" {
		panic("retrieval: reranker kind must not be empty")
	}
	if _, dup := rerankerRegistry[kind]; dup {
		panic("retrieval: duplicate reranker kind " + kind)
	}
	rerankerRegistry[kind] = factory
}

// NewRerankerByKind 按 kind 经注册表构建 rerank 后端；未知 kind 返回错误
// （fail-closed，附已注册 kind 列表）。
func NewRerankerByKind(kind string, cfg RerankerConfig) (RerankProvider, error) {
	factory, ok := rerankerRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("retrieval: unknown reranker kind %q (registered: %v)", kind, RerankerKinds())
	}
	r, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("retrieval: reranker factory %q returned nil", kind)
	}
	return r, nil
}

// RerankerKinds 返回已注册 rerank 后端 kind 列表（排序，供诊断/校验）。
func RerankerKinds() []string {
	out := make([]string, 0, len(rerankerRegistry))
	for k := range rerankerRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
