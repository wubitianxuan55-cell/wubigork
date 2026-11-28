package vecindex

// indexer.go — 后台索引器：写入打脏标记 → 合并窗口 → 增量对账。搜索路径不建索引。

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Indexer 一个库（或一组类目）的后台索引器。
//
// 用法：
//
//	ix := vecindex.NewIndexer(st, "bge-m3", embed, sources, "wc_quota", "wc_res", "wc_bill")
//	vecindex.SetDefault(ix)      // 之后库的写入点调 vecindex.MarkDirty("wc_quota") 即可
//	go ix.Run(ctx)               // ctx 随应用生命周期
//
// 一轮对账只做一件事：拿类目的全量文档集与库里的向量比，补缺、重嵌、清过期。
// 模型没起来时不丢待办，隔 RetryAfter 再试。
type Indexer struct {
	st      *Store
	model   string
	embed   func() Embedder // 每轮取一次；nil = 模型当前不可用
	sources map[string]Source
	kinds   []string // 固定顺序（对账按它串行）

	// Debounce 写入后的合并窗口：批量导入时成千上万次 MarkDirty 只触发一轮。
	Debounce time.Duration
	// ReconcileEvery 兜底全量对账的间隔：漏打脏标记的写入靠它追上；<=0 关闭。
	ReconcileEvery time.Duration
	// RetryAfter 模型不可用 / 对账出错后的重试间隔。
	RetryAfter time.Duration

	mu      sync.Mutex
	pending map[string]bool
	wake    chan struct{}
	last    map[string]Stats
	lastErr map[string]error // 最近一轮对账的错误；成功后清除。索引状态面读它，不用再去探测模型
	// fails 各类目连续对账失败的次数（成功清零）：日志去重与恢复提示靠它。
	fails map[string]int
	// coolUntil 退避截止时刻。对账出错后、到点之前，新的脏标记只记待办、不唤醒：
	// 否则「失败 → 放回待办并唤醒 → 合并窗口后再试」会把 RetryAfter 架空
	// （2026-11-27 实测：本地向量模型设备丢失后每 3 秒重试一次，20 小时 6.7 万次）。
	coolUntil time.Time
	// log 日志器；nil = slog.Default()。
	log *slog.Logger
}

var (
	errEmbedUnavailable = errors.New("embedding 模型不可用")
	errNoSource         = errors.New("索引器没有这个类目的文档源")
)

// NewIndexer 建索引器。sources 缺某个 kind 时该 kind 不对账。
func NewIndexer(st *Store, model string, embed func() Embedder, sources map[string]Source, kinds ...string) *Indexer {
	if model == "" {
		model = "bge-m3"
	}
	return &Indexer{
		st: st, model: model, embed: embed, sources: sources, kinds: append([]string(nil), kinds...),
		Debounce:       3 * time.Second,
		ReconcileEvery: 10 * time.Minute,
		RetryAfter:     time.Minute,
		pending:        map[string]bool{},
		wake:           make(chan struct{}, 1),
		last:           map[string]Stats{},
		lastErr:        map[string]error{},
		fails:          map[string]int{},
	}
}

var defaultIndexer atomic.Pointer[Indexer]

// SetDefault 登记进程级的默认索引器；库的写入点经 MarkDirty 找它。
func SetDefault(ix *Indexer) { defaultIndexer.Store(ix) }

// MarkDirty 给一个类目打脏标记（kind 为空 = 全部）。没有默认索引器时什么都不做。
// 包级入口是给 cost / workcost 这类不能依赖调用方的库包用的——它们只认识 vecindex。
func MarkDirty(kind string) {
	if ix := defaultIndexer.Load(); ix != nil {
		ix.Mark(kind)
	}
}

