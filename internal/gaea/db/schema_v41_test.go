package db

import (
	"database/sql"
	"testing"
)

// TestSchemaV41UpgradeFromV40 工料机分类字典种子（2026-11-25 用户令「补充工料机分类
// 字典，现在无法分类」）：从 V40 的库升上来——
//   - 分类树 34 项落进 gf_dicts（kind='category'），编号即自有码分类段（全局两位）；
//   - 用户已建的同名分类：没代号的补上，填过代号的不动；
//   - 目标段号被别的分类占用的项整条跳过（宁缺勿撞）；
//   - 专业等其他 kind 一行不动。
func TestSchemaV41UpgradeFromV40(t *testing.T) {
	dir := buildAtVersion(t, 40)
	{
		raw, err := sql.Open("sqlite", DatabasePath(dir))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = raw.Close() }()
		stmts := []string{
			// 用户自建、没代号 → 迁移补号
			`INSERT INTO gf_dicts(kind,name,code,parent,level,sort,note,created_at,updated_at)
VALUES('category','材料','','',5,0,'初始字典：取自库内已有取值','t','t')`,
			// 用户自建、已填代号 → 不动
			`INSERT INTO gf_dicts(kind,name,code,parent,level,sort,note,created_at,updated_at)
VALUES('category','自定义分类','77','','2',0,'','t','t')`,
			// 占住了段号 10（=种子表里 木材及模板 的号）→ 该项整条跳过
			`INSERT INTO gf_dicts(kind,name,code,parent,level,sort,note,created_at,updated_at)
VALUES('category','周转材料','10','','2',0,'','t','t')`,
			// 别的 kind 不许动
			`INSERT INTO gf_dicts(kind,name,code,parent,level,sort,note,created_at,updated_at)
VALUES('region','铜陵','','',1,0,'','t','t')`,
		}
		for _, q := range stmts {
			if _, err := raw.Exec(q); err != nil {
				t.Fatalf("摆旧库失败: %v\n%s", err, q)
			}
		}
	}

	g := GetDatabase(dir)
	if g == nil {
		t.Fatal("升级后库打不开（迁移失败，见日志）")
	}
	defer CloseDatabase(dir)

	var ver int
	if err := g.QueryRow("SELECT CAST(value AS INTEGER) FROM schema_meta WHERE key='user_version'").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != len(migrations) {
		t.Fatalf("user_version = %d, want %d", ver, len(migrations))
	}

	// ① 数量：34 项种子里「木材及模板」因段号 10 被占整条跳过（−1），「材料」
	// 与用户已建的同名行合并（−1），其余 32 项落库；再加用户 3 行 = 35。
	var n int
	if err := g.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind='category'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if want := len(PlannedCategories) - 2 + 3; n != want {
		t.Fatalf("category 字典 %d 行, want %d（种子 %d − 跳过 1 − 并入 1 + 用户 3）", n, want, len(PlannedCategories))
	}

	// ② 顶级与样例段号（表序即发号序）。
	cases := map[string]string{
		"人工":         "01",
		"材料":         "04",
		"材料/土建材料/钢材": "06",
		"外委/其他外委":    "34",
	}
	for name, want := range cases {
		var code string
		if err := g.QueryRow(`SELECT code FROM gf_dicts WHERE kind='category' AND name=?`, name).Scan(&code); err != nil {
			t.Fatalf("分类 %s 不在字典: %v", name, err)
		}
		if code != want {
			t.Fatalf("分类 %s code=%s, want %s", name, code, want)
		}
	}

	// ③ 用户口径优先：已填代号的不动；被跳过的不发号。
	var code string
	if err := g.QueryRow(`SELECT code FROM gf_dicts WHERE kind='category' AND name='自定义分类'`).Scan(&code); err != nil || code != "77" {
		t.Fatalf("自定义分类 code=%q err=%v, want 77（用户填过的不动）", code, err)
	}
	if err := g.QueryRow(`SELECT code FROM gf_dicts WHERE kind='category' AND name='木材及模板'`).Scan(&code); err == nil {
		t.Fatalf("木材及模板 应因段号 10 被占而整条跳过，实得 code=%q", code)
	}

	// ④ 段号全局唯一（同 kind=category 内无重码）。
	var dups int
	if err := g.QueryRow(`SELECT COUNT(*) FROM (SELECT code FROM gf_dicts WHERE kind='category' AND code<>'' GROUP BY code HAVING COUNT(*)>1)`).Scan(&dups); err != nil {
		t.Fatal(err)
	}
	if dups != 0 {
		t.Fatalf("分类段号出现 %d 组重码（分类段必须全局唯一）", dups)
	}

	// ⑤ 层级/上级：三级样例的 parent 与 level。
	var parent string
	var level int
	if err := g.QueryRow(`SELECT parent, level FROM gf_dicts WHERE kind='category' AND name='材料/土建材料/钢材'`).Scan(&parent, &level); err != nil {
		t.Fatal(err)
	}
	if parent != "材料/土建材料" || level != 3 {
		t.Fatalf("钢材 parent=%q level=%d, want 材料/土建材料 / 3", parent, level)
	}

	// ⑥ 别的 kind 一行不动。
	var regions int
	if err := g.QueryRow(`SELECT COUNT(*) FROM gf_dicts WHERE kind='region'`).Scan(&regions); err != nil {
		t.Fatal(err)
	}
	if regions != 1 {
		t.Fatalf("region 字典被种子污染：%d 行, want 1", regions)
	}
}
