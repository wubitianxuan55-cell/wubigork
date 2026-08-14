// Package repos — fts 全文搜索仓库
// 100% 对齐 ackem src/main/db/repos/fts.ts
package repos

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/whisper/db"
)

// RebuildFactsFTS 重建 memory_facts 全文索引（独立表全量同步）
func RebuildFactsFTS(dataRoot string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	// 清空 FTS 索引（V11 后为独立表，可安全 DELETE）
	if _, err := sqlDB.Exec("DELETE FROM memory_facts_fts"); err != nil {
		return err
	}

	// 先全量读入内存再插入：MaxOpenConns(1) 下 rows 未 Close 时 Exec 会死锁等连接
	rows, err := sqlDB.Query(
		"SELECT id, subject, summary, COALESCE(triggers_text, '') FROM memory_facts",
	)
	if err != nil {
		return err
	}
	type factRow struct{ id, subject, summary, triggers string }
	var facts []factRow
	for rows.Next() {
		var r factRow
		if err := rows.Scan(&r.id, &r.subject, &r.summary, &r.triggers); err != nil {
			continue
		}
		facts = append(facts, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range facts {
		if _, err := sqlDB.Exec(
			"INSERT INTO memory_facts_fts(fact_id, subject, summary, triggers_text) VALUES (?, ?, ?, ?)",
			r.id, r.subject, r.summary, r.triggers,
		); err != nil {
			return err
		}
	}
	return nil
}

func RebuildEpisodesFTS(dataRoot string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	if _, err := sqlDB.Exec("DELETE FROM episodes_fts"); err != nil {
		return err
	}

	// 同 RebuildFactsFTS：先读后写避免 MaxOpenConns(1) 死锁
	rows, err := sqlDB.Query(
		"SELECT id, summary, COALESCE(keywords, ''), COALESCE(dominant_emotion, '') FROM episodes",
	)
	if err != nil {
		return err
	}
	type epRow struct{ id, summary, keywords, emotion string }
	var eps []epRow
	for rows.Next() {
		var r epRow
		if err := rows.Scan(&r.id, &r.summary, &r.keywords, &r.emotion); err != nil {
			continue
		}
		eps = append(eps, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range eps {
		if _, err := sqlDB.Exec(
			"INSERT INTO episodes_fts(episode_id, summary, keywords_text, dominant_emotion) VALUES (?, ?, ?, ?)",
			r.id, r.summary, r.keywords, r.emotion,
		); err != nil {
			return err
		}
	}
	return nil
}

// ─── FTS 增量操作 ────────────────────────────────────────────────

// InsertFactFTS 插入单条事实到 FTS 索引
func InsertFactFTS(dataRoot, factID, subject, summary, triggersText string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	_, err := sqlDB.Exec(
		"INSERT INTO memory_facts_fts(fact_id, subject, summary, triggers_text) VALUES (?, ?, ?, ?)",
		factID, subject, summary, triggersText,
	)
	return err
}

// DeleteFactFTS 从 FTS 索引删除单条事实
func DeleteFactFTS(dataRoot, factID string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	_, err := sqlDB.Exec(
		"DELETE FROM memory_facts_fts WHERE fact_id = ?",
		factID,
	)
	return err
}

// InsertEpisodeFTS 插入单条情节到 FTS 索引
func InsertEpisodeFTS(dataRoot, episodeID, summary, keywordsText, dominantEmotion string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	_, err := sqlDB.Exec(
		"INSERT INTO episodes_fts(episode_id, summary, keywords_text, dominant_emotion) VALUES (?, ?, ?, ?)",
		episodeID, summary, keywordsText, dominantEmotion,
	)
	return err
}

// DeleteEpisodeFTS 从 FTS 索引删除单条情节
func DeleteEpisodeFTS(dataRoot, episodeID string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	_, err := sqlDB.Exec(
		"DELETE FROM episodes_fts WHERE episode_id = ?",
		episodeID,
	)
	return err
}

// ─── FTS 搜索 ────────────────────────────────────────────────────

// SearchFactIDsFTS 全文搜索记忆事实（返回 ID 列表）
func SearchFactIDsFTS(dataRoot, query string, limit int) ([]string, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	if limit <= 0 {
		limit = 10
	}

	// 构建 MATCH 查询：每个词用双引号 + OR 拼接
	match := buildFTSMatch(query)
	if match == "" {
		return nil, nil
	}

	rows, err := sqlDB.Query(
		fmt.Sprintf("SELECT fact_id FROM memory_facts_fts WHERE memory_facts_fts MATCH ? LIMIT %d", limit),
		match,
	)
	if err != nil {
		// FTS MATCH 失败，降级到 LIKE
		return searchFactsByLike(sqlDB, query, limit)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		// FTS 整词匹配对中文子串召回弱（如「美式」匹配不到「美式咖啡」整词），
		// MATCH 成功但空结果时同样降级到 LIKE 子串匹配
		return searchFactsByLike(sqlDB, query, limit)
	}
	return ids, nil
}

// SearchEpisodeIDsFTS 全文搜索情节（返回 ID 列表）
func SearchEpisodeIDsFTS(dataRoot, query string, limit int) ([]string, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	if limit <= 0 {
		limit = 10
	}

	match := buildFTSMatch(query)
	if match == "" {
		return nil, nil
	}

	rows, err := sqlDB.Query(
		fmt.Sprintf("SELECT episode_id FROM episodes_fts WHERE episodes_fts MATCH ? LIMIT %d", limit),
		match,
	)
	if err != nil {
		return searchEpisodesByLike(sqlDB, query, limit)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) == 0 {
		// 同 SearchFactIDsFTS：中文子串召回降级 LIKE
		return searchEpisodesByLike(sqlDB, query, limit)
	}
	return ids, nil
}

// ─── 内部 ────────────────────────────────────────────────────────

// buildFTSMatch 构建 FTS5 MATCH 查询字符串
// 每个词用双引号包裹做短语匹配，用 OR 连接
func buildFTSMatch(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	quoted := make([]string, len(words))
	for i, w := range words {
		// FTS5 双引号转义
		escaped := strings.ReplaceAll(w, `"`, `""`)
		quoted[i] = `"` + escaped + `"`
	}
	return strings.Join(quoted, " OR ")
}

// buildLikePatterns 生成中文友好的 LIKE 匹配模式
// 整句 + 相邻 2-gram（中文无空格，整句 LIKE 对口语问句几乎必空——用户说「咖啡」
// 而摘要「她喜欢喝美式咖啡」不含整句；2-gram「咖啡」能命中。去重避免重复扫描）
func buildLikePatterns(query string) []string {
	runes := []rune(strings.TrimSpace(query))
	if len(runes) == 0 {
		return nil
	}
	pats := []string{string(runes)}
	seen := map[string]bool{string(runes): true}
	for i := 0; i+1 < len(runes); i++ {
		g := string(runes[i : i+2])
		if !seen[g] {
			seen[g] = true
			pats = append(pats, g)
		}
	}
	return pats
}

// escapeLike 转义 LIKE 通配符
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// searchFactsByLike LIKE 降级搜索（整句 + 2-gram 多模式 OR）
func searchFactsByLike(sqldb *sql.DB, query string, limit int) ([]string, error) {
	pats := buildLikePatterns(query)
	if len(pats) == 0 {
		return nil, nil
	}
	// subject/summary/triggers_text 三字段 × N 模式，全部 OR
	fields := []string{"subject", "summary", "triggers_text"}
	var conds []string
	var args []interface{}
	for _, p := range pats {
		like := "%" + escapeLike(p) + "%"
		for _, f := range fields {
			conds = append(conds, f+" LIKE ? ESCAPE '\\'")
			args = append(args, like)
		}
	}
	rows, err := sqldb.Query(
		fmt.Sprintf("SELECT id FROM memory_facts WHERE %s LIMIT %d", strings.Join(conds, " OR "), limit),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// searchEpisodesByLike LIKE 降级搜索（整句 + 2-gram 多模式 OR）
func searchEpisodesByLike(sqldb *sql.DB, query string, limit int) ([]string, error) {
	pats := buildLikePatterns(query)
	if len(pats) == 0 {
		return nil, nil
	}
	// summary/dominant_emotion 两字段 × N 模式，全部 OR
	fields := []string{"summary", "dominant_emotion"}
	var conds []string
	var args []interface{}
	for _, p := range pats {
		like := "%" + escapeLike(p) + "%"
		for _, f := range fields {
			conds = append(conds, f+" LIKE ? ESCAPE '\\'")
			args = append(args, like)
		}
	}
	rows, err := sqldb.Query(
		fmt.Sprintf("SELECT id FROM episodes WHERE %s LIMIT %d", strings.Join(conds, " OR "), limit),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}
