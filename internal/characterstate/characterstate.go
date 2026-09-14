// Package characterstate 角色状态机差分更新器（t5 首刀）。
//
// 口径来源：docs/distill/05-character-career.md §3.3~§3.6/§7.2/§7.3。
// 本质 =「LLM 章节分析产出稀疏差分 → 后端按章节号单调守卫写回」。
// 七条不变量（spec §7.2 表）：
//  1. 心理状态水位单调；2. 存活状态水位单调；3. 存活短路（死亡/失踪后退场
//     不再改心理/关系/组织）；4. 存活级联三件套（关系→past+EndedAt、成员→
//     终态+LeftAt+Notes 追加）；5. 职业阶段 ∈ [1,MaxStage] 钳制；6. 副职业
//     上限 MaxSubCareers=2（修 MuMu 四处不一致）；7. 职业推进也带水位守卫
//     （MuMu 缺失，gaea 新增）。
//
// 纯函数：吃 CharacterFile 副本、返回净产出，落盘由调用方负责。
package characterstate

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/types"
)

// MaxSubCareers 副职业上限（spec §7.2 #6：统一 MuMu 四处不一致的口径）。
const MaxSubCareers = 2

// survivalDesc 存活状态中文描述（MuMu :205-209）。
var survivalDesc = map[string]string{
	"Dead": "死亡", "Missing": "失踪", "Retired": "退场", "Alive": "复活回归",
}

// survivalFromAnalysis 分析侧 survival_status（active|deceased|missing|retired）
// → gaea Character.Status 值域（Alive/Dead/Missing/Retired）。空返回空。
func survivalFromAnalysis(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "deceased":
		return "Dead"
	case "missing":
		return "Missing"
	case "retired":
		return "Retired"
	case "active":
		return "Alive"
	}
	return ""
}

// ── 亲密度算法（§7.3：最长匹配优先，禁子串累加）────────────────

// intimacyDict 亲密度词典（MuMu :13-26 全表；运行时按长度降序，最长匹配优先
// 命中即停——修 MuMu 子串累加缺陷：「不信任」同时命中「信任」+10 与
// 「不信任」-15 → 净 -5 的失真）。
var intimacyDict = []struct {
	kw    string
	delta int
}{
	{"和解", 20}, {"喜欢", 15}, {"亲近感", 15}, {"亲近", 15}, {"亲密", 15}, {"加深", 15},
	{"背叛", -30}, {"决裂", -30}, {"仇恨", -25}, {"破裂", -25}, {"反目", -25}, {"敌对", -25},
	{"厌恶", -20}, {"疏远", -15}, {"冲突", -15}, {"不信任", -15}, {"信任", 10}, {"友好", 10},
	{"认可", 10}, {"尊敬", 10}, {"感激", 10}, {"好转", 10}, {"增进", 10}, {"忠诚", 10},
	{"结盟", 10}, {"改善", 10}, {"合作", 5}, {"恶化", -10}, {"矛盾", -10}, {"怀疑", -10},
	{"猜忌", -10}, {"嫉妒", -10}, {"紧张", -5}, {"分离", -5}, {"初识", 0}, {"相遇", 0},
}

// CalcIntimacyDelta 关系变化描述 → 单次亲密度调整值（已钳 ±30）。
// 最长匹配优先：取命中的最长关键词的 delta，不跨词累加。
func CalcIntimacyDelta(desc string) int {
	best, bestLen := 0, 0
	for _, e := range intimacyDict {
		l := len([]rune(e.kw))
		if l > bestLen && strings.Contains(desc, e.kw) {
			best, bestLen = e.delta, l
		}
	}
	return types.ClampDeltaHint(best)
}

// relationshipDelta 关系增量：LLM DeltaHint 优先（≠0 直接采纳，钳 ±30），
// 缺失退回关键词词典（§7.3 更优解 A/B 并用）。
func relationshipDelta(hint int, desc string) int {
	if hint != 0 {
		return types.ClampDeltaHint(hint)
	}
	return CalcIntimacyDelta(desc)
}

// ── 差分应用主入口 ────────────────────────────────────────────

