// Package novelcontext 场景圣经 / 上下文编译。
//
// 给定一个场景（或整章），从项目本地文件（角色 / 世界观 / 大纲 / 伏笔 / 风格 /
// 实体数据库）中检索本场景相关的实体与事实，并按 POV 角色做视角掩码，最终编译
// 成一段可注入生成 prompt 的紧凑「场景圣经」，替代 CreateChapter 里扁平的
// prev_summary。
//
// 本包不调用 LLM、不做任何网络请求，纯本地读取与组装，可被 `go test` 无网验证。
package novelcontext

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/graph"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/style"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// ── 预算常量 ────────────────────────────────────────────────
//
// 各区段单独截断用的小预算，合计有意控制在 DefaultMaxRunes 之下，
// 再由 Render 做一次全局兜底截断，保证输出字符数绝不超过调用方给定的 maxRunes。
const (
	// DefaultMaxRunes Render 在未显式传预算时使用的默认总预算（约 2000 rune）。
	DefaultMaxRunes = 2000

	settingBudget     = 240 // 世界观要点
	sceneBudget       = 160 // 本场景元信息
	charBudget        = 420 // 出场角色块
	povViewBudget     = 380 // POV 已知事实
	hiddenBudget      = 160 // POV 不知情事实（约束）
	foreshadowBudget  = 280 // 未回收伏笔
	timeAnchorBudget  = 120 // 时间锚点
	styleBudget       = 240 // 文风指导
	threadBudget      = 100 // 故事主线
	foreshadowLineMax = 90  // 单条伏笔描述截断
	factBudget        = 120 // 单条事实（POVView/HiddenFacts）截断
	characterLineMax  = 140 // 单条出场角色行截断
	charItemsMax      = 4   // 每个角色最多渲染的持有物数量
	charKnownByMax    = 4   // 每个角色最多渲染的知晓者数量
)

// SceneBible 一场景的「场景圣经」——按 POV 裁剪后的紧凑上下文。
type SceneBible struct {
	Setting     string          // 世界观要点（截断）
	Scene       types.SceneMeta // 本场景 POV/地点/时间/情绪/标签
	Characters  []SceneChar     // 本场景出场角色 + 各自状态 + POV 可见信息
	POVView     string          // 仅当前 POV 角色知情的关键事实（视角掩码产物）
	HiddenFacts []string        // POV 不知情、且不得在本场景泄露的事实（生成时约束）
	Foreshadows []string        // 未回收伏笔（创作约束）
	TimeAnchor  string          // 时间锚点（本场景相对上一场景的时间）
	Style       string          // 文风指导（style.LoadProfile → ToStyleGuide，截断）
	Thread      string          // 故事主线（当前必须在推进的主线）
}

// SceneChar 本场景出场角色的一条浓缩信息。
type SceneChar struct {
	Name     string   // 角色名
	RoleType string   // 定位（主角/配角/反派/...）
	Status   string   // 状态（Alive/Dead/Missing/...）
	Location string   // 当前位置
	Items    []string // 持有物
	KnownBy  []string // 谁知晓其关键信息（POV 视角一部分）
}

// ── 编译入口 ────────────────────────────────────────────────

// CompileSceneBible 为一个具体场景编译场景圣经。
// 所有读取失败均静默降级（字段为空串/空切片），绝不中断、绝不 panic。
func CompileSceneBible(pm *project.Manager, chapterNum int, scene *types.Scene) (*SceneBible, error) {
	b := &SceneBible{}
	if pm == nil {
		return b, fmt.Errorf("project.Manager 不能为空")
	}
	if scene == nil {
		return b, fmt.Errorf("scene 不能为空")
	}
	b.Scene = scene.Meta

	// 世界观要点
	b.Setting = buildSetting(pm)

	// 检索子图 + POV 视角掩码（核心差异化）
	db := loadEntityDB(pm)
	entities := collectSceneEntities(pm, chapterNum, scene, db)
	b.Characters = buildSceneChars(scene, entities, charIndex(pm))
	b.POVView, b.HiddenFacts = buildPOVMask(pm, scene, entities)

	// 未回收伏笔
	b.Foreshadows = buildForeshadows(pm)

	// 时间锚点
	b.TimeAnchor = buildTimeAnchor(pm, chapterNum, scene)

	// 文风指导
	b.Style = buildStyle(pm)

	// 故事主线
	b.Thread = buildThread(pm)

	return b, nil
}

