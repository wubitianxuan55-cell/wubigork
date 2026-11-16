package app

// wenxue_tools_test.go — 图文专属工具四族单测（2026-11-15 刀3）。
// GAEA_TUWEN_ROOT 指向临时目录，实测 章节解析/覆盖闸/角色卡自动标题/时间线
// 解析与追加/统计快照 全链（纯 Go 零引擎依赖，hermetic）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

func wenxueToolsTestRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("GAEA_TUWEN_ROOT", t.TempDir())
	root := wenxueWorkspace()
	if err := os.MkdirAll(filepath.Join(root, "章节"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func mustRun(t *testing.T, tool *wenxueTool, args string) string {
	t.Helper()
	out, err := tool.Execute(t.Context(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s 执行失败: %v", tool.name, err)
	}
	return out
}

func TestWenxueChapterWriteListReadFlow(t *testing.T) {
	root := wenxueToolsTestRoot(t)

	// 新建第一章
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"雨夜。巷口的路灯忽明忽暗。他握紧了刀。"}`)
	// 同号写入沿用原文件名（保命名），overwrite=false 已存在即拒
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(`{"number":1,"content":"覆盖尝试"}`)); err == nil {
		t.Fatal("overwrite=false 对已存在章节应拒绝")
	}
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"content":"雨夜。巷口的路灯忽明忽暗。他握紧了刀，刀柄上的红绳是他妹妹编的。","overwrite":true}`)
	// 第二章（衔接提示带第一章）
	out := mustRun(t, wenxueChapterWriteTool(), `{"number":2,"title":"追查","content":"第二天清晨，他在废料站找到了第一件证物。"}`)
	if !wenxueContains(out, "001-开局.md") {
		t.Fatalf("第二章回执应带衔接提示（含第一章文件名）: %s", out)
	}

	// 清单：解析+排序+字数
	list := mustRun(t, wenxueChapterListTool(), `{}`)
	if !wenxueContains(list, `"count":2`) || !wenxueContains(list, `001-开局.md`) || !wenxueContains(list, `002-追查.md`) {
		t.Fatalf("章节清单不符: %s", list)
	}

	// 读取：number 定位 + prevTail
	read := mustRun(t, wenxueChapterReadTool(), `{"number":2}`)
	if !wenxueContains(read, "证物") || !wenxueContains(read, "prevTail") {
		t.Fatalf("章节读取缺正文或衔接段: %s", read)
	}

	_ = root
}

func TestWenxueCharacterWriteAutoTitle(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	out := mustRun(t, wenxueCharacterWriteTool(), `{"name":"林晚","content":"设定：法医，32 岁。"}`)
	if !wenxueContains(out, "林晚.md") {
		t.Fatalf("角色卡落点不符: %s", out)
	}
	// 已存在拒
	if _, err := wenxueCharacterWriteTool().Execute(t.Context(), json.RawMessage(`{"name":"林晚","content":"改动"}`)); err == nil {
		t.Fatal("overwrite=false 对已存在角色卡应拒绝")
	}
	// 清单可见 + 名称取自首行标题
	list := mustRun(t, wenxueCharacterListTool(nil), `{}`)
	if !wenxueContains(list, `"name":"林晚"`) {
		t.Fatalf("角色清单缺林晚: %s", list)
	}
	_ = root
}

func TestWenxueTimelineAppendReadParse(t *testing.T) {
	root := wenxueToolsTestRoot(t)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":1,"text":"巷口初遇尸体。"}`)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":2,"text":"废料站找到证物。"}`)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":12,"text":"真相反转。"}`)

	// 全量读（章节序）
	all := mustRun(t, wenxueTimelineReadTool(), `{}`)
	if !wenxueContains(all, "真相反转") || !wenxueContains(all, `"count":3`) {
		t.Fatalf("时间线全量读不符: %s", all)
	}
	// since 过滤：第 12 章起只剩 1 条
	since := mustRun(t, wenxueTimelineReadTool(), `{"sinceChapter":12}`)
	if !wenxueContains(since, "真相反转") || wenxueContains(since, "巷口初遇") {
		t.Fatalf("sinceChapter 过滤不符: %s", since)
	}
	_ = root
}

