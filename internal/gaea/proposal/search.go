package proposal

// search.go — 方案库全文路检索：关键词匹配（全文索引 + 两字词兜底）、分面计数、
// 分页、排序。语义路与混合检索见 semantic.go。
//
// 2026-11-28 索引优化批：此前一次检索最多回 30 条、没有总数也没有归并——常见词
// （「监测」在 450 多个小节里出现）只能看到前 30 条。现在全文路一次取回全部命中
// （上限 matchCap），回执带总数与分面（按方案 / 按章），调用方按偏移翻页；可以
// 按文档顺序列（限定到一份方案或一章之后按标题序过一遍）；限定在某标题之下检索；
// 多词全含一个都没有时放宽为任一词并注明。

import (
	"database/sql"
	"errors"
	"sort"
	"strconv"
	"strings"
)

// 排序方式。
const (
	SortRelevance = ""    // 默认：相关度
	SortDoc       = "doc" // 文档顺序：方案（清单序）× 标题序
)

const (
	// matchCap 一次检索最多取回的全文路命中数；分面计数与翻页都在这之内。
	matchCap = 2000
	// seqHiMax 「限定在某标题之下」右端无界时的上界（标题 seq 远小于它）。
	seqHiMax = 1<<31 - 1
)

// SearchInput 检索输入。
type SearchInput struct {
	Query       string
	Level       int    // 标题层级过滤 1~4（0=不过滤）
	ProjectType string // 项目类型过滤
	Year        string // 年度过滤
	Proposal    string // 限定方案：name 或标题（精确）
	Scope       string // ScopeBody（默认）| ScopeTitle
	TopN        int    // 本页条数（默认 10，上限 30）
	Offset      int    // 翻页：跳过前 Offset 条（按最终排序）
	Sort        string // SortRelevance（默认）| SortDoc
	Under       int64  // 限定在某个标题之下（含它的全部子孙小节）
	Facets      bool   // 是否要分面计数（界面用；模型工具面不要）

	pid          int64 // Proposal / Under 解析后的方案 id（resolve 后有效）
	seqLo, seqHi int   // Under 解析出的标题序区间 [seqLo, seqHi)（Under=0 时不用）
}

// Hit 是一次检索命中：定位到小节，带方案上下文与命中位置段落。
type Hit struct {
	ProposalName  string   `json:"proposalName"`
	ProposalTitle string   `json:"proposalTitle"`
	ProjectType   string   `json:"projectType"`
	Year          string   `json:"year"`
	Status        string   `json:"status"`
	SectionID     int64    `json:"sectionId"`
	Level         int      `json:"level"`
	Title         string   `json:"title"`
	TitlePath     string   `json:"titlePath"`
	Chars         int      `json:"chars"`
	ChildCount    int      `json:"childCount"`
	Snippet       string   `json:"snippet"`
	Matched       []string `json:"matched"`
	// Source 命中来源：keyword（全文路）/ semantic（只被语义路召回）/ both。
	Source        string  `json:"source,omitempty"`
	KeywordScore  float64 `json:"keywordScore,omitempty"`
	SemanticScore float64 `json:"semanticScore,omitempty"`
	RerankScore   float64 `json:"rerankScore,omitempty"`
	Children      []Child `json:"children,omitempty"`

	evidence string // 精排用：标题路径 + 最相关段落（不出网）
}

// Facet 分面计数的一格：按方案时 Key=方案名；按章时 Key=章标题所在小节的 id。
type Facet struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// SearchResult 检索回执数据。
type SearchResult struct {
	Hits []Hit  `json:"hits"`
	Mode string `json:"mode"` // title / keyword / hybrid
	Note string `json:"note,omitempty"`
	// Total 满足全部过滤的命中总数（混合检索时 = 全文路命中 + 只被语义路召回的候选）。
	Total int `json:"total"`
	// Offset 本页跳过的条数；翻页按 Offset += len(Hits) 接着取。
	Offset int `json:"offset"`
	// ByProposal 按方案计数（不带方案 / 标题范围过滤）；ByChapter 按章计数（限定在
	// 一份方案里时才有，不带标题范围过滤）。只要 SearchInput.Facets 时才算。
	ByProposal []Facet `json:"byProposal"`
	ByChapter  []Facet `json:"byChapter"`
	// Relaxed 多词全含一个都没有，已放宽为任一词命中。
	Relaxed bool `json:"relaxed,omitempty"`
	// SearchError 语义路失败的原因（本地 embedding 不可用等）；此时只按全文路检索。
	SearchError string `json:"searchError,omitempty"`
}

