package app

// websearch_settings.go — 搜索主源设置面（众神殿 2026-11-05 网络搜索刀3：
// 主源向导）。设置面板搜索卡的后端三绑定：读现值/可用性、保存主源（key→
// ~/.env + [search] 持久化 + builtin 热重载）、主源连通性实测。
//
// 密钥落点纪律：~/.env 是 gaea 的合法密钥源（internal/gaea/config/dotenv.go
// boot 时 loadDotEnv 注入进程环境，桌面端 cwd=$HOME 与 CLI 共用同源；已存在
// 的环境变量恒优先）。key 不进 config.toml——[search] 只存 env 变量名，与
// tavily_api_key_env 既有 schema 一致。
//
// 热重载：对齐 boot/sysprompt.go 的装配三件（SetSearchConfig/SetSearchProxy/
// SetSearchEngineOrder），保存后立即生效，无需重启；proxy 与窗口长度不动。

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gaeacfg "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// webSearchConfigLoad / webSearchConfigSave / webSearchEnvUpsert / webSearchCoreRun
// 测试缝：设置面读写、~/.env upsert 与统一检索核实测的注入点（测试不触真实
// 用户配置/家目录/外网）。
var (
	webSearchConfigLoad = func() *gaeacfg.Config { return gaeacfg.LoadForEdit(gaeacfg.UserConfigPath()) }
	webSearchConfigSave = gaeacfg.Save
	webSearchEnvUpsert  = upsertHomeEnvKey
	webSearchCoreRun    = builtin.RunSearch
)

// WebSearchSettingsGet 搜索主源设置读取：[search] 三键现值（env 变量名非密钥
// 本体）+ 主源可用性解析 + 引擎序。
func (a *App) WebSearchSettingsGet() (map[string]interface{}, error) {
	cfg := webSearchConfigLoad()
	if cfg == nil {
		return nil, fmt.Errorf("读取搜索设置失败：办公配置不可用")
	}
	s := cfg.Search
	return map[string]interface{}{
		"local_searxng_url":  s.LocalSearXNGURL,
		"tavily_api_key_env": s.TavilyAPIKeyEnv,
		"brave_api_key_env":  s.BraveAPIKeyEnv,
		"tavily_ready":       s.TavilyKey() != "",
		"brave_ready":        s.BraveKey() != "",
		"searxng_ready":      s.LocalSearXNGURL != "",
		"has_primary":        s.TavilyKey() != "" || s.BraveKey() != "" || s.LocalSearXNGURL != "",
		"engine_order":       append([]string(nil), s.EngineOrder...),
	}, nil
}

// WebSearchSettingsSet 保存搜索主源设置：key 落 ~/.env（合法密钥源+立即
// Setenv 热生效）+ [search] 键名/URL 持久化 + builtin 全局热重载。
// 空串=清除该源（移除 ~/.env 行 + Unsetenv + 键名置空）。
func (a *App) WebSearchSettingsSet(tavilyKey, braveKey, searxngURL string) error {
	tavilyKey = strings.TrimSpace(tavilyKey)
	braveKey = strings.TrimSpace(braveKey)
	searxngURL = strings.TrimSpace(searxngURL)

	if err := webSearchEnvUpsert("TAVILY_API_KEY", tavilyKey); err != nil {
		return fmt.Errorf("写入 ~/.env 失败（TAVILY_API_KEY）: %w", err)
	}
	if err := webSearchEnvUpsert("BRAVE_API_KEY", braveKey); err != nil {
		return fmt.Errorf("写入 ~/.env 失败（BRAVE_API_KEY）: %w", err)
	}

	cfg := webSearchConfigLoad()
	if cfg == nil {
		return fmt.Errorf("保存搜索设置失败：办公配置不可用")
	}
	if tavilyKey != "" {
		cfg.Search.TavilyAPIKeyEnv = "TAVILY_API_KEY"
	} else {
		cfg.Search.TavilyAPIKeyEnv = ""
	}
	if braveKey != "" {
		cfg.Search.BraveAPIKeyEnv = "BRAVE_API_KEY"
	} else {
		cfg.Search.BraveAPIKeyEnv = ""
	}
	cfg.Search.LocalSearXNGURL = searxngURL
	if err := webSearchConfigSave(cfg); err != nil {
		return fmt.Errorf("持久化 [search] 配置失败: %w", err)
	}

	// 热重载（对齐 boot/sysprompt.go 装配）：保存即生效。
	builtin.SetSearchConfig(cfg.Search)
	builtin.SetSearchEngineOrder(cfg.Search.EngineOrder)
	return nil
}

// WebSearchTestConnection 主源连通性实测（设置卡「测试连接」）：真实走统一
// 检索核出网一次，按状态+逐引擎诊断+样例回执（不落库、不注入对话）。
func (a *App) WebSearchTestConnection() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	start := time.Now()
	outcome, err := webSearchCoreRun(ctx, "gaea web search 连通性测试", 3)
	elapsed := time.Since(start).Milliseconds()

	engines := make([]map[string]interface{}, 0, len(outcome.Attempts))
	sources := make([]string, 0, 3)
	if outcome != nil {
		for _, at := range outcome.Attempts {
			engines = append(engines, map[string]interface{}{
				"name": at.Name, "outcome": at.Outcome, "elapsed_ms": at.ElapsedMS,
			})
		}
		for i, r := range outcome.Results {
			if i >= 3 {
				break
			}
			sources = append(sources, r.Title+"（"+r.Source+"）")
		}
	}
	state := ""
	degraded := ""
	if outcome != nil {
		state = string(outcome.State)
		degraded = outcome.DegradedReason
	}
	// ok 语义=拿到可用结果（OK/DEGRADED 都算连通；DEGRADED 的降级原因由
	// state/degraded_reason 字段如实携带）；只有全败/空才给配置指引。
	ok := state == string(builtin.SearchStateOK) || state == string(builtin.SearchStateDegraded)
	hint := ""
	if !ok {
		hint = "在上方填入 Tavily（tavily.com 免费注册）/ Brave key 或自建 SearXNG 地址后保存重试；免费兜底（Bing）仍可用。"
	}
	return map[string]interface{}{
		"ok":              ok,
		"state":           state,
		"degraded_reason": degraded,
		"elapsed_ms":      elapsed,
		"engines":         engines,
		"sample":          sources,
		"error":           errorText(err),
		"hint":            hint,
	}, nil
}

// errorText err → 字符串（nil → 空串），供绑定回执。
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// upsertHomeEnvKey ~/.env 单键 upsert（boot loadDotEnv 同源的合法密钥落点）。
// value 非空：替换既有 KEY=/export KEY= 行（或追加），并 os.Setenv 立即生效；
// value 空：移除该行并 os.Unsetenv。其余行原样保留，文件权限 0600。
func upsertHomeEnvKey(key, value string) error {
	if key == "" {
		return fmt.Errorf("env key 不能为空")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".env")
	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
	}
	out := make([]string, 0, len(lines)+1)
	prefix := key + "="
	exportPrefix := "export " + prefix
	found := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) || strings.HasPrefix(trimmed, exportPrefix) {
			found = true
			if value != "" {
				out = append(out, key+"="+value)
			}
			continue
		}
		out = append(out, line)
	}
	if !found && value != "" {
		out = append(out, key+"="+value)
	}
	content := strings.Join(out, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return err
	}
	if value != "" {
		// 已存在的环境变量优先是 loadDotEnv 的纪律；这里显式写入即为用户的
		// 最新拍板，Setenv 覆盖保证本次保存立即生效（重启后 ~/.env 同值）。
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
	return nil
}
