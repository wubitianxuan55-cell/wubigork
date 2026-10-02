package app

// gaea 通用任务调度器绑定与消费者（阶段 5 T5-1）。
//
// officeState.tasks 是全局调度器（Hephaestus.db tasks 表，SchemaV8）：
//   - 消费者注册：价格抓取（单源/全部+定时 cron）、工作区语义索引重建；
//   - 前端任务中心经 GaeaTaskList/Cancel/Retry 驱动（gaea-task 事件实时推送）；
//   - 重启续跑：Startup 把上次 running 的任务恢复 queued 重新排队。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/fileindex"
	"github.com/gaea/gaea/internal/gaea/filewatch"
	"github.com/gaea/gaea/internal/gaea/pricefeed"
	"github.com/gaea/gaea/internal/gaea/semantic"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// taskMgr 返回全局任务调度器（nil 安全：测试环境 officeState 可能未装配）。
func (a *App) taskMgr() *tasks.Manager {
	if a == nil || a.officeState == nil {
		return nil
	}
	return a.officeState.tasks
}

// startTaskScheduler 启动通用任务调度器（幂等）：注册消费者 → 重启续跑 → 拉起 worker。
func (a *App) startTaskScheduler() {
	a.officeState.taskOnce.Do(func() {
		gdb := db.GetDatabase(config.MemoryUserDir())
		if gdb == nil {
			slog.Warn("tasks: Hephaestus.db 不可用，任务调度器禁用")
			return
		}
		m := tasks.New(gdb, func(t tasks.Task) {
			a.emitTaskEvent(t)
		}, taskSchedulerOptions(config.Load()))
		m.Register(tasks.KindPriceFetch, a.priceFetchTaskHandler)
		m.Register(tasks.KindPriceFetchAll, a.priceFetchAllTaskHandler)
		m.Register(tasks.KindFileIndex, a.fileIndexTaskHandler)
		m.Register(tasks.KindOutlineReconstruct, a.outlineReconstructTaskHandler)
		a.officeState.tasks = m
		if n, err := m.Start(); err != nil {
			slog.Warn("tasks: 调度器启动失败", "error", err)
		} else if n > 0 {
			slog.Info("tasks: 调度器启动，续跑任务", "count", n)
		}
	})
}

// taskSchedulerOptions 把 [tasks] 配置解析为调度器选项（v4.5.1a 红线补课：
// S1.4 任务按空间分账内核生产启用）。配置加载失败/零值回退默认：
//   - max_concurrent 缺省 1（对齐 herdsman local_concurrency=1）；
//   - per_space 缺省 {work=1, play=1}（空间分账：各空间独立并发跑道，
//     某空间额度占满不阻塞其他空间出队）；显式 `per_space = {}` 关闭分账
//     回退全局 sem（旧行为）；
//   - priority 缺省 {price_fetch=20, price_fetch_all=20, file_index=10}
//     （用户触发的价格抓取优先于后台文件索引维护）。
func taskSchedulerOptions(cfg *config.Config, err error) tasks.Options {
	opts := tasks.Options{}
	if err == nil && cfg != nil {
		opts.MaxConcurrent = cfg.Tasks.MaxConcurrent
		opts.PerSpace = cfg.Tasks.PerSpace
		opts.Priority = cfg.Tasks.Priority
	}
	if opts.MaxConcurrent <= 0 {
		opts.MaxConcurrent = 1
	}
	if opts.PerSpace == nil {
		opts.PerSpace = map[string]int{spaces.SpaceWork: 1, spaces.SpacePlay: 1}
	}
	if opts.Priority == nil {
		opts.Priority = map[string]int{
			string(tasks.KindPriceFetch):    20,
			string(tasks.KindPriceFetchAll): 20,
			string(tasks.KindFileIndex):     10,
		}
	}
	return opts
}

// emitTaskEvent 把任务视图推给前端（gaea-task 事件通道）。
func (a *App) emitTaskEvent(t tasks.Task) {
	b, err := json.Marshal(t)
	if err != nil {
		return
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return
	}
	a.emit("gaea-task", m)
}

