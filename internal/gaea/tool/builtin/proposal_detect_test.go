package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// proposal_detect_test.go — 2026-11-27 AI 反馈 #5' / #11 / #12 回归钉：
//   - #12 认标题（action=detect_headings）：按编号形态列出标题样行，可写成 outline
//     草稿，再用 outline_file 录入——不写库；
//   - #11 display_title：源文件标题有 OCR 错字时入库用修正后的写法，回执列修正明细；
//   - #5' 回执的份数口径：全库 N 份（现行 / 录入中），与落点、录入进度分开。

// proposalDetectSample 技术标常见体例：没有井号标题，夹着目录行、表格行、代码块、
// 以数字开头的正文句（与 proposal 包 DetectHeadings 的样例同构）。
const proposalDetectSample = `包1技术标V5

目录
第一章 工程概况 ………… 5
第二章 施工部署 ………… 12

**第一章 工程概况**
本项目位于沿江一公里，工程内容包括道路、排水与绿化。
1.1 项目概况
道路全长 3.2 公里。
1.1.1 建设条件
（一）地形地貌
场地平坦。
（二）水文条件
1.1.2 周边环境
一、编制依据
依据招标文件及设计图纸。
二．编制原则
1.2 建设内容
第二章 施工部署
2.1 施工组织机构
2.1.1 组织机构设置
2.2 施工进度计划
3 台水泵运行正常
| **6.资源配置方案** |
| 人工 | 材料 |
` + "```\n1.1 代码块里不算\n```\n"

func TestProposalDetectHeadingsAndOutlineFile(t *testing.T) {
	proposalTestEnv(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tech.md"), []byte(proposalDetectSample), 0o644); err != nil {
		t.Fatal(err)
	}
	save := proposalSaveTool{workDir: dir}
	ctx := context.Background()
	st, err := proposalStore()
	if err != nil {
		t.Fatal(err)
	}

	// 只读认标题：表格回执，不写库。
	out, err := save.Execute(ctx, toJSON(t, map[string]interface{}{"action": "detect_headings", "source_file": "tech.md"}))
	if err != nil {
		t.Fatal(err)
	}
	msg := toolMessage(t, out)
	for _, want := range []string{"认出 15 个标题样行**（拿得准 13、拿不准 2）", "| 7 | 1 | 章 | 第一章 工程概况 |",
		"| 12 | 4 | （一） | （一）地形地貌 |", "| 18 | 4 | 一、 | 二．编制原则 |", "拿不准", "表格行"} {
		if !strings.Contains(msg, want) {
			t.Errorf("认标题回执缺 %q：\n%s", want, msg)
		}
	}
	if list, err := st.List(); err != nil || len(list) != 0 {
		t.Fatalf("认标题不该写库：%d 份（%v）", len(list), err)
	}

	// 写草稿：拿得准的进文件，拿不准的只在回执里列。
	out, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"action": "detect_headings", "source_file": "tech.md", "outline_file": ".gaea/work/outline.json",
	}))
	if err != nil {
		t.Fatal(err)
	}
	msg = toolMessage(t, out)
	for _, want := range []string{"草稿已写出**：.gaea/work/outline.json（13 条", "没写进草稿的 2 行", "第 24 行「3 台水泵运行正常」"} {
		if !strings.Contains(msg, want) {
			t.Errorf("草稿回执缺 %q：\n%s", want, msg)
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".gaea", "work", "outline.json"))
	if err != nil {
		t.Fatal(err)
	}
	// 草稿是标准 JSON 数组，且每行一条（便于 edit_file 逐行改）。
	if lines := strings.Split(strings.TrimSpace(string(raw)), "\n"); len(lines) != 15 || !strings.HasPrefix(lines[0], "[") {
		t.Fatalf("草稿应为数组、13 条各占一行：%q", string(raw))
	}
	var draft []proposalOutlineArg
	if err := json.Unmarshal(raw, &draft); err != nil || len(draft) != 13 {
		t.Fatalf("草稿应能解析成 13 条：%d（%v）", len(draft), err)
	}
	if first := draft[0]; first.Level != 1 || first.Title != "第一章 工程概况" || first.Line != 7 {
		t.Fatalf("草稿首条异常：%+v", first)
	}

	// 用草稿录入：份数一行写明全库口径。
	out, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"title": "技术标", "source_file": "tech.md", "outline_file": ".gaea/work/outline.json", "finalize": true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	msg = toolMessage(t, out)
	for _, want := range []string{"**落点**: 工作区 ", "**方案库**: 全库共 1 份（现行 1、录入中 0），含本方案"} {
		if !strings.Contains(msg, want) {
			t.Errorf("录入回执缺 %q：\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "方案库共 1 份") {
		t.Errorf("份数不该再和落点挤在一行：\n%s", msg)
	}
	if n := proposalSectionCount(t, "技术标"); n != 13 {
		t.Fatalf("草稿录入后应有 13 个标题节点：%d", n)
	}

	// 录到一半的方案算进「录入中」。
	out, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"title": "第二份", "source_file": "tech.md",
		"outline": []map[string]interface{}{{"level": 1, "title": "1.2 建设内容", "line": 19}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if msg := toolMessage(t, out); !strings.Contains(msg, "**方案库**: 全库共 2 份（现行 1、录入中 1），含本方案") {
		t.Errorf("续录中的方案应计入录入中：\n%s", msg)
	}

	// 每行一个对象的写法也认。
	jsonl := filepath.Join(dir, ".gaea", "work", "outline.jsonl")
	if err := os.WriteFile(jsonl, []byte("{\"level\":1,\"title\":\"第二章 施工部署\",\"line\":20}\n{\"level\":2,\"title\":\"2.2 施工进度计划\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"title": "逐行写法", "source_file": "tech.md", "outline_file": ".gaea/work/outline.jsonl", "finalize": true, "dry_run": true,
	}))
	if err != nil || !strings.Contains(toolMessage(t, out), "演练通过") {
		t.Errorf("每行一条的 outline_file 应能用：err=%v out=%s", err, out)
	}

	// outline 与 outline_file 同给、认标题带录入内容、草稿写到工作区外：都拒。
	if _, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"title": "甲", "source_file": "tech.md", "outline_file": ".gaea/work/outline.json",
		"outline": []map[string]interface{}{{"level": 1, "title": "甲"}},
	})); err == nil || !strings.Contains(err.Error(), "只能给一个") {
		t.Errorf("outline 与 outline_file 同给应拒：%v", err)
	}
	if _, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"action": "detect_headings", "source_file": "tech.md", "title": "甲",
	})); err == nil || !strings.Contains(err.Error(), "只认标题") {
		t.Errorf("认标题带录入内容应拒：%v", err)
	}
	if _, err = save.Execute(ctx, toJSON(t, map[string]interface{}{
		"action": "detect_headings", "source_file": "tech.md", "outline_file": filepath.Join(t.TempDir(), "out.json"),
	})); err == nil {
		t.Error("outline 草稿写到工作区外应拒")
	}
}

