package proposal

// semantic.go — 方案库语义路与混合检索：小节正文切块、本地 embedding 向量化、
// 向量召回、与全文路融合、本地 reranker 精排。embedding/rerank 一律走仓内
// 现成的本地模型客户端（retrieval.Embedder / retrieval.Reranker，Herdsman
// OpenAI 兼容接口），不另造检索工具；本地模型不可用时自动降级为纯全文路。

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/gaea/gaea/internal/gaea/retrieval"
)

const (
	// ChunkMaxRunes 一个语义块正文的上限（rune）。
	ChunkMaxRunes = 480
	// ChunkMinRunes 遇到段落边界（空行）时，攒够这个数就成块。
	ChunkMinRunes = 160
	// embedBatch 每次请求向量化的块数。
	embedBatch = 32
	// MinCosine 语义召回的余弦下限（低于视为不相关）。
	MinCosine = 0.35
	// vecTopK 语义路最多召回的块数。
	vecTopK = 40
	// rerankPool 进入精排的候选小节上限。
	rerankPool = 30
	// rrfK 名次融合常数（倒数排名融合）。
	rrfK = 60.0
	// kwPool 混合检索时全文路的候选上限。
	kwPool = 30
	// SearchIndexBudget 检索前顺带补齐未向量化块的时间预算（超出留到下次）。
	SearchIndexBudget = 6 * time.Second
	// RerankTimeout 精排请求超时。
	RerankTimeout = 15 * time.Second
)

var (
	// ErrEmbedUnavailable 本地 embedding 客户端不可用。
	ErrEmbedUnavailable = errors.New("本地 embedding 不可用")
	// ErrEmbedModelMismatch 向量索引所用模型与当前模型不一致（向量不可比）。
	ErrEmbedModelMismatch = errors.New("语义索引的 embedding 模型与当前配置不一致")
)

// Embedder 向量化依赖的最小接口（*retrieval.Embedder 满足）。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Reranker 精排依赖的最小接口（*retrieval.Reranker 满足）。
type Reranker interface {
	Available(ctx context.Context) bool
	Rerank(ctx context.Context, query string, docs []string, topN int) ([]retrieval.ScoredDoc, error)
}

// Semantic 混合检索的本地模型依赖；任一为 nil 即跳过对应环节。
type Semantic struct {
	Embed  Embedder
	Model  string // embedding 模型名（与向量索引记录的模型核对）
	Rerank Reranker
}

// ── 切块 ───────────────────────────────────────────────────────────

// Chunk 是小节正文里的一个语义块：Start/Len 为它在正文中的 rune 区间。
type Chunk struct {
	Seq   int
	Start int
	Len   int
	Text  string // 正文原样切片（超长表格的后续组前面带重复表头）
}

type span struct{ start, end int } // [start,end) rune 区间，不含行尾换行

// ChunkBody 把小节正文按 Markdown 行聚块：表格整表聚拢（超长按行组切并重复
// 表头），其余行攒到 ChunkMaxRunes，或在空行处攒够 ChunkMinRunes 即成块。
func ChunkBody(body string) []Chunk {
	rs := []rune(body)
	if len(rs) == 0 {
		return nil
	}
	var lines []span
	start := 0
	for i, r := range rs {
		if r == '\n' {
			lines = append(lines, span{start, i})
			start = i + 1
		}
	}
	if start < len(rs) {
		lines = append(lines, span{start, len(rs)})
	}
	trim := func(sp span) span {
		for sp.start < sp.end && unicode.IsSpace(rs[sp.start]) {
			sp.start++
		}
		for sp.end > sp.start && unicode.IsSpace(rs[sp.end-1]) {
			sp.end--
		}
		return sp
	}
	var out []Chunk
	var cur []span
	curRunes := 0
	flush := func() {
		if len(cur) == 0 {
			return
		}
		st, en := cur[0].start, cur[len(cur)-1].end
		out = append(out, Chunk{Start: st, Len: en - st, Text: string(rs[st:en])})
		cur, curRunes = nil, 0
	}
	for i := 0; i < len(lines); {
		t := trim(lines[i])
		if t.start == t.end { // 空行=段落边界
			if curRunes >= ChunkMinRunes {
				flush()
			}
			i++
			continue
		}
		if rs[t.start] == '|' {
			j, tblRunes := i, 0
			for j < len(lines) {
				tj := trim(lines[j])
				if tj.start == tj.end || rs[tj.start] != '|' {
					break
				}
				tblRunes += tj.end - tj.start + 1
				j++
			}
			if tblRunes <= ChunkMaxRunes {
				if curRunes+tblRunes > ChunkMaxRunes {
					flush()
				}
				cur = append(cur, lines[i:j]...)
				curRunes += tblRunes
			} else {
				flush()
				out = append(out, splitTable(rs, lines[i:j])...)
			}
			i = j
			continue
		}
		n := t.end - t.start
		if n > ChunkMaxRunes {
			flush()
			out = append(out, splitLongLine(rs, lines[i])...)
			i++
			continue
		}
		if curRunes+n > ChunkMaxRunes {
			flush()
		}
		cur = append(cur, lines[i])
		curRunes += n + 1
		if curRunes >= ChunkMaxRunes {
			flush()
		}
		i++
	}
	flush()
	for i := range out {
		out[i].Seq = i
	}
	return out
}

