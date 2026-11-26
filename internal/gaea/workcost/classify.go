package workcost

// classify.go — 整理归类（2026-11-25 用户拍板：库定规矩，文件不一定有那么详细，
// 由模型整理归类）。三件事：
//
//   - Dims         维度盘点（只读）：字典里有什么、库里现在怎么分布、还有多少
//                  待整理。模型归类前先看它对齐叫法，不另造新词。
//   - Classify     批量整理归类：项目/清单/定额/工料机的专业、分部、章节、分类、
//                  地区、期数。预览与落库共用同一套校验与冲突判定；落库整批一个
//                  事务。定额专业固定——只补空值或纠错，纠错以引用它的清单项为准。
//   - BorrowQuota  借鉴生成：本专业没有可用定额时，以外专业定额为底生成本专业的
//                  新定额，可同时把清单行的挂接改过去。
//
// 所有规范值都过规矩（rules.go）：对不上字典、期数不是年月的条目不执行，进冲突
// 清单交模型/用户裁决，不静默丢弃。

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// executor *sql.DB 与 *sql.Tx 的公共面——预览与落库共用一套逻辑。
type executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// ── 维度盘点 ─────────────────────────────────────────────────────────

// DimCount 一个取值与它的条数（Name 空=未标注）。
type DimCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DimProject 项目在维度上的现状。
type DimProject struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Location    string `json:"location"`
	Specialty   string `json:"specialty"`
	Region      string `json:"region"`
	PricePeriod string `json:"pricePeriod"`
	ItemCount   int    `json:"itemCount"`
}

// DimChapter 定额章节分布（专业内）。
type DimChapter struct {
	Specialty string `json:"specialty"`
	Chapter   string `json:"chapter"`
	Count     int    `json:"count"`
}

// DimDivision 清单分部分布（项目内）。
type DimDivision struct {
	ProjectID int64  `json:"projectId"`
	Project   string `json:"project"`
	Division  string `json:"division"`
	Count     int    `json:"count"`
}

// PendingCounts 待整理计数（相应字段为空的条数）。
type PendingCounts struct {
	ProjectSpecialty int `json:"projectSpecialty"` // 项目未标专业
	BillSpecialty    int `json:"billSpecialty"`    // 清单行归一后仍无专业
	QuotaSpecialty   int `json:"quotaSpecialty"`   // 定额未标专业
	QuotaChapter     int `json:"quotaChapter"`     // 定额未归章节
	ResourceRegion   int `json:"resourceRegion"`   // 工料机未标地区
	ResourceCategory int `json:"resourceCategory"` // 工料机未细分
	ResourcePeriod   int `json:"resourcePeriod"`   // 工料机现行价没有期
}

// Dims 维度盘点（只读）。所有切片构造即非 nil。
type Dims struct {
	Dicts            map[string][]DictItem `json:"dicts"`
	Projects         []DimProject          `json:"projects"`
	BillSpecialties  []DimCount            `json:"billSpecialties"`
	QuotaSpecialties []DimCount            `json:"quotaSpecialties"`
	Chapters         []DimChapter          `json:"chapters"`
	Regions          []DimCount            `json:"regions"`
	Periods          []DimCount            `json:"periods"`
	Categories       []DimCount            `json:"categories"`
	Divisions        []DimDivision         `json:"divisions"`
	Pending          PendingCounts         `json:"pending"`
}

