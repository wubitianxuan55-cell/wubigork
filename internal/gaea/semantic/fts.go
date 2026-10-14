package semantic

// ── 混合召回：FTS5 全文路 + 名称关键词路（阶段九 9.2）────────
//
// 与既有向量路（SearchReady，bge-m3 cosine）融合为三路：全文 / 名称 / 向量。
// 证据（competitor survey §5.5）：全文检索 67% vs 纯向量 14-18%——精确词
// （型号/编号/专名）全文路是主力，向量路补语义变体；名称命中是最强意图信号。
//
// 中文降级：FTS5 unicode61 不切 CJK 子串，MATCH 零命中回退 LIKE 子串扫描
// （session_fts/whisper 同款先例）。行维护=惰性重建（FTS 行数 vs 向量行数
// 不一致即全量重灌，量级毫秒级）。embedding 不可用时全文+名称两路照常工作
// （混合召回的降级红利：检索不再被模型可用性绑架）。

import (
	"fmt"
	"sort"
	"strings"
)

// EnsureFTS 惰性增量维护：FTS 行数与向量行数不一致时全量重灌（幂等）。
// 返回是否重建。db 不可用/行数一致时不做事。
func (s *Store) EnsureFTS() (bool, error) {
	if s.db == nil {
		return false, nil
	}
	var vecCount, ftsCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM semantic_vectors`).Scan(&vecCount); err != nil {
		return false, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM semantic_fts`).Scan(&ftsCount); err != nil {
		return false, fmt.Errorf("fts count: %w", err)
	}
	if vecCount == ftsCount {
		return false, nil
	}
	if _, err := s.db.Exec(`DELETE FROM semantic_fts`); err != nil {
		return false, err
	}
	if vecCount == 0 {
		return true, nil
	}
	if _, err := s.db.Exec(`INSERT INTO semantic_fts(doc, kind, id)
		SELECT doc, kind, id FROM semantic_vectors WHERE TRIM(doc) != ''`); err != nil {
		return false, err
	}
	return true, nil
}

// FTSHit 全文路命中。FTS 路的 Score=-rank（bm25 越小越相关，取负后大者相关）；
// LIKE 路的 Score=命中的查询词数。两路分不同量纲，融合层各自归一。
// UpdatedAt=semantic_vectors.updated_at（时序 tie-break 输入，JOIN 带出）。
type FTSHit struct {
	Kind      string
	ID        string
	Score     float64
	UpdatedAt string
}

// FTSSearch 全文路检索：FTS5 MATCH 优先（bm25 排序），零命中/出错回退 LIKE
// 子串扫描（每词计 1 分）。query 按空白/中文标点切词，词间 OR。返回 Score
// 降序、上限 topN。
func (s *Store) FTSSearch(query string, topN int) ([]FTSHit, error) {
	if s.db == nil || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	terms := SplitQueryTerms(query)
	if len(terms) == 0 {
		return nil, nil
	}
	if topN <= 0 {
		topN = 20
	}

	// FTS5 MATCH：词间 OR，rank（bm25，越小越相关）取负作 Score；JOIN 向量表
	// 带出 updated_at（时序 tie-break）。
	match := strings.Join(escapeFTS(terms), " OR ")
	rows, err := s.db.Query(`
SELECT f.kind, f.id, -f.rank, COALESCE(v.updated_at,'')
FROM semantic_fts f
LEFT JOIN semantic_vectors v ON v.kind = f.kind AND v.id = f.id
WHERE semantic_fts MATCH ?
ORDER BY rank LIMIT ?`, match, topN)
	if err == nil {
		hits, scanErr := scanFTSHits(rows)
		if scanErr == nil && len(hits) > 0 {
			return hits, nil
		}
		// MATCH 执行成功但零命中（unicode61 不切 CJK）→ 落 LIKE 降级
	} else {
		rows.Close() // MATCH 语法错等：落 LIKE 降级，不让连接锁残留
	}

	// LIKE 降级：每词一计，多词全中排前（确定性：同分按 kind+id 字典序）。
	conds := make([]string, len(terms))
	args := make([]any, 0, len(terms))
	for i, t := range terms {
		conds[i] = `doc LIKE ? ESCAPE '\'`
		args = append(args, "%"+escapeLike(t)+"%")
	}
	rows2, err := s.db.Query(
		`SELECT kind, id, doc, updated_at FROM semantic_vectors WHERE `+strings.Join(conds, " OR "), args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	hits := make([]FTSHit, 0, 16)
	for rows2.Next() {
		var kind, id, doc, updatedAt string
		if err := rows2.Scan(&kind, &id, &doc, &updatedAt); err != nil {
			return nil, err
		}
		n := 0
		lowerDoc := strings.ToLower(doc)
		for _, t := range terms {
			if strings.Contains(lowerDoc, strings.ToLower(t)) {
				n++
			}
		}
		if n > 0 {
			hits = append(hits, FTSHit{Kind: kind, ID: id, Score: float64(n), UpdatedAt: updatedAt})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Kind+hits[i].ID < hits[j].Kind+hits[j].ID
	})
	if len(hits) > topN {
		hits = hits[:topN]
	}
	return hits, nil
}

// SplitQueryTerms 查询切词：空白与中文标点分隔（FTS 词法与 LIKE 计分共用）。
func SplitQueryTerms(query string) []string {
	fields := strings.FieldsFunc(query, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '，', '、', '　', '。', '；', '：':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func scanFTSHits(rows interface {
	Next() bool
	Scan(...any) error
	Close() error
}) ([]FTSHit, error) {
	defer rows.Close()
	var hits []FTSHit
	for rows.Next() {
		var kind, id, updatedAt string
		var score float64
		if err := rows.Scan(&kind, &id, &score, &updatedAt); err != nil {
			return nil, err
		}
		hits = append(hits, FTSHit{Kind: kind, ID: id, Score: score, UpdatedAt: updatedAt})
	}
	return hits, rows.Close()
}

// escapeFTS FTS5 MATCH 词转义：双引号包裹（内部双引号双写），防御语法注入。
func escapeFTS(terms []string) []string {
	out := make([]string, len(terms))
	for i, t := range terms {
		out[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	return out
}

// escapeLike LIKE 通配符转义（配合 ESCAPE '\'）。
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
