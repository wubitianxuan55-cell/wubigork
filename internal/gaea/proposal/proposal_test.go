package proposal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/retrieval"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	st, err := Open(gdb)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// mdLines 一份迷你技术标的 Markdown 行：含封面、目录（标题会重复出现一次）、
// 1~3 级标题与一张表。
var mdLines = []string{
	"包3技术标 第三版",
	"",
	"目录",
	"第1章 项目实施方案 …… 3",
	"1.1 总体部署 …… 4",
	"",
	"# 第1章 项目实施方案",
	"本章说明总体部署。",
	"",
	"## 1.1 总体部署",
	"分 A、B、C 三个施工区，先北后南。",
	"",
	"## 1.2 施工降水",
	"基坑降水采用管井降水，降水期间监测周边建筑物沉降。",
	"如发现沉降超限，立即回灌并停止降水。",
	"",
	"# 第4章 安全保障方案",
	"安全管理体系与应急预案。",
	"",
	"## 4.3 危险源辨识",
	"基坑边坡失稳、扬尘、噪声为主要危险源。",
	"边坡失稳的防控措施：分层开挖、及时支护。",
	"",
	"### 4.3.1 危险源清单",
	"",
	"| 序号 | 危险源 | 控制措施 |",
	"|---|---|---|",
	"| 1 | 边坡失稳 | 分层开挖 |",
	"| 2 | 扬尘 | 洒水覆盖 |",
	"",
	"## 4.4 监测",
	"监测点布置与预警值。",
}

func lineOf(lines []string, text string) int {
	for i, l := range lines {
		if l == text {
			return i
		}
	}
	return -1
}

func hasTerm(terms []string, t string) bool {
	for _, x := range terms {
		if x == t {
			return true
		}
	}
	return false
}

type saveSection struct {
	Level int
	Title string
	Body  string
}

func saveA(t *testing.T, st *Store, title string, secs []saveSection, mut ...func(*SaveInput)) *SaveResult {
	t.Helper()
	in := SaveInput{Meta: Proposal{Title: title, ProjectType: "技术标", Year: "2026"}}
	for _, s := range secs {
		in.Sections = append(in.Sections, Section{Level: s.Level, Title: s.Title, Body: s.Body})
	}
	for _, f := range mut {
		f(&in)
	}
	res, err := st.Save(in)
	if err != nil {
		t.Fatalf("Save(%s): %v", title, err)
	}
	return res
}

// ingestMD 把 mdLines 分两批按源文件形态录入；第二批收尾。返回方案 id。
func ingestMD(t *testing.T, st *Store) int64 {
	t.Helper()
	b1, err := st.Save(SaveInput{
		Meta:  Proposal{Title: "包3技术标 第三版", ProjectType: "技术标", Year: "2026"},
		Lines: mdLines, SourceFile: "D:/work/bao3.md",
		Outline: []OutlineEntry{
			{Level: 1, Title: "第1章 项目实施方案"},
			{Level: 2, Title: "1.1 总体部署"},
			{Level: 2, Title: "1.2 施工降水"},
		},
	})
	if err != nil {
		t.Fatalf("第一批: %v", err)
	}
	if b1.Done || b1.PendingTitle != "1.2 施工降水" {
		t.Fatalf("第一批后应仍在录入中: %+v", b1)
	}
	b2, err := st.Save(SaveInput{
		Meta: Proposal{Title: "包3技术标 第三版"}, Append: true, Finalize: true,
		Lines: mdLines,
		Outline: []OutlineEntry{
			{Level: 1, Title: "第4章 安全保障方案"},
			{Level: 2, Title: "4.3 危险源辨识"},
			{Level: 3, Title: "危险源清单"},
			{Level: 2, Title: "监测"},
		},
	})
	if err != nil {
		t.Fatalf("第二批: %v", err)
	}
	if !b2.Done || b2.Added != 4 || b2.Stats.Sections != 7 {
		t.Fatalf("第二批回执: %+v", b2)
	}
	return b2.ProposalID
}

// ── 纯函数 ─────────────────────────────────────────────────────────

func TestNormHeadingAndGrade(t *testing.T) {
	cases := []struct {
		line, title string
		want        int
	}{
		{"### 4.3.1 危险源清单", "4.3.1 危险源清单", 5},    // 整行相等
		{"## 4.3 **危险源辨识**", "危险源辨识", 4},         // Markdown 标题行含标题
		{"第1章 项目实施方案 …… 3", "第1章 项目实施方案", 0},     // 目录行
		{"1.1 总体部署.......... 12", "1.1 总体部署", 0}, // 目录行（点线+页码）
		{"详见 4.3 危险源辨识 45", "危险源辨识", 0},          // 正文里的目录式引用
		{"4.3 危险源辨识与评价", "4.3 危险源辨识", 3},         // 标题样短行前缀命中
		{"本节讨论危险源辨识方法，内容较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长较长。", "危险源辨识", 1},
		{"本章说明总体部署。", "第4章 安全保障方案", 0},
		// 正文句以标题开头：真标题行（带编号的 Markdown 标题）必须排前面
		{"## 4.4 监测", "监测", 4},
		{"监测点布置与预警值。", "监测", 1},
		// 无 Markdown 标题的纯文本文档：短行（非句末标点）才算标题样
		{"（一）总体部署", "总体部署", 2},
		{"一、编制依据：", "编制依据", 2},
		{"总体部署详见附图", "总体部署", 3},
	}
	for _, c := range cases {
		if got := headingGrade(c.line, c.title); got != c.want {
			t.Errorf("headingGrade(%q, %q) = %d, want %d", c.line, c.title, got, c.want)
		}
	}
	if got := normHeading("\uFEFF #  **第4章 安全保障方案** ： "); got != "第4章 安全保障方案" {
		t.Errorf("normHeading = %q", got)
	}
}

func TestSplitTerms(t *testing.T) {
	got := SplitTerms("  边坡稳定，扬尘/HJ 25.1-2019（地下水）\"引号\" 扬尘 ")
	want := []string{"边坡稳定", "扬尘", "HJ", "25.1-2019", "地下水", "引号"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("SplitTerms = %v, want %v", got, want)
	}
}

