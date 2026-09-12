package control

// 刀B（v4.246）并发收敛：多 goroutine 并发写记忆，各自走「锁内取参→锁外
// Load→代号提交」链；WaitGroup 全部返回后，快照必须包含全部写入——代号
// 守卫保证最后换入的快照携带此前所有写入（慢的旧 Load 被拒绝换入而非覆盖，
// 且被跳过的写必然已被更高一代的 Load 读到：Load 起点晚于该写）。

import (
	"fmt"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestMemoryRefreshConvergesUnderConcurrency(t *testing.T) {
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{
		CWD:     t.TempDir(),
		UserDir: t.TempDir(),
		DB:      nil,
	})

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				if _, err := c.SaveDreamFacts("", "test", []memory.Memory{{
					Name:        fmt.Sprintf("fact-g%d-i%d", g, i),
					Description: "并发写入",
					Body:        "并发写入正文",
				}}); err != nil {
					t.Errorf("SaveDreamFacts g%d i%d: %v", g, i, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	want := map[string]bool{}
	for g := 0; g < 4; g++ {
		for i := 0; i < 5; i++ {
			want[fmt.Sprintf("fact-g%d-i%d", g, i)] = false
		}
	}
	list := c.Memory().Store.List()
	if len(list) != len(want) {
		t.Fatalf("facts = %d, want %d（并发收敛失败）", len(list), len(want))
	}
	for _, m := range list {
		if _, ok := want[m.Name]; !ok {
			t.Fatalf("意外条目 %q", m.Name)
		}
		want[m.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("条目 %q 丢失（快照回退）", name)
		}
	}
}
