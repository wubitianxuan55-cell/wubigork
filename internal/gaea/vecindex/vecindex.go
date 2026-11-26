// Package vecindex 本地向量索引底座：造价库（清单 / 定额 / 工料机）的语义索引都建在它上面。
//
// 与旧的 semantic.Store（semantic_vectors，JSON 向量）的区别，都是为「库越滚越大」准备的：
//   - 向量按二进制存（float32 小端，bge-m3 1024 维 = 4KB/条），带正文哈希与模型名 / 维度——
//     正文变了重嵌、换了模型整类目重嵌，都靠这两列判定；
//   - 查询只向量化 query，相似度在按类目缓存的内存矩阵上算（int8 量化，每条 1KB），
//     延迟与库的大小基本无关；
//   - 建索引在后台索引器里做（写入打脏标记 → 合并 → 增量补齐），搜索路径不再嵌文档。
//
// 规模口径（2026-11-26 用户拍板：单类目 50 万条以内）：50 万条 × 1024 维，int8 缓存约
// 512MB，库里的 float32 原件约 2GB；再大要上量化存储或 ANN。
//
// 本包是叶子包：只依赖标准库，cost / workcost / app / builtin 都可以直接引用。
package vecindex

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	// EmbedBatch 每次请求向量化的文档数（bge-m3 实测 512 条约 5 秒 ≈ 100 条/秒）。
	EmbedBatch = 128
	// MinCosine 向量路的余弦下限：bge-m3 的无关文本对也常在 0.3 上下，低于此值不算命中。
	MinCosine = 0.35
	// hashLen 正文哈希的字节数（sha256 截前 16 字节）。
	hashLen = 16
)

// Doc 参与索引的一条文档：ID 是库里的键（定额 / 资源编码、清单行 ID），Text 是用于向量化的正文。
type Doc struct {
	ID   string
	Text string
}

// Embedder 向量化能力（retrieval.Embedder 满足它）。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Hit 向量路的命中。
type Hit struct {
	ID    string
	Score float64
}

// Source 分块产出某个类目的全部文档。对账靠「全量」判断哪些向量过期，所以必须把库里
// 所有条目都吐出来（空正文的由索引器跳过）；yield 返回 error 时中止。
type Source func(yield func(docs []Doc) error) error

// Stats 一次对账的结果。
type Stats struct {
	Kind     string
	Total    int // 文档总数（正文非空）
	Embedded int // 本次新向量化的（新增 + 正文变了 + 模型变了）
	Removed  int // 本次清掉的过期向量
	Count    int // 对账后的向量条数
	Dur      time.Duration
}

// Meta 一个类目的索引元信息。
type Meta struct {
	Kind      string
	Version   int64
	Model     string
	Dim       int
	Count     int
	UpdatedAt string
}

// ErrDimMismatch 查询向量与索引维度不一致（换了模型而索引还没重嵌完）。
var ErrDimMismatch = errors.New("查询向量与索引维度不一致")

// Store 向量索引存储。同一个 *sql.DB 只有一个实例（内存缓存随实例走）：应用的界面、
// 工具面、后台索引器拿到的是同一份缓存。
type Store struct {
	db       *sql.DB
	ready    sync.Once
	readyErr error
	mu       sync.Mutex
	cache    map[string]*matrix
	apply    map[string]*sync.Mutex
}

var stores sync.Map // *sql.DB → *Store

// Open 取某个数据库的向量索引；db 为 nil 时返回 nil（调用方按「不可用」处理）。
func Open(db *sql.DB) *Store {
	if db == nil {
		return nil
	}
	if v, ok := stores.Load(db); ok {
		return v.(*Store)
	}
	v, _ := stores.LoadOrStore(db, &Store{
		db:    db,
		cache: map[string]*matrix{},
		apply: map[string]*sync.Mutex{},
	})
	return v.(*Store)
}

// Forget 丢掉某个数据库的索引实例与内存缓存（库关闭后调用）。
func Forget(db *sql.DB) {
	if v, ok := stores.LoadAndDelete(db); ok {
		st := v.(*Store)
		st.mu.Lock()
		st.cache = map[string]*matrix{}
		st.mu.Unlock()
	}
}