// GaeaTaskList 返回最近任务（新→旧，供任务中心）。变参 space（S1.4，对齐
// GaeaUnifiedSearch 先例——Wails 反射 Call 对变参安全，前端零参调用不破）：
// 不传/传空 = 不过滤（跨空间全量，旧行为零变化）；传 "work"/"play" = 按空间
// 过滤（只取首个；S2 接线后由绑定层透传当前空间）。
func (a *App) GaeaTaskList(space ...string) []tasks.Task {
	sc := ""
	if len(space) > 0 {
		sc = space[0]
	}
	return a.taskListInSpace(sc)
}

// taskListInSpace 按空间返回最近任务（新→旧）：space 为空不过滤（旧行为零
// 变化），非空经 tasks.Manager.ListInSpace 落 WHERE space_id=?。未导出是有意
// 之为：绑定面（bindings_office.go 生成物 + TestBindingsCompleteness 方法集
// 对账）保持不动，绑定面透传属 S4（gen_bindings 重新生成 + 前端同步）。
func (a *App) taskListInSpace(space string) []tasks.Task {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return nil
	}
	list, err := m.ListInSpace(50, space)
	if err != nil {
		slog.Warn("tasks: 列表读取失败", "error", err)
		return nil
	}
	out := make([]tasks.Task, 0, len(list))
	for _, t := range list {
		out = append(out, *t)
	}
	return out
}

// GaeaTaskCancel 取消一个任务（running 中断 / queued 直接取消）。
func (a *App) GaeaTaskCancel(id string) error {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return fmt.Errorf("任务调度器未启动")
	}
	return m.Cancel(id)
}

// GaeaTaskKill 强制终止一个任务：协作取消之外执行 handler 经
// Progress.OnForceKill 登记的强杀钩子（击杀任务自有 OS 进程树）；纯函数
// 任务等价 GaeaTaskCancel（诚实降级）。
func (a *App) GaeaTaskKill(id string) error {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return fmt.Errorf("任务调度器未启动")
	}
	return m.Kill(id)
}

// GaeaTaskRetry 重试一个失败/已取消的任务。
func (a *App) GaeaTaskRetry(id string) error {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return fmt.Errorf("任务调度器未启动")
	}
	return m.Retry(id)
}

// TaskOutputView 是任务实时输出的尾部回放视图（C1：任务输出面板）。
type TaskOutputView struct {
	Tail      string `json:"tail"`      // 输出尾部（整尾回放，前端按需轮询）
	Truncated bool   `json:"truncated"` // 超出行数/字节上限被截断
}

// GaeaTaskOutput 返回任务实时输出尾部（环形缓冲回放，不消费游标）。
// 任务无输出/不存在时返回空 tail。
func (a *App) GaeaTaskOutput(id string) (TaskOutputView, error) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return TaskOutputView{}, fmt.Errorf("任务调度器未启动")
	}
	tail, truncated := m.Output(id)
	return TaskOutputView{Tail: tail, Truncated: truncated}, nil
}

// ─── 消费者：价格抓取 ─────────────────────────────────────────

type priceFetchPayload struct {
	SourceID string `json:"sourceId"`
}

