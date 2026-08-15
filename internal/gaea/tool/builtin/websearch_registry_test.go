package builtin

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
)

// saveSearchGlobals 快照并恢复 searchCfg / searchEngineOrderCfg（包级全局注入）。
func saveSearchGlobals(t *testing.T) {
	t.Helper()
	oldCfg := searchCfg
	oldOrder := searchEngineOrderCfg
	t.Cleanup(func() {
		searchCfg = oldCfg
		searchEngineOrderCfg = oldOrder
	})
}

// TestSearchEngineRegistry_AllKinds 六个引擎均经注册表可构建，kind 列表完整且排序。
func TestSearchEngineRegistry_AllKinds(t *testing.T) {
	wantKinds := []string{
		SearchEngineKindLocalSearXNG, SearchEngineKindTavily, SearchEngineKindBrave,
		SearchEngineKindPublicSearXNG, SearchEngineKindBing, SearchEngineKindDuckDuckGo,
	}
	kinds := SearchEngineKinds()
	if len(kinds) != len(wantKinds) {
		t.Fatalf("SearchEngineKinds = %v, want %d kinds", kinds, len(wantKinds))
	}
	for i := 1; i < len(kinds); i++ {
		if kinds[i-1] > kinds[i] {
			t.Errorf("kinds 未排序: %v", kinds)
		}
	}
	joined := strings.Join(kinds, ",")
	for _, k := range wantKinds {
		if !strings.Contains(joined, k) {
			t.Errorf("缺少 kind %q: %v", k, kinds)
		}
	}

	// 每个 kind 都能经注册表构建出引擎，并报告正确的 Name。
	wantName := map[string]string{
		SearchEngineKindLocalSearXNG:  "local-searxng",
		SearchEngineKindTavily:        "tavily",
		SearchEngineKindBrave:         "brave",
		SearchEngineKindPublicSearXNG: "public-searxng",
		SearchEngineKindBing:          "bing",
		SearchEngineKindDuckDuckGo:    "duckduckgo-lite",
	}
	for _, k := range wantKinds {
		eng, err := NewSearchEngine(k, SearchEngineConfig{})
		if err != nil {
			t.Fatalf("NewSearchEngine(%q): %v", k, err)
		}
		if eng.Name() != wantName[k] {
			t.Errorf("kind %q Name = %q, want %q", k, eng.Name(), wantName[k])
		}
	}
}

// TestSearchEngineRegistry_ConfigRouting 同形配置 + 不同 kind 得到不同引擎实现：
// 切换后端只改 kind，消费方（接口）零改动。
func TestSearchEngineRegistry_ConfigRouting(t *testing.T) {
	searx, err := NewSearchEngine(SearchEngineKindLocalSearXNG, SearchEngineConfig{BaseURL: "http://localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := searx.(*localSearxNGEngine); !ok {
		t.Fatalf("kind=local-searxng 应返回 *localSearxNGEngine, got %T", searx)
	}
	if !searx.Available() {
		t.Error("配置了 BaseURL 的 local-searxng 应 Available")
	}

	tavily, err := NewSearchEngine(SearchEngineKindTavily, SearchEngineConfig{APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tavily.(*tavilyEngine); !ok {
		t.Fatalf("kind=tavily 应返回 *tavilyEngine, got %T", tavily)
	}
	if !tavily.Available() {
		t.Error("配置了 APIKey 的 tavily 应 Available")
	}
}

// TestSearchEngineRegistry_AvailableFiltering Available 语义与旧分支一致：
// 无 URL/key 的引擎不参与扇出，keyless 引擎始终可用。
func TestSearchEngineRegistry_AvailableFiltering(t *testing.T) {
	cases := []struct {
		kind  string
		cfg   SearchEngineConfig
		avail bool
	}{
		{SearchEngineKindLocalSearXNG, SearchEngineConfig{}, false},
		{SearchEngineKindLocalSearXNG, SearchEngineConfig{BaseURL: "http://x"}, true},
		{SearchEngineKindTavily, SearchEngineConfig{}, false},
		{SearchEngineKindTavily, SearchEngineConfig{APIKey: "k"}, true},
		{SearchEngineKindBrave, SearchEngineConfig{}, false},
		{SearchEngineKindBrave, SearchEngineConfig{APIKey: "k"}, true},
		{SearchEngineKindPublicSearXNG, SearchEngineConfig{}, true},
		{SearchEngineKindBing, SearchEngineConfig{}, true},
		{SearchEngineKindDuckDuckGo, SearchEngineConfig{}, true},
	}
	for _, c := range cases {
		eng, err := NewSearchEngine(c.kind, c.cfg)
		if err != nil {
			t.Fatalf("NewSearchEngine(%q): %v", c.kind, err)
		}
		if got := eng.Available(); got != c.avail {
			t.Errorf("Available(%q, %+v) = %v, want %v", c.kind, c.cfg, got, c.avail)
		}
	}
}

// TestSearchEngineRegistry_UnknownKindError 未知 kind fail-closed（附已注册列表）。
func TestSearchEngineRegistry_UnknownKindError(t *testing.T) {
	_, err := NewSearchEngine("no-such-engine", SearchEngineConfig{})
	if err == nil {
		t.Fatal("未知 kind 应报错")
	}
	if !strings.Contains(err.Error(), SearchEngineKindBing) {
		t.Errorf("错误应附已注册 kind 列表: %v", err)
	}
}

// TestSearchEngineRegistry_DuplicateKindPanics 互斥注册：重复即 panic。
func TestSearchEngineRegistry_DuplicateKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("重复注册应 panic")
		}
	}()
	RegisterSearchEngine(SearchEngineKindTavily, func(cfg SearchEngineConfig) (SearchEngine, error) { return nil, nil })
}

