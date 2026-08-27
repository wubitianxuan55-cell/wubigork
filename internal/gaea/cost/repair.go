// 成本库数据梳理（自愈）：修复历史遗留的平铺分类路径 + 回填地区/期数。
//
// 背景（2026-08-19 数据库梳理）：SchemaV7 把旧平铺分类（category）原样复制为
// category_path，未迁移进新分类树；此后分类树又演进（旧版「人工/材料/机械/
// 运输/检测」+ 新版「综合单价→专业→分部」叠加），导致 1420 条里 201 条
// category_path 在树上无法解析——树上看不到、统计对不上。本文件提供幂等的
// RepairCategoryPaths（把非法路径映射回树的合法路径）与 BackfillPriceMeta
// （从来源字符串保守提取地区/期数，不臆造），并在 Store.Open 时自动执行，
// 防止再次漂移。
package cost

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// ── 旧平铺分类 → 新树合法路径 的规则引擎 ──────────────────────────────

// legacyCategoryTarget 把一条旧平铺 category_path 映射为分类树的合法路径。
// source/title 提供上下文（手册章节、人工费标记、地区信息）；无法确定时返回 ""。
//
// 规则优先级（先判断的优先）：
//  1. 房建成本测算手册条目 → 综合单价/房建工程/{手册章节}
//  2. 市政成本测算手册条目 → 综合单价/{专业}/{分部}（按手册章节 + 标题关键词）
//  3. 普工/技工/特殊工种 平铺 → 人工/{同名}（人工类别，明确无歧义）
//  4. 人工费数据库的纯人工条目 → 人工/技工
//  5. 玻璃及玻璃制品 → 材料/土建材料/玻璃及玻璃制品（新增节点）
//  6. 木竹材料 → 材料/土建材料/木材及竹木制品（既有节点语义等价）
//  7. 土方 兜底 → 综合单价/土方（既有综合单价子目）
//  8. 其余叶名唯一匹配表
func legacyCategoryTarget(source, title, path string) string {
	src := strings.TrimSpace(source)
	t := strings.TrimSpace(title)
	p := strings.TrimSpace(path)

	// 1. 房建成本测算手册：手册自带专业章节，直接归入 房建工程 专业组。
	if strings.Contains(src, "房建成本测算手册") {
		switch {
		case strings.Contains(src, "给排水"):
			return "综合单价/房建工程/给排水工程"
		case strings.Contains(src, "电气"):
			return "综合单价/房建工程/电气工程"
		case strings.Contains(src, "通风"):
			return "综合单价/房建工程/通风空调工程"
		case strings.Contains(src, "采暖"):
			return "综合单价/房建工程/采暖工程"
		case strings.Contains(src, "弱电"):
			return "综合单价/房建工程/弱电工程"
		case strings.Contains(src, "土建"):
			return "综合单价/房建工程/土建工程"
		case strings.Contains(src, "单方指标"):
			return "综合单价/房建工程/单方指标"
		}
	}

	// 2. 市政成本测算手册：按章节（专业）+ 标题关键词（分部）。
	if strings.Contains(src, "市政成本测算手册") {
		section := ""
		if i := strings.LastIndex(src, "/"); i >= 0 {
			section = strings.TrimSpace(src[i+1:])
		}
		switch section {
		case "道路":
			switch {
			case strings.Contains(t, "土方") || strings.Contains(t, "回填") ||
				strings.Contains(t, "弃置") || strings.Contains(t, "挖"):
				return "综合单价/道路工程/土方工程"
			case strings.Contains(t, "路面") || strings.Contains(t, "碎石") ||
				strings.Contains(t, "沥青"):
				return "综合单价/道路工程/机动车道"
			}
			return "综合单价/道路工程"
		case "交通":
			switch {
			case strings.Contains(t, "标线"):
				return "综合单价/交通工程/标线"
			case strings.Contains(t, "标志") || strings.Contains(t, "护栏"):
				return "综合单价/交通工程/标识标牌"
			case strings.Contains(t, "信号灯"):
				return "综合单价/交通工程/信号灯"
			}
			return "综合单价/交通工程"
		case "绿化":
			switch {
			case strings.Contains(t, "乔木"):
				return "综合单价/绿化工程/乔木"
			case strings.Contains(t, "灌木"):
				return "综合单价/绿化工程/灌木"
			case strings.Contains(t, "草皮") || strings.Contains(t, "色带") ||
				strings.Contains(t, "地被"):
				return "综合单价/绿化工程/地被"
			case strings.Contains(t, "整理") || strings.Contains(t, "土方"):
				return "综合单价/绿化工程/土方工程"
			}
			return "综合单价/绿化工程"
		case "电力":
			switch {
			case strings.Contains(t, "管沟") || strings.Contains(t, "排管"):
				return "综合单价/电力工程/管沟与井室"
			case strings.Contains(t, "电缆"):
				return "综合单价/电力工程/电缆敷设"
			}
			return "综合单价/电力工程"
		case "给水":
			switch {
			case strings.Contains(t, "井"):
				return "综合单价/给水工程/井室及附件"
			case strings.Contains(t, "管"):
				return "综合单价/给水工程/管道铺设"
			}
			return "综合单价/给水工程"
		case "暖气":
			switch {
			case strings.Contains(t, "井"):
				return "综合单价/暖气工程/井室及附件"
			case strings.Contains(t, "管"):
				return "综合单价/暖气工程/管道铺设"
			}
			return "综合单价/暖气工程"
		case "雨污":
			switch {
			case strings.Contains(t, "检查井") || strings.Contains(t, "雨水口"):
				return "综合单价/雨污工程/检查井及雨水口"
			case strings.Contains(t, "管"):
				return "综合单价/雨污工程/管道铺设"
			}
			return "综合单价/雨污工程"
		case "照明":
			switch {
			case strings.Contains(t, "灯"):
				return "综合单价/照明工程/灯杆灯具安装"
			case strings.Contains(t, "电缆"):
				return "综合单价/照明工程/电缆敷设"
			}
			return "综合单价/照明工程"
		}
	}

	// 3. 人工类别平铺名：明确无歧义。
	if target, ok := map[string]string{
		"普工": "人工/普工", "技工": "人工/技工", "特殊工种": "人工/特殊工种",
	}[p]; ok {
		return target
	}

	// 4. 人工费数据库的纯人工条目（标题带「人工费」）→ 技工。
	if strings.Contains(src, "人工费数据库") && strings.Contains(t, "人工费") {
		return "人工/技工"
	}

	// 5/6/7. 特殊旧分类。
	switch p {
	case "玻璃及玻璃制品":
		return "材料/土建材料/玻璃及玻璃制品"
	case "木竹材料":
		return "材料/土建材料/木材及竹木制品"
	case "土方":
		return "综合单价/土方"
	}

	// 8. 叶名唯一映射表（名称在树中全局唯一，直接定位）。
	if target, ok := uniqueLeafTargets[p]; ok {
		return target
	}
	return ""
}