func TestProposalSaveDisplayTitleReceipt(t *testing.T) {
	proposalTestEnv(t)
	dir := t.TempDir()
	src := "5.2.5.2 抑制您味\n采用生物滤池除臭。\n5.2.5.3 恶臭监测\n设置监测点。\n"
	if err := os.WriteFile(filepath.Join(dir, "ocr.md"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := (proposalSaveTool{workDir: dir}).Execute(context.Background(), toJSON(t, map[string]interface{}{
		"title": "OCR 方案", "source_file": "ocr.md", "finalize": true,
		"outline": []map[string]interface{}{
			{"level": 1, "title": "抑制异味", "display_title": "抑制异味"}, // 只给修正后的写法
			{"level": 1, "title": "恶臭监测"},                          // 不带 display_title：照旧沿用模型给的写法
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	msg := toolMessage(t, out)
	for _, want := range []string{"**标题修正**", "源文件「5.2.5.2 抑制您味」→ 入库「5.2.5.2 抑制异味」（第 1 行）"} {
		if !strings.Contains(msg, want) {
			t.Errorf("回执缺 %q：\n%s", want, msg)
		}
	}
	if strings.Count(msg, "源文件「") != 1 {
		t.Errorf("没带 display_title 的条目不该进修正明细：\n%s", msg)
	}
	st, err := proposalStore()
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.Resolve("OCR 方案")
	if err != nil || p == nil {
		t.Fatalf("Resolve: %v %+v", err, p)
	}
	secs, err := st.Outline(p.ID)
	if err != nil || len(secs) != 2 {
		t.Fatalf("Outline: %v %+v", err, secs)
	}
	if secs[0].Title != "5.2.5.2 抑制异味" || secs[1].Title != "恶臭监测" {
		t.Errorf("入库标题：%q、%q", secs[0].Title, secs[1].Title)
	}
	if strings.Contains(secs[0].Title, "您味") {
		t.Error("入库标题不该留着源文件的错字")
	}
}

func proposalSectionCount(t *testing.T, title string) int {
	t.Helper()
	st, err := proposalStore()
	if err != nil {
		t.Fatal(err)
	}
	p, err := st.Resolve(title)
	if err != nil || p == nil {
		t.Fatalf("Resolve %s: %v %+v", title, err, p)
	}
	secs, err := st.Outline(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return len(secs)
}
