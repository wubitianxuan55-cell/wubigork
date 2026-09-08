package app

// logging.go — 长期日志机制（v4.163）
//
// 设计：按日分文件 <DataRoot>/logs/gaea-YYYYMMDD.log——天然轮转、归档友好、
// 单文件永不无限膨胀。启动时三件事：清理过期文件（保留 logKeepDays 天）、
// 总量兜底（logs 目录超 logDirCapBytes 从最旧删）、一次性迁移旧
// whisper_data/gaea.log 进 logs/gaea-legacy.log（历史不丢）。前端诊断
// （GaeaLogFrontendError → slog）自动落同一文件，无需单独通道。

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	// logKeepDays 长期保留天数（一年）：过期的按日日志启动时清理。
	logKeepDays = 365
	// logDirCapBytes logs 目录总量兜底（1GB）：超限从最旧文件删起。
	logDirCapBytes = int64(1) << 30
)

// logFileNameFor 按日日志文件名：gaea-20060102.log。
func logFileNameFor(t time.Time) string {
	return "gaea-" + t.Format("20060102") + ".log"
}

// setupLogging 建日志目录、迁移旧日志、清理过期、打开今日文件并设为 slog 默认。
// 返回关闭函数（Shutdown 时调用）；打开失败时调用方回退不设 slog（GUI 无控制台，
// 日志缺失但不致命）。启动头写版本与 Go 版本，排障可对齐发布线。
func setupLogging(dataRoot string, version string) (func(), error) {
	dir := filepath.Join(dataRoot, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("建日志目录失败: %w", err)
	}
	migrateLegacyLog(filepath.Join(dataRoot, "whisper_data", "gaea.log"), filepath.Join(dir, "gaea-legacy.log"))
	pruneLogs(dir, time.Now(), logKeepDays, logDirCapBytes)
	f, err := os.OpenFile(filepath.Join(dir, logFileNameFor(time.Now())), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("=== gaea startup ===", "version", version, "go", runtime.Version())
	return func() { _ = f.Close() }, nil
}

// migrateLegacyLog 一次性把旧 gaea.log 搬进 logs（目标已存在=已迁过，跳过）。
func migrateLegacyLog(oldPath, newPath string) {
	info, err := os.Stat(oldPath)
	if err != nil || info.IsDir() {
		return
	}
	if _, err := os.Stat(newPath); err == nil {
		return
	}
	_ = os.Rename(oldPath, newPath)
}

// pruneLogs 日志清理：删超过 keepDays 的按日日志（文件名日期解析失败者豁免，
// 如 gaea-legacy.log）；若目录总量仍超 cap，按修改时间从最旧删起。
func pruneLogs(dir string, now time.Time, keepDays int, capBytes int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := now.AddDate(0, 0, -keepDays)
	var total int64
	var byAge []logFileEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		total += info.Size()
		name := e.Name()
		day, ok := parseLogDay(name)
		if ok && day.Before(cutoff) {
			if rmErr := os.Remove(filepath.Join(dir, name)); rmErr == nil {
				total -= info.Size()
			}
			continue
		}
		byAge = append(byAge, logFileEntry{name: name, mod: info.ModTime(), size: info.Size()})
	}
	if total <= capBytes {
		return
	}
	sort.Slice(byAge, func(i, j int) bool { return byAge[i].mod.Before(byAge[j].mod) })
	for _, f := range byAge {
		if total <= capBytes {
			break
		}
		if rmErr := os.Remove(filepath.Join(dir, f.name)); rmErr == nil {
			total -= f.size
		}
	}
}

type logFileEntry struct {
	name string
	mod  time.Time
	size int64
}

// parseLogDay 解析按日日志名（gaea-20060102.log → 日期）；其余名（legacy/
// 异物）返回 false 不参与过期清理。
func parseLogDay(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, "gaea-") || !strings.HasSuffix(name, ".log") {
		return time.Time{}, false
	}
	day, err := time.ParseInLocation("20060102", strings.TrimSuffix(strings.TrimPrefix(name, "gaea-"), ".log"), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return day, true
}
