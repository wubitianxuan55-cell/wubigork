package workcost

// gcode.go — 造价自有编码（2026-11-25 用户拍板「造价编码自己设计一套，与现在的
// 体系对应」；主从=自有码为主，外部码保留成映射列，检索/导出双码都吃）。
//
// 三族段位（镜像库的维度；段值是授码时的快照——码是身份不是分类，归类/改章节
// 之后码不变）：
//
//	清单   Q + 专业2 + 分部2 + 流水3   如 QSZ01001
//	定额   D + 专业2 + 章节2 + 流水3   如 DSZ01001
//	工料机 R + 类别1 + 分类2 + 流水4   如 RC010001
//
// 段值来源：
//   - 专业段：字典 gf_dicts（kind=specialty）的 code 列。用户给过缩写就用缩写；
//     没给按名称序补两位数字号并写回字典。清单与定额共用一套专业、同一个专业码。
//     空=待整理，固定 00。
//   - 分部 / 章节 / 分类段：同一个名称恒得同一个段号，没归类（空）固定 00。
//     分部、章节在**专业内**编号（清单、定额的作用域是专业；它们没有字典，段号
//     记在水位表的名→号映射里）；分类是**全局**编号，落在字典 code 列（看得见、
//     改得动，一列唯一）。
//   - 工料机类别段：人工R/材料C/机械J/外委W；四类之外（「类别待整理」桶）用 X。
//   - 流水段：每前缀一个水位（gf_gcode_seq），只前进不复用——删行腾出的号
//     永不重发；唯一性由三表的部分唯一索引兜底。
//
// 授码只有一个入口（nextGCode），存量补授（EnsureGCodes）与新行落库
// （saveQuota/SaveBillItem/SaveResource 的 INSERT 分支）都走它。
//
// 坑（2026-11-25 真实库副本取证抓到）：段号计数器必须按**作用域**开，不能按
// 「作用域/名称」开——每个名称各开一个从 1 起的计数器，所有分部都会拿到 01，
// 这一段就白占了（当时 329 条清单全挤在 Q0001 一个前缀下）。

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// gcodeSpecWatermark 专业两位数字号的全局计数器键（清单/定额共用）。
const gcodeSpecWatermark = "SP"

// gcodeSlotPrefix 段号映射与计数器在水位表里的键前缀：计数器 "SL:"+族+作用域，
// 名→号映射 "SL:"+族+作用域+"/"+名称。
const gcodeSlotPrefix = "SL:"

// GCodeFamilies 三族的自述（报错与回执用）。
var GCodeFamilies = map[string]string{
	"Q": "清单",
	"D": "定额",
	"R": "工料机",
}

// gcodeKindCode 工料机类别段：人工R/材料C/机械J/外委W；四类之外（类别待整理）X。
func gcodeKindCode(kind string) string {
	switch normalizeKind(kind) {
	case KindLabor:
		return "R"
	case KindMaterial:
		return "C"
	case KindMachine:
		return "J"
	case KindOutsourced:
		return "W"
	}
	return "X"
}

