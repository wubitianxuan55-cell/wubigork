package app

// eval_snapshot_delete.go — 评测历史快照删除（v4.448）。
//
// persist 接线（v4.447）后快照随每次点按累积，删除是卫生需要。护栏：
// 文件名白名单（毫秒时间戳格式，v4.447 定形）——路径穿越与任意文件删除
// 双防；删除目标必须位于本项目 eval/snapshots/ 内。

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var evalSnapshotNameRe = regexp.MustCompile(`^\d{8}T\d{6}\.\d{3}Z\.json$`)

// NovelEvalSnapshotDelete 删除一份历史快照。name 必须匹配快照文件名白名单
// （防路径穿越/任意删），文件不存在时报错（幂等交给前端不重试）。
func (a *writingState) NovelEvalSnapshotDelete(name string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if !evalSnapshotNameRe.MatchString(name) {
		return fmt.Errorf("快照文件名非法: %q", name)
	}
	path := filepath.Join(pm.Dir, "eval", "snapshots", name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("快照不存在（可能已删除）: %s", name)
		}
		return fmt.Errorf("删除快照失败: %w", err)
	}
	return nil
}
