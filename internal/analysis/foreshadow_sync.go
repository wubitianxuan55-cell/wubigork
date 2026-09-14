package analysis

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// ── 分析结果 → 伏笔登记表同步（t1-P2 分析驱动自动回收）────────
//
// 口径来源：docs/distill/01-foreshadow-spec.md §3.3（对齐 MuMu
// foreshadow_service.py:1287-1471）。三条核心原则（spec §1.2）：
//  1. 只有埋入会创建记录，回收只更新——resolved 无匹配记跳过，绝不新建；
//  2. 缺失值优于猜值——计划回收章未知就不填，绝不给默认值；
//  3. 精确 ID 失效时宁可不匹配——reference_stable_id 查不到禁止回落内容匹配。
//
// gaea 适配（与 MuMu 的有意偏离，均在码内注明）：
//   - 写入口径状态 = revealed（gaea 原生 wire 值，见 types.ForeshadowRevealed）；
//   - 可回收状态从 planted 放宽到 planted/hinted/partially_resolved
//     （D15 同源口径：hinted 与部分回收也有回收压力）；
//   - 每章新建上限常量唯一声明在 types.ForeshadowMaxNewPerChapter（spec P0.5）。

// SkipReason 单条分析结果被跳过的原因（修正 MuMu 缺陷 D3：跳过必须可追溯，
// 不静默丢弃）。
type SkipReason struct {
	Kind    string `json:"kind"` // invalid_reference|already_resolved|not_planted|no_match|limit_reached|empty_content
	RefID   string `json:"ref_id,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message"`
}

// SyncResult 一轮分析的伏笔同步结果。
type SyncResult struct {
	PlantedCount        int          `json:"planted_count"`
	ResolvedCount       int          `json:"resolved_count"`
	CreatedCount        int          `json:"created_count"`
	UpdatedIDs          []string     `json:"updated_ids"`
	CreatedIDs          []string     `json:"created_ids"`
	MatchedByContent    int          `json:"matched_by_content"`
	SkippedResolveCount int          `json:"skipped_resolve_count"` // 仅回收路径的跳过数（D3 计数）
	SkippedReasons      []SkipReason `json:"skipped_reasons"`
	Errors              []string     `json:"errors"`
}

// foreshadowSkipKinds 跳过类型值域（P2.5 验收：每类均有测试覆盖）。
const (
	skipInvalidReference = "invalid_reference"
	skipAlreadyResolved  = "already_resolved"
	skipNotPlanted       = "not_planted"
	skipNoMatch          = "no_match"
	skipLimitReached     = "limit_reached"
	skipEmptyContent     = "empty_content"
)

// syncForeshadows 将分析出的伏笔变化（v2 契约 types.ForeshadowHit）同步进
// foreshadows.json。合并语义：既有条目（含 manual_ 手工登记）原样保留未被
// 命中的字段，AI 结果只做「更新既有 ID 的状态 / 追加新 ID」，绝不全量覆盖、
// 不重置既有状态；回收动作三级匹配（精确 ID → 内容兜底 → 跳过不新建）。
// 单条容错：一条失败只记 Errors，不中断整批；批次末尾一次性落盘。
func (a *Agent) SyncForeshadows(chapterNum int, hits []types.ForeshadowHit) SyncResult {
	res := SyncResult{
		UpdatedIDs:     []string{},
		CreatedIDs:     []string{},
		SkippedReasons: []SkipReason{},
		Errors:         []string{},
	}
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			// 读取失败（文件损坏/权限等）时放弃本次同步：避免拿空数据覆盖掉
			// 既有伏笔（含手工登记条目）。文件不存在属正常（新项目）。
			res.Errors = append(res.Errors, fmt.Sprintf("读取伏笔失败，本次同步放弃: %v", err))
			slog.Warn("syncForeshadows: 读取伏笔失败，跳过本次同步", "error", err)
			return res
		}
		ff = &types.ForeshadowFile{Items: []types.Foreshadow{}}
	}

	chapterFile := fmt.Sprintf("%03d.md", chapterNum)
	now := time.Now().Format(time.RFC3339)

	// 内容匹配候选池：存活可回收条目（planted/hinted/partially_resolved），
	// 按埋入章升序（同分取先出现者 = 最早埋入的，spec §3.4 采纳规则）。
	// 注意：不等同于候选清单渲染（foreshadow_prompt.go）——include_in_context=false
	// 只是退出注入面，登记表数据完整性不受影响，回收照样可匹配。
	pool := make([]types.Foreshadow, 0, len(ff.Items))
	for i := range ff.Items {
		if foreshadowResolvable(ff.Items[i].Status) {
			pool = append(pool, ff.Items[i])
		}
	}
	sort.SliceStable(pool, func(i, j int) bool {
		return types.ChapterNumOf(pool[i].PlantedIn) < types.ChapterNumOf(pool[j].PlantedIn)
	})

	newCount := 0
	for _, hit := range hits {
		switch strings.ToLower(strings.TrimSpace(hit.Type)) {
		case "planted":
			syncPlant(ff, chapterFile, now, hit, &newCount, &res)
		case "resolved", "revealed": // revealed = gaea 旧 wire 动词，宽容归一
			syncResolve(ff, chapterFile, now, hit, &pool, &res)
		default:
			res.Errors = append(res.Errors, fmt.Sprintf("未知伏笔类型 %q，该条跳过", hit.Type))
		}
	}

	// 清洗：清洗历史文件中已存在的重复 ID（旧版追加语义所致），保留首条
	seen := make(map[string]bool)
	var deduped []types.Foreshadow
	for _, f := range ff.Items {
		if !seen[f.ID] {
			seen[f.ID] = true
			deduped = append(deduped, f)
		}
	}
	ff.Items = deduped
	ff.SchemaVersion = 2

	a.pm.WriteForeshadows(ff)

	a.syncMu.Lock()
	a.lastSync = &res
	a.syncMu.Unlock()
	return res
}

