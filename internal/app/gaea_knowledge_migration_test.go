package app

// gaea_knowledge_migration_test.go — 审计 GA3-13 的 app 侧接线：旧知识库迁移
// 失败必须「可见」——经既有 gaea-event notice 通道（前端/面板同路）+ slog.Warn
// 留痕（含 Reason）；成功或未跑则静默（不制造启动噪音）。

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/knowledge"
)

// withKnowledgeMigrationState 注入迁移状态读取缝，返回恢复由 t.Cleanup 负责。
func withKnowledgeMigrationState(t *testing.T, st knowledge.MigrationState) {
	t.Helper()
	orig := knowledgeMigrationStateFn
	t.Cleanup(func() { knowledgeMigrationStateFn = orig })
	knowledgeMigrationStateFn = func() knowledge.MigrationState { return st }
}

// captureSlogWarn 抓取默认 slog 输出（断言日志留痕用）。
func captureSlogWarn(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return func() string { return buf.String() }
}

// TestKnowledgeMigrationFailureVisible 迁移失败 → notice（含 Reason）+ slog.Warn
// 双通道留痕。
func TestKnowledgeMigrationFailureVisible(t *testing.T) {
	const reason = "迁移标记读取失败: no such table: profile"
	withKnowledgeMigrationState(t, knowledge.MigrationState{Ran: true, Failed: true, Reason: reason})
	logs := captureSlogWarn(t)
	notices := captureGaeaNotices(t)

	a := &App{core: &core{}}
	a.reportKnowledgeMigrationState()

	// ① 既有 notice 通道可见（payload 文案含原因）
	got := notices()
	if len(got) != 1 {
		t.Fatalf("迁移失败应发 1 条 notice，实际 %d 条: %v", len(got), got)
	}
	if !strings.Contains(got[0], "迁移失败") || !strings.Contains(got[0], reason) {
		t.Errorf("notice 文案须点名迁移失败与原因，实际: %q", got[0])
	}
	// ② 日志留痕（slog.Warn 带 reason 字段）
	out := logs()
	if !strings.Contains(out, "level=WARN") || !strings.Contains(out, reason) {
		t.Errorf("迁移失败须 slog.Warn 留痕（含原因），实际日志: %q", out)
	}
}

// TestKnowledgeMigrationSuccessSilent 成功 / 未跑不算失败：不出声（零 notice、
// 零 WARN），避免启动噪音。
func TestKnowledgeMigrationSuccessSilent(t *testing.T) {
	for _, name := range []string{"success", "not-ran"} {
		t.Run(name, func(t *testing.T) {
			st := knowledge.MigrationState{Ran: true}
			if name == "not-ran" {
				st = knowledge.MigrationState{}
			}
			withKnowledgeMigrationState(t, st)
			logs := captureSlogWarn(t)
			notices := captureGaeaNotices(t)

			a := &App{core: &core{}}
			a.reportKnowledgeMigrationState()

			if got := notices(); len(got) != 0 {
				t.Errorf("非失败态不应发 notice，实际: %v", got)
			}
			if out := logs(); strings.Contains(out, "level=WARN") {
				t.Errorf("非失败态不应 WARN，实际日志: %q", out)
			}
		})
	}
}
