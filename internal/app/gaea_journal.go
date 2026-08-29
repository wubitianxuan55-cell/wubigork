package app

// GaeaJournalList 返回 v4.1 证据链 Journal 的最近证据卡（跨会话聚合，
// 按时间倒序，limit 上限）。读取 `.gaea/work/journal/*.jsonl`；
// 目录缺失/无记录 → 空列表（只读兼容：旧工作区无 Journal 不报错）。
import (
	"os"
	"path/filepath"
	"sort"

	"github.com/gaea/gaea/internal/gaea/evidence"
)

func (a *App) GaeaJournalList(limit int) []evidence.ChangeRecord {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	dir := filepath.Join(gaeaCwd(), ".gaea", "work", "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []evidence.ChangeRecord{}
	}
	st, err := evidence.OpenJournal(dir)
	if err != nil {
		return []evidence.ChangeRecord{}
	}
	var all []evidence.ChangeRecord
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		recs, err := st.List(e.Name()[:len(e.Name())-len(".jsonl")])
		if err != nil {
			continue
		}
		all = append(all, recs...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].At > all[j].At })
	if len(all) > limit {
		all = all[:limit]
	}
	return all
}
