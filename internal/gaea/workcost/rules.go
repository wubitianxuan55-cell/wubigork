package workcost

// rules.go — 造价库的规矩（2026-11-25 用户拍板「数据库要定规矩，不是比着几个
// 文件去适配」）。规范值由库说了算，文件和模型向库靠：
//
//   - 专业、地区、工料机分类是字典（gf_dicts）：写进库的值必须在字典里，空值=
//     待整理；新值只能显式新增字典项（cost_library action=dict add，经用户确认）。
//     字典里不放任何预设清单——初始内容是升级时从库里已有取值机械收进来的，
//     其余由用户按需加。
//   - 期数统一存 YYYY-MM：写入前归一，认不出的写法（如信息价「2026年第7期」）
//     不猜，拒收并让模型换算成年月。
//   - 工料机类别只有 人工/材料/机械/外委 四种。
//
// 校验落在存储层（唯一写路径）：工具、导入、资源化、绑定面一视同仁。

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 字典种类。
const (
	DictSpecialty = "specialty" // 专业（清单与定额共用一套）
	DictRegion    = "region"    // 地区
	DictCategory  = "category"  // 工料机分类（name=完整分类路径）
)

// DictKindLabel 字典种类的中文名（报错与回执用）。
var DictKindLabel = map[string]string{
	DictSpecialty: "专业",
	DictRegion:    "地区",
	DictCategory:  "工料机分类",
}