// Mark 打脏标记：合并进待办，唤醒后台循环（多次调用只触发一轮）。
func (ix *Indexer) Mark(kinds ...string) {
	if ix == nil {
		return
	}
	ix.mu.Lock()
	if len(kinds) == 0 {
		for _, k := range ix.kinds {
			ix.pending[k] = true
		}
	} else {
		for _, k := range kinds {
			if k == "" {
				for _, kk := range ix.kinds {
					ix.pending[kk] = true
				}
				continue
			}
			ix.pending[k] = true
		}
	}
	cooling := time.Now().Before(ix.coolUntil)
	ix.mu.Unlock()
	if cooling {
		return // 退避期内：待办已记下，到点由重试定时器处理，不立刻再试
	}
	select {
	case ix.wake <- struct{}{}:
	default: // 已经有一次唤醒在排队
	}
}

// failureLogEvery 连续失败期间隔多少次再提醒一行（1 分钟重试一次 ≈ 每半小时一行）。
const failureLogEvery = 30

// logger 本索引器用的日志器（没注入就用进程默认的）。
func (ix *Indexer) logger() *slog.Logger {
	if ix != nil && ix.log != nil {
		return ix.log
	}
	return slog.Default()
}

// requeue 把对账失败的类目放回待办并进入退避：到点之前新的脏标记不再触发即时重试。
func (ix *Indexer) requeue(kind string) {
	ix.mu.Lock()
	ix.pending[kind] = true
	ix.coolUntil = time.Now().Add(ix.RetryAfter)
	ix.mu.Unlock()
}

// logFailure 对账失败的日志：第一次 WARN，之后降到 DEBUG，每隔 failureLogEvery 次
// 提醒一行——模型一坏就是几小时，每分钟一行也是刷屏。
func (ix *Indexer) logFailure(kind string, fails int, err error) {
	switch {
	case fails == 1:
		ix.logger().Warn("vecindex: 对账失败，稍后重试", "kind", kind, "error", err)
	case fails%failureLogEvery == 0:
		ix.logger().Warn("vecindex: 对账持续失败，退避重试中", "kind", kind, "fails", fails, "error", err)
	default:
		ix.logger().Debug("vecindex: 对账仍失败，退避重试中", "kind", kind, "fails", fails, "error", err)
	}
}

// Last 最近一次成功对账的结果。
func (ix *Indexer) Last(kind string) (Stats, bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	st, ok := ix.last[kind]
	return st, ok
}

// LastError 最近一轮对账的错误（模型不可用 / 向量化失败…）。返回 ok=false 表示没有
// 记录在案的错误：要么一切正常，要么这个类目还没对账过（配 Last 一起看）。
func (ix *Indexer) LastError(kind string) (error, bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	err, ok := ix.lastErr[kind]
	return err, ok
}

// Model 索引使用的向量模型名。
func (ix *Indexer) Model() string {
	if ix == nil {
		return ""
	}
	return ix.model
}

// Kinds 索引器管的类目（固定顺序）。
func (ix *Indexer) Kinds() []string {
	if ix == nil {
		return nil
	}
	return append([]string(nil), ix.kinds...)
}

// record 记下一轮对账的结果：失败累加连续次数、留错误；成功清错误与连续次数，
// 并返回「之前是否处在连续失败中」（恢复提示用）。
func (ix *Indexer) record(kind string, st Stats, err error) (fails int, recovered bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if err != nil {
		ix.fails[kind]++
		ix.lastErr[kind] = err
		return ix.fails[kind], false
	}
	n := ix.fails[kind]
	delete(ix.fails, kind)
	delete(ix.lastErr, kind)
	ix.last[kind] = st
	return 0, n > 0
}

// ApplyNow 立刻对一个类目对账（不等合并窗口；显式补齐与测试用）。模型不可用时报错。
func (ix *Indexer) ApplyNow(ctx context.Context, kind string) (Stats, error) {
	src := ix.sources[kind]
	if src == nil {
		return Stats{}, fmt.Errorf("%w：%s", errNoSource, kind)
	}
	emb := ix.embed()
	if emb == nil {
		err := errEmbedUnavailable
		ix.record(kind, Stats{}, err)
		return Stats{}, err
	}
	st, err := ix.st.Apply(ctx, kind, ix.model, src, emb, EmbedBatch)
	ix.record(kind, st, err)
	return st, err
}