// ApplyChapterDiff 按 MuMu 语义把分析差分应用到角色文件（纯函数：
// 就地改写 cf，落盘由调用方负责）。三阶段严格顺序：
// 存活（短路+级联）→ 心理 → 关系 → 职业推进 → 组织成员 → 组织自身。
func ApplyChapterDiff(cf *types.CharacterFile, chapterNum int,
	states []types.CharacterStateDiff, rels []types.RelationshipChange, orgs []types.OrgStateDiff) *types.ChapterDiffResult {

	res := &types.ChapterDiffResult{Changes: []string{}, Skipped: []string{}}
	if cf == nil || chapterNum <= 0 {
		res.Skipped = append(res.Skipped, fmt.Sprintf("非法输入（file=%v chapter=%d）", cf != nil, chapterNum))
		return res
	}
	if len(states) == 0 && len(rels) == 0 && len(orgs) == 0 {
		return res // 空输入短路（MuMu :53-61）
	}

	charsByName := make(map[string]*types.Character, len(cf.Characters))
	for i := range cf.Characters {
		charsByName[cf.Characters[i].Name] = &cf.Characters[i]
	}
	orgsByName := make(map[string]*types.Organization, len(cf.Organizations))
	for i := range cf.Organizations {
		orgsByName[cf.Organizations[i].Name] = &cf.Organizations[i]
	}

	// ── 角色差分 ──
	for _, d := range states {
		c := charsByName[d.Name]
		if c == nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("角色 %q 不存在，跳过", d.Name))
			continue
		}
		// 0. 存活状态（短路优先）
		if d.Status != "" {
			if !types.SurvivalStatusAdvanceAllowed(*c, chapterNum) {
				res.Skipped = append(res.Skipped, fmt.Sprintf("%s 存活状态已在第%d章变更，跳过", d.Name, c.StatusChangedChapter))
			} else {
				desc := survivalDesc[d.Status]
				c.Status = d.Status
				c.StatusChangedChapter = chapterNum
				c.CurrentState = fmt.Sprintf("%s（第%d章）", desc, chapterNum)
				c.StateUpdatedChapter = chapterNum // 同步推高心理水位
				res.StateUpdated++
				line := fmt.Sprintf("%s %s", d.Name, desc)
				if d.KeyEvent != "" {
					line += "：" + truncateRunes(d.KeyEvent, 50)
				}
				res.Changes = append(res.Changes, line)
				cascadeSurvival(cf, c, chapterNum, desc, res)
				continue // 短路：退场后不再更新心理/关系/职业
			}
		}
		// 1. 心理状态（独立水位守卫）
		if d.NewState != "" {
			if !types.StateAdvanceAllowed(*c, chapterNum) {
				res.Skipped = append(res.Skipped, fmt.Sprintf("%s 心理状态已被第%d章更新，跳过", d.Name, c.StateUpdatedChapter))
			} else {
				c.CurrentState = d.NewState
				c.StateUpdatedChapter = chapterNum
				res.StateUpdated++
				res.Changes = append(res.Changes, fmt.Sprintf("%s 心理状态 → %s", d.Name, d.NewState))
			}
		}
		// 2. 职业阶段推进（水位守卫 + 钳制；spec #5/#7）
		for _, inc := range d.CareerIncs {
			applyCareerInc(c, chapterNum, inc, res)
		}
	}

	// ── 关系差分（双向无向语义）──
	for _, rc := range rels {
		from, to := charsByName[rc.FromName], charsByName[rc.ToName]
		if from == nil || to == nil || rc.ChangeDesc == "" {
			res.Skipped = append(res.Skipped, fmt.Sprintf("关系 %s→%s 跳过（角色缺失或描述为空）", rc.FromName, rc.ToName))
			continue
		}
		if i := findBidirectionalRel(cf, rc.FromName, rc.ToName); i >= 0 {
			rel := &cf.Relationships[i]
			delta := relationshipDelta(rc.DeltaHint, rc.ChangeDesc)
			rel.Description = appendRelNote(rel.Description, chapterNum, rc.ChangeDesc)
			rel.Intimacy = types.ClampIntimacy(rel.Intimacy + delta)
			rel.History = append(rel.History, types.RelChange{Chapter: chapterNum, Description: rc.ChangeDesc})
			if rc.Status != "" {
				rel.Status = rc.Status
			}
			if rc.NewType != "" {
				rel.RelationType = rc.NewType
			}
			res.RelUpdated++
			res.Changes = append(res.Changes, fmt.Sprintf("%s↔%s 关系更新：%s（亲密度 %d）", rc.FromName, rc.ToName, rc.ChangeDesc, rel.Intimacy))
		} else {
			delta := relationshipDelta(rc.DeltaHint, rc.ChangeDesc)
			cf.Relationships = append(cf.Relationships, types.Relationship{
				FromID: from.ID, ToID: to.ID,
				Description: fmt.Sprintf("[第%d章] %s", chapterNum, rc.ChangeDesc),
				Intimacy:    types.ClampIntimacy(50 + delta), // 基线 50（MuMu :426）
				Status:      "active", StartedAt: fmt.Sprintf("第%d章", chapterNum),
				History: []types.RelChange{{Chapter: chapterNum, Description: rc.ChangeDesc}},
			})
			res.RelCreated++
			res.Changes = append(res.Changes, fmt.Sprintf("%s↔%s 新建关系：%s", rc.FromName, rc.ToName, rc.ChangeDesc))
		}
	}

	// ── 组织差分 ──
	for _, od := range orgs {
		o := orgsByName[od.OrgName]
		if o == nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("组织 %q 不存在，跳过", od.OrgName))
			continue
		}
		if od.PowerValue != nil {
			pv := *od.PowerValue
			if pv < 0 {
				pv = 0
			}
			if pv > 100 {
				pv = 100
			}
			o.PowerValue = pv
			res.OrgStateUpdated++
		}
		if od.Destroyed != nil && *od.Destroyed {
			o.Destroyed = true
			o.DestroyedChapter = chapterNum
			res.OrgStateUpdated++
			res.Changes = append(res.Changes, fmt.Sprintf("组织 %s 覆灭（第%d章）", o.Name, chapterNum))
		}
		for _, mc := range od.Members {
			if applyOrgMemberChange(o, chapterNum, mc) {
				res.OrgMemberUpdated++
			}
		}
	}
	return res
}

