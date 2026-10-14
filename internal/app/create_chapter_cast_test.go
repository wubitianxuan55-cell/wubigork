package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newCastTestProject 建一个带三名角色的测试项目（主角/配角/反派）。
func newCastTestProject(t *testing.T) *project.Manager {
	t.Helper()
	pm := newStatesTestProject(t)
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive",
				Personality: "清冷", Background: "没落剑修", Motivation: "查灭门真相"},
			{ID: "c2", Name: "沈青", RoleType: "supporting", Status: "Alive", Personality: "圆滑"},
			{ID: "c3", Name: "鬼手", RoleType: "antagonist", Status: "Alive", Personality: "阴鸷"},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	return pm
}

// TestBuildChapterCastSection_FocusTiers 计划有角色焦点时装配两级：焦点角色
// 全量档案（登场），其余降名册（备查不登场）。
func TestBuildChapterCastSection_FocusTiers(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	plan := &types.ChapterPlan{CharacterFocus: []string{"林晚"}}

	got := a.buildChapterCastSection(pm, plan, "", 0)
	if !strings.Contains(got, "【本章出场角色（按本章计划与文本命中装配，剧情围绕他们展开）】") {
		t.Errorf("缺焦点区段头:\n%s", got)
	}
	if !strings.Contains(got, "- 林晚：主角") || !strings.Contains(got, "查灭门真相") {
		t.Errorf("焦点角色应给全量档案:\n%s", got)
	}
	if !strings.Contains(got, "【其余名册（备查") {
		t.Errorf("缺名册区段头:\n%s", got)
	}
	if !strings.Contains(got, "- 沈青：配角") || !strings.Contains(got, "- 鬼手：反派") {
		t.Errorf("其余角色应降名册:\n%s", got)
	}
	// 名册是紧凑行：不携带焦点级档案字段（动机）
	if strings.Contains(got, "名册") && strings.Count(got, "目标：") > 1 {
		t.Errorf("名册行不应携带完整档案:\n%s", got)
	}
}

// TestBuildChapterCastSection_FocusTolerantMatch 焦点项带缀饰也能匹配（作者手写
// 「林晚（主角）」这类形态）。
func TestBuildChapterCastSection_FocusTolerantMatch(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	plan := &types.ChapterPlan{CharacterFocus: []string{"林晚（主角）"}}

	got := a.buildChapterCastSection(pm, plan, "", 0)
	if !strings.Contains(got, "查灭门真相") {
		t.Errorf("缀饰焦点应命中角色档案:\n%s", got)
	}
}

// TestBuildBranchCastSection 选角会议区段：名册命中者全量档案+主角关系缀行；
// 未命中（库挑入/新建）按负载自带信息成行；未圈定角色不渲染；空负载退回全量；
// 纯字符串数组兼容；坏 JSON 不报错。
func TestBuildBranchCastSection(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}

	got := a.buildBranchCastSection(pm, parseBranchCast(`[{"name":"林晚","relation":"师妹"},{"name":"玄铁","relation":"死敌","note":"库·冷面刀客"}]`), 0)
	if !strings.Contains(got, "【本轮分支登场角色（作者圈定；关系为主角视角）】") {
		t.Errorf("缺选角区段头:\n%s", got)
	}
	if !strings.Contains(got, "- 林晚：主角") || !strings.Contains(got, "与主角关系：师妹") {
		t.Errorf("名册命中应全量档案+关系缀行:\n%s", got)
	}
	if !strings.Contains(got, "- 玄铁：新增角色·库·冷面刀客·与主角关系：死敌") {
		t.Errorf("未命中应按负载成行:\n%s", got)
	}
	if strings.Contains(got, "沈青") {
		t.Errorf("未圈定角色不应渲染:\n%s", got)
	}

	// 纯字符串数组兼容
	got = a.buildBranchCastSection(pm, parseBranchCast(`["林晚"]`), 0)
	if !strings.Contains(got, "- 林晚：主角") || strings.Contains(got, "与主角关系：") {
		t.Errorf("字符串负载=只圈名字不带关系:\n%s", got)
	}
	// 空负载/坏 JSON → 全量名册
	if full := a.buildCharacterSummary(pm, 0); a.buildBranchCastSection(pm, parseBranchCast(""), 0) != full {
		t.Errorf("空负载应退回全量名册")
	}
	if full := a.buildCharacterSummary(pm, 0); a.buildBranchCastSection(pm, parseBranchCast("not-json"), 0) != full {
		t.Errorf("坏 JSON 应退回全量名册")
	}
}

