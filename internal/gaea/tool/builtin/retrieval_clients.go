package builtin

// retrieval_clients.go — 工具侧本地检索客户端（embedding / 精排）与运行时配置注入。
//
// 原在 cost_tools.go：最早只给 cost_search 用，方案库检索（proposal_tools.go）
// 后来复用。旧成本库 2026-11-26 退役、cost_search / cost_save 一并下线后，这里
// 只留通用部分并改名归位。

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/retrieval"
)

var rerankerOverride *retrieval.Reranker
var embedderOverride *retrieval.Embedder

// SetRerankerForTest 注入隔离的 rerank 客户端（测试用）。
func SetRerankerForTest(r *retrieval.Reranker) { rerankerOverride = r }

// SetEmbedderForTest 注入隔离的 embedding 客户端（测试用）。
func SetEmbedderForTest(e *retrieval.Embedder) { embedderOverride = e }

// retrievalClientMu 保护两个检索客户端单例（刀D v4.247）：改前每次工具调用
// new 一个 Embedder/Reranker（各自新建 http.Client），60s 的 Available() 探测
// 缓存按实例存、永不命中——每次语义召回/精排都多做一次 /models 探测且
// keep-alive 全部失效。实例按注入的运行时配置为键缓存；SetRetrievalRuntime
// 改配置后下次调用自动重建。构造无网络 I/O，持锁构建即可。
var (
	retrievalClientMu sync.Mutex
	cachedEmbedder    *retrieval.Embedder
	cachedEmbedderCfg RetrievalRuntime
	cachedReranker    *retrieval.Reranker
	cachedRerankerCfg RetrievalRuntime
)

// localEmbedder 构造本地 embedding 客户端：config 驱动（SetRetrievalRuntime
// 注入的 embed kind/base/model），未注入时回落默认值（localhost:8080 + bge-m3）。
// 3.0 Step 3d #2：不读环境变量绑死。
func localEmbedder() *retrieval.Embedder {
	if embedderOverride != nil {
		return embedderOverride
	}
	kind := retrieval.EmbedderKindOpenAI
	cfg := retrieval.EmbedderConfig{BaseURL: "http://localhost:8080", Model: "bge-m3"}
	if retrievalRuntime.EmbedKind != "" {
		kind = retrievalRuntime.EmbedKind
	}
	if retrievalRuntime.EmbedBaseURL != "" {
		cfg.BaseURL = retrievalRuntime.EmbedBaseURL
	}
	if retrievalRuntime.EmbedModel != "" {
		cfg.Model = retrievalRuntime.EmbedModel
	}
	retrievalClientMu.Lock()
	defer retrievalClientMu.Unlock()
	if cachedEmbedder != nil && cachedEmbedderCfg == retrievalRuntime {
		return cachedEmbedder
	}
	e, err := retrieval.NewEmbedderByKind(kind, cfg)
	if err != nil {
		// 未知 kind fail-closed：不静默降级，返回 nil（调用方按"服务不可用"回退）。
		// 失败不缓存——下次调用重试，配置修复后无需重启。
		slog.Warn("retrieval: embedding 后端不可用", "kind", kind, "error", err)
		return nil
	}
	if em, ok := e.(*retrieval.Embedder); ok {
		cachedEmbedder = em
		cachedEmbedderCfg = retrievalRuntime
		return em
	}
	// 注册表可能返回非 *Embedder 实现（第三方 kind），本工具沿用具体类型，
	// 接口化后由调用方统一处理——此处仅保留兼容路径。
	return nil
}

// localReranker 构造本地 rerank 客户端：config 驱动（SetRetrievalRuntime
// 注入的 rerank kind/base/model），未注入时回落默认值（localhost:8080 +
// bge-reranker-v2-m3）。3.0 Step 3d #3：不读环境变量绑死。
func localReranker() *retrieval.Reranker {
	if rerankerOverride != nil {
		return rerankerOverride
	}
	kind := retrieval.RerankerKindOpenAI
	cfg := retrieval.RerankerConfig{BaseURL: "http://localhost:8080", Model: "bge-reranker-v2-m3"}
	if retrievalRuntime.RerankKind != "" {
		kind = retrievalRuntime.RerankKind
	}
	if retrievalRuntime.RerankBaseURL != "" {
		cfg.BaseURL = retrievalRuntime.RerankBaseURL
	}
	if retrievalRuntime.RerankModel != "" {
		cfg.Model = retrievalRuntime.RerankModel
	}
	retrievalClientMu.Lock()
	defer retrievalClientMu.Unlock()
	if cachedReranker != nil && cachedRerankerCfg == retrievalRuntime {
		return cachedReranker
	}
	r, err := retrieval.NewRerankerByKind(kind, cfg)
	if err != nil {
		// 未知 kind fail-closed：返回 nil（调用方回退原结果，不静默降级）。
		// 失败不缓存——下次调用重试，配置修复后无需重启。
		slog.Warn("retrieval: rerank 后端不可用", "kind", kind, "error", err)
		return nil
	}
	if rk, ok := r.(*retrieval.Reranker); ok {
		cachedReranker = rk
		cachedRerankerCfg = retrievalRuntime
		return rk
	}
	return nil
}

// ── retrieval 运行时配置注入（3.0 Step 3d #2/#3）────────────────────────

// RetrievalRuntime 是本地检索（embed/rerank）后端的运行时配置，由 boot 从
// gaea.toml 注入。零值 = 全默认（kind=openai，localhost:8080 + 各自默认模型）。
type RetrievalRuntime struct {
	EmbedKind     string // 注册表 kind（默认 retrieval.EmbedderKindOpenAI）
	EmbedBaseURL  string // embedding API 地址（默认 http://localhost:8080）
	EmbedModel    string // embedding 模型（默认 bge-m3）
	RerankKind    string // 注册表 kind（默认 retrieval.RerankerKindOpenAI）
	RerankBaseURL string // rerank API 地址（默认 http://localhost:8080）
	RerankModel   string // rerank 模型（默认 bge-reranker-v2-m3）
}

// retrievalRuntime 保存 SetRetrievalRuntime 注入的检索后端配置。
var retrievalRuntime RetrievalRuntime

// SetRetrievalRuntime 注入本地检索后端配置（boot 装配调用）。
// 切换 embedding/rerank 后端只改配置（kind/base/model），消费方代码零改动。
func SetRetrievalRuntime(cfg RetrievalRuntime) { retrievalRuntime = cfg }

// cell 表格单元格转义：竖线替换为斜杠，避免破坏 Markdown 表。
func cell(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "|", "/")
}
