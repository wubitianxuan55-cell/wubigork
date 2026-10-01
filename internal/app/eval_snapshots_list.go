package app

// eval_snapshots_list.go — 评测历史快照索引（v4.447）。
//
// 快照持久化（persist=true → eval/snapshots/<UTC 时间戳>.json）自 v4.435 存在，
// 但面板此前走 persist=false、快照从不落盘——历史无从谈起。本绑定补索引面：
// 时间倒序 + 每份核心指标（供前端历史表）。坏档跳过（continue），目录不存在
// 返回空切片非 nil（空态口径与 LintForeshadows 族一致）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NovelEvalSnapshotsList 历史快照索引（新→旧）。
func (a *writingState) NovelEvalSnapshotsList() ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	dir := filepath.Join(pm.Dir, "eval", "snapshots")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	// 文件名=UTC 时间戳（20060102T150405Z），字典序即时间序；倒序=新→旧。
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	out := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var body evalSnapshotBody
		if err := json.Unmarshal(raw, &body); err != nil {
			continue // 坏档跳过，不拖垮整个索引
		}
		out = append(out, map[string]interface{}{
			"name":         name,
			"chapters":     body.Chapters,
			"chars":        body.Chars,
			"tasteMean":    body.Taste.Mean,
			"s1":           body.Quality.S1,
			"s2":           body.Quality.S2,
			"s3":           body.Quality.S3,
			"recall":       body.Foreshadow.Recall,
			"tensionMean":  body.Tension.Mean,
			"tensionCover": body.Tension.Covered,
		})
	}
	return out, nil
}