// TestBuildChapterCastSection_NoPlanOrNoMatch 无计划/焦点为空/焦点没命中任何
// 角色时退回全量名册（旧口径），绝不空窗。
func TestBuildChapterCastSection_NoPlanOrNoMatch(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}

	full := a.buildCharacterSummary(pm, 0)
	if got := a.buildChapterCastSection(pm, nil, "", 0); got != full {
		t.Errorf("无计划应等于旧全量口径")
	}
	if got := a.buildChapterCastSection(pm, &types.ChapterPlan{}, "", 0); got != full {
		t.Errorf("空焦点应等于旧全量口径")
	}
	noHit := a.buildChapterCastSection(pm, &types.ChapterPlan{CharacterFocus: []string{"路人甲"}}, "", 0)
	if !strings.Contains(noHit, "- 林晚") || !strings.Contains(noHit, "- 沈青") || !strings.Contains(noHit, "- 鬼手") {
		t.Errorf("焦点未命中应退回全量名册:\n%s", noHit)
	}
}

// TestBuildChapterCastSection_AliasDetect 实体卡触发注入（8.1）：扫描文本命中
// 别名（外号/敬称）即升全量档案——名字不在计划焦点也注入。
func TestBuildChapterCastSection_AliasDetect(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	// 给沈青配别名「青儿」、鬼手配敬称「鬼手大人」
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive", Personality: "清冷"},
			{ID: "c2", Name: "沈青", RoleType: "supporting", Status: "Alive", Personality: "圆滑",
				Aliases: []string{"青儿", "沈师叔"}},
			{ID: "c3", Name: "鬼手", RoleType: "antagonist", Status: "Alive", Personality: "阴鸷",
				Aliases: []string{"鬼手大人"}},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	plan := &types.ChapterPlan{CharacterFocus: []string{"林晚"}}

	// 别名命中：计划只点了林晚，但剧情文本出现「青儿」→ 沈青升全量档案
	got := a.buildChapterCastSection(pm, plan, "林晚在渡口遇到了青儿，两人寒暄片刻", 0)
	if !strings.Contains(got, "- 林晚：主角") || !strings.Contains(got, "- 沈青：配角") {
		t.Errorf("别名命中应升全量档案:\n%s", got)
	}
	if !strings.Contains(got, "圆滑") {
		t.Errorf("命中的沈青应携带档案字段:\n%s", got)
	}
	if strings.Contains(got, "阴鸷") {
		t.Errorf("未命中的鬼手只应降名册（紧凑行不带档案）:\n%s", got)
	}
	// 敬称同样命中
	got = a.buildChapterCastSection(pm, plan, "鬼手大人缓步走来", 0)
	if !strings.Contains(got, "- 鬼手：反派") {
		t.Errorf("敬称别名应命中:\n%s", got)
	}
	// 确定性：同输入两次装配输出一致
	g1 := a.buildChapterCastSection(pm, plan, "林晚在渡口遇到了青儿", 0)
	g2 := a.buildChapterCastSection(pm, plan, "林晚在渡口遇到了青儿", 0)
	if g1 != g2 {
		t.Errorf("同输入装配应确定一致:\n%s\n---\n%s", g1, g2)
	}
}

// TestBuildChapterCastSection_ExcludeHits 排除词抑制：命中串只是排除词子串时
// 不触发（排除「鬼手大人」后该词不再让「鬼手」命中）。
func TestBuildChapterCastSection_ExcludeHits(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive", Personality: "清冷"},
			{ID: "c3", Name: "鬼手", RoleType: "antagonist", Status: "Alive", Personality: "阴鸷",
				ExcludeHits: []string{"鬼手大人"}},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	plan := &types.ChapterPlan{CharacterFocus: []string{"林晚"}}

	got := a.buildChapterCastSection(pm, plan, "传闻鬼手大人在北境现身", 0)
	if strings.Contains(got, "阴鸷") {
		t.Errorf("命中串是排除词子串时不应升档:\n%s", got)
	}
	if !strings.Contains(got, "- 鬼手：反派") {
		t.Errorf("未升档角色应保留名册备查行:\n%s", got)
	}
	got = a.buildChapterCastSection(pm, plan, "鬼手亲自现身了", 0)
	if !strings.Contains(got, "阴鸷") {
		t.Errorf("非排除场景应正常命中并升档:\n%s", got)
	}
}

