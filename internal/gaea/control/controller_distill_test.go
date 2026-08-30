package control

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func newDistillTestController(t *testing.T) *Controller {
	t.Helper()
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{
		CWD:     t.TempDir(),
		UserDir: t.TempDir(),
		DB:      nil,
	})
	return c
}

func TestDistillMergeArchivesOlderKeepsNewer(t *testing.T) {
	c := newDistillTestController(t)
	if _, err := c.SaveDreamFacts("", "test", []memory.Memory{
		// 注：存储后端自身把 name 归一成 kebab-case（同名异写在库内不会共存），
		// 所以这里用规则 B 的真实场景：异名 + 同 type+kind + 同描述。
		{Name: "deploy-checklist", Type: memory.TypeFeedback, Kind: memory.KindProcedural, Description: "发布前三步检查", Body: "构建/冒烟/回滚点"},
		{Name: "release-steps", Type: memory.TypeFeedback, Kind: memory.KindProcedural, Description: "发布前三步检查", Body: "重复沉淀"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// 执行器消费检测器的输出（keep/archive 方向以检测器为准）。
	cands := memory.DistillMergeCandidates(c.Memory().Store.List())
	if len(cands) != 1 {
		t.Fatalf("应检出 1 条同名异写候选, got %d: %+v", len(cands), cands)
	}
	msg, err := c.DistillMerge(cands[0].Keep, cands[0].Archive)
	if err != nil {
		t.Fatalf("DistillMerge: %v", err)
	}
	if !strings.HasPrefix(msg, "merged:") {
		t.Errorf("应返回 merged 前缀消息, got %q", msg)
	}

	list := c.Memory().Store.List()
	if len(list) != 1 {
		t.Fatalf("合并后应剩 1 条, got %d", len(list))
	}
	if list[0].Name != cands[0].Keep {
		t.Errorf("保留条应为 %q, got %q", cands[0].Keep, list[0].Name)
	}
	if _, ok := c.Memory().Store.Get(cands[0].Archive); ok {
		t.Errorf("归档条 %q 不应再出现在活动记忆", cands[0].Archive)
	}
}

func TestDistillMergeRejectsUnknownPair(t *testing.T) {
	c := newDistillTestController(t)
	if _, err := c.SaveDreamFacts("", "test", []memory.Memory{
		{Name: "fact-a", Type: memory.TypeUser, Kind: memory.KindSemantic, Description: "事实 A"},
		{Name: "fact-b", Type: memory.TypeUser, Kind: memory.KindSemantic, Description: "事实 B"},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := c.DistillMerge("fact-a", "fact-b"); err == nil ||
		!strings.Contains(err.Error(), "候选已过期或不存在") {
		t.Fatalf("不在候选集内的配对应拒绝, got %v", err)
	}
	if _, err := c.DistillMerge("fact-a", "fact-a"); err == nil ||
		!strings.Contains(err.Error(), "同一条记忆") {
		t.Fatalf("自身配对应拒绝, got %v", err)
	}
	if _, err := c.DistillMerge("", "fact-b"); err == nil {
		t.Fatal("空 keep 应拒绝")
	}
	// 拒绝路径不得破坏记忆。
	if got := len(c.Memory().Store.List()); got != 2 {
		t.Fatalf("拒绝后记忆不应变化, got %d 条", got)
	}
}