// BuildSceneBibleFromChapter 无具体 scene 时，按该章 Stitch 正文 + 大纲节点
// 合成一个「整章整体圣经」：先用章摘要 / 大纲节点 / 场景列表推断出场景元信息，
// 再走 CompileSceneBible。推断失败则返回空区段，不中断。
func BuildSceneBibleFromChapter(pm *project.Manager, chapterNum int) (*SceneBible, error) {
	scene := synthesizeChapterScene(pm, chapterNum)
	return CompileSceneBible(pm, chapterNum, scene)
}

// Render 把场景圣经渲染成一段可直接注入 prompt 的 markdown 文本。
// 任何区段为空时都不输出其标题（绝不出现空区段标题）。整体字数用 util.Truncate
// 兜底截断至 maxRunes（省略号预算已扣除），保证 len(返回) ≤ maxRunes。
func (b *SceneBible) Render(maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = DefaultMaxRunes
	}

	var sb strings.Builder
	addSection := func(title, body string) {
		body = strings.TrimSpace(body)
		if body == "" {
			return
		}
		sb.WriteString("## " + title + "\n\n" + body + "\n\n")
	}

	addSection("世界观要点", b.Setting)
	addSection("本场景", formatScene(b.Scene))

	if len(b.Characters) > 0 {
		var lines []string
		for _, c := range b.Characters {
			lines = append(lines, formatSceneChar(c))
		}
		addSection("出场角色", strings.Join(lines, "\n"))
	}

	addSection("POV 已知事实", b.POVView)

	if len(b.HiddenFacts) > 0 {
		// 这是给生成过程 / 一致性闸门的硬约束：当前 POV 不知情，绝不能写进正文。
		addSection("生成约束 · 当前 POV 不知情（不得泄露）", strings.Join(b.HiddenFacts, "\n"))
	}

	if len(b.Foreshadows) > 0 {
		addSection("未回收伏笔（创作约束）", strings.Join(b.Foreshadows, "\n"))
	}

	addSection("时间锚点", b.TimeAnchor)
	addSection("文风", b.Style)
	addSection("故事主线", b.Thread)

	rendered := sb.String()
	if runeLen := len([]rune(rendered)); runeLen > maxRunes {
		// util.Truncate 会追加 "..."，这里把预算提前扣掉省略号长度，
		// 从而保证最终长度不超过 maxRunes。
		budget := maxRunes - 3
		if budget < 0 {
			budget = 0
		}
		rendered = util.Truncate(rendered, budget)
	}
	return rendered
}

// ── 世界观 / 风格 / 主线 / 伏笔 ────────────────────────────

// buildSetting 组装世界观要点：只取非空 section 的「标题+内容」，截断。
func buildSetting(pm *project.Manager) string {
	wf, err := pm.ReadWorldviewFile()
	if err == nil && wf != nil {
		var parts []string
		for _, sec := range wf.Sections {
			content := strings.TrimSpace(sec.Content)
			if content == "" {
				continue
			}
			parts = append(parts, "**"+sec.Title+"**：\n"+util.Truncate(content, settingBudget/2))
		}
		if len(parts) > 0 {
			return util.Truncate(strings.Join(parts, "\n\n"), settingBudget)
		}
	}
	// 旧版 worldview.md 兜底
	legacy, _ := pm.ReadWorldview()
	if strings.TrimSpace(legacy) != "" {
		return util.Truncate(strings.TrimSpace(legacy), settingBudget)
	}
	return ""
}

// buildStyle 加载风格档案并转成风格指导文本（截断）。读取失败返回空串。
func buildStyle(pm *project.Manager) string {
	profile, err := style.LoadProfile(pm.Dir)
	if err != nil || profile == nil {
		return ""
	}
	body := strings.TrimSpace(profile.ToStyleGuide())
	if body == "" {
		return ""
	}
	return util.Truncate(body, styleBudget)
}

// buildThread 取故事主线（outline.json 的 StoryThread）。读取失败返回空串。
func buildThread(pm *project.Manager) string {
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		return ""
	}
	return util.Truncate(strings.TrimSpace(of.StoryThread), threadBudget)
}