// uniqueLeafTargets 旧平铺分类名 → 树内唯一同名节点的完整路径。
var uniqueLeafTargets = map[string]string{
	"水泥及水泥制品": "材料/土建材料/水泥及水泥制品",
	"钢材":      "材料/土建材料/钢材",
	"砖瓦灰砂石":   "材料/土建材料/砖瓦灰砂石",
	"土方机械":    "机械/土方机械",
	"辅助材料":    "材料/辅助材料",
	"燃料火工":    "材料/辅助材料/燃料火工",
	"土工合成材料":  "材料/辅助材料/土工合成材料",
	"桩基机械":    "机械/桩基机械",
	"修复处置":    "综合单价/修复处置",
	"临建设施":    "材料/辅助材料/临建设施",
	"场外运输":    "运输/场外运输",
	"脚手架":     "材料/周转材料/脚手架",
	"处置":      "其他/处置",
	"运输机械":    "机械/运输机械",
	"服务":      "其他/服务",
	"起重机械":    "机械/起重机械",
	"混凝土机械":   "机械/混凝土机械",
}

// ── 地区/期数从来源字符串提取 ───────────────────────────────────────

var (
	reYearMonth = regexp.MustCompile(`(\d{4})年(\d{1,2})月`)
	reYearDashM = regexp.MustCompile(`(\d{4})-(\d{1,2})`)
)

