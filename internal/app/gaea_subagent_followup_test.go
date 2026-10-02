package app

// v4.66 追问失败可感知（受理侧）：GaeaSubagentFollowUp 受理即同步清掉上一
// 次追问的失败摘要（followUpError）——清盘发生在返回「已受理」之前，前端
// 派发后的首轮轮询不会把旧失败误记到新一次头上。后台 runner 失败的原因写回
// 在 task 管道层（RunFollowUp → RecordFollowUpError，见 agent 包用例）。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGaeaSubagentFollowUp_ClearsStaleError(t *testing.T) {
	sessionPath, sessionDir := writeSubagentFixture(t)
	subDir := filepath.Join(sessionDir, "subagents")
	ref := "sa_20260902_100000_0000000004_d5d5d5d5"
	metaPath := filepath.Join(subDir, ref+".meta.json")
	writeJSON(t, metaPath, map[string]interface{}{
		"ref": ref, "status": "completed", "title": "可追问的运行",
		"followUpError": "上一枪的失败：provider 掉线",
		"createdAt":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		"updatedAt":     time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano),
	})

	// 接线一个立即失败的 runner：绑定照常受理（后台失败写回由 task 层负责，
	// 本用例只验证受理侧的同步清盘）。
	ga.mu.Lock()
	prev := ga.followUp
	ga.followUp = func(_ context.Context, _ string, _ string) error { return errors.New("后台失败") }
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.followUp = prev
		ga.mu.Unlock()
	})

	a := &App{core: &core{}}
	if _, err := a.GaeaSubagentFollowUp(sessionPath, ref, "追问：再展开第二点"); err != nil {
		t.Fatalf("GaeaSubagentFollowUp: %v", err)
	}

	// 同步清盘：返回时旧摘要已不在 meta 里（其余字段守恒由 store 用例覆盖）。
	b, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if strings.Contains(string(b), "上一枪的失败") {
		t.Fatal("stale followUpError should be cleared synchronously on dispatch")
	}
}

// TestGaeaSubagentFollowUp_PanicReleasesClaimAndNotifies 审计 P2 AP5-16：追问
// 后台 panic 后必须（1）释放单飞令牌（否则该 ref 永久卡在「已有追问正在运行」，
// 后续追问全被拒）、（2）发一条前端可见 notice（此前只 slog.Error 就吞掉，
// 前端只能看到「已受理但无进展」的僵态）。
func TestGaeaSubagentFollowUp_PanicReleasesClaimAndNotifies(t *testing.T) {
	ga.mu.Lock()
	prev := ga.followUp
	ga.followUp = func(context.Context, string, string) error { panic("followup boom") }
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.followUp = prev
		ga.mu.Unlock()
	})

	notices := captureGaeaNotices(t)
	ref := fmt.Sprintf("sa_20260902_100000_%012d_abcdef01", time.Now().UnixNano()%1000000000000)
	a := &App{core: &core{}}
	if _, err := a.GaeaSubagentFollowUp("", ref, "追问：再展开第二点"); err != nil {
		t.Fatalf("GaeaSubagentFollowUp: %v", err)
	}
	t.Cleanup(func() { followUpClaims.Delete(ref) })

	// 等后台 goroutine 的 defer 落定（复位 + notice）。
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, loaded := followUpClaims.Load(ref); !loaded && len(notices()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, loaded := followUpClaims.Load(ref); loaded {
		t.Fatal("panic 后单飞令牌未释放——该 ref 的追问将永久被拒")
	}
	got := notices()
	if len(got) == 0 {
		t.Fatal("panic 后未发前端可见 notice（AP5-16 要求可见化）")
	}
	if !strings.Contains(got[0], "子代理追问") || !strings.Contains(got[0], "followup boom") {
		t.Fatalf("notice 文案应含任务名与 panic 值: %q", got[0])
	}
}
