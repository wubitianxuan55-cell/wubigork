package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// briefView 是 GaeaMemoryMorningBrief 返回 JSON 的轻量解析视图（只取断言所需字段）。
type briefView struct {
	Items []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"items"`
	Rules       []string `json:"rules"`
	Dreamed24h  int      `json:"dreamed24h"`
	GeneratedAt int64    `json:"generatedAt"`
}

// morningBriefOf 调用绑定并解析 JSON，失败即 fatal。
func morningBriefOf(t *testing.T) briefView {
	t.Helper()
	raw, err := (&App{}).GaeaMemoryMorningBrief()
	if err != nil {
		t.Fatalf("GaeaMemoryMorningBrief error: %v", err)
	}
	var v briefView
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("返回 JSON 不可解析: %v\nraw=%s", err, raw)
	}
	return v
}

func TestGaeaMemoryMorningBriefJSON(t *testing.T) {
	store := newOfficeMemoryTestEnv(t)
	base := time.Now()
	save := func(m memory.Memory) {
		t.Helper()
		if _, err := store.Save(m); err != nil {
			t.Fatal(err)
		}
	}
	// 3 条 work 事实（LastUsedAt 错开，排序确定）+ 1 条 play 事实（不得混入）。
	save(memory.Memory{Name: "fact-c", Type: memory.TypeProject, Kind: memory.KindSemantic, Description: "第三条工作事实", LastUsedAt: base.Add(3 * time.Hour)})
	save(memory.Memory{Name: "fact-a", Type: memory.TypeUser, Kind: memory.KindProcedural, Description: "第一条用户事实", LastUsedAt: base.Add(1 * time.Hour)})
	save(memory.Memory{Name: "fact-b", Type: memory.TypeProject, Kind: memory.KindSemantic, Description: "第二条工作事实", LastUsedAt: base.Add(2 * time.Hour)})
	save(memory.Memory{Name: "play-only", Type: memory.TypeProject, Kind: memory.KindSemantic, Description: "play 空间事实", Space: "play", LastUsedAt: base.Add(10 * time.Hour)})

	v := morningBriefOf(t)
	if len(v.Items) != 3 {
		t.Fatalf("items = %d, want 3（play 事实不得混入 work 晨报）", len(v.Items))
	}
	// LastUsedAt 降序：fact-c(3h) fact-b(2h) fact-a(1h)
	want := []string{"fact-c", "fact-b", "fact-a"}
	for i, w := range want {
		if v.Items[i].Name != w {
			t.Errorf("Items[%d].Name = %q, want %q", i, v.Items[i].Name, w)
		}
	}
	if v.Items[0].Description != "第三条工作事实" {
		t.Errorf("description 未透传，got %q", v.Items[0].Description)
	}
	if v.GeneratedAt <= 0 {
		t.Errorf("generatedAt = %d, want > 0", v.GeneratedAt)
	}
}

func TestGaeaMemoryMorningBriefEmptyStore(t *testing.T) {
	_ = newOfficeMemoryTestEnv(t)
	t.Setenv("GAEA_DATA_ROOT", t.TempDir()) // 无审计文件 → dreamed24h 降级 0

	v := morningBriefOf(t)
	if v.Items == nil || len(v.Items) != 0 {
		t.Errorf("空记忆 items 应为空数组，got %#v", v.Items)
	}
	if v.Rules == nil || len(v.Rules) != 0 {
		t.Errorf("空记忆 rules 应为空数组，got %#v", v.Rules)
	}
	if v.Dreamed24h != 0 {
		t.Errorf("Dreamed24h = %d, want 0（审计缺失降级）", v.Dreamed24h)
	}
}

// writeDreamAudit 在 GAEA_DATA_ROOT 下写入 dream 审计文件（多行 JSONL）。
func writeDreamAudit(t *testing.T, lines []string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", dir)
	if err := os.WriteFile(filepath.Join(dir, "dream-audit.jsonl"), []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGaeaMemoryMorningBriefDreamCount(t *testing.T) {
	_ = newOfficeMemoryTestEnv(t)
	now := time.Now().UTC()
	writeDreamAudit(t, []string{
		fmt.Sprintf(`{"ts":%q,"source":"auto_dream","saved":3}`, now.Add(-1*time.Hour).Format(time.RFC3339)),
		fmt.Sprintf(`{"ts":%q,"source":"distill_merge","saved":0}`, now.Add(-2*time.Hour).Format(time.RFC3339)),
		// 显式来源不计入晨报口径
		fmt.Sprintf(`{"ts":%q,"source":"explicit","saved":9}`, now.Add(-3*time.Hour).Format(time.RFC3339)),
		// 超 24h 时窗不计
		fmt.Sprintf(`{"ts":%q,"source":"auto_dream","saved":2}`, now.Add(-25*time.Hour).Format(time.RFC3339)),
	})
	v := morningBriefOf(t)
	// 3 + 0 = 3（explicit 与超窗行排除）
	if v.Dreamed24h != 3 {
		t.Errorf("Dreamed24h = %d, want 3", v.Dreamed24h)
	}
}

func TestGaeaMemoryMorningBriefDreamCountDegrades(t *testing.T) {
	_ = newOfficeMemoryTestEnv(t)
	// 审计文件全部为损坏行：解析失败降级 0，不阻断晨报。
	writeDreamAudit(t, []string{
		`{not-json`,
		`{"ts":"","source":"auto_dream","saved":1}`,
	})
	v := morningBriefOf(t)
	if v.Dreamed24h != 0 {
		t.Errorf("损坏审计 Dreamed24h = %d, want 0（降级不阻断）", v.Dreamed24h)
	}
}
