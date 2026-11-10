package boot

// D6 publish-quarantine 启动清扫（dsh-ops 蒸馏 2026-11-10 第二波）。
// 挂在 Build 的工作区装配点：goroutine+recover 异步跑（启动关键路径不做
// 整树遍历），每进程每工作区一次。清扫纪律见 tool/publishguard_ledger.go
// （先记账后放句柄/账本外名字整目录跳过/自底向上/失败留账重试）。

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/gaea/gaea/internal/gaea/tool"
)

var (
	sweepQuarantineMu   sync.Mutex
	sweepQuarantineDone = map[string]bool{}
)

// sweepQuarantineOnce 每进程每工作区跑一次的启动清扫。
func sweepQuarantineOnce(cwd string) {
	sweepQuarantineMu.Lock()
	if sweepQuarantineDone[cwd] {
		sweepQuarantineMu.Unlock()
		return
	}
	sweepQuarantineDone[cwd] = true
	sweepQuarantineMu.Unlock()
	root := filepath.Join(cwd, ".gaea", "publish-quarantine")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("quarantine sweep panic recovered", "panic", r)
			}
		}()
		stamps, files, err := tool.SweepQuarantine(root, 0)
		switch {
		case err != nil:
			slog.Warn("quarantine sweep failed (will retry next launch)", "root", root, "error", err)
		case stamps > 0:
			slog.Info("quarantine sweep", "root", root, "stamps", stamps, "files", files)
		}
	}()
}