// priceMetaFromSource 从来源字符串保守提取 地区/期数（写回 region/price_date）。
// 只提取来源里明确写出的信息（如「重庆工程造价2026年第7期（2026年6月中心城区）」），
// 无法确定时返回空串，不臆造。
func priceMetaFromSource(source string) (region, priceDate string) {
	s := strings.TrimSpace(source)
	// 期数：第一个 YYYY年M月 或 YYYY-MM（统一归一为 YYYY年M月）。
	if m := reYearMonth.FindStringSubmatch(s); m != nil {
		priceDate = fmt.Sprintf("%s年%d月", m[1], atoiSafe(m[2]))
	} else if m := reYearDashM.FindStringSubmatch(s); m != nil {
		priceDate = fmt.Sprintf("%s年%d月", m[1], atoiSafe(m[2]))
	}
	// 地区：只认信息价来源的显式地区。
	switch {
	case strings.Contains(s, "重庆工程造价"):
		area := parenArea(s)
		area = strings.TrimSpace(reYearMonth.ReplaceAllString(area, ""))
		area = strings.Trim(area, "，, ；;")
		region = "重庆" + area
	case strings.Contains(s, "四川省工程造价信息网"):
		region = parenArea(s)
	case strings.Contains(s, "乐山市建筑材料市场信息价"):
		area := parenArea(s)
		if area == "市中区" {
			area = "乐山市中区"
		}
		region = area
	case strings.Contains(s, "重庆信息价"):
		region = "重庆"
	case strings.Contains(s, "四川信息价"):
		region = "四川"
	}
	return region, priceDate
}

// parenArea 取中英文括号内、第一个逗号前的文本。
func parenArea(s string) string {
	open, closeC := "（", "）"
	if !strings.Contains(s, open) {
		open, closeC = "(", ")"
	}
	i := strings.Index(s, open)
	if i < 0 {
		return ""
	}
	rest := s[i+len(open):]
	end := len(rest)
	if j := strings.Index(rest, "，"); j >= 0 && j < end {
		end = j
	}
	if j := strings.Index(rest, ","); j >= 0 && j < end {
		end = j
	}
	if j := strings.Index(rest, closeC); j >= 0 && j < end {
		end = j
	}
	return strings.TrimSpace(rest[:end])
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// ── 幂等修复与回填 ─────────────────────────────────────────────────

// pathResolves 判断 path 是否为分类树中的一条完整合法路径（逐段走树）。
func (s *Store) pathResolves(path string) bool {
	if s.db == nil || strings.TrimSpace(path) == "" {
		return false
	}
	segs := strings.Split(strings.TrimSpace(path), "/")
	parent := 0
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return false
		}
		var id int
		err := s.db.QueryRow("SELECT id FROM cost_categories WHERE parent_id=? AND name=?", parent, seg).Scan(&id)
		if err != nil {
			return false
		}
		parent = id
	}
	return true
}

// RepairCategoryPaths 修复全部非法 category_path（幂等）：
// 对每条 category_path 在树上解析失败的条目，用 legacyCategoryTarget 求目标路径，
// 目标路径的节点缺失时自动创建；无法确定目标（返回空）的条目保持不动并计入 left。
// 修复条目同时更新 category（叶子名）与 updated_at。
// 注意：解析阶段在事务外完成（连接池单连接，事务内不能再取连接），
// 事务内只做建节点 + 更新。
func (s *Store) RepairCategoryPaths() (fixed, left int, err error) {
	if s.db == nil {
		return 0, 0, nil
	}
	rows, err := s.db.Query("SELECT name, title, source, category_path FROM cost_entries")
	if err != nil {
		return 0, 0, err
	}
	type entry struct {
		name, title, source, path string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if rows.Scan(&e.name, &e.title, &e.source, &e.path) == nil {
			entries = append(entries, e)
		}
	}
	rows.Close()

	// 阶段一（事务外）：解析非法路径 → 目标。
	type fix struct{ name, target string }
	var fixes []fix
	for _, e := range entries {
		if s.pathResolves(e.path) {
			continue
		}
		if target := legacyCategoryTarget(e.source, e.title, e.path); target != "" {
			fixes = append(fixes, fix{e.name, target})
		} else {
			left++
		}
	}
	if len(fixes) == 0 {
		return 0, left, nil
	}

	// 阶段二（事务内）：建节点 + 更新。
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, left, err
	}
	defer tx.Rollback()
	for _, f := range fixes {
		if err := ensurePathTx(tx, f.target, now); err != nil {
			return fixed, left, fmt.Errorf("创建分类路径 %q 失败: %w", f.target, err)
		}
		if _, err := tx.Exec(
			"UPDATE cost_entries SET category_path=?, category=?, updated_at=? WHERE name=?",
			f.target, leafOfPath(f.target), now, f.name); err != nil {
			return fixed, left, err
		}
		fixed++
	}
	if err := tx.Commit(); err != nil {
		return fixed, left, err
	}
	return fixed, left, nil
}

