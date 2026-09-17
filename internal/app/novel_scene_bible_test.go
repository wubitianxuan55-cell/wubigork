package app

// POV 视图（刀7续收官，规格 进度计划/gaea-pov-view-20260917.md）测试：
// 整章合成路（sceneID 空）视图形状与恒非 nil 切片 / 未知场景报错 / 无项目报错。
// POV 掩码本身（povView/hiddenFacts 的裁剪语义）由 novelcontext 包测试钉死，
// 此处只钉 wire 投影。

import (
	"strings"
	"testing"
)

func TestNovelSceneBibleView_ChapterLevelShape(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteChapter(t, a, 1, "POV 视图样本正文。")

	v, err := a.NovelSceneBibleView(1, "")
	if err != nil {
		t.Fatalf("整章视图: %v", err)
	}
	// 恒非 nil 切片（前端零判空契约）
	if v.Characters == nil || v.HiddenFacts == nil || v.Foreshadows == nil || v.Memories == nil {
		t.Fatalf("切片字段应恒非 nil: %+v", v)
	}
	if v.Tags == nil {
		t.Fatalf("Tags 应恒非 nil: %+v", v)
	}
}

func TestNovelSceneBibleView_UnknownSceneErrors(t *testing.T) {
	a := newFingerprintTestApp(t)
	mustWriteChapter(t, a, 1, "样本正文。")
	_, err := a.NovelSceneBibleView(1, "no-such-scene")
	if err == nil || !strings.Contains(err.Error(), "场景不存在") {
		t.Fatalf("未知场景应报错，得到: %v", err)
	}
}

func TestNovelSceneBibleView_NoProjectErrors(t *testing.T) {
	a := newFingerprintTestApp(t)
	a.setPM(nil)
	if _, err := a.NovelSceneBibleView(1, ""); err == nil || !strings.Contains(err.Error(), "请先打开项目") {
		t.Fatalf("无项目应报错，得到: %v", err)
	}
}