// DictItem 一条字典项。
type DictItem struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Code      string `json:"code"`   // 外部标准代码（如国标专业代码），机检用；空=无
	Parent    string `json:"parent"` // 上级字典项（地区层级用；空=顶级）
	Level     int    `json:"level"`
	Sort      int    `json:"sort"`
	Note      string `json:"note"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ValidDictKind 报告 kind 是否是受支持的字典种类。
func ValidDictKind(kind string) bool {
	_, ok := DictKindLabel[strings.TrimSpace(kind)]
	return ok
}

// ── 期数 ─────────────────────────────────────────────────────────────

var periodRe = regexp.MustCompile(`^(19|20)\d{2}-(0[1-9]|1[0-2])$`)

// NormalizePeriod 期数写法规矩：库里统一存 YYYY-MM。
// 认这些写法：2026-06 / 2026.6 / 2026年6月 / 2026/6 / 202606。其余（如「2026年第
// 7 期」「一季度」）不猜——信息价期数与年月不是处处一一对应，让模型换算或问用户。
// 空串原样返回 ok=true（空=待整理）。
func NormalizePeriod(s string) (string, bool) {
	t := strings.Join(strings.Fields(s), "")
	if t == "" {
		return "", true
	}
	var year, month string
	switch {
	case strings.Contains(t, "年"):
		i := strings.Index(t, "年")
		year, month = t[:i], t[i+len("年"):]
		month = strings.TrimSuffix(month, "月")
	case strings.Contains(t, "-"):
		parts := strings.SplitN(t, "-", 2)
		year, month = parts[0], parts[1]
	case strings.Contains(t, "."):
		parts := strings.SplitN(t, ".", 2)
		year, month = parts[0], parts[1]
	case strings.Contains(t, "/"):
		parts := strings.SplitN(t, "/", 2)
		year, month = parts[0], parts[1]
	case len(t) == 6 && allDigits(t):
		year, month = t[:4], t[4:]
	default:
		return "", false
	}
	if !allDigits(year) || !allDigits(month) {
		return "", false
	}
	m, err := strconv.Atoi(month)
	if err != nil {
		return "", false
	}
	out := fmt.Sprintf("%s-%02d", year, m)
	if !periodRe.MatchString(out) {
		return "", false
	}
	return out, true
}

// CurrentPeriod 今天的期（YYYY-MM）——导入没给计价期时的缺省。
func CurrentPeriod() string { return time.Now().Format("2006-01") }

// ── 地区（省级行政区）────────────────────────────────────────────────

// provinces 省级行政区全称（34 个）。地区只到省、不能下分（2026-11-27 用户拍板）：
// 库里带地区的行（项目 / 工料机 / 价格事实 / 定额）一律以省为地域粒度——市县差价
// 是市场价口径，走工料机按期调价，不另开地区行。
var provinces = []string{
	"北京市", "天津市", "河北省", "山西省", "内蒙古自治区", "辽宁省", "吉林省", "黑龙江省",
	"上海市", "江苏省", "浙江省", "安徽省", "福建省", "江西省", "山东省", "河南省",
	"湖北省", "湖南省", "广东省", "广西壮族自治区", "海南省", "重庆市", "四川省", "贵州省",
	"云南省", "西藏自治区", "陕西省", "甘肃省", "青海省", "宁夏回族自治区", "新疆维吾尔自治区",
	"台湾省", "香港特别行政区", "澳门特别行政区",
}

// NormalizeRegion 地区归一到省级行政区。认三种写法：全称（四川省）、去尾缀简称
// （四川 / 内蒙古 / 香港）、「省/市」两级写法取省段（四川省/成都市 → 四川省）。
// 认不出（市县名、 别名简称）= false——地区不能下分。空串原样 ok=true（空=待整理）。
func NormalizeRegion(value string) (string, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", true
	}
	if i := strings.IndexByte(v, '/'); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	for _, p := range provinces {
		if v == p || v == provinceStem(p) {
			return p, true
		}
	}
	return "", false
}

// provinceStem 省级全称去行政区划尾缀的叫法（四川省→四川、新疆维吾尔自治区→新疆）。
func provinceStem(p string) string {
	for _, suffix := range []string{"特别行政区", "维吾尔自治区", "壮族自治区", "回族自治区", "自治区", "省", "市"} {
		if s, ok := strings.CutSuffix(p, suffix); ok {
			return s
		}
	}
	return p
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// IsKnownKind 类别是否在四种之内（按库内归一口径比较；供工具层的演练点名用）。
func IsKnownKind(k string) bool { return isKnownKind(normalizeKind(k)) }

// ── 存储层校验与字典读写 ─────────────────────────────────────────────

// checkDict 规范值校验：值必须在字典里；空=待整理，放行。
func (s *Store) checkDict(kind, value, where string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	ok, err := s.DictHas(kind, value)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s「%s」不在%s字典里（%s）——新增走 cost_library action=dict add（需用户确认），或改成字典里已有的值",
			DictKindLabel[kind], value, DictKindLabel[kind], where)
	}
	return nil
}

// checkRegion 地区校验并归一（地区只到省、不能下分；空=待整理放行）。先把门口写法
// 收成省级全称（「四川省/成都市」→「四川省」），再过字典；返回归一后的省名，
// 调用方必须落归一值。
func (s *Store) checkRegion(value, where string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	p, ok := NormalizeRegion(value)
	if !ok {
		return "", fmt.Errorf("地区「%s」不是省级行政区（%s）——地区只到省、不能下分，请写省名（如 四川省、内蒙古自治区）；新省份走 cost_library action=dict add（需用户确认）", value, where)
	}
	if err := s.checkDict(DictRegion, p, where); err != nil {
		return "", err
	}
	return p, nil
}

// checkPeriod 期数校验并归一成 YYYY-MM；认不出的写法拒收。
func checkPeriod(value, where string) (string, error) {
	p, ok := NormalizePeriod(value)
	if !ok {
		return "", fmt.Errorf("期数「%s」不是年月写法（%s）——统一写成 YYYY-MM，如 2026-06；信息价期数请换算成年月", strings.TrimSpace(value), where)
	}
	return p, nil
}

// DictHas 报告值是否在字典里。
func (s *Store) DictHas(kind, name string) (bool, error) {
	if err := s.requireDB(); err != nil {
		return false, err
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind=? AND name=?`,
		strings.TrimSpace(kind), strings.TrimSpace(name)).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// DictList 某一种字典的全部条目（按 sort, name）。kind 空=全部种类。
