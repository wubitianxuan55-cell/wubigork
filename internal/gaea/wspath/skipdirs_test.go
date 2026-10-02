package wspath

import "testing"

// 六份历史清单（收敛前逐字复制，作为对照基准；顺序按原文件）。
var (
	histFilewatch = []string{".git", "node_modules", "dist", "build", ".venv", "vendor", ".gaea", "releases", ".tmp", "backups"}
	histFileindex = []string{".git", "node_modules", "dist", "build", ".venv", "vendor", ".gaea", "releases"}
	histWssearch  = []string{".git", "node_modules", "dist", "build", ".cache", ".codegraph", ".tianxuan", ".reasonix"}
	histGrep      = []string{"node_modules", ".git", "dist", "build", "target", "__pycache__", ".next", ".nuxt", ".cache", ".venv", "venv", "coverage", "out", ".turbo", ".devenv"}
	histCompress  = []string{"node_modules", ".git", "dist", "build", "target", "__pycache__", ".next", ".nuxt", ".cache", ".venv", "venv", "coverage", "out", ".turbo", ".devenv"}
	// TODO(主代理收线)：internal/app/gaea_ui_extra.go:449 的 searchSkipDirs 属线 1
	// 领地，本批不动；此处按原文件逐字登记，供收线时改成
	// wspath.Merge(wspath.CoreSkipDirs, wspath.SearchExtra)（与本表逐项相同）。
	histApp = []string{".git", "node_modules", "dist", "build", ".cache", ".codegraph", ".tianxuan", ".reasonix"}
)

func asSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, d := range list {
		m[d] = true
	}
	return m
}

func sameSet(a, b []string) bool {
	sa, sb := asSet(a), asSet(b)
	if len(sa) != len(sb) {
		return false
	}
	for k := range sa {
		if !sb[k] {
			return false
		}
	}
	return true
}

// TestHistoricalProfilesPreserved 逐项对照：每个站点的「内核 ∪ 增量」必须与
// 收敛前该站点的清单**逐项相同**——这是「不偷偷改变任何面板扫描范围」的机器守卫。
func TestHistoricalProfilesPreserved(t *testing.T) {
	cases := []struct {
		site       string
		extra      []string
		historical []string
		ordered    bool // 原文件是切片且顺序参与语义时逐位比对
	}{
		{"filewatch.DefaultSkipDirs", WatchExtra, histFilewatch, true},
		{"fileindex.skipDirs", IndexExtra, histFileindex, false},
		{"wssearch.skipDirs", SearchExtra, histWssearch, false},
		{"tool/builtin/grep.go grepNoiseDirs", GrepExtra, histGrep, false},
		{"agent/compress.go noiseDirs", GrepExtra, histCompress, false},
		{"app/gaea_ui_extra.go searchSkipDirs（豁免）", SearchExtra, histApp, false},
	}
	for _, c := range cases {
		got := Merge(CoreSkipDirs, c.extra)
		if !sameSet(got, c.historical) {
			t.Errorf("%s：内核∪增量 = %v，与历史清单 %v 不是同一集合", c.site, got, c.historical)
			continue
		}
		if len(got) != len(c.historical) {
			t.Errorf("%s：长度 %d ≠ 历史 %d（存在重复元素）", c.site, len(got), len(c.historical))
		}
		if c.ordered {
			for i := range got {
				if got[i] != c.historical[i] {
					t.Errorf("%s：第 %d 项 %q ≠ 历史 %q（顺序也须一致，它由 New() 建 map 前无需排序，但保持原序以免调用方依赖变化）", c.site, i, got[i], c.historical[i])
				}
			}
		}
	}
}

