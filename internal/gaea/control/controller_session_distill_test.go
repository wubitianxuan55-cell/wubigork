package control

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/permission"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// T10/T11/T7 会话边界审计：/new 必须让 Session 作用域授权真正失效，同时
// 新 transcript 的章节快照要从空开始；否则旧 argv/精确授权和旧 facts 快照
// 会泄漏进新会话。
func TestNewSessionClearsSessionScopedDistillationState(t *testing.T) {
	exec := agent.New(nil, nil, agent.NewSession("system"), agent.Options{}, event.Discard)
	c := New(Options{Executor: exec, Sink: event.Discard, SectionDiff: true})
	g := c.newGate()

	c.mu.Lock()
	c.granted["bash\x00git status"] = true
	c.grantedRules = append(c.grantedRules, grantedRule{tool: "bash", subject: "git status*"})
	c.argvRules.Add(permission.ArgvRule{First: "git", Rest: []string{"status"}})
	c.deniedActions = append(c.deniedActions, deniedAction{id: "d1", tool: "bash", subject: "git status"})
	c.lastUserInput = "上一会话授权"
	c.mu.Unlock()
	c.guardianBreaker.RecordDenial()
	c.sessionFacts = []memory.Memory{mkFact("pref", "简洁")}
	if out := c.Compose("首轮"); !strings.Contains(out, "<session-facts>") {
		t.Fatalf("首轮应写入章节快照: %q", out)
	}

	if err := c.NewSession(); err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.granted) != 0 || len(c.grantedRules) != 0 || len(c.deniedActions) != 0 {
		t.Fatalf("会话授权/追认账本未清空: granted=%v rules=%v denied=%v",
			c.granted, c.grantedRules, c.deniedActions)
	}
	if c.argvRules.Len() != 0 || g.SessionArgv.Len() != 0 {
		t.Fatalf("旧 Gate 仍持有 argv 授权: controller=%d gate=%d", c.argvRules.Len(), g.SessionArgv.Len())
	}
	if c.lastUserInput != "" || c.guardianBreaker.Consecutive() != 0 {
		t.Fatalf("Guardian 会话态未清空: input=%q denials=%d", c.lastUserInput, c.guardianBreaker.Consecutive())
	}
	if _, ok := c.sectionStore.SnapshotOf(sessionFactsSectionID); ok {
		t.Fatal("新会话仍保留旧章节快照")
	}
}

// T7 恢复边界：恢复 transcript 已带上一轮注入，而内存无快照时必须按 Unknown
// 发替换通知；不能把当前 facts 当「首次出现」再叠一份。
func TestResumeMarksSessionFactsUnknown(t *testing.T) {
	s := agent.NewSession("system")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "上一轮"})
	c := New(Options{Sink: event.Discard, SectionDiff: true})
	c.Resume(s, "resumed.jsonl")
	c.sessionFacts = []memory.Memory{mkFact("pref", "简洁")}
	out := c.Compose("继续")
	if !strings.Contains(out, "替换此前给出的同类指引") {
		t.Fatalf("恢复首轮应发替换通知: %q", out)
	}
	if !strings.Contains(out, "<session-facts>") {
		t.Fatalf("替换通知后应携带当前 facts: %q", out)
	}
}