// 构造即非 nil（nil 切片经绑定面序列化成 null）。
func (s *Store) DictList(kind string) ([]DictItem, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	q := `SELECT id, kind, name, code, parent, level, sort, note, created_at, updated_at
FROM gf_dicts WHERE 1=1`
	var args []any
	if k := strings.TrimSpace(kind); k != "" {
		q += ` AND kind=?`
		args = append(args, k)
	}
	q += ` ORDER BY kind, sort, name`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []DictItem{}
	for rows.Next() {
		var it DictItem
		if e := rows.Scan(&it.ID, &it.Kind, &it.Name, &it.Code, &it.Parent, &it.Level, &it.Sort, &it.Note,
			&it.CreatedAt, &it.UpdatedAt); e != nil {
			continue
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DictAdd 新增一条字典项（已存在=幂等返回）。name 必须非空，kind 必须受支持。
// 地区只收省级行政区（不能下分）：门口归一成省级全称再入库。
func (s *Store) DictAdd(item DictItem) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	item.Kind = strings.TrimSpace(item.Kind)
	item.Name = strings.TrimSpace(item.Name)
	if !ValidDictKind(item.Kind) {
		return fmt.Errorf("字典种类须为 specialty（专业）/ region（地区）/ category（工料机分类），收到 %q", item.Kind)
	}
	if item.Name == "" {
		return fmt.Errorf("新增%s需要名称", DictKindLabel[item.Kind])
	}
	if item.Kind == DictRegion {
		p, ok := NormalizeRegion(item.Name)
		if !ok {
			return fmt.Errorf("地区只到省、不能下分：「%s」不是省级行政区——请写省名（如 四川省、内蒙古自治区）", item.Name)
		}
		item.Name = p
	}
	if item.Level <= 0 {
		item.Level = 1
	}
	ok, err := s.DictHas(item.Kind, item.Name)
	if err != nil || ok {
		return err
	}
	now := nowText()
	_, err = s.db.Exec(`INSERT INTO gf_dicts(kind, name, code, parent, level, sort, note, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?)`, item.Kind, item.Name, strings.TrimSpace(item.Code), strings.TrimSpace(item.Parent),
		item.Level, item.Sort, strings.TrimSpace(item.Note), now, now)
	if err != nil {
		return fmt.Errorf("新增%s「%s」失败: %w", DictKindLabel[item.Kind], item.Name, err)
	}
	return nil
}

// DictUpdate 改一条字典项：改名（级联到所有引用它的数据）、改上级/代码/排序/备注。
// rename 空=不改名；改名后的名字不能与同种类的另一条重名。返回级联改动的行数
// （按表分组的可读摘要，供回执转述）。
func (s *Store) DictUpdate(kind, name, rename, parent, code string, sort int, note string) (map[string]int, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	rename = strings.TrimSpace(rename)
	var id int64
	if err := s.db.QueryRow(`SELECT id FROM gf_dicts WHERE kind=? AND name=?`, kind, name).Scan(&id); err != nil {
		return nil, fmt.Errorf("%s字典里没有「%s」", DictKindLabel[kind], name)
	}
	if rename != "" && rename != name {
		if kind == DictRegion {
			// 地区改名同样只到省：新名必须是省级行政区（存归一后的全称）。
			p, ok := NormalizeRegion(rename)
			if !ok {
				return nil, fmt.Errorf("地区只到省、不能下分：「%s」不是省级行政区——请写省名（如 四川省、内蒙古自治区）", rename)
			}
			rename = p
		}
		if ok, err := s.DictHas(kind, rename); err != nil {
			return nil, err
		} else if ok {
			return nil, fmt.Errorf("%s字典里已有「%s」，不能重名", DictKindLabel[kind], rename)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	cascaded := map[string]int{}
	if rename != "" && rename != name {
		// cascade 对一张表的一个列做改名，记录改动行数。
		cascade := func(table, col, where string, args ...any) error {
			res, err := tx.Exec(`UPDATE `+table+` SET `+col+`=? WHERE `+col+`=?`+where, append([]any{rename, name}, args...)...)
			if err != nil {
				if strings.Contains(err.Error(), "UNIQUE constraint") {
					return fmt.Errorf("%s.%s 改名后与既有数据的唯一约束冲突（%s「%s」改成的名字与已有数据重复）: %w", table, col, DictKindLabel[kind], name, err)
				}
				return fmt.Errorf("%s.%s 改名失败: %w", table, col, err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				cascaded[table+"."+col] = int(n)
			}
			return nil
		}
		switch kind {
		case DictSpecialty:
			if err := cascade("gf_quotas", "specialty", ""); err != nil {
				return nil, err
			}
			if err := cascade("gf_projects", "specialty", ""); err != nil {
				return nil, err
			}
			if err := cascade("gf_bill_items", "specialty", ""); err != nil {
				return nil, err
			}
		case DictRegion:
			// 地区是工料机身份的一部分：改名后与既有行撞身份会被唯一索引拦下，整批回滚。
			if err := cascade("gf_resources", "region", ""); err != nil {
				return nil, err
			}
			if err := cascade("gf_resource_prices", "region", ""); err != nil {
				return nil, err
			}
			if err := cascade("gf_projects", "region", ""); err != nil {
				return nil, err
			}
		case DictCategory:
			// 分类改名级联到下级路径与工料机：「材料/土建材」改成「材料/土建材料」时，
			// 连同「…/钢材」这些下级一起换前缀。改名后重名会被唯一索引拦下，整批回滚。
			if err := cascade("gf_resources", "category_path", ""); err != nil {
				return nil, err
			}
			if err := cascade("gf_dicts", "name", " AND kind='category'"); err != nil {
				return nil, err
			}
			for _, t := range []struct{ table, col string }{{"gf_resources", "category_path"}, {"gf_dicts", "name"}} {
				where := ""
				if t.table == "gf_dicts" {
					where = " AND kind='category'"
				}
				res, err := tx.Exec(`UPDATE `+t.table+` SET `+t.col+`=? || substr(`+t.col+`, length(?)+1)
WHERE `+t.col+` LIKE ?`+where, rename, name, name+"/%")
				if err != nil {
					return nil, fmt.Errorf("%s.%s 下级改名失败: %w", t.table, t.col, err)
				}
				if n, _ := res.RowsAffected(); n > 0 {
					cascaded[t.table+"."+t.col+"（下级）"] = int(n)
				}
			}
		}
		if _, err := tx.Exec(`UPDATE gf_dicts SET name=? WHERE id=?`, rename, id); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(`UPDATE gf_dicts SET code=COALESCE(NULLIF(?,''), code), parent=?,
  sort=?, note=? , updated_at=? WHERE id=?`,
		strings.TrimSpace(code), strings.TrimSpace(parent), sort, strings.TrimSpace(note), nowText(), id); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cascaded, nil
}

// ── 文件里的行像不像一条数据 ────────────────────────────────────────

// MaxCodeRunes 编码长度上限（字符数）。编码是标识，不是句子：工作簿里的说明行
// 整格都是同一段话时，解析器会把它读成一条「编码=名称=单位=这段话」的数据——
// 真实库里确有这样一条零价工料机（2026-11-25 盘底）。库不收这种行。
const MaxCodeRunes = 32

// codePunct 编码里不应出现的句读（出现在编码里=这是说明文字）。
const codePunct = "。，；！？；\n"

// CodeWellFormed 编码像不像编码：非空、不超过 MaxCodeRunes 个字符、不含句读。
// 不合格时返回原因。
func CodeWellFormed(code string) (bool, string) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, "编码为空"
	}
	if n := len([]rune(code)); n > MaxCodeRunes {
		return false, fmt.Sprintf("编码长达 %d 个字符（上限 %d）——这是说明文字不是编码", n, MaxCodeRunes)
	}
	if i := strings.IndexAny(code, codePunct); i >= 0 {
		return false, "编码里有句读——这是说明文字不是编码"
	}
	return true, ""
}

// ── 列表筛选口径 ────────────────────────────────────────────────────

// FilterNone 筛选值哨兵：「只要该字段为空的行」（未标注/未细分的那一桶）。
// 筛选值为空串=不过滤；等于 FilterNone=取空值行；其余=精确相等。
const FilterNone = "__none__"

// FilterOutside 类别筛选哨兵（仅工料机）：「类别不在四类之内的行」。库定了类别只能是
// 人工/材料/机械/外委，存量里类别写成别的东西的行（多为错记的说明行）单列一桶，
// 界面上找得到、导出按同一口径走。
const FilterOutside = "__outside__"

// LibraryFilter 造价库列表的筛选口径（板块列表与导出共用，保证「导出（当前筛选）」
// 导的就是界面上看到的那些行）。零值=不过滤。
type LibraryFilter struct {
	ProjectID int64  `json:"projectId"` // 仅清单：项目 id（0=全部项目）
	Keyword   string `json:"keyword"`   // 名称/编码/特征等包含（忽略大小写）
	Specialty string `json:"specialty"` // 专业（清单按读取口径归一后的专业；定额）
	Division  string `json:"division"`  // 分部（清单）
	Chapter   string `json:"chapter"`   // 章节（定额）
	Kind      string `json:"kind"`      // 类别（工料机：人工/材料/机械/外委）
	Category  string `json:"category"`  // 分类路径（工料机；前缀匹配，选上级分类连带下级）
	Region    string `json:"region"`    // 地区（工料机）
	Period    string `json:"period"`    // 期数（工料机现行价所属期，YYYY-MM）
}

// DictDelete 删除一条字典项。仍被数据引用的字典项不能删（先改数据）。
func (s *Store) DictDelete(kind, name string) error {
	if err := s.requireDB(); err != nil {
		return err
	}
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	ok, err := s.DictHas(kind, name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s字典里没有「%s」", DictKindLabel[kind], name)
	}
	used, err := s.DictUsage(kind, name)
	if err != nil {
		return err
	}
	if used > 0 {
		return fmt.Errorf("%s「%s」仍被 %d 条数据引用，先把这些数据改成别的%s再删", DictKindLabel[kind], name, used, DictKindLabel[kind])
	}
	_, err = s.db.Exec(`DELETE FROM gf_dicts WHERE kind=? AND name=?`, kind, name)
	return err
}

// DictUsage 某个字典值被多少条数据引用。
func (s *Store) DictUsage(kind, name string) (int, error) {
	if err := s.requireDB(); err != nil {
		return 0, err
	}
	count := func(q string) (int, error) {
		var n int
		err := s.db.QueryRow(q, name).Scan(&n)
		return n, err
	}
	switch strings.TrimSpace(kind) {
	case DictSpecialty:
		a, err := count(`SELECT COUNT(*) FROM gf_quotas WHERE specialty=?`)
		if err != nil {
			return 0, err
		}
		b, err := count(`SELECT COUNT(*) FROM gf_projects WHERE specialty=?`)
		if err != nil {
			return 0, err
		}
		c, err := count(`SELECT COUNT(*) FROM gf_bill_items WHERE specialty=?`)
		if err != nil {
			return 0, err
		}
		return a + b + c, nil
	case DictRegion:
		a, err := count(`SELECT COUNT(*) FROM gf_resources WHERE region=?`)
		if err != nil {
			return 0, err
		}
		b, err := count(`SELECT COUNT(*) FROM gf_resource_prices WHERE region=?`)
		if err != nil {
			return 0, err
		}
		c, err := count(`SELECT COUNT(*) FROM gf_projects WHERE region=?`)
		return a + b + c, err
	case DictCategory:
		return count(`SELECT COUNT(*) FROM gf_resources WHERE category_path=?`)
	}
	return 0, fmt.Errorf("未知字典种类 %q", kind)
}
