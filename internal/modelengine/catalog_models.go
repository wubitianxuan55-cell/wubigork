package modelengine

import (
	_ "embed"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
)

// ── 通用模型目录（C 刀：目录通用化）────────────────────────────
//
// deepseek / xai / opencode-zen 三引擎的官方元数据目录（徽标数据 + 费用
// 估算），内嵌 JSON 随二进制发布。与 GLM 专属目录（glm_catalog.go，含覆盖
// 文件与远程缓存热更新）刻意分开：这三家价格稳定、变更随版本发布——
// 无覆盖文件、无远程拉取，GLM 专属热更新链路不动。
//
// 拍板决策：opencode-go 不进目录——订阅制无按量售价，目录里的参考价并非
// 售卖价，展示给用户会误导；其费用估算继续走内置表兜底，行为不变。
// custom 自定义引擎同理不进目录（用户自带 OpenAI 兼容服务商，价格未知）。
//
// schema v1：顶层 {"version":1,"updated":"...","engines":{"deepseek":[...],
// "xai":[...],"opencode-zen":[...]}}；条目 schema 与 glmCatalogEntry 一致
// （同字段名，points_* 这些引擎用不到但 schema 保持统一）。version 非 1
// 按无目录处理（估算回退内置表，只打日志）——防止未来 schema 升级被旧
// 解析器误读出脏数据。
//
// 数据来源与核实日期（写在这里——JSON 无注释）：
//   - deepseek：api-docs.deepseek.com/quick_start/pricing，2026-09-02。
//     官方峰谷双价取峰值入表，谷价/缓存价记 price_note。
//     deepseek-chat/deepseek-reasoner 官方 2026-07-24 停用，不加条目
//     （动态列表不再返回；自定义引擎旧名走内置表兜底）。
//   - xai：docs.x.ai/developers/models，2026-09-02。官方 <200k / ≥200k
//     双档取第一档，≥200k 长上下文档加价记 price_note；官方未逐模型标注
//     能力矩阵，caps 一律不填（宁缺勿滥）。grok-4/grok-4-fast/grok-3 系
//     官方已不列，不加条目（内置表兜底）。
//   - opencode-zen：opencode.ai/docs/zen，2026-09-02（免费条目 ID 已对照
//     官方文档核实）。官方模型 ID 形如 opencode/<id>，条目写裸名
//     （normalizeModelID 会去前缀，双方归一后精确匹配）；官方仅给计费
//     分档断点非 ctx 上限，context_length 一律不填（断点记 price_note）；
//     无能力矩阵，caps 不填。其余 Zen 条目价格未核实全，不加条目
//     （内置表兜底）。

//go:embed model_catalog.json
var modelCatalogJSON []byte

// modelCatalogDoc model_catalog.json 的解析形态。
type modelCatalogDoc struct {
	Version int                          `json:"version"`
	Updated string                       `json:"updated"`
	Engines map[string][]glmCatalogEntry `json:"engines"`
}

// parseModelCatalog 解析通用目录 JSON（坏 JSON 返回 error，调用方静默
// 回退无目录态——估算走内置表，绝不 panic）。
func parseModelCatalog(data []byte) (modelCatalogDoc, error) {
	var doc modelCatalogDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return modelCatalogDoc{}, err
	}
	return doc, nil
}

// modelCatalog 通用目录（懒解析一次，之后只读）。engines 为引擎目录快照
// （ModelInfo 形态，字段经 glmCatalogEntry.applyTo 填充），index 为归一化
// ID → 快照下标（enrich 精确匹配与查价前缀匹配共用）。
type modelCatalog struct {
	once    sync.Once
	engines map[string][]ModelInfo
	index   map[string]map[string]int
	version int
	updated string
	ok      bool // 内嵌目录可用（坏 JSON / 版本不符 = false）
}

// genericCatalog 进程级单例（内嵌目录只解析一次）。
var genericCatalog = &modelCatalog{}

// newModelCatalog 从任意字节构建目录实例（测试隔离用：坏 JSON/版本不符
// 的静默回退语义与内嵌加载完全同源）。经同一 once 消费初始化机会，实例
// 之后再调 load() 不会用内嵌数据重新初始化。
func newModelCatalog(data []byte) *modelCatalog {
	c := &modelCatalog{}
	c.once.Do(func() { c.init(data) })
	return c
}

// load 懒解析内嵌目录（幂等）。
func (c *modelCatalog) load() {
	c.once.Do(func() { c.init(modelCatalogJSON) })
}

// init 解析并构建目录（坏 JSON / version != 1 → ok=false 无目录态）。
func (c *modelCatalog) init(data []byte) {
	doc, err := parseModelCatalog(data)
	if err != nil {
		slog.Error("内嵌通用模型目录解析失败，估算回退内置表", "error", err)
		return
	}
	if doc.Version != 1 {
		slog.Error("内嵌通用模型目录版本不识别，估算回退内置表", "version", doc.Version)
		return
	}
	c.version, c.updated, c.ok = doc.Version, doc.Updated, true
	c.engines = make(map[string][]ModelInfo, len(doc.Engines))
	c.index = make(map[string]map[string]int, len(doc.Engines))
	for engineID, entries := range doc.Engines {
		engineID = strings.ToLower(strings.TrimSpace(engineID))
		models := make([]ModelInfo, 0, len(entries))
		idx := make(map[string]int, len(entries))
		for _, e := range entries {
			id := strings.TrimSpace(e.ID)
			if id == "" {
				continue
			}
			m := ModelInfo{ID: id, OwnedBy: engineID, Kind: ClassifyModelKind(EngineType(engineID), id)}
			e.applyTo(&m)
			idx[normalizeModelID(id)] = len(models)
			models = append(models, m)
		}
		c.engines[engineID] = models
		c.index[engineID] = idx
	}
}

