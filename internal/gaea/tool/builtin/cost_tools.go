package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() {
	tool.RegisterBuiltin(costSearch{})
	tool.RegisterBuiltin(costSave{})
}

// costStoreOverride 测试注入的隔离成本库；nil 时使用真实用户成本库
// （Hephaestus.db cost_entries，与记忆中枢 CostLibrary 同库）。
var costStoreOverride *cost.Store
var rerankerOverride *retrieval.Reranker
var embedderOverride *retrieval.Embedder
var semanticStoreOverride *semantic.Store

// SetCostStoreForTest 注入隔离的成本库存储（测试用，避免触碰真实用户库）。
func SetCostStoreForTest(s *cost.Store) { costStoreOverride = s }

// SetRerankerForTest 注入隔离的 rerank 客户端（测试用）。
func SetRerankerForTest(r *retrieval.Reranker) { rerankerOverride = r }

// SetEmbedderForTest 注入隔离的 embedding 客户端（测试用）。
func SetEmbedderForTest(e *retrieval.Embedder) { embedderOverride = e }

// SetSemanticStoreForTest 注入隔离的向量索引存储（测试用）。
func SetSemanticStoreForTest(s *semantic.Store) { semanticStoreOverride = s }

// openCostStore 打开成本库存储（可测试注入）。
func openCostStore() (*cost.Store, error) {
	if costStoreOverride != nil {
		return costStoreOverride, nil
	}
	return cost.Open(db.GetDatabase(config.MemoryUserDir())), nil
}

// costSearch 搜索成本库：测算前读取历史单价作为定价依据。
type costSearch struct{}

func (costSearch) Name() string { return "cost_search" }
func (costSearch) Description() string {
	return "搜索成本库：按关键词/分类路径/状态查询历史单价，供成本测算引用；分类支持完整路径（如「材料/钢材」，含子分类）或一级分类名；不填参数返回成本库概览。"
}
func (costSearch) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"搜索关键词（标题/规格/来源/正文，可选）"},
  "category":{"type":"string","description":"分类过滤（可选）：完整路径如「材料/钢材」（含子分类），或一级分类名 机械/材料/人工/运输/检测/综合单价/其他"},
  "status":{"type":"string","description":"状态过滤（可选）：现行/草稿/已归档"},
  "limit":{"type":"integer","description":"返回条数上限（默认20，最大50）"}
}
}`)
}
func (costSearch) ReadOnly() bool                 { return true }
func (costSearch) CompactDescription() string     { return compactDesc["cost_search"] }
func (costSearch) CompactSchema() json.RawMessage { return compactSchema["cost_search"] }

func (costSearch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query    string `json:"query,omitempty"`
		Category string `json:"category,omitempty"`
		Status   string `json:"status,omitempty"`
		Limit    int    `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}

	store, err := openCostStore()
	if err != nil {
		return "", err
	}
	if !store.Available() {
		return "", fmt.Errorf("成本库不可用（用户目录为空或数据库打开失败）")
	}

	if strings.TrimSpace(p.Query) == "" && strings.TrimSpace(p.Category) == "" && strings.TrimSpace(p.Status) == "" {
		return costOverview(store), nil
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	list := store.Search(p.Query, p.Category, p.Status)
	// 语义召回：关键词召回不足（<3）时用本地 bge-m3 补召回，覆盖别名/口语
	// 表达（如「液压振动锤」→ hp300），避免漏检。纯本地，不消耗云端 token。
	if len(list) < 3 && strings.TrimSpace(p.Query) != "" {
		if sem := semanticCostRecall(ctx, p.Query, list, store, 10); len(sem) > 0 {
			list = sem
		}
	}
	if len(list) == 0 {
		return "未找到匹配的成本条目。可先按合理估价测算，完成后用 cost_save 把采用的单价沉淀进成本库。", nil
	}
	// 本地语义精排（Herdsman bge-reranker-v2-m3）：候选多时提升排序精度；
	// 模型不可用或失败时自动回退 SQL 结果。纯本地推理，不消耗云端 token。
	if reranked := rerankCostResults(ctx, p.Query, list, limit); len(reranked) > 0 {
		list = reranked
	} else if len(list) > limit {
		list = list[:limit]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## 成本库搜索结果（%d 条）\n\n", len(list))
	b.WriteString("| 名称 | 标题 | 分类 | 单价(元) | 单位 | 规格 | 地区 | 期数 | 口径 | 来源 | 状态 |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, e := range list {
		price := fmt.Sprintf("%.2f", e.Price)
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			cell(e.Name), cell(e.Title), cell(e.Category), price, cell(e.Unit),
			cell(e.Spec), cell(e.Region), cell(e.PriceDate), cell(e.PriceType),
			cell(e.Source), cell(e.Status))
	}
	b.WriteString("\n同名条目可用 cost_save 覆盖更新（name 同上）。")
	return tool.WrapText(b.String()), nil
}

