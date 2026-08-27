package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/pins"
	"github.com/gaea/gaea/internal/whisper"
	whisperdb "github.com/gaea/gaea/internal/whisper/db/repos"
)

// ── 记忆中枢绑定（主脑前端入口的统一管理接口）───────────────────────
//
// 记忆中枢 = 三脑架构的前端呈现：左脑办公记忆（Hephaestus.db facts）、
// 主脑全局画像（profile）+ 知识库（knowledge）、右脑轻语记忆（hermes.db 只读）。
// 成本库等未来库沿用同一"库管理"模式扩展。

// ProfileFactView 主脑全局画像事实视图（与 memory.Memory 对应）。
type ProfileFactView struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Kind        string   `json:"kind"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
}

// WhisperMemoryView 轻语记忆事实只读视图。
type WhisperMemoryView struct {
	ID          string    `json:"id"`
	Domain      string    `json:"domain"`
	Subcategory string    `json:"subcategory"`
	Subject     string    `json:"subject"`
	Summary     string    `json:"summary"`
	Weight      float64   `json:"weight"`
	Confidence  float64   `json:"confidence"`
	Tier        string    `json:"tier"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WhisperEpisodeView 轻语情节记忆只读视图（时间倒序展示）。
type WhisperEpisodeView struct {
	ID                 string    `json:"id"`
	Summary            string    `json:"summary"`
	DominantEmotion    string    `json:"dominantEmotion"`
	EmotionalIntensity float64   `json:"emotionalIntensity"`
	Keywords           []string  `json:"keywords"`
	StartTurn          int       `json:"startTurn"`
	EndTurn            int       `json:"endTurn"`
	CreatedAt          time.Time `json:"createdAt"`
	SourceSessionID    string    `json:"sourceSessionId"`
}

// MemoryHubOverview 记忆中枢聚合总览（各库统计 + 最近条目）。
type MemoryHubOverview struct {
	KnowledgeCount int    `json:"knowledgeCount"`
	ProfileCount   int    `json:"profileCount"`
	OfficeCount    int    `json:"officeCount"`
	CostCount      int    `json:"costCount"`
	WhisperCount   int    `json:"whisperCount"`
	PinnedCount    int    `json:"pinnedCount"` // 项目资料：工作区固定常用文件数
	LatestUpdated  string `json:"latestUpdated"`
}

// hubProfileStore 构造主脑画像存储（nil 表示不可用）。
func (a *App) hubProfileStore() *memory.ProfileStore {
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return memory.NewProfileStore(nil)
	}
	return memory.NewProfileStore(db.GetDatabase(userDir))
}

// hubOfficeStore 构造左脑办公记忆存储（SQLite 后端）。
func (a *App) hubOfficeStore() memory.Store {
	if officeStoreOverrideSet {
		return officeStoreOverride
	}
	userDir := config.MemoryUserDir()
	cwd := gaeaCwd()
	return memory.SQLiteStoreFor(db.GetDatabase(userDir), userDir, cwd)
}

// whisperDataRootSafe 返回轻语数据根目录；whisperState 未初始化（测试/异常
// 状态）时返回空串，调用方按“无轻语数据”处理，避免 nil 提升字段崩溃。
func (a *App) whisperDataRootSafe() string {
	if a.whisperState == nil {
		return ""
	}
	return a.whisperDataRoot
}

// officeStoreOverride 测试注入的隔离办公记忆存储（避免触碰真实用户库）。
var (
	officeStoreOverride    memory.Store
	officeStoreOverrideSet bool
)

// SetOfficeStoreForTest 注入隔离的办公记忆存储（测试用）。
func SetOfficeStoreForTest(s memory.Store) {
	officeStoreOverride = s
	officeStoreOverrideSet = true
}

// ResetOfficeStoreForTest 清除测试注入，恢复真实办公记忆存储。
func ResetOfficeStoreForTest() { officeStoreOverrideSet = false }

