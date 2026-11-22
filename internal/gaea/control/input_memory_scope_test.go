package control

// 记忆注入分席的 Compose 端到端机器钉（用户拍板 2026-11-22：聊天接融合、
// 办公保缓存旧口径）。play 分区会话 → RecallBlockFused（图谱发现臂生效）；
// work 分区会话 → RecallBlock 逐字旧口径（零词面命中的互引邻居不入选）。

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestComposeMemoryScopeDispatch(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	defer db.CloseDatabase(dir)
	s := memory.SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	// A 词法命中「部署」；B 与查询零词面重叠，仅经 A 的 [[互引]] 可达——
	// 图谱发现臂只在聊天席生效的探针。
	if _, err := s.Save(memory.Memory{
		Name: "deploy-runbook", Title: "部署手册",
		Description: "部署步骤", Kind: memory.KindSemantic, Type: memory.TypeProject,
		Body: "部署前置检查，回滚见 [[rollback-policy]]",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(memory.Memory{
		Name: "rollback-policy", Title: "回滚口径",
		Description: "灰度失败回滚纪律", Kind: memory.KindSemantic, Type: memory.TypeProject,
		Body: "先摘流量再回滚版本",
	}); err != nil {
		t.Fatal(err)
	}

	// 聊天席（play 分区目录）：图谱发现臂经 Compose 全链生效。
	playC := New(Options{Sink: newAttentionRecorder(), Memory: &memory.Set{Store: s, DB: gdb},
		SessionPath: filepath.Join(dir, "play", "sess-1.jsonl")})
	playOut := playC.Compose("部署")
	if !strings.Contains(playOut, "[MEM:deploy-runbook]") {
		t.Fatalf("聊天席词法命中丢失: %q", playOut)
	}
	if !strings.Contains(playOut, "[MEM:rollback-policy]") {
		t.Fatalf("聊天席图谱发现臂未生效: %q", playOut)
	}

	// 办公席（work 分区目录）：同一存储同一查询，互引邻居不入选（旧口径）。
	workC := New(Options{Sink: newAttentionRecorder(), Memory: &memory.Set{Store: s, DB: gdb},
		SessionPath: filepath.Join(dir, "work", "sess-1.jsonl")})
	workOut := workC.Compose("部署")
	if !strings.Contains(workOut, "[MEM:deploy-runbook]") {
		t.Fatalf("办公席词法命中丢失: %q", workOut)
	}
	if strings.Contains(workOut, "[MEM:rollback-policy]") {
		t.Fatalf("办公席不应吃图谱发现面（保缓存旧口径）: %q", workOut)
	}

	// 空 sessionPath（会话未落盘）：保守回旧口径。
	emptyC := New(Options{Sink: newAttentionRecorder(), Memory: &memory.Set{Store: s, DB: gdb}})
	emptyOut := emptyC.Compose("部署")
	if strings.Contains(emptyOut, "[MEM:rollback-policy]") {
		t.Fatalf("空 sessionPath 应保守回旧口径: %q", emptyOut)
	}
}
