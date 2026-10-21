package fileutil

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// lastCorruptStamp 上一次损坏备份的时间戳，用于把隔离文件名做成进程内严格
// 递增：Windows 时钟粒度粗，同一进程内连续两次隔离可能拿到相同纳秒，纯
// time.Now().UnixNano() 会互相覆盖（corrupt_test 实测抓到）。
var lastCorruptStamp atomic.Int64

func nextCorruptStamp() int64 {
	for {
		now := time.Now().UnixNano()
		last := lastCorruptStamp.Load()
		if now > last && lastCorruptStamp.CompareAndSwap(last, now) {
			return now
		}
		// 同粒度或时钟回拨：在前值上 +1，保证严格递增不互覆。
		if lastCorruptStamp.CompareAndSwap(last, last+1) {
			return last + 1
		}
	}
}

// BackupCorrupt 把无法解析的持久化文件内容另存为 <path>.corrupt-<纳秒时间戳>
// 后备副本（原文件不动，等下次正常写盘覆盖），返回备份路径。加载侧损坏自愈
// 的共用单源（壳层 .gaea_config.json 与 gaea config.toml/recent-workspaces.json
// 的断电级损坏防护，2026-10-06 断电全零事故根修落地）：调用方拿到备份路径后
// Warn 记账并按默认值继续启动；备份失败返回错误，由调用方决定是否仍继续。
func BackupCorrupt(path string, data []byte) (string, error) {
	backup := fmt.Sprintf("%s.corrupt-%d", path, nextCorruptStamp())
	if err := os.WriteFile(backup, data, 0o644); err != nil {
		return "", err
	}
	return backup, nil
}
