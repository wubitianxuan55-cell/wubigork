package proposal

import (
	"strings"
	"testing"
)

// maintain_test.go — 维护面单测：元数据修改、小节增量修正（含子树路径 / 全文索引 /
// 语义块同步）、整册导出取数、录入演练不落库。

func seedMaint(t *testing.T, st *Store) int64 {
	t.Helper()
	res, err := st.Save(SaveInput{
		Meta: Proposal{Title: "甲方案", ProjectType: "技术标", Year: "2026", Workspace: `D:\新工作\甲`, Summary: "摘要"},
		Sections: []Section{
			{Level: 1, Title: "第1章 总体", Body: "第一章正文。"},
			{Level: 2, Title: "1.1 部署", Body: "部署正文，降水井布置。"},
			{Level: 3, Title: "1.1.1 细则", Body: "细则正文。"},
			{Level: 2, Title: "1.2 安全", Body: "安全正文。"},
		},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return res.ProposalID
}

func sectionIDByTitle(t *testing.T, st *Store, pid int64, title string) int64 {
	t.Helper()
	secs, err := st.Outline(pid)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range secs {
		if s.Title == title {
			return s.ID
		}
	}
	t.Fatalf("没有标题为 %q 的小节", title)
	return 0
}

func strp(s string) *string { return &s }

func TestUpdateMeta(t *testing.T) {
	st := newStore(t)
	pid := seedMaint(t, st)
	if _, err := st.Save(SaveInput{Meta: Proposal{Title: "丙方案"}, Sections: []Section{{Level: 1, Title: "一 概述", Body: "正文。"}}}); err != nil {
		t.Fatal(err)
	}

	got, err := st.UpdateMeta("甲方案", UpdateInput{Title: strp(" 乙方案 "), Year: strp("2027"), Summary: strp("")})
	if err != nil {
		t.Fatalf("UpdateMeta: %v", err)
	}
	if got.Title != "乙方案" || got.Name != "乙方案" || got.Year != "2027" || got.ProjectType != "技术标" || got.Summary != "" || got.Workspace != `D:\新工作\甲` {
		t.Fatalf("改动不符： %+v", got)
	}
	if p, _ := st.Get("甲方案"); p != nil {
		t.Fatal("旧方案名不应再查到")
	}
	if p, _ := st.Get("乙方案"); p == nil || p.ID != pid {
		t.Fatal("新方案名应查到同一份方案")
	}

	if _, err := st.UpdateMeta("丙方案", UpdateInput{Title: strp("乙方案")}); err == nil || !strings.Contains(err.Error(), "撞名") {
		t.Fatalf("改名撞到已有方案应报错，得到 err=%v", err)
	}
	if _, err := st.UpdateMeta("丙方案", UpdateInput{Title: strp("   ")}); err == nil {
		t.Fatal("空标题应报错")
	}
	if _, err := st.UpdateMeta("不存在", UpdateInput{Year: strp("2027")}); err == nil || !strings.Contains(err.Error(), "没有《不存在》") {
		t.Fatalf("方案不存在应报错，得到 err=%v", err)
	}
	if list, _ := st.List(); len(list) != 2 {
		t.Fatalf("改名不增不减，len=%d", len(list))
	}
}

func TestUpdateSectionRetitlePropagatesToSubtree(t *testing.T) {
	st := newStore(t)
	pid := seedMaint(t, st)
	sid := sectionIDByTitle(t, st, pid, "1.1 部署")
	leaf := sectionIDByTitle(t, st, pid, "1.1.1 细则")
	sibling := sectionIDByTitle(t, st, pid, "1.2 安全")
	if _, err := st.db.Exec(`UPDATE proposal_chunks SET vec=x'00' WHERE section_id IN (?, ?)`, sid, leaf); err != nil {
		t.Fatal(err)
	}

	affected, err := st.UpdateSection(sid, SectionEdit{Title: strp(" 1.1 调度安排 ")})
	if err != nil {
		t.Fatalf("UpdateSection: %v", err)
	}
	if len(affected) != 2 || affected[0].Title != "1.1 调度安排" || affected[1].Title != "1.1.1 细则" {
		t.Fatalf("受影响的小节不符：%+v", affected)
	}
	secs, _ := st.Outline(pid)
	want := []string{"第1章 总体", "第1章 总体 › 1.1 调度安排", "第1章 总体 › 1.1 调度安排 › 1.1.1 细则", "第1章 总体 › 1.2 安全"}
	for i, w := range want {
		if secs[i].TitlePath != w {
			t.Fatalf("第 %d 节路径 = %q, want %q", i, secs[i].TitlePath, w)
		}
	}
	// 全文索引跟着改：新标题查得到、路径是新路径；子节因路径含新标题一并命中
	res, err := st.Search(SearchInput{Query: "调度安排", Scope: ScopeTitle})
	if err != nil || len(res.Hits) != 2 || res.Hits[0].SectionID != sid || res.Hits[0].TitlePath != want[1] || res.Hits[1].TitlePath != want[2] {
		t.Fatalf("改标题后标题检索应命中本节与子节：res=%+v err=%v", res, err)
	}
	// 语义块：前缀换成新路径，向量置空待重嵌
	var text string
	var vec any
	if err := st.db.QueryRow(`SELECT text, vec FROM proposal_chunks WHERE section_id=?`, leaf).Scan(&text, &vec); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, want[2]+"\n") || vec != nil {
		t.Fatalf("子节语义块应换新路径前缀且向量为空：text=%q vec=%v", text, vec)
	}
	var siblingVec any
	if err := st.db.QueryRow(`SELECT vec FROM proposal_chunks WHERE section_id=?`, sibling).Scan(&siblingVec); err != nil {
		t.Fatal(err)
	}
	_ = siblingVec
	// 改路径不改层级与顺序
	if secs[1].Level != 2 || secs[2].Level != 3 || secs[3].TitlePath != want[3] {
		t.Fatalf("层级/兄弟节被误动：%+v", secs)
	}
}

func TestUpdateSectionReplaceBody(t *testing.T) {
	st := newStore(t)
	pid := seedMaint(t, st)
	sid := sectionIDByTitle(t, st, pid, "1.2 安全")

	affected, err := st.UpdateSection(sid, SectionEdit{Body: strp("安全文明施工措施。\r\n第二行。")})
	if err != nil {
		t.Fatalf("UpdateSection: %v", err)
	}
	if len(affected) != 1 || affected[0].Title != "1.2 安全" {
		t.Fatalf("只替换正文时不应动别的节：%+v", affected)
	}
	r, err := st.Read(sid, ReadOpt{Limit: ReadAll})
	if err != nil || r.Text != "安全文明施工措施。\n第二行。" {
		t.Fatalf("正文未替换：text=%q err=%v", r.Text, err)
	}
	if res, _ := st.Search(SearchInput{Query: "安全文明"}); len(res.Hits) != 1 || res.Hits[0].SectionID != sid {
		t.Fatalf("新正文应可检索：res=%+v", res)
	}
	if res, _ := st.Search(SearchInput{Query: "安全正文"}); len(res.Hits) != 0 {
		t.Fatalf("旧正文不应再命中：res=%+v", res)
	}
	if pending, total := st.PendingCount(pid); pending != total {
		t.Fatalf("替换后的语义块应待重嵌：pending=%d total=%d", pending, total)
	}

	tooLong := strings.Repeat("字", MaxBodyChars+1)
	if _, err := st.UpdateSection(sid, SectionEdit{Body: &tooLong}); err == nil || !strings.Contains(err.Error(), "单节上限") {
		t.Fatalf("超长正文应拒绝，得到 err=%v", err)
	}
	if _, err := st.UpdateSection(sid, SectionEdit{}); err == nil {
		t.Fatal("没有改动项应报错")
	}
	if _, err := st.UpdateSection(99999, SectionEdit{Body: strp("x")}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("小节不存在应报错，得到 err=%v", err)
	}
}

func TestFullTextAndAssetPath(t *testing.T) {
	st := newStore(t)
	pid := seedMaint(t, st)
	p, secs, err := st.FullText("甲方案")
	if err != nil || p.ID != pid || len(secs) != 4 || secs[1].Body != "部署正文，降水井布置。" || secs[3].Chars != 5 {
		t.Fatalf("FullText: p=%+v secs=%+v err=%v", p, secs, err)
	}
	if _, _, err := st.FullText("不存在"); err == nil {
		t.Fatal("方案不存在应报错")
	}
	if _, err := st.AssetPath("gaea-asset://" + strings.Repeat("ab", 32)); err == nil {
		t.Fatal("不存在的托管图片应报错")
	}
	if _, err := st.AssetPath("gaea-asset://not-a-hash"); err == nil {
		t.Fatal("非法引用应报错")
	}
}

func TestSaveDryRunWritesNothing(t *testing.T) {
	st := newStore(t)
	in := SaveInput{
		Meta:     Proposal{Title: "演练方案"},
		DryRun:   true,
		Sections: []Section{{Level: 1, Title: "一 概述", Body: "正文。"}, {Level: 2, Title: "1.1 细则", Body: "细则。"}},
	}
	res, err := st.Save(in)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !res.DryRun || !res.Created || res.Added != 2 || res.Stats.Sections != 2 || res.Stats.LevelCounts[2] != 1 {
		t.Fatalf("演练回执应给出「假如写入」的结构： %+v", res)
	}
	if list, _ := st.List(); len(list) != 0 {
		t.Fatalf("演练不得写库，List=%+v", list)
	}

	// 演练里的校验是真的：跳级照样拦
	bad := SaveInput{Meta: Proposal{Title: "演练方案"}, DryRun: true, Sections: []Section{{Level: 1, Title: "一", Body: "a"}, {Level: 3, Title: "跳级", Body: "b"}}}
	if _, err := st.Save(bad); err == nil || !strings.Contains(err.Error(), "跳级") {
		t.Fatalf("演练应拦住跳级，得到 err=%v", err)
	}

	in.DryRun = true
	if _, err := st.Save(in); err != nil {
		t.Fatalf("再次演练: %v", err)
	}
	if list, _ := st.List(); len(list) != 0 {
		t.Fatalf("演练仍不得写库，List=%+v", list)
	}
	// 正式写入
	in.DryRun = false
	real, err := st.Save(in)
	if err != nil || real.DryRun || !real.Created {
		t.Fatalf("正式写入: res=%+v err=%v", real, err)
	}
	// 已存在的方案上演练整册覆盖：库不变，回执按「覆盖后」统计
	over := SaveInput{Meta: Proposal{Title: "演练方案"}, DryRun: true, Overwrite: true, Sections: []Section{{Level: 1, Title: "新结构", Body: "x"}}}
	ov, err := st.Save(over)
	if err != nil || !ov.DryRun || !ov.Replaced || ov.Stats.Sections != 1 {
		t.Fatalf("覆盖演练: res=%+v err=%v", ov, err)
	}
	if p, _ := st.Resolve("演练方案"); p == nil {
		t.Fatal("方案应仍在")
	} else if secs, _ := st.Outline(p.ID); len(secs) != 2 {
		t.Fatalf("覆盖演练不得动库，节数=%d", len(secs))
	}
}
