package memoryeval

// 三域 runner：把种子语料接进各域真实检索路径（story=internal/memory BM25、
// work/experience=internal/gaea/bm25、persona=whisper SQLite FTS），
// 返回 Evaluate 所需的「query → 排序命中 ID」函数。CI 可跑、零外部引擎依赖。

import (
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/gaea/bm25"
	"github.com/gaea/gaea/internal/memory"
	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db/repos"
)

// RunStory 项目 StoryMemory 域：BM25 索引（章节摘要种子）→ Search 取前 K 的 ID。
func RunStory(seeds []memory.Memory, items []Item) DomainReport {
	idx := memory.NewIndex()
	for _, m := range seeds {
		idx.Add(m)
	}
	return Evaluate("story", func(query string) []string {
		var ids []string
		for _, m := range idx.Search(query, TopK) {
			ids = append(ids, m.ID)
		}
		return ids
	}, items)
}

// RunWork 办公事实/知识域：bm25.Ranker（docs[i].ID == i ↔ labels[i]）。
func RunWork(docs []bm25.Doc, labels []string, items []Item) DomainReport {
	ranker := bm25.NewRanker(docs)
	return Evaluate("work", func(query string) []string {
		var ids []string
		for _, s := range ranker.Rank(query) {
			if s.ID >= 0 && s.ID < len(labels) {
				ids = append(ids, labels[s.ID])
			}
		}
		return ids
	}, items)
}

// RunExperience 经验习得域：与 work 同款 bm25（经验条目语料独立成库，
// 与 7.2 技能结晶的未来评测口径一致）。
func RunExperience(docs []bm25.Doc, labels []string, items []Item) DomainReport {
	report := RunWork(docs, labels, items)
	report.Domain = "experience"
	return report
}

// RunPersona 轻语人格记忆域：种子事实全量写入临时数据根目录的 hermes 库，
// 重建 FTS 后走生产检索路径 SearchFactIDsFTS（含中文 2-gram LIKE 降级）。
// root 由调用方管理生命周期（测试用独立临时目录）。
func RunPersona(root string, facts []whisper.MemoryFact, items []Item) (DomainReport, error) {
	now := time.Now()
	for i := range facts {
		if facts[i].ID == "" {
			return DomainReport{}, fmt.Errorf("第 %d 条种子事实缺 ID", i)
		}
		facts[i].CreatedAt = now
		facts[i].UpdatedAt = now
	}
	if err := repos.ReplaceFactsInDB(root, facts); err != nil {
		return DomainReport{}, fmt.Errorf("写入种子事实失败: %w", err)
	}
	if err := repos.RebuildFactsFTS(root); err != nil {
		return DomainReport{}, fmt.Errorf("重建 FTS 失败: %w", err)
	}
	return Evaluate("persona", func(query string) []string {
		ids, err := repos.SearchFactIDsFTS(root, query, TopK)
		if err != nil {
			return nil // 单条失败不中断整轮（与既有测评口径一致）
		}
		return ids
	}, items), nil
}