func isSeparatorRow(rs []rune, sp span) bool {
	for i := sp.start; i < sp.end; i++ {
		if !strings.ContainsRune("|-: ", rs[i]) {
			return false
		}
	}
	return true
}

// splitTable 超长表格按行组切块，每组不超过约两个块上限；非首组前面重复表头
// （表头+分隔行）。Start/Len 指向该组的行区间（不含重复的表头）。
func splitTable(rs []rune, rows []span) []Chunk {
	header := 1
	if len(rows) >= 2 && isSeparatorRow(rs, rows[1]) {
		header = 2
	}
	headRunes := 0
	for k := 0; k < header && k < len(rows); k++ {
		headRunes += rows[k].end - rows[k].start + 1
	}
	var out []Chunk
	groupStart, runes := header, 0
	emit := func(end int) {
		if groupStart >= end {
			return
		}
		// 首组与表头在正文里相连，区间从表头起；后续组区间只覆盖本组行，文本前面补表头。
		st, en := rows[groupStart].start, rows[end-1].end
		var sb strings.Builder
		if groupStart > header {
			for k := 0; k < header; k++ {
				sb.WriteString(string(rs[rows[k].start:rows[k].end]))
				sb.WriteByte('\n')
			}
		} else {
			st = rows[0].start
		}
		sb.WriteString(string(rs[st:en]))
		out = append(out, Chunk{Start: st, Len: en - st, Text: sb.String()})
	}
	for k := groupStart; k < len(rows); k++ {
		n := rows[k].end - rows[k].start + 1
		if runes > 0 && headRunes+runes+n > ChunkMaxRunes*2 {
			emit(k)
			groupStart, runes = k, 0
		}
		runes += n
	}
	emit(len(rows))
	return out
}

// splitLongLine 超长单行按句读（。；！？）切段。
func splitLongLine(rs []rune, sp span) []Chunk {
	var out []Chunk
	st, lastCut := sp.start, -1
	for i := sp.start; i < sp.end; i++ {
		if i-st >= ChunkMaxRunes {
			cut := lastCut
			if cut < st {
				cut = i
			}
			out = append(out, Chunk{Start: st, Len: cut - st, Text: string(rs[st:cut])})
			st, lastCut = cut, -1
		}
		if strings.ContainsRune("。；！？", rs[i]) {
			lastCut = i + 1
		}
	}
	if st < sp.end {
		out = append(out, Chunk{Start: st, Len: sp.end - st, Text: string(rs[st:sp.end])})
	}
	return out
}

// insertChunks 为一个小节生成语义块（未向量化）。向量化原文=标题路径+换行+块正文
// （上下文前缀：段落单看没有信息，带上标题路径才知道在讲什么）。
func insertChunks(tx *sql.Tx, pid, sid int64, titlePath, body, now string) error {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	for i, c := range ChunkBody(body) {
		if _, err := tx.Exec(`INSERT INTO proposal_chunks(proposal_id, section_id, seq, start, len, text, updated_at)
VALUES(?,?,?,?,?,?,?)`, pid, sid, i, c.Start, c.Len, titlePath+"\n"+c.Text, now); err != nil {
			return err
		}
	}
	return nil
}