// Available 报告索引是否可用（表已建）。
func (s *Store) Available() bool {
	if s == nil || s.db == nil {
		return false
	}
	s.ready.Do(func() {
		_, s.readyErr = s.db.Exec(`SELECT COUNT(*) FROM semantic_vec2_meta WHERE 0`)
	})
	return s.readyErr == nil
}

func (s *Store) lockApply(kind string) func() {
	s.mu.Lock()
	mu, ok := s.apply[kind]
	if !ok {
		mu = &sync.Mutex{}
		s.apply[kind] = mu
	}
	s.mu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// Apply 对账一个类目：缺的、正文变了的、模型名对不上的 → 重新向量化；库里已经没有的 →
// 删掉。幂等；同一类目同时只跑一个。对账中途失败时已写入的那部分留在库里，下次续上。
func (s *Store) Apply(ctx context.Context, kind, model string, src Source, emb Embedder, batch int) (Stats, error) {
	start := time.Now()
	if s == nil || s.db == nil || !s.Available() {
		return Stats{}, errors.New("向量索引不可用（表未建或数据库已关闭）")
	}
	if emb == nil {
		return Stats{}, errors.New("embedding 模型不可用")
	}
	if batch <= 0 {
		batch = EmbedBatch
	}
	defer s.lockApply(kind)()

	have, err := s.rows(kind)
	if err != nil {
		return Stats{}, err
	}
	st := Stats{Kind: kind}
	seen := make(map[string]struct{}, len(have))
	var pend []Doc
	embedded := 0
	flush := func() error {
		if len(pend) == 0 {
			return nil
		}
		texts := make([]string, len(pend))
		for i, d := range pend {
			texts[i] = d.Text
		}
		vecs, err := emb.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("向量化失败（本次已写入 %d 条）: %w", embedded, err)
		}
		n, err := s.write(kind, model, pend, vecs)
		if err != nil {
			return err
		}
		embedded += n
		pend = pend[:0]
		// 每批写完就 bump：首次建大索引要十几分钟，中途这段已经建好的部分就能被检索到、
		// 覆盖数也如实；对账失败留下的进度同样可见。
		return s.bump(kind, model)
	}
	err = src(func(docs []Doc) error {
		for _, d := range docs {
			if d.ID == "" || strings.TrimSpace(d.Text) == "" {
				continue
			}
			st.Total++
			seen[d.ID] = struct{}{}
			h := hashOf(d.Text)
			if r, ok := have[d.ID]; ok && r.hash == h && r.model == model {
				continue
			}
			pend = append(pend, Doc{ID: d.ID, Text: d.Text})
			if len(pend) >= batch {
				if err := flush(); err != nil { // 冲掉这一批，接着遍历后面的文档
					return err
				}
			}
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		return st, err
	}
	st.Embedded = embedded
	var stale []string
	for id := range have {
		if _, ok := seen[id]; !ok {
			stale = append(stale, id)
		}
	}
	if len(stale) > 0 {
		if err := s.remove(kind, stale); err != nil {
			return st, err
		}
		st.Removed = len(stale)
	}
	if st.Removed > 0 { // 写入的各批已各自 bump 过，这里只需在清掉过期向量后再来一次
		if err := s.bump(kind, model); err != nil {
			return st, err
		}
	}
	// 条数以库为准（没改动时元信息里也是现数）；有改动就顺手把内存缓存换新，下一笔查询不用等重载
	if meta, err := s.Meta(kind); err == nil {
		st.Count = meta.Count
		if st.Embedded > 0 || st.Removed > 0 {
			_, _ = s.matrixFor(kind, meta)
		}
	}
	st.Dur = time.Since(start)
	return st, nil
}

type rowRef struct {
	hash  [hashLen]byte
	model string
}

func (s *Store) rows(kind string) (map[string]rowRef, error) {
	r, err := s.db.Query(`SELECT id, hash, model FROM semantic_vec2 WHERE kind=?`, kind)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	out := map[string]rowRef{}
	for r.Next() {
		var id, model string
		var h []byte
		if err := r.Scan(&id, &h, &model); err != nil {
			return nil, err
		}
		var ref rowRef
		if len(h) == hashLen {
			copy(ref.hash[:], h)
		}
		ref.model = model
		out[id] = ref
	}
	return out, r.Err()
}

func (s *Store) write(kind, model string, docs []Doc, vecs [][]float32) (int, error) {
	if len(vecs) != len(docs) {
		return 0, fmt.Errorf("向量化返回 %d 条、预期 %d 条", len(vecs), len(docs))
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`INSERT INTO semantic_vec2(kind,id,model,dim,vec,hash,doc,updated_at) VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(kind,id) DO UPDATE SET model=excluded.model, dim=excluded.dim, vec=excluded.vec,
  hash=excluded.hash, doc=excluded.doc, updated_at=excluded.updated_at`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = stmt.Close() }()
	now := time.Now().UTC().Format(time.RFC3339)
	n := 0
	for i, d := range docs {
		v := vecs[i]
		if len(v) == 0 {
			continue
		}
		h := hashOf(d.Text)
		if _, err := stmt.Exec(kind, d.ID, model, len(v), encodeVec(v), h[:], d.Text, now); err != nil {
			return n, err
		}
		n++
	}
	return n, tx.Commit()
}

func (s *Store) remove(kind string, ids []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM semantic_vec2 WHERE kind=? AND id=?`, kind, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// bump 版本号 +1 并刷新元信息（count / dim 以库为准）。
func (s *Store) bump(kind, model string) error {
	var count, dim int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM semantic_vec2 WHERE kind=?`, kind).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		if err := s.db.QueryRow(`SELECT dim FROM semantic_vec2 WHERE kind=? LIMIT 1`, kind).Scan(&dim); err != nil {
			return err
		}
	}
	_, err := s.db.Exec(`INSERT INTO semantic_vec2_meta(kind,version,model,dim,count,updated_at) VALUES(?,1,?,?,?,?)
ON CONFLICT(kind) DO UPDATE SET version=version+1, model=excluded.model, dim=excluded.dim,
  count=excluded.count, updated_at=excluded.updated_at`,
		kind, model, dim, count, time.Now().UTC().Format(time.RFC3339))
	return err
}

