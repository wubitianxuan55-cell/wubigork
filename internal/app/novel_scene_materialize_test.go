package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 1A 主路径接通：blob 章物化为场景 ─────────────────────────────
//
// CreateChapter 主路径写整章 blob（chapters/NNN.md），此前阅读页
// GetChapterScenes 对 V4 项目走 SceneManager.List()，纯 blob 章返回空列表，
// 前端逐场景生成因「无绑定 ID」全禁用。物化后首场景承载 blob 全文。

// newMaterializeApp 创建项目并迁到 v4 结构（此时无章节 → 迁移只写 v4 标记），
// 随后写一章 blob 正文，返回待测 App 与项目句柄。
func newMaterializeApp(t *testing.T, blob string) (*App, *project.Manager) {
	t.Helper()
	a := newGateEmptyApp()
	pm := newGateProject(t)
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4 失败: %v", err)
	}
	if !pm.IsV4() {
		t.Fatalf("项目应为 v4 结构")
	}
	if err := pm.WriteChapter(3, blob); err != nil {
		t.Fatalf("写 blob 章失败: %v", err)
	}
	a.setPM(pm)
	return a, pm
}

// TestGetChapterScenes_MaterializesBlobChapter 纯 blob 章首次读场景 →
// 物化出唯一场景（真实 id + blob 全文 + done）；再读幂等不重复物化。
func TestGetChapterScenes_MaterializesBlobChapter(t *testing.T) {
	blob := "第三章正文：夜雨落进青云宗，叶辰握紧了断剑。"
	a, _ := newMaterializeApp(t, blob)

	scenes, err := a.GetChapterScenes(3)
	if err != nil {
		t.Fatalf("GetChapterScenes 不应报错: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("应物化出 1 个场景，得到 %d", len(scenes))
	}
	first := scenes[0]
	id, _ := first["id"].(string)
	if !strings.HasSuffix(id, "-chapter") {
		t.Errorf("物化场景 id 应以 -chapter 结尾，得到 %q", id)
	}
	if got, _ := first["content"].(string); got != blob {
		t.Errorf("物化场景正文应等于 blob 全文，得到 %q", got)
	}
	if got, _ := first["status"].(string); got != string(types.SceneDone) {
		t.Errorf("物化场景状态应为 done，得到 %q", got)
	}
	if got, _ := first["wordCount"].(int); got != len([]rune(blob)) {
		t.Errorf("字数应与 blob 一致 %d，得到 %v", len([]rune(blob)), got)
	}

	// 幂等：再次读取仍是一个场景，id 不变
	scenes2, err := a.GetChapterScenes(3)
	if err != nil {
		t.Fatalf("二次读取不应报错: %v", err)
	}
	if len(scenes2) != 1 {
		t.Fatalf("二次读取应仍为 1 个场景，得到 %d", len(scenes2))
	}
	if got, _ := scenes2[0]["id"].(string); got != id {
		t.Errorf("二次读取 id 应不变 %q，得到 %q", id, got)
	}
}

// TestCreateScene_MaterializesBlobFirst 纯 blob 章上加场景 →
// 先物化首场景（blob 全文），新场景排其后且为空正文（draft）。
func TestCreateScene_MaterializesBlobFirst(t *testing.T) {
	blob := "第三章正文：剑光过处，山石崩裂。"
	a, _ := newMaterializeApp(t, blob)

	created, err := a.CreateScene(3, "confrontation", "对峙")
	if err != nil {
		t.Fatalf("CreateScene 不应报错: %v", err)
	}
	if got, _ := created["id"].(string); !strings.HasSuffix(got, "-confrontation") {
		t.Errorf("新场景 id 应以 -confrontation 结尾，得到 %q", got)
	}

	scenes, err := a.GetChapterScenes(3)
	if err != nil {
		t.Fatalf("GetChapterScenes 不应报错: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("物化+新建应共 2 个场景，得到 %d", len(scenes))
	}
	if got, _ := scenes[0]["content"].(string); got != blob {
		t.Errorf("首场景应为物化的 blob 全文，得到 %q", got)
	}
	if got, _ := scenes[1]["content"].(string); got != "" {
		t.Errorf("新场景应为空正文，得到 %q", got)
	}
	if got, _ := scenes[1]["status"].(string); got != string(types.SceneDraft) {
		t.Errorf("新场景状态应为 draft，得到 %q", got)
	}
}

// TestGetChapterScenes_EmptyBlobNoMaterialize 无正文的章节不物化空场景。
func TestGetChapterScenes_EmptyBlobNoMaterialize(t *testing.T) {
	a, _ := newMaterializeApp(t, "")

	scenes, err := a.GetChapterScenes(3)
	if err != nil {
		t.Fatalf("GetChapterScenes 不应报错: %v", err)
	}
	if len(scenes) != 0 {
		t.Fatalf("空 blob 章不应物化场景，得到 %d 个", len(scenes))
	}
}

// TestGetChapterScenes_ExistingScenesUntouched 已有场景的章不被 blob 覆盖：
// 直接经 scene.Manager 建场景（绕开 CreateScene 的物化副作用），再读列表
// 保持调用方数据，不注入 blob 场景（幂等不重物化）。
func TestGetChapterScenes_ExistingScenesUntouched(t *testing.T) {
	a, pm := newMaterializeApp(t, "第三章正文：旧稿。")
	sm := pm.SceneManager(3)
	if _, err := sm.Create("opening", "开场"); err != nil {
		t.Fatalf("建场景失败: %v", err)
	}
	scenes, err := a.GetChapterScenes(3)
	if err != nil {
		t.Fatalf("GetChapterScenes 不应报错: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("已有场景的章应保持 1 个场景，得到 %d", len(scenes))
	}
	if id, _ := scenes[0]["id"].(string); strings.HasSuffix(id, "-chapter") {
		t.Fatalf("已有场景的章不应再物化 blob 场景，得到 %q", id)
	}
}