// priceFetchTaskHandler 单源抓取任务：解析 → 匹配成本库 → 存 pending 记录。
func (a *App) priceFetchTaskHandler(ctx context.Context, t *tasks.Task, p *tasks.Progress) error {
	var req priceFetchPayload
	if err := json.Unmarshal([]byte(t.Payload), &req); err != nil || req.SourceID == "" {
		return fmt.Errorf("任务参数缺失（sourceId）")
	}
	store := a.hubPriceStore()
	src, ok := store.GetSource(req.SourceID)
	if !ok {
		return fmt.Errorf("价格源不存在: %s", req.SourceID)
	}
	p.Output(fmt.Sprintf("[%s] 抓取 %s（%s）", time.Now().Format("15:04:05"), src.Name, src.URL))
	p.Report(5, "正在抓取 "+src.Name)
	res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
	if err != nil {
		p.Output(fmt.Sprintf("[%s] 抓取失败：%v", time.Now().Format("15:04:05"), err))
		return err
	}
	res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
	p.Output(fmt.Sprintf("[%s] 解析出 %d 条候选价格，异常检测完成", time.Now().Format("15:04:05"), len(res.Candidates)))
	rec := pricefeed.FetchRecord{
		ID:       fmt.Sprintf("fetch-%d", time.Now().UnixNano()), // SaveFetch 按值拷贝，ID 须先预生成
		SourceID: src.ID, SourceName: src.Name, URL: src.URL,
		Period: res.Period, FetchedAt: res.FetchedAt, Status: "pending",
		Candidates: res.Candidates,
	}
	if err := store.SaveFetch(rec); err != nil {
		return err
	}
	_ = store.TouchSource(src.ID, res.FetchedAt)
	out, _ := json.Marshal(map[string]any{"fetchId": rec.ID, "count": len(res.Candidates)})
	p.Result(string(out))
	p.Report(100, fmt.Sprintf("抓取完成：%d 条价格待确认", len(res.Candidates)))
	return nil
}

type priceFetchAllPayload struct {
	Cron bool `json:"cron"` // true=定时（只抓到期源）；false=手动（全部启用源）
}

// priceFetchAllTaskHandler 全部/定时抓取任务：逐源抓取并报告进度，单源失败
// 不阻断其他源；全部失败返回错误（任务标记 failed），部分失败在结果里说明。
func (a *App) priceFetchAllTaskHandler(ctx context.Context, t *tasks.Task, p *tasks.Progress) error {
	var req priceFetchAllPayload
	_ = json.Unmarshal([]byte(t.Payload), &req)

	store := a.hubPriceStore()
	sources := store.ListSources()
	now := time.Now()
	var todo []pricefeed.Source
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		if req.Cron {
			if src.FrequencyHours <= 0 {
				continue
			}
			if src.LastFetchAt != "" {
				if ft, err := time.Parse(time.RFC3339, src.LastFetchAt); err == nil && now.Sub(ft) < time.Duration(src.FrequencyHours)*time.Hour {
					continue
				}
			}
		}
		todo = append(todo, src)
	}
	if len(todo) == 0 {
		if req.Cron {
			out, _ := json.Marshal(map[string]any{"fetched": 0, "failed": 0})
			p.Result(string(out))
			p.Report(100, "暂无到期的价格源")
			return nil
		}
		return fmt.Errorf("没有启用的价格源，请先添加并启用")
	}

	var errs []string
	fetched := 0
	p.Output(fmt.Sprintf("[%s] 待抓取 %d 个价格源", time.Now().Format("15:04:05"), len(todo)))
	for i, src := range todo {
		if ctx.Err() != nil {
			p.Output(fmt.Sprintf("[%s] 已中断（用户取消）", time.Now().Format("15:04:05")))
			return ctx.Err()
		}
		p.Output(fmt.Sprintf("[%s] (%d/%d) 抓取 %s", time.Now().Format("15:04:05"), i+1, len(todo), src.Name))
		p.Report(i*100/len(todo), fmt.Sprintf("正在抓取 %s（%d/%d）", src.Name, i+1, len(todo)))
		res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
		if err != nil {
			errs = append(errs, src.Name+": "+err.Error())
			p.Output(fmt.Sprintf("[%s]   ✗ %s：%v", time.Now().Format("15:04:05"), src.Name, err))
			continue
		}
		res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
		p.Output(fmt.Sprintf("[%s]   ✓ %s：%d 条候选", time.Now().Format("15:04:05"), src.Name, len(res.Candidates)))
		rec := pricefeed.FetchRecord{
			ID:       fmt.Sprintf("fetch-%d", time.Now().UnixNano()), // SaveFetch 按值拷贝，ID 须先预生成
			SourceID: src.ID, SourceName: src.Name, URL: src.URL,
			Period: res.Period, FetchedAt: res.FetchedAt, Status: "pending",
			Candidates: res.Candidates,
		}
		if err := store.SaveFetch(rec); err != nil {
			errs = append(errs, src.Name+": "+err.Error())
			continue
		}
		_ = store.TouchSource(src.ID, res.FetchedAt)
		fetched++
	}
	out, _ := json.Marshal(map[string]any{"fetched": fetched, "failed": len(errs), "errors": errs})
	p.Result(string(out))
	if fetched == 0 && len(errs) > 0 {
		return fmt.Errorf("全部 %d 个价格源抓取失败：%s", len(todo), strings.Join(errs, "；"))
	}
	if len(errs) > 0 {
		p.Report(100, fmt.Sprintf("抓取完成：%d 个成功，%d 个失败", fetched, len(errs)))
		return nil
	}
	p.Report(100, fmt.Sprintf("抓取完成：%d 个价格源", fetched))
	return nil
}

