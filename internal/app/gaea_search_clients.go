package app

// gaea_search_clients.go — app 侧本地检索客户端（embedding / 精排）单例与测试注入缝。
//
// 原名 gaea_cost_rerank.go：最早只给成本库检索用，后来文件索引、记忆召回、统一
// 语义搜索都走这两个客户端。旧成本库 2026-11-26 退役后本文件只剩通用部分，改名归位。

import (
	"os"
	"strings"
	"sync"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
)

// appEmbedderOverride / appSemanticStoreOverride 测试注入（避免触碰真实库/引擎）。
var (
	appEmbedderOverride      *retrieval.Embedder
	appSemanticStoreOverride *semantic.Store
)

// SetAppEmbedderForTest 注入隔离的 embedding 客户端（测试用）。
func SetAppEmbedderForTest(e *retrieval.Embedder) { appEmbedderOverride = e }

// SetAppSemanticStoreForTest 注入隔离的向量索引存储（测试用）。
func SetAppSemanticStoreForTest(s *semantic.Store) { appSemanticStoreOverride = s }

// appSearchClientMu 保护 app 侧检索客户端单例（刀E v4.249 走查补刀，普查#4
// 的 app 侧孪生）：localSearchEmbedder/localSearchReranker 改前每次调用 new
// 新客户端，60s 的 Available() 探测缓存按实例存、永不命中。按解析出的
// (baseURL, model) 为键缓存；引擎配置/环境变量变更自动重建。
var (
	appSearchClientMu  sync.Mutex
	cachedSearchEmbed  *retrieval.Embedder
	cachedSearchEmbKey string
	cachedSearchRrk    *retrieval.Reranker
	cachedSearchRrkKey string
)

// localSearchReranker 构造本地语义精排客户端：优先用 Herdsman 引擎配置的
// BaseURL（与模型中心一致），模型名默认 bge-reranker-v2-m3，可用环境变量覆盖。
func (a *App) localSearchReranker() *retrieval.Reranker {
	base, model := a.resolveHerdsmanSearchModel("HERDSMAN_RERANK_MODEL", []string{"qwen3-reranker"}, "bge-reranker-v2-m3")
	if base == "" {
		return nil
	}
	key := base + "\x00" + model
	appSearchClientMu.Lock()
	defer appSearchClientMu.Unlock()
	if cachedSearchRrk != nil && cachedSearchRrkKey == key {
		return cachedSearchRrk
	}
	r := retrieval.New(base, model)
	cachedSearchRrk = r
	cachedSearchRrkKey = key
	return r
}

// localSearchEmbedder 构造本地 embedding 客户端（bge-m3，引擎配置取 Herdsman）。
func (a *App) localSearchEmbedder() *retrieval.Embedder {
	if appEmbedderOverride != nil {
		return appEmbedderOverride
	}
	base, model := a.resolveHerdsmanSearchModel("HERDSMAN_EMBED_MODEL", []string{"qwen3-embedding"}, "bge-m3")
	if base == "" {
		return nil
	}
	key := base + "\x00" + model
	appSearchClientMu.Lock()
	defer appSearchClientMu.Unlock()
	if cachedSearchEmbed != nil && cachedSearchEmbKey == key {
		return cachedSearchEmbed
	}
	e := retrieval.NewEmbedder(base, model)
	cachedSearchEmbed = e
	cachedSearchEmbKey = key
	return e
}

// resolveHerdsmanSearchModel 动态选择检索模型（P4 检索升级）：
// 环境变量显式指定 > 引擎已装模型按优先级匹配（qwen3 系优先，回退 bge 系）> 默认名。
// 用户下载并启动 qwen3-embedding-4b / qwen3-reranker-4b 后自动升级，无需改代码。
//
// 只服务旧的 semantic.Store 检索路（文件 / 记忆 / 方案类目）。造价库的向量索引
// （internal/gaea/vecindex）不走这里——它固定读 [retrieval] 配置，默认 bge-m3。
func (a *App) resolveHerdsmanSearchModel(envKey string, preferred []string, fallback string) (baseURL, model string) {
	if a.engineMgr == nil {
		return "", ""
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok || !eng.Enabled || eng.BaseURL == "" {
		return "", ""
	}
	if explicit := strings.TrimSpace(os.Getenv(envKey)); explicit != "" {
		return eng.BaseURL, explicit
	}
	for _, prefix := range preferred {
		for _, m := range eng.Models {
			if strings.Contains(strings.ToLower(m.ID), prefix) {
				return eng.BaseURL, m.ID
			}
		}
	}
	return eng.BaseURL, fallback
}

// hubSemanticStore 构造共享向量索引存储（Hephaestus.db SchemaV5）。
func (a *App) hubSemanticStore() *semantic.Store {
	if appSemanticStoreOverride != nil {
		return appSemanticStoreOverride
	}
	return semantic.Open(db.GetDatabase(config.MemoryUserDir()))
}