// info 引擎目录快照（引擎不在目录内或目录不可用 → ok=false）。返回副本，
// 调用方修改不影响目录本体。
func (c *modelCatalog) info(engineID string) ([]ModelInfo, bool) {
	c.load()
	if !c.ok {
		return nil, false
	}
	models, ok := c.engines[strings.ToLower(strings.TrimSpace(engineID))]
	if !ok {
		return nil, false
	}
	return append([]ModelInfo(nil), models...), true
}

// price 目录查价（estimatePrice 的目录优先层，GLM 之外的目录引擎）。
// normalized 为 normalizeModelID 归一后的模型 ID：精确匹配优先；否则取
// 目录 ID 前缀匹配中的最长前缀（与 glmCatalogPrice 同口径，对齐内置表
// 长前缀在前语义）。条目须带价（free 或 price 非零）才返回 ok；free 条目
// 返回 {0,0,"CNY"} 口径价（与 GLM 免费档一致）；无价条目返回 false，
// 调用方回退内置定价表。
func (c *modelCatalog) price(engineID, normalized string) (modelPrice, bool) {
	c.load()
	if !c.ok {
		return modelPrice{}, false
	}
	engineID = strings.ToLower(strings.TrimSpace(engineID))
	idx, ok := c.index[engineID]
	if !ok {
		return modelPrice{}, false
	}
	pos, ok := idx[normalized]
	if !ok {
		pos, ok = -1, false
		best := ""
		for norm, p := range idx {
			if len(norm) > len(best) && strings.HasPrefix(normalized, norm) {
				best, pos, ok = norm, p, true
			}
		}
	}
	if !ok {
		return modelPrice{}, false
	}
	m := c.engines[engineID][pos]
	if m.Free {
		return modelPrice{Currency: "CNY"}, true
	}
	if m.PriceIn == 0 && m.PriceOut == 0 {
		return modelPrice{}, false // 目录条目无价：估算回退内置表
	}
	return modelPrice{InputPerM: m.PriceIn, OutputPerM: m.PriceOut, Currency: m.Currency, Unit: m.Unit}, true
}

// enrich 按目录为动态模型列表补充元数据（只填空字段，绝不覆盖引擎返回的
// 既有值；目录外模型原样保留）。双方经 normalizeModelID 归一后精确匹配
// （官方 opencode/ 前缀、日期后缀等变体均命中裸名条目）。Kind/OwnedBy/
// points_* 不在补充范围。目录不含该引擎时原样返回（opencode-go/custom
// 不进目录，见文件头拍板决策）。
func (c *modelCatalog) enrich(engineID string, models []ModelInfo) []ModelInfo {
	c.load()
	if !c.ok {
		return models
	}
	engineID = strings.ToLower(strings.TrimSpace(engineID))
	idx, ok := c.index[engineID]
	if !ok {
		return models
	}
	snapshot := c.engines[engineID]
	for i := range models {
		pos, hit := idx[normalizeModelID(models[i].ID)]
		if !hit {
			continue
		}
		fillCatalogMeta(&models[i], snapshot[pos])
	}
	return models
}

// fillCatalogMeta 把目录条目元数据写进引擎返回的模型（m 为待改写的元素
// 指针）：只填空字段（零值/空串），既有值一律保留。
func fillCatalogMeta(m *ModelInfo, c ModelInfo) {
	if c.ContextLength > 0 && m.ContextLength == 0 {
		m.ContextLength = c.ContextLength
	}
	if c.MaxOutput > 0 && m.MaxOutput == 0 {
		m.MaxOutput = c.MaxOutput
	}
	if c.PriceIn != 0 && m.PriceIn == 0 {
		m.PriceIn = c.PriceIn
	}
	if c.PriceOut != 0 && m.PriceOut == 0 {
		m.PriceOut = c.PriceOut
	}
	if c.Currency != "" && m.Currency == "" {
		m.Currency = c.Currency
	}
	if c.Unit != "" && m.Unit == "" {
		m.Unit = c.Unit
	}
	if c.Free && !m.Free {
		m.Free = true
	}
	if len(c.Caps) > 0 && len(m.Caps) == 0 {
		m.Caps = append([]string(nil), c.Caps...)
	}
	if c.PriceNote != "" && m.PriceNote == "" {
		m.PriceNote = c.PriceNote
	}
}

// engineCatalogInfo 引擎目录快照（deepseek/xai/opencode-zen；其余引擎
// ok=false）。估算与徽标的目录层数据源，GLM 目录（glmStaticModels）之外
// 的通用形态。
func engineCatalogInfo(engineID string) ([]ModelInfo, bool) {
	return genericCatalog.info(engineID)
}

// engineCatalogPrice 通用目录查价（estimatePrice 目录优先层）。
func engineCatalogPrice(engineID, normalizedModelID string) (modelPrice, bool) {
	return genericCatalog.price(engineID, normalizedModelID)
}

// enrichCatalogMeta 动态模型列表按目录补充元数据（fetchModels 非 GLM 公共
// 出口调用；GLM 静态目录分支不经过该出口，无双重 enrich）。
func enrichCatalogMeta(engineID string, models []ModelInfo) []ModelInfo {
	return genericCatalog.enrich(engineID, models)
}
