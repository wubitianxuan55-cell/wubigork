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

// TestSaveSceneMeta 元数据专职写路径：字段落盘 + 身份字段不可变 +
// 非法状态整单拒绝 + 标题空串保持原值 + blob 投影不受元数据影响。
func TestSaveSceneMeta(t *testing.T) {
	a, id := mustMaterializedChapter(t, "旧稿第一段。")

	meta := `{"title":"雨夜码头","summary":"交接失败","povCharId":"ch_lin","location":"码头仓库","timeOfDay":"深夜","emotion":"紧张","tags":["climax","action"],"status":"revising"}`
	if err := a.SaveSceneMeta(3, id, meta); err != nil {
		t.Fatalf("SaveSceneMeta 失败: %v", err)
	}
	scenes, err := a.GetChapterScenes(3)
	if err != nil || len(scenes) != 1 {
		t.Fatalf("读回失败: %v", err)
	}
	got := scenes[0]
	for k, want := range map[string]string{
		"title": "雨夜码头", "summary": "交接失败", "povCharId": "ch_lin",
		"location": "码头仓库", "timeOfDay": "深夜", "emotion": "紧张", "status": "revising",
	} {
		if got[k] != want {
			t.Fatalf("%s = %v, want %q", k, got[k], want)
		}
	}
	tags, _ := got["tags"].([]string)
	if len(tags) != 2 || tags[0] != "climax" {
		t.Fatalf("tags 应落盘: %v", got["tags"])
	}

	// 身份字段不可变：ID/Slug/Order 原样；标题空串=保持原标题。
	if err := a.SaveSceneMeta(3, id, `{"title":"   ","slug":"hacked","order":99,"wordCount":1}`); err != nil {
		t.Fatalf("空标题保存应成功: %v", err)
	}
	scenes2, _ := a.GetChapterScenes(3)
	if scenes2[0]["title"] != "雨夜码头" || scenes2[0]["slug"] == "hacked" {
		t.Fatalf("身份字段被改写: %+v", scenes2[0])
	}

	// 非法状态整单拒绝（本次的 summary 也不落）。
	if err := a.SaveSceneMeta(3, id, `{"summary":"不该落盘","status":"published"}`); err == nil {
		t.Fatal("非法状态应拒绝")
	}
	scenes3, _ := a.GetChapterScenes(3)
	if scenes3[0]["summary"] == "不该落盘" {
		t.Fatalf("拒绝路径不应部分落盘: %+v", scenes3[0])
	}

	// 元数据保存不触发 blob 变化。
	blob, _ := a.GetChapter(3)
	if got, _ := blob["content"].(string); got != "旧稿第一段。" {
		t.Fatalf("元数据保存不应改 blob: %q", got)
	}

	// 不存在的场景报错。
	if err := a.SaveSceneMeta(3, "nope", `{"title":"x"}`); err == nil {
		t.Fatal("不存在场景应报错")
	}
}
