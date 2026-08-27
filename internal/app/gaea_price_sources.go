package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/pricefeed"
	"github.com/gaea/gaea/internal/gaea/tasks"
)

// hubPriceStore 构造价格源存储（Hephaestus.db SchemaV4，与成本库同库）。
func (a *App) hubPriceStore() *pricefeed.Store {
	return pricefeed.Open(db.GetDatabase(config.MemoryUserDir()))
}

// startPriceCron 启动价格源定时抓取调度：每 30 分钟检查一次到期源
// （frequency_hours>0 且距上次抓取超过频率），到期即抓取并存 pending 记录。
// 幂等：首次调用启动，Shutdown 时停止。
func (a *App) startPriceCron() {
	a.officeState.priceCronOnce.Do(func() {
		stop := make(chan struct{})
		a.officeState.priceCronStop = stop
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("price cron panic recovered", "panic", r)
				}
			}()
			a.tickPriceCron()
			ticker := time.NewTicker(30 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					a.tickPriceCron()
				}
			}
		}()
	})
}

// tickPriceCron 检查并提交「定时价格抓取」任务（T5-1）：到期源过滤在任务
// handler 内按 cron 语义执行；已有排队/运行中的抓取任务则跳过（去重）。
func (a *App) tickPriceCron() {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return
	}
	if m.HasActive(tasks.KindPriceFetchAll) || m.HasActive(tasks.KindPriceFetch) {
		return
	}
	if _, err := m.Submit(tasks.KindPriceFetchAll, "定时价格抓取", map[string]any{"cron": true}); err != nil {
		slog.Warn("tasks: 定时价格抓取提交失败", "error", err)
	}
}

// GaeaPriceSources 返回全部价格源订阅。
func (a *App) GaeaPriceSources() []pricefeed.Source {
	return a.hubPriceStore().ListSources()
}

// GaeaPriceSourceSave 保存/更新价格源订阅。
func (a *App) GaeaPriceSourceSave(src pricefeed.Source) error {
	return a.hubPriceStore().SaveSource(src)
}

// GaeaPriceSourceDelete 删除价格源订阅（保留抓取记录与价格历史）。
func (a *App) GaeaPriceSourceDelete(id string) error {
	return a.hubPriceStore().DeleteSource(id)
}

// GaeaPriceFetch 立即抓取指定价格源（异步任务，T5-1）：提交任务入队并立即
// 返回任务视图，前端经 gaea-task 事件观察进度/完成，完成后读 GaeaPriceFetches
// 取 pending 记录。同一价格源已有排队/运行中的抓取任务时不重复提交。
func (a *App) GaeaPriceFetch(id string) (*tasks.Task, error) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return nil, fmt.Errorf("任务调度器未启动")
	}
	store := a.hubPriceStore()
	src, ok := store.GetSource(id)
	if !ok {
		return nil, fmt.Errorf("价格源不存在: %s", id)
	}
	// 去重：同一价格源已有活动抓取任务则直接返回
	for _, t := range a.GaeaTaskList() {
		if t.Kind == string(tasks.KindPriceFetch) && (t.Status == "queued" || t.Status == "running") {
			var req priceFetchPayload
			if json.Unmarshal([]byte(t.Payload), &req) == nil && req.SourceID == id {
				return &t, nil
			}
		}
	}
	return m.Submit(tasks.KindPriceFetch, "抓取 "+src.Name, map[string]any{"sourceId": id})
}

// GaeaPriceFetchAll 一键抓取全部启用的价格源（异步任务，T5-1）：提交任务
// 入队并立即返回任务视图，逐源进度经 gaea-task 事件推送，失败明细在任务
// 结果/消息里（前端任务中心可见）。
func (a *App) GaeaPriceFetchAll() (*tasks.Task, error) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return nil, fmt.Errorf("任务调度器未启动")
	}
	return m.Submit(tasks.KindPriceFetchAll, "一键抓取全部价格源", map[string]any{"cron": false})
}

// GaeaPriceFetches 返回最近抓取记录（含 pending 待确认）。
func (a *App) GaeaPriceFetches() []pricefeed.FetchRecord {
	return a.hubPriceStore().ListFetches(30)
}

// GaeaPriceFetchApply 确认发布抓取结果：把选中的候选写回成本库并记录价格历史，
// 然后整条抓取记录标记 applied；返回实际写入条数。
func (a *App) GaeaPriceFetchApply(fetchID string, titles []string) (int, error) {
	store := a.hubPriceStore()
	var rec pricefeed.FetchRecord
	for _, f := range store.ListFetches(50) {
		if f.ID == fetchID {
			rec = f
			break
		}
	}
	if rec.ID == "" {
		return 0, fmt.Errorf("抓取记录不存在: %s", fetchID)
	}
	if rec.Status != "pending" {
		return 0, fmt.Errorf("该抓取记录已处理（%s）", rec.Status)
	}

	want := map[string]bool{}
	for _, t := range titles {
		want[strings.TrimSpace(t)] = true
	}
	costStore := a.hubCostStore()
	applied := 0
	// 抓价来源的地区（造价信息网价格表通常按城市/区发布），写回条目时作为
	// 价格三要素的「地区」维度（zaojia-database 蒸馏）。
	var area string
	if src, ok := store.GetSource(rec.SourceID); ok {
		area = src.Area
	}
	for _, c := range rec.Candidates {
		if !want[c.Title] {
			continue
		}
		name := strings.TrimSpace(c.ExistingName)
		if name == "" {
			name = cost.SlugName(c.Title)
		}
		entry := cost.Entry{
			Name: name, Title: c.Title, Category: "其他", Unit: c.Unit,
			Price: c.Price, Spec: c.Spec, Status: "现行",
			Source: fmt.Sprintf("%s（期数 %s）", rec.SourceName, rec.Period),
			Region: area, PriceDate: rec.Period,
		}
		if existing, err := costStore.Get(name); err == nil && existing != nil {
			entry.Category = existing.Category
			entry.Tags = existing.Tags
			entry.Body = existing.Body
			entry.PriceType = existing.PriceType
			entry.ValidUntil = existing.ValidUntil
			entry.SourceRow = existing.SourceRow
			if existing.CreatedAt.Unix() > 0 {
				entry.CreatedAt = existing.CreatedAt
			}
		}
		if err := costStore.Save(entry); err != nil {
			return applied, fmt.Errorf("写入 %s 失败: %w", c.Title, err)
		}
		_ = store.AddHistory(pricefeed.History{
			Name: name, Title: c.Title, Unit: c.Unit, Price: c.Price,
			Source: rec.SourceName, Period: rec.Period,
			Region: entry.Region, PriceType: entry.PriceType, FetchedAt: rec.FetchedAt,
			Note: "价格源更新",
		})
		applied++
	}
	_ = store.SetFetchStatus(fetchID, "applied")
	return applied, nil
}

// GaeaPriceFetchIgnore 忽略整条抓取结果（不写库）。
func (a *App) GaeaPriceFetchIgnore(fetchID string) error {
	return a.hubPriceStore().SetFetchStatus(fetchID, "ignored")
}

// GaeaPriceHistory 返回某条目的价格历史（新→旧）。
func (a *App) GaeaPriceHistory(name string) []pricefeed.History {
	return a.hubPriceStore().ListHistory(name, 30)
}

// priceHistoryLookup 供价格异常识别读取某条目的最近历史价格。
func (a *App) priceHistoryLookup() func(string) []pricefeed.History {
	return func(name string) []pricefeed.History {
		return a.hubPriceStore().ListHistory(name, 8)
	}
}
