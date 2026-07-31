package context

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestBudget(t *testing.T) {
	b := NewBudget(128000)

	b.Track("系统提示", 12000)
	b.Track("当前场景", 18000)

	if b.Used != 30000 {
		t.Fatalf("expected used 30000, got %d", b.Used)
	}
	if b.Remaining() != 98000 {
		t.Fatalf("expected remaining 98000, got %d", b.Remaining())
	}

	pct := b.UsagePercent()
	if pct < 20 || pct > 25 {
		t.Fatalf("unexpected usage percent: %.1f%%", pct)
	}
}

func TestBudget_Sections(t *testing.T) {
	b := NewBudget(100000)
	if len(b.Sections) != 7 {
		t.Fatalf("expected 7 sections, got %d", len(b.Sections))
	}
	if b.Sections[0].Name != "系统提示" {
		t.Fatalf("expected first section 系统提示, got %s", b.Sections[0].Name)
	}
}

func TestFindTriggers(t *testing.T) {
	engine := &Engine{
		rules: []InjectionRule{
			{Priority: 1, Entry: types.LorebookEntry{Key: "青云宗", Content: "青云宗是苍山上的修炼门派", Category: "location"}},
			{Priority: 2, Entry: types.LorebookEntry{Key: "Elara", Content: "Elara是青云宗弟子", Category: "character"}},
			{Priority: 3, Entry: types.LorebookEntry{Key: "灵石", Content: "灵石是修炼资源", Category: "item"}},
		},
	}

	text := "Elara站在青云宗的大殿中"
	triggered := engine.FindTriggers(text, 1000)

	if len(triggered) != 2 {
		t.Fatalf("expected 2 triggered rules, got %d", len(triggered))
	}
	if triggered[0].Entry.Key != "青云宗" {
		t.Fatalf("expected 青云宗 first (priority 1), got %s", triggered[0].Entry.Key)
	}
}

func TestFindTriggers_NoMatch(t *testing.T) {
	engine := &Engine{
		rules: []InjectionRule{
			{Priority: 1, Entry: types.LorebookEntry{Key: "青云宗", Content: "test", Category: "location"}},
		},
	}

	triggered := engine.FindTriggers("无关文本", 1000)
	if len(triggered) != 0 {
		t.Fatalf("expected 0 triggered, got %d", len(triggered))
	}
}

func TestInject(t *testing.T) {
	engine := &Engine{
		rules: []InjectionRule{
			{Priority: 1, Entry: types.LorebookEntry{Key: "青云宗", Content: "青云宗设定", Category: "location"}},
		},
	}

	systemPrompt := "你是一个小说作家"
	userText := "在青云宗的大殿中"

	result, triggered := engine.Inject(systemPrompt, userText, 1000, nil)
	if len(triggered) != 1 {
		t.Fatalf("expected 1 triggered, got %d", len(triggered))
	}
	if !strings.Contains(result, "青云宗设定") {
		t.Fatalf("injected content should contain lorebook entry, got: %s", result)
	}
}
