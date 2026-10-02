// Package wspath 是工作区「噪声目录」判定的唯一真相源（审计 2026-10-02 GA5-17）。
//
// 收敛前同一语义在六处各写各的：filewatch.DefaultSkipDirs（10 项）、
// fileindex.skipDirs（8）、wssearch.skipDirs（8）、tool/builtin/grep.go 的
// grepNoiseDirs（15）、agent/compress.go 的 noiseDirs（15），以及本次豁免收敛的
// app/gaea_ui_extra.go searchSkipDirs（8）。六份互不相等——只有
// .git / node_modules / dist / build 是交集。
//
// 本包把它们合并为一张并集目录 SkipDirs（新增产物目录只改这里），站点差异用
// 参数化增量表达：历史站点一律调用 IsSkippedIn(name, 该站点增量)，口径为
// 「交集内核 CoreSkipDirs ∪ 增量」，与收敛前该站点的清单逐项相同，因此
// **不改变任何面板的扫描范围**。
//
// 为什么不把并集直接当站点默认：并集含 .gaea，而 wssearch 的红线是
// 「.gaea/exports 交付产物必须可被工位检索」（wssearch_test.go 的
// TestSearchSkipsNoiseDirs，以及 S1.2「工位搜索搜不到乐园记忆/产物」验收线）；
// 把并集当默认会直接破线并打挂既有测试。故：
//
//	IsSkipped(name)           并集口径：新调用方的默认（不含历史包袱）；
//	IsSkippedIn(name, extra)  内核口径：CoreSkipDirs ∪ extra，历史站点唯一入口。
//
// 本包另外持有「工作区路径归一 / 穿越防护」两个原语（Within / ResolveRelWithin，
// 审计 2026-10-02 AP5-09 收口），见 resolve.go——同属工作区路径语义，故合包而非
// 另立 leaf 包。本包仍是叶子包：不得反向 import internal/app 等大包（防依赖倒挂）。
package wspath

// SkipDirs 是全部已知噪声目录（六份历史清单的并集，23 项，去重）。
//
// 这是本语义的单一真相源：新增产物目录（如 .tmp-pdf/.dsh-*）只加这里，
// 各站点通过各自的增量自动可见（增量必须是 SkipDirs 的子集，见测试）。
var SkipDirs = []string{
	// ── 六份清单的交集（CoreSkipDirs，4 项）──
	".git", "node_modules", "dist", "build",
	// ── fileindex / filewatch 增量 ──
	".venv", "vendor", ".gaea", "releases",
	// ── filewatch 独有增量 ──
	".tmp", "backups",
	// ── wssearch / app 资料概览增量 ──
	".cache", ".codegraph", ".tianxuan", ".reasonix",
	// ── grep / agent compress 增量 ──
	"target", "__pycache__", ".next", ".nuxt",
	"venv", "coverage", "out", ".turbo", ".devenv",
}

// CoreSkipDirs 是六份历史清单的交集（4 项）：所有站点都跳过的内核，也是
// IsSkippedIn 的基准集。
var CoreSkipDirs = []string{".git", "node_modules", "dist", "build"}

// IndexExtra 是 fileindex.skipDirs 相对核心的增量（收敛前 8 项 = 核心 + 本增量）。
var IndexExtra = []string{".venv", "vendor", ".gaea", "releases"}

// WatchExtra 是 filewatch.DefaultSkipDirs 相对核心的增量（收敛前 10 项）。
var WatchExtra = []string{".venv", "vendor", ".gaea", "releases", ".tmp", "backups"}

// SearchExtra 是 wssearch.skipDirs（以及 app 层资料概览 searchSkipDirs）相对
// 核心的增量（收敛前两者都是 8 项 = 核心 + 本增量）。
//
// 注意本增量**不含 .gaea**：.gaea 下只有 sessions/archive/cache/play 是噪音，
// 由 wssearch.isNoiseRel 单独判定，.gaea/exports 必须保持可检索。
var SearchExtra = []string{".cache", ".codegraph", ".tianxuan", ".reasonix"}

// GrepExtra 是 grep 工具与 agent 树压缩器相对核心的增量（收敛前两者都是 15 项，
// 逐项相同 = 核心 + 本增量）。
var GrepExtra = []string{
	".venv", "target", "__pycache__", ".next", ".nuxt",
	".cache", "venv", "coverage", "out", ".turbo", ".devenv",
}

// Merge 返回 base ∪ extra 的新切片（不改动入参，允许重复元素）。
func Merge(base, extra []string) []string {
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
}

// IsSkipped 判断 name 是否属于并集 SkipDirs（最完整口径，供新调用方使用）。
func IsSkipped(name string) bool {
	for _, d := range SkipDirs {
		if d == name {
			return true
		}
	}
	return false
}

// IsSkippedIn 判断 name 是否属于 CoreSkipDirs ∪ extra。
//
// 历史站点用它 + 各自的增量（IndexExtra/WatchExtra/SearchExtra/GrepExtra）来
// 保持收敛前的扫描范围逐项不变；新增差异一律走 extra，不要再新建并列清单。
func IsSkippedIn(name string, extra []string) bool {
	for _, d := range CoreSkipDirs {
		if d == name {
			return true
		}
	}
	for _, d := range extra {
		if d == name {
			return true
		}
	}
	return false
}