func (in *SearchInput) normalize() {
	if in.TopN <= 0 {
		in.TopN = 10
	}
	if in.TopN > 30 {
		in.TopN = 30
	}
	if in.Level < 0 || in.Level > MaxLevels {
		in.Level = 0
	}
	if in.Offset < 0 {
		in.Offset = 0
	}
	if in.Sort != SortDoc {
		in.Sort = SortRelevance
	}
}

// resolve 把 Proposal / Under 过滤解析成方案 id 与标题序区间；指定了但不存在返回 false。
func (s *Store) resolve(in *SearchInput) (bool, error) {
	if in.Under > 0 {
		var pid int64
		var level, seq int
		err := s.db.QueryRow(`SELECT proposal_id, level, seq FROM proposal_sections WHERE id=?`, in.Under).Scan(&pid, &level, &seq)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, unreadable(err)
		}
		var end sql.NullInt64
		if err := s.db.QueryRow(`SELECT MIN(seq) FROM proposal_sections WHERE proposal_id=? AND seq>? AND level<=?`, pid, seq, level).Scan(&end); err != nil {
			return false, unreadable(err)
		}
		in.pid, in.seqLo, in.seqHi = pid, seq, seqHiMax
		if end.Valid {
			in.seqHi = int(end.Int64)
		}
		return true, nil
	}
	if strings.TrimSpace(in.Proposal) == "" {
		return true, nil
	}
	p, err := s.Resolve(strings.TrimSpace(in.Proposal))
	if err != nil {
		return false, err
	}
	if p == nil {
		return false, nil
	}
	in.pid = p.ID
	return true, nil
}

type candRow struct {
	id, pid                              int64
	seq                                  int
	level                                int
	title                                string
	titlePath                            string
	body                                 string
	chars                                int
	pname, ptitle, ptype, pyear, pstatus string
	rank                                 float64
}

// splitLongShort 按长度把检索词分成全文索引短语（≥3 字符）与 LIKE 兜底词
// （<3 字符——trigram 下更短的短语恒零行，不能留在 MATCH 里）。
func splitLongShort(terms []string) (long, short []string) {
	for _, t := range terms {
		if len([]rune(t)) >= 3 {
			long = append(long, t)
		} else {
			short = append(short, t)
		}
	}
	return long, short
}

// candCols 候选行的列；排序用第 14 列（得分）与第 3 列（标题序）。
const candCols = `s.id, s.proposal_id, s.seq, s.level, s.title, s.title_path, s.body, LENGTH(s.body),
  p.name, p.title, p.project_type, p.year, p.status`

// matchFilter 一组过滤开关：算分面时故意不带自己那一维的过滤，计数才不会恒为 1。
type matchFilter struct {
	proposal bool // 限定方案
	under    bool // 限定在某标题之下
}

