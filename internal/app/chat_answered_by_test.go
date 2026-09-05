package app

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/httpbridge"
)

// ── v4.15 聊天路由归位 + answered_by 回显 ──────────────────────────

// TestPlainChatOfflineFilter 聊天路由归位回归：plain 聊天主链路改走 routeModel
// 后，全局离线过滤对聊天生效——离线开启 + 聊天功能绑定云端引擎时：
//   - 有本地引擎可用 → 路由本地（不回云端）；
//   - 仅剩云端引擎 → 路由为空（调用方按「模型不可用」降级，不再误发云端）。
func TestPlainChatOfflineFilter(t *testing.T) {
	c := newRouterTestCore(t)
	if err := c.SetFeatureModel("chat", "xai", "grok-4.20"); err != nil {
		t.Fatal(err)
	}
	c.cfg.OfflineMode = true

	// 停用 ollama/cosyvoice；herdsman 注册序在 modelhub 之前，仍为本地兜底首选
	// （xai 云端被滤）。
	for _, id := range []string{"ollama", "cosyvoice"} {
		if e, ok := c.engineMgr.GetEngine(id); ok {
			e.Enabled = false
			if err := c.engineMgr.SaveEngine(*e); err != nil {
				t.Fatal(err)
			}
		}
	}
	eng, _, source := c.routeModel("chat")
	if eng != "herdsman" || source != "fallback" {
		t.Fatalf("离线 + 绑定云端 + 本地可用应路由本地，实际 (%q,%q)", eng, source)
	}

	// 停用全部本地引擎（仅剩云端 xai）→ 路由为空。
	for _, id := range []string{"herdsman", "modelhub"} {
		if e, ok := c.engineMgr.GetEngine(id); ok {
			e.Enabled = false
			if err := c.engineMgr.SaveEngine(*e); err != nil {
				t.Fatal(err)
			}
		}
	}
	eng, model, source := c.routeModel("chat")
	if eng != "" || model != "" || source != "" {
		t.Fatalf("离线 + 仅云端应返回空（不误发云端），实际 (%q,%q,%q)", eng, model, source)
	}
}

// TestChatDoneFrameAnsweredBy 流式 done 帧含 answered_by 四字段：
// engine/model/source 来自 routeModel（此处 feature 绑定 herdsman/qwen3-8b），
// 本地引擎 cost_cny=0（诚实，usage 缺失/本地不计价均不虚报）。
func TestChatDoneFrameAnsweredBy(t *testing.T) {
	a := newChatServiceTestApp(t)
	topic, err := a.ChatTopicCreate("回显", "plain")
	if err != nil {
		t.Fatalf("ChatTopicCreate: %v", err)
	}

	orig := newChatStreamRunID
	newChatStreamRunID = func() string { return "cs_answered_by" }
	t.Cleanup(func() { newChatStreamRunID = orig })

	// 通过 httpbridge SSE 订阅固定事件名（订阅先行，避免与异步 emit 竞态漏帧）。
	srv := httptest.NewServer(httpbridge.New(a).Handler())
	defer srv.Close()
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/stream?id=chat-stream:cs_answered_by", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("打开 SSE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE status = %d", resp.StatusCode)
	}

	lines := make(chan string, 64)
	go func() {
		br := bufio.NewReader(resp.Body)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				close(lines)
				return
			}
			lines <- line
		}
	}()
	// 消费连接帧（event/data/blank 三行）。
	for i := 0; i < 3; i++ {
		select {
		case <-lines:
		case <-time.After(3 * time.Second):
			t.Fatal("等待 SSE 连接帧超时")
		}
	}

	if _, err := a.ChatStreamPlain(topic.ID, "你好", false, false, false); err != nil {
		t.Fatalf("ChatStreamPlain: %v", err)
	}

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("流在收到 done 终态前关闭")
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev map[string]interface{}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
				t.Fatalf("解析事件: %v", err)
			}
			if ev["type"] != "done" {
				continue // delta / reasoning 帧继续等待
			}
			ab, ok := ev["answered_by"].(map[string]interface{})
			if !ok {
				t.Fatal("done 帧缺少 answered_by")
			}
			for _, k := range []string{"engine", "model", "source", "cost_cny"} {
				if _, ok := ab[k]; !ok {
					t.Errorf("answered_by 缺少字段 %q", k)
				}
			}
			if ab["engine"] != "herdsman" || ab["model"] != "qwen3-8b" || ab["source"] != "feature" {
				t.Errorf("answered_by = %v, want engine=herdsman model=qwen3-8b source=feature", ab)
			}
			if c, ok := ab["cost_cny"].(float64); !ok || c != 0 {
				t.Errorf("本地引擎 cost_cny 应为 0，实际 %v", ab["cost_cny"])
			}
			return
		case <-time.After(3 * time.Second):
			t.Fatal("未在 3s 内收到 done 帧")
		}
	}
}