// semanticCostRecall 语义召回：持久化向量索引（Ensure 增量向量化 + 查询只嵌
// query），候选库规模不影响单次查询成本。
func semanticCostRecall(ctx context.Context, query string, have []cost.Summary, store *cost.Store, topN int) []cost.Summary {
	e := costEmbedder()
	if e == nil {
		return nil
	}
	full := store.List()
	if len(full) == 0 {
		return nil
	}
	st := openSemanticStore()
	if st == nil || !st.Available() {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	docs := make([]semantic.Doc, len(full))
	keep := make(map[string]bool, len(full))
	for i, s := range full {
		docs[i] = semantic.Doc{ID: s.Name, Text: retrieval.DocText(s)}
		keep[s.Name] = true
	}
	_, _ = st.Stale("cost", keep)
	hits, err := st.Search(ctx, e, "cost", docs, query, topN)
	if err != nil || len(hits) == 0 {
		return nil
	}
	haveNames := make(map[string]bool, len(have))
	for _, s := range have {
		haveNames[s.Name] = true
	}
	out := append([]cost.Summary{}, have...)
	byName := make(map[string]cost.Summary, len(full))
	for _, s := range full {
		byName[s.Name] = s
	}
	for _, h := range hits {
		if s, ok := byName[h.ID]; ok && !haveNames[h.ID] {
			out = append(out, s)
		}
	}
	return out
}

// openSemanticStore 打开共享向量索引存储（测试可注入）。
func openSemanticStore() *semantic.Store {
	if semanticStoreOverride != nil {
		return semanticStoreOverride
	}
	return semantic.Open(db.GetDatabase(config.MemoryUserDir()))
}

// costEmbedder 构造本地 embedding 客户端：config 驱动（SetRetrievalRuntime
// 注入的 embed kind/base/model），未注入时回落默认值（等价旧 HERDSMAN_BASE_URL
// 缺省行为：localhost:8080 + bge-m3）。3.0 Step 3d #2：不再读环境变量绑死。
func costEmbedder() *retrieval.Embedder {
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
	e, err := retrieval.NewEmbedderByKind(kind, cfg)
	if err != nil {
		// 未知 kind fail-closed：不静默降级，返回 nil（调用方按"服务不可用"回退）。
		slog.Warn("cost_search: embedding 后端不可用", "kind", kind, "error", err)
		return nil
	}
	if em, ok := e.(*retrieval.Embedder); ok {
		return em
	}
	// 注册表可能返回非 *Embedder 实现（第三方 kind），本工具沿用具体类型，
	// 接口化后由调用方统一处理——此处仅保留兼容路径。
	return nil
}

// rerankCostResults 用本地 rerank 对粗召回结果精排；失败返回 nil（调用方回退）。
func rerankCostResults(ctx context.Context, query string, list []cost.Summary, limit int) []cost.Summary {
	if len(list) <= 8 || strings.TrimSpace(query) == "" || limit <= 0 {
		return nil
	}
	r := costReranker()
	if r == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if !r.Available(ctx) {
		return nil
	}
	docs := make([]string, len(list))
	for i, e := range list {
		docs[i] = costDocText(e)
	}
	scored, err := r.Rerank(ctx, query, docs, limit)
	if err != nil || len(scored) == 0 {
		return nil
	}
	out := make([]cost.Summary, 0, len(scored))
	for _, s := range scored {
		if s.Index >= 0 && s.Index < len(list) {
			out = append(out, list[s.Index])
		}
	}
	return out
}

// costReranker 构造本地 rerank 客户端：config 驱动（SetRetrievalRuntime
// 注入的 rerank kind/base/model），未注入时回落默认值（等价旧 HERDSMAN_BASE_URL
// 缺省行为：localhost:8080 + bge-reranker-v2-m3）。3.0 Step 3d #3：不再读环境变量绑死。
func costReranker() *retrieval.Reranker {
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
	r, err := retrieval.NewRerankerByKind(kind, cfg)
	if err != nil {
		// 未知 kind fail-closed：返回 nil（调用方回退 SQL 结果，不静默降级）。
		slog.Warn("cost_search: rerank 后端不可用", "kind", kind, "error", err)
		return nil
	}
	if rk, ok := r.(*retrieval.Reranker); ok {
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

// costDocText 把成本条目摘要拼成精排文档串。
func costDocText(e cost.Summary) string {
	var b strings.Builder
	b.WriteString(e.Title)
	if e.Spec != "" {
		b.WriteString("（" + e.Spec + "）")
	}
	if e.Unit != "" {
		b.WriteString(" 单位" + e.Unit)
	}
	b.WriteString(fmt.Sprintf(" 单价%.2f元", e.Price))
	if e.Category != "" {
		b.WriteString(" 分类" + e.Category)
	}
	if e.Source != "" {
		b.WriteString(" 来源" + e.Source)
	}
	if len(e.Tags) > 0 {
		b.WriteString(" 标签" + strings.Join(e.Tags, ","))
	}
	return b.String()
}

// costOverview 成本库概览：按分类计数，引导模型测算前先引用。
func costOverview(store *cost.Store) string {
	list := store.List()
	if len(list) == 0 {
		return "成本库为空。测算完成后用 cost_save 把采用的单价沉淀进来，下次即可直接引用。"
	}

	catCount := make(map[string]int)
	for _, s := range list {
		catCount[s.Category]++
	}

	var b strings.Builder
	b.WriteString("## 成本库概览\n\n")
	b.WriteString("| 分类 | 条目数 |\n|------|--------|\n")
	total := 0
	for _, cat := range []string{"机械", "材料", "人工", "运输", "检测", "综合单价", "其他"} {
		if count, ok := catCount[cat]; ok {
			fmt.Fprintf(&b, "| %s | %d |\n", cat, count)
			total += count
		}
	}
	fmt.Fprintf(&b, "| **合计** | **%d** |\n\n", total)
	b.WriteString("测算前用 `cost_search` 按科目查单价，完成后用 `cost_save` 沉淀新单价。")
	return b.String()
}

// cell 表格单元格转义：竖线替换为斜杠，避免破坏 Markdown 表。
func cell(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "|", "/")
}

// costSave 写入/更新成本库条目（同名 UPSERT）。
type costSave struct{}

func (costSave) Name() string { return "cost_save" }
func (costSave) Description() string {
	return "写入/更新成本库条目（同名覆盖）：测算完成后把采用的单价沉淀为成本条目，来源标注本次项目/文件。"
}
func (costSave) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "name":{"type":"string","description":"条目名称（唯一键，可选；不填自动从标题生成）"},
  "title":{"type":"string","description":"标题，如 HP300 高频液压振动锤"},
  "category":{"type":"string","description":"分类（叶子名）：机械/材料/人工/运输/检测/综合单价/其他（默认其他）"},
  "categoryPath":{"type":"string","description":"完整分类路径，如 材料/钢材（可选；不填则取 category）"},
  "unit":{"type":"string","description":"单位：台班/吨/m³/工日等"},
  "price":{"type":"number","description":"单价（元）"},
  "spec":{"type":"string","description":"规格型号，如 300kW"},
  "source":{"type":"string","description":"来源：定额/市场询价/历史项目/本次测算文件等（建议必填）"},
  "tags":{"type":"string","description":"标签，逗号分隔，如 振动锤,桩基"},
  "status":{"type":"string","description":"状态：现行/草稿/已归档（默认现行）"},
  "body":{"type":"string","description":"备注/计算说明"}
},
"required":["title","price"]
}`)
}
func (costSave) ReadOnly() bool                 { return false }
func (costSave) PersistWrite() bool              { return true }
func (costSave) CompactDescription() string     { return compactDesc["cost_save"] }
func (costSave) CompactSchema() json.RawMessage { return compactSchema["cost_save"] }

func (costSave) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name         string  `json:"name,omitempty"`
		Title        string  `json:"title"`
		Category     string  `json:"category,omitempty"`
		CategoryPath string  `json:"categoryPath,omitempty"`
		Unit         string  `json:"unit,omitempty"`
		Price        float64 `json:"price"`
		Spec         string  `json:"spec,omitempty"`
		Source       string  `json:"source,omitempty"`
		Tags         string  `json:"tags,omitempty"`
		Status       string  `json:"status,omitempty"`
		Body         string  `json:"body,omitempty"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if strings.TrimSpace(p.Title) == "" {
		return "", fmt.Errorf("title 为必填项")
	}
	if p.Price < 0 {
		return "", fmt.Errorf("price 不能为负数")
	}

	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = cost.SlugName(p.Title)
	}
	category := strings.TrimSpace(p.Category)
	if category == "" {
		category = "其他"
	}
	status := strings.TrimSpace(p.Status)
	if status == "" {
		status = "现行"
	}

	var tags []string
	if strings.TrimSpace(p.Tags) != "" {
		for _, t := range strings.Split(p.Tags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	store, err := openCostStore()
	if err != nil {
		return "", err
	}
	if !store.Available() {
		return "", fmt.Errorf("成本库不可用（用户目录为空或数据库打开失败）")
	}

	action := "新增"
	e := cost.Entry{
		Name: name, Title: strings.TrimSpace(p.Title), Category: category,
		CategoryPath: strings.TrimSpace(p.CategoryPath), Unit: strings.TrimSpace(p.Unit),
		Price: p.Price, Spec: strings.TrimSpace(p.Spec), Source: strings.TrimSpace(p.Source),
		Tags: tags, Status: status, Body: strings.TrimSpace(p.Body),
	}
	if existing, _ := store.Get(name); existing != nil {
		action = "覆盖更新"
		// 工具不管理人材机二级组成/费率：更新时保留，避免改价抹掉子目明细。
		e.Components = existing.Components
		e.LaborFee, e.MaterialFee, e.MachineFee = existing.LaborFee, existing.MaterialFee, existing.MachineFee
		e.ManagementFee, e.ProfitFee, e.AdvanceFee, e.TaxRate =
			existing.ManagementFee, existing.ProfitFee, existing.AdvanceFee, existing.TaxRate
	}
	if err := store.Save(e); err != nil {
		return "", fmt.Errorf("保存失败: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "✅ 已%s成本条目「%s」\n\n", action, e.Title)
	fmt.Fprintf(&b, "**名称**: `%s`\n", e.Name)
	fmt.Fprintf(&b, "**分类**: %s | **单价**: %.2f 元", e.Category, e.Price)
	if e.Unit != "" {
		fmt.Fprintf(&b, "/%s", e.Unit)
	}
	fmt.Fprintf(&b, " | **状态**: %s\n", e.Status)
	if e.Spec != "" {
		fmt.Fprintf(&b, "**规格**: %s\n", e.Spec)
	}
	if e.Source != "" {
		fmt.Fprintf(&b, "**来源**: %s\n", e.Source)
	}
	b.WriteString("\n同名条目自动覆盖更新；可在记忆中枢「成本库」页查看/编辑。")
	return tool.WrapText(b.String()), nil
}