// buildMatch 拼一次全文路匹配的 FROM / WHERE / 排序。≥3 字的词走全文索引短语，
// 更短的词走 LIKE；any=true 时全部词改走 LIKE、任一命中即可（放宽模式——trigram
// 索引里两字词恒零行，没法和长短词混在一个 OR 里）。terms 须非空。
func buildMatch(in SearchInput, terms []string, anyHit bool, f matchFilter) (from, where, rank, order string, whereArgs, rankArgs, orderArgs []any) {
	join := " AND "
	if anyHit {
		join = " OR "
	}
	from, rank, order = `proposal_sections s`, `0`, `s.proposal_id, s.seq`
	var conds []string
	fts, short := splitLongShort(terms)
	if anyHit {
		fts, short = nil, terms
	}
	if in.Scope == ScopeTitle {
		for _, t := range terms { // 标题路：标题或标题路径含全部词，标题本身命中者优先
			conds = append(conds, `(s.title LIKE ? ESCAPE '\' OR s.title_path LIKE ? ESCAPE '\')`)
			pat := likePattern(t)
			whereArgs = append(whereArgs, pat, pat)
		}
		order = `(CASE WHEN s.title LIKE ? ESCAPE '\' THEN 0 ELSE 1 END), s.level, s.seq`
		orderArgs = append(orderArgs, likePattern(terms[0]))
	} else if len(fts) > 0 {
		parts := make([]string, len(fts))
		for i, t := range fts {
			parts[i] = quoteFTS(t)
		}
		from = `proposal_fts f JOIN proposal_sections s ON s.id = f.rowid`
		conds = append(conds, `proposal_fts MATCH ?`)
		whereArgs = append(whereArgs, strings.Join(parts, join))
		rank, order = `bm25(proposal_fts, 6.0, 1.0)`, `14, 3`
	} else if len(short) > 0 {
		// 只有短词、无 bm25 可用：按命中次数排序（标题命中加权）。
		occ := make([]string, 0, len(short))
		for range short {
			occ = append(occ, `((LENGTH(s.body)-LENGTH(REPLACE(s.body, ?, ''))) + 6*(LENGTH(s.title)-LENGTH(REPLACE(s.title, ?, ''))) + 2*(LENGTH(s.title_path)-LENGTH(REPLACE(s.title_path, ?, ''))))`)
		}
		rank, order = `-(`+strings.Join(occ, " + ")+`)`, `14, 3`
		for _, t := range short {
			rankArgs = append(rankArgs, t, t, t)
		}
	}
	from += ` JOIN proposals p ON p.id = s.proposal_id`
	if in.Scope != ScopeTitle && len(short) > 0 {
		likes := make([]string, 0, len(short))
		for _, t := range short {
			likes = append(likes, `(s.body LIKE ? ESCAPE '\' OR s.title LIKE ? ESCAPE '\' OR s.title_path LIKE ? ESCAPE '\')`)
			pat := likePattern(t)
			whereArgs = append(whereArgs, pat, pat, pat)
		}
		conds = append(conds, `(`+strings.Join(likes, join)+`)`)
	}
	if f.proposal && in.pid > 0 {
		conds = append(conds, `s.proposal_id = ?`)
		whereArgs = append(whereArgs, in.pid)
	}
	if f.under && in.Under > 0 {
		conds = append(conds, `s.proposal_id = ? AND s.seq >= ? AND s.seq < ?`)
		whereArgs = append(whereArgs, in.pid, in.seqLo, in.seqHi)
	}
	if in.Level > 0 {
		conds = append(conds, `s.level = ?`)
		whereArgs = append(whereArgs, in.Level)
	}
	if in.ProjectType != "" {
		conds = append(conds, `p.project_type = ?`)
		whereArgs = append(whereArgs, in.ProjectType)
	}
	if in.Year != "" {
		conds = append(conds, `p.year = ?`)
		whereArgs = append(whereArgs, in.Year)
	}
	if in.Sort == SortDoc && in.Scope != ScopeTitle {
		order, orderArgs = `p.updated_at DESC, p.id DESC, s.seq`, nil
	}
	return from, strings.Join(conds, " AND "), rank, order, whereArgs, rankArgs, orderArgs
}

// runMatch 跑一遍匹配：返回命中行（最多 limit 条；limit=0 只计数）与命中总数。
func (s *Store) runMatch(in SearchInput, terms []string, anyHit bool, f matchFilter, limit int) ([]candRow, int, error) {
	from, where, rank, order, whereArgs, rankArgs, orderArgs := buildMatch(in, terms, anyHit, f)
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM `+from+` WHERE `+where, whereArgs...).Scan(&total); err != nil {
		return nil, 0, unreadable(err)
	}
	if total == 0 || limit <= 0 {
		return nil, total, nil
	}
	// 参数按占位符在语句里出现的先后绑定：SELECT 列表里的得分表达式 → WHERE →
	// 末尾 ORDER BY 里的参数 → LIMIT。顺序错了 SQLite 不报错，只是悄悄绑定错位。
	args := append(append(append([]any{}, rankArgs...), whereArgs...), orderArgs...)
	rows, err := s.db.Query(`SELECT `+candCols+`, `+rank+`
FROM `+from+`
WHERE `+where+`
ORDER BY `+order+`
LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, 0, unreadable(err)
	}
	out, err := scanCands(rows)
	return out, total, err
}

// kwResult 全文路一次匹配的结果。rows 是全部命中行（SQL 序）；rows[:head] 是
// 「加强排序窗口」——窗口内另按全文路得分重排（标题 / 路径命中加分），其后保持
// SQL 序，拼起来就是最终顺序，翻页在它上面切。
type kwResult struct {
	rows    []candRow
	head    int
	total   int
	relaxed bool
	title   bool // 标题路

	headHits []Hit // 窗口内重排后的命中（按需算，算过即缓存）
	headDone bool
}

// keywordMatch 跑全文路。多词全含一个都没有时放宽为任一词（记 Relaxed）。
// headN=0 不设窗口；limit 是取回的行数上限。
func (s *Store) keywordMatch(in SearchInput, terms []string, headN, limit int) (*kwResult, error) {
	kw := &kwResult{title: in.Scope == ScopeTitle, head: headN}
	anyHit := false
	all := matchFilter{proposal: true, under: true}
	if len(terms) > 1 { // 先看全含有没有结果，没有再放宽
		if _, n, err := s.runMatch(in, terms, false, all, 0); err != nil {
			return nil, err
		} else if n == 0 {
			anyHit = true
		}
	}
	rows, total, err := s.runMatch(in, terms, anyHit, all, limit)
	if err != nil {
		return nil, err
	}
	kw.rows, kw.total, kw.relaxed = rows, total, anyHit && total > 0
	if kw.head > len(rows) {
		kw.head = len(rows)
	}
	return kw, nil
}