// ── 向量化 ─────────────────────────────────────────────────────────

// EmbedModel 返回向量索引记录的 embedding 模型名（尚无向量为空串）。
func (s *Store) EmbedModel() (string, error) {
	if err := s.available(); err != nil {
		return "", err
	}
	var model string
	err := s.db.QueryRow(`SELECT model FROM proposal_embed_meta WHERE id=1`).Scan(&model)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", unreadable(err)
	}
	return model, nil
}

// ResetEmbedding 切换向量索引的 embedding 模型：旧向量与新模型不可比，整库
// 置空待重嵌。只在写入侧（录入后重建索引）调用——读侧遇到模型不一致是跳过
// 语义路，不在这里来回清库。
func (s *Store) ResetEmbedding(model string) error {
	if _, err := s.db.Exec(`UPDATE proposal_chunks SET vec=NULL WHERE vec IS NOT NULL`); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO proposal_embed_meta(id, model, dim) VALUES(1, ?, 0)
ON CONFLICT(id) DO UPDATE SET model=excluded.model, dim=0`, model)
	return err
}

// PendingCount 返回未向量化的块数与总块数（proposalID=0 为全库）。
func (s *Store) PendingCount(proposalID int64) (pending, total int) {
	q := `SELECT COUNT(*), COALESCE(SUM(vec IS NOT NULL),0) FROM proposal_chunks`
	var args []any
	if proposalID > 0 {
		q += ` WHERE proposal_id=?`
		args = append(args, proposalID)
	}
	var indexed int
	if err := s.db.QueryRow(q, args...).Scan(&total, &indexed); err != nil {
		return 0, 0
	}
	return total - indexed, total
}

// EmbedPending 把未向量化的块补齐（proposalID=0 为全库），受 budget 约束，到期
// 即停、余量留给下次。模型与索引记录不一致时不动库，返回 ErrEmbedModelMismatch
// （调用方决定是否 ResetEmbedding 后重嵌）。返回本次向量化数与剩余待向量化的数。
func (s *Store) EmbedPending(ctx context.Context, e Embedder, model string, proposalID int64, budget time.Duration) (done, pending int, err error) {
	if err := s.available(); err != nil {
		return 0, 0, err
	}
	if e == nil || model == "" {
		return 0, 0, ErrEmbedUnavailable
	}
	cur, err := s.EmbedModel()
	if err != nil {
		return 0, 0, err
	}
	if cur != "" && cur != model {
		return 0, 0, ErrEmbedModelMismatch
	}
	if budget <= 0 {
		budget = SearchIndexBudget
	}
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	deadline := time.Now().Add(budget)
	for {
		// 预算判据看墙钟：context 的超时靠定时器异步触发（Windows 定时器粒度毫秒
		// 级），只看 ctx.Err() 会在极短预算下仍然多跑一批。
		if ctx.Err() != nil || !time.Now().Before(deadline) {
			break
		}
		q := `SELECT id, text FROM proposal_chunks WHERE vec IS NULL`
		var args []any
		if proposalID > 0 {
			q += ` AND proposal_id=?`
			args = append(args, proposalID)
		}
		args = append(args, embedBatch)
		rows, err := s.db.Query(q+` ORDER BY proposal_id, section_id, seq LIMIT ?`, args...)
		if err != nil {
			return done, 0, unreadable(err)
		}
		var ids []int64
		var texts []string
		for rows.Next() {
			var id int64
			var text string
			if err := rows.Scan(&id, &text); err != nil {
				_ = rows.Close()
				return done, 0, unreadable(err)
			}
			ids, texts = append(ids, id), append(texts, text)
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return done, 0, unreadable(err)
		}
		if len(ids) == 0 {
			break
		}
		vecs, err := e.Embed(ctx, texts)
		if err != nil {
			p, _ := s.PendingCount(proposalID)
			return done, p, err
		}
		n, err := s.storeVectors(ids, vecs, model)
		if err != nil {
			return done, 0, err
		}
		done += n
		if n == 0 {
			break // 全部被维度校验拒掉：不留死循环
		}
	}
	pending, _ = s.PendingCount(proposalID)
	return done, pending, nil
}

// storeVectors 落一批向量：float32 小端 BLOB；维度须与索引记录一致（首个向量
// 定维度）。返回实际落库条数。
func (s *Store) storeVectors(ids []int64, vecs [][]float32, model string) (int, error) {
	var dim int
	_ = s.db.QueryRow(`SELECT dim FROM proposal_embed_meta WHERE id=1`).Scan(&dim)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339)
	n := 0
	for i, id := range ids {
		if i >= len(vecs) || len(vecs[i]) == 0 {
			continue
		}
		if dim == 0 {
			dim = len(vecs[i])
		}
		if len(vecs[i]) != dim {
			continue
		}
		if _, err := tx.Exec(`UPDATE proposal_chunks SET vec=?, updated_at=? WHERE id=?`, packVec(vecs[i]), now, id); err != nil {
			return 0, err
		}
		n++
	}
	if n > 0 {
		if _, err := tx.Exec(`INSERT INTO proposal_embed_meta(id, model, dim) VALUES(1, ?, ?)
ON CONFLICT(id) DO UPDATE SET model=excluded.model, dim=excluded.dim`, model, dim); err != nil {
			return 0, err
		}
	}
	return n, tx.Commit()
}

func packVec(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(f))
	}
	return b
}

func cosineBlob(q []float32, b []byte) float64 {
	n := len(b) / 4
	if n == 0 || n != len(q) {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		x := math.Float32frombits(binary.LittleEndian.Uint32(b[4*i:]))
		y := q[i]
		dot += float64(x) * float64(y)
		na += float64(x) * float64(x)
		nb += float64(y) * float64(y)
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// ── 召回与融合 ─────────────────────────────────────────────────────

// ChunkHit 是一个语义命中的块。
type ChunkHit struct {
	ChunkID    int64
	ProposalID int64
	SectionID  int64
	Start, Len int
	Score      float64
}

// SemanticRecall 语义召回：只嵌 query，与已向量化的块做余弦，按分降序取 topK。
// 索引模型与当前模型不一致时不动库，返回 ErrEmbedModelMismatch。
func (s *Store) SemanticRecall(ctx context.Context, e Embedder, model, query string, proposalID int64, topK int) ([]ChunkHit, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	cur, err := s.EmbedModel()
	if err != nil || cur == "" {
		return nil, err
	}
	if cur != model {
		return nil, ErrEmbedModelMismatch
	}
	qv, err := e.Embed(ctx, []string{query})
	if err != nil || len(qv) == 0 || len(qv[0]) == 0 {
		if err == nil {
			err = ErrEmbedUnavailable
		}
		return nil, err
	}
	q := `SELECT c.id, c.proposal_id, c.section_id, c.start, c.len, c.vec FROM proposal_chunks c WHERE c.vec IS NOT NULL`
	var args []any
	if proposalID > 0 {
		q += ` AND c.proposal_id=?`
		args = append(args, proposalID)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = rows.Close() }()
	var hits []ChunkHit
	for rows.Next() {
		var h ChunkHit
		var vec []byte
		if err := rows.Scan(&h.ChunkID, &h.ProposalID, &h.SectionID, &h.Start, &h.Len, &vec); err != nil {
			return nil, unreadable(err)
		}
		if score := cosineBlob(qv[0], vec); score >= MinCosine {
			h.Score = score
			hits = append(hits, h)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, unreadable(err)
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ChunkID < hits[j].ChunkID
	})
	if topK > 0 && len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

// HybridSearch 混合检索：全文路（精确词）与语义路（段落向量）各出候选，按小节
// 用名次融合，再用本地 reranker 对候选小节精排。语义或精排不可用时逐级降级
// （全文路照常），不因本地模型状态让检索失败。
//
// 翻页（2026-11-28）：候选池（全文路前 kwPool 条 + 语义召回）的精排 / 融合序排在
// 最前，其后接没进候选池的其余全文命中（按全文路排序）；偏移超过候选池容量时
// 直接走纯全文路，不再调本地模型。语义路失败时退成纯全文检索并在回执里
// SearchError 注明原因。
func (s *Store) HybridSearch(ctx context.Context, in SearchInput, sem *Semantic) (*SearchResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	in.normalize()
	if sem == nil || sem.Embed == nil || strings.TrimSpace(in.Query) == "" || in.Scope == ScopeTitle ||
		in.Sort == SortDoc || in.Offset >= rerankPool {
		return s.Search(in)
	}
	if ok, err := s.resolve(&in); err != nil || !ok {
		return &SearchResult{Offset: in.Offset}, err
	}
	terms := SplitTerms(in.Query)
	pool := in
	pool.TopN, pool.Offset, pool.Facets = kwPool, 0, false
	limit := kwPool*3 + 20
	if in.Offset > 0 || in.Facets {
		limit = matchCap // 要翻页或算分面：取全
	}
	kw, err := s.keywordMatch(pool, terms, kwPool*3+20, limit)
	if err != nil {
		return nil, err
	}
	kwHead, err := s.kwHits(kw, pool, terms, 0, kwPool)
	if err != nil {
		return nil, err
	}

	// 语义路：先小预算补齐未向量化的块（模型一致才补），再召回。
	var vecHits []ChunkHit
	if pend, _ := s.PendingCount(pool.pid); pend > 0 {
		_, _, _ = s.EmbedPending(ctx, sem.Embed, sem.Model, pool.pid, SearchIndexBudget)
	}
	if vecHits, err = s.SemanticRecall(ctx, sem.Embed, sem.Model, in.Query, pool.pid, vecTopK); err != nil {
		kwRes, sErr := s.Search(in)
		if sErr != nil {
			return nil, sErr
		}
		kwRes.SearchError = err.Error()
		if errors.Is(err, ErrEmbedModelMismatch) {
			kwRes.Note = "语义索引的 embedding 模型已变更，旧向量不可用，本次只按全文路检索；重建索引后恢复语义检索"
		} else {
			kwRes.Note = "本地 embedding 不可用，本次只按全文路检索"
		}
		return kwRes, nil
	}
	res := &SearchResult{Mode: "hybrid", Offset: in.Offset, Total: kw.total, Relaxed: kw.relaxed}

	byID := map[int64]*cand{}
	var order []*cand
	for i, h := range kwHead {
		c := &cand{hit: h}
		c.fused += 1 / (rrfK + float64(i+1))
		byID[h.SectionID], order = c, append(order, c)
	}

	// 语义路独有命中的小节先取元数据并过一遍检索过滤。全文路的候选已在 SQL 里过滤过；
	// 语义召回只按方案收窄，层级/类型/年度/标题范围必须在这里补上——否则这些过滤条件
	// 会被语义路绕过（真实程序实测：限定 2 级却混进 4 级命中）。
	uniq := make([]ChunkHit, 0, len(vecHits))
	seen := map[int64]bool{}
	for _, vh := range vecHits {
		if !seen[vh.SectionID] {
			seen[vh.SectionID] = true
			uniq = append(uniq, vh)
		}
	}
	vecHit, vecBody := map[int64]Hit{}, map[int64]string{}
	var fetch []int64
	for _, vh := range uniq {
		if _, ok := byID[vh.SectionID]; !ok {
			fetch = append(fetch, vh.SectionID)
		}
	}
	if len(fetch) > 0 {
		rows, err := s.rowsByIDs(fetch)
		if err != nil {
			return nil, err
		}
		kept := make([]candRow, 0, len(rows))
		for _, r := range rows {
			if rowMatches(r, in) {
				kept = append(kept, r)
				vecBody[r.id] = r.body
			}
		}
		hs, err := s.enrich(kept, nil, false, true)
		if err != nil {
			return nil, err
		}
		for _, h := range hs {
			vecHit[h.SectionID] = h
		}
	}
	semOnly := 0
	rank := 0
	for _, vh := range uniq {
		c, ok := byID[vh.SectionID]
		if !ok {
			h, allowed := vecHit[vh.SectionID]
			if !allowed {
				continue // 被层级/类型/年度/标题范围过滤掉
			}
			h.Source = "semantic"
			c = &cand{hit: h}
			byID[vh.SectionID], order = c, append(order, c)
			semOnly++
			// 没有全文命中段落：摘要取语义路最相关的那一块原文。
			if rs := []rune(vecBody[vh.SectionID]); vh.Start <= len(rs) {
				end := min(vh.Start+vh.Len, len(rs))
				c.vecText = string(rs[vh.Start:end])
				c.hit.Snippet = c.vecText
			}
		}
		rank++
		v := vh
		c.vec, c.hit.SemanticScore = &v, vh.Score
		if c.hit.Source == "keyword" {
			c.hit.Source = "both"
		}
		c.fused += 1 / (rrfK + float64(rank))
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].fused > order[j].fused })
	if len(order) > rerankPool {
		order = order[:rerankPool]
	}
	var ranked []Hit
	reranked := false
	if sem.Rerank != nil {
		if list, ok := rerankCands(ctx, sem.Rerank, in.Query, order); ok {
			reranked = true
			for _, c := range list {
				ranked = append(ranked, c.hit)
			}
		}
	}
	if reranked {
		res.Note = "已本地精排"
	} else {
		for _, c := range order {
			ranked = append(ranked, c.hit)
		}
		if len(vecHits) > 0 {
			res.Note = "本地精排不可用，按全文与语义名次融合排序"
		} else {
			res.Note = "语义路无命中，按全文路排序"
		}
	}
	res.Total = kw.total + semOnly

	// 本页：候选池里从 Offset 起的一段，不够再从候选池之后的全文命中里接。
	off := in.Offset
	var page []Hit
	if off < len(ranked) {
		page = append(page, ranked[off:min(len(ranked), off+in.TopN)]...)
	}
	if len(page) < in.TopN {
		from := len(kwHead) + max(0, off-len(ranked))
		more, err := s.kwHits(kw, pool, terms, from, from+in.TopN-len(page))
		if err != nil {
			return nil, err
		}
		page = append(page, more...)
	}
	res.Hits = page
	if in.Facets {
		if res.ByProposal, res.ByChapter, err = s.facets(in, terms, kw.relaxed); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// cand 是混合检索里的一个候选小节。
type cand struct {
	hit     Hit
	vec     *ChunkHit // 语义路最相关的块（无语义命中为 nil）
	vecText string    // 该块在小节正文里的原文
	fused   float64   // 名次融合分
}

// evidence 拼精排文档串：标题路径 + 最相关段落。全文命中段落优先；全文路只
// 命中标题（摘要就是标题路径）时改用语义命中块。
func evidence(c *cand) string {
	body := strings.NewReplacer("【", "", "】", "").Replace(c.hit.Snippet)
	if body == c.hit.TitlePath {
		body = c.vecText
	}
	return c.hit.TitlePath + "\n" + truncRunes(body, 700)
}

// rerankCands 用本地 reranker 给候选小节排序；不可用或失败返回 false（调用方
// 沿用融合名次）。
func rerankCands(ctx context.Context, r Reranker, query string, cands []*cand) ([]*cand, bool) {
	if len(cands) <= 1 {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(ctx, RerankTimeout)
	defer cancel()
	if !r.Available(ctx) {
		return nil, false
	}
	docs := make([]string, len(cands))
	for i, c := range cands {
		docs[i] = evidence(c)
	}
	scored, err := r.Rerank(ctx, query, docs, len(cands))
	if err != nil || len(scored) == 0 {
		return nil, false
	}
	out := make([]*cand, 0, len(scored))
	for _, sc := range scored {
		if sc.Index < 0 || sc.Index >= len(cands) {
			continue
		}
		c := cands[sc.Index]
		c.hit.RerankScore = sc.Score
		out = append(out, c)
	}
	return out, true
}

// rowMatches 判别一个小节是否满足检索过滤（层级/类型/年度/标题范围；限定方案已在
// 召回时收窄）。
func rowMatches(r candRow, in SearchInput) bool {
	if in.Level > 0 && r.level != in.Level {
		return false
	}
	if in.ProjectType != "" && r.ptype != in.ProjectType {
		return false
	}
	if in.Year != "" && r.pyear != in.Year {
		return false
	}
	if in.Under > 0 && (r.seq < in.seqLo || r.seq >= in.seqHi) {
		return false
	}
	return true
}
