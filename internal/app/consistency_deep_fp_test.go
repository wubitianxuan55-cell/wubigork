package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── 误报缓解单测：文本归一化 / 时间粒度 / 别名归一 / 分级判定 ──
//
// 误报分类盘点（与 internal/app/consistency_deep_handler.go 头注释一致）：
//   ① 措辞/格式差异：全半角、空白、包裹标点、大小写、修饰语（「那柄玄铁剑」）；
//   ② 角色别名/称谓：姓名 vs 称呼 vs 外号（项目数据无别名字段，用名单+包含+剥称谓）；
//   ③ 时间表述粒度差异：「三年后」vs 具体日期、闪回/插叙；
//   ④ 数字/单位格式差异：归一化折叠后比对；
//   ⑤ 伏笔型「未回收/无中生有」：仅因无交代消失入账的物品降级为疑似。
// 缓解口径：降级只调 severity/confidence 并附 reason，绝不静默吞告警。

// TestDeepNormalizeText 归一化表驱动用例：全半角/空白/包裹标点/大小写/首尾标点
func TestDeepNormalizeText(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  林晚  ", "林晚"},                  // 首尾空白
		{"林 晚", "林晚"},                     // 名内空白
		{"玄铁剑。", "玄铁剑"},                   // 首尾标点
		{"「玄铁剑」", "玄铁剑"},                  // 包裹引号
		{"那柄玄铁剑", "那柄玄铁剑"},                // 修饰语保留（由包含匹配消化）
		{"玄铁剑（残）", "玄铁剑残"},                // 括号剥离，内容保留
		{"ＢＯＳＳ之剑", "boss之剑"},              // 全角字母折叠 + 小写
		{"第１２３章", "第123章"},                // 全角数字折叠
		{"青云宗　后山", "青云宗后山"},               // 全角空格
		{"「北境」，", "北境"},                   // 引号+首尾标点
		{"Sword of Dawn！", "swordofdawn"}, // 大小写+全角叹号+内部空格移除
		{"ＡＬＩＡＳ", "alias"},                // 全角大写字母
	}
	for _, c := range cases {
		if got := deepNormalizeText(c.in); got != c.want {
			t.Fatalf("deepNormalizeText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// 幂等：归一化结果再归一化不变
	if once := deepNormalizeText("「玄铁剑（残）」！"); deepNormalizeText(once) != once {
		t.Fatalf("归一化应幂等: %q -> %q", once, deepNormalizeText(once))
	}
}

// TestDeepNormContains 归一化包含判定：相等恒真；单字针只允许相等；修饰语/细化互含
func TestDeepNormContains(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"玄铁剑", "玄铁剑", true},     // 相等
		{"那柄玄铁剑", "玄铁剑", true},   // 修饰语包含
		{"玄铁剑（残）", "玄铁剑", true},  // 反向包含（括号剥离后仍含本名）
		{"玄铁剑", "玄铁剑（残）", false}, // 针比堆长不算
		{"青云宗后山", "青云宗", true},   // 位置细化
		{"北境荒原", "北境", true},
		{"玄铁长剑", "玄铁剑", false}, // 字序不同不算
		{"剑", "剑", true},       // 单字相等
		{"铁剑", "剑", false},     // 单字针不允许包含
		{"南疆", "", false},      // 空针
		{"", "南疆", false},      // 空堆
	}
	for _, c := range cases {
		if got := deepNormContains(c.a, c.b); got != c.want {
			t.Fatalf("deepNormContains(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// TestDeepCoarseTimeMark 时间粒度判定：空/间隔表达/闪回类 → 粗粒度；具体日期 → 精确
func TestDeepCoarseTimeMark(t *testing.T) {
	coarse := []string{"", "  ", "三年后", "数月之后", "半月前", "几日", "翌日清晨", "次日", "当年", "昔日", "回忆里", "闪回", "十年之后", "三载"}
	for _, s := range coarse {
		if !deepCoarseTimeMark(s) {
			t.Fatalf("deepCoarseTimeMark(%q) 应为粗粒度", s)
		}
	}
	precise := []string{"第三日夜", "第一日", "1247年春", "腊月初八", "子时三刻", "黄昏", "深夜"}
	for _, s := range precise {
		if deepCoarseTimeMark(s) {
			t.Fatalf("deepCoarseTimeMark(%q) 应为精确", s)
		}
	}
}

// TestDeepAliasResolverResolve 别名归一表驱动：精确命中 / 唯一包含 / 剥称谓 / 歧义放弃
func TestDeepAliasResolverResolve(t *testing.T) {
	res := newDeepAliasResolver([]string{"林晚", "陈九斤", "青云宗主", "小蝶"})
	cases := []struct {
		in   string
		want string
	}{
		{"林晚", "林晚"},     // 精确命中
		{"  林晚 ", "林晚"},  // 空白归一后命中
		{"林晚师姐", "林晚"},   // 唯一包含（称谓后缀）
		{"小蝶姑娘", "小蝶"},   // 包含 + 剥称谓双通道
		{"晚儿", "林晚"},     // 剥「儿」后单字唯一包含
		{"阿晚", "林晚"},     // 剥「阿」前缀
		{"陈九", "陈九斤"},    // 「陈九」⊂「陈九斤」唯一包含 → 并入名单名
		{"无名路人", "无名路人"}, // 无候选保持原名（归一化形式）
		{"青云", "青云宗主"},   // 唯一包含
		{"小蝶儿", "小蝶"},    // 唯一包含（后缀变体）
	}
	for _, c := range cases {
		if got := res.Resolve(c.in); got != c.want {
			t.Fatalf("Resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// 歧义不并：两个候选包含同一名字残片 → 保持原名（宁对不齐不错并）
	amb := newDeepAliasResolver([]string{"王铁柱", "王铁锤"})
	if got := amb.Resolve("王铁"); got != "王铁" {
		t.Fatalf("歧义候选应保持原名: got %q", got)
	}
	// 空名单 / nil 接收者：返回归一化原名
	if got := newDeepAliasResolver(nil); got != nil {
		t.Fatalf("空名单应返回 nil resolver")
	}
	if got := (*deepAliasResolver)(nil).Resolve("林晚"); got != "林晚" {
		t.Fatalf("nil resolver 应返回归一化原名: %q", got)
	}
}

// TestDeepCompareStateLineGranularityTime 时间倒流缓解：粗粒度时间标记 → warning +
// granularity；精确标记仍 error
func TestDeepCompareStateLineGranularityTime(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 1, TimeMark: "第一日", TimeRelation: "unknown"},
		{Chapter: 2, TimeMark: "三年后", TimeRelation: "earlier"},
	}
	issues := deepCompareStateLine(cards, nil)
	if len(issues) != 1 {
		t.Fatalf("期望 1 条时间倒流告警: %+v", issues)
	}
	iss := issues[0]
	if iss.Severity != "warning" || iss.Reason != "granularity" || iss.Confidence != "medium" {
		t.Fatalf("粗粒度时间倒流应降级 warning/granularity/medium: %+v", iss)
	}
	if !strings.Contains(iss.Description, "粒度差异") {
		t.Fatalf("描述应说明粒度差异: %s", iss.Description)
	}

	// 双方时间标记精确 → 仍为 error（真冲突不降级）
	cards[1].TimeMark = "第一日晨"
	issues = deepCompareStateLine(cards, nil)
	if len(issues) != 1 || issues[0].Severity != "error" || issues[0].Reason != "" {
		t.Fatalf("精确时间标记的倒流应保持 error: %+v", issues)
	}

	// time_relation 改回 later → 不报
	cards[1].TimeRelation = "later"
	if issues = deepCompareStateLine(cards, nil); len(issues) != 0 {
		t.Fatalf("later 不应报时间倒流: %+v", issues)
	}
}

// TestDeepCompareStateLineAliasDeadReappear 别名归一参与的死亡再出场：仍报 error（不漏报）
// 但标 alias/medium；resolver 为 nil 时对不上号 → 不误报他人
func TestDeepCompareStateLineAliasDeadReappear(t *testing.T) {
	res := newDeepAliasResolver([]string{"林晚"})
	cards := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive"}}},
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "dead"}}},
		// 后续章 AI 以称谓提取同一角色 → 归一后命中死亡台账
		{Chapter: 3, Characters: []deepCharacterState{{Name: "林晚师姐", Status: "alive"}}},
	}
	issues := deepCompareStateLine(cards, res)
	if len(issues) != 1 {
		t.Fatalf("期望 1 条死亡再出场告警: %+v", issues)
	}
	iss := issues[0]
	if iss.Severity != "error" || iss.Reason != "alias" || iss.Confidence != "medium" {
		t.Fatalf("别名参与的死亡再出场应 error + alias/medium: %+v", iss)
	}
	if !strings.Contains(iss.Evidence, "林晚师姐") || !strings.Contains(iss.Evidence, "林晚") {
		t.Fatalf("证据应包含归一前后名字: %s", iss.Evidence)
	}

	// 无 resolver：称谓名对不上死亡台账，不误报（也不漏报出别名线索）
	issues = deepCompareStateLine(cards, nil)
	if len(issues) != 0 {
		t.Fatalf("无名单时称谓名不应与死亡台账挂钩: %+v", issues)
	}
}

// TestDeepCompareStateLineLocationWording 位置误报缓解：同地异写不报；同区域细化 →
// info + wording；跨区域无交代 → warning（别名归一参与时降置信度）
func TestDeepCompareStateLineLocationWording(t *testing.T) {
	// 同地异写（空白/标点/全半角差异）→ 不告警
	same := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: " 青云宗。"}}},
	}
	if issues := deepCompareStateLine(same, nil); len(issues) != 0 {
		t.Fatalf("同地异写不应告警: %+v", issues)
	}

	// 同区域细化（青云宗 → 青云宗后山）→ info 提示而非冲突
	refine := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗后山"}}},
	}
	issues := deepCompareStateLine(refine, nil)
	if len(issues) != 1 {
		t.Fatalf("同区域细化应产出 1 条提示: %+v", issues)
	}
	if issues[0].Severity != "info" || issues[0].Reason != "wording" || issues[0].Confidence != "low" {
		t.Fatalf("同区域细化应为 info/wording/low: %+v", issues[0])
	}
	if !strings.Contains(issues[0].Description, "非冲突") {
		t.Fatalf("提示应标明非冲突: %s", issues[0].Description)
	}

	// 真跨区域 + 无 travel_notes → warning；有交代 → 不报
	cross := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "南疆万蛊窟"}}},
	}
	issues = deepCompareStateLine(cross, nil)
	if len(issues) != 1 || issues[0].Severity != "warning" || issues[0].Reason != "" || issues[0].Confidence != "high" {
		t.Fatalf("跨区域瞬移应 warning/high 无 reason: %+v", issues)
	}
	cross[1].TravelNotes = []string{"连夜御剑南下"}
	if issues = deepCompareStateLine(cross, nil); len(issues) != 0 {
		t.Fatalf("有移动交代不应报瞬移: %+v", issues)
	}

	// 跨区域 + 角色名靠别名归一 → warning + alias/medium
	res := newDeepAliasResolver([]string{"林晚"})
	cross[0].Characters[0].Name = "林晚"
	cross[1].Characters[0].Name = "林晚前辈"
	cross[1].TravelNotes = nil
	issues = deepCompareStateLine(cross, res)
	if len(issues) != 1 || issues[0].Reason != "alias" || issues[0].Confidence != "medium" {
		t.Fatalf("别名归一参与的瞬移应 alias/medium: %+v", issues)
	}
}

