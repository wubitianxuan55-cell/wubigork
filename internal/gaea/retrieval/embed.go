package retrieval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Embedder 是本地 embedding 客户端（OpenAI 兼容 POST /v1/embeddings）。
// 用于语义召回：关键词召回不足时，把候选库向量化后按余弦相似度补召回。
// 纯本地推理（如 Herdsman bge-m3），不消耗云端 token。
type Embedder struct {
	BaseURL string
	Model   string
	client  *http.Client

	mu        sync.Mutex
	checkedAt time.Time
	available bool
}

// NewEmbedder 构造 embedding 客户端；baseURL 如 "http://localhost:8080/v1"。
func NewEmbedder(baseURL, model string) *Embedder {
	if model == "" {
		model = "bge-m3"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.Contains(baseURL, "/v1") {
		baseURL += "/v1"
	}
	return &Embedder{
		BaseURL: baseURL,
		Model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Available 探测模型是否可用（结果缓存 60 秒）。
func (e *Embedder) Available(ctx context.Context) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if time.Since(e.checkedAt) < 60*time.Second {
		return e.available
	}
	e.checkedAt = time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.BaseURL+"/models", nil)
	if err != nil {
		e.available = false
		return false
	}
	resp, err := e.client.Do(req)
	if err != nil {
		e.available = false
		return false
	}
	defer resp.Body.Close()
	e.available = false
	if resp.StatusCode == http.StatusOK {
		var out struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out) == nil {
			for _, m := range out.Data {
				if m.ID == e.Model {
					e.available = true
					break
				}
			}
		}
	}
	return e.available
}

// Embed 批量向量化文本，返回与输入等长的向量列表（维度 1024 for bge-m3）。
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	body, err := json.Marshal(map[string]any{"model": e.Model, "input": texts})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding 请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("embedding HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("embedding 响应解析失败: %w", err)
	}
	res := make([][]float32, len(texts))
	for _, d := range out.Data {
		if d.Index >= 0 && d.Index < len(texts) {
			res[d.Index] = d.Embedding
		}
	}
	return res, nil
}

// Cosine 计算两个向量的余弦相似度（维度不一致返回 0）。
func Cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
