package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/outline"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newNovelSearchFixtureApp 构造带项目 + 大纲代理的测试 App（临时小说目录 fixture）。
func newNovelSearchFixtureApp(t *testing.T) *App {
	t.Helper()
	a := newCharacterLibTestApp(t)
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	a.outlineAgent = outline.New(nil, pm, &config.Config{}, nil)
	return a
}

// mustWriteOutlines 写入大纲节点（失败即终止）。
func mustWriteOutlines(t *testing.T, pm *project.Manager, nodes ...types.OutlineNode) {
	t.Helper()
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}
}

// TestNovelSearch_Guards 覆盖全文检索守卫分支（不读取真实项目文件）。
func TestNovelSearch_Guards(t *testing.T) {
	a := newCharacterLibTestApp(t)

	if _, err := a.NovelSearch("剑修"); err == nil || !strings.Contains(err.Error(), "请先打开项目") {
		t.Fatalf("无项目时应报错: %v", err)
	}
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	if _, err := a.NovelSearch("剑修"); err == nil || !strings.Contains(err.Error(), "大纲未初始化") {
		t.Fatalf("outlineAgent 为空时应报错: %v", err)
	}
	hits, err := a.NovelSearch("   ")
	if err != nil || len(hits) != 0 {
		t.Fatalf("空查询应返回空结果: %v %v", hits, err)
	}
}

// TestNovelSearch_MultiHitsWithPositions 覆盖多命中返回、命中序次、段落索引与
// 段内 rune 偏移的正确性，含标题命中、分支章节与 CRLF 分段。
func TestNovelSearch_MultiHitsWithPositions(t *testing.T) {
	a := newNovelSearchFixtureApp(t)
	pm := a.getPM()

	mustWriteOutlines(t, pm,
		types.OutlineNode{ID: "vol", Title: "第一卷", OrderIndex: 0}, // 分组节点不参与
		types.OutlineNode{ID: "n1", Title: "第一章 启程", OrderIndex: 1},
		types.OutlineNode{ID: "n2", Title: "第二章 剑修传说", OrderIndex: 2}, // 标题命中
		types.OutlineNode{ID: "n3", Title: "第三章 分支", OrderIndex: 3, Branch: "a"},
	)
	// 段落切分含 \n\n 与 \r\n\r\n 两种（后者对齐前端阅读渲染器）。
	ch1 := "山门外的少年，一心想成为剑修。\n\n另一位剑修路过。\r\n\r\n剑修再临。"
	if err := pm.WriteChapter(1, ch1); err != nil {
		t.Fatalf("写章节: %v", err)
	}
	if err := pm.WriteChapterBranch(3, "a", "分支中的剑修。"); err != nil {
		t.Fatalf("写分支章节: %v", err)
	}

	hits, err := a.NovelSearch("剑修")
	if err != nil {
		t.Fatalf("NovelSearch: %v", err)
	}
	// 第一章正文 3 处 + 第三章分支正文 1 处 + 第二章标题 1 处 = 5 处 / 3 章
	if len(hits) != 5 {
		t.Fatalf("应返回 5 条命中, 得到 %d: %+v", len(hits), hits)
	}
	if hits[0].TotalHits != 5 || hits[0].ChapterCount != 3 {
		t.Fatalf("汇总字段应为 total=5 chapters=3: %+v", hits[0])
	}

	// 正文命中按出现顺序：段 0 偏移 12 → 段 1 偏移 3 → 段 2（CRLF 后）偏移 0
	wantOffsets := []struct {
		para  int
		off   int
		match int
	}{{0, 12, 1}, {1, 3, 2}, {2, 0, 3}}
	for i, w := range wantOffsets {
		h := hits[i]
		if h.ChapterNum != 1 || h.TitleHit {
			t.Fatalf("hits[%d] 应为第一章正文命中: %+v", i, h)
		}
		if h.MatchIndex != w.match || h.ParagraphIndex != w.para || h.CharOffset != w.off {
			t.Fatalf("hits[%d] 位置 = (match=%d, para=%d, off=%d), 期望 %v",
				i, h.MatchIndex, h.ParagraphIndex, h.CharOffset, w)
		}
		if h.MatchLen != 2 {
			t.Fatalf("hits[%d].MatchLen 应为 2: %+v", i, h)
		}
		if !strings.Contains(h.Snippet, "剑修") {
			t.Fatalf("hits[%d].Snippet 应包含命中词: %q", i, h.Snippet)
		}
	}

	// 标题命中：位置信息不适用（-1），snippet 语义不变
	th := hits[3]
	if th.ChapterNum != 2 || !th.TitleHit || th.ParagraphIndex != -1 || th.CharOffset != -1 {
		t.Fatalf("标题命中字段异常: %+v", th)
	}
	if th.Snippet != "章节标题命中" || th.MatchIndex != 1 {
		t.Fatalf("标题命中 snippet/序次异常: %+v", th)
	}

	// 分支章节命中
	bh := hits[4]
	if bh.ChapterNum != 3 || bh.Branch != "a" || bh.ParagraphIndex != 0 || bh.CharOffset != 4 {
		t.Fatalf("分支命中位置异常: %+v", bh)
	}
}

