package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestDistillMergeViewsEnrichesDisplayFields(t *testing.T) {
	now := time.Now()
	ms := []memory.Memory{
		{Name: "release-steps", Title: "发布步骤", Description: "发布前三步检查", Type: memory.TypeFeedback, Kind: memory.KindProcedural, UpdatedAt: now},
		{Name: "deploy-checklist", Title: "部署清单", Description: "发布前三步检查", Type: memory.TypeFeedback, Kind: memory.KindProcedural, UpdatedAt: now.Add(-time.Hour)},
	}
	views := distillMergeViews(ms)
	if len(views) != 1 {
		t.Fatalf("应产出 1 张合并卡, got %d", len(views))
	}
	v := views[0]
	if v.Keep != "release-steps" || v.KeepTitle != "发布步骤" {
		t.Errorf("保留条应带标题, got %+v", v)
	}
	if v.Archive != "deploy-checklist" || v.ArchiveTitle != "部署清单" {
		t.Errorf("归档条应带标题, got %+v", v)
	}
	if !strings.Contains(v.KeepUpdatedAt, "-") {
		t.Errorf("保留条时间应格式化, got %q", v.KeepUpdatedAt)
	}
	if !strings.HasPrefix(v.ID, "merge-") {
		t.Errorf("候选应带稳定 ID, got %q", v.ID)
	}
	if again := distillMergeViews(ms); again[0].ID != v.ID {
		t.Errorf("ID 应稳定, got %q vs %q", again[0].ID, v.ID)
	}
}