// GaeaProfileList 返回主脑全局画像事实列表。
func (a *App) GaeaProfileList() []ProfileFactView {
	ps := a.hubProfileStore()
	all := ps.All()
	out := make([]ProfileFactView, 0, len(all))
	for _, m := range all {
		out = append(out, ProfileFactView{
			Name: m.Name, Title: m.Title, Description: m.Description,
			Type: string(m.Type), Kind: string(m.Kind), Tags: nonNilStrings(m.Tags), Body: m.Body,
		})
	}
	return out
}

// nonNilStrings 把 nil 切片归一化为空切片，避免序列化成 JSON null
// 导致前端 `arr.length` 崩溃（如记忆中枢画像 tags）。
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// GaeaProfileSave 保存一条主脑画像事实（同名覆盖）。
func (a *App) GaeaProfileSave(f ProfileFactView) error {
	return a.hubProfileStore().Save(memory.Memory{
		Name: f.Name, Title: f.Title, Description: f.Description,
		Type: memory.NormalizeType(f.Type), Kind: memory.NormalizeKind(f.Kind),
		Tags: f.Tags, Body: f.Body,
	})
}

// GaeaProfileDelete 删除一条主脑画像事实。
func (a *App) GaeaProfileDelete(name string) error {
	return a.hubProfileStore().Delete(name)
}

// GaeaProfileConflicts 返回画像与办公 facts 中同名且描述不一致的冲突项
// （主脑画像 vs 左脑遗留 facts 对同一事实说法不同）。
func (a *App) GaeaProfileConflicts() []string {
	ps := a.hubProfileStore()
	store := a.hubOfficeStore()
	return ps.DetectConflicts(store.List())
}

// GaeaWhisperMemories 返回轻语（hermes.db）记忆事实只读列表。
// 轻语记忆由 gaea 自己管理，记忆中枢只读浏览，不提供写入。
func (a *App) GaeaWhisperMemories() []WhisperMemoryView {
	facts := whisperdb.LoadFactsFromDB(a.whisperDataRoot)
	out := make([]WhisperMemoryView, 0, len(facts))
	for _, f := range facts {
		out = append(out, WhisperMemoryView{
			ID: f.ID, Domain: f.Domain, Subcategory: f.Subcategory,
			Subject: f.Subject, Summary: f.Summary, Weight: f.Weight,
			Confidence: f.Confidence, Tier: f.Tier, Status: f.Status,
			UpdatedAt: f.UpdatedAt,
		})
	}
	return out
}

