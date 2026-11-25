package db

// schema_specialties_test.go — 规划专业进字典（SchemaV40，2026-11-25 用户令「清单和
// 定额的专业重新规划」并逐项调整定稿 10 个）：代号互不相同、库里已有的同名专业只补
// 代号不覆盖用户手填的码、幂等、地区与工料机分类仍不预设。

import (
	"database/sql"
	"strings"
	"testing"
)

// 用户逐项点名定稿的专业，与点名不要的专业——钉住，防日后再被「顺手加回」。
var (
	wantSpecialties = []string{"土壤修复", "地下水修复", "水污染防治", "固废处置",
		"房屋建筑与装饰工程", "通用安装工程", "市政工程", "临建", "措施", "检测"}
	rejectedSpecialties = []string{"大气污染防治", "物理污染防治", "构筑物工程",
		"仿古建筑工程", "园林绿化工程", "矿山工程", "城市轨道交通工程", "爆破工程"}
)

// specialtyRow 专业字典的一行。
type specialtyRow struct {
	Name string
	Code string
	Sort int
	Note string
}

// specialtyRows 读专业字典（按 sort）。
func specialtyRows(t *testing.T, db *sql.DB) []specialtyRow {
	t.Helper()
	rows, err := db.Query(`SELECT name, COALESCE(code,''), sort, COALESCE(note,'')
FROM gf_dicts WHERE kind='specialty' ORDER BY sort, name`)
	if err != nil {
		t.Fatalf("读专业字典失败: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []specialtyRow
	for rows.Next() {
		var r specialtyRow
		if err := rows.Scan(&r.Name, &r.Code, &r.Sort, &r.Note); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("读专业字典中断: %v", err)
	}
	return out
}

// TestPlannedSpecialtiesWellFormed 规划表本身：代号两位、互不相同、互不重名；正好是
// 用户点名的那 10 个。
func TestPlannedSpecialtiesWellFormed(t *testing.T) {
	codes, names := map[string]bool{}, map[string]bool{}
	for _, p := range PlannedSpecialties {
		if len(p.Code) != 2 || strings.ToUpper(p.Code) != p.Code {
			t.Errorf("专业 %s 的代号 %q 应为两位大写字母数字", p.Name, p.Code)
		}
		if codes[p.Code] {
			t.Errorf("代号 %s 重复（%s）", p.Code, p.Name)
		}
		if names[p.Name] {
			t.Errorf("专业 %s 重复", p.Name)
		}
		codes[p.Code], names[p.Name] = true, true
	}
	if len(PlannedSpecialties) != len(wantSpecialties) {
		t.Errorf("规划专业应 %d 个，得到 %d", len(wantSpecialties), len(PlannedSpecialties))
	}
	for _, n := range wantSpecialties {
		if !names[n] {
			t.Errorf("规划表缺专业 %q", n)
		}
	}
	for _, n := range rejectedSpecialties {
		if names[n] {
			t.Errorf("用户点名不要，规划表不该有 %q", n)
		}
	}
}

// TestSchemaV40UpgradeFromV39 从 V39 升上来：库里已有的「土壤修复」（没代号、说明是
// V38 的「初始字典」）只补代号 / 排序 / 说明；用户手填过代号的不动（自己加的「市政」
// 填了 JS）；其余补进来。地区 / 工料机分类字典不受影响。
func TestSchemaV40UpgradeFromV39(t *testing.T) {
	dir := buildAtVersion(t, 39)
	{
		raw, err := sql.Open("sqlite", DatabasePath(dir))
		if err != nil {
			t.Fatal(err)
		}
		ins := `INSERT INTO gf_dicts(kind, name, code, note) VALUES(?,?,?,?)`
		for _, d := range [][4]string{
			{"specialty", "土壤修复", "", "初始字典：取自库内已有取值"},
			{"specialty", "市政", "JS", "用户自己加的，自己填了代号"},
			{"region", "乐山市", "", "初始字典：取自库内已有取值"},
			{"category", "材料/钢材", "", "初始字典：取自库内已有取值"},
		} {
			if _, err := raw.Exec(ins, d[0], d[1], d[2], d[3]); err != nil {
				t.Fatalf("摆字典 %v 失败: %v", d, err)
			}
		}
		_ = raw.Close()
	}

	db := GetDatabase(dir)
	if db == nil {
		t.Fatal("升级后库打不开（迁移失败，见日志）")
	}
	defer CloseDatabase(dir)

	var ver int
	if err := db.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Fatalf("user_version = %d, want %d", ver, len(migrations))
	}

	got := specialtyRows(t, db)
	// 规划专业 + 用户自己加的「市政」。
	if len(got) != len(PlannedSpecialties)+1 {
		t.Fatalf("专业字典应 %d 条，得到 %d：%v", len(PlannedSpecialties)+1, len(got), got)
	}
	byName := map[string]specialtyRow{}
	for _, r := range got {
		byName[r.Name] = r
	}
	for i, p := range PlannedSpecialties {
		r, ok := byName[p.Name]
		if !ok {
			t.Errorf("专业 %q 未进字典", p.Name)
			continue
		}
		if r.Code != p.Code {
			t.Errorf("%s 代号 = %q, want %q", p.Name, r.Code, p.Code)
		}
		if r.Note != p.Note {
			t.Errorf("%s 说明 = %q, want %q", p.Name, r.Note, p.Note)
		}
		if i > 0 {
			if prev, _ := byName[PlannedSpecialties[i-1].Name]; prev.Sort >= r.Sort {
				t.Errorf("%s 应排在 %s 之后（sort %d vs %d）", p.Name, PlannedSpecialties[i-1].Name, r.Sort, prev.Sort)
			}
		}
	}
	// 用户手填过代号的条目：名字、代号、说明都不动。
	if r := byName["市政"]; r.Code != "JS" || r.Note != "用户自己加的，自己填了代号" {
		t.Errorf("用户手填代号的专业不应被动，得到 %v", r)
	}
	for kind, want := range map[string]int{"specialty": len(PlannedSpecialties) + 1, "region": 1, "category": 1} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind=?`, kind).Scan(&n); err != nil || n != want {
			t.Errorf("%s 字典应 %d 条，得到 %d（err=%v）——地区与工料机分类不预设", kind, want, n, err)
		}
	}

	// 幂等：把同一段迁移再跑一遍，字典不变。
	before := specialtyRows(t, db)
	if _, err := db.Exec(SchemaV40); err != nil {
		t.Fatalf("重跑 V40 失败: %v", err)
	}
	after := specialtyRows(t, db)
	if len(before) != len(after) {
		t.Fatalf("重跑 V40 不应增减条目：%d → %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("重跑 V40 改动了第 %d 条：%v → %v", i, before[i], after[i])
		}
	}
}

// TestSchemaV40CodesUnique 库里全部专业的代号两两不同（自有码的专业段靠它区分）。
func TestSchemaV40CodesUnique(t *testing.T) {
	dir := t.TempDir()
	db := GetDatabase(dir)
	if db == nil {
		t.Fatal("GetDatabase returned nil")
	}
	defer CloseDatabase(dir)
	var dup int
	if err := db.QueryRow(`SELECT COUNT(*) FROM (SELECT code FROM gf_dicts WHERE kind='specialty' GROUP BY code HAVING COUNT(*)>1)`).Scan(&dup); err != nil {
		t.Fatal(err)
	}
	if dup != 0 {
		t.Errorf("专业字典代号应两两不同，有 %d 个重码", dup)
	}
}