func TestFindPassages(t *testing.T) {
	text := "第一段讲降水。\n\n第二段讲沉降监测，沉降超限要回灌。\n\n第三段讲扬尘。"
	ps := FindPassages(text, []string{"沉降", "扬尘"}, 6, 10, 5)
	if len(ps) != 2 {
		t.Fatalf("passages = %d, want 2: %+v", len(ps), ps)
	}
	if ps[0].Start > ps[1].Start {
		t.Fatalf("应按文中位置返回: %+v", ps)
	}
	rs := []rune(text)
	out := RenderPassage(rs, ps[0], []string{"沉降", "扬尘"})
	if !strings.Contains(out, "【沉降】") || !hasTerm(ps[1].Terms, "扬尘") {
		t.Fatalf("渲染/词归属不对: %q / %+v", out, ps)
	}
	for _, p := range ps {
		if p.Start < 0 || p.End > len(rs) || p.End <= p.Start {
			t.Fatalf("窗口越界: %+v", p)
		}
	}
	if FindPassages(text, []string{"不存在"}, 6, 6, 5) != nil {
		t.Fatal("无命中应返回 nil")
	}
}

func TestChunkBody(t *testing.T) {
	para := strings.Repeat("甲", 100)
	body := para + "。\n\n" + para + "。\n短行\n" +
		"| 序号 | 名称 |\n|---|---|\n" + strings.Repeat("| 1 | 某某物料某某规格 |\n", 60) +
		"\n" + strings.Repeat("长。", 400)
	rs := []rune(body)
	chunks := ChunkBody(body)
	if len(chunks) < 5 {
		t.Fatalf("chunks = %d", len(chunks))
	}
	for i, c := range chunks {
		if c.Seq != i || c.Start < 0 || c.Start+c.Len > len(rs) {
			t.Fatalf("块 %d 区间越界: start=%d len=%d", i, c.Start, c.Len)
		}
		// 块文本=正文原样切片；超长表格的后续组例外：前面带重复表头，正文切片是它的后缀。
		if slice := string(rs[c.Start : c.Start+c.Len]); c.Text != slice && !strings.HasSuffix(c.Text, slice) {
			t.Fatalf("块 %d 与正文区间对不上: start=%d len=%d", i, c.Start, c.Len)
		}
	}
	// 表格被整表识别并按行组切分：至少有一块带重复表头（非首组）。
	tableChunks := 0
	for _, c := range chunks {
		if strings.HasPrefix(c.Text, "| 序号 |") {
			tableChunks++
		}
	}
	if tableChunks < 2 {
		t.Fatalf("超长表格应切成多块: %d", tableChunks)
	}
	for _, c := range chunks {
		if strings.Contains(c.Text, "| 1 |") && !strings.HasPrefix(c.Text, "| 序号 |") {
			t.Fatal("表格块缺表头")
		}
	}
	// 所有非表格字符都被覆盖（块之间允许跳过空行）。
	covered := make([]bool, len(rs))
	for _, c := range chunks {
		for i := c.Start; i < c.Start+c.Len; i++ {
			covered[i] = true
		}
	}
	for i, r := range rs {
		if r != '\n' && !covered[i] {
			t.Fatalf("正文第 %d 个字符未被任何块覆盖: %q", i, string(r))
		}
	}
	if ChunkBody("") != nil {
		t.Fatal("空正文应无块")
	}
}

// ── 录入 ───────────────────────────────────────────────────────────

func TestSaveFormAOutlineAndList(t *testing.T) {
	st := newStore(t)
	res := saveA(t, st, "技术标 甲", []saveSection{
		{1, "第1章 项目实施方案", "总体实施部署与分区施工顺序。执行 HJ 25.1-2019 标准。"},
		{2, "1.2 施工分区", "分 A/B/C 三区。"},
		{1, "第4章 安全保障方案", "安全管理体系与应急预案。"},
		{2, "4.3 危险源辨识", "基坑边坡失稳、扬尘、噪声为主要危险源。"},
	})
	if !res.Created || !res.Done || res.Added != 4 || res.Stats.Sections != 4 {
		t.Fatalf("回执: %+v", res)
	}
	if res.Stats.LevelCounts != [MaxLevels + 1]int{0, 2, 2, 0, 0} {
		t.Fatalf("层级统计: %v", res.Stats.LevelCounts)
	}
	secs, err := st.Outline(res.ProposalID)
	if err != nil {
		t.Fatal(err)
	}
	if secs[1].ParentID != secs[0].ID || secs[3].ParentID != secs[2].ID || secs[0].ParentID != 0 {
		t.Fatalf("父节点不对: %+v", secs)
	}
	if secs[1].TitlePath != "第1章 项目实施方案 › 1.2 施工分区" {
		t.Fatalf("title_path = %q", secs[1].TitlePath)
	}
	if secs[0].Chars != len([]rune("总体实施部署与分区施工顺序。执行 HJ 25.1-2019 标准。")) {
		t.Fatalf("Chars = %d", secs[0].Chars)
	}
	list, err := st.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Sections != 4 || list[0].Chars == 0 || list[0].Status != StatusActive || list[0].Title != "技术标 甲" {
		t.Fatalf("list = %+v", list)
	}
	p, err := st.Get("技术标 甲")
	if err != nil || p == nil {
		t.Fatalf("Get（方案名=标题）: %+v %v", p, err)
	}
	if r, _ := st.Resolve("技术标 甲"); r == nil || r.ID != res.ProposalID {
		t.Fatalf("Resolve: %+v", r)
	}
	if r, _ := st.Resolve("不存在"); r != nil {
		t.Fatalf("Resolve 不存在应 nil: %+v", r)
	}
}

