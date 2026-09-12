package app

// sin_export 工具测试（v4.267 入册，源自 .tmp/sin-next-knife 留存件的导出半边；
// sin_illustrate 仍是设计档「刻意不做、留待拍板」，不入册）：
// 导出落盘不覆盖同名、文件名净化、工具集顺序与只读口径。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sinToolByName 从工具集里按名取工具（找不到 = nil）——定义在 sin_tools.go。

// TestSinExportToolWritesMarkdown sin_export：导出图文 Markdown 到原罪自有
// 导出目录，同名不覆盖（-2 后缀），内容含正文与插图引用。
func TestSinExportToolWritesMarkdown(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("唐末浮生")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	extra := `{"illustrations":{"0":"C:\\art\\a.png"}}`
	if err := a.appendChatExchange(story.ID, "开始吧",
		"他推门进来。\n@@插图|雨夜里的短发女人@@\n", extra); err != nil {
		t.Fatalf("appendChatExchange: %v", err)
	}

	tool := sinToolByName(a.sinToolSet(story.ID), sinToolExport)
	if tool == nil {
		t.Fatal("sin_export 未注册")
	}
	if tool.ReadOnly() {
		t.Error("sin_export 会新建文件，ReadOnly 必须为 false")
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"filename":"唐末浮生-第一章.md"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	path := filepath.Join(sinExportsDir(), "唐末浮生-第一章.md")
	if !strings.Contains(out, path) {
		t.Fatalf("返回文本应给出路径 %s, got %q", path, out)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
	md := string(raw)
	for _, want := range []string{"他推门进来。", "![雨夜里的短发女人](C:\\art\\a.png)"} {
		if !strings.Contains(md, want) {
			t.Errorf("导出内容缺少 %q:\n%s", want, md)
		}
	}

	// 同名不覆盖：第二次导出落 -2，第一份内容原样保留。
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"filename":"唐末浮生-第一章"}`)); err != nil {
		t.Fatalf("第二次 Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sinExportsDir(), "唐末浮生-第一章-2.md")); err != nil {
		t.Errorf("同名导出应落 -2 后缀: %v", err)
	}

	// 空故事（未知 id）诚实报错，不落空文件。
	empty := sinExportTool{app: a, topicID: "sin_missing_1"}
	if _, err := empty.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("无内容故事应报错")
	}
}

// TestSinExportFileName 文件名净化：路径分隔/上跳/保留字符剔除，空名走故事 id
// + 时间戳，已带 .md 不重复加后缀。
func TestSinExportFileName(t *testing.T) {
	now := time.Date(2026, 9, 12, 21, 30, 5, 0, time.Local)
	cases := []struct {
		raw, topicID, want string
		wantErr            bool
	}{
		{raw: "", topicID: "sin_1", want: "sin_1-20260912-213005"},
		{raw: "   ", topicID: "sin_1", want: "sin_1-20260912-213005"},
		{raw: "第一章", topicID: "sin_1", want: "第一章"},
		{raw: " 第一章.md ", topicID: "sin_1", want: "第一章"},
		// 路径分隔剔除后，首尾的点也被裁掉（Windows 不允许以点结尾的文件名，
		// 且上跳名一律不留）——结果与输入完全不同，绝不还原成可执行路径。
		{raw: "../etc/passwd", topicID: "sin_1", want: "etcpasswd"},
		{raw: `a:b*c?d"e<f>g|h`, topicID: "sin_1", want: "abcdefgh"},
		{raw: "...", topicID: "sin_1", wantErr: true},
		{raw: "/", topicID: "sin_1", wantErr: true},
	}
	for _, c := range cases {
		got, err := sinExportFileName(c.raw, c.topicID, now)
		if c.wantErr {
			if err == nil {
				t.Errorf("sinExportFileName(%q) 应报错, got %q", c.raw, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("sinExportFileName(%q): %v", c.raw, err)
			continue
		}
		if got != c.want {
			t.Errorf("sinExportFileName(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

// TestSinExportInToolSet sin_export 在工具集末位（收尾动作），工具集整体仍按
// sinToolOrder 稳定输出。
func TestSinExportInToolSet(t *testing.T) {
	a, _ := newSinCastTestApp(t)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	tools := a.sinToolSet(story.ID)
	if len(tools) == 0 || tools[len(tools)-1].Name() != sinToolExport {
		names := make([]string, 0, len(tools))
		for _, tl := range tools {
			names = append(names, tl.Name())
		}
		t.Fatalf("sin_export 应在工具集末位, got %v", names)
	}
}