// Run 阻塞运行到 ctx 结束。启动先对一轮（立即做，不等合并窗口——开机那一刻没有需要
// 合并的写入，索引越早可用越好）；之后被脏标记唤醒（过合并窗口再动手），另按
// ReconcileEvery 兜底全量对账。对账出错后进入退避：RetryAfter 到点再试，期间新的
// 脏标记只记待办、不唤醒。
func (ix *Indexer) Run(ctx context.Context) {
	ix.Mark()
	if ctx.Err() != nil {
		return
	}
	var retryC <-chan time.Time
	if !ix.runPending(ctx) {
		retryC = time.After(ix.RetryAfter)
	}
	debounce := time.NewTimer(ix.Debounce)
	defer debounce.Stop()
	retry := time.NewTimer(time.Hour)
	if !retry.Stop() {
		<-retry.C
	}
	var recC <-chan time.Time
	if ix.ReconcileEvery > 0 {
		rec := time.NewTicker(ix.ReconcileEvery)
		defer rec.Stop()
		recC = rec.C
	}
	arm := func() {
		if !debounce.Stop() {
			select { // 已经触发过：把滞留的那一次排掉再重计时
			case <-debounce.C:
			default:
			}
		}
		debounce.Reset(ix.Debounce)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ix.wake:
			arm()
		case <-debounce.C:
			if !ix.runPending(ctx) {
				retry.Reset(ix.RetryAfter)
				retryC = retry.C
			}
		case <-retryC:
			retryC = nil
			if !ix.runPending(ctx) {
				retry.Reset(ix.RetryAfter)
				retryC = retry.C
			}
		case <-recC:
			ix.Mark()
		}
	}
}

// runPending 把待办里的类目依次对账。返回 false = 还有类目没处理完（模型不可用或出错）。
func (ix *Indexer) runPending(ctx context.Context) bool {
	ix.mu.Lock()
	var todo []string
	for _, k := range ix.kinds {
		if ix.pending[k] {
			todo = append(todo, k)
		}
	}
	if len(todo) == 0 {
		ix.mu.Unlock()
		return true
	}
	emb := ix.embed()
	if emb == nil {
		ix.mu.Unlock()
		ix.logger().Debug("vecindex: embedding 模型不可用，索引待办保留", "kinds", todo)
		for _, k := range todo {
			ix.record(k, Stats{}, errEmbedUnavailable)
		}
		return false
	}
	// 取走待办：对账期间新来的脏标记留到下一轮（那一轮会看到这期间的全部改动）
	for _, k := range todo {
		delete(ix.pending, k)
	}
	ix.mu.Unlock()

	ok := true
	for i, k := range todo {
		if ctx.Err() != nil {
			ix.Mark(todo[i:]...) // 没处理的留到下次
			return false
		}
		src := ix.sources[k]
		if src == nil {
			continue
		}
		st, err := ix.st.Apply(ctx, k, ix.model, src, emb, EmbedBatch)
		if err != nil {
			// 失败的类目留在待办里，但不唤醒：到点由重试定时器再来（requeue 进入退避）。
			fails, _ := ix.record(k, st, err)
			ix.requeue(k)
			ix.logFailure(k, fails, err)
			ok = false
			continue
		}
		if st.Embedded > 0 || st.Removed > 0 {
			ix.logger().Info("vecindex: 索引已更新", "kind", k, "total", st.Total, "embedded", st.Embedded,
				"removed", st.Removed, "count", st.Count, "dur", st.Dur.Round(time.Millisecond))
		}
		if _, recovered := ix.record(k, st, nil); recovered {
			ix.logger().Info("vecindex: 索引恢复", "kind", k, "total", st.Total, "embedded", st.Embedded,
				"removed", st.Removed, "count", st.Count, "dur", st.Dur.Round(time.Millisecond))
		}
	}
	return ok
}
