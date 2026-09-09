package app

import (
	"strings"
	"testing"
)

// ── 阅读页场景化：blob↔场景 双向同步 ─────────────────────────────
//
// 场景写路径（保存/重排/快照恢复/生成）之后 blob = 场景投影（空段拼接）；
// 整章重写（CreateChapter 主线完成点）之后场景重置为 blob 单场景。

// mustMaterializedChapter 造一个 V4 项目：blob 章先经 GetChapterScenes 物化。
func mustMaterializedChapter(t *testing.T, blob string) (*App, string) {
	t.Helper()
	a, pm := newMaterializeApp(t, blob)
	scenes, err := a.GetChapterScenes(3)
	if err != nil || len(scenes) != 1 {
		t.Fatalf("物化前置失败: %v scenes=%v", err, scenes)
	}
	id, _ := scenes[0]["id"].(string)
	_ = pm
	return a, id
}

// TestSaveScene_SyncsBlobProjection 逐场景保存后 blob = 场景投影。
func TestSaveScene_SyncsBlobProjection(t *testing.T) {
	a, id := mustMaterializedChapter(t, "旧稿第一段。")

	if err := a.SaveScene(3, id, "新正文第一段。"); err != nil {
		t.Fatalf("SaveScene 失败: %v", err)
	}
	blob, err := a.GetChapter(3)
	if err != nil {
		t.Fatalf("GetChapter 失败: %v", err)
	}
	if got, _ := blob["content"].(string); got != "新正文第一段。" {
		t.Fatalf("blob 应同步为场景投影，得到 %q", got)
	}

	// 两场景：投影 = 空段拼接（无分隔线）
	created, err := a.CreateScene(3, "second", "第二场")
	if err != nil {
		t.Fatalf("CreateScene 失败: %v", err)
	}
	id2, _ := created["id"].(string)
	if err := a.SaveScene(3, id2, "第二场正文。"); err != nil {
		t.Fatalf("SaveScene(2) 失败: %v", err)
	}
	blob2, _ := a.GetChapter(3)
	want := "新正文第一段。\n\n第二场正文。"
	if got, _ := blob2["content"].(string); got != want {
		t.Fatalf("blob 应为 %q，得到 %q", want, got)
	}
}

// TestReorderScenes_SyncsBlob 重排后 blob 投影跟随新顺序。
func TestReorderScenes_SyncsBlob(t *testing.T) {
	a, id1 := mustMaterializedChapter(t, "第一场。")
	created, _ := a.CreateScene(3, "second", "第二场")
	id2, _ := created["id"].(string)
	_ = a.SaveScene(3, id2, "第二场。")

	if err := a.ReorderScenes(3, []string{id2, id1}); err != nil {
		t.Fatalf("ReorderScenes 失败: %v", err)
	}
	blob, err := a.GetChapter(3)
	if err != nil {
		t.Fatalf("GetChapter 失败: %v", err)
	}
	want := "第二场。\n\n第一场。"
	if got, _ := blob["content"].(string); got != want {
		t.Fatalf("重排后 blob 应为 %q，得到 %q", want, got)
	}
}

// TestRebuildScenesFromBlob_ResetsScenes 整章重写后场景重置为单场景。
func TestRebuildScenesFromBlob_ResetsScenes(t *testing.T) {
	a, pm := newMaterializeApp(t, "旧稿。")
	if _, err := a.CreateScene(3, "extra", "手动加的场景"); err != nil {
		t.Fatalf("CreateScene 失败: %v", err)
	}
	// 模拟 CreateChapter 整章重写：写新 blob → 重建场景
	newBlob := "重写后的全新正文。"
	if err := pm.WriteChapter(3, newBlob); err != nil {
		t.Fatalf("WriteChapter 失败: %v", err)
	}
	rebuildScenesFromBlob(pm, 3)

	scenes, err := a.GetChapterScenes(3)
	if err != nil {
		t.Fatalf("GetChapterScenes 失败: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("重建后应只剩 1 个场景，得到 %d", len(scenes))
	}
	if got, _ := scenes[0]["content"].(string); got != newBlob {
		t.Fatalf("重建场景正文应为新 blob，得到 %q", got)
	}
	if id, _ := scenes[0]["id"].(string); !strings.HasSuffix(id, "-chapter") {
		t.Fatalf("重建场景 id 应为物化规则 -chapter，得到 %q", id)
	}
}
