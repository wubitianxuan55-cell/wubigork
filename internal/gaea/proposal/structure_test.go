package proposal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// structure_test.go — 结构修正单测：升降级、同级换位、拆分、并入、增删小节；
// 每一步核对标题树（层级 / 路径 / 顺序）、全文索引、语义块与向量置空。

func seedTree(t *testing.T, st *Store) int64 {
	t.Helper()
	res, err := st.Save(SaveInput{
		Meta: Proposal{Title: "结构测试方案", ProjectType: "技术标", Year: "2026"},
		Sections: []Section{
			{Level: 1, Title: "第1章 总体", Body: "第一章正文。"},
			{Level: 2, Title: "1.1 部署", Body: "部署正文，降水井布置。"},
			{Level: 3, Title: "1.1.1 细则", Body: "细则正文甲。"},
			{Level: 3, Title: "1.1.2 补充", Body: "补充正文乙。"},
			{Level: 2, Title: "1.2 安全", Body: "安全正文。\n\n第二段讲围挡。\n第三段讲警示。"},
			{Level: 1, Title: "第2章 保障", Body: "第二章正文。"},
			{Level: 2, Title: "2.1 质量保证", Body: "质量正文。"},
		},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return res.ProposalID
}

// wantTree 逐行核对标题树：标题、层级、路径。
func wantTree(t *testing.T, st *Store, pid int64, want [][]any) []Section {
	t.Helper()
	secs, err := st.Outline(pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != len(want) {
		t.Fatalf("小节数 = %d, want %d：%+v", len(secs), len(want), secs)
	}
	for i, w := range want {
		if secs[i].Title != w[0] || secs[i].Level != w[1] || secs[i].TitlePath != w[2] || secs[i].Seq != i {
			t.Fatalf("第 %d 行 = {%q L%d %q seq=%d}, want %v", i, secs[i].Title, secs[i].Level, secs[i].TitlePath, secs[i].Seq, w)
		}
	}
	for i, s := range secs { // 父节点 = 前面最近的更高级标题
		var parent int64
		for k := i - 1; k >= 0; k-- {
			if secs[k].Level < s.Level {
				parent = secs[k].ID
				break
			}
		}
		if s.ParentID != parent {
			t.Fatalf("「%s」parent = %d, want %d", s.Title, s.ParentID, parent)
		}
	}
	return secs
}

func readBody(t *testing.T, st *Store, id int64) string {
	t.Helper()
	r, err := st.Read(id, ReadOpt{Limit: ReadAll})
	if err != nil {
		t.Fatal(err)
	}
	return r.Text
}

func markVectorized(t *testing.T, st *Store) {
	t.Helper()
	if _, err := st.db.Exec(`UPDATE proposal_chunks SET vec=x'00' WHERE vec IS NULL`); err != nil {
		t.Fatal(err)
	}
}

// chunksOf 返回某小节的（块数，已向量化块数）。
func chunksOf(t *testing.T, st *Store, id int64) (total, indexed int) {
	t.Helper()
	if err := st.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(vec IS NOT NULL),0) FROM proposal_chunks WHERE section_id=?`, id).Scan(&total, &indexed); err != nil {
		t.Fatal(err)
	}
	return total, indexed
}

func indexedChunks(t *testing.T, st *Store) (indexed, total int) {
	t.Helper()
	if err := st.db.QueryRow(`SELECT COALESCE(SUM(vec IS NOT NULL),0), COUNT(*) FROM proposal_chunks`).Scan(&indexed, &total); err != nil {
		t.Fatal(err)
	}
	return indexed, total
}

func ftsRows(t *testing.T, st *Store) int {
	t.Helper()
	var n int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM proposal_fts`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func searchTitles(t *testing.T, st *Store, q string) []Hit {
	t.Helper()
	res, err := st.Search(SearchInput{Query: q, Scope: ScopeTitle, TopN: 30})
	if err != nil {
		t.Fatal(err)
	}
	return res.Hits
}

func TestRestructurePromoteDemote(t *testing.T) {
	st := newStore(t)
	pid := seedTree(t, st)
	dep := sectionIDByTitle(t, st, pid, "1.1 部署")
	safe := sectionIDByTitle(t, st, pid, "1.2 安全")
	markVectorized(t, st)

	// 升级 1.2 安全：变 1 级，紧跟在第1章之后；它原本的上级标题不动。
	res, err := st.Restructure(safe, StructureInput{Op: OpPromote})
	if err != nil || res.Created != 0 || res.FocusID != safe || res.Affected != 1 {
		t.Fatalf("升级回执：%+v err=%v", res, err)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"},
		{"1.1 部署", 2, "第1章 总体 › 1.1 部署"},
		{"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"},
		{"1.2 安全", 1, "1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"},
		{"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	// 全文索引与语义块跟着换路径：向量置空待重嵌；没动的小节向量还在。
	if hits := searchTitles(t, st, "安全"); len(hits) != 1 || hits[0].SectionID != safe || hits[0].TitlePath != "1.2 安全" {
		t.Fatalf("升级后标题检索：%+v", hits)
	}
	var text string
	_ = st.db.QueryRow(`SELECT text FROM proposal_chunks WHERE section_id=? AND seq=0`, safe).Scan(&text)
	if !strings.HasPrefix(text, "1.2 安全\n安全正文。") {
		t.Fatalf("语义块前缀应换成新路径：%q", text)
	}
	if total, indexed := chunksOf(t, st, safe); total == 0 || indexed != 0 {
		t.Fatalf("升级的小节向量应置空：%d/%d", indexed, total)
	}
	if total, indexed := chunksOf(t, st, dep); total == 0 || indexed != total {
		t.Fatalf("没动的小节向量不应被置空：%d/%d", indexed, total)
	}

	// 降级 第2章 保障：归到它前面的同级小节（刚升上来的 1.2 安全）名下，位置不变；
	// 子树跟着走一层。
	res, err = st.Restructure(sectionIDByTitle(t, st, pid, "第2章 保障"), StructureInput{Op: OpDemote})
	if err != nil || res.FocusID == 0 {
		t.Fatalf("降级回执：%+v err=%v", res, err)
	}
	ch2 := res.FocusID
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"},
		{"1.1 部署", 2, "第1章 总体 › 1.1 部署"},
		{"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"},
		{"1.2 安全", 1, "1.2 安全"},
		{"第2章 保障", 2, "1.2 安全 › 第2章 保障"},
		{"2.1 质量保证", 3, "1.2 安全 › 第2章 保障 › 2.1 质量保证"},
	})
	// 它已是本层最后一个：不能再下移；2.1 质量保证前面没有同级小节：不能再降级。
	if _, err := st.Restructure(ch2, StructureInput{Op: OpMoveDown}); err == nil || !strings.Contains(err.Error(), "最后一个小节") {
		t.Fatalf("降级后的边界应报错，得到 err=%v", err)
	}

	// 降级后再升级：回到原位、原层级。
	st = newStore(t)
	pid = seedTree(t, st)
	safe = sectionIDByTitle(t, st, pid, "1.2 安全")
	if _, err := st.Restructure(safe, StructureInput{Op: OpDemote}); err != nil {
		t.Fatal(err)
	}
	if hits := searchTitles(t, st, "安全"); len(hits) != 1 || hits[0].TitlePath != "第1章 总体 › 1.1 部署 › 1.2 安全" {
		t.Fatalf("降级后路径：%+v", hits)
	}
	if _, err := st.Restructure(safe, StructureInput{Op: OpPromote}); err != nil {
		t.Fatal(err)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"},
		{"1.1 部署", 2, "第1章 总体 › 1.1 部署"},
		{"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"},
		{"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"},
		{"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})

	// 拒绝：1 级不能再升；本层第一个小节降级后无处挂靠；4 级不能再降。
	ch1 := sectionIDByTitle(t, st, pid, "第1章 总体")
	dep = sectionIDByTitle(t, st, pid, "1.1 部署")
	if _, err := st.Restructure(ch1, StructureInput{Op: OpPromote}); err == nil || !strings.Contains(err.Error(), "1 级") {
		t.Fatalf("1 级标题升级应报错，得到 err=%v", err)
	}
	if _, err := st.Restructure(dep, StructureInput{Op: OpDemote}); err == nil || !strings.Contains(err.Error(), "前面没有同级小节") {
		t.Fatalf("第一个子节降级应报错，得到 err=%v", err)
	}
	leaf := sectionIDByTitle(t, st, pid, "1.1.2 补充")
	res, err = st.Restructure(leaf, StructureInput{Op: OpAddChild, Title: "四级小节"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Restructure(res.FocusID, StructureInput{Op: OpDemote}); err == nil || !strings.Contains(err.Error(), "4 级") {
		t.Fatalf("4 级标题降级应报错，得到 err=%v", err)
	}
}

func TestRestructureMoveSiblings(t *testing.T) {
	st := newStore(t)
	pid := seedTree(t, st)
	ch1 := sectionIDByTitle(t, st, pid, "第1章 总体")
	dep := sectionIDByTitle(t, st, pid, "1.1 部署")
	safe := sectionIDByTitle(t, st, pid, "1.2 安全")
	qual := sectionIDByTitle(t, st, pid, "2.1 质量保证")
	markVectorized(t, st)

	// 上移 1.2 安全：排到 1.1 部署（连同它的子节）之前。
	res, err := st.Restructure(safe, StructureInput{Op: OpMoveUp})
	if err != nil || res.Affected != 4 || res.FocusID != safe {
		t.Fatalf("上移回执：%+v err=%v", res, err)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"},
		{"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"1.1 部署", 2, "第1章 总体 › 1.1 部署"},
		{"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"},
		{"第2章 保障", 1, "第2章 保障"},
		{"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	// 换位不改路径：全文索引与向量都不动。
	if indexed, total := indexedChunks(t, st); indexed != total {
		t.Fatalf("换位不应置空向量：%d/%d", indexed, total)
	}

	// 下移 1.2 安全（现在排在最前）：排到 1.1 部署（连同子节）之后，回到原位。
	if _, err := st.Restructure(safe, StructureInput{Op: OpMoveDown}); err != nil {
		t.Fatal(err)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"},
		{"1.1 部署", 2, "第1章 总体 › 1.1 部署"},
		{"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"},
		{"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"},
		{"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})

	// 边界：本层第一个不能再上移，最后一个不能再下移；顶层标题也一样。
	for _, tc := range []struct {
		id  int64
		op  string
		msg string
	}{
		{dep, OpMoveUp, "第一个小节"}, {ch1, OpMoveUp, "第一个小节"},
		{qual, OpMoveDown, "最后一个小节"}, {sectionIDByTitle(t, st, pid, "第2章 保障"), OpMoveDown, "最后一个小节"},
	} {
		if _, err := st.Restructure(tc.id, StructureInput{Op: tc.op}); err == nil || !strings.Contains(err.Error(), tc.msg) {
			t.Fatalf("%s 应报「%s」，得到 err=%v", tc.op, tc.msg, err)
		}
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.1 部署", 2, "第1章 总体 › 1.1 部署"}, {"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"}, {"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
}

func TestRestructureSplitSection(t *testing.T) {
	st := newStore(t)
	pid := seedTree(t, st)
	safe := sectionIDByTitle(t, st, pid, "1.2 安全")

	// 光标在「第二段」里：该行及之后切成新子节，放在本节后面；标题折成一行。
	res, err := st.Restructure(safe, StructureInput{Op: OpSplit, Offset: 9, Title: " 围挡与\n警示 "})
	if err != nil || res.Created != 1 || res.FocusID == safe || res.Removed != 0 {
		t.Fatalf("拆分回执：%+v err=%v", res, err)
	}
	secs := wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.1 部署", 2, "第1章 总体 › 1.1 部署"}, {"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"}, {"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"围挡与 警示", 3, "第1章 总体 › 1.2 安全 › 围挡与 警示"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	if secs[5].ParentID != safe || secs[5].SrcLine != 0 {
		t.Fatalf("新子节应挂在本节名下：%+v", secs[5])
	}
	if b := readBody(t, st, safe); b != "安全正文。" {
		t.Fatalf("本节正文 = %q", b)
	}
	if b := readBody(t, st, secs[5].ID); b != "第二段讲围挡。\n第三段讲警示。" {
		t.Fatalf("新子节正文 = %q", b)
	}
	// 新旧两节各有自己的全文索引与语义块。
	if hits := searchTitles(t, st, "警示"); len(hits) != 1 || hits[0].SectionID != secs[5].ID {
		t.Fatalf("新子节应能按标题查到：%+v", hits)
	}
	if res, err := st.Search(SearchInput{Query: "围挡"}); err != nil || len(res.Hits) != 1 || res.Hits[0].SectionID != secs[5].ID {
		t.Fatalf("拆出去的正文应在新小节里：%+v err=%v", res, err)
	}
	if total, _ := chunksOf(t, st, secs[5].ID); total == 0 {
		t.Fatal("新小节应已切出语义块")
	}
	if ftsRows(t, st) != len(secs) {
		t.Fatalf("全文索引行数 = %d, want %d", ftsRows(t, st), len(secs))
	}

	// 拒绝：光标在第一行、标题为空、光标之后没有正文、4 级不能再拆。
	for _, tc := range []struct {
		in  StructureInput
		msg string
	}{
		{StructureInput{Op: OpSplit, Offset: 0, Title: "x"}, "第一行"},
		{StructureInput{Op: OpSplit, Offset: 2, Title: "x"}, "第一行"},
		{StructureInput{Op: OpSplit, Offset: 9, Title: "  "}, "标题不能为空"},
	} {
		if _, err := st.Restructure(safe, tc.in); err == nil || !strings.Contains(err.Error(), tc.msg) {
			t.Fatalf("拆分 %+v 应报「%s」，得到 err=%v", tc.in, tc.msg, err)
		}
	}
	wantTree(t, st, pid, [][]any{ // 被拒的拆分一行都没动
		{"第1章 总体", 1, "第1章 总体"}, {"1.1 部署", 2, "第1章 总体 › 1.1 部署"}, {"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"}, {"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"围挡与 警示", 3, "第1章 总体 › 1.2 安全 › 围挡与 警示"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	// 正文末尾只有空行时（手改过的小节），光标在末尾没有正文可拆。
	padded := sectionIDByTitle(t, st, pid, "围挡与 警示")
	if _, err := st.UpdateSection(padded, SectionEdit{Body: strp("第二段讲围挡。\n\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Restructure(padded, StructureInput{Op: OpSplit, Offset: 1 << 30, Title: "x"}); err == nil || !strings.Contains(err.Error(), "没有正文") {
		t.Fatalf("光标之后无正文应报错，得到 err=%v", err)
	}
	leaf := sectionIDByTitle(t, st, pid, "1.1.1 细则")
	l4, err := st.Restructure(leaf, StructureInput{Op: OpAddChild, Title: "四级小节"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Restructure(l4.FocusID, StructureInput{Op: OpSplit, Offset: 1, Title: "x"}); err == nil || !strings.Contains(err.Error(), "4 级") {
		t.Fatalf("4 级标题拆分应报错，得到 err=%v", err)
	}
}

func TestRestructureMergeSections(t *testing.T) {
	st := newStore(t)
	pid := seedTree(t, st)
	ch1 := sectionIDByTitle(t, st, pid, "第1章 总体")
	dep := sectionIDByTitle(t, st, pid, "1.1 部署")
	safe := sectionIDByTitle(t, st, pid, "1.2 安全")

	// 并入父节：只许第一个子节（阅读顺序才不变）。
	if _, err := st.Restructure(safe, StructureInput{Op: OpMergeParent}); err == nil || !strings.Contains(err.Error(), "第一个小节") {
		t.Fatalf("非第一个子节并入父节应报错，得到 err=%v", err)
	}
	if _, err := st.Restructure(ch1, StructureInput{Op: OpMergeParent}); err == nil || !strings.Contains(err.Error(), "1 级") {
		t.Fatalf("1 级标题并入父节应报错，得到 err=%v", err)
	}
	res, err := st.Restructure(dep, StructureInput{Op: OpMergeParent})
	if err != nil || res.FocusID != ch1 || res.Removed != 1 || res.RemovedChars != 0 {
		t.Fatalf("并入父节回执：%+v err=%v", res, err)
	}
	secs := wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"},
		{"1.1.1 细则", 2, "第1章 总体 › 1.1.1 细则"},
		{"1.1.2 补充", 2, "第1章 总体 › 1.1.2 补充"},
		{"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"},
		{"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	if b := readBody(t, st, ch1); b != "第一章正文。\n\n部署正文，降水井布置。" {
		t.Fatalf("父节正文 = %q", b)
	}
	// 正文搬家后索引跟着走：全库只有父节命中「降水井」，全文索引不多不少一行。
	if res, err := st.Search(SearchInput{Query: "降水井"}); err != nil || len(res.Hits) != 1 || res.Hits[0].SectionID != ch1 {
		t.Fatalf("并入后正文检索：%+v err=%v", res, err)
	}
	if hits := searchTitles(t, st, "细则"); len(hits) != 1 || hits[0].TitlePath != "第1章 总体 › 1.1.1 细则" {
		t.Fatalf("升上来的子节路径：%+v", hits)
	}
	if ftsRows(t, st) != len(secs) {
		t.Fatalf("全文索引行数 = %d, want %d", ftsRows(t, st), len(secs))
	}
	if total, _ := chunksOf(t, st, dep); total != 0 {
		t.Fatalf("被并入的小节语义块应清除，剩 %d", total)
	}

	// 并入上一同级小节：正文接在它后面；上一节原有的子节不动。
	st = newStore(t)
	pid = seedTree(t, st)
	ch1 = sectionIDByTitle(t, st, pid, "第1章 总体")
	dep = sectionIDByTitle(t, st, pid, "1.1 部署")
	safe = sectionIDByTitle(t, st, pid, "1.2 安全")
	res, err = st.Restructure(safe, StructureInput{Op: OpMergePrev})
	if err != nil || res.FocusID != dep {
		t.Fatalf("并入上一同级回执：%+v err=%v", res, err)
	}
	if b := readBody(t, st, dep); b != "部署正文，降水井布置。\n\n安全正文。\n\n第二段讲围挡。\n第三段讲警示。" {
		t.Fatalf("上一同级小节正文 = %q", b)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.1 部署", 2, "第1章 总体 › 1.1 部署"}, {"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"}, {"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	// 1.1 部署升成 1 级（前面是第1章 总体），再并入第1章：它的两个子节原层级归到第1章名下。
	if _, err := st.Restructure(dep, StructureInput{Op: OpPromote}); err != nil {
		t.Fatal(err)
	}
	res, err = st.Restructure(dep, StructureInput{Op: OpMergePrev})
	if err != nil || res.FocusID != ch1 {
		t.Fatalf("带子节并入回执：%+v err=%v", res, err)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.1.1 细则", 2, "第1章 总体 › 1.1.1 细则"}, {"1.1.2 补充", 2, "第1章 总体 › 1.1.2 补充"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	if b := readBody(t, st, ch1); !strings.HasPrefix(b, "第一章正文。") || !strings.Contains(b, "降水井布置。") {
		t.Fatalf("父节正文 = %q", b)
	}
	if _, err := st.Restructure(sectionIDByTitle(t, st, pid, "2.1 质量保证"), StructureInput{Op: OpMergePrev}); err == nil || !strings.Contains(err.Error(), "前面没有同级小节") {
		t.Fatalf("第一个子节并入上一同级应报错，得到 err=%v", err)
	}
}

func TestRestructureAddRemoveAndGuards(t *testing.T) {
	st := newStore(t)
	pid := seedTree(t, st)
	ch1 := sectionIDByTitle(t, st, pid, "第1章 总体")
	dep := sectionIDByTitle(t, st, pid, "1.1 部署")

	// 新增子节落在子树末尾；新增同级小节紧跟其后。
	res, err := st.Restructure(dep, StructureInput{Op: OpAddChild, Title: "1.1.3 新增", Body: "新增正文讲复勘。"})
	if err != nil || res.Created != 1 || res.FocusID == dep {
		t.Fatalf("新增子节回执：%+v err=%v", res, err)
	}
	added := res.FocusID
	res, err = st.Restructure(added, StructureInput{Op: OpAddSibling, Title: "1.1.4 同级"})
	if err != nil || res.Created != 1 {
		t.Fatalf("新增同级回执：%+v err=%v", res, err)
	}
	secs := wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.1 部署", 2, "第1章 总体 › 1.1 部署"}, {"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"}, {"1.1.3 新增", 3, "第1章 总体 › 1.1 部署 › 1.1.3 新增"},
		{"1.1.4 同级", 3, "第1章 总体 › 1.1 部署 › 1.1.4 同级"}, {"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	if res, err := st.Search(SearchInput{Query: "复勘"}); err != nil || len(res.Hits) != 1 || res.Hits[0].SectionID != added {
		t.Fatalf("新增小节应可检索：%+v err=%v", res, err)
	}
	if ftsRows(t, st) != len(secs) {
		t.Fatalf("全文索引行数 = %d, want %d", ftsRows(t, st), len(secs))
	}
	if _, err := st.Restructure(added, StructureInput{Op: OpAddChild, Title: "x"}); err != nil {
		t.Fatal(err)
	}
	fourth, err := st.Restructure(sectionIDByTitle(t, st, pid, "1.1.4 同级"), StructureInput{Op: OpAddChild, Title: "四级小节"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Restructure(fourth.FocusID, StructureInput{Op: OpAddChild, Title: "x"}); err == nil || !strings.Contains(err.Error(), "4 级") {
		t.Fatalf("4 级标题加子节应报错，得到 err=%v", err)
	}

	// 删除 1.1 部署：连同 6 个子孙一起清；它是父节的第一个子节，落到父节。
	res, err = st.Restructure(dep, StructureInput{Op: OpRemove})
	if err != nil || res.Removed != 7 || res.Created != 0 || res.FocusID != ch1 || res.RemovedChars != 31 {
		t.Fatalf("删除回执：%+v err=%v", res, err)
	}
	secs = wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
	if ftsRows(t, st) != len(secs) {
		t.Fatalf("删除后全文索引行数 = %d, want %d（被删小节的索引行应一并清除）", ftsRows(t, st), len(secs))
	}
	var n int
	_ = st.db.QueryRow(`SELECT COUNT(*) FROM proposal_chunks WHERE section_id=?`, dep).Scan(&n)
	if n != 0 {
		t.Fatalf("被删小节的语义块剩 %d", n)
	}
	if res, err := st.Search(SearchInput{Query: "细则正文"}); err != nil || len(res.Hits) != 0 {
		t.Fatalf("删掉的小节不应再被检索到：%+v err=%v", res, err)
	}
	// 顶层标题删掉后落到下一个顶层；整册删空返回 0。
	safe := sectionIDByTitle(t, st, pid, "1.2 安全")
	res, err = st.Restructure(safe, StructureInput{Op: OpRemove})
	if err != nil || res.FocusID != ch1 {
		t.Fatalf("删除回执：%+v err=%v", res, err)
	}
	res, err = st.Restructure(ch1, StructureInput{Op: OpRemove})
	if err != nil || res.FocusID != sectionIDByTitle(t, st, pid, "第2章 保障") {
		t.Fatalf("删除首个顶层标题回执：%+v err=%v", res, err)
	}
	if _, err := st.Restructure(sectionIDByTitle(t, st, pid, "2.1 质量保证"), StructureInput{Op: OpRemove}); err != nil {
		t.Fatal(err)
	}
	res, err = st.Restructure(sectionIDByTitle(t, st, pid, "第2章 保障"), StructureInput{Op: OpRemove})
	if err != nil || res.FocusID != 0 {
		t.Fatalf("删空回执：%+v err=%v", res, err)
	}
	if ftsRows(t, st) != 0 {
		t.Fatalf("删空后全文索引剩 %d 行", ftsRows(t, st))
	}

	// 守卫：小节不存在、未知操作、录入中未收尾的方案不动。
	st = newStore(t)
	pid = seedTree(t, st)
	dep = sectionIDByTitle(t, st, pid, "1.1 部署")
	if _, err := st.Restructure(99999, StructureInput{Op: OpPromote}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("小节不存在应报错，得到 err=%v", err)
	}
	if _, err := st.Restructure(dep, StructureInput{Op: "bogus"}); err == nil || !strings.Contains(err.Error(), "未知的结构操作") {
		t.Fatalf("未知操作应报错，得到 err=%v", err)
	}
	if _, err := st.Restructure(dep, StructureInput{Op: OpAddChild, Title: "　"}); err == nil || !strings.Contains(err.Error(), "标题不能为空") {
		t.Fatalf("空标题应报错，得到 err=%v", err)
	}
	if _, err := st.db.Exec(`UPDATE proposals SET status=? WHERE id=?`, StatusIngesting, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Restructure(dep, StructureInput{Op: OpRemove}); err == nil || !strings.Contains(err.Error(), "尚未收尾") {
		t.Fatalf("录入中的方案应拒绝调整结构，得到 err=%v", err)
	}
	wantTree(t, st, pid, [][]any{
		{"第1章 总体", 1, "第1章 总体"}, {"1.1 部署", 2, "第1章 总体 › 1.1 部署"}, {"1.1.1 细则", 3, "第1章 总体 › 1.1 部署 › 1.1.1 细则"},
		{"1.1.2 补充", 3, "第1章 总体 › 1.1 部署 › 1.1.2 补充"}, {"1.2 安全", 2, "第1章 总体 › 1.2 安全"},
		{"第2章 保障", 1, "第2章 保障"}, {"2.1 质量保证", 2, "第2章 保障 › 2.1 质量保证"},
	})
}

func TestRestructureMovesAssets(t *testing.T) {
	st, _ := newAssetStore(t)
	sourceDir := t.TempDir()
	raw := writeTinyPNG(t, filepath.Join(sourceDir, "flow.png"))
	res, err := st.Save(SaveInput{
		Meta:         Proposal{Title: "图片方案"},
		AssetBaseDir: sourceDir,
		Sections:     []Section{{Level: 1, Title: "第一章 工艺", Body: "前文说明。\n\n![工艺流程图](flow.png)\n\n后文说明。"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	secs, _ := st.Outline(res.ProposalID)
	head := secs[0].ID
	assetRows := func(id int64) int {
		var n int
		_ = st.db.QueryRow(`SELECT COUNT(*) FROM proposal_assets WHERE section_id=?`, id).Scan(&n)
		return n
	}
	if assetRows(head) != 1 {
		t.Fatalf("录入后图片归属：%d", assetRows(head))
	}
	var ref string
	_ = st.db.QueryRow(`SELECT sha256 FROM proposal_assets WHERE section_id=?`, head).Scan(&ref)
	assetPath := st.assetPath(ref, "image/png")

	// 从图片那一行拆分：图片跟着正文搬到新小节，托管文件不动。
	out, err := st.Restructure(head, StructureInput{Op: OpSplit, Offset: 6, Title: "工艺流程图"})
	if err != nil {
		t.Fatal(err)
	}
	if assetRows(head) != 0 || assetRows(out.FocusID) != 1 {
		t.Fatalf("拆分后图片归属：本节 %d，新节 %d", assetRows(head), assetRows(out.FocusID))
	}
	if b := readBody(t, st, out.FocusID); !strings.Contains(b, AssetURIPrefix) || strings.Contains(b, "flow.png") {
		t.Fatalf("新小节正文 = %q", b)
	}
	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("拆分不应删托管文件：%v", err)
	}
	// 删掉新小节：图片已无人引用，托管文件回收。
	if _, err := st.Restructure(out.FocusID, StructureInput{Op: OpRemove}); err != nil {
		t.Fatal(err)
	}
	if assetRows(out.FocusID) != 0 {
		t.Fatal("删除后图片归属应清除")
	}
	if _, err := os.Stat(assetPath); err == nil {
		t.Fatal("无人引用的托管文件应被回收")
	}
	if len(raw) == 0 {
		t.Fatal("unreachable")
	}
}