// syncResolve 回收路径三级匹配（spec §3.3 resolved 流程，严格按序）：
// 精确 ID → 内容兜底 → 跳过不新建。
func syncResolve(ff *types.ForeshadowFile, chapterFile, now string, hit types.ForeshadowHit, pool *[]types.Foreshadow, res *SyncResult) {
	skip := func(kind, message string) {
		res.SkippedReasons = append(res.SkippedReasons, SkipReason{
			Kind: kind, RefID: hit.ReferenceStableID, Title: hit.Title, Message: message,
		})
		res.SkippedResolveCount++
	}

	var existing *types.Foreshadow
	if refID := strings.TrimSpace(hit.ReferenceStableID); refID != "" {
		// ① 精确 ID：查不到禁止回落内容匹配（spec §1.2 原则 3，宁漏收不错收）
		existing = lookupByID(ff, refID)
		if existing == nil {
			skip(skipInvalidReference, "reference_stable_id 无效或已删除")
			return
		}
	} else {
		// ② 内容兜底：六策略加权匹配（types.MatchForeshadowByContent 唯一实现）
		matched, _ := types.MatchForeshadowByContent(hit, *pool, 0.5)
		if matched == nil {
			skip(skipNoMatch, "未找到匹配的已埋入伏笔，跳过回收（不创建新记录）")
			return
		}
		existing = lookupByID(ff, matched.ID)
		res.MatchedByContent++
	}
	if existing == nil { // 理论不可达（pool 与 ff 同源），防御性兜底
		skip(skipNoMatch, "候选条目已在同批被消费")
		return
	}

	// ③ 状态闸门：已回收不重置；不可回收状态如实跳过
	if types.IsResolvedStatus(existing.Status) {
		if n := types.ChapterNumOf(existing.RevealedIn); existing.RevealedIn == chapterFile {
			skip(skipAlreadyResolved, "已在本章回收过")
		} else if n > 0 {
			skip(skipAlreadyResolved, fmt.Sprintf("已在第%d章回收", n))
		} else {
			skip(skipAlreadyResolved, "该伏笔已是回收状态")
		}
		return
	}
	if !foreshadowResolvable(existing.Status) {
		skip(skipNotPlanted, fmt.Sprintf("状态为 %q，不是可回收状态", existing.Status))
		return
	}

	// ④ 写回收（写入口径 revealed，gaea wire 值）
	existing.Status = types.ForeshadowRevealed
	existing.RevealedIn = chapterFile
	existing.ResolvedAt = now
	existing.UpdatedAt = now
	if strings.TrimSpace(hit.Content) != "" {
		existing.ResolutionText = hit.Content
	}
	res.ResolvedCount++
	res.UpdatedIDs = append(res.UpdatedIDs, existing.ID)

	// 从候选池移除，防止同批多条 resolved 重复命中同一条（MuMu :1356）
	for i := range *pool {
		if (*pool)[i].ID == existing.ID {
			*pool = append((*pool)[:i], (*pool)[i+1:]...)
			break
		}
	}
}