// TestSixListsAreNotIdentical 把六份清单的实际差异钉死（差异就是差异，不抹平）。
// 已知成对的相同项只有两组：grep ≡ compress（审计记 "mirrors"）、wssearch ≡ app
// （wssearch 注释自称「与 app 层资料概览保持一致」）；其余两两必须不同。
func TestSixListsAreNotIdentical(t *testing.T) {
	pairs := []struct {
		name string
		list []string
	}{
		{"filewatch", histFilewatch}, {"fileindex", histFileindex},
		{"wssearch", histWssearch}, {"grep", histGrep},
		{"compress", histCompress}, {"app", histApp},
	}
	expectedSame := map[string]bool{"grep|compress": true, "wssearch|app": true}
	for i := range pairs {
		for j := i + 1; j < len(pairs); j++ {
			key := pairs[i].name + "|" + pairs[j].name
			same := sameSet(pairs[i].list, pairs[j].list)
			switch {
			case expectedSame[key] && !same:
				t.Errorf("%s 与 %s 历史清单应逐项相同，实测不同", pairs[i].name, pairs[j].name)
			case !expectedSame[key] && same:
				t.Errorf("%s 与 %s 历史清单实测相同——审计称各写各的/不一致，需复核", pairs[i].name, pairs[j].name)
			}
		}
	}
	// 三处真实差异（doc 级证据）：
	if IsSkippedIn(".tmp", SearchExtra) || IsSkippedIn("backups", GrepExtra) {
		t.Error(".tmp/backups 只属 filewatch 增量，不应出现在其它站点")
	}
	if IsSkippedIn(".codegraph", GrepExtra) || IsSkippedIn("target", SearchExtra) {
		t.Error("搜索类增量与 grep 类增量不得互相渗透")
	}
	// 交集必须恰为 4 项
	if len(CoreSkipDirs) != 4 {
		t.Errorf("内核应恰为 4 项（六份清单的交集），实为 %d", len(CoreSkipDirs))
	}
	for _, d := range CoreSkipDirs {
		for _, site := range [][]string{histFilewatch, histFileindex, histWssearch, histGrep, histCompress, histApp} {
			if !asSet(site)[d] {
				t.Errorf("%q 不在每个站点的历史清单里，不能算交集", d)
			}
		}
	}
}

// TestUnionIsCompleteAndDeduped 并集 SkipDirs 必须恰好覆盖六份清单的并集且无重复。
func TestUnionIsCompleteAndDeduped(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range SkipDirs {
		if seen[d] {
			t.Errorf("SkipDirs 含重复项 %q", d)
		}
		seen[d] = true
	}
	for _, site := range [][]string{histFilewatch, histFileindex, histWssearch, histGrep, histCompress, histApp} {
		for _, d := range site {
			if !seen[d] {
				t.Errorf("SkipDirs 缺 %q（某站点历史清单里有它）", d)
			}
		}
	}
	if len(seen) != 23 {
		t.Errorf("并集应为 23 项，实为 %d：%v", len(seen), SkipDirs)
	}
	// 新增目录只需改 SkipDirs：每个增量都必须是它的子集。
	for name, extra := range map[string][]string{
		"IndexExtra": IndexExtra, "WatchExtra": WatchExtra,
		"SearchExtra": SearchExtra, "GrepExtra": GrepExtra,
	} {
		for _, d := range extra {
			if !IsSkipped(d) {
				t.Errorf("%s 的元素 %q 不在 SkipDirs 里——增量与并集漂移", name, d)
			}
		}
	}
}

// TestPredicateSemantics 钉死两个口径的差别与语义（并集 vs 内核）。
func TestPredicateSemantics(t *testing.T) {
	if !IsSkipped(".tianxuan") {
		t.Error("IsSkipped 应按并集口径认 .tianxuan")
	}
	if IsSkippedIn(".tianxuan", SearchExtra) != true {
		t.Error("wssearch 增量含 .tianxuan")
	}
	if IsSkippedIn(".tianxuan", GrepExtra) {
		t.Error("grep 增量不含 .tianxuan")
	}
	if !IsSkippedIn("node_modules", nil) {
		t.Error("内核成员对任意 extra 都应为 true")
	}
	if IsSkippedIn("src", nil) || IsSkipped("src") {
		t.Error("普通目录不应被判为跳过")
	}
}

// TestWssearchKeepsGaeaSearchable 是红线守卫：并集口径含 .gaea，但 wssearch 的
// 口径绝不能含它——.gaea/exports 的交付产物必须保持可被工位检索
// （wssearch_test.go 的 TestSearchSkipsNoiseDirs 与 S1.2 验收线）。
func TestWssearchKeepsGaeaSearchable(t *testing.T) {
	if !IsSkipped(".gaea") {
		t.Error("并集口径应含 .gaea（filewatch/fileindex 会跳过整棵 .gaea 子树）")
	}
	if IsSkippedIn(".gaea", SearchExtra) {
		t.Error("wssearch/app 资料概览口径不得含 .gaea：会把 .gaea/exports 交付产物挡在检索面外（破 S1.2 验收线）")
	}
	for _, d := range SearchExtra {
		if d == ".gaea" || d == ".venv" || d == "vendor" || d == "releases" || d == ".tmp" || d == "backups" {
			t.Errorf("SearchExtra 混入了文件索引/监听类增量 %q，会改变工作区搜索范围", d)
		}
	}
}
