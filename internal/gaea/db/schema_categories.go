package db

// schema_categories.go — 工料机分类字典种子（2026-11-25 用户令「补充工料机分类字典，
// 现在无法分类」）。清库时把 V38「取自库内已有取值」的旧分类字典一起删了（50 项里
// 混着「措施费/设备/基坑」这类没按类别命名的顶层），而库规矩是「值必须先在字典里」
// ——空字典 = 谁都分不了类。
//
// 树形口径（顶层=四个类别，与工料机身份字段同名，杜绝旧数据「顶层不按类别命名」
// 的脏形态再出现）：
//
//	人工：普工 / 技工
//	材料：土建材料（钢材、水泥及拌合料、砂石骨料、砖砌及板块、木材及模板、
//	      防水防腐材料、其他土建材料）、药剂与化工材料（修复药剂、其他化工材料）、
//	      油燃料（成品油、润滑油）、防渗材料、电气材料、其他材料
//	机械：土方及压实机械、桩工与地基处理机械、运输机械、起重吊装机械、
//	      混凝土机械、检测与试验设备、其他机械
//	外委：检测监测外委、处置与清运外委、专业作业外委、其他外委
//
// 编号即分类段（自有码 R+类别1+两位段号+流水）：分类段**全局**编号（gcode.go
// categorySlot），这里按表序发 01..NN，两位上限 99、超了在生成器里 panic。
// 用户可改（gcode.go：字典 code 列看得见、可改，改口径不改已发的码）。
//
// 幂等防撞（三个分支都有守卫，2026-11-25 gcode 批的「字典重码」教训）：
//   - 库里已有同名分类且 code 为空（用户自己 add 的）→ 补段号/排序/说明；
//   - 同名且已填 code → 不动（用户口径优先）；
//   - 目标段号已被**别的**分类占用 → 该项跳过不插不发号（宁缺勿撞，缺的让用户
//     dict_add 时由 categorySlot 现场发号）。

import (
	"fmt"
	"strings"
)

// PlannedCategory 工料机分类字典的一行。
type PlannedCategory struct {
	Name   string // 完整路径，如 材料/土建材料/钢材
	Parent string // 上一级路径（顶级为空）
	Level  int    // 1=顶级（人工/材料/机械/外委）
	Note   string
}

// PlannedCategories 工料机分类树（表序=发号序）。
var PlannedCategories = []PlannedCategory{
	// ── 人工 ──
	{Name: "人工", Level: 1, Note: "工料机分类：人工"},
	{Name: "人工/普工", Parent: "人工", Level: 2, Note: "普通工、壮工等一般劳务"},
	{Name: "人工/技工", Parent: "人工", Level: 2, Note: "技术工、带班、持证作业人员"},
	// ── 材料 ──
	{Name: "材料", Level: 1, Note: "工料机分类：材料（外委独立成类，不并入）"},
	{Name: "材料/土建材料", Parent: "材料", Level: 2, Note: "通用建筑材料"},
	{Name: "材料/土建材料/钢材", Parent: "材料/土建材料", Level: 3, Note: "钢筋、型钢、钢板、钢管"},
	{Name: "材料/土建材料/水泥及拌合料", Parent: "材料/土建材料", Level: 3, Note: "水泥、混凝土、砂浆、稳定土拌合料"},
	{Name: "材料/土建材料/砂石骨料", Parent: "材料/土建材料", Level: 3, Note: "砂、碎石、卵石、回填料"},
	{Name: "材料/土建材料/砖砌及板块", Parent: "材料/土建材料", Level: 3, Note: "砖、砌块、预制板块"},
	{Name: "材料/土建材料/木材及模板", Parent: "材料/土建材料", Level: 3, Note: "木材、胶合板、模板及支撑料"},
	{Name: "材料/土建材料/防水防腐材料", Parent: "材料/土建材料", Level: 3, Note: "防水卷材、涂料、防腐面层材料"},
	{Name: "材料/土建材料/其他土建材料", Parent: "材料/土建材料", Level: 3, Note: "上述未覆盖的通用建材"},
	{Name: "材料/药剂与化工材料", Parent: "材料", Level: 2, Note: "修复药剂与化工品"},
	{Name: "材料/药剂与化工材料/修复药剂", Parent: "材料/药剂与化工材料", Level: 3, Note: "氧化剂、固化剂、淋洗药剂、药剂等"},
	{Name: "材料/药剂与化工材料/其他化工材料", Parent: "材料/药剂与化工材料", Level: 3, Note: "絮凝剂、酸碱、实验室试剂等"},
	{Name: "材料/油燃料", Parent: "材料", Level: 2, Note: "成品油与润滑油"},
	{Name: "材料/油燃料/成品油", Parent: "材料/油燃料", Level: 3, Note: "柴油、汽油"},
	{Name: "材料/油燃料/润滑油", Parent: "材料/油燃料", Level: 3, Note: "机油、液压油、润滑脂"},
	{Name: "材料/防渗材料", Parent: "材料", Level: 2, Note: "HDPE 膜、GCL、土工布等防渗与屏障材料"},
	{Name: "材料/电气材料", Parent: "材料", Level: 2, Note: "电缆电线、配电箱柜、照明与自控元件"},
	{Name: "材料/其他材料", Parent: "材料", Level: 2, Note: "未归入上述分类的材料"},
	// ── 机械 ──
	{Name: "机械", Level: 1, Note: "工料机分类：机械（台班或折算台时）"},
	{Name: "机械/土方及压实机械", Parent: "机械", Level: 2, Note: "挖掘机、装载机、推土机、压路机等"},
	{Name: "机械/桩工与地基处理机械", Parent: "机械", Level: 2, Note: "三轴搅拌桩机、旋喷机、打桩机等"},
	{Name: "机械/运输机械", Parent: "机械", Level: 2, Note: "自卸车、平板拖车等场内外运输"},
	{Name: "机械/起重吊装机械", Parent: "机械", Level: 2, Note: "汽车吊、履带吊、随车吊"},
	{Name: "机械/混凝土机械", Parent: "机械", Level: 2, Note: "搅拌站、罐车、泵车、振捣设备"},
	{Name: "机械/检测与试验设备", Parent: "机械", Level: 2, Note: "自有检测仪器按台班摊销的设备"},
	{Name: "机械/其他机械", Parent: "机械", Level: 2, Note: "未归入上述分类的机械"},
	// ── 外委 ──
	{Name: "外委", Level: 1, Note: "工料机分类：外委（独立行类，三费归集并入材料桶）"},
	{Name: "外委/检测监测外委", Parent: "外委", Level: 2, Note: "环境检测、监测、验收等外委服务"},
	{Name: "外委/处置与清运外委", Parent: "外委", Level: 2, Note: "水泥窑协同、填埋、外运处置等"},
	{Name: "外委/专业作业外委", Parent: "外委", Level: 2, Note: "钻井、勘察、测绘等专业分包作业"},
	{Name: "外委/其他外委", Parent: "外委", Level: 2, Note: "未归入上述分类的外委"},
}