// cascadeSurvival 存活级联三件套（§3.4）：active 关系 → past+EndedAt；
// active 组织成员 → deceased/retired + LeftAt + Notes 追加时间线。
func cascadeSurvival(cf *types.CharacterFile, c *types.Character, chapterNum int, desc string, res *types.ChapterDiffResult) {
	at := fmt.Sprintf("第%d章", chapterNum)
	for i := range cf.Relationships {
		rel := &cf.Relationships[i]
		if rel.FromID != c.ID && rel.ToID != c.ID {
			continue
		}
		if rel.Status != "" && rel.Status != "active" {
			continue
		}
		rel.Status = "past"
		rel.EndedAt = at
	}
	memberStatus := "retired"
	if c.Status == "Dead" {
		memberStatus = "deceased"
	}
	for oi := range cf.Organizations {
		o := &cf.Organizations[oi]
		for mi := range o.MemberList {
			m := &o.MemberList[mi]
			if m.CharacterID != c.ID || (m.Status != "" && m.Status != "active") {
				continue
			}
			m.Status = memberStatus
			m.LeftAt = at
			m.Notes = strings.TrimSpace(m.Notes + fmt.Sprintf("\n[第%d章] 角色%s", chapterNum, desc))
		}
	}
	res.Changes = append(res.Changes, fmt.Sprintf("%s 级联：关系终态化 + 组织成员终态化", c.Name))
}

// applyCareerInc 单条职业推进：水位守卫 + 主职业唯一 + 副职业上限 2 + 阶段 ≥1。
func applyCareerInc(c *types.Character, chapterNum int, inc types.CareerStageChange, res *types.ChapterDiffResult) {
	if inc.NewStage < 1 {
		inc.NewStage = 1
	}
	if inc.IsMain {
		if c.MainCareerID != "" && c.MainCareerID != inc.CareerID {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s 主职业冲突（现 %s），拒绝替换", c.Name, c.MainCareerID))
			return
		}
		// 主职业水位未落独立字段（Character 无 MainCareerUpdatedChapter）——
		// 守卫退化为「同章幂等覆盖、跨章允许」；副职业的 UpdatedChapter 守卫完整。
		c.MainCareerID = inc.CareerID
		c.MainCareerStage = inc.NewStage
		res.CareerUpdated++
		res.Changes = append(res.Changes, fmt.Sprintf("%s 主职业推进至 %d 阶", c.Name, inc.NewStage))
		return
	}
	// 副职业：找既有引用或新增（上限 MaxSubCareers）
	idx := -1
	for i := range c.SubCareers {
		if c.SubCareers[i].CareerID == inc.CareerID {
			idx = i
			break
		}
	}
	if idx < 0 {
		if len(c.SubCareers) >= MaxSubCareers {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s 副职业已达上限 %d", c.Name, MaxSubCareers))
			return
		}
		c.SubCareers = append(c.SubCareers, types.CharacterCareerRef{CareerID: inc.CareerID})
		idx = len(c.SubCareers) - 1
	}
	ref := c.SubCareers[idx]
	if !types.CareerStageAdvanceAllowed(ref, chapterNum) {
		res.Skipped = append(res.Skipped, fmt.Sprintf("%s 副职业 %s 已在第%d章后推进，跳过", c.Name, inc.CareerID, ref.UpdatedChapter))
		return
	}
	c.SubCareers[idx].Stage = inc.NewStage
	c.SubCareers[idx].UpdatedChapter = chapterNum
	res.CareerUpdated++
	res.Changes = append(res.Changes, fmt.Sprintf("%s 副职业 %s 推进至 %d 阶", c.Name, inc.CareerID, inc.NewStage))
}

