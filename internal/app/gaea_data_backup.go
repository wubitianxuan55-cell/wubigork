package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gaeaBackup "github.com/gaea/gaea/internal/gaea/backup"
	gaeadb "github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/config"
)

// ── P4-3 数据可迁移（2026-08-14，个人使用收口）────────────────────
// 一键备份/恢复：Hephaestus.db（记忆/知识/成本/语义向量）+ whisper_data
// （hermes/office/角色库/聊天）+ config.toml + sessions + home 配置。

// dataBackupPlan 构造备份计划（数据根 + 显式条目）。
// 所有条目统一从 config.DataRoot() 派生（GAEA_DATA_ROOT 测试隔离可覆盖；
// 未设置时 DataRoot == MemoryUserDir/UserConfigPath/ArchiveDir 等，生产行为不变）——
// 避免混用 MemoryUserDir 系路径导致测试 zip 混入真实用户目录（相对路径穿越）。
func (a *App) dataBackupPlan() *gaeaBackup.Plan {
	root := config.DataRoot()
	skip := []string{"gaea.log", "-wal", "-shm", ".restore-", "backups"}
	sources := []gaeaBackup.Source{
		{ZipRel: "whisper_data", Abs: filepath.Join(root, "whisper_data")},
	}
	// Hephaestus.db：位于数据根下（默认 UserConfigDir/gaea/Hephaestus.db，与历史一致）
	hepPath := gaeadb.DatabasePath(root)
	if _, err := os.Stat(hepPath); err == nil {
		sources = append(sources, gaeaBackup.Source{ZipRel: filepath.Base(hepPath), Abs: hepPath})
	}
	// config.toml（办公引擎配置）
	if p := filepath.Join(root, "config.toml"); p != "" {
		if _, err := os.Stat(p); err == nil {
			sources = append(sources, gaeaBackup.Source{ZipRel: filepath.Base(p), Abs: p})
		}
	}
	// sessions（用户级会话）
	if p := filepath.Join(root, "sessions"); p != "" {
		if _, err := os.Stat(p); err == nil {
			sources = append(sources, gaeaBackup.Source{ZipRel: "sessions", Abs: p})
		}
	}
	// archive（压缩归档）
	if p := filepath.Join(root, "archive"); p != "" {
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
	pending, pendingErr := gaeaBackup.ReadPending(config.DataRoot())
	res := map[string]interface{}{
		"data_root":   plan.Root,
		"entries":     entries,
		"total_bytes": total,
		"pending":     pending != nil,
		"app_version": AppVersion,
	}
	if pendingErr != nil {
		res["pending_error"] = pendingErr.Error() // #16：损坏标记不再静默显示"无 pending"
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
	// #10：文件名毫秒级时间戳 + 随机后缀，防同秒覆盖
	zipPath := filepath.Join(destDir, fmt.Sprintf("gaea-backup-%s-%s-%s.zip", AppVersion, time.Now().Format("20060102-150405.000"), randomSuffix(3)))
	if _, err := os.Stat(zipPath); err == nil {
		return nil, fmt.Errorf("备份文件已存在: %s", zipPath)
	}
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
	// #5：已有待应用恢复时拒绝再次恢复，避免堆叠/覆盖与孤儿 staging
	if existing, _ := gaeaBackup.ReadPending(root); existing != nil {
		return nil, fmt.Errorf("已有待应用恢复（%s，%s）。请先取消或重启完成后再试",
			existing.ZipName, existing.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	m, err := gaeaBackup.ReadManifest(zipPath)
	if err != nil {
		return nil, err
	}
	stageDir := filepath.Join(root, ".restore-stage-"+time.Now().Format("20060102-150405")+"-"+randomSuffix(4))
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
	pending, err := gaeaBackup.ReadPending(config.DataRoot())
	if err != nil {
		return map[string]interface{}{"pending": false, "pending_error": err.Error()}
	}
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

// GaeaDataBackupCancel 取消待应用恢复（清理 staging + 标记 + 孤儿 staging）。
func (a *App) GaeaDataBackupCancel() error {
	return gaeaBackup.ClearPending(config.DataRoot())
}

// GaeaDataBackupRollback 把恢复前数据备份目录（.restore-before）移回数据根（#7）。
// 用于恢复失败/取消后回滚到恢复前状态。返回是否执行了回滚。
func (a *App) GaeaDataBackupRollback() (bool, error) {
	return gaeaBackup.RollbackBefore(config.DataRoot())
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

// randomSuffix 生成短随机后缀（staging 目录防同秒撞名）。
func randomSuffix(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return string(b)
}

// dirSizeCache 目录大小缓存（#6：避免每次进设置页全量递归扫描大目录阻塞 Wails 调用）。
// 键 = 目录绝对路径；值按目录 mtime 失效——目录内容变化必然更新父目录 mtime（含子目录新增/删除），
// 足够接近真实；对 GB 级 whisper_data 从"每次全扫"降到"目录改动才扫"。
var dirSizeCache = struct {
	sync.Mutex
	m map[string]dirSizeEntry
}{m: make(map[string]dirSizeEntry)}

type dirSizeEntry struct {
	size  int64
	modTS int64 // 目录 mtime unix nano
	at    time.Time
}

const dirSizeCacheTTL = 15 * time.Second

// dirSize 递归统计目录大小（带缓存：mtime + TTL 失效）。
func dirSize(dir string) int64 {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return 0
	}
	key := dir
	modTS := info.ModTime().UnixNano()
	now := time.Now()

	dirSizeCache.Lock()
	if e, ok := dirSizeCache.m[key]; ok && e.modTS == modTS && now.Sub(e.at) < dirSizeCacheTTL {
		dirSizeCache.Unlock()
		return e.size
	}
	dirSizeCache.Unlock()

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

	dirSizeCache.Lock()
	dirSizeCache.m[key] = dirSizeEntry{size: total, modTS: modTS, at: now}
	dirSizeCache.Unlock()
	return total
}
