// Package whisper — fact_embedding_cache.go
// 100% 对齐 ackem memory/factEmbeddingCache.ts
// 事实 Embedding 缓存：管理事实的 embedding 向量缓存

package whisper

import "sync"

// FactEmbeddingCache 事实 Embedding 内存缓存
type FactEmbeddingCache struct {
	mu             sync.RWMutex
	cache          map[string][]float64
	modelSignature string
}

// NewFactEmbeddingCache 创建缓存
func NewFactEmbeddingCache() *FactEmbeddingCache {
	return &FactEmbeddingCache{cache: make(map[string][]float64)}
}

// NeedsRebuild 检查模型是否切换，需要重建缓存
func (c *FactEmbeddingCache) NeedsRebuild(modelSig string) bool {
	return c.modelSignature != "" && c.modelSignature != modelSig
}

// Build 批量构建缓存
func (c *FactEmbeddingCache) Build(facts []*Fact, embeddings [][]float64, modelSig string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.modelSignature != "" && c.modelSignature != modelSig {
		c.cache = make(map[string][]float64)
	}
	for i, f := range facts {
		if i < len(embeddings) && len(embeddings[i]) > 0 {
			c.cache[f.ID] = embeddings[i]
		}
	}
	c.modelSignature = modelSig
}

// Set 单条写入
func (c *FactEmbeddingCache) Set(factID string, embedding []float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[factID] = embedding
}

// Get 获取缓存
func (c *FactEmbeddingCache) Get(factID string) []float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[factID]
}

// Delete 删除缓存
func (c *FactEmbeddingCache) Delete(factID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, factID)
}

// Size 缓存大小
func (c *FactEmbeddingCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Clear 清空
func (c *FactEmbeddingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string][]float64)
}
