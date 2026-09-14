package project

import (
	"testing"
	"time"

	"github.com/gaea/gaea/internal/types"
)

func TestRewriteVersionStore(t *testing.T) {
	dir := t.TempDir()
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	// 空列表正常态
	list, err := pm.ListRewriteVersions(3)
	if err != nil || len(list) != 0 {
		t.Fatalf("空列表应正常: %v %v", err, list)
	}
	// 非法 ID 拒绝（路径护栏）
	if _, err := pm.GetRewriteVersion(3, "../evil"); err == nil {
		t.Fatalf("含路径分隔符的 ID 应拒绝")
	}

	mk := func(id, original, newContent string, status types.RewriteStatus) *types.RewriteVersion {
		return &types.RewriteVersion{
			ID: id, ChapterNum: 3, Mode: types.RewriteModeWhole, Status: status,
			Source: types.RewriteSourceCustom, CustomInstr: "收紧",
			OriginalContent: original, OriginalWordCount: len([]rune(original)),
			NewContent: newContent, NewWordCount: len([]rune(newContent)),
			Similarity: 66.6,
		}
	}

	// 保存两版（ID 自动生成；时间可区分）
	v1 := mk("", "原文一", "重写一", types.RewriteCompleted)
	if err := pm.SaveRewriteVersion(v1); err != nil {
		t.Fatalf("保存 v1: %v", err)
	}
	if v1.ID == "" {
		t.Fatalf("ID 应自动生成")
	}
	time.Sleep(2 * time.Millisecond)
	v2 := mk("custom-id", "原文二", "重写二", types.RewriteCompleted)
	if err := pm.SaveRewriteVersion(v2); err != nil {
		t.Fatalf("保存 v2: %v", err)
	}

	list, err = pm.ListRewriteVersions(3)
	if err != nil || len(list) != 2 {
		t.Fatalf("应 2 条索引: %v %d", err, len(list))
	}
	// 时间倒序：最新在前
	if list[0].ID != v2.ID {
		t.Fatalf("索引应时间倒序: %+v", list)
	}

	// 全文读回（含快照）
	got, err := pm.GetRewriteVersion(3, v1.ID)
	if err != nil || got.OriginalContent != "原文一" || got.NewContent != "重写一" {
		t.Fatalf("全文读回不一致: %v %+v", err, got)
	}

	// 状态流转更新（索引同步）
	got.Status = types.RewriteApplied
	now := time.Now()
	got.AppliedAt = &now
	if err := pm.UpdateRewriteVersion(got); err != nil {
		t.Fatalf("更新: %v", err)
	}
	list, _ = pm.ListRewriteVersions(3)
	for _, e := range list {
		if e.ID == v1.ID && e.Status != types.RewriteApplied {
			t.Fatalf("索引状态应同步: %+v", e)
		}
	}
}