// TestBuildChapterCastSection_InjectModes 四态语义：always 恒注入；muted 命中
// 也不升档（仅名册）；never 全链不可见。
func TestBuildChapterCastSection_InjectModes(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c1", Name: "林晚", RoleType: "protagonist", Status: "Alive", Personality: "清冷", InjectMode: "always"},
			{ID: "c2", Name: "沈青", RoleType: "supporting", Status: "Alive", Personality: "圆滑", InjectMode: "muted"},
			{ID: "c3", Name: "鬼手", RoleType: "antagonist", Status: "Alive", Personality: "阴鸷", InjectMode: "never"},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	plan := &types.ChapterPlan{CharacterFocus: []string{"沈青"}} // muted：焦点点名也不升档

	got := a.buildChapterCastSection(pm, plan, "沈青与鬼手都出现了", 0)
	if !strings.Contains(got, "- 林晚：主角") || !strings.Contains(got, "清冷") {
		t.Errorf("always 应无条件升全量档案:\n%s", got)
	}
	if strings.Contains(got, "圆滑") {
		t.Errorf("muted 命中不应升档:\n%s", got)
	}
	if !strings.Contains(got, "- 沈青：配角") {
		t.Errorf("muted 应保留名册备查行:\n%s", got)
	}
	if strings.Contains(got, "鬼手") {
		t.Errorf("never 应全链不可见:\n%s", got)
	}

	// never 在全量名册回退路径同样不可见
	full := a.buildCharacterSummary(pm, 0)
	if strings.Contains(full, "鬼手") {
		t.Errorf("never 不应进全量名册:\n%s", full)
	}
	// 未知模式值按默认检测注入处理
	if m := normalizeInjectMode("bogus"); m != "" {
		t.Errorf("未知注入模式应回退默认，得到 %q", m)
	}
}

// TestBuildChapterCastSection_BudgetCap 全档区预算封顶：命中角色档案合计超过
// charSummaryBudget 时截断（8.1 出口判据：预算封顶可证）。
func TestBuildChapterCastSection_BudgetCap(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	filler := strings.Repeat("漫长的背景设定一笔", 10) // 90 rune；每行档案字段各截 60 后单行仍 ~200 rune
	chars := []types.Character{
		{ID: "hero", Name: "林晚", RoleType: "protagonist", Status: "Alive",
			Personality: filler, Background: filler, Motivation: filler},
	}
	for i := 0; i < 24; i++ {
		chars = append(chars, types.Character{
			ID: fmt.Sprintf("c%d", i), Name: fmt.Sprintf("群演%d", i),
			RoleType: "minor", Status: "Alive", Personality: filler, Background: filler,
		})
	}
	if err := pm.WriteCharacters(&types.CharacterFile{Characters: chars}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	focus := make([]string, 0, 25)
	for _, c := range chars {
		focus = append(focus, c.Name)
	}
	plan := &types.ChapterPlan{CharacterFocus: focus}

	got := a.buildChapterCastSection(pm, plan, "", 0)
	if !strings.Contains(got, "...") {
		t.Errorf("超预算全档区应截断（util.Truncate ASCII 省略号）:\n%s", got)
	}
	if n := runeLen(got); n > charSummaryBudget+charRosterBudget+120 {
		t.Errorf("装配总长应受双预算封顶（得到 %d rune）", n)
	}
}

// TestCharacterProgressions_TwoChapterProjection 阶段八 8.2 出口判据：状态卡
// 在第 N 章前后生成两章，注入内容不同（替换/增补/未来章不可见/取最新四象限）。
func TestCharacterProgressions_TwoChapterProjection(t *testing.T) {
	pm := newCastTestProject(t)
	a := &writingState{core: &core{}}
	if err := pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{
			{ID: "c2", Name: "沈青", RoleType: "supporting", Status: "Alive", Personality: "圆滑",
				Progressions: []types.CharacterProgression{
					{Chapter: 5, Field: "personality", Text: "黑化阴沉"},
					{Chapter: 5, Field: "status", Text: "Missing"},
					{Chapter: 8, Text: "左臂被斩断"},
				}},
		},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}

	// 第 4 章：锚点未到，基础状态原样
	before := a.buildCharacterSummary(pm, 4)
	if !strings.Contains(before, "圆滑") || strings.Contains(before, "黑化阴沉") || strings.Contains(before, "Missing") {
		t.Errorf("第 4 章不应看到第 5 章起的演进:\n%s", before)
	}
	// 第 5 章：替换生效（性格换、状态换），旧值不再出现
	at5 := a.buildCharacterSummary(pm, 5)
	if !strings.Contains(at5, "黑化阴沉") || strings.Contains(at5, "圆滑") {
		t.Errorf("第 5 章应取替换后性格:\n%s", at5)
	}
	if !strings.Contains(at5, "·Missing") {
		t.Errorf("第 5 章状态应替换为 Missing:\n%s", at5)
	}
	// 未来章不可见：第 7 章看不到第 8 章增补
	at7 := a.buildCharacterSummary(pm, 7)
	if strings.Contains(at7, "左臂被斩断") {
		t.Errorf("第 7 章不应看到第 8 章增补:\n%s", at7)
	}
	// 第 8 章：增补以「第N章起」缀行尾
	at8 := a.buildCharacterSummary(pm, 8)
	if !strings.Contains(at8, "第8章起：左臂被斩断") {
		t.Errorf("第 8 章应见增补演进:\n%s", at8)
	}
	// 章 0 = 取最新（面板/分支等无章号语境）
	latest := a.buildCharacterSummary(pm, 0)
	if !strings.Contains(latest, "第8章起：左臂被斩断") || !strings.Contains(latest, "黑化阴沉") {
		t.Errorf("章 0 应取全部演进:\n%s", latest)
	}
}