// ─── 消费者：工作区语义索引 ───────────────────────────────────

// fileIndexTaskHandler 扫描工作区并增量建立语义索引（分批 Ensure 报告进度，
// 末批 Stale 清理已删除文件；ctx 中断可取消）。
func (a *App) fileIndexTaskHandler(ctx context.Context, t *tasks.Task, p *tasks.Progress) error {
	docs, skipped, err := fileindex.Scan(gaeaCwd())
	if err != nil {
		return fmt.Errorf("扫描工作区失败: %w", err)
	}
	e := a.localSearchEmbedder()
	if e == nil {
		return fmt.Errorf("本地 embedding 未配置（Herdsman bge-m3）")
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return fmt.Errorf("向量索引不可用")
	}
	total := len(docs)
	if total == 0 {
		_, _ = st.Stale("file", map[string]bool{})
		p.Result(`{"total":0,"skipped":` + fmt.Sprint(skipped) + `}`)
		p.Report(100, "工作区无可索引文件")
		return nil
	}
	const batch = 64
	keep := make(map[string]bool, total)
	for _, d := range docs {
		keep[d.Path] = true
	}
	p.Output(fmt.Sprintf("[%s] 扫描到 %d 个文件（跳过 %d），开始建索引", time.Now().Format("15:04:05"), total, skipped))
	for start := 0; start < total; start += batch {
		if ctx.Err() != nil {
			p.Output(fmt.Sprintf("[%s] 已中断（用户取消）", time.Now().Format("15:04:05")))
			return ctx.Err()
		}
		end := start + batch
		if end > total {
			end = total
		}
		semDocs := make([]semantic.Doc, 0, end-start)
		for _, d := range docs[start:end] {
			semDocs = append(semDocs, fileindex.Doc(d))
		}
		if _, err := st.Ensure(ctx, e, "file", semDocs); err != nil {
			p.Output(fmt.Sprintf("[%s] ✗ 批次 %d-%d 失败：%v", time.Now().Format("15:04:05"), start+1, end, err))
			return err
		}
		p.Output(fmt.Sprintf("[%s] 已索引 %d/%d 个文件", time.Now().Format("15:04:05"), end, total))
		p.Report(end*100/total, fmt.Sprintf("已索引 %d/%d 个文件", end, total))
	}
	_, _ = st.Stale("file", keep)
	out, _ := json.Marshal(map[string]any{"total": total, "skipped": skipped})
	p.Result(string(out))
	p.Output(fmt.Sprintf("[%s] 索引完成：%d 个文件（跳过 %d）", time.Now().Format("15:04:05"), total, skipped))
	p.Report(100, fmt.Sprintf("索引完成：%d 个文件（跳过 %d）", total, skipped))
	return nil
}

// errFileIndexTaskUnavailable 是调度器未启动时给 manual 入口的人话错误
// （与收敛前 GaeaFileIndexRebuild 的文案逐字一致：绑定面错误文本零变化）。
var errFileIndexTaskUnavailable = errors.New("任务调度器未启动")

