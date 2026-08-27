package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/tasks"
)

// FileIndexStatus 是工作区文件语义索引状态。
type FileIndexStatus struct {
	Total   int    `json:"total"`   // 已索引文件数
	Skipped int    `json:"skipped"` // 跳过（不支持/超限/空文本）
	Error   string `json:"error"`
}

// FileSemanticHit 是文件语义检索命中。
type FileSemanticHit struct {
	Path    string  `json:"path"`
	Score   float64 `json:"score"`
	Snippet string  `json:"snippet"`
}

// GaeaFileIndexRebuild 扫描工作区并增量建立文件语义索引（异步任务，T5-1）：
// 提交索引任务入队并立即返回任务视图，进度经 gaea-task 事件推送；结果
// （total/skipped）在任务 result 里，失败原因在任务 error 里。
func (a *App) GaeaFileIndexRebuild() (*tasks.Task, error) {
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return nil, fmt.Errorf("任务调度器未启动")
	}
	if m.HasActive(tasks.KindFileIndex) {
		return nil, fmt.Errorf("索引任务已在队列中，请稍候")
	}
	return m.Submit(tasks.KindFileIndex, "工作区语义索引", map[string]any{"reason": "manual"})
}

// startFileIndexCron 文件语义索引自动维护：启动即查 + 每 10 分钟增量重建
// （Ensure 内容感知，只处理新增/变更文件）。embedding 不可用时静默跳过。
func (a *App) startFileIndexCron() {
	a.officeState.fileIndexOnce.Do(func() {
		stop := make(chan struct{})
		a.officeState.fileIndexStop = stop
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("file index cron panic recovered", "panic", r)
				}
			}()
			a.tickFileIndex()
			ticker := time.NewTicker(10 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					a.tickFileIndex()
				}
			}
		}()
	})
}

// tickFileIndex 轮询兜底：实时监听健康时跳过（增量由 watch 负责），否则
// 提交全量索引任务（T5-1 队列去重）。
func (a *App) tickFileIndex() {
	if a.officeState != nil {
		if w := a.officeState.fileWatch; w != nil && w.Healthy() {
			return
		}
	}
	m := a.taskMgr()
	if m == nil || !m.Available() {
		return
	}
	if m.HasActive(tasks.KindFileIndex) {
		return
	}
	if _, err := m.Submit(tasks.KindFileIndex, "工作区语义索引（轮询兜底）", map[string]any{"reason": "cron"}); err != nil {
		slog.Warn("tasks: 索引任务提交失败", "error", err)
	}
}

// GaeaFileSemanticSearch 对已索引的工作区文件做语义检索（本地 bge-m3）。
func (a *App) GaeaFileSemanticSearch(query string, topN int) ([]FileSemanticHit, error) {
	return a.fileSemanticHits(query, topN)
}

// fileSemanticHits 文件语义检索的共用私有实现（统一检索聚合复用，T-记忆统一层）：
// 逻辑与 GaeaFileSemanticSearch 完全一致——embedding/索引不可用时返回空数组
// 而不报错（与跨库语义检索同降级语义），查询超时 60s。
func (a *App) fileSemanticHits(query string, topN int) ([]FileSemanticHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	e := a.localSearchEmbedder()
	if e == nil {
		return nil, nil
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil, nil
	}
	if topN <= 0 {
		topN = 10
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	hits, err := st.SearchReady(ctx, e, "file", query, topN)
	if err != nil {
		return nil, err
	}
	out := make([]FileSemanticHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, FileSemanticHit{
			Path:    h.ID,
			Score:   h.Score,
			Snippet: fileSnippet(h.Text),
		})
	}
	return out, nil
}

func fileSnippet(s string) string {
	clean := strings.Join(strings.Fields(s), " ")
	r := []rune(clean)
	if len(r) > 240 {
		return string(r[:240]) + "…"
	}
	return clean
}
