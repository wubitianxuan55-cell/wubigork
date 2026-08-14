package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	gaeaBackup "github.com/gaea/gaea/internal/gaea/backup"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	gaeadb "github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/config"
)

// ── P4-3 数据可迁移（2026-08-14，个人使用收口）────────────────────
// 一键备份/恢复：Hephaestus.db（记忆/知识/成本/语义向量）+ whisper_data
// （hermes/office/角色库/聊天）+ config.toml + sessions + home 配置。

// dataBackupPlan 构造备份计划（数据根 + 显式条目）。
func (a *App) dataBackupPlan() *gaeaBackup.Plan {
	root := config.DataRoot()
	skip := []string{"gaea.log", "-wal", "-shm", ".restore-", "backups"}
	sources := []gaeaBackup.Source{
		{ZipRel: "whisper_data", Abs: filepath.Join(root, "whisper_data")},
	}
	// Hephaestus.db：默认在 DataRoot 下；若 MemoryUserDir 与 DataRoot 不同则显式加入
	hepPath := gaeadb.DatabasePath(gaeaConfig.MemoryUserDir())
	if _, err := os.Stat(hepPath); err == nil {
		sources = append(sources, gaeaBackup.Source{ZipRel: filepath.Base(hepPath), Abs: hepPath})
	}
	// config.toml（办公引擎配置）
	if p := gaeaConfig.UserConfigPath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			sources = append(sources, gaeaBackup.Source{ZipRel: filepath.Base(p), Abs: p})
		}
	}
	// sessions（用户级会话）
	if p := gaeaConfig.SessionDir(); p != "" {
		if _, err := os.Stat(p); err == nil {
			sources = append(sources, gaeaBackup.Source{ZipRel: "sessions", Abs: p})
		}
	}
	// archive（压缩归档）
	if p := gaeaConfig.ArchiveDir(); p != "" {
		if _, err := os.Stat(p); err == nil {
			sources = append(sources, gaeaBackup.Source{ZipRel: "archive", Abs: p})
		}
	}
	// home 配置（zip 内 home-config/gaea_config.json）
	if home, err := os.UserHomeDir(); err == nil {
		cfgPath := filepath.Join(home, ".gaea_config.json")
		if _, err := os.Stat(cfgPath); err == nil {
			sources = append(sources, gaeaBackup.Source{ZipRel: gaeaBackup.HomeConfigRel, Abs: cfgPath})
		}
	}
	return gaeaBackup.NewPlan(root, sources, skip)
}

// BackupEntryView 备份清单条目（前端展示）。
type BackupEntryView struct {
	Path   string `json:"path"`
	Abs    string `json:"abs"`
	Size   int64  `json:"size"`
	Exists bool   `json:"exists"`
	SQLite bool   `json:"sqlite,omitempty"`
}

