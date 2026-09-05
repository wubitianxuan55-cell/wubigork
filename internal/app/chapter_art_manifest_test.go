package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/project"
)

func TestChapterArtManifestRoundTrip(t *testing.T) {
	pm := &project.Manager{Dir: t.TempDir()}
	if err := appendChapterArt(pm, 3, "ih-a", filepath.Join(pm.Dir, "a.png")); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := appendChapterArt(pm, 4, "ih-b", filepath.Join(pm.Dir, "b.png")); err != nil {
		t.Fatalf("append: %v", err)
	}
	all := listChapterArt(pm, 0)
	if len(all) != 2 || all[0].AssetID != "ih-b" || all[1].AssetID != "ih-a" {
		t.Fatalf("list all = %+v", all)
	}
	ch3 := listChapterArt(pm, 3)
	if len(ch3) != 1 || ch3[0].AssetID != "ih-a" {
		t.Fatalf("list ch3 = %+v", ch3)
	}
}

func TestChapterArtManifestPerChapterCap(t *testing.T) {
	orig := chapterArtPerChapter
	chapterArtPerChapter = 3
	defer func() { chapterArtPerChapter = orig }()

	pm := &project.Manager{Dir: t.TempDir()}
	for i := 1; i <= 5; i++ {
		if err := appendChapterArt(pm, 1, assetIDN(i), filepath.Join(pm.Dir, assetIDN(i)+".png")); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	got := listChapterArt(pm, 1)
	if len(got) != 3 {
		t.Fatalf("单章应折叠到 3 条，got %d", len(got))
	}
	// 最新在前：3/4/5。
	if got[0].AssetID != assetIDN(5) || got[2].AssetID != assetIDN(3) {
		t.Fatalf("折叠保留最近 3 条: %+v", got)
	}
}

func TestChapterArtManifestCorruptFileTolerated(t *testing.T) {
	pm := &project.Manager{Dir: t.TempDir()}
	p := chapterArtManifestPath(pm)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte("not-json"), 0o644); err != nil {
		t.Fatalf("write corrupt: %v", err)
	}
	if got := listChapterArt(pm, 0); len(got) != 0 {
		t.Fatalf("坏清单应返回空: %+v", got)
	}
	// 坏清单不阻断追加（重建）。
	if err := appendChapterArt(pm, 1, "ih-x", filepath.Join(pm.Dir, "x.png")); err != nil {
		t.Fatalf("坏清单后追加失败: %v", err)
	}
	if got := listChapterArt(pm, 1); len(got) != 1 {
		t.Fatalf("坏清单重建失败: %+v", got)
	}
}

func assetIDN(n int) string {
	return "ih-" + string(rune('0'+n))
}
