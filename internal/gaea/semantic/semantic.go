// Package semantic 本地语义向量索引：共享 semantic_vectors 表（Hephaestus.db
// SchemaV5），按 (kind,id) 存 bge-m3 向量 JSON。Ensure 增量向量化（只处理缺失
// 项），Search 只嵌 query + 余弦相似度，避免每查询全量批量 embedding；跨库
// 统一语义检索用 SearchMany 合并多 kind。纯本地推理，不消耗云端 token。
package semantic

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/retrieval"
)

const (
	// EmbedBatch 每批向量化的文档数（bge-m3 批量上限内）。
	EmbedBatch = 64
	// MinCosine 语义召回的余弦阈值（低于视为不相关）。
	MinCosine = 0.1
)

// Doc 是参与向量索引的一条文档。
type Doc struct {
	ID   string
	Text string
}

// Hit 是语义检索结果。
type Hit struct {
	Kind  string  `json:"kind"`
	ID    string  `json:"id"`
	Score float64 `json:"score"`
	Text  string  `json:"text"`
}

// Store 向量索引存储。
type Store struct {
	db *sql.DB
}

// Open 打开向量索引存储；gdb 为 nil 时不可用。
func Open(gdb *sql.DB) *Store { return &Store{db: gdb} }

// Available 报告存储是否可用。
func (s *Store) Available() bool { return s.db != nil }

// Counts 返回各 kind 的向量条数（D3-1 索引状态可见性）。
func (s *Store) Counts() (map[string]int, error) {
	if s.db == nil {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT kind, COUNT(*) FROM semantic_vectors GROUP BY kind`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var kind string
		var n int
		if rows.Scan(&kind, &n) == nil {
			out[kind] = n
		}
	}
	return out, rows.Err()
}

// Ensure 增量向量化：缺失向量或正文快照变化的文档重新向量化（分批），
// 返回新增/更新条数。正文快照比对保证「内容变更自动重嵌」，编辑过的
// 条目不会被陈旧向量命中。
func (s *Store) Ensure(ctx context.Context, e *retrieval.Embedder, kind string, docs []Doc) (int, error) {
	if s.db == nil || e == nil || len(docs) == 0 {
		return 0, nil
	}
	have, err := s.vectorDocs(kind)
	if err != nil {
		return 0, err
	}
	var missing []Doc
	for _, d := range docs {
		if d.ID == "" {
			continue
		}
		if strings.TrimSpace(d.Text) == "" {
			continue
		}
		if doc, ok := have[d.ID]; ok && doc == d.Text {
			continue
		}
		missing = append(missing, d)
	}
	added := 0
	now := time.Now().UTC().Format(time.RFC3339)
	for start := 0; start < len(missing); start += EmbedBatch {
		end := start + EmbedBatch
		if end > len(missing) {
			end = len(missing)
		}
		batch := missing[start:end]
		texts := make([]string, len(batch))
		for i, d := range batch {
			texts[i] = d.Text
		}
		vecs, err := e.Embed(ctx, texts)
		if err != nil {
			return added, err
		}
		for i, v := range vecs {
			if len(v) == 0 {
				continue
			}
			b, _ := json.Marshal(v)
			if err := s.upsert(kind, batch[i].ID, string(b), batch[i].Text, now); err != nil {
				return added, err
			}
			added++
		}
	}
	return added, nil
}

// Search 对指定 kind 做语义检索：确保向量就绪后只嵌 query，余弦 topN 降序。
func (s *Store) Search(ctx context.Context, e *retrieval.Embedder, kind string, docs []Doc, query string, topN int) ([]Hit, error) {
	if _, err := s.Ensure(ctx, e, kind, docs); err != nil {
		return nil, err
	}
	return s.SearchReady(ctx, e, kind, query, topN)
}

// SearchMany 跨多 kind 统一语义检索：每 kind 取 topN 后合并按得分降序。
func (s *Store) SearchMany(ctx context.Context, e *retrieval.Embedder, kindDocs map[string][]Doc, query string, perKind, topN int) ([]Hit, error) {
	var all []Hit
	for kind, docs := range kindDocs {
		hits, err := s.Search(ctx, e, kind, docs, query, perKind)
		if err != nil {
			return nil, err
		}
		all = append(all, hits...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
	if len(all) > topN {
		all = all[:topN]
	}
	return all, nil
}

// SearchReady 在向量已就绪的前提下查询（只嵌 query），供已 Ensure 的路径复用。
func (s *Store) SearchReady(ctx context.Context, e *retrieval.Embedder, kind, query string, topN int) ([]Hit, error) {
	if s.db == nil || e == nil || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT id, vec, doc FROM semantic_vectors WHERE kind=?`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type item struct {
		id   string
		text string
		vec  []float32
	}
	var items []item
	for rows.Next() {
		var id, vecRaw, doc string
		if err := rows.Scan(&id, &vecRaw, &doc); err != nil {
			continue
		}
		var vec []float32
		if json.Unmarshal([]byte(vecRaw), &vec) == nil && len(vec) > 0 {
			items = append(items, item{id: id, text: doc, vec: vec})
		}
	}
	if len(items) == 0 {
		return nil, nil
	}
	qvec, err := e.Embed(ctx, []string{query})
	if err != nil || len(qvec) == 0 {
		return nil, err
	}
	var scored []Hit
	for _, it := range items {
		score := retrieval.Cosine(qvec[0], it.vec)
		if score >= MinCosine {
			scored = append(scored, Hit{Kind: kind, ID: it.id, Score: score, Text: it.text})
		}
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if len(scored) > topN && topN > 0 {
		scored = scored[:topN]
	}
	return scored, nil
}

func (s *Store) vectorDocs(kind string) (map[string]string, error) {
	rows, err := s.db.Query(`SELECT id, doc FROM semantic_vectors WHERE kind=?`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, doc string
		if rows.Scan(&id, &doc) == nil {
			out[id] = doc
		}
	}
	return out, nil
}

func (s *Store) upsert(kind, id, vec, doc, at string) error {
	_, err := s.db.Exec(`
INSERT INTO semantic_vectors(kind,id,vec,doc,updated_at) VALUES(?,?,?,?,?)
ON CONFLICT(kind,id) DO UPDATE SET vec=excluded.vec, doc=excluded.doc, updated_at=excluded.updated_at`,
		kind, id, vec, doc, at)
	return err
}

// Remove 删除单个 (kind,id) 向量（文件删除事件实时清理用，阶段 5 T5-2）。
func (s *Store) Remove(kind, id string) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM semantic_vectors WHERE kind=? AND id=?`, kind, id)
	return err
}

// Stale 删除已不存在的条目向量（文档集合变化后清理，返回删除数）。
func (s *Store) Stale(kind string, keep map[string]bool) (int, error) {
	if s.db == nil {
		return 0, nil
	}
	rows, err := s.db.Query(`SELECT id FROM semantic_vectors WHERE kind=?`, kind)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var del []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && !keep[id] {
			del = append(del, id)
		}
	}
	for _, id := range del {
		if _, err := s.db.Exec(`DELETE FROM semantic_vectors WHERE kind=? AND id=?`, kind, id); err != nil {
			return 0, err
		}
	}
	return len(del), nil
}