func TestSaveOverwriteGuardAndAppend(t *testing.T) {
	st := newStore(t)
	saveA(t, st, "技术标 甲", []saveSection{{1, "第一章", "正文"}})
	_, err := st.Save(SaveInput{Meta: Proposal{Title: "技术标 甲"}, Sections: []Section{{Level: 1, Title: "别的", Body: "x"}}})
	var exists *ExistsError
	if !errors.As(err, &exists) || exists.Sections != 1 {
		t.Fatalf("同名未确认应拒: %v", err)
	}
	if strings.Contains(err.Error(), "未注明") == false && exists.Source != "" {
		t.Fatalf("ExistsError 文案: %v", err)
	}
	// 追加：接续标题路径栈
	res := saveA(t, st, "技术标 甲", []saveSection{{2, "1.1 小节", "y"}}, func(in *SaveInput) { in.Append = true })
	if res.Stats.Sections != 2 {
		t.Fatalf("追加后节数: %+v", res.Stats)
	}
	secs, _ := st.Outline(res.ProposalID)
	if secs[1].ParentID != secs[0].ID || secs[1].TitlePath != "第一章 › 1.1 小节" {
		t.Fatalf("追加节点应挂到上一批末节下: %+v", secs[1])
	}
	// 追加到不存在的方案：拒
	if _, err := st.Save(SaveInput{Meta: Proposal{Title: "没有的"}, Append: true, Sections: []Section{{Level: 1, Title: "x"}}}); err == nil {
		t.Fatal("追加不存在的方案应报错")
	}
	// 确认覆盖：旧节让位
	res = saveA(t, st, "技术标 甲", []saveSection{{1, "新第一章", "z"}}, func(in *SaveInput) { in.Overwrite = true })
	if !res.Replaced || res.Stats.Sections != 1 || res.Created {
		t.Fatalf("覆盖回执: %+v", res)
	}
	secs, _ = st.Outline(res.ProposalID)
	if len(secs) != 1 || secs[0].Title != "新第一章" {
		t.Fatalf("覆盖后: %+v", secs)
	}
	// 参数互斥与跳级
	if _, err := st.Save(SaveInput{Meta: Proposal{Title: "x"}, Append: true, Overwrite: true, Sections: []Section{{Level: 1, Title: "a"}}}); err == nil {
		t.Fatal("append+overwrite 应拒")
	}
	_, err = st.Save(SaveInput{Meta: Proposal{Title: "跳级"}, Sections: []Section{
		{Level: 1, Title: "第一章", Body: "x"}, {Level: 3, Title: "直接三级", Body: "y"}}})
	if err == nil || !strings.Contains(err.Error(), "跳级") {
		t.Fatalf("跳级应 fail-closed: %v", err)
	}
	if _, err := st.Save(SaveInput{Meta: Proposal{Title: "空节"}}); err == nil || !strings.Contains(err.Error(), "至少给一个") {
		t.Fatalf("空录入应报错: %v", err)
	}
	// 超 4 级压到 4 级并计数
	res = saveA(t, st, "压级", []saveSection{{1, "一", "a"}, {2, "二", "b"}, {3, "三", "c"}, {4, "四", "d"}, {6, "六", "e"}})
	if res.Clamped != 1 || res.Stats.LevelCounts[4] != 2 {
		t.Fatalf("压级回执: %+v", res.Stats)
	}
}

func TestSaveFormBIngestBatches(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	p, err := st.Get("包3技术标 第三版")
	if err != nil || p == nil {
		t.Fatalf("Get: %v %v", p, err)
	}
	if p.Status != StatusActive || p.SourceFile != "D:/work/bao3.md" || p.ProjectType != "技术标" {
		t.Fatalf("追加不应抹掉首批元信息: %+v", p)
	}
	secs, err := st.Outline(pid)
	if err != nil {
		t.Fatal(err)
	}
	wantTitles := []string{"第1章 项目实施方案", "1.1 总体部署", "1.2 施工降水", "第4章 安全保障方案", "4.3 危险源辨识", "4.3.1 危险源清单", "4.4 监测"}
	for i, w := range wantTitles {
		if secs[i].Title != w {
			t.Fatalf("第 %d 节标题 = %q, want %q", i, secs[i].Title, w)
		}
	}
	if secs[6].SrcLine != lineOf(mdLines, "## 4.4 监测")+1 {
		t.Fatalf("「监测」应锚到标题行而不是以它开头的正文句: %+v", secs[6])
	}
	if secs[5].TitlePath != "第4章 安全保障方案 › 4.3 危险源辨识 › 4.3.1 危险源清单" || secs[5].ParentID != secs[4].ID {
		t.Fatalf("三级节点: %+v", secs[5])
	}
	if secs[0].SrcLine != lineOf(mdLines, "# 第1章 项目实施方案")+1 {
		t.Fatalf("src_line = %d", secs[0].SrcLine)
	}
	// 正文逐字切出：末批首节之前的正文归上一批末节。
	r2, err := st.Read(secs[2].ID, ReadOpt{})
	if err != nil || r2.Text != "基坑降水采用管井降水，降水期间监测周边建筑物沉降。\n如发现沉降超限，立即回灌并停止降水。" {
		t.Fatalf("1.2 正文: %q %v", r2.Text, err)
	}
	r6, _ := st.Read(secs[5].ID, ReadOpt{})
	if !strings.HasPrefix(r6.Text, "| 序号 | 危险源 | 控制措施 |") || !strings.HasSuffix(r6.Text, "| 2 | 扬尘 | 洒水覆盖 |") {
		t.Fatalf("表格应原样保留: %q", r6.Text)
	}
	// 语义块：带标题路径前缀，覆盖全部小节。
	pend, total := st.PendingCount(pid)
	if pend == 0 || pend != total {
		t.Fatalf("录入后应全部待向量化: pending=%d total=%d", pend, total)
	}
	var text string
	_ = st.db.QueryRow(`SELECT text FROM proposal_chunks WHERE section_id=? AND seq=0`, secs[2].ID).Scan(&text)
	if !strings.HasPrefix(text, secs[2].TitlePath+"\n") {
		t.Fatalf("块文本应带标题路径前缀: %q", text)
	}
}