// applyOrgMemberChange 组织成员变更（joined/left/expelled/betrayed/promoted/status）。
// 返回是否产生变更。
func applyOrgMemberChange(o *types.Organization, chapterNum int, mc types.OrgMemberChange) bool {
	at := fmt.Sprintf("第%d章", chapterNum)
	find := func() *types.OrgMember {
		for i := range o.MemberList {
			if o.MemberList[i].CharacterID == mc.CharacterID {
				return &o.MemberList[i]
			}
		}
		return nil
	}
	switch mc.ChangeType {
	case types.OrgMemberJoined:
		if m := find(); m != nil {
			m.Status = "active"
			return true
		}
		loyalty := 50
		if mc.LoyaltyHint != nil {
			loyalty = clamp100(*mc.LoyaltyHint)
		}
		o.MemberList = append(o.MemberList, types.OrgMember{
			CharacterID: mc.CharacterID, Position: mc.Position, Rank: mc.Rank,
			Status: "active", Loyalty: loyalty, JoinedAt: at, UpdatedChapter: chapterNum,
		})
		return true
	case types.OrgMemberLeft, types.OrgMemberExpelled, types.OrgMemberBetrayed:
		m := find()
		if m == nil {
			return false
		}
		m.Status = map[string]string{types.OrgMemberLeft: "retired", types.OrgMemberExpelled: "expelled", types.OrgMemberBetrayed: "expelled"}[mc.ChangeType]
		m.LeftAt = at
		m.UpdatedChapter = chapterNum
		if mc.Reason != "" {
			m.Notes = strings.TrimSpace(m.Notes + fmt.Sprintf("\n[第%d章] %s", chapterNum, mc.Reason))
		}
		return true
	case types.OrgMemberPromoted, types.OrgMemberStatus:
		m := find()
		if m == nil {
			return false
		}
		if mc.Position != "" {
			m.Position = mc.Position
		}
		if mc.Status != "" {
			m.Status = mc.Status
		}
		if mc.LoyaltyHint != nil {
			m.Loyalty = clamp100(*mc.LoyaltyHint)
		}
		m.UpdatedChapter = chapterNum
		return true
	}
	return false
}

func clamp100(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// findBidirectionalRel 无向语义：A→B 与 B→A 视为同一条关系（MuMu :377-390）。
func findBidirectionalRel(cf *types.CharacterFile, nameA, nameB string) int {
	// 以名字索引 ID 的映射在调用方已保证角色存在；这里按 ID 找。
	idOf := map[string]string{}
	for i := range cf.Characters {
		idOf[cf.Characters[i].Name] = cf.Characters[i].ID
	}
	a, b := idOf[nameA], idOf[nameB]
	for i := range cf.Relationships {
		rel := cf.Relationships[i]
		if (rel.FromID == a && rel.ToID == b) || (rel.FromID == b && rel.ToID == a) {
			return i
		}
	}
	return -1
}

// appendRelNote 关系描述追加变更时间线注释（MuMu :402-405）。
func appendRelNote(desc string, chapterNum int, note string) string {
	line := fmt.Sprintf("[第%d章] %s", chapterNum, note)
	if strings.TrimSpace(desc) == "" {
		return line
	}
	return desc + "\n" + line
}

// truncateRunes rune 截断（本包局部，避免与 analysis 包重名冲突）。
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