// Dims 维度盘点（只读）：字典、各维度的取值分布、待整理计数。
func (s *Store) Dims() (*Dims, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	d := &Dims{
		Dicts:            map[string][]DictItem{},
		Projects:         []DimProject{},
		BillSpecialties:  []DimCount{},
		QuotaSpecialties: []DimCount{},
		Chapters:         []DimChapter{},
		Regions:          []DimCount{},
		Periods:          []DimCount{},
		Categories:       []DimCount{},
		Divisions:        []DimDivision{},
	}
	for _, kind := range []string{DictSpecialty, DictRegion, DictCategory} {
		items, err := s.DictList(kind)
		if err != nil {
			return nil, err
		}
		d.Dicts[kind] = items
	}
	projects, err := s.BillProjects()
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		d.Projects = append(d.Projects, DimProject{
			ID: p.ID, Name: p.Name, Location: p.Location, Specialty: p.Specialty, Region: p.Region,
			PricePeriod: p.PricePeriod, ItemCount: p.ItemCount,
		})
		if p.Specialty == "" {
			d.Pending.ProjectSpecialty++
		}
	}
	counts := func(dst *[]DimCount, q string, args ...any) error {
		rows, err := s.db.Query(q, args...)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var c DimCount
			if err := rows.Scan(&c.Name, &c.Count); err != nil {
				continue
			}
			*dst = append(*dst, c)
		}
		return rows.Err()
	}
	// 清单专业按读取口径归一（行自身空则项目默认专业）。
	if err := counts(&d.BillSpecialties, `SELECT COALESCE(NULLIF(b.specialty,''), NULLIF(p.specialty,''), ''), COUNT(*)
FROM gf_bill_items b JOIN gf_projects p ON p.id=b.project_id GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return nil, err
	}
	if err := counts(&d.QuotaSpecialties, `SELECT COALESCE(specialty,''), COUNT(*) FROM gf_quotas GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return nil, err
	}
	if err := counts(&d.Regions, `SELECT COALESCE(region,''), COUNT(*) FROM gf_resources GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return nil, err
	}
	if err := counts(&d.Periods, `SELECT COALESCE(price_date,''), COUNT(*) FROM gf_resources GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return nil, err
	}
	if err := counts(&d.Categories, `SELECT COALESCE(category_path,''), COUNT(*) FROM gf_resources GROUP BY 1 ORDER BY 2 DESC`); err != nil {
		return nil, err
	}
	for _, c := range d.BillSpecialties {
		if c.Name == "" {
			d.Pending.BillSpecialty = c.Count
		}
	}
	for _, c := range d.QuotaSpecialties {
		if c.Name == "" {
			d.Pending.QuotaSpecialty = c.Count
		}
	}
	for _, c := range d.Regions {
		if c.Name == "" {
			d.Pending.ResourceRegion = c.Count
		}
	}
	for _, c := range d.Periods {
		if c.Name == "" {
			d.Pending.ResourcePeriod = c.Count
		}
	}
	for _, c := range d.Categories {
		if c.Name == "" {
			d.Pending.ResourceCategory = c.Count
		}
	}
	rows, err := s.db.Query(`SELECT COALESCE(specialty,''), chapter, COUNT(*) FROM gf_quotas WHERE chapter<>'' GROUP BY 1,2 ORDER BY 1, 3 DESC`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c DimChapter
		if err := rows.Scan(&c.Specialty, &c.Chapter, &c.Count); err != nil {
			continue
		}
		d.Chapters = append(d.Chapters, c)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var unlabeled int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM gf_quotas WHERE chapter=''`).Scan(&unlabeled)
	d.Pending.QuotaChapter = unlabeled

	rows, err = s.db.Query(`SELECT p.id, p.name, b.division, COUNT(*) FROM gf_bill_items b JOIN gf_projects p ON p.id=b.project_id
WHERE b.division<>'' GROUP BY p.id, b.division ORDER BY p.id, 4 DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v DimDivision
		if err := rows.Scan(&v.ProjectID, &v.Project, &v.Division, &v.Count); err != nil {
			continue
		}
		d.Divisions = append(d.Divisions, v)
	}
	return d, rows.Err()
}

// PendingSamplesKey 一类待整理条目的键（与 PendingCounts 的字段一一对应）。
type PendingSamplesKey struct {
	Field string // PendingCounts 的字段名（回执据此取总数）
	Title string // 回执里的类别名
}

// PendingSampleKeys 待整理明细的类别与出场顺序。
var PendingSampleKeys = []PendingSamplesKey{
	{"ProjectSpecialty", "项目未标专业"},
	{"BillSpecialty", "清单无专业"},
	{"QuotaSpecialty", "定额未标专业"},
	{"QuotaChapter", "定额未归章节"},
	{"ResourceRegion", "工料机未标地区"},
	{"ResourceCategory", "工料机未细分"},
	{"ResourcePeriod", "工料机无期数"},
}