// TestDeepCompareStateLineItemWording 物品名措辞/格式差异：归一化包含匹配后不再误报
// 凭空消失（缓解①④）
func TestDeepCompareStateLineItemWording(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"玄铁剑"}}}},
		// 同一物品的修饰/括号变体 → 视为仍持有
		{Chapter: 2, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"那柄玄铁剑"}}}},
	}
	if issues := deepCompareStateLine(cards, nil); len(issues) != 0 {
		t.Fatalf("物品措辞差异不应报凭空消失: %+v", issues)
	}

	// 全角/空白变体同理
	cards[1].Characters[0].Items = []string{"「玄铁剑」"}
	if issues := deepCompareStateLine(cards, nil); len(issues) != 0 {
		t.Fatalf("全角/包裹变体不应报凭空消失: %+v", issues)
	}

	// 反向：ch1 记修饰名，ch2 记本名 → 也不报
	cards[0].Characters[0].Items = []string{"玄铁剑（封印）"}
	cards[1].Characters[0].Items = []string{"玄铁剑"}
	if issues := deepCompareStateLine(cards, nil); len(issues) != 0 {
		t.Fatalf("反向措辞差异不应报凭空消失: %+v", issues)
	}
}

// TestDeepLostLookup 失去台账查找：相等优先、唯一包含、歧义不命中
func TestDeepLostLookup(t *testing.T) {
	lostAt := map[string]int{"灵玉": 2, "玉吊坠": 3}
	// 相等优先
	if at, key, ok := deepLostLookup(lostAt, "灵玉"); !ok || at != 2 || key != "灵玉" {
		t.Fatalf("相等命中失败: %d %q %v", at, key, ok)
	}
	// 唯一包含
	if at, _, ok := deepLostLookup(lostAt, "灵玉佩"); !ok || at != 2 {
		t.Fatalf("唯一包含应命中灵玉: %d %v", at, ok)
	}
	// 双候选互含 → 歧义不命中
	if _, _, ok := deepLostLookup(lostAt, "灵玉吊坠"); ok {
		t.Fatalf("歧义候选不应命中")
	}
	// 单字针只允许相等
	if _, _, ok := deepLostLookup(lostAt, "玉"); ok {
		t.Fatalf("单字针不应包含命中")
	}
}

