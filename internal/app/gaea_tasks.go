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
		}, tasks.Options{})
		m.Register(tasks.KindPriceFetch, a.priceFetchTaskHandler)
		m.Register(tasks.KindPriceFetchAll, a.priceFetchAllTaskHandler)
		m.Register(tasks.KindFileIndex, a.fileIndexTaskHandler)
		a.officeState.tasks = m
		if n, err := m.Start(); err != nil {
			slog.Warn("tasks: 调度器启动失败", "error", err)
		} else if n > 0 {
			slog.Info("tasks: 调度器启动，续跑任务", "count", n)
		}
	})
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

// GaeaTaskList 返回最近任务（新→旧，供任务中心）。
func (a *App) GaeaTaskList() []tasks.Task {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return nil
	}
	list, err := m.List(50)
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

// GaeaTaskRetry 重试一个失败/已取消的任务。
func (a *App) GaeaTaskRetry(id string) error {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return fmt.Errorf("任务调度器未启动")
	}
	return m.Retry(id)
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
	p.Report(5, "正在抓取 "+src.Name)
	res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
	if err != nil {
		return err
	}
	res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
	rec := pricefeed.FetchRecord{
		ID: fmt.Sprintf("fetch-%d", time.Now().UnixNano()), // SaveFetch 按值拷贝，ID 须先预生成
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
	for i, src := range todo {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		p.Report(i*100/len(todo), fmt.Sprintf("正在抓取 %s（%d/%d）", src.Name, i+1, len(todo)))
		res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
		if err != nil {
			errs = append(errs, src.Name+": "+err.Error())
			continue
		}
		res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
		rec := pricefeed.FetchRecord{
			ID: fmt.Sprintf("fetch-%d", time.Now().UnixNano()), // SaveFetch 按值拷贝，ID 须先预生成
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
	for start := 0; start < total; start += batch {
		if ctx.Err() != nil {
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
			return err
		}
		p.Report(end*100/total, fmt.Sprintf("已索引 %d/%d 个文件", end, total))
	}
	_, _ = st.Stale("file", keep)
	out, _ := json.Marshal(map[string]any{"total": total, "skipped": skipped})
	p.Result(string(out))
	p.Report(100, fmt.Sprintf("索引完成：%d 个文件（跳过 %d）", total, skipped))
	return nil
}

// submitFileIndexTask 提交索引任务（去重：已有 queued/running 则跳过）。
func (a *App) submitFileIndexTask(reason string) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return
	}
	if m.HasActive(tasks.KindFileIndex) {
		return
	}
	if _, err := m.Submit(tasks.KindFileIndex, "工作区语义索引", map[string]any{"reason": reason}); err != nil {
		slog.Warn("tasks: 索引任务提交失败", "error", err)
	}
}

// ─── 消费者：实时文件监听增量索引（T5-2） ────────────────────

// startFileWatch 启动工作区 fsnotify 实时监听（失败回退 10 分钟轮询）。
func (a *App) startFileWatch() {
	root := gaeaCwd()
	w, err := filewatch.New(root, filewatch.DefaultSkipDirs, 2*time.Second)
	if err != nil {
		slog.Warn("filewatch: 实时监听不可用，回退轮询", "error", err)
		return
	}
	a.officeState.fileWatch = w
	go a.fileWatchLoop()
	if err := w.Start(); err != nil {
		slog.Warn("filewatch: 启动失败，回退轮询", "error", err)
		return
	}
	slog.Info("filewatch: 工作区实时监听已启动", "root", root)
}

func (a *App) fileWatchLoop() {
	w := a.officeState.fileWatch
	if w == nil {
		return
	}
	for ev := range w.Events() {
		if ev.Full {
			// 目录级变更/事件风暴：全量重建（经任务队列去重）
			a.submitFileIndexTask("watch-full")
			continue
		}
		a.applyIncrementalFileIndex(ev)
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
			a.submitFileIndexTask("watch-selfheal")
			return
		}
	}
}
