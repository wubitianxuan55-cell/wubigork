package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Target DevTools /json/list 里的一项目标（只取本包关心的字段）。
type Target struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
	// WSURL webSocketDebuggerUrl：页面级 CDP 会话拨号地址。
	WSURL string `json:"webSocketDebuggerUrl"`
}

// devtoolsHTTP DevTools HTTP 端点客户端（短超时，探活/列举/建页都是快请求）。
var devtoolsHTTP = &http.Client{Timeout: 5 * time.Second}

// probeInterval 探活轮询间隔。
const probeInterval = 100 * time.Millisecond

// waitDevtools 轮询 /json/version 直到 DevTools 端口就绪或超时
// （dead-start 上限由调用方给，默认 ~20s）。
func waitDevtools(ctx context.Context, httpBase string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		var v map[string]any
		if err := devtoolsGet(ctx, http.MethodGet, httpBase+"/json/version", &v); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待 DevTools 端口就绪超时（%s，>%v）", httpBase, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(probeInterval):
		}
	}
}

// devtoolsGet 请求 DevTools HTTP 端点并把 JSON body 解进 out。
func devtoolsGet(ctx context.Context, method, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, path, nil)
	if err != nil {
		return err
	}
	resp, err := devtoolsHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("devtools %s %s: HTTP %d", method, path, resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

// listTargets 拉取 /json/list 全部目标。
func listTargets(ctx context.Context, httpBase string) ([]Target, error) {
	var targets []Target
	if err := devtoolsGet(ctx, http.MethodGet, httpBase+"/json/list", &targets); err != nil {
		return nil, err
	}
	return targets, nil
}

// firstPageTarget 返回第一个 type=="page" 的目标；没有则 (nil, nil)。
func firstPageTarget(ctx context.Context, httpBase string) (*Target, error) {
	targets, err := listTargets(ctx, httpBase)
	if err != nil {
		return nil, fmt.Errorf("browser: 列举目标失败: %w", err)
	}
	for i := range targets {
		if targets[i].Type == "page" {
			return &targets[i], nil
		}
	}
	return nil, nil
}

// newPageTarget 建一个新页面目标（Edge 新版要求 PUT；url 作为 query 传递）。
func newPageTarget(ctx context.Context, httpBase, pageURL string) (*Target, error) {
	var t Target
	path := httpBase + "/json/new?" + url.Values{"url": []string{pageURL}}.Encode()
	if err := devtoolsGet(ctx, http.MethodPut, path, &t); err != nil {
		return nil, fmt.Errorf("browser: 新建页面失败: %w", err)
	}
	return &t, nil
}

// closePageTarget 关闭指定页面目标（best effort，/json/close 无 JSON 响应体）。
func closePageTarget(ctx context.Context, httpBase, targetID string) error {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return nil
	}
	return devtoolsGet(ctx, http.MethodGet, httpBase+"/json/close/"+url.PathEscape(targetID), nil)
}
