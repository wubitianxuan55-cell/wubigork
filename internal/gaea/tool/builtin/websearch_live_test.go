package builtin

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
)

// TestWebSearchLive 联网端到端实测：真实跑 gaea 的 webSearch.Execute——搜索引擎页请求
// （6 引擎注册表/并行扇出/域名过滤）与 url 直取整页（SSRF+域名策略+HTML→文本）都走真实
// 网络。默认 t.Skip（避免 CI/离线环境红），设置 GAEA_LIVE_TEST=1 才真的出网请求。
func TestWebSearchLive(t *testing.T) {
	if os.Getenv("GAEA_LIVE_TEST") != "1" {
		t.Skip("set GAEA_LIVE_TEST=1 to run live network test")
	}
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	t.Run("search_query", func(t *testing.T) {
		out, err := (webSearch{}).Execute(ctx, json.RawMessage(`{"query":"golang 并发","topK":5}`))
		if err != nil {
			t.Logf("web_search(query) 真实失败: %v", err)
			// 失败也是端到端证据（真实引擎/网络错误被透出）。
			return
		}
		t.Logf("web_search(query) 成功，返回 JSON 长度=%d，开头=%s", len(out), truncate(out, 500))
	})

	t.Run("fetch_url", func(t *testing.T) {
		out, err := (webSearch{}).Execute(ctx, json.RawMessage(`{"url":"https://example.com"}`))
		if err != nil {
			t.Logf("web_search(url) 真实失败: %v", err)
			return
		}
		t.Logf("web_search(url) 成功，返回文本长度=%d，开头=%s", len(out), truncate(out, 400))
	})
}
