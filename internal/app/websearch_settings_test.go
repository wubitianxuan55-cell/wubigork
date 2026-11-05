package app

// websearch_settings_test.go — 搜索主源设置面测试（众神殿刀3）：全部走测试缝，
// 不触真实用户配置/~/.env/外网。验证读现值可用性解析、保存持久化+env upsert、
// 清空=清除语义、连通性实测回执。

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	gaeacfg "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// stubWebSearchSettingsSeams 替换设置面四缝并在测试结束恢复。
func stubWebSearchSettingsSeams(t *testing.T, cfg *gaeacfg.Config, run func(ctx context.Context, query string, topK int) (*builtin.SearchOutcome, error)) {
	t.Helper()
	origLoad, origSave, origEnv, origRun := webSearchConfigLoad, webSearchConfigSave, webSearchEnvUpsert, webSearchCoreRun
	webSearchConfigLoad = func() *gaeacfg.Config { return cfg }
	webSearchEnvUpsert = func(key, value string) error { return nil }
	if run != nil {
		webSearchCoreRun = run
	}
	t.Cleanup(func() {
		webSearchConfigLoad, webSearchConfigSave, webSearchEnvUpsert, webSearchCoreRun = origLoad, origSave, origEnv, origRun
	})
	return
}

// TestWebSearchSettingsGet_Availability 可用性按解析后的 key（env 变量名→值）
// 归出，非 env 变量名本身。
func TestWebSearchSettingsGet_Availability(t *testing.T) {
	cfg := &gaeacfg.Config{}
	cfg.Search.TavilyAPIKeyEnv = "TAVILY_API_KEY"
	cfg.Search.LocalSearXNGURL = "http://localhost:8080"
	t.Setenv("TAVILY_API_KEY", "tvly-test-key") // env 已设 → tavily_ready
	stubWebSearchSettingsSeams(t, cfg, nil)

	out, err := (&App{}).WebSearchSettingsGet()
	if err != nil {
		t.Fatalf("WebSearchSettingsGet: %v", err)
	}
	if out["tavily_ready"] != true || out["searxng_ready"] != true || out["brave_ready"] != false {
		t.Errorf("可用性解析异常: %v", out)
	}
	if out["has_primary"] != true {
		t.Errorf("has_primary 应为 true: %v", out)
	}
	if out["tavily_api_key_env"] != "TAVILY_API_KEY" {
		t.Errorf("回执应为 env 变量名而非密钥本体: %v", out["tavily_api_key_env"])
	}
}

// TestWebSearchSettingsSet_SaveAndEnv 保存：key 落 env 缝 + [search] 键名/URL
// 持久化 + 空串=清除语义。
func TestWebSearchSettingsSet_SaveAndEnv(t *testing.T) {
	cfg := &gaeacfg.Config{}
	saved := false
	var savedCfg *gaeacfg.Config
	envUpserts := map[string]string{}
	stubWebSearchSettingsSeams(t, cfg, nil)
	webSearchConfigSave = func(c *gaeacfg.Config) error { saved = true; savedCfg = c; return nil }
	webSearchEnvUpsert = func(key, value string) error { envUpserts[key] = value; return nil }

	a := &App{}
	if err := a.WebSearchSettingsSet("tvly-new-key", "", "http://localhost:8888"); err != nil {
		t.Fatalf("WebSearchSettingsSet: %v", err)
	}
	if !saved {
		t.Fatal("office 配置应已持久化")
	}
	if savedCfg.Search.TavilyAPIKeyEnv != "TAVILY_API_KEY" {
		t.Errorf("tavily key env 应在册: %q", savedCfg.Search.TavilyAPIKeyEnv)
	}
	if savedCfg.Search.BraveAPIKeyEnv != "" {
		t.Errorf("空 brave key 应清除键名: %q", savedCfg.Search.BraveAPIKeyEnv)
	}
	if savedCfg.Search.LocalSearXNGURL != "http://localhost:8888" {
		t.Errorf("SearXNG URL 应持久化: %q", savedCfg.Search.LocalSearXNGURL)
	}
	if envUpserts["TAVILY_API_KEY"] != "tvly-new-key" || envUpserts["BRAVE_API_KEY"] != "" {
		t.Errorf("env upsert 入参异常: %v", envUpserts)
	}
}