// TestNovelSearch_Caps 覆盖单章上限（20）与总量上限（300），
// 且 total_hits/chapter_count 统计不受返回条数上限影响。
func TestNovelSearch_Caps(t *testing.T) {
	a := newNovelSearchFixtureApp(t)
	pm := a.getPM()

	const chapters, perChapter = 16, 25
	nodes := make([]types.OutlineNode, 0, chapters)
	for i := 1; i <= chapters; i++ {
		nodes = append(nodes, types.OutlineNode{
			ID: "n" + string(rune('a'+i-1)), Title: "第几章", OrderIndex: i,
		})
		if err := pm.WriteChapter(i, strings.Repeat("目标词", perChapter)); err != nil {
			t.Fatalf("写章节 %d: %v", i, err)
		}
	}
	mustWriteOutlines(t, pm, nodes...)

	hits, err := a.NovelSearch("目标词")
	if err != nil {
		t.Fatalf("NovelSearch: %v", err)
	}
	if len(hits) != 300 {
		t.Fatalf("总量上限 300, 得到 %d", len(hits))
	}
	if hits[0].TotalHits != chapters*perChapter || hits[0].ChapterCount != chapters {
		t.Fatalf("汇总应 total=%d chapters=%d, 得到 %d/%d",
			chapters*perChapter, chapters, hits[0].TotalHits, hits[0].ChapterCount)
	}
	// 每章最多返回 20 条：前 15 章各 20 条恰好填满 300，第 16 章不再返回
	countCh1, countCh16 := 0, 0
	for _, h := range hits {
		switch h.ChapterNum {
		case 1:
			countCh1++
		case 16:
			countCh16++
		}
	}
	if countCh1 != 20 || countCh16 != 0 {
		t.Fatalf("单章应截断为 20（ch1=%d, ch16=%d）", countCh1, countCh16)
	}
	// 命中序次在章内连续递增
	for i, h := range hits[:20] {
		if h.MatchIndex != i+1 {
			t.Fatalf("第一章第 %d 条命中序次应为 %d: %d", i+1, i+1, h.MatchIndex)
		}
	}
}

func TestSnippetAround(t *testing.T) {
	body := strings.Repeat("甲", 100) + "目标词" + strings.Repeat("乙", 100)
	idx := strings.Index(body, "目标词")
	s := snippetAround(body, idx, 3)
	if !strings.Contains(s, "目标词") || !strings.HasPrefix(s, "…") || !strings.HasSuffix(s, "…") {
		t.Fatalf("snippet 应包含命中词且带省略号: %q", s)
	}
	short := snippetAround("目标词", 0, 3)
	if short != "目标词" {
		t.Fatalf("短文本不应加省略号: %q", short)
	}
}
