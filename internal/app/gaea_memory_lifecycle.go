package app

// ── 办公记忆生命周期（T6-8.2）──────────────────────────────────
// facts 归档保留策略与滚动清理：
//   - 归档超过 ArchivedRetention（90 天）的事实由 GaeaMemoryCleanupArchived
//     硬删除，溯源字段（名称/描述/正文/归档时间/来源会话）落
//     <userDir>/memory/purge-audit.jsonl 审计侧，删除即留痕；
//   - GaeaMemoryArchivedList 分页返回归档（总量 + 当前页），防止全量返回；
//   - 活跃事实（List）不参与清理，误归档用户仍可在 90 天内恢复。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// MemoryArchivedView 是归档记忆的前端视图（分页条目）。
type MemoryArchivedView struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Kind        string `json:"kind"`
	ArchivedAt  string `json:"archivedAt"`
}

// MemoryArchivedPage 是归档记忆分页负载。
type MemoryArchivedPage struct {
	Items  []MemoryArchivedView `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
	// RetentionDays 是归档保留期（天）：归档超过该时长为硬删除候选
	// （GaeaMemoryCleanupArchived 清理），前端据此展示「归档保留 N 天」。
	RetentionDays int `json:"retentionDays"`
}

// GaeaMemoryArchivedList 分页返回归档事实（updated_at 倒序）。limit 钳制
// [1,200]（默认 50），offset 下限 0；办公引擎未初始化时返回空页。
func (a *App) GaeaMemoryArchivedList(limit, offset int) (MemoryArchivedPage, error) {
	page := MemoryArchivedPage{
		Limit:         limit,
		Offset:        offset,
		Items:         []MemoryArchivedView{},
		RetentionDays: int(memory.ArchivedRetention / (24 * time.Hour)),
	}
	store := a.hubOfficeStore()
	items, total, err := store.ListArchivedPaged(limit, offset)
	if err != nil {
		return page, fmt.Errorf("归档列表: %w", err)
	}
	page.Total = total
	page.Limit = limit
	for _, am := range items {
		page.Items = append(page.Items, MemoryArchivedView{
			Name:        am.Name,
			Title:       am.Title,
			Description: am.Description,
			Type:        string(am.Type),
			Kind:        string(am.Kind),
			ArchivedAt:  fmtTimeOrEmpty(am.ArchivedAt),
		})
	}
	return page, nil
}

// GaeaMemoryUnarchive 恢复一条已归档记忆（reverse of Archive，记忆统一层
// 第一刀「生命周期闭环」）：误归档可在保留期内一键恢复回活跃列表。
// 未归档/已被清理的事实返回明确错误。
func (a *App) GaeaMemoryUnarchive(name string) error {
	store := a.hubOfficeStore()
	if err := store.Unarchive(name); err != nil {
		return fmt.Errorf("恢复归档: %w", err)
	}
	slog.Info("记忆归档恢复", "name", name)
	return nil
}

// purgeAuditEntry 是归档清理审计行：硬删前把溯源字段完整落盘。
type purgeAuditEntry struct {
	TS            string `json:"ts"`
	Name          string `json:"name"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Kind          string `json:"kind"`
	Body          string `json:"body,omitempty"`
	ArchivedAt    string `json:"archivedAt"`
	SourceSession string `json:"sourceSession,omitempty"`
	SourceMessage string `json:"sourceMessage,omitempty"`
}

// purgeAuditPath 返回归档清理审计文件（<userDir>/memory/purge-audit.jsonl）。
func purgeAuditPath(userDir string) string {
	return filepath.Join(userDir, "memory", "purge-audit.jsonl")
}

// appendPurgeAudit 追加一条清理审计（JSONL，尽力而为，失败仅记日志）。
func appendPurgeAudit(userDir string, e purgeAuditEntry) error {
	if userDir == "" {
		return nil
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(purgeAuditPath(userDir)), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(purgeAuditPath(userDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// memoryAuditUserDir 返回记忆审计目录（GAEA_DATA_ROOT 优先，测试隔离用；
// 生产等同 gaeaConfig.MemoryUserDir）。
func memoryAuditUserDir() string {
	if v := os.Getenv("GAEA_DATA_ROOT"); v != "" {
		return v
	}
	return gaeaConfig.MemoryUserDir()
}

// GaeaMemoryCleanupArchived 硬删除归档超过保留期（90 天）的事实，返回删除
// 条数。每条删除都留 slog 日志 + 溯源审计行（purge-audit.jsonl）；无超期
// 条目返回 0 且不报错。幂等：重复调用第二次通常返回 0。
func (a *App) GaeaMemoryCleanupArchived() (int, error) {
	userDir := memoryAuditUserDir()
	store := a.hubOfficeStore()
	cutoff := time.Now().Add(-memory.ArchivedRetention)
	removed, err := store.CleanupArchived(cutoff)
	if err != nil {
		return 0, fmt.Errorf("归档清理: %w", err)
	}
	for _, am := range removed {
		entry := purgeAuditEntry{
			TS:            time.Now().UTC().Format(time.RFC3339),
			Name:          am.Name,
			Title:         am.Title,
			Description:   am.Description,
			Type:          string(am.Type),
			Kind:          string(am.Kind),
			Body:          am.Body,
			ArchivedAt:    fmtTimeOrEmpty(am.ArchivedAt),
			SourceSession: am.SourceSession,
			SourceMessage: am.SourceMessage,
		}
		slog.Info("记忆归档清理：硬删除超期事实",
			"name", am.Name, "archived_at", entry.ArchivedAt)
		if err := appendPurgeAudit(userDir, entry); err != nil {
			slog.Warn("归档清理审计写入失败", "name", am.Name, "error", err)
		}
	}
	if len(removed) > 0 {
		// 显式触发 DB 一致性兜底（无操作也安全）。
		_ = db.GetDatabase(userDir)
	}
	return len(removed), nil
}