// TestWebSearchTestConnection_Outcomes 连通性实测：ok / degraded / 全败三形态回执。
func TestWebSearchTestConnection_Outcomes(t *testing.T) {
	cfg := &gaeacfg.Config{}
	stubWebSearchSettingsSeams(t, cfg, func(_ context.Context, _ string, _ int) (*builtin.SearchOutcome, error) {
		return &builtin.SearchOutcome{
			State:   builtin.SearchStateOK,
			Results: []builtin.SearchResult{{Title: "正题", URL: "https://x", Source: "tavily"}},
			Attempts: []builtin.SearchAttempt{
				{Name: "tavily", Outcome: "ok", ElapsedMS: 1100},
				{Name: "bing", Outcome: "error", Err: "timeout"},
			},
		}, nil
	})
	out, err := (&App{}).WebSearchTestConnection()
	if err != nil || out["ok"] != true || out["state"] != "ok" {
		t.Fatalf("ok 形态回执异常: out=%v err=%v", out, err)
	}
	if len(out["engines"].([]map[string]interface{})) != 2 || len(out["sample"].([]string)) != 1 {
		t.Errorf("engines/sample 回执异常: %v", out)
	}

	stubWebSearchSettingsSeams(t, cfg, func(_ context.Context, _ string, _ int) (*builtin.SearchOutcome, error) {
		return &builtin.SearchOutcome{
			State:          builtin.SearchStateDegraded,
			DegradedReason: "primary-timeout",
			Results:        []builtin.SearchResult{{Title: "兜底", Source: "bing"}},
		}, errors.New("主源缺位")
	})
	out, _ = (&App{}).WebSearchTestConnection()
	if out["ok"] != true || out["state"] != "degraded" || out["degraded_reason"] != "primary-timeout" {
		t.Errorf("degraded 形态应如实回执（ok 语义=有可用结果）: %v", out)
	}

	stubWebSearchSettingsSeams(t, cfg, func(_ context.Context, _ string, _ int) (*builtin.SearchOutcome, error) {
		return &builtin.SearchOutcome{State: builtin.SearchStateAllFailed}, errors.New("所有搜索引擎失败")
	})
	out, _ = (&App{}).WebSearchTestConnection()
	if out["ok"] != false || !strings.Contains(out["hint"].(string), "Tavily") {
		t.Errorf("全败应附配置指引: %v", out)
	}
}

// TestUpsertHomeEnvKey ~/.env 单键 upsert（临时目录）：替换既有行、追加新键、
// 空值删行、其余行原样保留。
func TestUpsertHomeEnvKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir) // os.UserHomeDir Windows 口径
	t.Setenv("HOME", dir)        // Git Bash 口径兜底
	path := dir + "/.env"
	_ = os.WriteFile(path, []byte("EXISTING=keep\nTAVILY_API_KEY=old-key\n# comment\n"), 0o600)

	if err := upsertHomeEnvKey("TAVILY_API_KEY", "new-key"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got := osReadFileHelper(t, path)
	for _, want := range []string{"EXISTING=keep", "TAVILY_API_KEY=new-key", "# comment"} {
		if !strings.Contains(got, want) {
			t.Errorf("应含 %q: %q", want, got)
		}
	}
	if strings.Contains(got, "old-key") {
		t.Errorf("旧值应被替换: %q", got)
	}

	if err := upsertHomeEnvKey("BRAVE_API_KEY", "brave-1"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if got = osReadFileHelper(t, path); !strings.Contains(got, "BRAVE_API_KEY=brave-1") {
		t.Errorf("新键应追加: %q", got)
	}

	if err := upsertHomeEnvKey("BRAVE_API_KEY", ""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got = osReadFileHelper(t, path); strings.Contains(got, "BRAVE_API_KEY") {
		t.Errorf("空值应删行: %q", got)
	}
}

// osReadFileHelper 读小文件全文（测试辅助）。
func osReadFileHelper(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s: %v", path, err)
	}
	return string(data)
}
