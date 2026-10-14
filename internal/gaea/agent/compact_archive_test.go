package agent

// 阶段九 9.1「摘要后可回溯全量记录」：过闸摘要（LLM 或机械回落）统一携带
// 归档指针——原文永不丢，摘要只是视图。

import (
	"context"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
)

func TestCompactArchivePointerInDigest(t *testing.T) {
	dir := t.TempDir()
	a := testAgentC(4000, 0.8, 3)
	a.compaction.ArchiveDir = dir
	s := NewSession("system")
	for i := 0; i < 15; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "request " + strings.Repeat("q", 100)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "response " + strings.Repeat("r", 200)})
		s.Add(provider.Message{Role: provider.RoleTool, Name: "read_file", Content: "content " + strings.Repeat("c", 300)})
	}
	a.session = s

	// nil provider → 机械摘要回落；两条路径的归档指针统一由 compact() 注入。
	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatalf("CompactNow: %v", err)
	}
	var digest string
	found := false
	for _, m := range a.session.Messages {
		if isCompactionSummary(m) {
			found = true
			digest = m.Content
			break
		}
	}
	if !found {
		t.Fatal("压缩后应有摘要消息")
	}
	const marker = "Full transcript of the folded region: "
	i := strings.Index(digest, marker)
	if i < 0 {
		t.Fatalf("摘要应携带归档指针:\n%s", digest)
	}
	path := strings.TrimSpace(digest[i+len(marker):])
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("归档指针应指向 ArchiveDir 下: %q (dir=%q)", path, dir)
	}
}

func TestCompactNoArchiveDir_NoPointer(t *testing.T) {
	a := testAgentC(4000, 0.8, 3)
	// ArchiveDir 为空：不归档，摘要不带指针行（无路可指不伪造）。
	s := NewSession("system")
	for i := 0; i < 15; i++ {
		s.Add(provider.Message{Role: provider.RoleUser, Content: "request " + strings.Repeat("q", 100)})
		s.Add(provider.Message{Role: provider.RoleAssistant, Content: "response " + strings.Repeat("r", 200)})
	}
	a.session = s
	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatalf("CompactNow: %v", err)
	}
	for _, m := range a.session.Messages {
		if isCompactionSummary(m) && strings.Contains(m.Content, "Full transcript of the folded region") {
			t.Fatalf("无归档目录不得出现指针行:\n%s", m.Content)
		}
	}
}
