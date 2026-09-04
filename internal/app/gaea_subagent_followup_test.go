package app

// v4.66 追问失败可感知（受理侧）：GaeaSubagentFollowUp 受理即同步清掉上一
// 次追问的失败摘要（followUpError）——清盘发生在返回「已受理」之前，前端
// 派发后的首轮轮询不会把旧失败误记到新一次头上。后台 runner 失败的原因写回
// 在 task 管道层（RunFollowUp → RecordFollowUpError，见 agent 包用例）。

import (
	"context"
	"errors"
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
