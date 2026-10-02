package memory

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gaea/gaea/internal/gaea/bm25"
)

// TestSearchIndexBuildAndSearch validates the full index→search pipeline.
func TestSearchIndexBuildAndSearch(t *testing.T) {
	dir := t.TempDir()
	store := Store{Dir: dir}

	// Save a few memories
	store.Save(Memory{
		Name:        "api-key-location",
		Title:       "API Key Location",
		Description: "Where the API key is stored",
		Type:        TypeReference,
		Body:        "The API key lives in ~/.config/gaea/api.key.",
	})
	store.Save(Memory{
		Name:        "user-prefers-go",
		Title:       "User Prefers Go",
		Description: "User likes Go for backend work",
		Type:        TypeUser,
		Body:        "Use Go for all backend services. Avoid Python when possible.",
	})
	store.Save(Memory{
		Name:        "project-deadlines",
		Title:       "Project Deadlines",
		Description: "Key project deadlines for Q3",
		Type:        TypeProject,
		Body:        "Authentication module must be done by end of July. Database migration by August.",
	})

	idx := store.BuildSearchIndex(nil)
	if idx == nil {
		t.Fatal("BuildSearchIndex returned nil")
	}

	// Search for "api key" → should match api-key-location
	matches := idx.Search("api key")
	if len(matches) == 0 {
		t.Fatal("no matches for 'api key'")
	}
	if matches[0].Name != "api-key-location" {
		t.Fatalf("expected api-key-location first, got %s", matches[0].Name)
	}

	// Search for "go backend" → should match user-prefers-go
	matches = idx.Search("go backend")
	if len(matches) == 0 {
		t.Fatal("no matches for 'go backend'")
	}

	// Search for nonexistent → empty
	matches = idx.Search("xyzzy_nonexistent")
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for nonexistent, got %d", len(matches))
	}
}

// TestSearchIndexEmptyStore returns nil.
func TestSearchIndexEmptyStore(t *testing.T) {
	dir := t.TempDir()
	store := Store{Dir: dir}
	idx := store.BuildSearchIndex(nil)
	if idx != nil {
		t.Fatal("expected nil index for empty store")
	}
}

// TestSearchIndexDisabledStore returns nil.
func TestSearchIndexDisabledStore(t *testing.T) {
	store := Store{Dir: ""}
	idx := store.BuildSearchIndex(nil)
	if idx != nil {
		t.Fatal("expected nil index for disabled store")
	}
}

// TestSearchIndexNilReceiver safe for nil.
func TestSearchIndexNilReceiver(t *testing.T) {
	var idx *SearchIndex
	if matches := idx.Search("test"); matches != nil {
		t.Fatal("expected nil from nil receiver")
	}
}

// TestSearchIndexEmptyQuery returns nil.
func TestSearchIndexEmptyQuery(t *testing.T) {
	// 收敛后内部结构：ranker（bm25 单源）+ names/previews/kinds 映射。
	idx := &SearchIndex{
		ranker:   bm25.NewRanker([]bm25.Doc{{ID: 0, Text: "test"}}),
		names:    []string{"a"},
		previews: map[string]string{"a": "A"},
		kinds:    map[string]Kind{"a": KindSemantic},
	}
	if matches := idx.Search(""); matches != nil {
		t.Fatal("expected nil for empty query")
	}
}