// TestResolveProgressions_ReplacementChain 同字段多版本：取章号最大者，
// 与列表顺序无关（作者乱序补录以章号为准）；同章后者覆盖前者。
func TestResolveProgressions_ReplacementChain(t *testing.T) {
	ch := types.Character{Progressions: []types.CharacterProgression{
		{Chapter: 10, Field: "personality", Text: "疯癫"},
		{Chapter: 5, Field: "personality", Text: "黑化阴沉"},
	}}
	repl, _ := resolveProgressions(ch, 9)
	if repl["personality"] != "黑化阴沉" {
		t.Errorf("第 9 章应取第 5 章版本: %v", repl)
	}
	repl, _ = resolveProgressions(ch, 10)
	if repl["personality"] != "疯癫" {
		t.Errorf("第 10 章应取章号最大版本: %v", repl)
	}
	tie := types.Character{Progressions: []types.CharacterProgression{
		{Chapter: 5, Field: "status", Text: "Missing"},
		{Chapter: 5, Field: "status", Text: "Transformed"},
	}}
	repl, _ = resolveProgressions(tie, 5)
	if repl["status"] != "Transformed" {
		t.Errorf("同章并列取列表靠后者: %v", repl)
	}
}

// TestResolveProgressions_UnknownFieldFallsBackToAddition 白名单外/空字段
// 按增补降级（诚实降级，不静默丢弃）；空文本与非法章号跳过。
func TestResolveProgressions_UnknownFieldFallsBackToAddition(t *testing.T) {
	ch := types.Character{Progressions: []types.CharacterProgression{
		{Chapter: 3, Field: "wardrobe", Text: "改穿玄色劲装"},
		{Chapter: 4, Text: "佩剑换了"},
		{Chapter: 5, Text: ""},    // 空文本跳过
		{Chapter: 0, Text: "无章号"}, // 非法章号跳过
	}}
	repl, adds := resolveProgressions(ch, 9)
	if len(repl) != 0 {
		t.Errorf("白名单外字段不应产生替换: %v", repl)
	}
	if len(adds) != 2 || adds[0].Chapter != 3 || adds[1].Chapter != 4 {
		t.Errorf("增补应按章号升序且跳过空文本/非法章号: %+v", adds)
	}
}

// TestCharacterSetProgressions 绑定面见 characterlib_handler_test.go（需要
// newCharacterLibTestApp 的项目上下文，不在本文件装配域内）。