// kwHits 取全文路最终顺序里 [from, to) 的一段。文档顺序模式不分窗口。
func (s *Store) kwHits(kw *kwResult, in SearchInput, terms []string, from, to int) ([]Hit, error) {
	if to > kw.total {
		to = kw.total
	}
	if from >= to {
		return nil, nil
	}
	if in.Sort == SortDoc || kw.head == 0 {
		return s.enrich(kw.rows[from:min(to, len(kw.rows))], terms, kw.title, true)
	}
	if !kw.headDone {
		hits, err := s.enrich(kw.rows[:kw.head], terms, kw.title, false)
		if err != nil {
			return nil, err
		}
		kw.headHits, kw.headDone = hits, true
	}
	out := append([]Hit(nil), kw.headHits[from:min(to, len(kw.headHits))]...)
	if to > kw.head {
		tail, err := s.enrich(kw.rows[kw.head:min(to, len(kw.rows))], terms, kw.title, true)
		if err != nil {
			return nil, err
		}
		out = append(out, tail...)
	}
	return out, nil
}

// facets 分面计数：按方案（不带方案 / 标题范围过滤）；按章只在一份方案里有意义
// （带上方案过滤、不带标题范围过滤，章=命中所在的一级标题，按标题序排）。
func (s *Store) facets(in SearchInput, terms []string, anyHit bool) (byProposal, byChapter []Facet, err error) {
	from, where, _, _, args, _, _ := buildMatch(in, terms, anyHit, matchFilter{})
	rows, err := s.db.Query(`SELECT p.name, p.title, COUNT(*) FROM `+from+` WHERE `+where+`
GROUP BY p.id ORDER BY p.updated_at DESC, p.id DESC`, args...)
	if err != nil {
		return nil, nil, unreadable(err)
	}
	var facets []Facet
	for rows.Next() {
		var f Facet
		if err := rows.Scan(&f.Key, &f.Label, &f.Count); err != nil {
			_ = rows.Close()
			return nil, nil, unreadable(err)
		}
		facets = append(facets, f)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, unreadable(err)
	}
	byProposal = facets
	if in.pid <= 0 || in.Scope == ScopeTitle {
		return byProposal, nil, nil
	}
	matched, _, err := s.runMatch(in, terms, anyHit, matchFilter{proposal: true}, matchCap)
	if err != nil {
		return nil, nil, err
	}
	chaps, err := s.db.Query(`SELECT id, seq, title FROM proposal_sections WHERE proposal_id=? AND level=1 ORDER BY seq`, in.pid)
	if err != nil {
		return nil, nil, unreadable(err)
	}
	type chapter struct {
		id    int64
		seq   int
		title string
		count int
	}
	var chapters []chapter
	for chaps.Next() {
		var c chapter
		if err := chaps.Scan(&c.id, &c.seq, &c.title); err != nil {
			_ = chaps.Close()
			return nil, nil, unreadable(err)
		}
		chapters = append(chapters, c)
	}
	_ = chaps.Close()
	if err := chaps.Err(); err != nil {
		return nil, nil, unreadable(err)
	}
	for _, r := range matched {
		i := sort.Search(len(chapters), func(k int) bool { return chapters[k].seq > r.seq }) - 1
		if i < 0 {
			continue // 排在首个一级标题之前的小节不归任何章
		}
		chapters[i].count++
	}
	for _, c := range chapters {
		if c.count > 0 {
			byChapter = append(byChapter, Facet{Key: strconv.FormatInt(c.id, 10), Label: c.title, Count: c.count})
		}
	}
	return byProposal, byChapter, nil
}