// TestDeepCompareStateLineItemLostExplicitRestored items_lost 明确交代过的失去，
// 物品再次出现仍按 error 报无中生有（相等命中不被歧义保护吞掉）
func TestDeepCompareStateLineItemLostExplicitRestored(t *testing.T) {
	cards := []*deepStateCard{
		{Chapter: 1, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"灵玉"}}}},
		{Chapter: 2, ItemsLost: []string{"灵玉"}, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}}},
		{Chapter: 3, Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗", Items: []string{"灵玉"}}}},
	}
	issues := deepCompareStateLine(cards, nil)
	var conjured bool
	for _, iss := range issues {
		if iss.Category == "item" && iss.Severity == "error" && strings.Contains(iss.EntityName, "灵玉") && strings.Contains(iss.Description, "无中生有") {
			conjured = true
		}
	}
	if !conjured {
		t.Fatalf("明确交代过的失去应报 error 无中生有: %+v", issues)
	}
}

// TestDeepIssueToMapConfidenceDefaults deepIssueToMap 的置信度缺省：规则层/未标注 → high
func TestDeepIssueToMapConfidenceDefaults(t *testing.T) {
	m := deepIssueToMap(deepIssue{}, "rule")
	if m["confidence"] != "high" || m["reason"] != "" {
		t.Fatalf("未标注 issue 应缺省 confidence=high/reason='': %v", m)
	}
	m = deepIssueToMap(deepIssue{Confidence: "low", Reason: "wording"}, "ai")
	if m["confidence"] != "low" || m["reason"] != "wording" || m["source"] != "ai" {
		t.Fatalf("标注 issue 应透传 confidence/reason: %v", m)
	}
}

