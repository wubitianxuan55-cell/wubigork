package proposal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// index_test.go — 同题对照、索引体检 / 重建、整册导出的单测。

func TestTwinKey(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		same bool
	}{
		{"第1章 总体 › 1.1 部署 › 1.1.1 细则", "总体 › 部署 › 细则", true},
		{"1 项目实施方案", "项目实施方案", true},
		{"1.4.7.2 备选修复技术介绍", "备选修复技术介绍", true},
		{"第一章 施工 › （一）降水措施", "1 施工 › 1.1 降水措施", true},
		{"第1章 总体 › 3 号地块污染情况", "第1章 总体 › 4 号地块污染情况", false}, // 非首段：标题里的数字不是编号
		{"3 号地块污染情况", "4 号地块污染情况", true},                    // 首段的纯数字没法和章号区分：认成编号（多一个可点的提示，无害）
		{"1.1 降水措施", "1.2 降水措施", true},                      // 编号不同的同名节算同题（界面上逐节列出，标题照原样显示）
		{"第1章 总体 › 1.1 部署", "第1章 总体 › 1.1 安全", false},
	} {
		if got := twinKey(tc.a) == twinKey(tc.b); got != tc.same {
			t.Fatalf("twinKey(%q) == twinKey(%q) = %v, want %v（%q / %q）", tc.a, tc.b, got, tc.same, twinKey(tc.a), twinKey(tc.b))
		}
	}
}

func TestTwins(t *testing.T) {
	st := newStore(t)
	pid, pidBrief := seedWide(t, st) // 宽方案（数字编号）+ 监测简报（含「概述」）
	saveA(t, st, "同骨架方案", []saveSection{
		{1, "概述", "同骨架方案也有概述。"},
		{1, "第一章 施工", "施工总述。"}, {2, "（一）降水措施", "降水措施甲。"}, {2, "（二）降水措施", "降水措施乙。"},
		{1, "第二章 安全", "安全总述。"}, {2, "文明施工", "文明施工正文。"},
	})

	// 编号写法不同也算同题；本方案里的节点不算。
	twins, err := st.Twins(sectionIDByTitle(t, st, pid, "第2章 安全"))
	if err != nil || len(twins) != 1 || twins[0].ProposalName != "同骨架方案" || twins[0].Title != "第二章 安全" ||
		twins[0].TitlePath != "第二章 安全" || twins[0].Chars != 5 || twins[0].Level != 1 || twins[0].SectionID == 0 {
		t.Fatalf("章的同题：%+v err=%v", twins, err)
	}
	twins, err = st.Twins(sectionIDByTitle(t, st, pid, "2.1 文明施工"))
	if err != nil || len(twins) != 1 || twins[0].TitlePath != "第二章 安全 › 文明施工" || twins[0].Level != 2 {
		t.Fatalf("节的同题：%+v err=%v", twins, err)
	}
	// 几节同题时逐节列出；本方案里同名的兄弟节（1.2 / 1.3 降水措施）不算。
	twins, err = st.Twins(sectionIDByTitle(t, st, pid, "1.1 降水措施"))
	if err != nil || len(twins) != 2 || twins[0].Title != "（一）降水措施" || twins[1].Title != "（二）降水措施" {
		t.Fatalf("多节同题：%+v err=%v", twins, err)
	}
	twins, err = st.Twins(sectionIDByTitle(t, st, pidBrief, "概述"))
	if err != nil || len(twins) != 1 || twins[0].ProposalTitle != "同骨架方案" {
		t.Fatalf("跨方案同题：%+v err=%v", twins, err)
	}
	if _, err := st.Twins(99999); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("小节不存在应报错，得到 err=%v", err)
	}
	// 没有同题时为空，不报错。
	lone := saveA(t, st, "独一份", []saveSection{{1, "只此一家的标题", "正文。"}})
	if twins, err = st.Twins(sectionIDByTitle(t, st, lone.ProposalID, "只此一家的标题")); err != nil || len(twins) != 0 {
		t.Fatalf("没有同题应为空：%+v err=%v", twins, err)
	}
}