func TestWenxueStatsSnapshot(t *testing.T) {
	wenxueToolsTestRoot(t)
	mustRun(t, wenxueChapterWriteTool(), `{"number":1,"title":"开局","content":"雨夜。"}`)
	mustRun(t, wenxueCharacterWriteTool(), `{"name":"林晚","content":"设定。"}`)
	mustRun(t, wenxueTimelineAppendTool(), `{"chapter":1,"text":"开局事件。"}`)
	stats := mustRun(t, wenxueStatsTool(), `{}`)
	for _, want := range []string{`"chapterCount":1`, `"characterCards":1`, `"timelineEvents":1`, `"settingFiles":0`} {
		if !wenxueContains(stats, want) {
			t.Fatalf("统计快照缺 %s: %s", want, stats)
		}
	}
}

func TestWenxueFinalDomainIsolation(t *testing.T) {
	wenxueToolsTestRoot(t)
	// 定稿写入（final_write 建目录+落盘）
	mustRun(t, wenxueFinalWriteTool(), `{"number":1,"title":"开局","content":"发布稿：雨夜。"}`)
	// 章节工具拒写定稿目录（防草稿入库）
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(`{"path":"定稿/001-开局.md","content":"草稿误入"}`)); err == nil {
		t.Fatal("章节工具写定稿目录应被拒绝（域隔离）")
	}
	// 定稿工具拒写定稿目录之外
	if _, err := wenxueFinalWriteTool().Execute(t.Context(), json.RawMessage(`{"path":"章节/001-开局.md","content":"越域"}`)); err == nil {
		t.Fatal("定稿工具写章节目录应被拒绝（域隔离）")
	}
	// 清单/读取只在定稿域
	list := mustRun(t, wenxueFinalListTool(), `{}`)
	if !wenxueContains(list, `"count":1`) {
		t.Fatalf("定稿清单应含 1 章: %s", list)
	}
	read := mustRun(t, wenxueFinalReadTool(), `{"number":1}`)
	if !wenxueContains(read, "发布稿：雨夜。") {
		t.Fatalf("定稿读取不符: %s", read)
	}
}

func TestWenxueCreateNovelScaffoldsFinalDir(t *testing.T) {
	wenxueToolsTestRoot(t)
	a := &App{}
	if _, err := a.WenxueCreateNovel("脚手架书", ""); err != nil {
		t.Fatalf("建书: %v", err)
	}
	root := wenxueWorkspace()
	if info, err := os.Stat(filepath.Join(root, "定稿")); err != nil || !info.IsDir() {
		t.Fatalf("新建小说应脚手架定稿目录（err=%v）", err)
	}
}

func TestWenxueToolsRejectPathEscape(t *testing.T) {
	wenxueToolsTestRoot(t)
	if _, err := wenxueChapterWriteTool().Execute(t.Context(), json.RawMessage(`{"path":"../逃逸.md","content":"x"}`)); err == nil {
		t.Fatal("穿越路径应被拒绝")
	}
	if _, err := wenxueChapterReadTool().Execute(t.Context(), json.RawMessage(`{"path":"../../外部.md"}`)); err == nil {
		t.Fatal("穿越路径读取应被拒绝")
	}
}

func wenxueContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// TestWenxueBaseToolsImageGenWorkTagged 图文生图面回归（2026-11-16）：媒体根
// 分档裁决的 image_gen 此前被 work 空间闸静默过滤（play 名表），wxWorkSpaceTool
// 适配后图文装配面在场；办公 gaeaExtraTools 原样 play 语义不变（红线）。
func TestWenxueBaseToolsImageGenWorkTagged(t *testing.T) {
	_ = wenxueToolsTestRoot(t)
	var wxTag, officeTag string
	for _, tl := range wenxueBaseTools(&App{}) {
		if tl.Name() == "image_gen" {
			wxTag = builtin.SpaceTagOf(tl)
		}
	}
	for _, tl := range gaeaExtraTools(&App{}) {
		if tl.Name() == "image_gen" {
			officeTag = builtin.SpaceTagOf(tl)
		}
	}
	if wxTag != "work" {
		t.Fatalf("图文 image_gen 空间标签 = %q, want work（生图面应装配）", wxTag)
	}
	if officeTag != "play" {
		t.Fatalf("办公 image_gen 空间标签 = %q, want play（办公语义不变红线）", officeTag)
	}
}