// ensurePathTx 在事务内确保路径上所有节点存在（幂等，父先子后）。
func ensurePathTx(tx *sql.Tx, path, now string) error {
	segs := strings.Split(path, "/")
	parent := 0
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		var id int
		err := tx.QueryRow("SELECT id FROM cost_categories WHERE parent_id=? AND name=?", parent, seg).Scan(&id)
		if err == sql.ErrNoRows {
			res, e := tx.Exec(
				"INSERT INTO cost_categories(parent_id, name, sort, created_at, updated_at) VALUES(?,?,?,?,?)",
				parent, seg, 0, now, now)
			if e != nil {
				return e
			}
			id64, _ := res.LastInsertId()
			id = int(id64)
		} else if err != nil {
			return err
		}
		parent = id
	}
	return nil
}

// BackfillPriceMeta 对 region/price_date 为空、且来源字符串可提取的条目回填
// （幂等）。不回填 updated_at（元数据富化，不视为内容变更）。
func (s *Store) BackfillPriceMeta() (regionCount, dateCount int, err error) {
	if s.db == nil {
		return 0, 0, nil
	}
	rows, err := s.db.Query("SELECT name, source, region, price_date FROM cost_entries WHERE region='' OR price_date=''")
	if err != nil {
		return 0, 0, err
	}
	type entry struct{ name, source, region, priceDate string }
	var entries []entry
	for rows.Next() {
		var e entry
		if rows.Scan(&e.name, &e.source, &e.region, &e.priceDate) == nil {
			entries = append(entries, e)
		}
	}
	rows.Close()

	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	for _, e := range entries {
		r, d := priceMetaFromSource(e.source)
		if (r == "" || e.region != "") && (d == "" || e.priceDate != "") {
			continue
		}
		newRegion, newDate := e.region, e.priceDate
		if e.region == "" && r != "" {
			newRegion = r
			regionCount++
		}
		if e.priceDate == "" && d != "" {
			newDate = d
			dateCount++
		}
		if newRegion == e.region && newDate == e.priceDate {
			continue
		}
		if _, err := tx.Exec("UPDATE cost_entries SET region=?, price_date=? WHERE name=?", newRegion, newDate, e.name); err != nil {
			return regionCount, dateCount, err
		}
	}
	if err := tx.Commit(); err != nil {
		return regionCount, dateCount, err
	}
	return regionCount, dateCount, nil
}

// SelfHeal 启动自愈：修复分类路径并回填价格元数据（幂等；仅记录失败不中断）。
func (s *Store) SelfHeal() {
	fixed, left, err := s.RepairCategoryPaths()
	if err != nil {
		log.Printf("[cost] 分类路径自愈失败: %v", err)
	} else if fixed > 0 || left > 0 {
		log.Printf("[cost] 分类路径自愈：修复 %d 条，未解决 %d 条", fixed, left)
	}
	rc, dc, err := s.BackfillPriceMeta()
	if err != nil {
		log.Printf("[cost] 价格元数据回填失败: %v", err)
	} else if rc > 0 || dc > 0 {
		log.Printf("[cost] 价格元数据回填：地区 %d 条，期数 %d 条", rc, dc)
	}
}
