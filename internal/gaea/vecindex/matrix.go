package vecindex

// matrix.go — 按类目缓存的内存向量矩阵（int8 量化）与相似度扫描。

import (
	"container/heap"
	"math"
	"runtime"
	"sort"
	"sync"
)

// parallelRows 行数到了这个量级才拆 goroutine 并行扫（小库拆分开销不划算）。
var parallelRows = 16384

// matrix 一个类目的向量矩阵。每行是归一化后按行量化的 int8：float32 向量 ≈ int8 * coef。
// 1024 维一条占 1KB——50 万条约 512MB，是 float32 原件的四分之一；量化误差在末位，
// 排序靠后面的融合与精排兜住（见 matrix_test 的对照用例）。
type matrix struct {
	model   string
	dim     int
	version int64
	ids     []string
	q       []int8    // n*dim
	coef    []float32 // 每行的还原系数
}

func newMatrix(model string, dim, n int, version int64) *matrix {
	if n < 0 {
		n = 0
	}
	return &matrix{model: model, dim: dim, version: version,
		ids: make([]string, 0, n), q: make([]int8, 0, n*dim), coef: make([]float32, 0, n)}
}

// add 追加一行：归一化后量化。零向量丢弃。
func (m *matrix) add(id string, v []float32) {
	d := m.dim
	if len(v) != d {
		return
	}
	var norm, max float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for _, x := range v {
		if a := math.Abs(float64(x)) / norm; a > max {
			max = a
		}
	}
	if max == 0 {
		return
	}
	scale := 127 / max
	start := len(m.q)
	m.q = m.q[:start+d]
	q := m.q[start : start+d]
	for i, x := range v {
		q[i] = int8(math.Round(float64(x) / norm * scale))
	}
	m.ids = append(m.ids, id)
	m.coef = append(m.coef, float32(max/127))
}

// quantize 把查询向量按同样的办法量化：int8 向量 + 还原系数。
func (m *matrix) quantize(v []float32) ([]int8, float64, bool) {
	d := m.dim
	if len(v) != d {
		return nil, 0, false
	}
	var norm, max float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return nil, 0, false
	}
	for _, x := range v {
		if a := math.Abs(float64(x)) / norm; a > max {
			max = a
		}
	}
	if max == 0 {
		return nil, 0, false
	}
	scale := 127 / max
	q := make([]int8, d)
	for i, x := range v {
		q[i] = int8(math.Round(float64(x) / norm * scale))
	}
	return q, max / 127, true
}

// search 扫全矩阵取相似度最高的 topN（余弦，低于 min 的丢掉；keep 非空时只留满足
// 条件的 ID）。行数多时按 CPU 数切块并行，各块取局部 topN 再合并。
func (m *matrix) search(qvec []float32, topN int, keep func(id string) bool, min float64) []Hit {
	qt, cq, ok := m.quantize(qvec)
	if !ok {
		return nil
	}
	n := len(m.ids)
	if n == 0 {
		return nil
	}
	if n < parallelRows {
		return m.scan(0, n, qt, cq, topN, keep, min)
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > 8 {
		workers = 8
	}
	if workers < 2 {
		return m.scan(0, n, qt, cq, topN, keep, min)
	}
	chunk := (n + workers - 1) / workers
	parts := make([][]Hit, workers)
	var wg sync.WaitGroup
	for p := 0; p < workers; p++ {
		a := p * chunk
		if a >= n {
			break
		}
		b := a + chunk
		if b > n {
			b = n
		}
		wg.Add(1)
		go func(p, a, b int) {
			defer wg.Done()
			parts[p] = m.scan(a, b, qt, cq, topN, keep, min)
		}(p, a, b)
	}
	wg.Wait()
	var all []Hit
	for _, part := range parts {
		all = append(all, part...)
	}
	sortHits(all)
	if len(all) > topN {
		all = all[:topN]
	}
	return all
}

// scan 在 [a,b) 行上算点积，维护一个大小 topN 的小顶堆。
func (m *matrix) scan(a, b int, qt []int8, cq float64, topN int, keep func(id string) bool, min float64) []Hit {
	d := m.dim
	qt = qt[:d]
	h := &hitHeap{}
	for i := a; i < b; i++ {
		row := m.q[i*d : i*d+d]
		var acc int32
		for k, r := range row {
			acc += int32(r) * int32(qt[k])
		}
		score := float64(acc) * cq * float64(m.coef[i])
		if score < min || (h.Len() == topN && score <= (*h)[0].Score) {
			continue
		}
		id := m.ids[i]
		if keep != nil && !keep(id) {
			continue
		}
		if h.Len() < topN {
			heap.Push(h, Hit{ID: id, Score: score})
		} else {
			(*h)[0] = Hit{ID: id, Score: score}
			heap.Fix(h, 0)
		}
	}
	out := []Hit(*h)
	sortHits(out)
	return out
}

func sortHits(h []Hit) {
	sort.Slice(h, func(i, j int) bool {
		if h[i].Score != h[j].Score {
			return h[i].Score > h[j].Score
		}
		return h[i].ID < h[j].ID // 同分按 ID 字典序，结果确定
	})
}

// hitHeap 按得分的小顶堆（堆顶是已入选里最小的，用来淘汰）。
type hitHeap []Hit

func (h hitHeap) Len() int           { return len(h) }
func (h hitHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h hitHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *hitHeap) Push(x any)        { *h = append(*h, x.(Hit)) }
func (h *hitHeap) Pop() any          { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }

// load 从库里读出一个类目的全部向量建矩阵（只取与元信息同模型同维度的行——换了模型
// 重嵌到一半时，旧模型的行不参与检索）。
func (s *Store) load(kind string, meta Meta) (*matrix, error) {
	if meta.Count == 0 || meta.Dim == 0 {
		return newMatrix(meta.Model, meta.Dim, 0, meta.Version), nil
	}
	rows, err := s.db.Query(`SELECT id, vec FROM semantic_vec2 WHERE kind=? AND model=? AND dim=?`, kind, meta.Model, meta.Dim)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	m := newMatrix(meta.Model, meta.Dim, meta.Count, meta.Version)
	buf := make([]float32, meta.Dim)
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		if len(raw) != meta.Dim*4 {
			continue
		}
		decodeVec(raw, buf)
		m.add(id, buf)
	}
	return m, rows.Err()
}

// matrixFor 取类目的内存矩阵：版本号与库里一致就用缓存的，否则重载。版本号在索引器
// 每次写入后 +1，所以它一致就说明缓存与库逐条一致。
func (s *Store) matrixFor(kind string, meta Meta) (*matrix, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.cache[kind]; ok && m.version == meta.Version {
		return m, nil
	}
	m, err := s.load(kind, meta)
	if err != nil {
		return nil, err
	}
	s.cache[kind] = m
	return m, nil
}
