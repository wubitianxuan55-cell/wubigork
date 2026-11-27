package proposal

import (
	"strings"
	"testing"
)

// detectSample 技术标常见体例：没有井号标题，夹着目录行、表格行、代码块、
// 以数字开头的正文句。
var detectSample = strings.Join([]string{
	"包1技术标V5",
	"",
	"目录",
	"第一章 工程概况 ………… 5",
	"第二章 施工部署 ………… 12",
	"",
	"**第一章 工程概况**",
	"本项目位于沿江一公里，工程内容包括道路、排水与绿化。",
	"1.1 项目概况",
	"道路全长 3.2 公里。",
	"1.1.1 建设条件",
	"（一）地形地貌",
	"场地平坦。",
	"（二）水文条件",
	"1.1.2 周边环境",
	"一、编制依据",
	"依据招标文件及设计图纸。",
	"二．编制原则",
	"1.2 建设内容",
	"第二章 施工部署",
	"2.1 施工组织机构",
	"2.1.1 组织机构设置",
	"2.2 施工进度计划",
	"3 台水泵运行正常",
	"| **6.资源配置方案** |",
	"| 人工 | 材料 |",
	"```",
	"1.1 代码块里不算",
	"```",
}, "\n")

func TestDetectHeadings(t *testing.T) {
	got := DetectHeadings(strings.Split(detectSample, "\n"))
	byLine := map[int]HeadingCandidate{}
	for _, c := range got {
		byLine[c.Line] = c
	}
	// 拿得准的 13 行：行号 → 建议层级。
	want := map[int]int{7: 1, 9: 2, 11: 3, 12: 4, 14: 4, 15: 3, 16: 4, 18: 4, 19: 2, 20: 1, 21: 2, 22: 3, 23: 2}
	sure := 0
	for _, c := range got {
		if c.Weak {
			continue
		}
		sure++
		if lv, ok := want[c.Line]; !ok || lv != c.Level {
			t.Errorf("第 %d 行「%s」层级 %d，期望 %d（已列入=%v）", c.Line, c.Title, c.Level, lv, ok)
		}
	}
	if sure != len(want) {
		t.Errorf("拿得准的应有 %d 行，实得 %d：%+v", len(want), sure, got)
	}
	for line, want := range map[int]string{7: "第一章 工程概况", 12: "（一）地形地貌", 18: "二．编制原则"} {
		if c := byLine[line]; c.Title != want {
			t.Errorf("第 %d 行标题 %q，期望 %q", line, c.Title, want)
		}
	}
	// 单个数字开头的正文句：章号里没有 3，只列为拿不准。
	if c := byLine[24]; !c.Weak || c.Title != "3 台水泵运行正常" {
		t.Errorf("第 24 行应列为拿不准：%+v", byLine[24])
	}
	// 藏在表格行里的章名。
	if c := byLine[25]; !c.InTable || c.Title != "6.资源配置方案" {
		t.Errorf("表格行里的章名应认出来：%+v", byLine[25])
	}
	// 目录行、代码块、普通表格与正文都不算。
	for _, line := range []int{4, 5, 8, 10, 13, 17, 26, 28} {
		if _, ok := byLine[line]; ok {
			t.Errorf("第 %d 行不该被当成标题：%+v", line, byLine[line])
		}
	}
}

func TestDetectHeadingsNumberedWithoutChapter(t *testing.T) {
	lines := strings.Split(strings.Join([]string{
		"## 1 工程概况",
		"## 1.1 项目概况",
		"### （1）建设地点",
		"2 施工部署",
		"2.1 组织机构",
		"2026 年度计划安排如下",
		"100 人以上",
	}, "\n"), "\n")
	got := DetectHeadings(lines)
	want := map[int]int{1: 1, 2: 2, 3: 3, 4: 1, 5: 2}
	for _, c := range got {
		if c.Weak {
			if c.Line != 7 {
				t.Errorf("第 %d 行不该出现：%+v", c.Line, c)
			}
			continue
		}
		if lv, ok := want[c.Line]; !ok || lv != c.Level {
			t.Errorf("第 %d 行「%s」层级 %d，期望 %d（已列入=%v）", c.Line, c.Title, c.Level, lv, ok)
		}
	}
	if len(got) != len(want)+1 {
		t.Errorf("应认出 5 行拿得准的 + 1 行拿不准的：%+v", got)
	}
}

