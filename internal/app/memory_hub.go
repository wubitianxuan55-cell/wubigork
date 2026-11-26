package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/pins"
	"github.com/gaea/gaea/internal/whisper"
	whisperdb "github.com/gaea/gaea/internal/whisper/db/repos"
)

// hubRefRe 是记忆事件图谱的 [[引用]] 提取正则（刀E v4.249 普查#20：
// 提为包级，避免每次面板调用重编译）。
var hubRefRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// ── 记忆中枢绑定（主脑前端入口的统一管理接口）───────────────────────
//
// 记忆中枢 = 三脑架构的前端呈现：左脑办公记忆（Hephaestus.db facts）、
// 主脑全局画像（profile）、右脑轻语记忆（hermes.db 只读）。
// 其它库沿用同一"库管理"模式扩展。

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
	ProfileCount int `json:"profileCount"`
	OfficeCount  int `json:"officeCount"`
	WhisperCount int `json:"whisperCount"`
	WenxueCount  int `json:"wenxueCount"`
	PinnedCount  int `json:"pinnedCount"` // 项目资料：工作区固定常用文件数
	// 数据库存档（v4.479）：本设备聊天会话总数（活跃+归档）与上传附件总数，
	// 作用域 = 当前 + 最近工作区（GaeaDatabaseOverview 同口径）。
	SessionCount  int    `json:"sessionCount"`
	UploadCount   int    `json:"uploadCount"`
	LatestUpdated string `json:"latestUpdated"`
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

// nonNilSlice 是 nonNilStrings 的泛型版：绑定面无 error 出口的列表返回值
// 构造即非 nil——Wails 把 nil 切片序列化成 JSON null，前端裸读 .length/.map
// 即崩（v4.467.1 造价清单弹窗、v4.469.2 知识库面板两次真机发作后的
// 造价族按契约扫收敛）。
func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
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

// ── 图文脑（wenxue-brain，2026-11-16 记忆左右脑批新建的独立库）─────────
//
// 记忆中枢只读浏览：物理隔离不动（读写面仍只在图文引擎记忆五件工具，boot
// MemoryUserDir=wenxue-brain 独立 DB），中枢同右脑轻语「只读」先例做跨书
// 全览。MemoryUserDir 不可得（dir 空）=「无图文脑」，Available=false。

// WenxueMemoryFactView 图文脑记忆事实只读视图。
type WenxueMemoryFactView struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Body        string `json:"body"`
	Space       string `json:"space,omitempty"`     // S1 空间归属（work/play）
	UpdatedAt   string `json:"updatedAt,omitempty"` // 最近写入（RFC3339）
	LastUsedAt  string `json:"lastUsedAt,omitempty"`
	Pinned      bool   `json:"pinned,omitempty"`
	Invalid     bool   `json:"invalid,omitempty"` // 9.3 失效标记（原文保留可查）
}

// WenxueMemoryBrowseView 图文脑记忆只读浏览回执。
type WenxueMemoryBrowseView struct {
	Available bool                   `json:"available"`
	StoreDir  string                 `json:"storeDir"`
	Count     int                    `json:"count"`
	Facts     []WenxueMemoryFactView `json:"facts"`
}

// hubWenxueBrainStore 图文脑 facts 存储（跨书全览：List 不过滤项目）。
func (a *App) hubWenxueBrainStore() memory.Store {
	dir := wenxueBrainDir()
	if dir == "" {
		return memory.Store{}
	}
	return memory.SQLiteStoreFor(db.GetDatabase(dir), dir, "")
}

// GaeaWenxueMemoryBrowse 图文脑记忆只读浏览（中枢图文库数据面）。
func (a *App) GaeaWenxueMemoryBrowse() WenxueMemoryBrowseView {
	view := WenxueMemoryBrowseView{Facts: []WenxueMemoryFactView{}}
	dir := wenxueBrainDir()
	if dir == "" {
		return view
	}
	view.Available = true
	view.StoreDir = dir
	for _, m := range a.hubWenxueBrainStore().List() {
		view.Facts = append(view.Facts, WenxueMemoryFactView{
			Name:        m.Name,
			Title:       m.Title,
			Description: m.Description,
			Type:        string(m.Type),
			Body:        m.Body,
			Space:       m.Space,
			UpdatedAt:   fmtTimeOrEmpty(m.UpdatedAt),
			LastUsedAt:  fmtTimeOrEmpty(m.LastUsedAt),
			Pinned:      m.Pinned,
			Invalid:     !m.InvalidAt.IsZero(),
		})
	}
	view.Count = len(view.Facts)
	return view
}

// GaeaMemoryHubOverview 返回记忆中枢聚合总览：各库条目数 + 最近更新时间。
func (a *App) GaeaMemoryHubOverview() MemoryHubOverview {
	ov := MemoryHubOverview{}

	ov.ProfileCount = len(a.hubProfileStore().All())
	ov.OfficeCount = len(a.hubOfficeStore().List())
	ov.WhisperCount = len(whisperdb.LoadFactsFromDB(a.whisperDataRoot))
	if wenxueBrainDir() != "" {
		ov.WenxueCount = len(a.hubWenxueBrainStore().List())
	}
	ov.PinnedCount = hubPinnedCount()
	// 数据库存档计数（v4.479）：跨工作区走查会话/附件目录，只 stat 不读内容。
	for _, root := range databaseRoots() {
		active, archived, _, _ := walkSessionFiles(config.WorkspaceSessionDir(root, ""))
		ov.SessionCount += active + archived
		ups, _, _ := statUploadsDir(filepath.Join(root, ".gaea", "uploads"))
		ov.UploadCount += ups
	}

	// 最近更新时间：办公 facts（与文学脑，若启用）里最近一次写入/修订的时间。
	// 旧成本库退役（2026-11-26）前这里只看成本条目的 UpdatedAt。
	var latest time.Time
	stores := []memory.Store{a.hubOfficeStore()}
	if wenxueBrainDir() != "" {
		stores = append(stores, a.hubWenxueBrainStore())
	}
	for _, st := range stores {
		for _, m := range st.List() {
			if m.UpdatedAt.After(latest) {
				latest = m.UpdatedAt
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
	Type string  `json:"type"` // profile / office / whisper / material
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
	refRe := hubRefRe
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
