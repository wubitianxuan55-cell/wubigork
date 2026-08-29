package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

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
