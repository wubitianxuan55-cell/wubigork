package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── 定稿档 chapters/NNN{,a,b,c}-final.json（阶段八 8.4）───────

func finFixture(num int, branch string) *types.ChapterFinalization {
	return &types.ChapterFinalization{
		Version:     types.ChapterFinalizationVersion,
		ChapterNum:  num,
		Branch:      branch,
		FinalizedAt: "2026-10-14T23:00:00+08:00",
		Note:        "本卷收官",
		Applied: &types.FinalizeApplied{
			Characters: 2,
			Foreshadows: &types.FinalizeForeshadowSync{
				PlantedCount: 1, ResolvedCount: 1, CreatedCount: 0,
			},
			StageRecap: &types.StageRecapEntry{Start: 1, End: 10, Text: "第一阶段合账"},
		},
	}
}

func TestChapterFinalization_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	pm := &Manager{Dir: dir}

	if err := pm.WriteChapterFinalization(finFixture(10, "")); err != nil {
		t.Fatalf("写主线定稿档: %v", err)
	}
	if err := pm.WriteChapterFinalization(finFixture(5, "a")); err != nil {
		t.Fatalf("写分支定稿档: %v", err)
	}

	cf, ok, err := pm.ReadChapterFinalization(10, "")
	if err != nil || !ok || cf == nil {
		t.Fatalf("主线定稿档读回: ok=%v err=%v", ok, err)
	}
	if cf.Note != "本卷收官" || cf.Applied == nil || cf.Applied.Characters != 2 {
		t.Fatalf("定稿档往返不一致: %+v", cf)
	}
	// 分支档独立路径，互不覆盖
	bf, ok, err := pm.ReadChapterFinalization(5, "a")
	if err != nil || !ok || bf == nil {
		t.Fatalf("分支定稿档读回: ok=%v err=%v", ok, err)
	}
	if _, ok, _ := pm.ReadChapterFinalization(5, ""); ok {
		t.Fatal("第5章主线不应有定稿档（分支档不得串线）")
	}
	assertNoTempLeftovers(t, filepath.Join(dir, "chapters"))
}

func TestChapterFinalization_MissingIsNormalAndCorruptKept(t *testing.T) {
	dir := t.TempDir()
	pm := &Manager{Dir: dir}

	// 缺失 = (nil,false,nil) 正常态（未定稿）
	if cf, ok, err := pm.ReadChapterFinalization(7, ""); err != nil || ok || cf != nil {
		t.Fatalf("未定稿应为正常态: cf=%+v ok=%v err=%v", cf, ok, err)
	}

	// 损坏 = error 且原文件原封不动
	if err := pm.WriteChapterFinalization(finFixture(10, "")); err != nil {
		t.Fatal(err)
	}
	path := pm.ChapterFinalizationPath(10, "")
	corrupt := []byte("{bad json")
	if err := os.WriteFile(path, corrupt, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := pm.ReadChapterFinalization(10, ""); err == nil {
		t.Fatal("损坏档应报错")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("原文件应仍在: %v", err)
	}
	if string(after) != string(corrupt) {
		t.Fatalf("损坏档被覆盖: %q", after)
	}

	// 无主档拒收
	if err := pm.WriteChapterFinalization(&types.ChapterFinalization{ChapterNum: 0}); err == nil {
		t.Fatal("章号 0 应拒收")
	}
	if err := pm.WriteChapterFinalization(nil); err == nil {
		t.Fatal("nil 应拒收")
	}
}

// ── 阶段合账缓存 chapters/stage-recaps.json ──────────────────

func TestStageRecapCache_UpsertAndRead(t *testing.T) {
	dir := t.TempDir()
	pm := &Manager{Dir: dir}

	// 缺失=空表正常态
	if e, ok, _ := pm.ReadStageRecap(1); ok || e != nil {
		t.Fatalf("空缓存应 miss: %+v", e)
	}

	e1 := types.StageRecapEntry{Start: 1, End: 10, Text: "第一阶段合账", SourceChapter: 10}
	if err := pm.UpsertStageRecap(1, e1); err != nil {
		t.Fatalf("upsert 阶段1: %v", err)
	}
	e2 := types.StageRecapEntry{Start: 11, End: 20, Text: "第二阶段合账", SourceChapter: 20}
	if err := pm.UpsertStageRecap(2, e2); err != nil {
		t.Fatalf("upsert 阶段2: %v", err)
	}

	got, ok, err := pm.ReadStageRecap(1)
	if err != nil || !ok || got.Text != "第一阶段合账" {
		t.Fatalf("阶段1读回: %+v ok=%v err=%v", got, ok, err)
	}
	got2, ok, _ := pm.ReadStageRecap(2)
	if !ok || got2.Text != "第二阶段合账" {
		t.Fatalf("阶段2读回: %+v ok=%v", got2, ok)
	}
	// 同阶段重写=覆盖（重定稿语义）
	e1.Text = "重定稿后的合账"
	if err := pm.UpsertStageRecap(1, e1); err != nil {
		t.Fatal(err)
	}
	got, _, _ = pm.ReadStageRecap(1)
	if got.Text != "重定稿后的合账" {
		t.Fatalf("同阶段应覆盖: %+v", got)
	}
	// 非法阶段号拒收
	if err := pm.UpsertStageRecap(0, e1); err == nil {
		t.Fatal("阶段号 0 应拒收")
	}
}