func TestSaveFormBGuards(t *testing.T) {
	st := newStore(t)
	first := func(outline []OutlineEntry, mut ...func(*SaveInput)) error {
		in := SaveInput{Meta: Proposal{Title: "T"}, Lines: mdLines, Outline: outline}
		for _, f := range mut {
			f(&in)
		}
		_, err := st.Save(in)
		return err
	}
	// 标题找不到：带附近的标题样行
	err := first([]OutlineEntry{{Level: 1, Title: "不存在的标题"}})
	if err == nil || !strings.Contains(err.Error(), "不存在的标题") || !strings.Contains(err.Error(), "标题样行") {
		t.Fatalf("找不到标题应带上下文: %v", err)
	}
	// 指定行号但行里没有该标题
	err = first([]OutlineEntry{{Level: 1, Title: "第1章 项目实施方案", Line: 1}})
	if err == nil || !strings.Contains(err.Error(), "不含此标题") {
		t.Fatalf("指错行应报错: %v", err)
	}
	// 目录行不会被当成标题
	res, err := st.Save(SaveInput{Meta: Proposal{Title: "T"}, Lines: mdLines, Finalize: true,
		Outline: []OutlineEntry{{Level: 1, Title: "第1章 项目实施方案"}, {Level: 2, Title: "1.1 总体部署"}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.SkippedLines != lineOf(mdLines, "# 第1章 项目实施方案") {
		t.Fatalf("首个标题前（封面+目录）的行数应计入 SkippedLines: %d", res.SkippedLines)
	}
	secs, _ := st.Outline(res.ProposalID)
	if secs[0].SrcLine != lineOf(mdLines, "# 第1章 项目实施方案")+1 {
		t.Fatalf("锚到了目录行: %+v", secs[0])
	}
	// 文件到末尾自动收尾
	if !res.Done || res.Stats.Sections != 2 {
		t.Fatalf("回执: %+v", res)
	}
	// 已完成的源文件方案不能再追加标题
	if err := first([]OutlineEntry{{Level: 2, Title: "4.4 监测"}}, func(in *SaveInput) { in.Append = true }); err == nil || !strings.Contains(err.Error(), "已录入完成") {
		t.Fatalf("完成后追加应拒: %v", err)
	}
	// 纯文本（无 Markdown 标题）：正文句「本章规定适用范围与术语。」不应被当成标题，
	// 短标题行「1.1 适用范围」才落锚；短标题行是纯文本文档的常态，不算弱匹配。
	plain := []string{
		"第一章 总则", "本章规定适用范围与术语。", "", "1.1 适用范围",
		"本标准适用于新建、扩建、改建的工业与民用建筑。", "", "1.2 术语", "本标准采用下列术语和定义。",
	}
	res, err = st.Save(SaveInput{Meta: Proposal{Title: "纯文本"}, Lines: plain, Finalize: true, Outline: []OutlineEntry{
		{Level: 1, Title: "总则"}, {Level: 2, Title: "适用范围"}, {Level: 2, Title: "术语"}}})
	if err != nil {
		t.Fatal(err)
	}
	secs, _ = st.Outline(res.ProposalID)
	if secs[1].SrcLine != 4 || secs[2].SrcLine != 7 || secs[1].Title != "适用范围" {
		t.Fatalf("正文句「本章规定适用范围与术语。」不应被当成标题: %+v", secs)
	}
	if len(res.WeakMatches) != 0 {
		t.Fatalf("短标题行不应报弱匹配: %v", res.WeakMatches)
	}
	// 标题只出现在正文句里：照样定位（不中断录入），但点名让模型核对行号
	res, err = st.Save(SaveInput{Meta: Proposal{Title: "句中标题"}, Lines: []string{
		"一、总则", "本章规定适用范围与术语。", "", "1.1 具体要求", "要求如下。"}, Finalize: true, Outline: []OutlineEntry{
		{Level: 1, Title: "总则"}, {Level: 2, Title: "适用范围"}, {Level: 2, Title: "具体要求"}}})
	if err != nil {
		t.Fatal(err)
	}
	if secs, _ = st.Outline(res.ProposalID); secs[1].SrcLine != 2 || secs[1].Title != "适用范围" || secs[2].SrcLine != 4 {
		t.Fatalf("顺序定位/句中标题: %+v", secs)
	}
	if len(res.WeakMatches) != 1 || !strings.Contains(res.WeakMatches[0], "第 2 项") || !strings.Contains(res.WeakMatches[0], "第 2 行") {
		t.Fatalf("弱匹配应点名行号: %v", res.WeakMatches)
	}
}

func TestSaveFormBSourceChangedRejected(t *testing.T) {
	st := newStore(t)
	_, err := st.Save(SaveInput{Meta: Proposal{Title: "T"}, Lines: mdLines, Outline: []OutlineEntry{{Level: 1, Title: "第1章 项目实施方案"}}})
	if err != nil {
		t.Fatal(err)
	}
	// 改动已录前缀里的行（封面）：拒续录
	changed := append([]string{}, mdLines...)
	changed[0] = "封面被改过了。"
	_, err = st.Save(SaveInput{Meta: Proposal{Title: "T"}, Append: true, Lines: changed,
		Outline: []OutlineEntry{{Level: 2, Title: "1.1 总体部署"}}})
	if err == nil || !strings.Contains(err.Error(), "被修改") {
		t.Fatalf("源文件被改应拒续录: %v", err)
	}
	// 录入中、末节待截取时不能用直给正文形态追加
	_, err = st.Save(SaveInput{Meta: Proposal{Title: "T"}, Append: true, Sections: []Section{{Level: 2, Title: "x"}}})
	if err == nil || !strings.Contains(err.Error(), "待截取") {
		t.Fatalf("待截取时直给正文追加应拒: %v", err)
	}
	// 只收尾
	res, err := st.Save(SaveInput{Meta: Proposal{Title: "T"}, Append: true, Finalize: true, Lines: mdLines})
	if err != nil || !res.Done || res.Added != 0 {
		t.Fatalf("只收尾: %+v %v", res, err)
	}
}

func TestSaveBodyTooLongRejected(t *testing.T) {
	st := newStore(t)
	_, err := st.Save(SaveInput{Meta: Proposal{Title: "长"}, Sections: []Section{{Level: 1, Title: "x", Body: strings.Repeat("字", MaxBodyChars+1)}}})
	if err == nil || !strings.Contains(err.Error(), "单节上限") {
		t.Fatalf("超长正文应拒而不是截断: %v", err)
	}
}

// ── 读取 ───────────────────────────────────────────────────────────

func TestReadPagesPassagesSubtree(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	secs, _ := st.Outline(pid)
	danger, table, monitor := secs[4], secs[5], secs[6]

	r, err := st.Read(danger.ID, ReadOpt{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != danger.Body && r.Text != "基坑边坡失稳、扬尘、噪声为主要危险源。\n边坡失稳的防控措施：分层开挖、及时支护。" {
		t.Fatalf("整节正文: %q", r.Text)
	}
	if len(r.Children) != 1 || r.Children[0].ID != table.ID || r.Children[0].Title != "4.3.1 危险源清单" {
		t.Fatalf("子节清单: %+v", r.Children)
	}
	if r.Proposal.Title != "包3技术标 第三版" || r.Section.TitlePath != danger.TitlePath || r.Total != len([]rune(r.Text)) {
		t.Fatalf("回执头: %+v", r)
	}

	// 关键词：只回命中段落，带标记
	r, err = st.Read(danger.ID, ReadOpt{Query: "扬尘 措施"})
	if err != nil || len(r.Passages) == 0 || r.Text != "" {
		t.Fatalf("关键词阅读: %+v %v", r, err)
	}
	if !strings.Contains(r.Passages[0].Text, "【扬尘】") || r.Passages[0].SectionID != danger.ID {
		t.Fatalf("命中段落: %+v", r.Passages[0])
	}
	if r, _ = st.Read(danger.ID, ReadOpt{Query: "完全没有的词"}); len(r.Passages) != 0 {
		t.Fatalf("无命中不应有段落: %+v", r.Passages)
	}

	// 分页
	r, _ = st.Read(monitor.ID, ReadOpt{Limit: 3})
	if r.Text != "监测点" || !r.HasMore || r.Offset != 0 {
		t.Fatalf("分页首页: %+v", r)
	}
	r2, _ := st.Read(monitor.ID, ReadOpt{Offset: r.Offset + len([]rune(r.Text)), Limit: 100})
	if r2.Text != "布置与预警值。" || r2.HasMore {
		t.Fatalf("分页次页: %+v", r2)
	}

	// 子树整读：目标节 + 全部子孙节
	r, err = st.Read(danger.ID, ReadOpt{Scope: "subtree", Query: "洒水"})
	if err != nil || len(r.Parts) != 2 || len(r.Passages) != 1 {
		t.Fatalf("子树: %+v %v", r, err)
	}
	if r.Parts[1].SectionID != table.ID || r.Parts[1].Start != r.Parts[0].Chars+2 {
		t.Fatalf("子树各节起点（节间以空行相连）: %+v", r.Parts)
	}
	if r.Passages[0].SectionID != table.ID || r.Passages[0].Title != "4.3.1 危险源清单" {
		t.Fatalf("子树命中应归到子节: %+v", r.Passages[0])
	}
	r, _ = st.Read(secs[0].ID, ReadOpt{Scope: "subtree"})
	if len(r.Parts) != 3 || r.Total != len([]rune(r.Text)) || !strings.Contains(r.Text, "管井降水") {
		t.Fatalf("章级子树应含两个子节: %d parts", len(r.Parts))
	}
	// 不存在的小节
	if _, err := st.Read(99999, ReadOpt{}); err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("不存在的小节: %v", err)
	}
}

// ── 全文路检索 ─────────────────────────────────────────────────────

func TestSearchKeywordRoad(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	saveA(t, st, "调查报告 乙", []saveSection{{1, "第二章 场地调查", "地下水监测井布设，扬尘控制。"}},
		func(in *SaveInput) { in.Meta.ProjectType = "调查报告" })
	secs, _ := st.Outline(pid)
	danger := secs[4]

	search := func(in SearchInput) *SearchResult {
		t.Helper()
		in.normalize()
		res, err := st.Search(in)
		if err != nil {
			t.Fatalf("Search(%+v): %v", in, err)
		}
		return res
	}
	ids := func(res *SearchResult) []int64 {
		var out []int64
		for _, h := range res.Hits {
			out = append(out, h.SectionID)
		}
		return out
	}

	// 两字词：LIKE 兜底能命中
	res := search(SearchInput{Query: "扬尘"})
	if len(res.Hits) != 3 { // 危险源辨识正文 + 表格行 + 另一份方案
		t.Fatalf("两字词应 3 命中: %+v", ids(res))
	}
	// 混合长短词 AND：≥3 字走全文索引、短词走 LIKE
	res = search(SearchInput{Query: "危险源 扬尘"})
	if len(ids(res)) != 2 {
		t.Fatalf("长短词 AND: %+v", ids(res))
	}
	// 摘要取命中位置而不是节首
	res = search(SearchInput{Query: "周边建筑物沉降"})
	if len(res.Hits) != 1 || !strings.Contains(res.Hits[0].Snippet, "【周边建筑物沉降】") || res.Hits[0].Title != "1.2 施工降水" {
		t.Fatalf("摘要应在命中位置: %+v", res.Hits)
	}
	// 命中带子节清单供下钻
	res = search(SearchInput{Query: "噪声"})
	if len(res.Hits) != 1 || res.Hits[0].ChildCount != 1 || len(res.Hits[0].Children) != 1 || res.Hits[0].Children[0].ID != secs[5].ID {
		t.Fatalf("子节清单: %+v", res.Hits)
	}
	// 层级 / 方案 / 类型过滤
	if res = search(SearchInput{Query: "扬尘", Level: 3}); len(res.Hits) != 1 || res.Hits[0].Level != 3 {
		t.Fatalf("层级过滤: %+v", ids(res))
	}
	if res = search(SearchInput{Query: "扬尘", Proposal: "包3技术标 第三版"}); len(res.Hits) != 2 {
		t.Fatalf("方案过滤: %+v", ids(res))
	}
	if res = search(SearchInput{Query: "扬尘", Proposal: "没有的方案"}); len(res.Hits) != 0 {
		t.Fatalf("不存在的方案应空: %+v", ids(res))
	}
	if res = search(SearchInput{Query: "扬尘", ProjectType: "调查报告"}); len(res.Hits) != 1 || res.Hits[0].ProposalTitle != "调查报告 乙" {
		t.Fatalf("类型过滤: %+v", ids(res))
	}
	// 标题路：只看标题树，标题本身命中优先
	res = search(SearchInput{Query: "危险源", Scope: ScopeTitle})
	if res.Mode != "title" || len(res.Hits) != 2 || res.Hits[0].SectionID != danger.ID {
		t.Fatalf("标题路: %+v", res.Hits)
	}
	if !strings.Contains(res.Hits[0].Snippet, "【危险源】") || res.Hits[0].Snippet != "第4章 安全保障方案 › 4.3 【危险源】辨识" {
		t.Fatalf("标题路摘要: %q", res.Hits[0].Snippet)
	}
	// 特殊字符不炸（search 遇错即 Fatal）；这些词正文里都没有，应为空
	if res = search(SearchInput{Query: "\"引号\" (括号) %百分% _下划线_ OR NEAR"}); len(res.Hits) != 0 {
		t.Fatalf("特殊字符查询不应误命中: %+v", ids(res))
	}
	if res = search(SearchInput{Query: "   "}); len(res.Hits) != 0 {
		t.Fatalf("空查询: %+v", res)
	}
	_ = danger
}

func TestSearchEscapedTermsHit(t *testing.T) {
	st := newStore(t)
	res := saveA(t, st, "规范引用", []saveSection{{1, "依据", "执行 HJ 25.1-2019（地下水）标准；100% 达标；a_b 标记。"}})
	secs, _ := st.Outline(res.ProposalID)
	for _, q := range []string{"HJ 25.1-2019（地下水）", "100% 达标", "a_b", "\"HJ\""} {
		r, err := st.Search(SearchInput{Query: q})
		if err != nil || len(r.Hits) != 1 || r.Hits[0].SectionID != secs[0].ID {
			t.Fatalf("查询 %q 应命中且不报错: %v %+v", q, err, r.Hits)
		}
	}
}

func TestLongBodySnippetAtMatch(t *testing.T) {
	st := newStore(t)
	body := strings.Repeat("铺垫内容。", 80) + "此处才是关键：基坑降水引起周边沉降。" + strings.Repeat("收尾内容。", 80)
	saveA(t, st, "长节", []saveSection{{1, "第一章", body}})
	r, _ := st.Search(SearchInput{Query: "周边沉降"})
	if len(r.Hits) != 1 || !strings.Contains(r.Hits[0].Snippet, "【周边沉降】") || strings.HasPrefix(r.Hits[0].Snippet, "铺垫内容。") {
		t.Fatalf("摘要应落在命中位置: %q", r.Hits[0].Snippet)
	}
	if n := len([]rune(r.Hits[0].Snippet)); n > 520 {
		t.Fatalf("摘要过长: %d", n)
	}
}

// ── 语义路与混合检索 ───────────────────────────────────────────────

// fakeEmbedder 确定性假向量：字符二元组散列进 dim 维再归一（共用二元组越多越
// 相似）。全程零网络。
type fakeEmbedder struct {
	dim   int
	err   error
	delay time.Duration // 模拟本地服务的每批耗时
	calls int
}

func (f *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.err != nil {
		return nil, f.err
	}
	f.calls++
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = fakeVec(t, f.dim)
	}
	return out, nil
}

func fakeVec(text string, dim int) []float32 {
	v := make([]float32, dim)
	rs := []rune(strings.ToLower(text))
	for i := 0; i+1 < len(rs); i++ {
		v[int(rs[i]*131+rs[i+1]*7)%dim] += 1
	}
	var n float64
	for _, x := range v {
		n += float64(x) * float64(x)
	}
	if n == 0 {
		v[0] = 1
		return v
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / n)
	}
	return v
}

// fakeReranker 假精排：按文档里出现的检索词字符数排序。
type fakeReranker struct{ available bool }

func (f fakeReranker) Available(context.Context) bool { return f.available }

func (f fakeReranker) Rerank(_ context.Context, query string, docs []string, topN int) ([]retrieval.ScoredDoc, error) {
	type item struct {
		idx   int
		score float64
	}
	var items []item
	for i, d := range docs {
		var s float64
		for _, r := range query {
			if strings.ContainsRune(d, r) {
				s++
			}
		}
		items = append(items, item{i, s})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
	if topN > 0 && topN < len(items) {
		items = items[:topN]
	}
	out := make([]retrieval.ScoredDoc, 0, len(items))
	for _, it := range items {
		out = append(out, retrieval.ScoredDoc{Index: it.idx, Score: it.score})
	}
	return out, nil
}

func TestEmbedPendingRecallAndModelGuard(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	secs, _ := st.Outline(pid)
	ctx := context.Background()

	// 模型不一致：不动库、不召回
	if err := st.ResetEmbedding("别的模型"); err != nil {
		t.Fatal(err)
	}
	em := &fakeEmbedder{dim: 32}
	if _, _, err := st.EmbedPending(ctx, em, "fake-embed", pid, time.Second); !errors.Is(err, ErrEmbedModelMismatch) {
		t.Fatalf("模型不一致应拒: %v", err)
	}
	if _, err := st.SemanticRecall(ctx, em, "fake-embed", "降水", pid, 10); !errors.Is(err, ErrEmbedModelMismatch) {
		t.Fatalf("模型不一致不应召回: %v", err)
	}
	if p, _ := st.PendingCount(pid); p == 0 {
		t.Fatal("模型不一致时不得改库")
	}

	// 切回并补齐向量
	if err := st.ResetEmbedding("fake-embed"); err != nil {
		t.Fatal(err)
	}
	done, pending, err := st.EmbedPending(ctx, em, "fake-embed", pid, 5*time.Second)
	if err != nil || done == 0 || pending != 0 {
		t.Fatalf("EmbedPending: done=%d pending=%d err=%v", done, pending, err)
	}
	list, _ := st.List()
	if list[0].IndexDone != list[0].IndexTotal || list[0].IndexTotal == 0 {
		t.Fatalf("清单索引进度: %+v", list[0])
	}
	// 预算耗尽即停在整批边界，余量留给下次：假服务每批耗时 60ms，预算 50ms 只够
	// 发第一批（32 块）。不拿纳秒级预算测——Windows 墙钟粒度下连续两次读数可能相同。
	if err := st.ResetEmbedding("fake-embed"); err != nil {
		t.Fatal(err)
	}
	big := []saveSection{{1, "第一章", "总述。"}}
	for i := 1; i <= 40; i++ {
		big = append(big, saveSection{2, fmt.Sprintf("1.%d 节", i), fmt.Sprintf("第 %d 节正文。", i)})
	}
	bigID := saveA(t, st, "大方案", big).ProposalID
	slow := &fakeEmbedder{dim: 32, delay: 60 * time.Millisecond}
	done, pending, err = st.EmbedPending(ctx, slow, "fake-embed", bigID, 50*time.Millisecond)
	if err != nil || done != 32 || pending != 9 {
		t.Fatalf("预算耗尽: done=%d pending=%d err=%v", done, pending, err)
	}
	if _, _, err = st.EmbedPending(ctx, em, "fake-embed", 0, 5*time.Second); err != nil || slow.calls != 1 {
		t.Fatalf("补齐: %v calls=%d", err, slow.calls)
	}

	// 召回：换了说法也能落到 1.2 施工降水
	emb := &fakeEmbedder{dim: 32}
	hits, err := st.SemanticRecall(ctx, emb, "fake-embed", "降水期间周边沉降", pid, 10)
	if err != nil || len(hits) == 0 || hits[0].SectionID != secs[2].ID {
		t.Fatalf("语义召回: %v %+v", err, hits)
	}
	// 维度不一致的向量不落库（索引维度已由首批向量定下；混进别的维度=索引被污染）
	extra := saveA(t, st, "多出来的一份", []saveSection{{1, "第一章", "新增正文。"}}).ProposalID
	if _, pend2, _ := st.EmbedPending(ctx, &fakeEmbedder{dim: 16}, "fake-embed", extra, time.Second); pend2 != 1 {
		t.Fatalf("不同维度的向量不应落库: pending=%d", pend2)
	}
	if _, _, err := st.EmbedPending(ctx, nil, "fake-embed", pid, time.Second); !errors.Is(err, ErrEmbedUnavailable) {
		t.Fatalf("无 embedding 应报不可用: %v", err)
	}
}

func TestHybridSearch(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	secs, _ := st.Outline(pid)
	ctx := context.Background()
	em := &fakeEmbedder{dim: 32}
	if _, _, err := st.EmbedPending(ctx, em, "fake-embed", pid, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	sem := &Semantic{Embed: em, Model: "fake-embed", Rerank: fakeReranker{available: true}}

	// 换了说法：全文路零命中，语义路补上，再精排
	res, err := st.HybridSearch(ctx, SearchInput{Query: "降水的时候旁边房子沉了怎么办"}, sem)
	if err != nil || res.Mode != "hybrid" || len(res.Hits) == 0 || res.Hits[0].SectionID != secs[2].ID {
		t.Fatalf("语义补召回: %v %+v", err, res)
	}
	if res.Note != "已本地精排" || res.Hits[0].RerankScore == 0 || res.Hits[0].SemanticScore == 0 {
		t.Fatalf("精排回执: %+v", res)
	}
	if res.Hits[0].Snippet == "" || !strings.Contains(res.Hits[0].Snippet, "降水") {
		t.Fatalf("语义命中的摘要应是命中段原文: %q", res.Hits[0].Snippet)
	}
	if len(res.Hits[0].Children) != 0 && res.Hits[0].ChildCount != len(res.Hits[0].Children) {
		t.Fatal("语义独有命中的子节清单应补齐")
	}

	// 精确词：全文路与语义路融合，命中段落带标记
	res, err = st.HybridSearch(ctx, SearchInput{Query: "边坡失稳"}, sem)
	if err != nil || len(res.Hits) == 0 || !strings.Contains(res.Hits[0].Snippet, "【边坡失稳】") {
		t.Fatalf("精确词: %v %+v", err, res.Hits)
	}
	// 标题路不走语义
	res, _ = st.HybridSearch(ctx, SearchInput{Query: "危险源", Scope: ScopeTitle}, sem)
	if res.Mode != "title" {
		t.Fatalf("标题路 mode: %q", res.Mode)
	}
	// 精排不可用：沿用融合名次
	res, err = st.HybridSearch(ctx, SearchInput{Query: "边坡失稳"}, &Semantic{Embed: em, Model: "fake-embed"})
	if err != nil || res.Mode != "hybrid" || len(res.Hits) == 0 || !strings.Contains(res.Note, "融合") {
		t.Fatalf("无精排: %v %+v", err, res)
	}
	// 本地 embedding 挂了：降级为全文路，不报错
	res, err = st.HybridSearch(ctx, SearchInput{Query: "边坡失稳"}, &Semantic{Embed: &fakeEmbedder{dim: 32, err: errors.New("down")}, Model: "fake-embed"})
	if err != nil || res.Mode != "keyword" || len(res.Hits) != 2 || !strings.Contains(res.Note, "全文路") {
		t.Fatalf("embedding 不可用降级: %v %+v", err, res)
	}
	// 无语义依赖=纯全文
	res, _ = st.HybridSearch(ctx, SearchInput{Query: "扬尘"}, nil)
	if res.Mode != "keyword" || len(res.Hits) != 2 {
		t.Fatalf("nil 语义依赖: %+v", res)
	}
	if res, _ = st.HybridSearch(ctx, SearchInput{Query: "边坡失稳", TopN: 1}, sem); len(res.Hits) != 1 {
		t.Fatalf("TopN: %d", len(res.Hits))
	}
}

// ── 删除 / 不可读 / 迁移 ──────────────────────────────────────────

func TestDeleteCascades(t *testing.T) {
	st := newStore(t)
	pid := ingestMD(t, st)
	em := &fakeEmbedder{dim: 32}
	if _, _, err := st.EmbedPending(context.Background(), em, "fake-embed", pid, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := st.Delete("包3技术标 第三版"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM proposal_sections`, `SELECT COUNT(*) FROM proposal_chunks`,
		`SELECT COUNT(*) FROM proposal_fts`, `SELECT COUNT(*) FROM proposals`,
	} {
		var n int
		if err := st.db.QueryRow(q).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s = %d (%v)，应级联清空", q, n, err)
		}
	}
	if err := st.Delete("包3技术标 第三版"); err == nil {
		t.Fatal("删除不存在的方案应报错")
	}
}

func TestUnreadableStore(t *testing.T) {
	st := &Store{db: nil}
	if _, err := st.List(); err == nil {
		t.Fatal("nil 库 List 应报错")
	}
	if _, err := st.Search(SearchInput{Query: "x"}); err == nil {
		t.Fatal("nil 库 Search 应报错")
	}
	if _, err := st.Read(1, ReadOpt{}); err == nil {
		t.Fatal("nil 库 Read 应报错")
	}
}

// TestMigrationFromV35 已在 V35 建过方案表的库，升级时 V36/V37 必须能跑通。
func TestMigrationFromV35(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Hephaestus.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{db.SchemaV1, db.SchemaV35, `INSERT INTO schema_meta(key, value) VALUES('user_version', '35')`} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("预置 V35 库失败: %v", err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("V35→V37 迁移失败：GetDatabase 返回 nil")
	}
	t.Cleanup(func() { _ = db.CloseDatabase(dir) })
	st, err := Open(gdb)
	if err != nil {
		t.Fatal(err)
	}
	res := saveA(t, st, "升级后录入", []saveSection{{1, "第一章", "正文"}})
	var v string
	_ = gdb.QueryRow(`SELECT value FROM schema_meta WHERE key='user_version'`).Scan(&v)
	if v != "37" || res.Stats.Sections != 1 {
		t.Fatalf("user_version=%s sections=%d", v, res.Stats.Sections)
	}
}

// TestReadAllNotTruncated ReadAll 一次返回整节；缺省与显式上限照旧钳制在 MaxReadLimit。
func TestReadAllNotTruncated(t *testing.T) {
	st := newStore(t)
	body := strings.Repeat("字", proposalReadLenForTest)
	res := saveA(t, st, "超长节", []saveSection{{1, "第一章", body}})
	secs, _ := st.Outline(res.ProposalID)

	r, err := st.Read(secs[0].ID, ReadOpt{})
	if err != nil || len([]rune(r.Text)) != DefaultReadLimit || !r.HasMore {
		t.Fatalf("缺省应按 %d 字分页并提示未完: %d %v hasMore=%v", DefaultReadLimit, len([]rune(r.Text)), err, r.HasMore)
	}
	r, err = st.Read(secs[0].ID, ReadOpt{Limit: 10_000_000})
	if err != nil || len([]rune(r.Text)) != MaxReadLimit {
		t.Fatalf("显式上限仍应钳在 %d 字: %d %v", MaxReadLimit, len([]rune(r.Text)), err)
	}
	r, err = st.Read(secs[0].ID, ReadOpt{Limit: ReadAll})
	if err != nil || len([]rune(r.Text)) != proposalReadLenForTest || r.HasMore || r.Total != proposalReadLenForTest {
		t.Fatalf("ReadAll 应整节返回: %d %v hasMore=%v", len([]rune(r.Text)), err, r.HasMore)
	}
}

// proposalReadLenForTest 超过 MaxReadLimit、又远低于单节正文上限的测试正文长度。
const proposalReadLenForTest = MaxReadLimit + 5000

// TestHybridSearchFiltersApplyToSemanticRoad 层级/类型/年度过滤对语义路独有的命中
// 同样生效（真实程序实测回归：限定 2 级时语义路把 4 级小节带了进来——全文路的候选
// 在 SQL 里过滤，语义路只按方案收窄）。这里的假向量按字符二元组散列，相似度有噪声，
// 所以只断言「过滤性质」与「两份方案各留各的」，不断言具体排序。
func TestHybridSearchFiltersApplyToSemanticRoad(t *testing.T) {
	st := newStore(t)
	ingestMD(t, st)
	saveA(t, st, "调查报告 乙", []saveSection{{1, "降水沉降处置", "降水的时候旁边房子沉了怎么办：立即回灌并停止降水。"}},
		func(in *SaveInput) { in.Meta.ProjectType = "调查报告"; in.Meta.Year = "2025" })
	ctx := context.Background()
	em := &fakeEmbedder{dim: 32}
	if _, _, err := st.EmbedPending(ctx, em, "fake-embed", 0, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	sem := &Semantic{Embed: em, Model: "fake-embed"}
	q := "降水的时候旁边房子沉了怎么办" // 全文路零命中，只走语义路

	check := func(label string, in SearchInput, ok func(h Hit) bool) []Hit {
		t.Helper()
		res, err := st.HybridSearch(ctx, in, sem)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if len(res.Hits) == 0 {
			t.Fatalf("%s: 应有命中", label)
		}
		for _, h := range res.Hits {
			if !ok(h) {
				t.Fatalf("%s: 混入不符合过滤的命中 L%d %s（%s）", label, h.Level, h.Title, h.ProposalName)
			}
		}
		return res.Hits
	}
	all := check("不过滤", SearchInput{Query: q, TopN: 10}, func(Hit) bool { return true })
	sawA, sawB := false, false
	for _, h := range all {
		sawA = sawA || h.ProposalName == "包3技术标 第三版"
		sawB = sawB || h.ProposalName == "调查报告 乙"
	}
	if !sawA || !sawB {
		t.Fatalf("不过滤时两份方案都应有语义命中: %+v", all)
	}
	check("层级=1", SearchInput{Query: q, Level: 1, TopN: 10}, func(h Hit) bool { return h.Level == 1 })
	check("层级=2", SearchInput{Query: q, Level: 2, TopN: 10}, func(h Hit) bool { return h.Level == 2 })
	check("类型=技术标", SearchInput{Query: q, ProjectType: "技术标", TopN: 10},
		func(h Hit) bool { return h.ProposalName == "包3技术标 第三版" })
	check("类型=调查报告", SearchInput{Query: q, ProjectType: "调查报告", TopN: 10},
		func(h Hit) bool { return h.ProposalName == "调查报告 乙" })
	check("年度=2025", SearchInput{Query: q, Year: "2025", TopN: 10},
		func(h Hit) bool { return h.ProposalName == "调查报告 乙" })
	// 关键词 + 语义都有候选时：限定 3 级只剩 4.3.1（4.3 是 2 级，语义再相关也不进）
	res, err := st.HybridSearch(ctx, SearchInput{Query: "边坡失稳", Level: 3, TopN: 10}, sem)
	if err != nil || len(res.Hits) != 1 || res.Hits[0].Title != "4.3.1 危险源清单" {
		t.Fatalf("层级=3 应只剩 4.3.1: %v %+v", err, res.Hits)
	}
}