// submitFileIndexTaskIn 是工作区语义索引任务的**唯一**入队入口（审计 AP4-07
// 单源）：去重判据（HasActiveInSpace）与落库空间（SubmitSpaceSession）取同一个
// space，三个入口（manual 绑定 / cron 轮询兜底 / watch 兜底与自愈）不再各写一份。
//
// 收敛前是三套口径：gaea_file_index.go 两处 HasActive（全局、跨空间）+ Submit
// （space 落缺省 work），gaea_tasks.go 一处 HasActiveInSpace(work) +
// SubmitSpace(work)——同一件事的判据空间与落库空间可以不同值。
//
//   - space：调用方所在空间（空/非法回退 work，与 spaces.Normalize 同侧）；
//   - session：调用方会话标识（manual 绑定签名不含会话、cron/watch 无会话
//     上下文 → 诚实留空，不造数；会话内索引调用方传自己的 session）；
//   - 返回 (nil, nil) = 去重命中：**同一空间**已有 queued/running 的索引任务
//     （调用方按「已在处理」处理：manual 转人话错误，后台入口静默跳过）；
//   - 跨空间互不吞并：play 在途不挡 work 入队，反之亦然（S1.4 按空间去重）。
func (a *App) submitFileIndexTaskIn(space, session, label, reason string) (*tasks.Task, error) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return nil, errFileIndexTaskUnavailable
	}
	space = spaces.SpaceOr(strings.ToLower(strings.TrimSpace(space)), spaces.SpaceWork)
	if m.HasActiveInSpace(tasks.KindFileIndex, space) {
		return nil, nil
	}
	return m.SubmitSpaceSession(tasks.KindFileIndex, label, map[string]any{"reason": reason}, space, session)
}

// fileIndexSpace 是工作区索引任务空间归属的单点取法（AP4-07）：取调用方当前
// 生效空间（gaeaEffectiveSpace：play 会话 → play；space.mode=off 或引擎未初始化
// 为 ""），空/非法经 spaces.SpaceOr 回退 work（与提交入口同一次归一）。
//
// 三个入口共用本函数，保证任意时刻「去重判据的空间」与「落库的空间」同值：
// 收敛前 manual/cron 落 work 却按全局判重、watch 按 work 判重，play 空间的
// 在途索引对 watch 不可见（并发扫同一工作区，Stale 互相覆盖 keep 集合）。
//
// 与价格抓取 cron（gaea_price_sources.go 显式 work）的差别是有意的：价格源是
// 工作空间的领域数据，而文件索引扫的是**整个工作区**，其空间归属只能跟随当前
// 生效空间，否则 play 模式下 manual（play）与 cron（work）会各起一个并发全量扫。
func fileIndexSpace() string {
	return spaces.SpaceOr(strings.ToLower(gaeaEffectiveSpace()), spaces.SpaceWork)
}

// submitFileIndexTask 后台维护入口（cron 轮询兜底 / watch 兜底与自愈；无会话
// 空间，空间取 fileIndexSpace 与 manual 同点）：去重命中静默跳过（正常路径），
// 提交失败只留日志（后台入口无接收方，不外抛；调度器未启动启动路径已告警，
// 保持收敛前的静默）。
func (a *App) submitFileIndexTask(label, reason string) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return
	}
	if _, err := a.submitFileIndexTaskIn(fileIndexSpace(), "", label, reason); err != nil {
		slog.Warn("tasks: 索引任务提交失败", "error", err)
	}
}

// ─── 消费者：实时文件监听增量索引（T5-2） ────────────────────

// watchHealthInterval 是 fileWatchLoop 周期健康检查间隔（默认 2 分钟；包级变量
// 便于测试缩短）。
var watchHealthInterval = 2 * time.Minute

// watchPollInterval 是回退轮询间隔（默认 10 分钟；包级变量便于测试缩短）。
var watchPollInterval = 10 * time.Minute

