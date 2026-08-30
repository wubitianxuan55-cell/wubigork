package memory

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDistillMergeCandidates_SlugVariants(t *testing.T) {
	now := time.Now()
	ms := []Memory{
		{Name: "user-timezone", Title: "User Timezone", Description: "用户在东八区", Type: TypeUser, UpdatedAt: now.Add(-2 * time.Hour)},
		{Name: "user_timezone", Title: "User Timezone 2", Description: "用户时区是 UTC+8", Type: TypeUser, UpdatedAt: now},
		{Name: "unrelated-fact", Description: "完全无关", Type: TypeProject},
	}
	got := DistillMergeCandidates(ms)
	if len(got) != 1 {
		t.Fatalf("应产出 1 条同名异写候选, got %d: %+v", len(got), got)
	}
	c := got[0]
	if c.Keep != "user_timezone" || c.Archive != "user-timezone" {
		t.Errorf("较新者应保留, got keep=%q archive=%q", c.Keep, c.Archive)
	}
	if !strings.Contains(c.Reason, "同名异写") {
		t.Errorf("reason 应说明规则, got %q", c.Reason)
	}
}

func TestDistillMergeCandidates_SameTypeSameDescription(t *testing.T) {
	now := time.Now()
	ms := []Memory{
		{Name: "deploy-steps", Description: "发布前三步检查", Type: TypeFeedback, Kind: KindProcedural, UpdatedAt: now.Add(-time.Hour)},
		{Name: "release-checklist", Description: "发布前三步检查", Type: TypeFeedback, Kind: KindProcedural, UpdatedAt: now},
		{Name: "same-desc-diff-type", Description: "发布前三步检查", Type: TypeProject, Kind: KindSemantic, UpdatedAt: now.Add(-2 * time.Hour)},
	}
	got := DistillMergeCandidates(ms)
	if len(got) != 1 {
		t.Fatalf("同 type+kind 同描述才成候选, got %d: %+v", len(got), got)
	}
	if got[0].Keep != "release-checklist" || got[0].Archive != "deploy-steps" {
		t.Errorf("应保留较新者, got %+v", got[0])
	}
	if !strings.Contains(got[0].Reason, "同类型同描述") {
		t.Errorf("reason 应说明规则, got %q", got[0].Reason)
	}
}

func TestDistillMergeCandidates_CrossSpaceExcluded(t *testing.T) {
	now := time.Now()
	ms := []Memory{
		{Name: "user-timezone", Description: "用户在东八区", Type: TypeUser, Space: "work", UpdatedAt: now},
		{Name: "user_timezone", Description: "用户在东八区", Type: TypeUser, Space: "play", UpdatedAt: now.Add(-time.Hour)},
	}
	if got := DistillMergeCandidates(ms); len(got) != 0 {
		t.Errorf("跨空间不得合并（双空间红线）, got %+v", got)
	}
}

func TestDistillMergeCandidates_StableAndCapped(t *testing.T) {
	now := time.Now()
	// 每对独立描述 → 各成一组；大组（同描述多成员）按设计只出 1 个候选，
	// 合并后再次扫描渐进收账。
	var ms []Memory
	for i := 0; i < 20; i++ {
		ms = append(ms,
			Memory{Name: "dup-a-" + string(rune('a'+i)), Description: fmt.Sprintf("重复主题-%d", i), Type: TypeUser, UpdatedAt: now},
			Memory{Name: "dup-b-" + string(rune('a'+i)), Description: fmt.Sprintf("重复主题-%d", i), Type: TypeUser, UpdatedAt: now.Add(-time.Hour)},
		)
	}
	got := DistillMergeCandidates(ms)
	if len(got) != distillMaxMerges {
		t.Errorf("候选应封顶 %d, got %d", distillMaxMerges, len(got))
	}
	again := DistillMergeCandidates(ms)
	if len(got) != len(again) || got[0] != again[0] {
		t.Errorf("纯函数应同输入同输出: %+v vs %+v", got[0], again[0])
	}
}