// buildForeshadows 只取 Planting / Hinted（未回收）的伏笔，一行一条，截断。
func buildForeshadows(pm *project.Manager) []string {
	ff, err := pm.ReadForeshadows()
	if err != nil || ff == nil {
		return nil
	}
	var out []string
	for _, f := range ff.Items {
		if f.Status != types.ForeshadowPlanted && f.Status != types.ForeshadowHinted {
			continue
		}
		line := strings.TrimSpace(f.Description)
		if line == "" {
			line = strings.TrimSpace(f.ID)
		}
		if line == "" {
			continue
		}
		out = append(out, util.Truncate(line, foreshadowLineMax))
	}
	return out
}

// buildTimeAnchor 组装一个简短时间锚点：场景自身时间 + 上一章摘要承接。
// 任何一项读取失败都跳过，仍能给出可用的时间基准。
func buildTimeAnchor(pm *project.Manager, chapterNum int, scene *types.Scene) string {
	var parts []string
	if scene != nil && strings.TrimSpace(scene.Meta.TimeOfDay) != "" {
		parts = append(parts, "本场景时间: "+strings.TrimSpace(scene.Meta.TimeOfDay))
	}
	if prev := readPrevSummary(pm, chapterNum); prev != nil {
		if t := strings.TrimSpace(prev.Title); t != "" {
			parts = append(parts, "承接上一章《"+t+"》")
		}
		if s := strings.TrimSpace(prev.Summary); s != "" {
			parts = append(parts, "上一章: "+util.Truncate(s, timeAnchorBudget/2))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return util.Truncate(strings.Join(parts, "\n"), timeAnchorBudget)
}

// ── 实体子图检索 ────────────────────────────────────────────

// loadEntityDB 加载实体数据库，任何错误都降级为空库（不中断）。
func loadEntityDB(pm *project.Manager) *graph.EntityDB {
	db, err := graph.LoadEntityDB(pm.Dir)
	if err != nil || db == nil {
		return &graph.EntityDB{}
	}
	return db
}

// collectSceneEntities 只检索本场景「相关」的实体子图，而不是整个 EntityDB。
// 相关 = 场景自身指向的实体 + 本章出场的角色，四类来源：
//   - POV 角色实体；
//   - 场景地点实体；
//   - 场景正文中实际出现的实体（人物 / 道具 / 地点 / 概念）；
//   - 本章「cast」（章节摘要 CharactersAppeared + 大纲节点 Characters）——
//     用于让「POV 不知情、但系统中存在」的他人事实能进入 HiddenFacts。
//
// 按 ID 去重并按 ID 排序，保证输出确定性（便于测试与缓存）。
func collectSceneEntities(pm *project.Manager, chapterNum int, scene *types.Scene, db *graph.EntityDB) []graph.Entity {
	if db == nil {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]graph.Entity, 0, 4)
	add := func(e *graph.Entity) {
		if e == nil || seen[e.ID] {
			return
		}
		seen[e.ID] = true
		out = append(out, *e)
	}

	if scene == nil {
		return out
	}

	// 1. POV 角色（按 ID 或名字）
	if id := scene.Meta.POVCharID; id != "" {
		add(db.GetByID(id))
		add(db.GetByName(id))
	}

	// 2. 地点
	if loc := strings.TrimSpace(scene.Meta.Location); loc != "" {
		add(db.GetByName(loc))
		for i := range db.Entities {
			if db.Entities[i].Type == graph.EntityLocation &&
				(entityIDEquals(db.Entities[i].ID, loc) || nameEqual(db.Entities[i].Name, loc)) {
				add(&db.Entities[i])
			}
		}
	}

	// 3. 场景正文中出现的实体
	for i := range db.Entities {
		e := &db.Entities[i]
		if e.Name != "" && containsFold(scene.Content, e.Name) {
			add(e)
		}
		if e.ID != "" && containsFold(scene.Content, e.ID) {
			add(e)
		}
	}

	// 4. 本章 cast（供隐藏事实 / 角色块使用）
	for _, chName := range chapterCastNames(pm, chapterNum) {
		add(db.GetByName(chName))
	}

	// 排序保证确定性
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// chapterCastNames 收集本章出场角色名（章节摘要 CharactersAppeared + 大纲节点 Characters）。
// 任何读取失败都返回空，不中断。
func chapterCastNames(pm *project.Manager, chapterNum int) []string {
	var out []string
	seen := make(map[string]bool)
	addName := func(n string) {
		n = strings.TrimSpace(n)
		if n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}

	if cs, err := pm.ReadChapterSummary(chapterNum); err == nil && cs != nil {
		for _, n := range cs.CharactersAppeared {
			addName(n)
		}
	}
	if of, err := pm.ReadOutlines(); err == nil && of != nil {
		for i := range of.Nodes {
			if node := findChapterNode(of.Nodes[i], chapterNum); node != nil {
				for _, n := range node.Characters {
					addName(n)
				}
				break
			}
		}
	}
	return out
}

// ── 角色块 ──────────────────────────────────────────────────

// buildSceneChars 组装本场景出场角色。候选 = 子图中 character 类实体 + POV 角色
// （即便实体库没建，也从角色文件补上）。字段来自 entity.Properties，缺失时以
// characters.json 兜底。
// buildSceneChars 组装本场景出场角色。候选 = 子图中 character 类实体 + POV 角色
// （即便实体库没建，也从角色文件补上）。字段来自 entity.Properties，缺失时以
// characters.json 兜底。charByName 为角色索引（id/name -> Character）。
func buildSceneChars(scene *types.Scene, entities []graph.Entity, charByName map[string]types.Character) []SceneChar {
	byID := make(map[string]graph.Entity, len(entities))
	for _, e := range entities {
		byID[e.ID] = e
	}

	var chars []SceneChar
	added := make(map[string]bool)
	addChar := func(e graph.Entity) {
		if seen := e.ID; seen != "" && added[seen] {
			return
		}
		if e.ID != "" {
			added[e.ID] = true
		}
		chars = append(chars, sceneCharFromEntity(e, charByName))
	}

	// 1. 子图中的 character 实体
	for _, e := range entities {
		if e.Type == graph.EntityCharacter {
			addChar(e)
		}
	}

	// 2. POV 角色即使不出现在实体库，也补进来
	if scene != nil && scene.Meta.POVCharID != "" {
		if e, ok := byID[scene.Meta.POVCharID]; ok && e.ID != "" {
			addChar(e)
		} else if c, ok := charByName[scene.Meta.POVCharID]; ok {
			addChar(entityFromCharacter(c, scene.Meta.POVCharID))
		}
	}

	// 按名排序，保证确定性
	sort.Slice(chars, func(i, j int) bool { return chars[i].Name < chars[j].Name })

	// 若全局都没有 sceneChar 候选，回退到角色文件中本场景出现的角色名
	if len(chars) == 0 {
		for _, c := range charByName {
			if scene != nil && containsFold(scene.Content, c.Name) {
				chars = append(chars, sceneCharFromEntity(entityFromCharacter(c, c.ID), charByName))
			}
		}
		sort.Slice(chars, func(i, j int) bool { return chars[i].Name < chars[j].Name })
	}

	return chars
}

// charIndex 返回角色索引（id/name -> Character）。
func charIndex(pm *project.Manager) map[string]types.Character {
	return loadCharIndex(pm)
}

// entityFromCharacter 把 types.Character 转成一个临时实体（当实体库未建该角色时用）。
func entityFromCharacter(c types.Character, id string) graph.Entity {
	props := map[string]string{
		"role_type": c.RoleType,
		"status":    c.Status,
	}
	return graph.Entity{
		ID:         id,
		Name:       c.Name,
		Type:       graph.EntityCharacter,
		Properties: props,
	}
}

// sceneCharFromEntity 由实体（+角色文件兜底）构造 SceneChar。
func sceneCharFromEntity(e graph.Entity, charByName map[string]types.Character) SceneChar {
	sc := SceneChar{Name: e.Name}
	if e.Properties != nil {
		sc.RoleType = e.Properties["role_type"]
		sc.Status = e.Properties["status"]
		sc.Location = e.Properties["location"]
		sc.Items = splitList(e.Properties["items"])
		sc.KnownBy = splitList(e.Properties["known_by"])
	}
	// 角色文件兜底
	if c, ok := charByName[e.Name]; ok {
		if sc.RoleType == "" {
			sc.RoleType = c.RoleType
		}
		if sc.Status == "" {
			sc.Status = c.Status
		}
	}
	if sc.RoleType == "" {
		sc.RoleType = "角色"
	}
	return sc
}

// loadCharIndex 建索引：id->Character、name->Character（后者可能被覆盖，但查找足够）。
func loadCharIndex(pm *project.Manager) map[string]types.Character {
	idx := make(map[string]types.Character)
	cf, err := pm.ReadCharacters()
	if err != nil || cf == nil {
		return idx
	}
	for _, c := range cf.Characters {
		idx[c.ID] = c
		if _, exists := idx[c.Name]; !exists {
			idx[c.Name] = c
		}
	}
	return idx
}

// ── POV 视角掩码（核心差异化）────────────────────────────────

// buildPOVMask 依据 scene.Meta.POVCharID 生成本场景的视角掩码：
//   - POVView:     只包含当前 POV 角色「已知」的关键事实；
//   - HiddenFacts: POV 不知情、且系统里存在的关键事实（生成时约束，不得泄露）。
//
// 未设置 POVCharID 时不做掩码：所有子图事实视为可见，HiddenFacts 为空。
func buildPOVMask(pm *project.Manager, scene *types.Scene, entities []graph.Entity) (povView string, hidden []string) {
	if scene == nil {
		return "", nil
	}
	povID := strings.TrimSpace(scene.Meta.POVCharID)
	povName := resolveCharName(pm, povID)

	var knownParts []string
	var hiddenParts []string

	for _, e := range entities {
		if e.Properties == nil {
			continue
		}
		// 排序键，保证确定性输出
		for _, key := range sortedKeys(e.Properties) {
			// 元数据字段（known_by / name）本身不算一条事实
			if key == "known_by" || key == "name" {
				continue
			}
			val := strings.TrimSpace(e.Properties[key])
			if val == "" {
				continue
			}
			fact := util.Truncate(formatFact(e, key, val), factBudget)
			if povID == "" || povKnowsFact(povID, povName, scene, e, key, val) {
				knownParts = append(knownParts, fact)
			} else {
				hiddenParts = append(hiddenParts, fact)
			}
		}
	}

	if len(knownParts) > 0 {
		povView = util.Truncate(strings.Join(knownParts, "\n"), povViewBudget)
	}
	return povView, hiddenParts
}

// povKnowsFact 判定实体 e 的某条属性事实是否被当前 POV 角色知晓。
//
// 推断规则（保守优先：宁可少给已知，绝不把 POV 不知情的秘密泄漏进正文）：
//
//  1. 实体本身就是 POV 角色（ID 或名字匹配）→ 已知（角色自知自身状态）。
//  2. 实体属性显式声明 known_by，且其中包含 POV 角色（ID 或名字）→ 已知
//     （作者 / 系统显式授权该 POV 知情）。
//  3. 事实键属于「可观测键」（如 位置/外观/定位/性别/年龄/当前状态/描述），
//     且实体在当前场景中可见（出现在场景正文 / 场景地点 / 即为 POV 自己）→ 已知
//     （POV 可当场直接观察到）。
//  4. 其它情况（其它角色的秘密 / 私密状态 / 未在场景中出现的实体事实）→
//     POV 不知情 → 归入 HiddenFacts。
func povKnowsFact(povID, povName string, scene *types.Scene, e graph.Entity, key, val string) bool {
	// 规则 1：POV 自己的实体
	if e.ID == povID || nameEqual(e.Name, povName) {
		return true
	}

	// 规则 2：显式 known_by 授权
	if kb := e.Properties["known_by"]; kb != "" {
		for _, knower := range splitList(kb) {
			if knower == povID || nameEqual(knower, povName) {
				return true
			}
		}
	}

	// 规则 3：可观测键 + 场景中可见
	if isObservableKey(key) && isVisibleInScene(scene, e) {
		return true
	}

	// 规则 4：保守 → 秘密
	return false
}

// isObservableKey 判定某属性键是否属于「POV 当场可观察到」的公开属性。
// 注意：secret / 计划 / 秘密交易等键不在此列——即使用 known_by 标注才知情。
func isObservableKey(key string) bool {
	switch key {
	case "status", "location", "role_type", "gender", "age",
		"appearance", "figure", "description", "motto", "power_level":
		return true
	default:
		return false
	}
}

// isVisibleInScene 判定实体是否在当前场景中可见：
// 名字出现在场景正文 / 实体即场景地点 / 实体即 POV 自己。
func isVisibleInScene(scene *types.Scene, e graph.Entity) bool {
	if scene == nil {
		return false
	}
	if e.ID != "" && e.ID == scene.Meta.POVCharID {
		return true
	}
	if e.Name != "" && containsFold(scene.Content, e.Name) {
		return true
	}
	if e.ID != "" && containsFold(scene.Content, e.ID) {
		return true
	}
	if loc := strings.TrimSpace(scene.Meta.Location); loc != "" {
		if nameEqual(e.Name, loc) || entityIDEquals(e.ID, loc) {
			return true
		}
	}
	return false
}

// formatFact 渲染一条实体事实为「实体 · 标签: 值」。
func formatFact(e graph.Entity, key, val string) string {
	return fmt.Sprintf("%s · %s: %s", e.Name, factLabel(key), val)
}

// factLabel 把属性键映射为中文标签。
func factLabel(key string) string {
	switch key {
	case "status":
		return "状态"
	case "location":
		return "位置"
	case "items":
		return "持有物"
	case "secret":
		return "秘密"
	case "plans":
		return "计划"
	case "role_type":
		return "定位"
	default:
		return key
	}
}

// resolveCharName 解析角色名（用于名字匹配），失败时返回空串。
func resolveCharName(pm *project.Manager, id string) string {
	if id == "" {
		return ""
	}
	idx := loadCharIndex(pm)
	if c, ok := idx[id]; ok {
		return c.Name
	}
	return id
}

// ── 整章合成（BuildSceneBibleFromChapter 用）───────────────

// synthesizeChapterScene 无具体 scene 时，用章摘要 / 大纲节点 / 场景列表
// 合成一个代表「整章」的场景对象。任何推断失败都不影响返回非 nil 的 scene。
func synthesizeChapterScene(pm *project.Manager, chapterNum int) *types.Scene {
	meta := types.SceneMeta{}

	// 从大纲节点取信息
	if of, err := pm.ReadOutlines(); err == nil && of != nil {
		for i := range of.Nodes {
			if node := findChapterNode(of.Nodes[i], chapterNum); node != nil {
				meta.Title = node.Title
				meta.Summary = node.Summary
				meta.Emotion = node.Emotion
				if meta.Location == "" {
					meta.Location = inferLocation(*node)
				}
				if len(node.Characters) > 0 {
					meta.POVCharID = node.Characters[0]
				}
				break
			}
		}
	}

	// 从章摘要取信息
	if cs, err := pm.ReadChapterSummary(chapterNum); err == nil && cs != nil {
		if meta.Title == "" {
			meta.Title = cs.Title
		}
		if meta.Summary == "" {
			meta.Summary = cs.Summary
		}
		if meta.Emotion == "" {
			meta.Emotion = cs.EmotionTone
		}
	}

	// 从场景列表合成 POV / 地点 / 时间 / 基调
	if sm := pm.SceneManager(chapterNum); sm != nil {
		if metas, err := sm.List(); err == nil && len(metas) > 0 {
			if meta.Location == "" {
				meta.Location = metas[0].Location
			}
			if meta.TimeOfDay == "" {
				meta.TimeOfDay = metas[0].TimeOfDay
			}
			if meta.Emotion == "" {
				meta.Emotion = metas[0].Emotion
			}
			if meta.POVCharID == "" {
				meta.POVCharID = metas[0].POVCharID
			}
			if meta.Summary == "" {
				meta.Summary = metas[0].Summary
			}
		}
	}

	content, _ := pm.ReadChapterAsStitch(chapterNum)
	return &types.Scene{Meta: meta, Content: content}
}

// findChapterNode 在（可能嵌套的）大纲树中查找对应章节号的节点。
func findChapterNode(node types.OutlineNode, chapterNum int) *types.OutlineNode {
	if node.ChapterFile != "" && chapterNumFromFile(node.ChapterFile) == chapterNum {
		return &node
	}
	for i := range node.Children {
		if child := findChapterNode(node.Children[i], chapterNum); child != nil {
			return child
		}
	}
	return nil
}

// chapterNumFromFile 从章节文件名（如 "001.md" / "001a.md"）提取章节号。
func chapterNumFromFile(file string) int {
	var n, started int
	for _, r := range file {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
			started = 1
		} else if started == 1 {
			break
		}
	}
	return n
}

// inferLocation 从大纲节点摘要中粗猜场景地点：若摘要里带「在/于某地」则截取之，
// 否则返回空（由场景列表 / 章摘要兜底）。
func inferLocation(node types.OutlineNode) string {
	s := strings.TrimSpace(node.Summary)
	if s == "" {
		return ""
	}
	for _, marker := range []string{"在", "于", "来到", "抵达"} {
		if idx := strings.Index(s, marker); idx >= 0 {
			rest := []rune(s[idx+len(marker):])
			if len(rest) > 12 {
				rest = rest[:12]
			}
			return strings.TrimSpace(string(rest))
		}
	}
	return ""
}

// ── 渲染辅助 ────────────────────────────────────────────────

// formatScene 渲染场景元信息为一个紧凑块。
func formatScene(m types.SceneMeta) string {
	var bits []string
	if m.Title != "" {
		bits = append(bits, "场景「"+m.Title+"」")
	}
	if m.POVCharID != "" {
		bits = append(bits, "POV: "+m.POVCharID)
	}
	if m.Location != "" {
		bits = append(bits, "地点: "+m.Location)
	}
	if m.TimeOfDay != "" {
		bits = append(bits, "时间: "+m.TimeOfDay)
	}
	if m.Emotion != "" {
		bits = append(bits, "基调: "+m.Emotion)
	}
	if len(m.Tags) > 0 {
		bits = append(bits, "标签: "+strings.Join(m.Tags, "/"))
	}
	if m.Summary != "" {
		bits = append(bits, util.Truncate(m.Summary, 80))
	}
	return util.Truncate(strings.Join(bits, " · "), sceneBudget)
}

// formatSceneChar 渲染一条出场角色行。
func formatSceneChar(c SceneChar) string {
	var sb strings.Builder
	sb.WriteString("- " + c.Name)
	if c.RoleType != "" {
		sb.WriteString(" [" + c.RoleType + "]")
	}
	var bits []string
	if c.Status != "" {
		bits = append(bits, "状态: "+c.Status)
	}
	if c.Location != "" {
		bits = append(bits, "位置: "+c.Location)
	}
	if len(c.Items) > 0 {
		items := c.Items
		if len(items) > charItemsMax {
			items = items[:charItemsMax]
		}
		bits = append(bits, "持有: "+strings.Join(items, "、"))
	}
	if len(c.KnownBy) > 0 {
		known := c.KnownBy
		if len(known) > charKnownByMax {
			known = known[:charKnownByMax]
		}
		bits = append(bits, "知晓者: "+strings.Join(known, "、"))
	}
	if len(bits) > 0 {
		sb.WriteString(" · " + strings.Join(bits, " · "))
	}
	return util.Truncate(sb.String(), characterLineMax)
}

// ── 小工具 ──────────────────────────────────────────────────

// splitList 按常见分隔符拆分列表字段（逗号/分号/竖线/换行），去空去空白。
func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '|' || r == '\n'
	}) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// sortedKeys 返回 map 的有序键列表（保证确定性输出）。
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// readPrevSummary 读取上一章摘要：优先 chapterNum-1，退化为任何小于本章的最近摘要。
func readPrevSummary(pm *project.Manager, chapterNum int) *types.ChapterSummary {
	if chapterNum > 1 {
		if s, err := pm.ReadChapterSummary(chapterNum - 1); err == nil && s != nil {
			return s
		}
	}
	all, err := pm.ReadAllChapterSummaries()
	if err != nil {
		return nil
	}
	var best *types.ChapterSummary
	for i := range all {
		if s := &all[i]; s != nil && s.Title != "" {
			best = s // ReadAllChapterSummaries 已是章节序，取最后一个即最近已写章
		}
	}
	return best
}

// containsFold 是否包含子串（忽略大小写）。
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// nameEqual 名字相等（忽略大小写）。
func nameEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// entityIDEquals 实体 ID 相等，兼容「lorebook:xxx」等前缀以及大小写。
func entityIDEquals(a, b string) bool {
	return nameEqual(a, b) || strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}
