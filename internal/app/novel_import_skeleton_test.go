package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/bookimport"
	"github.com/gaea/gaea/internal/project"
)

// newSkeletonTestApp 构造 60 章 × ~600 字的中篇**导入**工程（走真实导入收口
// createImportedProject，章级大纲节点 imp-NNN 齐备；无模型：反推走规则兜底）。
func newSkeletonTestApp(t *testing.T) *App {
	t.Helper()
	a := newCharacterLibTestApp(t)
	a.cfg.NovelsDir = t.TempDir()
	body := strings.Repeat("这一段正文用来撑起篇幅，讲一个完整的小事件且有起承转合。", 30) // ~600+ 字/章，60 章 >3 万字
	chapters := make([]importChapter, 0, 60)
	for i := 1; i <= 60; i++ {
		chapters = append(chapters, importChapter{
			Title:   "第" + strconvItoa(i) + "章 试炼",
			Content: "第" + strconvItoa(i) + "章正文。\n\n" + body,
		})
	}
	res, err := createImportedProject(a.cfg.NovelsDir, "中篇测试", "玄幻", "默认", chapters, bookimport.Report{SplitStrategy: "booksource"})
	if err != nil {
		t.Fatalf("建导入工程: %v", err)
	}
	pm, err := project.Open(res.Path)
	if err != nil {
		t.Fatalf("打开工程: %v", err)
	}
	a.setPM(pm)
	return a
}

func strconvItoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return strconvItoa(n/10) + string(rune('0'+n%10))
}

// oh-story T4 篇幅路由：中篇（60 章 × ~600 字）反推应聚合为 6 个卷级骨架节点，
// apply 新建 seg-* 节点（planned）且重复应用幂等不堆积。
func TestNovelOutlineReconstruct_MidTierAggregates(t *testing.T) {
	a := newSkeletonTestApp(t)

	preview, err := a.NovelOutlineReconstruct()
	if err != nil {
		t.Fatalf("反推: %v", err)
	}
	if preview.Tier != "mid" || preview.SegmentSize != 10 {
		t.Fatalf("篇幅路由应为中篇 10 章聚合: %s %d", preview.Tier, preview.SegmentSize)
	}
	if len(preview.Items) != 6 {
		t.Fatalf("60 章应聚合为 6 个骨架节点: %d", len(preview.Items))
	}
	first := preview.Items[0]
	if first.ChapterFrom != 1 || first.ChapterTo != 10 || !strings.Contains(first.Title, "第1-10章") {
		t.Fatalf("首段跨度不符: %+v", first)
	}

	// apply：新建 seg-* 卷级参考节点
	raw, err := json.Marshal(preview.Items)
	if err != nil {
		t.Fatal(err)
	}
	n, err := a.NovelOutlineReconstructApply(string(raw))
	if err != nil {
		t.Fatalf("应用: %v", err)
	}
	if n != 6 {
		t.Fatalf("应新建 6 个骨架节点: %d", n)
	}
	of, err := a.getPM().ReadOutlines()
	if err != nil {
		t.Fatal(err)
	}
	segs := 0
	chaps := 0
	for _, node := range of.Nodes {
		if strings.HasPrefix(node.ID, "seg-") {
			segs++
			if node.Status != "planned" {
				t.Fatalf("骨架节点应为 planned: %+v", node)
			}
		} else {
			chaps++
		}
	}
	if segs != 6 || chaps != 60 {
		t.Fatalf("大纲应 60 章节节点 + 6 骨架节点: %d/%d", chaps, segs)
	}

	// 重复应用：seg-* 重建不堆积（可重复执行契约的骨架版）
	if _, err := a.NovelOutlineReconstructApply(string(raw)); err != nil {
		t.Fatalf("重复应用: %v", err)
	}
	of2, _ := a.getPM().ReadOutlines()
	segs2 := 0
	for _, node := range of2.Nodes {
		if strings.HasPrefix(node.ID, "seg-") {
			segs2++
		}
	}
	if segs2 != 6 {
		t.Fatalf("重复应用不应堆积骨架节点: %d", segs2)
	}
}

// 短篇路由回归：小工程反推保持章级条目（不聚合、无 tier 语义变化）。
func TestNovelOutlineReconstruct_ShortTierStaysChapterLevel(t *testing.T) {
	a := newFingerprintTestApp(t)
	body := strings.Repeat("这一段正文用来撑起篇幅。", 20) // ~400 字
	for i := 1; i <= 3; i++ {
		if err := a.getPM().WriteChapter(i, "第"+string(rune('0'+i))+"章 开篇\n\n"+body); err != nil {
			t.Fatal(err)
		}
	}
	preview, err := a.NovelOutlineReconstruct()
	if err != nil {
		t.Fatalf("反推: %v", err)
	}
	if preview.Tier != "short" || preview.SegmentSize != 0 {
		t.Fatalf("小工程应为短篇章级: %s %d", preview.Tier, preview.SegmentSize)
	}
	if len(preview.Items) != 3 || preview.Items[0].ChapterFrom != 0 {
		t.Fatalf("短篇应保持章级条目: %+v", preview.Items[0])
	}
}
