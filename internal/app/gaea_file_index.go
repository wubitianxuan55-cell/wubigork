package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileindex"
	"github.com/gaea/gaea/internal/gaea/semantic"
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

// GaeaFileIndexRebuild 扫描工作区并增量建立文件语义索引（复用 semantic_vectors
// kind=file）；已索引文件跳过、删除的清理、新增的向量化。返回索引状态。
func (a *App) GaeaFileIndexRebuild() (FileIndexStatus, error) {
	return a.rebuildFileIndex()
}

// rebuildFileIndex 扫描工作区并增量建立/刷新文件语义索引（复用 semantic_vectors
// kind=file）；Ensure 内容感知（变更自动重嵌），Stale 清理已删除文件。
func (a *App) rebuildFileIndex() (FileIndexStatus, error) {
	docs, skipped, err := fileindex.Scan(gaeaCwd())
	if err != nil {
		return FileIndexStatus{Error: err.Error()}, err
	}
	e := a.localSearchEmbedder()
	if e == nil {
		return FileIndexStatus{Error: "本地 embedding 未配置（Herdsman bge-m3）"}, fmt.Errorf("本地 embedding 未配置")
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return FileIndexStatus{Error: "向量索引不可用"}, fmt.Errorf("向量索引不可用")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	semDocs := make([]semantic.Doc, 0, len(docs))
	keep := make(map[string]bool, len(docs))
	for _, d := range docs {
		semDocs = append(semDocs, fileindex.Doc(d))
		keep[d.Path] = true
	}
	if _, err := st.Ensure(ctx, e, "file", semDocs); err != nil {
		return FileIndexStatus{Total: len(docs), Skipped: skipped, Error: err.Error()}, err
	}
	_, _ = st.Stale("file", keep)
	return FileIndexStatus{Total: len(docs), Skipped: skipped}, nil
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

func (a *App) tickFileIndex() {
	if _, err := a.rebuildFileIndex(); err != nil {
		slog.Warn("文件语义索引自动重建失败", "error", err)
	}
}

// GaeaFileSemanticSearch 对已索引的工作区文件做语义检索（本地 bge-m3）。
func (a *App) GaeaFileSemanticSearch(query string, topN int) ([]FileSemanticHit, error) {
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