func TestIndexStatusAndReindex(t *testing.T) {
	st, _ := newAssetStore(t)
	pid := ingestMD(t, st)
	secs, _ := st.Outline(pid)
	ctx := context.Background()
	markVectorized(t, st)
	if before, err := st.IndexStatus(); err != nil || before.Broken() || before.VecDone != before.VecTotal {
		t.Fatalf("刚录入的库索引应无问题：%+v err=%v", before, err)
	}

	// 造出五类索引问题：孤儿全文索引行、缺全文索引行、全文索引内容过期、有正文没语义块、块前缀过期。
	if _, err := st.db.Exec(`INSERT INTO proposal_fts(rowid, title, body, proposal_id, section_id) VALUES(9999, '旧方案的一节', '孤儿行里的正文。', 9999, 9999)`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`DELETE FROM proposal_fts WHERE rowid=?`, secs[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`DELETE FROM proposal_chunks WHERE section_id=?`, secs[2].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`UPDATE proposal_sections SET title_path=? WHERE id=?`, "第1章 项目实施方案 › 改过的标题", secs[1].ID); err != nil {
		t.Fatal(err)
	}
	before, err := st.IndexStatus()
	if err != nil || before.OrphanFts != 1 || before.MissingFts != 1 || before.StaleFts != 1 || before.MissingChunks != 1 || before.StaleChunks == 0 || !before.Broken() {
		t.Fatalf("体检快照：%+v err=%v", before, err)
	}
	if res, err := st.Search(SearchInput{Query: "孤儿行里的正文"}); err != nil || res.Total != 0 {
		t.Fatalf("孤儿行不应被检索到：%+v err=%v", res, err)
	}

	// 重建：结构类问题清零，补出来的块也向量化。
	em := &fakeEmbedder{dim: 32}
	after, err := st.Reindex(ctx, em, "fake-embed")
	if err != nil || after.Broken() ||
		after.OrphanFts+after.MissingFts+after.StaleFts+after.MissingChunks+after.StaleChunks != 0 ||
		after.VecDone != after.VecTotal || after.Model != "fake-embed" {
		t.Fatalf("重建后：%+v err=%v", after, err)
	}
	if ftsRows(t, st) != len(secs) {
		t.Fatalf("重建后全文索引行数 = %d, want %d", ftsRows(t, st), len(secs))
	}
	if total, indexed := chunksOf(t, st, secs[2].ID); total == 0 || indexed != total {
		t.Fatalf("重建后应补出语义块并向量化：%d/%d", indexed, total)
	}
	var text string
	_ = st.db.QueryRow(`SELECT text FROM proposal_chunks WHERE section_id=? AND seq=0`, secs[1].ID).Scan(&text)
	if !strings.HasPrefix(text, "第1章 项目实施方案 › 改过的标题\n") {
		t.Fatalf("块前缀应换成新路径：%q", text)
	}
	_ = st.db.QueryRow(`SELECT body FROM proposal_fts WHERE rowid=?`, secs[1].ID).Scan(&text)
	if !strings.HasPrefix(text, "第1章 项目实施方案 › 改过的标题 ") {
		t.Fatalf("全文索引内容应换成新路径：%q", text)
	}
	if res, err := st.Search(SearchInput{Query: "本章说明总体部署"}); err != nil || res.Total != 1 || res.Hits[0].SectionID != secs[0].ID {
		t.Fatalf("缺的全文索引行应补回：%+v err=%v", res, err)
	}

	// 切了 embedding 模型：整库置空重嵌，索引记录换成新模型。
	if _, err := st.Reindex(ctx, em, "另一个模型"); err != nil {
		t.Fatal(err)
	}
	if model, _ := st.EmbedModel(); model != "另一个模型" {
		t.Fatalf("索引记录的模型 = %q", model)
	}
	if indexed, total := indexedChunks(t, st); indexed != total {
		t.Fatalf("换模型后应整库重嵌：%d/%d", indexed, total)
	}
	// 向量化中断：结构类修复照样落库，回执带剩余状态与原因。
	if err := st.ResetEmbedding("fake-embed"); err != nil {
		t.Fatal(err)
	}
	st2, err := st.Reindex(ctx, &fakeEmbedder{dim: 32, err: errors.New("down")}, "fake-embed")
	if err == nil || !strings.Contains(err.Error(), "向量化中断") || st2.Broken() || st2.VecDone != 0 {
		t.Fatalf("向量化中断：%+v err=%v", st2, err)
	}
	// 本地模型不可用：只做结构类修复，不报错（余下的向量留给检索时补）。
	st3, err := st.Reindex(ctx, nil, "")
	if err != nil || st3.Broken() || st3.VecDone == st3.VecTotal {
		t.Fatalf("无模型重建：%+v err=%v", st3, err)
	}
}

func TestExportMarkdown(t *testing.T) {
	st, _ := newAssetStore(t)
	sourceDir := t.TempDir()
	writeTinyPNG(t, filepath.Join(sourceDir, "flow.png"))
	saveA(t, st, "图片方案", []saveSection{
		{1, "第一章 工艺", "工艺路线如下：\n\n![工艺流程图](flow.png)"},
		{2, "1.1 细则", "细则正文。"},
	}, func(in *SaveInput) { in.AssetBaseDir = sourceDir })

	out := filepath.Join(t.TempDir(), "导出", "图片方案.md")
	res, err := st.Export("图片方案", out)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	md := string(raw)
	for _, want := range []string{"> 方案库导出：图片方案（技术标 / 2026）", "# 第一章 工艺", "## 1.1 细则", "细则正文。", "](图片方案-assets/"} {
		if !strings.Contains(md, want) {
			t.Fatalf("导出内容缺 %q：\n%s", want, md)
		}
	}
	if strings.Contains(md, AssetURIPrefix) {
		t.Fatalf("托管引用应改成相对路径：\n%s", md)
	}
	assetsDir := filepath.Join(filepath.Dir(out), "图片方案-assets")
	if res.Sections != 2 || res.Chars <= 0 || res.Images != 1 || res.Missing != 0 || res.AssetsDir != assetsDir || res.Path != out || res.Title != "图片方案" {
		t.Fatalf("导出回执：%+v", res)
	}
	if ents, err := os.ReadDir(assetsDir); err != nil || len(ents) != 1 {
		t.Fatalf("图片文件应落到 -assets 目录：%v %v", ents, err)
	}

	if _, err := st.Export("图片方案", filepath.Join(t.TempDir(), "x.docx")); err == nil || !strings.Contains(err.Error(), ".md/.markdown/.txt") {
		t.Fatalf("扩展名不符应报错，得到 err=%v", err)
	}
	if _, err := st.Export("没有的方案", filepath.Join(t.TempDir(), "x.md")); err == nil || !strings.Contains(err.Error(), "没有《") {
		t.Fatalf("方案不存在应报错，得到 err=%v", err)
	}
	// 元信息缺项时表头用破折号；没有图片时不建 -assets 目录。
	saveA(t, st, "裸方案", []saveSection{{1, "一 概述", "正文。"}}, func(in *SaveInput) { in.Meta.ProjectType, in.Meta.Year = "", "" })
	bare := filepath.Join(t.TempDir(), "裸方案.md")
	res, err = st.Export("裸方案", bare)
	if err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(bare); !strings.HasPrefix(string(raw), "> 方案库导出：裸方案（— / —）") {
		t.Fatalf("缺项表头：%s", raw)
	}
	if res.AssetsDir != "" || res.Images != 0 {
		t.Fatalf("无图片的导出回执：%+v", res)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(bare), "裸方案-assets")); err == nil {
		t.Fatal("没有图片不应建 -assets 目录")
	}
}