// SchemaV41 工料机分类字典种子（幂等防撞，见文件头）。
var SchemaV41 = categorySeedSQL(PlannedCategories)

// categorySeedSQL 把分类树生成幂等的落库 SQL（编号=表序，全局两位段号）。
func categorySeedSQL(items []PlannedCategory) string {
	var b strings.Builder
	code := 0
	for _, it := range items {
		code++
		if code > 99 {
			panic("工料机分类超过 99 项——分类段（自有码第 3、4 位）只有两位，请精简分类树")
		}
		if it.Level <= 0 {
			panic("工料机分类层级未填: " + it.Name)
		}
		if it.Level == 1 {
			it.Parent = ""
		}
		// 名与说明都是本文件里的字面量，不含单引号；断言防日后改表拼坏 SQL。
		if strings.ContainsAny(it.Name+it.Parent+it.Note, "'") {
			panic("工料机分类的名称/上级/说明不能含单引号: " + it.Name)
		}
		slot := fmt.Sprintf("%02d", code)
		// 已有同名且没代号 → 补段号/排序/说明（note 仍只替换「初始字典」遗留）。
		fmt.Fprintf(&b,
			`UPDATE gf_dicts SET code='%s', sort=%d, note=CASE WHEN note LIKE '初始字典%%' THEN '%s' ELSE note END, updated_at=%s `+
				`WHERE kind='category' AND name='%s' AND code='' `+
				`AND NOT EXISTS(SELECT 1 FROM gf_dicts x WHERE x.kind='category' AND x.code='%s' AND x.name<>'%s');`,
			slot, code*10, it.Note, specialtySeedNow, it.Name, slot, it.Name)
		// 没有 → 插入；目标段号已被别的分类占用则整条跳过（宁缺勿撞）。
		fmt.Fprintf(&b,
			`INSERT INTO gf_dicts(kind, name, code, parent, level, sort, note, created_at, updated_at) `+
				`SELECT 'category','%s','%s','%s',%d,%d,'%s',%s,%s `+
				`WHERE NOT EXISTS(SELECT 1 FROM gf_dicts WHERE kind='category' AND name='%s') `+
				`AND NOT EXISTS(SELECT 1 FROM gf_dicts WHERE kind='category' AND code='%s');`,
			it.Name, slot, it.Parent, it.Level, code*10, it.Note,
			specialtySeedNow, specialtySeedNow, it.Name, slot)
		b.WriteString("\n")
	}
	return b.String()
}