// TestSearchIndexRanking returns results sorted by score.
func TestSearchIndexRanking(t *testing.T) {
	// 与收敛前等价的索引形态：三文档 tf=1、docLen 10/8/12（均值 10），
	// auth-docs 命中 auth+database 两个查询词 → 应排第一。
	idx := &SearchIndex{
		ranker: bm25.NewRankerOpts([]bm25.Doc{
			{ID: 0, Text: "auth database aa bb cc dd ee ff gg hh"},
			{ID: 1, Text: "auth aa bb cc dd ee ff"},
			{ID: 2, Text: "database migration aa bb cc dd ee ff gg hh ii"},
		}, memTokenizeOpts, bm25.DefaultParams),
		names:    []string{"auth-docs", "login-flow", "db-migration"},
		previews: map[string]string{"auth-docs": "Auth docs", "login-flow": "Login flow", "db-migration": "DB migration"},
		kinds:    map[string]Kind{"auth-docs": KindSemantic, "login-flow": KindSemantic, "db-migration": KindSemantic},
	}

	matches := idx.Search("auth database")
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	// auth-docs matches both "auth" and "database" → score 2 → should be first
	if matches[0].Name != "auth-docs" {
		t.Fatalf("expected auth-docs first (score 2), got %s (score %d)", matches[0].Name, int(matches[0].Score))
	}
}

// TestTokenizeCJK handles CJK characters as individual tokens.
func TestTokenizeCJK(t *testing.T) {
	tokens := tokenize("你好世界 hello world")
	// "你好世界" should produce individual CJK chars + "hello" + "world"
	foundHello := false
	foundWorld := false
	for _, tok := range tokens {
		if tok == "hello" {
			foundHello = true
		}
		if tok == "world" {
			foundWorld = true
		}
	}
	if !foundHello || !foundWorld {
		t.Fatalf("expected hello/world in tokens, got %v", tokens)
	}
}

// TestLoadBuildsSearchIndex verifies Load() populates the Search field.
func TestLoadBuildsSearchIndex(t *testing.T) {
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "memory")
	os.MkdirAll(storeDir, 0755)

	// Fake a user dir structure
	userDir := dir
	cwd := dir

	store := Store{Dir: storeDir}
	store.Save(Memory{
		Name:        "test-memory",
		Title:       "Test Memory",
		Description: "A test memory entry",
		Type:        TypeProject,
		Body:        "This is a test memory for search index verification.",
	})

	// Can't easily test Load() because it depends on global config paths,
	// but we can test Store.BuildSearchIndex(nil) which Load() calls internally.
	idx := store.BuildSearchIndex(nil)
	if idx == nil {
		t.Fatal("BuildSearchIndex returned nil")
	}
	matches := idx.Search("test memory")
	if len(matches) == 0 {
		t.Fatal("no matches for 'test memory'")
	}

	_ = userDir
	_ = cwd
}

// ── 行为冻结快照（审计 GA3-03/GA3-04/IN2-11）─────────────────────────

// freezeCorpus 覆盖各口径分歧维度（与 internal/gaea/bm25 冻结语料同形，
// 逐字节同源拷贝）。
var freezeCorpus = []string{
	"你好",
	"你好世界",
	"高频液压振动锤",
	"的了一个是这个",
	"这个功能如何使用",
	"HP300 高频振动锤 台班",
	"Hello World 123",
	"test_case var-name",
	"カタカナと한글",
	"，。！？、；：（）【】.,!?-",
	"～℃±",
	"混合mixed文本text2026",
	"",
	"   ",
}

func fmtTokensFreeze(ts []string) string {
	out := ""
	for i, t := range ts {
		if i > 0 {
			out += "|"
		}
		out += t
	}
	return out
}

// freezeMemories 检索快照语料：kind 覆盖 semantic/episodic/procedural，
// zzq-forecast 无 Title（preview 退化为 name 去连字符），substring
// fallback 命中路径依赖它；其 Body 刻意不含 zzq（BM25 零命中才走 fallback）。
var freezeMemories = []Memory{
	{Name: "api-key-location", Title: "API Key Location", Description: "Where the API key is stored", Type: TypeReference, Kind: KindSemantic, Body: "The API key lives in ~/.config/gaea/api.key."},
	{Name: "deploy-runbook", Title: "Deploy Runbook", Description: "How to ship a release", Type: TypeProject, Kind: KindProcedural, Body: "Run make release, then tag the build. Rollback via git revert."},
	{Name: "incident-jan", Title: "January Incident", Description: "Postgres outage retrospective", Type: TypeProject, Kind: KindEpisodic, Body: "Primary database failed over; replicas lagged behind."},
	{Name: "zzq-forecast", Description: "quarterly numbers", Type: TypeProject, Kind: KindSemantic, Body: "numbers live in the vault"},
}