// GaeaWhisperEpisodes 返回轻语（hermes.db）情节记忆只读列表（时间倒序）。
func (a *App) GaeaWhisperEpisodes() []WhisperEpisodeView {
	eps, err := whisperdb.LoadEpisodesFromDB(a.whisperDataRoot)
	if err != nil {
		return []WhisperEpisodeView{}
	}
	out := make([]WhisperEpisodeView, 0, len(eps))
	for _, ep := range eps {
		out = append(out, WhisperEpisodeView{
			ID: ep.ID, Summary: ep.Summary,
			DominantEmotion: ep.DominantEmotion, EmotionalIntensity: ep.EmotionalIntensity,
			Keywords: ep.Keywords, StartTurn: ep.StartTurn, EndTurn: ep.EndTurn,
			CreatedAt: ep.CreatedAt, SourceSessionID: ep.SourceSessionID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// GaeaWhisperExportArchive 导出轻语记忆归档（hermes.db → Markdown 按领域/子类分目录）。
// 对齐 ackem archiveExporter：生成 README 索引 + 每个领域/子类一个 .md 文件。
func (a *App) GaeaWhisperExportArchive(dir string) (int, error) {
	facts := whisperdb.LoadFactsFromDB(a.whisperDataRoot)
	if len(facts) == 0 {
		return 0, nil
	}
	wrapped := make([]*whisper.Fact, 0, len(facts))
	for i := range facts {
		wrapped = append(wrapped, &whisper.Fact{MemoryFact: facts[i], Active: true})
	}
	n, err := whisper.WriteArchiveFromFacts(wrapped, nil, dir)
	if err != nil {
		return 0, fmt.Errorf("archive export: %w", err)
	}
	return n, nil
}

// GaeaMemoryHubOverview 返回记忆中枢聚合总览：各库条目数 + 最近更新时间。
func (a *App) GaeaMemoryHubOverview() MemoryHubOverview {
	ov := MemoryHubOverview{}

	if store, err := knowledge.Global().Store(); err == nil {
		ov.KnowledgeCount = len(store.List())
	}
	ov.ProfileCount = len(a.hubProfileStore().All())
	ov.OfficeCount = len(a.hubOfficeStore().List())
	ov.CostCount = len(a.hubCostStore().List())
	ov.WhisperCount = len(whisperdb.LoadFactsFromDB(a.whisperDataRoot))
	ov.PinnedCount = hubPinnedCount()

	// 最近更新时间：知识库条目带 UpdatedAt；办公 facts/画像的时间由各自
	// 后端维护（SQLite updated_at），前端按条目展示。
	var latest time.Time
	if store, err := knowledge.Global().Store(); err == nil {
		for _, s := range store.List() {
			if s.UpdatedAt.After(latest) {
				latest = s.UpdatedAt
			}
		}
	}
	if !latest.IsZero() {
		ov.LatestUpdated = latest.Format("2006-01-02 15:04")
	}
	return ov
}

// hubPinnedCount 返回当前工作区固定常用资料数（项目资料卡片计数）。
func hubPinnedCount() int {
	paths, err := pins.Load(gaeaCwd())
	if err != nil {
		return 0
	}
	return len(paths)
}

// ── 记忆 3D 图谱 ──────────────────────────────────────────────────

// GraphNode 图谱节点（three-forcegraph nodes）。
type GraphNode struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Type string  `json:"type"` // knowledge / profile / office / whisper / material
	Desc string  `json:"desc"`
	Val  float64 `json:"val"` // 节点大小（重要度）
}

// GraphLink 图谱边（three-forcegraph links）。
type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // same-tag / same-category / reference
}

// MemoryGraphView 记忆图谱数据。
type MemoryGraphView struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

const (
	maxGraphNodes       = 400
	maxGraphLinks       = 800
	whisperGraphN       = 80 // 轻语节点上限（按权重取前 N）
	whisperTripleGraphN = 80 // 轻语三元组入图上限（按置信度取前 N）
)

// GaeaMemoryGraph 构建记忆图谱：知识/画像/办公/轻语为节点，
// 同标签/同分类/[[引用]]为边。前端 three-forcegraph 渲染。
func (a *App) GaeaMemoryGraph() MemoryGraphView {
	g := MemoryGraphView{Nodes: []GraphNode{}, Links: []GraphLink{}}
	idSet := make(map[string]bool, maxGraphNodes)
	tagIndex := make(map[string][]string)
	catIndex := make(map[string][]string)
	nameID := make(map[string]string) // 记忆名 → 节点 id（[[引用]]解析用）

	addNode := func(id, name, typ, desc string, val float64) {
		if len(g.Nodes) >= maxGraphNodes || idSet[id] || id == "" {
			return
		}
		idSet[id] = true
		g.Nodes = append(g.Nodes, GraphNode{ID: id, Name: name, Type: typ, Desc: desc, Val: val})
	}
	addLink := func(src, tgt, typ string) {
		if len(g.Links) >= maxGraphLinks || src == "" || tgt == "" || src == tgt {
			return
		}
		for _, l := range g.Links { // 小图线性去重足够
			if (l.Source == src && l.Target == tgt) || (l.Source == tgt && l.Target == src) {
				return
			}
		}
		g.Links = append(g.Links, GraphLink{Source: src, Target: tgt, Type: typ})
	}

	// 知识条目
	if store, err := knowledge.Global().Store(); err == nil {
		for _, e := range store.ReadAll() {
			id := "k:" + e.Name
			addNode(id, e.Title, "knowledge", e.Category+" · "+e.Discipline, 1)
			nameID[e.Name] = id
			if e.Category != "" {
				catIndex[e.Category] = append(catIndex[e.Category], id)
			}
			for _, t := range e.Tags {
				tagIndex[t] = append(tagIndex[t], id)
			}
		}
	}

	// 主脑画像
	for _, m := range a.hubProfileStore().All() {
		id := "p:" + m.Name
		addNode(id, displayName(m.Title, m.Name), "profile", m.Description, 1.2)
		nameID[m.Name] = id
		for _, t := range m.Tags {
			tagIndex[t] = append(tagIndex[t], id)
		}
	}

	// 办公事实
	for _, m := range a.hubOfficeStore().List() {
		id := "o:" + m.Name
		addNode(id, displayName(m.Title, m.Name), "office", m.Description, 1)
		nameID[m.Name] = id
		for _, t := range m.Tags {
			tagIndex[t] = append(tagIndex[t], id)
		}
	}

	// 项目资料：工作区固定常用文件作为资料节点入图（与三脑记忆同图可见）
	if paths, err := pins.Load(gaeaCwd()); err == nil {
		for _, rel := range paths {
			name := filepath.Base(filepath.FromSlash(rel))
			addNode("m:"+rel, name, "material", rel+" · 固定常用资料（新会话自动带入上下文）", 1.0)
		}
	}

	// 轻语事实（按权重取前 N，控制规模）；数据根目录未配置时跳过
	var whisperFacts []whisper.MemoryFact
	if a.whisperDataRootSafe() != "" {
		whisperFacts = whisperdb.LoadFactsFromDB(a.whisperDataRootSafe())
	}
	sort.Slice(whisperFacts, func(i, j int) bool { return whisperFacts[i].Weight > whisperFacts[j].Weight })
	for i, f := range whisperFacts {
		if i >= whisperGraphN {
			break
		}
		addNode("w:"+f.ID, f.Subject, "whisper", f.Summary, 0.8+f.Weight)
	}

	// 轻语知识图谱三元组：实体节点 + 关系边（按置信度取前 N，实体名去重合并）
	var triples []whisper.Triple
	if a.whisperDataRootSafe() != "" {
		triples, _ = whisperdb.LoadTriplesFromDB(a.whisperDataRootSafe())
	}
	sort.Slice(triples, func(i, j int) bool { return triples[i].Confidence > triples[j].Confidence })
	for i, t := range triples {
		if i >= whisperTripleGraphN {
			break
		}
		subjID := "t:" + t.Subject
		objID := "t:" + t.Object
		addNode(subjID, t.Subject, "whisper", "轻语图谱实体", 1)
		addNode(objID, t.Object, "whisper", "轻语图谱实体", 1)
		addLink(subjID, objID, t.Predicate)
	}

	// 成本条目：分类/标签与知识、办公记忆共享边索引（同分类/同标签可互连）
	for _, s := range a.hubCostStore().List() {
		id := "c:" + s.Name
		desc := strings.TrimSpace(s.Category + " · " + s.Unit + " · ¥" + strconv.FormatFloat(s.Price, 'f', -1, 64))
		addNode(id, displayName(s.Title, s.Name), "cost", desc, 1)
		nameID[s.Name] = id
		if s.Category != "" {
			catIndex[s.Category] = append(catIndex[s.Category], id)
		}
		for _, t := range s.Tags {
			tagIndex[t] = append(tagIndex[t], id)
		}
	}

	// 边：同标签（每 tag 限 15 对）
	for _, ids := range tagIndex {
		if len(ids) < 2 {
			continue
		}
		for i := 0; i < len(ids) && i < 15; i++ {
			for j := i + 1; j < len(ids) && j < 16; j++ {
				addLink(ids[i], ids[j], "same-tag")
			}
		}
	}

	// 边：知识同分类（每分类限 10 对）
	for _, ids := range catIndex {
		if len(ids) < 2 {
			continue
		}
		for i := 0; i < len(ids) && i < 10; i++ {
			for j := i + 1; j < len(ids) && j < 11; j++ {
				addLink(ids[i], ids[j], "same-category")
			}
		}
	}

	// 边：[[引用]]（办公/画像 body 指向其他记忆）
	refRe := regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
	for _, m := range a.hubOfficeStore().List() {
		if tid, ok := nameID[m.Name]; ok {
			for _, ref := range refRe.FindAllStringSubmatch(m.Body, -1) {
				if rid, ok2 := nameID[strings.TrimSpace(ref[1])]; ok2 {
					addLink(tid, rid, "reference")
				}
			}
		}
	}
	for _, m := range a.hubProfileStore().All() {
		if tid, ok := nameID[m.Name]; ok {
			for _, ref := range refRe.FindAllStringSubmatch(m.Body, -1) {
				if rid, ok2 := nameID[strings.TrimSpace(ref[1])]; ok2 {
					addLink(tid, rid, "reference")
				}
			}
		}
	}

	return g
}

// displayName 标题回退为 name（与 memory 包 displayTitle 一致）。
func displayName(title, name string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	return strings.ReplaceAll(name, "-", " ")
}

// ── 成本库（记忆中枢扩展库）────────────────────────────────────────

// CostSummary 成本条目轻量视图。
type CostSummary struct {
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Category     string    `json:"category"`
	CategoryPath string    `json:"categoryPath"`
	Unit         string    `json:"unit"`
	Price        float64   `json:"price"`
	// 人材机二级汇总（综合单价子目）：人工费/材料费/机械费（元）。
	LaborFee    float64 `json:"laborFee,omitempty"`
	MaterialFee float64 `json:"materialFee,omitempty"`
	MachineFee  float64 `json:"machineFee,omitempty"`
	// 人材机组成行数（二级明细规模，综合单价子目才有）。
	ComponentCount int `json:"componentCount,omitempty"`
	Spec         string    `json:"spec"`
	Source       string    `json:"source"`
	Region       string    `json:"region,omitempty"`       // 地区（价格三要素）
	PriceDate    string    `json:"priceDate,omitempty"`    // 价格时间/期数
	PriceType    string    `json:"priceType,omitempty"`    // 价格口径：出厂价/到场价/安装综合价
	ValidUntil   string    `json:"validUntil,omitempty"`   // 有效期至
	SourceRow    int       `json:"sourceRow,omitempty"`    // 导入原始行号（0=手动）
	Tags         []string  `json:"tags"`
	Status       string    `json:"status"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CostEntry 完整成本条目。
type CostEntry struct {
	CostSummary
	// 费率（仅展示追溯，不参与计算）：管理费/利润/垫资 为金额（元），税率为百分比。
	ManagementFee float64             `json:"managementFee,omitempty"`
	ProfitFee     float64             `json:"profitFee,omitempty"`
	AdvanceFee    float64             `json:"advanceFee,omitempty"`
	TaxRate       float64             `json:"taxRate,omitempty"`
	Components    []CostComponentView `json:"components,omitempty"`
	Body          string              `json:"body"`
	CreatedAt     time.Time           `json:"createdAt"`
}

// CostComponentView 综合单价子目的人材机组成明细行（二级）。
type CostComponentView struct {
	Kind     string  `json:"kind"`
	Title    string  `json:"title"`
	Unit     string  `json:"unit,omitempty"`
	Quantity float64 `json:"quantity,omitempty"`
	Price    float64 `json:"price,omitempty"`
	Amount   float64 `json:"amount,omitempty"`
	Note     string  `json:"note,omitempty"`
	Sort     int     `json:"sort,omitempty"`
}

// CostCategoryView 成本分类树节点视图。
type CostCategoryView struct {
	ID       int                 `json:"id"`
	ParentID int                 `json:"parentId"`
	Name     string              `json:"name"`
	Sort     int                 `json:"sort"`
	Count    int                 `json:"count"`
	Children []*CostCategoryView `json:"children,omitempty"`
}

// costStoreOverride 测试注入的隔离成本库（避免触碰真实用户库）。
var (
	costStoreOverride    *cost.Store
	costStoreOverrideSet bool
)

// SetCostStoreForTest 注入隔离的成本库存储（测试用）。
func SetCostStoreForTest(s *cost.Store) {
	costStoreOverride = s
	costStoreOverrideSet = true
}

// ResetCostStoreForTest 清除测试注入，恢复真实成本库存储。
func ResetCostStoreForTest() { costStoreOverrideSet = false }

// hubCostStore 构造成本库存储。
func (a *App) hubCostStore() *cost.Store {
	if costStoreOverrideSet {
		return costStoreOverride
	}
	return cost.Open(db.GetDatabase(config.MemoryUserDir()))
}

// GaeaCostList 返回成本条目摘要列表。
func (a *App) GaeaCostList() []CostSummary {
	list := a.hubCostStore().List()
	out := make([]CostSummary, 0, len(list))
	for _, s := range list {
		out = append(out, toCostSummary(s))
	}
	return out
}

// GaeaCostSearch 检索成本条目（关键词 + 分类/状态过滤）。
func (a *App) GaeaCostSearch(query, category, status string) []CostSummary {
	list := a.hubCostStore().Search(query, category, status)
	// 语义召回：关键词召回不足（<3）时用本地 bge-m3 补召回（别名/口语表达）。
	if len(list) < 3 && strings.TrimSpace(query) != "" {
		if sem := a.semanticCostRecall(query, list, 10); len(sem) > 0 {
			list = sem
		}
	}
	// 本地语义精排（Herdsman bge-reranker-v2-m3）：候选多时提升排序精度，
	// 模型不可用/失败自动回退 SQL 结果；纯本地推理，不消耗云端 token。
	if reranked := a.rerankCostSearch(query, list, 20); len(reranked) > 0 {
		list = reranked
	}
	out := make([]CostSummary, 0, len(list))
	for _, s := range list {
		out = append(out, toCostSummary(s))
	}
	return out
}

// GaeaCostGet 返回单条成本条目（未找到返回 nil）。
func (a *App) GaeaCostGet(name string) *CostEntry {
	e, err := a.hubCostStore().Get(name)
	if err != nil || e == nil {
		return nil
	}
	return &CostEntry{
		CostSummary: toCostSummary(cost.Summary{
			Name: e.Name, Title: e.Title, Category: e.Category, CategoryPath: e.CategoryPath,
			Unit: e.Unit, Price: e.Price,
			LaborFee: e.LaborFee, MaterialFee: e.MaterialFee, MachineFee: e.MachineFee,
			Spec: e.Spec, Source: e.Source,
			Region: e.Region, PriceDate: e.PriceDate, PriceType: e.PriceType,
			ValidUntil: e.ValidUntil, SourceRow: e.SourceRow,
			Tags: e.Tags, Status: e.Status, UpdatedAt: e.UpdatedAt,
		}),
		ManagementFee: e.ManagementFee, ProfitFee: e.ProfitFee,
		AdvanceFee: e.AdvanceFee, TaxRate: e.TaxRate,
		Components: toCostComponentViews(e.Components),
		Body:       e.Body, CreatedAt: e.CreatedAt,
	}
}

// GaeaCostSave 保存成本条目。
func (a *App) GaeaCostSave(e CostEntry) error {
	return a.hubCostStore().Save(cost.Entry{
		Name: e.Name, Title: e.Title, Category: e.Category, CategoryPath: e.CategoryPath,
		Unit: e.Unit, Price: e.Price,
		LaborFee: e.LaborFee, MaterialFee: e.MaterialFee, MachineFee: e.MachineFee,
		ManagementFee: e.ManagementFee, ProfitFee: e.ProfitFee,
		AdvanceFee: e.AdvanceFee, TaxRate: e.TaxRate,
		Components: fromCostComponentViews(e.Components),
		Spec:       e.Spec, Source: e.Source,
		Region: e.Region, PriceDate: e.PriceDate, PriceType: e.PriceType,
		ValidUntil: e.ValidUntil, SourceRow: e.SourceRow,
		Tags: e.Tags, Status: e.Status, Body: e.Body, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

// GaeaCostDelete 删除成本条目。
func (a *App) GaeaCostDelete(name string) error {
	return a.hubCostStore().Delete(name)
}

// GaeaCostCategories 返回成本分类树（含每节点直接条目数）。
func (a *App) GaeaCostCategories() []CostCategoryView {
	list := a.hubCostStore().Categories()
	out := make([]CostCategoryView, 0, len(list))
	for _, c := range list {
		out = append(out, toCostCategoryView(&c))
	}
	return out
}

// GaeaCostCategorySave 新建/更新分类节点（id<=0 新建，id>0 改名/排序）。
func (a *App) GaeaCostCategorySave(parentID int, name string, sort int, id int) (int, error) {
	return a.hubCostStore().SaveCategory(parentID, name, sort, id)
}

// GaeaCostCategoryDelete 删除分类节点（有子节点或条目时返回错误）。
func (a *App) GaeaCostCategoryDelete(id int) error {
	return a.hubCostStore().DeleteCategory(id)
}

func toCostSummary(s cost.Summary) CostSummary {
	return CostSummary{
		Name: s.Name, Title: s.Title, Category: s.Category, CategoryPath: s.CategoryPath,
		Unit: s.Unit, Price: s.Price,
		LaborFee: s.LaborFee, MaterialFee: s.MaterialFee, MachineFee: s.MachineFee,
		ComponentCount: s.ComponentCount,
		Spec: s.Spec, Source: s.Source,
		Region: s.Region, PriceDate: s.PriceDate, PriceType: s.PriceType,
		ValidUntil: s.ValidUntil, SourceRow: s.SourceRow,
		Tags: s.Tags, Status: s.Status, UpdatedAt: s.UpdatedAt,
	}
}

func toCostComponentViews(list []cost.Component) []CostComponentView {
	if len(list) == 0 {
		return nil
	}
	out := make([]CostComponentView, 0, len(list))
	for _, c := range list {
		out = append(out, CostComponentView{
			Kind: c.Kind, Title: c.Title, Unit: c.Unit,
			Quantity: c.Quantity, Price: c.Price, Amount: c.Amount,
			Note: c.Note, Sort: c.Sort,
		})
	}
	return out
}

func fromCostComponentViews(list []CostComponentView) []cost.Component {
	if len(list) == 0 {
		return nil
	}
	out := make([]cost.Component, 0, len(list))
	for _, c := range list {
		if strings.TrimSpace(c.Title) == "" {
			continue
		}
		out = append(out, cost.Component{
			Kind: c.Kind, Title: c.Title, Unit: c.Unit,
			Quantity: c.Quantity, Price: c.Price, Amount: c.Amount,
			Note: c.Note, Sort: c.Sort,
		})
	}
	return out
}

func toCostCategoryView(c *cost.CategoryView) CostCategoryView {
	if c == nil {
		return CostCategoryView{}
	}
	v := CostCategoryView{
		ID: c.ID, ParentID: c.ParentID, Name: c.Name, Sort: c.Sort, Count: c.Count,
	}
	for _, ch := range c.Children {
		child := toCostCategoryView(ch)
		v.Children = append(v.Children, &child)
	}
	return v
}

// ── 画像冲突一键裁决 ───────────────────────────────────────────────

// GaeaProfileResolveConflict 裁决画像与办公 facts 的冲突。
// prefer="profile"：删除办公 facts 中的同名 user 事实（以画像为准）；
// prefer="facts"：删除画像（以办公 facts 为准）。
func (a *App) GaeaProfileResolveConflict(name, prefer string) error {
	switch prefer {
	case "profile":
		return a.hubOfficeStore().Delete(name)
	case "facts":
		return a.hubProfileStore().Delete(name)
	default:
		return fmt.Errorf("prefer 必须是 profile 或 facts")
	}
}