// GaeaDataBackupInfo 返回数据根、备份清单与各条目大小（供设置页「数据」展示）。
func (a *App) GaeaDataBackupInfo() map[string]interface{} {
	plan := a.dataBackupPlan()
	entries := make([]BackupEntryView, 0, len(plan.Sources))
	var total int64
	for _, s := range plan.Sources {
		info, err := os.Stat(s.Abs)
		if err != nil {
			entries = append(entries, BackupEntryView{Path: s.ZipRel, Abs: s.Abs, Exists: false})
			continue
		}
		size := int64(0)
		if info.IsDir() {
			size = dirSize(s.Abs)
		} else {
			size = info.Size()
		}
		total += size
		entries = append(entries, BackupEntryView{Path: s.ZipRel, Abs: s.Abs, Size: size, Exists: true, SQLite: strings.HasSuffix(strings.ToLower(s.ZipRel), ".db")})
	}
	pending, _ := gaeaBackup.ReadPending(config.DataRoot())
	res := map[string]interface{}{
		"data_root":   plan.Root,
		"entries":     entries,
		"total_bytes": total,
		"pending":     pending != nil,
		"app_version": AppVersion,
	}
	if pending != nil {
		res["pending_zip"] = pending.ZipName
		res["pending_at"] = pending.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return res
}

// GaeaDataBackupCreate 一键备份到 destDir（目录选择器返回；空则用数据根/backups）。
func (a *App) GaeaDataBackupCreate(destDir string) (map[string]interface{}, error) {
	if destDir == "" {
		destDir = filepath.Join(config.DataRoot(), "backups")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建备份目录失败: %w", err)
	}
	zipPath := filepath.Join(destDir, fmt.Sprintf("gaea-backup-%s-%s.zip", AppVersion, time.Now().Format("20060102-150405")))
	m, err := a.dataBackupPlan().Create(zipPath, AppVersion)
	if err != nil {
		return nil, err
	}
	sum, _ := gaeaBackup.SHA256(zipPath)
	slog.Info("数据备份完成", "zip", zipPath, "entries", m.EntryCount, "bytes", m.TotalBytes)
	return map[string]interface{}{
		"zip_path":    zipPath,
		"entries":     m.EntryCount,
		"total_bytes": m.TotalBytes,
		"sha256":      sum,
		"created_at":  m.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// GaeaDataBackupRestore 从 zip 恢复（两阶段）：校验 → 解压到 staging → 写 pending。
func (a *App) GaeaDataBackupRestore(zipPath string) (map[string]interface{}, error) {
	root := config.DataRoot()
	m, err := gaeaBackup.ReadManifest(zipPath)
	if err != nil {
		return nil, err
	}
	stageDir := filepath.Join(root, ".restore-stage-"+time.Now().Format("20060102-150405"))
	if _, err := gaeaBackup.Extract(zipPath, stageDir); err != nil {
		_ = os.RemoveAll(stageDir)
		return nil, fmt.Errorf("解压备份失败: %w", err)
	}
	homeDir, _ := os.UserHomeDir()
	state := gaeaBackup.PendingState{
		StageDir:  stageDir,
		ZipName:   filepath.Base(zipPath),
		CreatedAt: time.Now(),
		DataRoot:  root,
		HomeDir:   homeDir,
	}
	if err := gaeaBackup.WritePending(state); err != nil {
		_ = os.RemoveAll(stageDir)
		return nil, fmt.Errorf("写入恢复标记失败: %w", err)
	}
	slog.Info("数据恢复已暂存，重启后生效", "zip", zipPath, "stage", stageDir)
	return map[string]interface{}{
		"restart_required": true,
		"zip_name":         state.ZipName,
		"stage_dir":        stageDir,
		"backup_version":   m.Version,
		"created_at":       state.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// GaeaDataBackupPending 查询是否有待应用恢复。
func (a *App) GaeaDataBackupPending() map[string]interface{} {
	pending, _ := gaeaBackup.ReadPending(config.DataRoot())
	if pending == nil {
		return map[string]interface{}{"pending": false}
	}
	return map[string]interface{}{
		"pending":    true,
		"zip_name":   pending.ZipName,
		"created_at": pending.CreatedAt.Format("2006-01-02 15:04:05"),
		"stage_dir":  pending.StageDir,
	}
}

// GaeaDataBackupCancel 取消待应用恢复（清理 staging + 标记）。
func (a *App) GaeaDataBackupCancel() error {
	return gaeaBackup.ClearPending(config.DataRoot())
}

// GaeaDataBackupRestoreResult 读取上次恢复应用结果（重启后前端提示）。
func (a *App) GaeaDataBackupRestoreResult() map[string]interface{} {
	root := config.DataRoot()
	data, err := os.ReadFile(filepath.Join(root, ".restore-result.json"))
	if err != nil {
		return map[string]interface{}{"has_result": false}
	}
	var r map[string]interface{}
	if err := json.Unmarshal(data, &r); err != nil {
		return map[string]interface{}{"has_result": false}
	}
	out := map[string]interface{}{"has_result": true}
	for _, k := range []string{"applied", "zip_name", "before_dir", "applied_at", "error"} {
		if v, ok := r[k]; ok {
			out[k] = v
		}
	}
	return out
}

// applyPendingRestore 应用待恢复数据（Startup 早期调用；失败保留 pending）。
func (a *App) applyPendingRestore() {
	root := config.DataRoot()
	home, _ := os.UserHomeDir()
	res, err := gaeaBackup.ApplyPending(root, home)
	if err != nil {
		slog.Error("应用数据恢复失败（保留 pending 可重试）", "error", err, "zip", res.ZipName)
		return
	}
	if res.Applied {
		slog.Info("数据恢复已应用", "zip", res.ZipName, "before", res.BeforeDir)
	}
}

// dirSize 递归统计目录大小（只统计普通文件）。
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if fi, e := d.Info(); e == nil {
				total += fi.Size()
			}
		}
		return nil
	})
	return total
}