// syncPlant 埋入路径（spec §3.3 planted 流程）：两道防重 + 每章新建上限。
func syncPlant(ff *types.ForeshadowFile, chapterFile, now string, hit types.ForeshadowHit, newCount *int, res *SyncResult) {
	skip := func(kind, message string) {
		res.SkippedReasons = append(res.SkippedReasons, SkipReason{
			Kind: kind, RefID: hit.ReferenceStableID, Title: hit.Title, Message: message,
		})
	}
	content := strings.TrimSpace(hit.Content)
	// ① 内容为空如实跳过（MuMu :1370-1373）
	if content == "" {
		skip(skipEmptyContent, "伏笔内容为空")
		return
	}
	// ② 标题缺省：内容前 50 字（MuMu :1375-1377）
	title := strings.TrimSpace(hit.Title)
	if title == "" {
		title = truncateRunes(content, 50)
	}
	// ③ 稳定 ID（gaea 唯一实现 GenerateStableID）
	stableID := GenerateStableID(hit.Category, chapterFile, content)

	// ④ 两道防重：稳定 ID 命中（ID 或 SourceMemoryID）；同章同题的分析条目
	existing := lookupByID(ff, stableID)
	if existing == nil {
		existing = lookupByAnalysisDuplicate(ff, title, chapterFile)
	}

	// ⑤ 已存在 → 更新不新建、不占额度、不重置状态（MuMu :1406-1417）
	if existing != nil {
		existing.UpdatedAt = now
		existing.SourceMemoryID = stableID
		if t := strings.TrimSpace(hit.Title); t != "" {
			existing.Title = t
		}
		existing.Description = content
		if kw := strings.TrimSpace(hit.Keyword.Text); kw != "" {
			existing.HintText = kw
		}
		if hit.Category != "" {
			existing.Category = hit.Category
		}
		if hit.Strength > 0 {
			existing.Strength = clampScore10(hit.Strength)
		}
		if hit.Subtlety > 0 {
			existing.Subtlety = clampScore10(hit.Subtlety)
		}
		if len(hit.RelatedChars) > 0 {
			existing.RelatedCharacters = hit.RelatedChars
		}
		if hit.EstimateResolve > 0 {
			existing.TargetResolveIn = fmt.Sprintf("%03d.md", hit.EstimateResolve)
		}
		existing.IsLongTerm = existing.IsLongTerm || hit.IsLongTerm
		return
	}

	// ⑥ 新建：每章上限只约束新建（MuMu :1421-1464）
	if *newCount >= types.ForeshadowMaxNewPerChapter {
		skip(skipLimitReached, fmt.Sprintf("已达每章新伏笔上限(%d个)", types.ForeshadowMaxNewPerChapter))
		return
	}
	strength := clampScore10(hit.Strength) // 缺省语义 5（0 → 5）
	subtlety := clampScore10(hit.Subtlety)
	item := types.Foreshadow{
		ID:          stableID,
		Category:    hit.Category,
		Title:       title,
		Description: content,
		HintText:    strings.TrimSpace(hit.Keyword.Text),

		SourceType:     types.ForeshadowSourceAnalysis,
		SourceMemoryID: stableID,

		PlantedIn: chapterFile,
		Status:    types.ForeshadowPlanted,

		Importance: minFloat(float64(strength)/10.0, 1.0), // 评分派生唯一规则（MuMu :1447）
		Strength:   strength,
		Subtlety:   subtlety,

		IsLongTerm:        hit.IsLongTerm,
		RelatedCharacters: hit.RelatedChars,

		AutoRemind:           boolPtr(true),
		RemindBeforeChapters: types.ForeshadowRemindBeforeDefault,
		IncludeInContext:     boolPtr(true),

		CreatedAt: now,
		UpdatedAt: now,
		PlantedAt: now,
	}
	// 计划回收章：缺失值优于猜值——模型没预估就不填（MuMu :1428-1431）
	if hit.EstimateResolve > 0 {
		item.TargetResolveIn = fmt.Sprintf("%03d.md", hit.EstimateResolve)
	}
	ff.Items = append(ff.Items, item)
	*newCount++
	res.PlantedCount++
	res.CreatedCount++
	res.CreatedIDs = append(res.CreatedIDs, item.ID)
}

// ── 助手 ─────────────────────────────────────────────────────

// foreshadowResolvable 报告状态是否可被回收动作推进（gaea D15 口径：
// hinted 与 partially_resolved 视同 planted 有回收压力；pending 未埋入、
// abandoned 已废弃不可回收）。
func foreshadowResolvable(s types.ForeshadowStatus) bool {
	switch s {
	case types.ForeshadowPlanted, types.ForeshadowHinted, types.ForeshadowPartial:
		return true
	}
	return false
}

// lookupByID 按精确 ID 查条目（返回切片内指针，调用方可直接改写）。
func lookupByID(ff *types.ForeshadowFile, id string) *types.Foreshadow {
	for i := range ff.Items {
		if ff.Items[i].ID == id {
			return &ff.Items[i]
		}
	}
	return nil
}

// lookupByAnalysisDuplicate 防重第二道：同章同题的分析来源条目（MuMu :1385-1402）。
func lookupByAnalysisDuplicate(ff *types.ForeshadowFile, title, chapterFile string) *types.Foreshadow {
	if title == "" {
		return nil
	}
	for i := range ff.Items {
		if ff.Items[i].SourceType == types.ForeshadowSourceAnalysis &&
			ff.Items[i].PlantedIn == chapterFile && ff.Items[i].Title == title {
			return &ff.Items[i]
		}
	}
	return nil
}

// clampScore10 1-10 评分钳制：0/负值取缺省 5，>10 封顶 10。
func clampScore10(v int) int {
	if v <= 0 {
		return 5
	}
	if v > 10 {
		return 10
	}
	return v
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func boolPtr(v bool) *bool { return &v }

// truncateRunes 按 rune 截断（超长补省略号，未超长不加——D14 口径）。
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