// TestSearchEngineRegistry_EmptyKindPanics 空 kind 注册直接 panic。
func TestSearchEngineRegistry_EmptyKindPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("空 kind 应 panic")
		}
	}()
	RegisterSearchEngine("", func(cfg SearchEngineConfig) (SearchEngine, error) { return nil, nil })
}

// TestBuildEngines_DefaultOrder 无配置时按默认优先级扇出（与改造前硬编码序一致）。
func TestBuildEngines_DefaultOrder(t *testing.T) {
	saveSearchGlobals(t)
	searchCfg = nil
	searchEngineOrderCfg = nil

	ws := webSearch{}
	engines := ws.buildEngines()
	// 无任何凭据：local-searxng/tavily/brave 被 Available 过滤，只留 keyless 三件套。
	if len(engines) != 3 {
		t.Fatalf("无配置时引擎数 = %d, want 3（public-searxng/bing/duckduckgo-lite）: %v", len(engines), engineNames(engines))
	}
	for i, k := range []string{SearchEngineKindPublicSearXNG, SearchEngineKindBing, SearchEngineKindDuckDuckGo} {
		if engines[i].Name() != k {
			t.Errorf("engines[%d] = %q, want %q", i, engines[i].Name(), k)
		}
	}
}

// TestBuildEngines_ConfiguredOrder config 可配序：注入的引擎顺序生效，
// 未配置凭据的引擎仍被过滤；切换顺序只改配置、消费方零改动。
func TestBuildEngines_ConfiguredOrder(t *testing.T) {
	saveSearchGlobals(t)
	t.Setenv("TAVILY_API_KEY", "tk")
	t.Setenv("BRAVE_API_KEY", "bk")
	SetSearchConfig(config.SearchConfig{
		LocalSearXNGURL: "http://localhost:8888",
		TavilyAPIKeyEnv: "TAVILY_API_KEY",
		BraveAPIKeyEnv:  "BRAVE_API_KEY",
	})
	SetSearchEngineOrder([]string{SearchEngineKindBing, SearchEngineKindTavily, SearchEngineKindBrave, SearchEngineKindLocalSearXNG})

	ws := webSearch{}
	engines := ws.buildEngines()
	want := []string{SearchEngineKindBing, SearchEngineKindTavily, SearchEngineKindBrave, SearchEngineKindLocalSearXNG}
	if len(engines) != len(want) {
		t.Fatalf("引擎数 = %d, want %d: %v", len(engines), len(want), engineNames(engines))
	}
	for i, k := range want {
		if engines[i].Name() != k {
			t.Errorf("engines[%d] = %q, want %q（config 顺序生效）", i, engines[i].Name(), k)
		}
	}
}

// TestBuildEngines_CustomOrderSkipsMissing 顺序里可缺省引擎：只配置部分引擎时
// 其余不参与扇出。
func TestBuildEngines_CustomOrderSkipsMissing(t *testing.T) {
	saveSearchGlobals(t)
	t.Setenv("TAVILY_API_KEY", "tk")
	SetSearchConfig(config.SearchConfig{TavilyAPIKeyEnv: "TAVILY_API_KEY"})
	SetSearchEngineOrder([]string{SearchEngineKindTavily})

	ws := webSearch{}
	engines := ws.buildEngines()
	if len(engines) != 1 || engines[0].Name() != SearchEngineKindTavily {
		t.Fatalf("只配置 tavily 时应只扇出 tavily: %v", engineNames(engines))
	}
}

func engineNames(engines []SearchEngine) []string {
	out := make([]string, 0, len(engines))
	for _, e := range engines {
		out = append(out, e.Name())
	}
	return out
}
