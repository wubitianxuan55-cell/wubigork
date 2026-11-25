package db

// schema_specialties.go — 清单与定额共用的规划专业（2026-11-25 用户令「清单和定额的
// 专业你重新进行规划」，随后逐项调整定稿 10 个）。清单与定额共用一套专业名——套
// 定额时「同专业优先」靠的就是同名，两套名字对不上就没法比。
//
// 这里只是一张数据表：专业名、自有码里的专业段代号（两位字母）、字典里的排序、
// 一句说明。改专业改这张表；SchemaV40 的迁移 SQL 由它生成。地区、工料机分类的
// 字典仍然不预设（V38 的口径不变，只有专业这一项被用户翻了案）。
//
// 分组：
//   - 环境修复：土壤 / 地下水按介质拆开（官方的「污染修复工程」是一类）；水污染
//     防治、固废处置对应环境工程设计专项资质里的两个专项。
//   - 建设工程：现行国家清单体系（GB/T 50500-2024 及配套工程量计算标准，
//     2025-09-01 施行）里公司用得上的三个专业。
//   - 项目配套：临建 / 措施 / 检测是各项目公用的，单列成专业，定额有处可放。

import (
	"fmt"
	"strings"
)

// PlannedSpecialty 规划专业表的一行。
type PlannedSpecialty struct {
	Name string
	Code string // 自有码的专业段（两位字母数字）
	Sort int
	Note string
}

// PlannedSpecialties 清单与定额共用的规划专业。
var PlannedSpecialties = []PlannedSpecialty{
	// 环境修复
	{Name: "土壤修复", Code: "TR", Sort: 10, Note: "规划专业：污染土壤的清挖、筛分、处置、修复、回填与场地恢复"},
	{Name: "地下水修复", Code: "DX", Sort: 20, Note: "规划专业：地下水污染的阻隔、抽提处理、防控与监测"},
	{Name: "水污染防治", Code: "SW", Sort: 30, Note: "规划专业：对应环境工程设计专项「水污染防治工程」"},
	{Name: "固废处置", Code: "GF", Sort: 40, Note: "规划专业：对应环境工程设计专项「固体废物处理处置工程」"},
	// 建设工程（现行国家清单体系里公司用得上的）
	{Name: "房屋建筑与装饰工程", Code: "FJ", Sort: 50, Note: "国家清单专业：工程量计算标准 GB/T 50854-2024（2025-09-01 施行）"},
	{Name: "通用安装工程", Code: "AZ", Sort: 60, Note: "国家清单专业：工程量计算标准 GB/T 50856-2024（2025-09-01 施行）"},
	{Name: "市政工程", Code: "SZ", Sort: 70, Note: "国家清单专业：工程量计算标准 GB/T 50857-2024（2025-09-01 施行）"},
	// 项目配套（各项目公用）
	{Name: "临建", Code: "LJ", Sort: 80, Note: "规划专业（各项目公用）：施工便道、围挡、临水临电、办公与生活区等临时设施"},
	{Name: "措施", Code: "CS", Sort: 90, Note: "规划专业（各项目公用）：安全文明施工、支护降水、降尘、二次污染防治等措施项目"},
	{Name: "检测", Code: "JC", Sort: 100, Note: "规划专业（各项目公用）：环境监测、检测与验收"},
}

// specialtySeedNow 迁移里写时间戳用的 SQLite 表达式（与 workcost.nowText 同形：
// RFC3339 的 UTC）。
const specialtySeedNow = "strftime('%Y-%m-%dT%H:%M:%SZ','now')"

// SchemaV40 规划专业进字典（幂等）：
//   - 库里已有同名专业（如 V38 从数据里收进来的「土壤修复」）且还没代号 → 只补
//     代号、排序和说明；用户自己填过代号的不动。
//   - 库里没有的 → 补进来；已存在（INSERT OR IGNORE）不重复。
//
// V38 定的是「字典不预设任何清单」；2026-11-25 用户令专业重新规划，专业这一项
// 改为预设上面这张表，地区与工料机分类仍不预设。
var SchemaV40 = specialtySeedSQL(PlannedSpecialties)

// specialtySeedSQL 把规划专业表生成幂等的落库 SQL。
func specialtySeedSQL(items []PlannedSpecialty) string {
	var b strings.Builder
	for _, it := range items {
		// 名与说明都是本文件里的字面量，不含单引号；这里断言一下，免得日后改表时
		// 把 SQL 拼坏。
		if strings.ContainsAny(it.Name+it.Note, "'") {
			panic("规划专业的名称与说明不能含单引号: " + it.Name)
		}
		fmt.Fprintf(&b,
			`UPDATE gf_dicts SET code='%s', sort=%d, note=CASE WHEN note LIKE '初始字典%%' THEN '%s' ELSE note END, updated_at=%s WHERE kind='specialty' AND name='%s' AND code='';`,
			it.Code, it.Sort, it.Note, specialtySeedNow, it.Name)
		fmt.Fprintf(&b,
			`INSERT OR IGNORE INTO gf_dicts(kind, name, code, parent, level, sort, note, created_at, updated_at) VALUES('specialty','%s','%s','',1,%d,'%s',%s,%s);`,
			it.Name, it.Code, it.Sort, it.Note, specialtySeedNow, specialtySeedNow)
		b.WriteString("\n")
	}
	return b.String()
}
