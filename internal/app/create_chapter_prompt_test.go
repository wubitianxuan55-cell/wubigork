package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/worldview"
)

// TestCreateChapterPromptInjectsSetting 回归测试：生成章节时发给 AI 的提示词必须注入小说设定。
func TestCreateChapterPromptInjectsSetting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	// 本地 OpenAI 兼容 mock：捕获请求体并返回一个极短的流
	gotRequests := make(chan []byte, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotRequests <- body
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"第一章正文"},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`)
	}))
	defer srv.Close()

	cfg := &config.Config{}
	cfg.FuncNovelEngine = "herdsman"
	cfg.FuncNovelModel = "test-model"
	cfg.FuncNovelEnabled = true
	cfg.ActiveEngineID = "herdsman"

	engMgr := modelengine.NewManager("", "")
	if err := engMgr.SaveEngine(modelengine.EngineConfig{
		ID: "herdsman", Name: "Herdsman", Type: modelengine.EngineHerdsman,
		BaseURL: srv.URL, Enabled: true, DefaultModel: "test-model",
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}

	client := ai.NewClient(cfg)
	client.SetEngineManager(engMgr)

	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := project.Create(dir, "测试小说", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	a := &App{core: &core{cfg: cfg, client: client, engineMgr: engMgr}}
	a.writingState = &writingState{core: a.core, app: a, eng: prompt.NewEngine("../../prompts"), mu: sync.RWMutex{}}
	a.setPM(pm)
	a.worldviewAgent = worldview.New(client, pm, cfg, a.eng)

	// 1. 保存小说设定
	settingText := "【主角】林晚，剑修，性格清冷。世界背景：九州大陆，灵气复苏。"
	if err := a.SaveWorldview(settingText); err != nil {
		t.Fatalf("保存设定: %v", err)
	}

	// 2. 前端读取设定
	got := a.GetWorldview()
	if got == "" {
		t.Fatalf("GetWorldview 返回空，设定未保存成功")
	}

	// 3. 生成章节，捕获实际发送给 AI 的请求
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	a.ctx = ctx

	if _, err := a.CreateChapter(got, "", "主角踏上旅途", 1, "", "", 0, 0); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}

	select {
	case body := <-gotRequests:
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("解析请求体失败: %v", err)
		}
		var userPrompt string
		for _, m := range req.Messages {
			if m.Role == "user" {
				userPrompt = m.Content
			}
		}
		if userPrompt == "" {
			t.Fatalf("请求中缺少 user 消息")
		}
		if !strings.Contains(userPrompt, "小说设定") {
			t.Errorf("user prompt 缺少「小说设定」小节:\n%s", userPrompt)
		}
		if !strings.Contains(userPrompt, "林晚") {
			t.Errorf("user prompt 未包含设定内容:\n%s", userPrompt)
		}
	case <-ctx.Done():
		t.Fatalf("未捕获到 AI 请求: %v", ctx.Err())
	}
}