// freezeDocsSource 与记忆一并入索引的层级文档（doc: 命名空间路径）。
var freezeDocsSource = []Source{{Path: "AGENTS.md", Body: "Always run go vet before commit. Keep memory files small."}}

// TestTokenizeFreezeSnapshot 钉死本包分词口径：与 bm25 同一算法族
// （四脚本 CJK 整串+行内 bigram、下划线词字符、逐 rune 小写），差异仅
// 在停用词过滤（的/了/一个/这个/如何…整 token 丢弃）与两字串不去重
// ——统一后由 bm25.TokenizeOpts 开关组合逐字节复现（GA3-03 单源）。
func TestTokenizeFreezeSnapshot(t *testing.T) {
	want := []string{
		"你好|你好",
		"你好世界|你好|好世|世界",
		"高频液压振动锤|高频|频液|液压|压振|振动|动锤",
		"的了一个是这个|的了|了一|个是|是这",
		"这个功能如何使用|个功|功能|能如|何使|使用",
		"hp300|高频振动锤|高频|频振|振动|动锤|台班|台班",
		"hello|world|123",
		"test_case|var|name",
		"カタカナと한글|カタ|タカ|カナ|ナと|と한|한글",
		"",
		"",
		"混合|混合|mixed|文本|文本|text2026",
		"",
		"",
	}
	for i, text := range freezeCorpus {
		if got := fmtTokensFreeze(tokenize(text)); got != want[i] {
			t.Errorf("tokenize(%q) = %q, want %q", text, got, want[i])
		}
	}
}

// TestSearchIndexRetrievalFreezeSnapshot 钉死检索结果（名字+全精度分值+
// 顺序）：覆盖普通命中、kind 过滤命中、kind 过滤为空、substring fallback
// （BM25 零命中→preview 子串 0.1）。SearchIndex 改为 bm25.Ranker 薄封装
// （IN2-11）后此快照必须一字不改。
func TestSearchIndexRetrievalFreezeSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := Store{Dir: dir}
	for _, m := range freezeMemories {
		if _, err := store.Save(m); err != nil {
			t.Fatalf("save %s: %v", m.Name, err)
		}
	}
	idx := store.BuildSearchIndex(freezeDocsSource)
	if idx == nil {
		t.Fatal("BuildSearchIndex nil")
	}
	format := func(ms []SearchMatch) string {
		out := ""
		for i, m := range ms {
			if i > 0 {
				out += ", "
			}
			out += m.Name + ":" + strconv.FormatFloat(m.Score, 'g', -1, 64)
		}
		return out
	}
	type qc struct{ query, want string }
	cases := []qc{
		{"api key", "api-key-location:4.399225378976851"},
		{"database", "incident-jan:1.4313364161743274"},
		{"zzq", "zzq-forecast:0.1"},
		{"deploy release", "deploy-runbook:2.9179129196070823"},
	}
	for _, c := range cases {
		if got := format(idx.Search(c.query)); got != c.want {
			t.Errorf("Search(%q) = [%s], want [%s]", c.query, got, c.want)
		}
	}
	// kind 过滤与 fallback 的行为面（pin 由 KIND: 前缀查询一并生成）。
	kindCases := []qc{
		{"deploy release", "deploy-runbook:2.9179129196070823"},
		{"api key", ""},
	}
	for _, c := range kindCases {
		got := format(idx.SearchByKind(c.query, KindProcedural))
		if got != c.want {
			t.Errorf("SearchByKind(%q, procedural) = [%s], want [%s]", c.query, got, c.want)
		}
	}
}