// startFileWatch 启动工作区 fsnotify 实时监听；不可用/启动失败时立即触发一次
// 全量索引并回退轮询（兑现「实时监听失败回退轮询」承诺，调度器去重）。
func (a *App) startFileWatch() {
	root := gaeaCwd()
	w, err := filewatch.New(root, filewatch.DefaultSkipDirs, 2*time.Second)
	if err != nil {
		slog.Warn("filewatch: 实时监听不可用，回退轮询", "error", err)
		a.submitFileIndexTask("工作区语义索引", "watch-fallback-new")
		a.officeState.fileWatchPollStop = a.startWatchPollingFallback("new-failed")
		return
	}
	// 先关旧 watcher 再换新（2026-09-20 审计：Startup 不幂等，直接覆盖泄漏
	// 旧 fsnotify 句柄与事件循环）。
	if old := a.officeState.fileWatch; old != nil {
		_ = old.Close()
		a.officeState.fileWatch = nil
	}
	a.officeState.fileWatch = w
	go a.fileWatchLoop()
	if err := w.Start(); err != nil {
		slog.Warn("filewatch: 启动失败，回退轮询", "error", err)
		a.submitFileIndexTask("工作区语义索引", "watch-fallback-start")
		a.officeState.fileWatchPollStop = a.startWatchPollingFallback("start-failed")
		return
	}
	slog.Info("filewatch: 工作区实时监听已启动", "root", root)
}

// fileWatchSource 是 fileWatchLoop 依赖的监听器最小接口（便于测试注入 fake）。
type fileWatchSource interface {
	Events() <-chan filewatch.Event
	WatchErr() error
}

func (a *App) fileWatchLoop() {
	w := a.officeState.fileWatch
	if w == nil {
		return
	}
	a.fileWatchLoopWith(w, watchHealthInterval)
}

// fileWatchLoopWith 消费监听事件批次，并周期检查 WatchErr：实时监听中途出现
// 错误（WatchErr 非空）时触发全量重建兜底（经任务队列去重），兑现「实时监听
// 异常回退轮询」承诺——不再依赖事件静默丢失。interval 供测试缩短。
func (a *App) fileWatchLoopWith(w fileWatchSource, interval time.Duration) {
	if w == nil {
		return
	}
	if interval <= 0 {
		interval = watchHealthInterval
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case ev, ok := <-w.Events():
			if !ok {
				return
			}
			if ev.Full {
				// 目录级变更/事件风暴：全量重建（经任务队列去重）
				a.submitFileIndexTask("工作区语义索引", "watch-full")
				continue
			}
			a.applyIncrementalFileIndex(ev)
		case <-t.C:
			if err := w.WatchErr(); err != nil {
				slog.Warn("filewatch: 监听异常（WatchErr），触发全量重建兜底", "error", err)
				a.submitFileIndexTask("工作区语义索引", "watch-error")
			}
		}
	}
}

// startWatchPollingFallback 启动回退轮询：周期提交全量索引任务（调度器去重），
// 兑现「实时监听不可用/启动失败时 10 分钟轮询」承诺。返回停止函数（测试用）。
func (a *App) startWatchPollingFallback(reason string) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(watchPollInterval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				a.submitFileIndexTask("工作区语义索引", "watch-poll-"+reason)
			case <-stop:
				return
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

// applyIncrementalFileIndex 对监听批次做增量索引：删除的直接清向量，变更的
// 提取正文后 Ensure（内容感知，只重嵌变化文件）；失败自愈为全量重建任务。
func (a *App) applyIncrementalFileIndex(ev filewatch.Event) {
	e := a.localSearchEmbedder()
	if e == nil {
		return
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, rel := range ev.Removed {
		if err := st.Remove("file", rel); err != nil {
			slog.Warn("filewatch: 删除向量失败", "path", rel, "error", err)
		}
	}
	for _, rel := range ev.Changed {
		abs := filepath.Join(gaeaCwd(), rel)
		info, err := os.Stat(abs)
		if err != nil || !info.Mode().IsRegular() || info.Size() > fileindex.MaxFileBytes {
			continue
		}
		if !fileindex.Supported(rel) {
			continue
		}
		text, err := fileindex.Extract(abs)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}
		if _, err := st.Ensure(ctx, e, "file", []semantic.Doc{{ID: rel, Text: text}}); err != nil {
			slog.Warn("filewatch: 增量索引失败，触发全量重建", "path", rel, "error", err)
			a.submitFileIndexTask("工作区语义索引", "watch-selfheal")
			return
		}
	}
}
