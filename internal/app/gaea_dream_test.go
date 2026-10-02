package app

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// captureGaeaNotices 注入 notice 发射缝，返回已捕获的 notice 文案读取函数。
// 只收 gaea-event 且 kind=notice 的载荷（后台 panic 的可见通道）。
func captureGaeaNotices(t *testing.T) func() []string {
	t.Helper()
	orig := gaeaNoticeSink
	t.Cleanup(func() { gaeaNoticeSink = orig })
	var mu sync.Mutex
	var texts []string
	gaeaNoticeSink = func(name string, data map[string]interface{}) {
		if name != "gaea-event" {
			return
		}
		if kind, _ := data["kind"].(string); kind != "notice" {
			return
		}
		mu.Lock()
		texts = append(texts, fmt.Sprint(data["text"]))
		mu.Unlock()
	}
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), texts...)
	}
}

// TestMaybeDreamAfterTurn_PanicResetsStateAndNotifies 审计 P2 AP5-16：后台整理
// panic 后必须（1）复位单飞状态位 running（否则自动做梦永久停摆）、（2）发一条
// 前端可见的 notice（此前只 slog.Error 就吞掉，用户侧「功能开着但再也不工作」）。
func TestMaybeDreamAfterTurn_PanicResetsStateAndNotifies(t *testing.T) {
	oldCfg := ga.cfg
	ga.cfg = &gaeaConfig.Config{
		Dream:  gaeaConfig.DreamConfig{Mode: "suggest"},
		Memory: gaeaConfig.MemoryConfig{Enabled: true},
	}
	t.Cleanup(func() { ga.cfg = oldCfg })

	origRun := gaeaDreamRun
	t.Cleanup(func() { gaeaDreamRun = origRun })
	gaeaDreamRun = func(*App, string) error { panic("dream boom") }

	notices := captureGaeaNotices(t)

	a := &App{core: &core{}}
	a.maybeDreamAfterTurn("work")

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		gaeaDreamState.Lock()
		running := gaeaDreamState.running
		gaeaDreamState.Unlock()
		if !running && len(notices()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	gaeaDreamState.Lock()
	running := gaeaDreamState.running
	gaeaDreamState.Unlock()
	if running {
		t.Fatal("panic 后单飞位 running 未复位——自动做梦将永久停摆")
	}
	got := notices()
	if len(got) == 0 {
		t.Fatal("panic 后未发前端可见 notice（AP5-16 要求可见化）")
	}
	if !strings.Contains(got[0], "自动做梦") || !strings.Contains(got[0], "dream boom") {
		t.Fatalf("notice 文案应含任务名与 panic 值: %q", got[0])
	}
}

// TestGaeaBackgroundPanicNotice_DefaultsToGaeaEvent 不注入缝时走真实 emit 通道，
// 且不 panic（生产路径冒烟：core.ctx 为 nil 时 emit 内部自守）。
func TestGaeaBackgroundPanicNotice_DefaultsToGaeaEvent(t *testing.T) {
	c := &core{}
	c.gaeaBackgroundPanicNotice("测试任务", "boom", map[string]interface{}{"k": "v"})
}

func TestParseDreamOutput(t *testing.T) {
	cases := []struct {
		name string
		in   string
		fact int
		note int
		err  bool
	}{
		{
			name: "fenced JSON",
			in:   "```json\n{\"facts\":[{\"name\":\"user-unit\",\"type\":\"user\",\"description\":\"单位\",\"body\":\"XX 公司\"}],\"notes\":[{\"scope\":\"local\",\"note\":\"口径\"}]}\n```",
			fact: 1,
			note: 1,
		},
		{
			name: "plain JSON with prefix text",
			in:   "好的：{\"facts\":[],\"notes\":[]}",
			fact: 0,
			note: 0,
		},
		{
			name: "empty name skipped",
			in:   "{\"facts\":[{\"name\":\"\",\"description\":\"x\"},{\"name\":\"ok\",\"description\":\"y\"}]}",
			fact: 1,
		},
		{
			name: "bad JSON",
			in:   "不是 JSON",
			err:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDreamOutput(tc.in)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Facts) != tc.fact || len(got.Notes) != tc.note {
				t.Fatalf("facts=%d notes=%d, want %d/%d", len(got.Facts), len(got.Notes), tc.fact, tc.note)
			}
		})
	}
}

func TestDreamTurnMessages(t *testing.T) {
	hist := []provider.Message{
		{Role: provider.RoleUser, Content: "旧问题"},
		{Role: provider.RoleAssistant, Content: "旧回答"},
		{Role: provider.RoleTool, Content: "tool"},
		{Role: provider.RoleUser, Content: "新问题"},
		{Role: provider.RoleAssistant, Content: "新回答"},
	}
	got := dreamTurnMessages(hist)
	if len(got) != 2 || got[0].Content != "新问题" || got[1].Content != "新回答" {
		t.Fatalf("dreamTurnMessages = %+v, want last user+assistant", got)
	}
	if dreamTurnMessages(nil) != nil {
		t.Fatal("nil history should return nil")
	}
}

func TestDreamWorthwhile(t *testing.T) {
	short := []provider.Message{
		{Role: provider.RoleUser, Content: "你好"},
		{Role: provider.RoleAssistant, Content: "你好！有什么可以帮你？"},
	}
	if dreamWorthwhile(short) {
		t.Fatal("greeting should not trigger dream")
	}
	long := []provider.Message{
		{Role: provider.RoleUser, Content: "帮我做成本测算"},
		{Role: provider.RoleAssistant, Content: strings.Repeat("已完成成本测算，总额 120 万。", 20)},
	}
	if !dreamWorthwhile(long) {
		t.Fatal("substantive turn should trigger dream")
	}
	if dreamWorthwhile([]provider.Message{{Role: provider.RoleUser, Content: "x"}}) {
		t.Fatal("single message should not trigger dream")
	}
}

func TestDreamInputTruncatesLongMessages(t *testing.T) {
	in := dreamInput([]provider.Message{
		{Role: provider.RoleUser, Content: strings.Repeat("长", 5000)},
	})
	if !strings.Contains(in, "…") {
		t.Fatal("dream input should truncate long content")
	}
}

// C3 no-op：相同内容的整理输入指纹一致（跳过重复 LLM 提炼），内容变化则指纹
// 变化；指纹记录仅在完整处理成功后写入（由 runDream 收尾保证，此处测纯函数）。
// S1.2 A：指纹键含会话空间，此处 space=""（mode=off 纯内容形态）验证旧行为。
func TestDreamInputHashNoop(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "帮我整理这份成本测算表"},
		{Role: provider.RoleAssistant, Content: "已完成，公式与汇总如下……（超过 100 字的实质内容省略）"},
	}
	h1 := dreamInputHash("", dreamInput(msgs))
	h2 := dreamInputHash("", dreamInput(msgs))
	if h1 == "" || h1 != h2 {
		t.Fatalf("相同输入指纹应一致: %q vs %q", h1, h2)
	}
	msgs2 := append(append([]provider.Message{}, msgs...), provider.Message{Role: provider.RoleUser, Content: "再补一列环比"})
	if h3 := dreamInputHash("", dreamInput(msgs2)); h3 == h1 {
		t.Fatal("内容变化后指纹应变化")
	}
}