// display_title：title 照源文件写法定位、入库用显示名；模型只给修正后的写法
// 时，差一两个字也能对上；编号补回去；太短的标题不容错。
func TestSaveOutlineDisplayTitle(t *testing.T) {
	st := newStore(t)
	src := strings.Join([]string{
		"5.2.5.2 抑制您味",
		"采用生物滤池除臭。",
		"5.2.5.3 恶臭监测",
		"设置监测点。",
	}, "\n")
	cases := []struct {
		name      string
		entry     OutlineEntry
		wantTitle string
		wantErr   bool
	}{
		{"照源文件写法定位", OutlineEntry{Level: 1, Title: "5.2.5.2 抑制您味", DisplayTitle: "抑制异味"}, "5.2.5.2 抑制异味", false},
		{"只给修正后的写法", OutlineEntry{Level: 1, Title: "抑制异味", DisplayTitle: "抑制异味"}, "5.2.5.2 抑制异味", false},
		{"指了行号", OutlineEntry{Level: 1, Title: "抑制异味", Line: 1, DisplayTitle: "抑制异味"}, "5.2.5.2 抑制异味", false},
		{"显示名自带编号", OutlineEntry{Level: 1, Title: "抑制您味", DisplayTitle: "5.2.5.2 抑制异味"}, "5.2.5.2 抑制异味", false},
		{"不带显示名不容错", OutlineEntry{Level: 1, Title: "抑制异味"}, "", true},
	}
	for i, c := range cases {
		title := "错字方案"
		if i > 0 {
			title = strings.Repeat("错", i) + "方案" // 每例一份方案
		}
		in := SaveInput{Meta: Proposal{Title: title}, Finalize: true, SourceFile: "a.md", Outline: []OutlineEntry{c.entry}, Lines: strings.Split(src, "\n")}
		res, err := st.Save(in)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s：应定位失败", c.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s：%v", c.name, err)
		}
		secs, err := st.Outline(res.ProposalID)
		if err != nil || len(secs) != 1 || secs[0].Title != c.wantTitle {
			t.Fatalf("%s：入库标题 %q，期望 %q（%v）", c.name, secs[0].Title, c.wantTitle, err)
		}
		if len(res.TitleFixes) != 1 || !strings.Contains(res.TitleFixes[0], "源文件「5.2.5.2 抑制您味」→ 入库「5.2.5.2 抑制异味」") {
			t.Errorf("%s：修正明细应列出源文件写法与入库写法：%v", c.name, res.TitleFixes)
		}
	}

	// 太短的标题差一个字就是另一个标题，不容错。
	_, err := st.Save(SaveInput{Meta: Proposal{Title: "短标题"}, Finalize: true, SourceFile: "a.md",
		Outline: []OutlineEntry{{Level: 1, Title: "除臭法", DisplayTitle: "除臭法"}}, Lines: []string{"除奥法", "正文"}})
	if err == nil {
		t.Error("三字标题不该容错定位")
	}
}

func TestTypoDistance(t *testing.T) {
	if d := typoDistance("5.2.5.2 抑制您味", "抑制异味"); d != 1 {
		t.Errorf("差一个字应为 1：%d", d)
	}
	if d := typoDistance("5.2.5.2 恶臭监测", "抑制异味"); d != -1 {
		t.Errorf("不相干的标题应为 -1：%d", d)
	}
	if d := typoDistance("除奥法", "除臭法"); d != -1 {
		t.Errorf("三字标题不容错：%d", d)
	}
	if d := typoDistance("除臭工艺选择您的研究。", "除臭工艺选择"); d != -1 {
		t.Errorf("句末标点的正文句不算：%d", d)
	}
}
