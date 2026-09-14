package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestChapterMemories_RoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "novel")
	pm, err := Create(dir, "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}

	// 缺文件 = 空文件 + nil error（新项目正常态）
	f, err := pm.ReadChapterMemories(12)
	if err != nil || len(f.Items) != 0 {
		t.Fatalf("缺文件应返回空: %v %+v", err, f)
	}

	items := []types.StoryMemory{
		{ID: "012-foreshadow-1", ChapterNum: 12, ChapterFile: "012.md",
			Type: types.MemoryTypeForeshadow, Content: "月圆星门开", Importance: 0.8,
			IsForeshadow: types.ForeshadowFlagPlanted, CreatedAt: "2026-09-15T00:00:00Z"},
		{ID: "012-chapter_summary-1", ChapterNum: 12,
			Type: types.MemoryTypeChapterSummary, Content: "本章摘要", Importance: 0.6},
	}
	if err := pm.WriteChapterMemories(12, items); err != nil {
		t.Fatalf("写入: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "memories", "MMM-012-memory.json")); err != nil {
		t.Fatalf("文件应存在: %v", err)
	}
	f, err = pm.ReadChapterMemories(12)
	if err != nil || len(f.Items) != 2 || f.Items[0].ID != "012-foreshadow-1" {
		t.Fatalf("读回不一致: %v %+v", err, f)
	}
	if f.Items[0].IsForeshadow != types.ForeshadowFlagPlanted {
		t.Fatalf("伏笔标记应回环: %+v", f.Items[0])
	}

	// 空 items = 删除该章记忆文件（重分析后无合格记忆不留陈旧档）
	if err := pm.WriteChapterMemories(12, nil); err != nil {
		t.Fatalf("空写: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "memories", "MMM-012-memory.json")); !os.IsNotExist(err) {
		t.Fatalf("空写后文件应删除: %v", err)
	}
	// 再读回 = 空文件正常态
	if f, err = pm.ReadChapterMemories(12); err != nil || len(f.Items) != 0 {
		t.Fatalf("删除后读回应空: %v %+v", err, f)
	}
}