func scanCands(rows *sql.Rows) ([]candRow, error) {
	defer func() { _ = rows.Close() }()
	var out []candRow
	for rows.Next() {
		var r candRow
		if err := rows.Scan(&r.id, &r.pid, &r.seq, &r.level, &r.title, &r.titlePath, &r.body, &r.chars,
			&r.pname, &r.ptitle, &r.ptype, &r.pyear, &r.pstatus, &r.rank); err != nil {
			return nil, unreadable(err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// rowsByIDs 按小节 id 批量取候选行（语义路独有命中补元数据用）。
func (s *Store) rowsByIDs(ids []int64) ([]candRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i], args[i] = "?", id
	}
	rows, err := s.db.Query(`SELECT `+candCols+`, 0
FROM proposal_sections s JOIN proposals p ON p.id = s.proposal_id
WHERE s.id IN (`+strings.Join(ph, ",")+`)`, args...)
	if err != nil {
		return nil, unreadable(err)
	}
	return scanCands(rows)
}

func termsIn(text string, terms []string) []string {
	low := strings.ToLower(text)
	var out []string
	for _, t := range terms {
		if strings.Contains(low, strings.ToLower(t)) {
			out = append(out, t)
		}
	}
	return out
}

// enrich 把候选行整理成命中：匹配词、命中位置段落、标题加权、子节清单。
// keepOrder=true 保持来料顺序（文档顺序模式、窗口之后的行），否则按全文路得分排。
func (s *Store) enrich(rows []candRow, terms []string, titleOnly, keepOrder bool) ([]Hit, error) {
	outlines := map[int64][]Section{}
	pos := map[int64]int{}
	hits := make([]Hit, 0, len(rows))
	for _, r := range rows {
		if _, ok := outlines[r.pid]; !ok {
			ol, err := s.Outline(r.pid)
			if err != nil {
				return nil, err
			}
			outlines[r.pid] = ol
			for i := range ol {
				pos[ol[i].ID] = i
			}
		}
		h := Hit{
			ProposalName: r.pname, ProposalTitle: r.ptitle, ProjectType: r.ptype, Year: r.pyear, Status: r.pstatus,
			SectionID: r.id, Level: r.level, Title: r.title, TitlePath: r.titlePath, Chars: r.chars, Source: "keyword",
		}
		bodyHits, titleHits, pathHits := termsIn(r.body, terms), termsIn(r.title, terms), termsIn(r.titlePath, terms)
		seen := map[string]bool{}
		for _, group := range [][]string{titleHits, pathHits, bodyHits} {
			for _, t := range group {
				if !seen[t] {
					seen[t] = true
					h.Matched = append(h.Matched, t)
				}
			}
		}
		h.KeywordScore = -r.rank + 2*float64(len(h.Matched)) + 3*float64(len(titleHits)) + float64(len(pathHits))
		if titleOnly {
			h.Snippet = markTerms(r.titlePath, terms)
		} else if ps := FindPassages(r.body, terms, 160, 240, 3); len(ps) > 0 {
			h.Snippet = RenderPassage([]rune(r.body), ps[0], terms)
		} else {
			h.Snippet = markTerms(truncRunes(r.titlePath, 120), terms)
		}
		h.evidence = h.Snippet
		if i, ok := pos[r.id]; ok {
			h.Children = childrenOf(outlines[r.pid], i)
			h.ChildCount = len(h.Children)
		}
		hits = append(hits, h)
	}
	if !keepOrder {
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].KeywordScore > hits[j].KeywordScore })
	}
	return hits, nil
}

// markTerms 在整段文本上给命中词加【】标记。
func markTerms(text string, terms []string) string {
	rs := []rune(text)
	return RenderPassage(rs, Passage{Start: 0, End: len(rs)}, terms)
}

// Search 只走关键词路（全文 + 标题），不带语义。空 query 返回空结果
// （方案清单/目录面由调用方另行取）。
func (s *Store) Search(in SearchInput) (*SearchResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	in.normalize()
	res := &SearchResult{Offset: in.Offset}
	terms := SplitTerms(in.Query)
	if len(terms) == 0 {
		return res, nil
	}
	if ok, err := s.resolve(&in); err != nil || !ok {
		return res, err
	}
	head := in.TopN*3 + 20
	if in.Scope == ScopeTitle {
		head = in.TopN*6 + 10
	}
	limit := head
	if in.Offset > 0 || in.Facets {
		limit = matchCap // 要翻页或算分面：取全
	}
	kw, err := s.keywordMatch(in, terms, head, limit)
	if err != nil {
		return nil, err
	}
	res.Relaxed, res.Total = kw.relaxed, kw.total
	res.Hits, err = s.kwHits(kw, in, terms, in.Offset, in.Offset+in.TopN)
	if err != nil {
		return nil, err
	}
	if in.Scope == ScopeTitle {
		res.Mode = "title"
	} else {
		res.Mode = "keyword"
	}
	if in.Facets {
		if res.ByProposal, res.ByChapter, err = s.facets(in, terms, kw.relaxed); err != nil {
			return nil, err
		}
	}
	return res, nil
}