// validGCodeSeg 两位段是否合格（两位字母数字；字典里手填的缩写归一成大写用）。
func validGCodeSeg(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if !(r >= '0' && r <= '9' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

// normGCodeSeg 段归一（大写）；不合格返回空串。
func normGCodeSeg(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !validGCodeSeg(s) {
		return ""
	}
	return s
}

// bumpWatermark 水位 +1 并持久化，返回新号。水位只前进（MAX 兜底），删行不回退。
func bumpWatermark(ex executor, key string) (int64, error) {
	var last int64
	if err := ex.QueryRow(`SELECT last FROM gf_gcode_seq WHERE prefix=?`, key).Scan(&last); err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	next := last + 1
	if _, err := ex.Exec(`INSERT INTO gf_gcode_seq(prefix, last) VALUES(?,?)
ON CONFLICT(prefix) DO UPDATE SET last=MAX(last, excluded.last)`, key, next); err != nil {
		return 0, err
	}
	return next, nil
}

// watermarkAt 读水位（get 语义，不动指针）；没有返回 0。
func watermarkAt(ex executor, key string) (int64, error) {
	var last int64
	err := ex.QueryRow(`SELECT last FROM gf_gcode_seq WHERE prefix=?`, key).Scan(&last)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return last, err
}

// ── 专业段 ───────────────────────────────────────────────────────────

// specCode 专业段两位码。空专业固定 "00"（待整理），不占号。
//
// 字典里有这个专业：code 列就是它的段（用户给过缩写就用缩写）；没给或已被别的
// 专业占，发一个号并**写回字典**——段号跟着字典行走，专业改名后新行仍得同一段。
// 字典里没有（存量未认领）：段号只记在映射里。
func (s *Store) specCode(ex executor, specialty string) (string, error) {
	specialty = strings.TrimSpace(specialty)
	if specialty == "" {
		return "00", nil
	}
	var dictID int64
	var dictCode string
	err := ex.QueryRow(`SELECT id, COALESCE(code,'') FROM gf_dicts WHERE kind='specialty' AND name=?`,
		specialty).Scan(&dictID, &dictCode)
	inDict := err == nil
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if inDict {
		if c := normGCodeSeg(dictCode); c != "" {
			taken, err := s.specSegTaken(ex, c, specialty)
			if err != nil {
				return "", err
			}
			if !taken {
				return c, nil
			}
		}
	}
	// 映射优先（号一旦发过就稳定），首见才从全局计数器领新号。
	mapping := "SP:" + specialty
	n, err := watermarkAt(ex, mapping)
	if err != nil {
		return "", err
	}
	if n == 0 {
		if n, err = bumpWatermark(ex, gcodeSpecWatermark); err != nil {
			return "", err
		}
		if _, err := ex.Exec(`INSERT INTO gf_gcode_seq(prefix, last) VALUES(?,?)
ON CONFLICT(prefix) DO UPDATE SET last=excluded.last`, mapping, n); err != nil {
			return "", err
		}
	}
	seg := fmt.Sprintf("%02d", n)
	if inDict {
		if _, err := ex.Exec(`UPDATE gf_dicts SET code=?, updated_at=? WHERE id=?`, seg, nowText(), dictID); err != nil {
			return "", err
		}
	}
	return seg, nil
}

// specSegTaken 报告专业段两位码是否已被别的专业占用（任一表里该段有码、且那些行
// 的专业不是它自己）——用户在字典里改码撞车时不沿用撞车的码。
func (s *Store) specSegTaken(ex executor, code, specialty string) (bool, error) {
	for _, t := range []struct{ table, spec string }{
		{"gf_quotas", "specialty"}, {"gf_bill_items", "specialty"},
	} {
		var n int
		err := ex.QueryRow(`SELECT COUNT(*) FROM `+t.table+` WHERE substr(gcode,2,2)=? AND specialty<>?`,
			code, specialty).Scan(&n)
		if err != nil {
			return false, err
		}
		if n > 0 {
			return true, nil
		}
	}
	return false, nil
}

// ── 分部 / 章节 / 分类段 ─────────────────────────────────────────────

// slotOrd 第三段：在作用域内按名称发两位号。
//
//	scopeID：清单/定额=专业段（随字典 code，专业改名不漂）。工料机不用（分类编号
//	        是全局的，见 categorySlot）。
//	name：分部 / 章节 / 分类路径。空=没归类，固定 "00"。
//
// 分类走字典 code 列（categorySlot）；分部、章节没有字典，走水位表里的
// 名→号映射（mappedSlot）。
func (s *Store) slotOrd(ex executor, family, scopeID, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "00", nil
	}
	if family == "R" {
		return s.categorySlot(ex, name)
	}
	return s.mappedSlot(ex, family, scopeID, name)
}

// mappedSlot 作用域内 get-or-create：名称发过号就原样返回；首见让**作用域的**
// 计数器 +1（不是每个名称各开一个计数器），并把名→号记下来。
func (s *Store) mappedSlot(ex executor, family, scopeID, name string) (string, error) {
	counter := gcodeSlotPrefix + family + scopeID
	mapping := counter + "/" + name
	n, err := watermarkAt(ex, mapping)
	if err != nil {
		return "", err
	}
	if n == 0 {
		if n, err = bumpWatermark(ex, counter); err != nil {
			return "", err
		}
		if _, err := ex.Exec(`INSERT INTO gf_gcode_seq(prefix, last) VALUES(?,?)
ON CONFLICT(prefix) DO UPDATE SET last=excluded.last`, mapping, n); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%02d", n), nil
}

// categorySlot 工料机的分类段。分类是字典项：code 列就是它的段（用户可改）；
// 没给就发一个**没人用过的**号并写回字典。
//
// 编号是全局的（所有类别共用一个计数器），所以字典 code 列一列唯一，同一个分类
// 不管挂在哪个类别下都是同一段——真实库里确有「外委的行挂着 材料/辅助材料」这类
// 跨类别用法。按类别各编一套会让字典里出现四个「01」（2026-11-25 真实库副本
// 取证抓到），那一列就不再是机检可用的代码了。库里约 60 个分类，两位够用到 99，
// 超了自然长成三位（唯一性不受影响）。
func (s *Store) categorySlot(ex executor, category string) (string, error) {
	var dictID int64
	var dictCode string
	err := ex.QueryRow(`SELECT id, COALESCE(code,'') FROM gf_dicts WHERE kind='category' AND name=?`,
		category).Scan(&dictID, &dictCode)
	inDict := err == nil
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if inDict {
		if c := normGCodeSeg(dictCode); c != "" {
			return c, nil
		}
	}
	slot, err := s.freeCategorySlot(ex, category)
	if err != nil {
		return "", err
	}
	if inDict {
		if _, err := ex.Exec(`UPDATE gf_dicts SET code=?, updated_at=? WHERE id=?`, slot, nowText(), dictID); err != nil {
			return "", err
		}
	}
	return slot, nil
}

// freeCategorySlot 给分类发一个没人用过的全局号（发过则原样返回）。占用看两处：
// 字典里别的分类手填了同一个码，或库里已有行用了这个段（不论类别）。
func (s *Store) freeCategorySlot(ex executor, category string) (string, error) {
	counter := gcodeSlotPrefix + "R"
	mapping := counter + "/" + category
	if n, err := watermarkAt(ex, mapping); err != nil {
		return "", err
	} else if n > 0 {
		return fmt.Sprintf("%02d", n), nil
	}
	for i := 0; i < 200; i++ {
		n, err := bumpWatermark(ex, counter)
		if err != nil {
			return "", err
		}
		slot := fmt.Sprintf("%02d", n)
		// 分类段在码的第 3、4 位（R + 类别字母 + 两位段号）。
		var inData, inDict int
		if err := ex.QueryRow(`SELECT COUNT(*) FROM gf_resources WHERE gcode<>'' AND substr(gcode,3,2)=?`, slot).Scan(&inData); err != nil {
			return "", err
		}
		if err := ex.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind='category' AND code=? AND name<>?`, slot, category).Scan(&inDict); err != nil {
			return "", err
		}
		if inData == 0 && inDict == 0 {
			if _, err := ex.Exec(`INSERT INTO gf_gcode_seq(prefix, last) VALUES(?,?)
ON CONFLICT(prefix) DO UPDATE SET last=excluded.last`, mapping, n); err != nil {
				return "", err
			}
			return slot, nil
		}
	}
	return "", fmt.Errorf("分类段号已连续 200 个被占用，无法再给分类「%s」发号", category)
}

// ── 授码入口 ─────────────────────────────────────────────────────────

// nextGCode 授码唯一入口：族+专业段+作用域段+流水。width 是流水位数（Q/D 3 位、
// R 4 位；超宽自然增长，唯一性不受影响）。水位持久化在调用方事务里。
func (s *Store) nextGCode(ex executor, table, family, spec, slot string, width int) (string, error) {
	prefix := family + spec + slot
	var last int64
	err := ex.QueryRow(`SELECT last FROM gf_gcode_seq WHERE prefix=?`, prefix).Scan(&last)
	switch {
	case err == nil:
		// 水位已在。
	case err == sql.ErrNoRows:
		// 首次发放：从既有数据收水位（补授中断重跑时数据里可能已有该前缀的码）。
		if e := ex.QueryRow(`SELECT COALESCE(MAX(CAST(substr(gcode,?) AS INTEGER)),0)
FROM `+table+` WHERE substr(gcode,1,?)=? AND length(gcode)>?`,
			len(prefix)+1, len(prefix), prefix, len(prefix)).Scan(&last); e != nil {
			return "", e
		}
	default:
		return "", err
	}
	next := last + 1
	if _, err := ex.Exec(`INSERT INTO gf_gcode_seq(prefix, last) VALUES(?,?)
ON CONFLICT(prefix) DO UPDATE SET last=MAX(last, excluded.last)`, prefix, next); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%0*d", prefix, width, next), nil
}

// newQuotaGCode 给新定额行授自有码（D + 专业2 + 章节2 + 流水3）。
func (s *Store) newQuotaGCode(ex executor, q *Quota) (string, error) {
	spec, err := s.specCode(ex, q.Specialty)
	if err != nil {
		return "", err
	}
	slot, err := s.slotOrd(ex, "D", spec, q.Chapter)
	if err != nil {
		return "", err
	}
	return s.nextGCode(ex, "gf_quotas", "D", spec, slot, 3)
}

// newBillGCode 给新清单行授自有码（Q + 专业2 + 分部2 + 流水3）。专业按读取口径
// 归一：行自身为空取项目默认，与详情/列表显示的专业一致。
func (s *Store) newBillGCode(ex executor, it *BillItem) (string, error) {
	specName := strings.TrimSpace(it.Specialty)
	if specName == "" && it.ProjectID > 0 {
		_ = ex.QueryRow(`SELECT COALESCE(specialty,'') FROM gf_projects WHERE id=?`, it.ProjectID).Scan(&specName)
	}
	spec, err := s.specCode(ex, specName)
	if err != nil {
		return "", err
	}
	slot, err := s.slotOrd(ex, "Q", spec, it.Division)
	if err != nil {
		return "", err
	}
	return s.nextGCode(ex, "gf_bill_items", "Q", spec, slot, 3)
}

// newResourceGCode 给新工料机行授自有码（R + 类别1 + 分类2 + 流水4）。
func (s *Store) newResourceGCode(ex executor, r *Resource) (string, error) {
	kind := gcodeKindCode(r.Kind)
	slot, err := s.slotOrd(ex, "R", "", r.CategoryPath)
	if err != nil {
		return "", err
	}
	return s.nextGCode(ex, "gf_resources", "R", kind, slot, 4)
}

// ── 存量补授 ─────────────────────────────────────────────────────────

// EnsureGCodes 存量补授（幂等，Open 时调用）：给三表 gcode 为空的行授码，一个
// 事务内完成。库还没升到 V39（无 gcode 列）时静默跳过——旧夹具与迁移前的读面
// 不碰码。补授期间与并发写靠 SQLite 写事务串行化；失败整批回滚，下次开库重试。
// 没有待授行时只跑一条计数查询（每次开库都会走到这里，不能重）。
func (s *Store) EnsureGCodes() error {
	if err := s.requireDB(); err != nil {
		return err
	}
	for _, t := range []string{"gf_quotas", "gf_bill_items", "gf_resources"} {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='gcode'`, t).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}
	var pending int
	if err := s.db.QueryRow(`SELECT
  (SELECT COUNT(*) FROM gf_quotas WHERE gcode='')
+ (SELECT COUNT(*) FROM gf_bill_items WHERE gcode='')
+ (SELECT COUNT(*) FROM gf_resources WHERE gcode='')`).Scan(&pending); err != nil {
		return err
	}
	if pending == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// 专业字典先齐码：后面两表的专业段都从字典取（一处映射，Q/D 同码）。
	if err := s.seedSpecialtyCodes(tx); err != nil {
		return err
	}
	if err := s.backfillQuotaGCodes(tx); err != nil {
		return err
	}
	if err := s.backfillBillGCodes(tx); err != nil {
		return err
	}
	if err := s.backfillResourceGCodes(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	slog.Info("cost: 造价自有编码补授完成", "rows", pending)
	return nil
}

// seedSpecialtyCodes 专业字典没码的条目按名称序补两位数字号（跳过已占用的号，
// 含字典里手填的数字码）；水位跟到已发号最大值。
func (s *Store) seedSpecialtyCodes(ex executor) error {
	rows, err := ex.Query(`SELECT name, COALESCE(code,'') FROM gf_dicts WHERE kind='specialty' ORDER BY name`)
	if err != nil {
		return err
	}
	type entry struct{ name, code string }
	var entries []entry
	used := map[int]bool{}
	maxNum := 0
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.name, &e.code); err != nil {
			continue
		}
		entries = append(entries, e)
		// 数字码占号（字母码是另一套命名空间，不占数字号）。
		if c := normGCodeSeg(e.code); c != "" && c[0] >= '0' && c[0] <= '9' && c[1] >= '0' && c[1] <= '9' {
			n := int(c[0]-'0')*10 + int(c[1]-'0')
			used[n] = true
			if n > maxNum {
				maxNum = n
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_ = rows.Close()
	for _, e := range entries {
		if normGCodeSeg(e.code) != "" {
			continue
		}
		n := 1
		for used[n] {
			n++
		}
		used[n] = true
		if n > maxNum {
			maxNum = n
		}
		if _, err := ex.Exec(`UPDATE gf_dicts SET code=?, updated_at=? WHERE kind='specialty' AND name=?`,
			fmt.Sprintf("%02d", n), nowText(), e.name); err != nil {
			return err
		}
	}
	if maxNum > 0 {
		if _, err := ex.Exec(`INSERT INTO gf_gcode_seq(prefix, last) VALUES(?,?)
ON CONFLICT(prefix) DO UPDATE SET last=MAX(last, excluded.last)`, gcodeSpecWatermark, maxNum); err != nil {
			return err
		}
	}
	return nil
}

// codeRow 补授时读到的一行：id + 专业（或类别）+ 分部 / 章节 / 分类。
type codeRow struct {
	id    int64
	scope string
	name  string
}

// pendingRows 读出一张表里待授码的行（按 id，保证发号顺序稳定）。
func pendingRows(ex executor, query string) ([]codeRow, error) {
	rows, err := ex.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []codeRow
	for rows.Next() {
		var r codeRow
		if err := rows.Scan(&r.id, &r.scope, &r.name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// backfillQuotaGCodes 定额：D + 专业2 + 章节2 + 流水3。
func (s *Store) backfillQuotaGCodes(ex executor) error {
	batch, err := pendingRows(ex, `SELECT id, COALESCE(specialty,''), COALESCE(chapter,'') FROM gf_quotas WHERE gcode='' ORDER BY id`)
	if err != nil {
		return err
	}
	for _, r := range batch {
		q := Quota{Specialty: r.scope, Chapter: r.name}
		code, err := s.newQuotaGCode(ex, &q)
		if err != nil {
			return err
		}
		if _, err := ex.Exec(`UPDATE gf_quotas SET gcode=? WHERE id=?`, code, r.id); err != nil {
			return err
		}
	}
	return nil
}

// backfillBillGCodes 清单：Q + 专业2 + 分部2 + 流水3。专业按读取口径归一
// （行自身为空取项目默认），与详情/列表显示的专业一致。
func (s *Store) backfillBillGCodes(ex executor) error {
	batch, err := pendingRows(ex, `SELECT i.id,
  COALESCE(NULLIF(i.specialty,''), NULLIF(p.specialty,''), ''), COALESCE(i.division,'')
FROM gf_bill_items i JOIN gf_projects p ON p.id=i.project_id WHERE i.gcode='' ORDER BY i.id`)
	if err != nil {
		return err
	}
	for _, r := range batch {
		it := BillItem{Specialty: r.scope, Division: r.name}
		code, err := s.newBillGCode(ex, &it)
		if err != nil {
			return err
		}
		if _, err := ex.Exec(`UPDATE gf_bill_items SET gcode=? WHERE id=?`, code, r.id); err != nil {
			return err
		}
	}
	return nil
}

// backfillResourceGCodes 工料机：R + 类别1 + 分类2 + 流水4。
func (s *Store) backfillResourceGCodes(ex executor) error {
	batch, err := pendingRows(ex, `SELECT id, COALESCE(kind,''), COALESCE(category_path,'') FROM gf_resources WHERE gcode='' ORDER BY id`)
	if err != nil {
		return err
	}
	for _, r := range batch {
		res := Resource{Kind: r.scope, CategoryPath: r.name}
		code, err := s.newResourceGCode(ex, &res)
		if err != nil {
			return err
		}
		if _, err := ex.Exec(`UPDATE gf_resources SET gcode=? WHERE id=?`, code, r.id); err != nil {
			return err
		}
	}
	return nil
}