// Meta 取类目的索引元信息（没建过索引时 Count=0）。
func (s *Store) Meta(kind string) (Meta, error) {
	if s == nil || s.db == nil || !s.Available() {
		return Meta{Kind: kind}, nil
	}
	var m Meta
	err := s.db.QueryRow(`SELECT version, model, dim, count, updated_at FROM semantic_vec2_meta WHERE kind=?`, kind).
		Scan(&m.Version, &m.Model, &m.Dim, &m.Count, &m.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Meta{Kind: kind}, nil
	}
	m.Kind = kind
	return m, err
}

// Search 向量化 query 后在类目里取相似度最高的 topN 条。
func (s *Store) Search(ctx context.Context, kind string, emb Embedder, query string, topN int, keep func(id string) bool) ([]Hit, error) {
	qvec, err := Query(ctx, emb, query)
	if err != nil || qvec == nil {
		return nil, err
	}
	return s.SearchVec(kind, qvec, topN, keep)
}

// SearchVec 在类目里按 qvec 取相似度最高的 topN 条。keep 非空时只留满足条件的 ID
// （按专业 / 类型预过滤）。只读内存缓存，不做向量化；缓存落后于库时重载一次。
func (s *Store) SearchVec(kind string, qvec []float32, topN int, keep func(id string) bool) ([]Hit, error) {
	if s == nil || s.db == nil || !s.Available() || topN <= 0 || len(qvec) == 0 {
		return nil, nil
	}
	meta, err := s.Meta(kind)
	if err != nil || meta.Count == 0 {
		return nil, err
	}
	m, err := s.matrixFor(kind, meta)
	if err != nil {
		return nil, err
	}
	if m.dim > 0 && len(qvec) != m.dim {
		return nil, fmt.Errorf("%w：查询 %d 维、索引 %d 维", ErrDimMismatch, len(qvec), m.dim)
	}
	return m.search(qvec, topN, keep, MinCosine), nil
}

// Query 向量化一句查询（多个类目共用一次向量化）。
func Query(ctx context.Context, emb Embedder, query string) ([]float32, error) {
	if emb == nil || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	vecs, err := emb.Embed(ctx, []string{query})
	if err != nil || len(vecs) == 0 {
		return nil, err
	}
	return vecs[0], nil
}

func hashOf(text string) [hashLen]byte {
	sum := sha256.Sum256([]byte(text))
	var h [hashLen]byte
	copy(h[:], sum[:hashLen])
	return h
}

func encodeVec(v []float32) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

func decodeVec(b []byte, v []float32) {
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
}