// PendingSamples 待整理条目明细（只读）：dims 的「待整理」只给计数，模型只能逐条
// 枚举去找是哪几条（2026-11-26 造价反馈，旺平 #6）。返回 PendingCounts 字段名 →
// 条目摘要（每类最多 limit 条，0 取 20；没有待整理的类别不出键）。条目带 classify
// 要用的键：清单行带数字 id，定额与工料机带编码。
func (s *Store) PendingSamples(limit int) (map[string][]string, error) {
	if limit <= 0 {
		limit = 20
	}
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	out := map[string][]string{}
	collect := func(field, q string, scan func(rows *sql.Rows) (string, error)) error {
		rows, err := s.db.Query(q, limit)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			item, serr := scan(rows)
			if serr != nil {
				return serr
			}
			out[field] = append(out[field], item)
		}
		return rows.Err()
	}
	text := func(v sql.NullString) string { return strings.TrimSpace(v.String) }
	if err := collect("ProjectSpecialty", `SELECT id, name FROM gf_projects
WHERE TRIM(COALESCE(specialty,''))='' ORDER BY id`, func(rows *sql.Rows) (string, error) {
		var id int64
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return "", err
		}
		return fmt.Sprintf("#%d %s", id, text(name)), nil
	}); err != nil {
		return nil, err
	}
	// 清单专业按读取口径归一（行自身空则项目默认专业），与 Dims 的计数同口径。
	if err := collect("BillSpecialty", `SELECT b.id, b.code, b.title, p.name
FROM gf_bill_items b JOIN gf_projects p ON p.id=b.project_id
WHERE COALESCE(NULLIF(b.specialty,''), NULLIF(p.specialty,''), '')='' ORDER BY b.id`, func(rows *sql.Rows) (string, error) {
		var id int64
		var code, title, project sql.NullString
		if err := rows.Scan(&id, &code, &title, &project); err != nil {
			return "", err
		}
		return fmt.Sprintf("#%d [%s] %s（%s）", id, text(code), text(title), text(project)), nil
	}); err != nil {
		return nil, err
	}
	if err := collect("QuotaSpecialty", `SELECT code, title FROM gf_quotas
WHERE COALESCE(specialty,'')='' ORDER BY code`, func(rows *sql.Rows) (string, error) {
		var code, title sql.NullString
		if err := rows.Scan(&code, &title); err != nil {
			return "", err
		}
		return fmt.Sprintf("[%s] %s", text(code), text(title)), nil
	}); err != nil {
		return nil, err
	}
	if err := collect("QuotaChapter", `SELECT code, title, COALESCE(specialty,'') FROM gf_quotas
WHERE chapter='' ORDER BY code`, func(rows *sql.Rows) (string, error) {
		var code, title, specialty sql.NullString
		if err := rows.Scan(&code, &title, &specialty); err != nil {
			return "", err
		}
		item := fmt.Sprintf("[%s] %s", text(code), text(title))
		if sp := text(specialty); sp != "" {
			item += "（专业 " + sp + "）"
		}
		return item, nil
	}); err != nil {
		return nil, err
	}
	resScan := func(rows *sql.Rows) (string, error) {
		var code, kind, title sql.NullString
		if err := rows.Scan(&code, &kind, &title); err != nil {
			return "", err
		}
		return fmt.Sprintf("[%s] %s %s", text(code), text(kind), text(title)), nil
	}
	for field, col := range map[string]string{"ResourceRegion": "region", "ResourceCategory": "category_path", "ResourcePeriod": "price_date"} {
		q := fmt.Sprintf(`SELECT code, kind, title FROM gf_resources WHERE COALESCE(%s,'')='' ORDER BY code`, col)
		if err := collect(field, q, resScan); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ── 批量整理归类 ─────────────────────────────────────────────────────

// ClassifyPlan 一次批量整理归类的输入。每个字段给了才改；没给的条目与字段不动。
// 专业、地区、分类必须是字典里的值，期数是 YYYY-MM（或能归一的年月写法）。
type ClassifyPlan struct {
	Projects  []ClassifyProject  `json:"projects"`
	Bills     []ClassifyBill     `json:"bills"`
	Quotas    []ClassifyQuota    `json:"quotas"`
	Resources []ClassifyResource `json:"resources"`
}

// ClassifyProject 项目的专业/地区/计价期。
type ClassifyProject struct {
	ID          int64  `json:"id"`
	Specialty   string `json:"specialty"`
	Region      string `json:"region"`
	PricePeriod string `json:"price_period"`
}

// ClassifyBill 清单行的专业（覆盖）与分部（整理命名）。
type ClassifyBill struct {
	ID        int64  `json:"id"`
	Specialty string `json:"specialty"`
	Division  string `json:"division"`
}

// ClassifyQuota 定额的专业与章节。专业固定：只补空值，或按引用它的清单项纠错。
type ClassifyQuota struct {
	Code      string `json:"code"`
	Specialty string `json:"specialty"`
	Chapter   string `json:"chapter"`
}

// ClassifyResource 工料机的类别纠错/分类/地区/期数。
// Kind 只用于把不在四类内的类别改对；Region 只能填空值；Period 只能填空值。
type ClassifyResource struct {
	Code         string `json:"code"`
	Kind         string `json:"kind"`
	CategoryPath string `json:"category_path"`
	Region       string `json:"region"`
	Period       string `json:"period"`
}

// ClassifySummary 按目标值汇总的一行（回执直接转述）。
type ClassifySummary struct {
	Dimension string `json:"dimension"` // 项目专业 / 清单专业 / 定额章节 / 工料机分类 …
	Value     string `json:"value"`
	Count     int    `json:"count"`
}

// ClassifyConflict 一条没执行的条目及原因（交模型/用户裁决）。
type ClassifyConflict struct {
	Tier   string `json:"tier"` // project / bill / quota / resource
	Code   string `json:"code"`
	Title  string `json:"title"`
	Field  string `json:"field"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// ClassifyResult 整理归类结果：改动汇总 + 没执行的冲突清单（构造即非 nil）。
type ClassifyResult struct {
	Applied   ClassifyTally      `json:"applied"`
	Summary   []ClassifySummary  `json:"summary"`
	Conflicts []ClassifyConflict `json:"conflicts"`
}

// ClassifyTally 实际改动的条数（按层级）。
type ClassifyTally struct {
	Projects  int `json:"projects"`
	Bills     int `json:"bills"`
	Quotas    int `json:"quotas"`
	Resources int `json:"resources"`
}

// PreviewClassify 演练：与 ApplyClassify 同一套校验与冲突判定，不写一行。
func (s *Store) PreviewClassify(plan ClassifyPlan) (*ClassifyResult, error) {
	return s.classify(plan, false)
}

// ApplyClassify 落库：整批一个事务；个别条目的冲突不执行（列入 Conflicts），
// 其余照常落。意外错误整批回滚。
func (s *Store) ApplyClassify(plan ClassifyPlan) (*ClassifyResult, error) {
	return s.classify(plan, true)
}

func (s *Store) classify(plan ClassifyPlan, write bool) (*ClassifyResult, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	// 演练与落库跑的是同一批语句——区别只在最后：演练回滚，落库提交。所以两者
	// 一律在事务里跑（演练绝不能走库句柄：那就成了真写）。
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var ex executor = tx
	res := &ClassifyResult{Summary: []ClassifySummary{}, Conflicts: []ClassifyConflict{}}
	tally := map[string]int{}
	count := func(dimension, value string) { tally[dimension+"\x00"+value]++ }
	conflict := func(tier, code, title, field, value, reason string) {
		res.Conflicts = append(res.Conflicts, ClassifyConflict{Tier: tier, Code: code, Title: title, Field: field, Value: value, Reason: reason})
	}
	// 规范值校验：返回空串=通过。
	checkValue := func(label, value string) string {
		switch label {
		case "专业":
			return errText(s.checkDict(DictSpecialty, value, "专业"))
		case "地区":
			return errText(s.checkDict(DictRegion, value, "地区"))
		case "分类":
			return errText(s.checkDict(DictCategory, value, "工料机分类"))
		case "期数":
			if _, ok := NormalizePeriod(value); !ok {
				return fmt.Sprintf("期数「%s」不是年月写法——统一写成 YYYY-MM", value)
			}
		}
		return ""
	}
	now := nowText()

	// ── 项目 ──
	for _, in := range plan.Projects {
		var name string
		if err := ex.QueryRow(`SELECT name FROM gf_projects WHERE id=?`, in.ID).Scan(&name); err != nil {
			conflict("project", fmt.Sprintf("#%d", in.ID), "", "", "", "项目不存在")
			continue
		}
		type field struct{ col, val, dim, label string }
		fields := []field{
			{"specialty", strings.TrimSpace(in.Specialty), "项目专业", "专业"},
			{"region", strings.TrimSpace(in.Region), "项目地区", "地区"},
			{"price_period", strings.TrimSpace(in.PricePeriod), "项目计价期", "期数"},
		}
		bad := ""
		for _, f := range fields {
			if f.val != "" {
				if reason := checkValue(f.label, f.val); reason != "" {
					bad = reason
					conflict("project", fmt.Sprintf("#%d", in.ID), name, f.col, f.val, reason)
				}
			}
		}
		if bad != "" {
			continue
		}
		changed := false
		for _, f := range fields {
			if f.val == "" {
				continue
			}
			val := f.val
			if f.label == "期数" {
				val, _ = NormalizePeriod(f.val)
			}
			out, err := ex.Exec(`UPDATE gf_projects SET `+f.col+`=?, updated_at=? WHERE id=? AND `+f.col+`<>?`, val, now, in.ID, val)
			if err != nil {
				return nil, err
			}
			if n, _ := out.RowsAffected(); n > 0 {
				count(f.dim, val)
				changed = true
			}
		}
		if changed {
			res.Applied.Projects++
		}
	}

	// ── 清单行 ──
	for _, in := range plan.Bills {
		var title string
		if err := ex.QueryRow(`SELECT title FROM gf_bill_items WHERE id=?`, in.ID).Scan(&title); err != nil {
			conflict("bill", fmt.Sprintf("#%d", in.ID), "", "", "", "清单行不存在")
			continue
		}
		sp := strings.TrimSpace(in.Specialty)
		div := strings.TrimSpace(in.Division)
		if sp != "" {
			if reason := checkValue("专业", sp); reason != "" {
				conflict("bill", fmt.Sprintf("#%d", in.ID), title, "specialty", sp, reason)
				continue
			}
		}
		changed := false
		if sp != "" {
			out, err := ex.Exec(`UPDATE gf_bill_items SET specialty=? WHERE id=? AND COALESCE(specialty,'')<>?`, sp, in.ID, sp)
			if err != nil {
				return nil, err
			}
			if n, _ := out.RowsAffected(); n > 0 {
				count("清单专业", sp)
				changed = true
			}
		}
		if div != "" {
			out, err := ex.Exec(`UPDATE gf_bill_items SET division=? WHERE id=? AND COALESCE(division,'')<>?`, div, in.ID, div)
			if err != nil {
				return nil, err
			}
			if n, _ := out.RowsAffected(); n > 0 {
				count("清单分部", div)
				changed = true
			}
		}
		if changed {
			res.Applied.Bills++
		}
	}

	// ── 定额（专业固定：只补空值或纠错） ──
	for _, in := range plan.Quotas {
		code := strings.TrimSpace(in.Code)
		var title, cur string
		if err := ex.QueryRow(`SELECT title, COALESCE(specialty,'') FROM gf_quotas WHERE code=?`, code).Scan(&title, &cur); err != nil {
			conflict("quota", code, "", "", "", "定额不存在")
			continue
		}
		sp := strings.TrimSpace(in.Specialty)
		ch := strings.TrimSpace(in.Chapter)
		changed := false
		if sp != "" && sp != cur {
			if reason := checkValue("专业", sp); reason != "" {
				conflict("quota", code, title, "specialty", sp, reason)
				continue
			}
			refs, err := s.quotaRefSpecialties(ex, code)
			if err != nil {
				return nil, err
			}
			allowed, reason := quotaSpecialtyChange(cur, sp, code, refs)
			if !allowed {
				conflict("quota", code, title, "specialty", sp, reason)
				continue
			}
			if _, err := ex.Exec(`UPDATE gf_quotas SET specialty=?, updated_at=? WHERE code=?`, sp, now, code); err != nil {
				return nil, err
			}
			count("定额专业", sp)
			cur = sp
			changed = true
		}
		if ch != "" {
			out, err := ex.Exec(`UPDATE gf_quotas SET chapter=?, updated_at=? WHERE code=? AND COALESCE(chapter,'')<>?`, ch, now, code, ch)
			if err != nil {
				return nil, err
			}
			if n, _ := out.RowsAffected(); n > 0 {
				count("定额章节·"+cur, ch)
				changed = true
			}
		}
		if changed {
			res.Applied.Quotas++
		}
	}

	// ── 工料机：逐字段判定。一个维度对不上只拦那一个，同一行其他合法维度照常写入，
	// 每个对不上的维度各报一条冲突（不让第一个冲突把后面的都吞掉）。 ──
	for _, in := range plan.Resources {
		code := strings.TrimSpace(in.Code)
		r, err := scanResource(ex.QueryRow(resourceCols+` WHERE code=?`, code))
		if err != nil {
			conflict("resource", code, "", "", "", "工料机不存在")
			continue
		}
		changed := false
		fail := func(field, value, reason string) {
			conflict("resource", code, r.Title, field, value, reason)
		}
		// 类别：只纠不在四类内的。类别是身份的一部分，改之前查重。
		if raw := strings.TrimSpace(in.Kind); raw != "" {
			k := normalizeKind(raw)
			probe := *r
			probe.Kind = k
			switch {
			case !isKnownKind(k):
				fail("kind", raw, "类别须为 人工/材料/机械/外委")
			case k == normalizeKind(r.Kind):
				// 已是该类别，不动。
			case isKnownKind(r.Kind):
				fail("kind", raw, "类别已是四类之一，不改身份")
			default:
				if reason := s.identityFree(ex, probe, r.ID); reason != "" {
					fail("kind", raw, reason)
				} else if _, err := ex.Exec(`UPDATE gf_resources SET kind=?, updated_at=? WHERE id=?`, k, now, r.ID); err != nil {
					return nil, err
				} else {
					count("工料机类别", k)
					r.Kind = k
					changed = true
				}
			}
		}
		if cp := strings.TrimSpace(in.CategoryPath); cp != "" {
			if reason := checkValue("分类", cp); reason != "" {
				fail("category_path", cp, reason)
			} else if out, err := ex.Exec(`UPDATE gf_resources SET category_path=?, updated_at=? WHERE id=? AND COALESCE(category_path,'')<>?`, cp, now, r.ID, cp); err != nil {
				return nil, err
			} else if n, _ := out.RowsAffected(); n > 0 {
				count("工料机分类", cp)
				changed = true
			}
		}
		if rg := strings.TrimSpace(in.Region); rg != "" {
			probe := *r
			probe.Region = rg
			switch {
			case r.Region != "" && r.Region != rg:
				fail("region", rg, fmt.Sprintf("工料机已归属地区「%s」——地区是身份的一部分，不改；要「%s」的请另建一条或 dedupe 合并", r.Region, rg))
			default:
				if reason := checkValue("地区", rg); reason != "" {
					fail("region", rg, reason)
				} else if reason := s.identityFree(ex, probe, r.ID); reason != "" {
					fail("region", rg, reason)
				} else if _, err := ex.Exec(`UPDATE gf_resources SET region=?, updated_at=? WHERE id=?`, rg, now, r.ID); err != nil {
					return nil, err
				} else {
					count("工料机地区", rg)
					r.Region = rg
					changed = true
				}
			}
		}
		if p := strings.TrimSpace(in.Period); p != "" {
			np, ok := NormalizePeriod(p)
			switch {
			case !ok:
				fail("price_date", p, "期数不是年月写法——统一写成 YYYY-MM")
			case periodRe.MatchString(r.PriceDate):
				// 期已是规范的年月：不改（调价走 set_price）。存量里认不出的旧写法可以被纠正。
				fail("price_date", p, fmt.Sprintf("期数已是「%s」，不改；调价走 set_price", r.PriceDate))
			default:
				if price := r.EffectivePrice(); price > 0 {
					var facts int
					_ = ex.QueryRow(`SELECT COUNT(*) FROM gf_resource_prices WHERE resource_id=?`, r.ID).Scan(&facts)
					if facts == 0 {
						// 现行价还没有价格事实：按现行价补记一条，历史表不至于空白。
						if _, err := ex.Exec(`INSERT INTO gf_resource_prices(resource_id, price, period, region, price_type, source, fetched_at, note)
VALUES(?,?,?,?,?,?,?,?)`, r.ID, price, np, r.Region, r.PriceType, "整理归类", now, "整理归类时按现行价补记"); err != nil {
							return nil, err
						}
					}
				}
				if _, err := ex.Exec(`UPDATE gf_resources SET price_date=?, updated_at=? WHERE id=?`, np, now, r.ID); err != nil {
					return nil, err
				}
				count("工料机期数", np)
				changed = true
			}
		}
		if changed {
			res.Applied.Resources++
		}
	}

	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := strings.SplitN(k, "\x00", 2)
		res.Summary = append(res.Summary, ClassifySummary{Dimension: parts[0], Value: parts[1], Count: tally[k]})
	}
	if write {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func errText(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// quotaRefSpecialties 引用该定额的清单项的专业集合（读取口径归一，空值不计）。
// 查询走 ex：与本批整理归类同一个事务，看得到本批刚改的项目专业。
func (s *Store) quotaRefSpecialties(ex executor, code string) ([]string, error) {
	rows, err := ex.Query(`SELECT DISTINCT COALESCE(NULLIF(b.specialty,''), NULLIF(p.specialty,''), '')
FROM gf_bill_items b JOIN gf_projects p ON p.id=b.project_id
WHERE CASE WHEN EXISTS(SELECT 1 FROM gf_bill_quota_links x WHERE x.bill_item_id=b.id)
  THEN (SELECT COUNT(*) FROM gf_bill_quota_links l WHERE l.bill_item_id=b.id AND l.quota_code=?) > 0
  ELSE b.quota_code=? END
  AND COALESCE(NULLIF(b.specialty,''), NULLIF(p.specialty,''), '') <> ''
ORDER BY 1`, code, code)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if rows.Scan(&v) == nil {
			out = append(out, v)
		}
	}
	return out, rows.Err()
}

// quotaSpecialtyChange 定额专业能不能从 cur 改成 next（纯函数）。
//
// 定额专业固定，随来源定：只允许两种改法——
//   - 补空值：cur 为空。引用它的清单项要么没标专业，要么同属 next；
//   - 纠错：cur 非空，但引用它的清单项**同属** next（来源唯一且指向 next）。
//
// 其余一律不改：被多个专业的清单共用的定额保持原专业，别的专业借鉴生成
// （borrow_quota）；没有任何清单引用、又已标专业的定额来源不明，不猜。
func quotaSpecialtyChange(cur, next, code string, refs []string) (bool, string) {
	single := ""
	multi := false
	for _, r := range refs {
		if single == "" {
			single = r
		} else if r != single {
			multi = true
		}
	}
	switch {
	case cur == "" && multi:
		return false, fmt.Sprintf("定额尚未标注专业，却被 %s 等多个专业的清单项引用——先不标；各专业用 borrow_quota 借鉴生成", strings.Join(refs, "、"))
	case cur == "":
		if single == "" || single == next {
			return true, ""
		}
		return false, fmt.Sprintf("定额尚未标注专业，引用它的清单项都是「%s」专业——要标就标「%s」", single, single)
	case multi:
		return false, fmt.Sprintf("定额是「%s」专业，被 %s 等多个专业的清单项共用——保持原专业，其他专业的清单行用 borrow_quota 借鉴生成", cur, strings.Join(refs, "、"))
	case len(refs) == 1 && single == next:
		return true, ""
	case single == "":
		return false, fmt.Sprintf("定额已标「%s」专业且没有清单项引用它，来源不明——不改；要别的专业的请 borrow_quota 借鉴生成", cur)
	default:
		return false, fmt.Sprintf("定额是「%s」专业，引用它的清单项是「%s」专业——专业对不上应借鉴生成本专业的（borrow_quota from_code=%s specialty=%s）并改挂，而不是改这条定额的专业", cur, single, code, single)
	}
}

// identityFree 把 probe 当作改后的行，检查它的身份有没有被**别的**行占用；
// 占用了返回原因（空=可用）。地区与类别是身份的一部分，改它们之前必须查。
func (s *Store) identityFree(ex executor, probe Resource, selfID int64) string {
	var id int64
	var code string
	err := ex.QueryRow(`SELECT id, code FROM gf_resources WHERE kind=? AND title=? AND spec=? AND unit=? AND region=? AND id<>? LIMIT 1`,
		normalizeKind(probe.Kind), strings.TrimSpace(probe.Title), strings.TrimSpace(probe.Spec), strings.TrimSpace(probe.Unit),
		strings.TrimSpace(probe.Region), selfID).Scan(&id, &code)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("改后与工料机 %s 的身份（类别+名称+规格+单位+地区）重复——用 dedupe 合并到那一条", code)
}

// ── 借鉴生成 ─────────────────────────────────────────────────────────

// BorrowRelink 一条将改挂（或已改挂）到新定额上的清单行。
type BorrowRelink struct {
	ItemID    int64   `json:"itemId"`
	Code      string  `json:"code"`
	Title     string  `json:"title"`
	Project   string  `json:"project"`
	Specialty string  `json:"specialty"` // 清单行专业（与目标专业不一致时回执要提醒）
	Quantity  float64 `json:"quantity"`  // 该定额在这条清单上的工程量（0=跟随清单工程量）
}

// BorrowResult 借鉴生成的结果（构造即非 nil）。
type BorrowResult struct {
	From      Quota          `json:"from"`
	New       Quota          `json:"new"`     // 演练时未落库（ID=0）
	Existed   bool           `json:"existed"` // 目标专业下已有同源借鉴定额，复用那条
	Relinked  []BorrowRelink `json:"relinked"`
	Conflicts []string       `json:"conflicts"`
}

// BorrowRequest 借鉴生成的输入。
type BorrowRequest struct {
	FromCode  string  `json:"fromCode"`
	Specialty string  `json:"specialty"` // 目标专业（字典值）
	NewCode   string  `json:"newCode"`   // 新定额编码（空=原码-N）
	Title     string  `json:"title"`     // 新定额名称（空=沿用原定额）
	ItemIDs   []int64 `json:"itemIds"`   // 把这些清单行的挂接从原定额改到新定额
}

// BorrowQuota 借鉴生成：清单套定额先找本专业，本专业没有就借鉴别的专业生成本
// 专业的——以外专业定额 fromCode 为底，复制含量行生成目标专业 specialty 的新定额
// （溯源记「借鉴 <专业> <码>」）。目标专业下已借鉴过同一条的复用那条（幂等）。
//
// write=false 演练：同一套校验，不写一行。
func (s *Store) BorrowQuota(req BorrowRequest, write bool) (*BorrowResult, error) {
	if err := s.requireDB(); err != nil {
		return nil, err
	}
	from, err := s.GetQuota(strings.TrimSpace(req.FromCode))
	if err != nil {
		return nil, err
	}
	specialty := strings.TrimSpace(req.Specialty)
	if specialty == "" {
		return nil, fmt.Errorf("借鉴生成需要目标专业 specialty")
	}
	if err := s.checkDict(DictSpecialty, specialty, "目标专业"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(from.Specialty) == specialty {
		return nil, fmt.Errorf("定额 %s 本来就是「%s」专业——同专业直接挂它，不用借鉴", from.Code, specialty)
	}
	derived := "借鉴 " + strings.TrimSpace(from.Specialty) + " " + from.Code
	var ex executor = s.db
	var tx *dirtyTx
	if write {
		var err error
		if tx, err = s.db.Begin(); err != nil {
			return nil, err
		}
		defer func() { _ = tx.Rollback() }()
		ex = tx
	}
	res := &BorrowResult{From: *from, Conflicts: []string{}, Relinked: []BorrowRelink{}}
	// 既有同源借鉴：本专业下已由同一条外专业定额借鉴生成的定额。
	var existCode string
	e := ex.QueryRow(`SELECT code FROM gf_quotas WHERE specialty=? AND derived_from=? ORDER BY code LIMIT 1`, specialty, derived).Scan(&existCode)
	if e != nil && !strings.Contains(e.Error(), "no rows") {
		return nil, e
	}
	code := strings.TrimSpace(req.NewCode)
	switch {
	case e == nil && (code == "" || code == existCode):
		code = existCode
		res.Existed = true
	case code == "":
		code, err = s.freeQuotaCode(from.Code)
		if err != nil {
			return nil, err
		}
	default:
		var n int
		if err := ex.QueryRow(`SELECT COUNT(*) FROM gf_quotas WHERE code=?`, code).Scan(&n); err != nil {
			return nil, err
		}
		if n > 0 {
			return nil, fmt.Errorf("定额编码 %s 已被占用", code)
		}
	}
	idx, err := s.ResourceIndex()
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = from.Title
	}
	items := make([]QuotaItem, 0, len(from.Items))
	for _, it := range from.Items {
		// 只搬资源码、含量、损耗与备注；名称/规格/单位从资源库派生（单一事实源）。
		items = append(items, QuotaItem{ResourceCode: it.ResourceCode, Quantity: it.Quantity, LossRate: it.LossRate, Note: it.Note})
	}
	q := Quota{
		Code: code, OrigCode: from.Code, Title: title, Specialty: specialty, Unit: from.Unit,
		CategoryPath: from.CategoryPath, Source: "借鉴 " + from.Code,
		Note:        fmt.Sprintf("借鉴「%s」专业的 %s 生成；含量行照搬，按本项目特征核对后再用", strings.TrimSpace(from.Specialty), from.Code),
		DerivedFrom: derived, Items: items,
	}
	if !res.Existed {
		if write {
			saved, err := s.saveQuota(ex, q, idx)
			if err != nil {
				return nil, err
			}
			q = *saved
		} else {
			// 演练：把展示字段按资源库补齐，回执里能看到含量行原貌。
			preview := q
			for i := range preview.Items {
				if r, ok := idx[preview.Items[i].ResourceCode]; ok {
					preview.Items[i].Kind, preview.Items[i].Title = r.Kind, r.Title
					preview.Items[i].Spec, preview.Items[i].Unit = r.Spec, r.Unit
				}
			}
			q = preview
		}
	} else if existing, err := s.GetQuota(code); err == nil {
		q = *existing
	}
	res.New = q

	now := nowText()
	for _, id := range req.ItemIDs {
		var it BorrowRelink
		it.ItemID = id
		var linked int
		err := ex.QueryRow(`SELECT b.title, b.code, p.name, COALESCE(NULLIF(b.specialty,''), NULLIF(p.specialty,''), ''),
  COALESCE((SELECT l.quantity FROM gf_bill_quota_links l WHERE l.bill_item_id=b.id AND l.quota_code=?), b.quantity),
  CASE WHEN EXISTS(SELECT 1 FROM gf_bill_quota_links x WHERE x.bill_item_id=b.id)
    THEN (SELECT COUNT(*) FROM gf_bill_quota_links l WHERE l.bill_item_id=b.id AND l.quota_code=?)
    ELSE CASE WHEN b.quota_code=? THEN 1 ELSE 0 END END
FROM gf_bill_items b JOIN gf_projects p ON p.id=b.project_id WHERE b.id=?`, from.Code, from.Code, from.Code, id).
			Scan(&it.Title, &it.Code, &it.Project, &it.Specialty, &it.Quantity, &linked)
		if err != nil {
			res.Conflicts = append(res.Conflicts, fmt.Sprintf("清单行 #%d 不存在", id))
			continue
		}
		if linked == 0 {
			res.Conflicts = append(res.Conflicts, fmt.Sprintf("清单行 %s「%s」（%s）没有挂接定额 %s，不改挂", it.Code, it.Title, it.Project, from.Code))
			continue
		}
		if it.Code == "" {
			it.Code = fmt.Sprintf("#%d", id)
		}
		if write {
			// 已挂着新定额的行只解除旧挂接；其余原位改挂。主定额列同步。
			if _, err := ex.Exec(`UPDATE gf_bill_quota_links SET quota_code=?, updated_at=?
WHERE bill_item_id=? AND quota_code=?
  AND NOT EXISTS(SELECT 1 FROM gf_bill_quota_links y WHERE y.bill_item_id=? AND y.quota_code=?)`,
				code, now, id, from.Code, id, code); err != nil {
				return nil, err
			}
			if _, err := ex.Exec(`DELETE FROM gf_bill_quota_links WHERE bill_item_id=? AND quota_code=?`, id, from.Code); err != nil {
				return nil, err
			}
			if _, err := ex.Exec(`UPDATE gf_bill_items SET quota_code=? WHERE id=? AND quota_code=?`, code, id, from.Code); err != nil {
				return nil, err
			}
		}
		res.Relinked = append(res.Relinked, it)
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	return res, nil
}