// TestDeepCanonicalPeople 项目人物名单：characters.json + lorebook 人物词条
func TestDeepCanonicalPeople(t *testing.T) {
	env := newConsistencyDeepEnv(t, nil, true)
	if err := env.pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{ID: "c1", Name: "林晚", Status: "Alive"}},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	if err := env.pm.WriteLorebook(&types.LorebookFile{
		Entries: []types.LorebookEntry{
			{Key: "玄铁剑", Content: "林晚的佩剑", Category: "item"},
			{Key: "陈九", Content: "黑市商人", Category: "character"},
		},
	}); err != nil {
		t.Fatalf("写 lorebook: %v", err)
	}
	names := deepCanonicalPeople(env.pm)
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "林晚") || !strings.Contains(joined, "陈九") {
		t.Fatalf("人物名单应含角色与 lorebook 人物词条: %v", names)
	}
	if strings.Contains(joined, "玄铁剑") {
		t.Fatalf("人物名单不应含物品词条: %v", names)
	}
}

// TestCheckConsistencyDeepFPAnnotations 端到端：AI 告警携带 confidence/reason 标注，
// 别名归一在线上生效（characters.json 提供名单）
func TestCheckConsistencyDeepFPAnnotations(t *testing.T) {
	replies := []string{
		stateCardReply(t, deepStateCard{
			Chapter: 1, TimeMark: "第三日夜", TimeRelation: "unknown",
			Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}},
		}),
		stateCardReply(t, deepStateCard{
			Chapter: 2, TimeMark: "第三日夜", TimeRelation: "earlier", // 粗粒度?「第三日夜」精确 → error 路径
			Characters: []deepCharacterState{{Name: "林晚", Status: "alive", Location: "青云宗"}},
		}),
	}
	env := newConsistencyDeepEnv(t, replies, false)
	if err := env.pm.WriteCharacters(&types.CharacterFile{
		Characters: []types.Character{{ID: "c1", Name: "林晚", Status: "Alive"}},
	}); err != nil {
		t.Fatalf("写角色: %v", err)
	}
	for n := 1; n <= 2; n++ {
		env.seedChapter(t, n, "林晚在青云宗练剑。")
	}

	res, err := env.a.CheckConsistencyDeep(20)
	if err != nil {
		t.Fatalf("CheckConsistencyDeep: %v", err)
	}
	issues, _ := res["issues"].([]map[string]interface{})
	if len(issues) == 0 {
		t.Fatalf("应有告警: %v", res)
	}
	var sawAI bool
	for _, iss := range issues {
		if c, ok := iss["confidence"].(string); !ok || c == "" {
			t.Fatalf("告警缺 confidence 标注: %v", iss)
		}
		if _, ok := iss["reason"].(string); !ok {
			t.Fatalf("告警缺 reason 字段: %v", iss)
		}
		if s, _ := iss["source"].(string); s == "ai" {
			sawAI = true
		}
	}
	if !sawAI {
		t.Fatalf("缺 AI 来源告警: %v", issues)
	}
}
