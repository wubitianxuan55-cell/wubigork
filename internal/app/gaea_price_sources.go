package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/cost"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/pricefeed"
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

// tickPriceCron 检查并抓取所有到期的启用价格源。
func (a *App) tickPriceCron() {
	store := a.hubPriceStore()
	if !store.Available() {
		return
	}
	now := time.Now()
	for _, src := range store.ListSources() {
		if !src.Enabled || src.FrequencyHours <= 0 {
			continue
		}
		if src.LastFetchAt != "" {
			if t, err := time.Parse(time.RFC3339, src.LastFetchAt); err == nil && now.Sub(t) < time.Duration(src.FrequencyHours)*time.Hour {
				continue
			}
		}
		ctx := a.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
		if err != nil {
			slog.Warn("价格源定时抓取失败", "source", src.Name, "error", err)
			continue
		}
		res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
		_ = store.SaveFetch(pricefeed.FetchRecord{
			SourceID: src.ID, SourceName: src.Name, URL: src.URL,
			Period: res.Period, FetchedAt: res.FetchedAt, Status: "pending",
			Candidates: res.Candidates,
		})
		_ = store.TouchSource(src.ID, res.FetchedAt)
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

// GaeaPriceFetch 立即抓取指定价格源：解析 → 匹配成本库 → 存 pending 记录。
func (a *App) GaeaPriceFetch(id string) (pricefeed.FetchRecord, error) {
	store := a.hubPriceStore()
	src, ok := store.GetSource(id)
	if !ok {
		return pricefeed.FetchRecord{}, fmt.Errorf("价格源不存在: %s", id)
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
	if err != nil {
		return pricefeed.FetchRecord{}, err
	}
	res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
	rec := pricefeed.FetchRecord{
		SourceID: src.ID, SourceName: src.Name, URL: src.URL,
		Period: res.Period, FetchedAt: res.FetchedAt, Status: "pending",
		Candidates: res.Candidates,
	}
	if err := store.SaveFetch(rec); err != nil {
		return pricefeed.FetchRecord{}, err
	}
	_ = store.TouchSource(src.ID, res.FetchedAt)
	return rec, nil
}

// GaeaPriceFetchAll 一键抓取全部启用的价格源（并发）。单个源失败不阻断
// 其他源；返回成功抓取数与失败汇总（无失败时汇总为空串）。
func (a *App) GaeaPriceFetchAll() (int, string) {
	store := a.hubPriceStore()
	sources := store.ListSources()
	var enabled []pricefeed.Source
	for _, src := range sources {
		if src.Enabled {
			enabled = append(enabled, src)
		}
	}
	if len(enabled) == 0 {
		return 0, "没有启用的价格源，请先添加并启用"
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var (
		mu   sync.Mutex
		ok   int
		errs []string
		wg   sync.WaitGroup
	)
	for _, src := range enabled {
		wg.Add(1)
		go func(src pricefeed.Source) {
			defer wg.Done()
			res, err := pricefeed.Fetch(ctx, src, a.hubCostStore())
			if err != nil {
				mu.Lock()
				errs = append(errs, src.Name+": "+err.Error())
				mu.Unlock()
				return
			}
			res.Candidates = pricefeed.DetectAnomalies(res.Candidates, a.priceHistoryLookup())
			rec := pricefeed.FetchRecord{
				SourceID: src.ID, SourceName: src.Name, URL: src.URL,
				Period: res.Period, FetchedAt: res.FetchedAt, Status: "pending",
				Candidates: res.Candidates,
			}
			if err := store.SaveFetch(rec); err != nil {
				mu.Lock()
				errs = append(errs, src.Name+": "+err.Error())
				mu.Unlock()
				return
			}
			_ = store.TouchSource(src.ID, res.FetchedAt)
			mu.Lock()
			ok++
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	return ok, strings.Join(errs, "；")
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
		}
		if existing, err := costStore.Get(name); err == nil && existing != nil {
			entry.Category = existing.Category
			entry.Tags = existing.Tags
			entry.Body = existing.Body
			if existing.CreatedAt.Unix() > 0 {
				entry.CreatedAt = existing.CreatedAt
			}
		}
		if err := costStore.Save(entry); err != nil {
			return applied, fmt.Errorf("写入 %s 失败: %w", c.Title, err)
		}
		_ = store.AddHistory(pricefeed.History{
			Name: name, Title: c.Title, Unit: c.Unit, Price: c.Price,
			Source: rec.SourceName, Period: rec.Period, FetchedAt: rec.FetchedAt,
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
