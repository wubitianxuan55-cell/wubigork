package app

// wenxue_tools_warnings_test.go — 章节四类 fixture + 时间线 fixture + 回写句
// 三子串断言（2026-11-15 众神殿首刀验收合同，pan_1794754879290）。
// 章节四类：001-标题.md（正常解析）/ 第一章.md 式中文序号（Number=0）/
// 无编号文件 / 根目录回退——断言=告警条数 + 0 号组排序确定（无编号排编号前）。
// 时间线四类：合法锚 / 中文数字锚 / 标题行（不进 events）/ 缺章号行（聚合告警）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWenxueChapterFixtureFourClasses(t *testing.T) {
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	root := wenxueWorkspace()
	dir := filepath.Join(root, "章节")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("001-开局.md", "正文一。")    // ① 正常编号
	write("第二章 乱入.md", "中文序号正文。") // ② 中文序号 → Number=0
	write("随笔.md", "无编号正文。")      // ③ 无编号 → Number=0
	// ④ 根目录回退另测（本例有约定目录，不回退）。

	chapters, fellBack := wenxueListChaptersWithFallback(root)
	if fellBack {
		t.Fatal("有约定章节目录不应报告回退")
	}
	if len(chapters) != 3 {
		t.Fatalf("应枚举 3 个章节文件，得 %d: %+v", len(chapters), chapters)
	}
	// 0 号组排序确定：无编号（含中文序号）排编号前，组内按文件名字节序。
	if chapters[0].Name != "第二章 乱入.md" || chapters[1].Name != "随笔.md" || chapters[2].Name != "001-开局.md" {
		t.Fatalf("0 号组应排最前且序确定: %v %v %v", chapters[0].Name, chapters[1].Name, chapters[2].Name)
	}
	if chapters[2].Number != 1 || chapters[2].Title != "开局" {
		t.Fatalf("正常编号文件应解析出 章号/标题: %+v", chapters[2])
	}
	// 告警恰好一条（未编号聚合；无回退告警），且不逐文件重复。
	w := wenxueChapterWarnings(chapters, fellBack)
	if len(w) != 1 || !strings.Contains(w[0], "2 个章节文件章号未解析") {
		t.Fatalf("章节告警应恰一条且含未编号计数: %v", w)
	}

	// 工具回执：warnings 仅非空时携带；假 dir 输入参数已删（schema 无 dir 字段）。
	tl := wenxueChapterListTool()
	if strings.Contains(tl.schema, `"dir"`) {
		t.Fatalf("chapter_list schema 不应再声明 dir 输入参数: %s", tl.schema)
	}
	out := mustRun(t, tl, `{}`)
	if !strings.Contains(out, `"warnings"`) {
		t.Fatalf("有告警时回执应携带 warnings: %s", out)
	}
}

func TestWenxueChapterRootFallbackWarning(t *testing.T) {
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	root := wenxueWorkspace()
	// 无任何约定章节目录：工作区根一层放一个正常编号文件。
	if err := os.WriteFile(filepath.Join(root, "003-根上.md"), []byte("根目录正文。"), 0o644); err != nil {
		t.Fatal(err)
	}
	chapters, fellBack := wenxueListChaptersWithFallback(root)
	if !fellBack {
		t.Fatal("无约定目录应报告根目录回退")
	}
	if len(chapters) != 1 || chapters[0].Number != 3 {
		t.Fatalf("根回退应枚举并解析编号: %+v", chapters)
	}
	w := wenxueChapterWarnings(chapters, fellBack)
	if len(w) != 1 || !strings.Contains(w[0], "回退扫描工作区根") {
		t.Fatalf("回退告警应恰一条: %v", w)
	}
}

func TestWenxueTimelineHeaderSkipAndWarnings(t *testing.T) {
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	root := wenxueWorkspace()
	dir := filepath.Join(root, "时间线")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# 时间线\n\n- 第3章 · 主角入门派\n第十二章 · 中文序号事件\n- 无锚事件行\n"
	if err := os.WriteFile(filepath.Join(dir, "时间线.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := wenxueTimelineEntries(root)
	if len(entries) != 3 {
		t.Fatalf("标题行不进 events（期望 3 条，得 %d）: %+v", len(entries), entries)
	}
	// 排序=章号→文件→行：0 号组（缺锚）排最前，组内按文件行序——确定序。
	if entries[0].Chapter != 0 || entries[0].Line != 4 || entries[1].Chapter != 0 || entries[1].Line != 5 {
		t.Fatalf("0 号组应排最前且行序确定: %+v", entries)
	}
	if entries[2].Chapter != 3 || entries[2].Line != 3 {
		t.Fatalf("合法锚应解析出章 3（原第 3 行）: %+v", entries)
	}
	// 聚合告警一条，附两处 文件:行号（封顶 8 处）。
	w := wenxueTimelineWarnings(entries)
	if len(w) != 1 {
		t.Fatalf("缺锚告警应聚合一条: %v", w)
	}
	if !strings.Contains(w[0], "时间线/时间线.md:4") || !strings.Contains(w[0], "时间线/时间线.md:5") {
		t.Fatalf("告警应附 文件:行号: %s", w[0])
	}
	// 工具回执携带 warnings；stats 面同源（标题不再计成事件）。
	out := mustRun(t, wenxueTimelineReadTool(), `{}`)
	if !strings.Contains(out, `"warnings"`) || !strings.Contains(out, `"count":3`) {
		t.Fatalf("timeline_read 回执不符: %s", out)
	}
}

func TestWenxueTimelineAppendChapterHint(t *testing.T) {
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	_ = wenxueWorkspace()
	// 缺章号：落盘成功但回执带提示（不强制必填，保留非章节事件弹性）。
	out := mustRun(t, wenxueTimelineAppendTool(), `{"text":"全书纲要注意事项。"}`)
	if !strings.Contains(out, "本章事件应传 chapter") || !strings.Contains(out, "第12章") {
		t.Fatalf("缺锚回执应带锚点提示: %s", out)
	}
	// 带章号：无提示，落盘格式为「第12章 · 事件」。
	out = mustRun(t, wenxueTimelineAppendTool(), `{"chapter":12,"text":"真相反转。"}`)
	if strings.Contains(out, "hint") {
		t.Fatalf("带锚回执不应有提示: %s", out)
	}
	b, err := os.ReadFile(filepath.Join(wenxueWorkspace(), "时间线", "时间线.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "第12章 · 真相反转。") {
		t.Fatalf("落盘格式不符: %s", string(b))
	}
}

func TestWenxueWorkbenchWritebackSentence(t *testing.T) {
	// 回写句三子串（众神殿首刀 B 文案收尾，grep 断言钉死）：
	// 「写完一章必须回写」+ 阿拉伯数字锚点样例「第12章」+「不要新建 foreshadows.json」。
	for _, sub := range []string{
		"写完一章必须回写",
		"第12章",
		"不要新建 foreshadows.json",
	} {
		if !strings.Contains(wenxueWorkbenchBody, sub) {
			t.Fatalf("workbench 回写句缺子串 %q（断言对象=改后正文）", sub)
		}
	}
}
