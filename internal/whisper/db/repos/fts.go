// Package repos — fts 全文搜索仓库
// 100% 对齐 ackem src/main/db/repos/fts.ts
package repos

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/wubigork/wubigork/internal/whisper/db"
)

// ─── FTS 重建 ────────────────────────────────────────────────────

// RebuildFactsFTS 重建 memory_facts 全文索引
func RebuildFactsFTS(dataRoot string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	// 清空 FTS 索引
	if _, err := sqlDB.Exec("DELETE FROM memory_facts_fts"); err != nil {
		// FTS 表可能不存在（首次运行），忽略
	}

	// 从主表重建
	_, err := sqlDB.Exec(`
		INSERT INTO memory_facts_fts(memory_facts_fts)
		VALUES('rebuild')
	`)
	return err
}

// RebuildEpisodesFTS 重建 episodes 全文索引
func RebuildEpisodesFTS(dataRoot string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	if _, err := sqlDB.Exec("DELETE FROM episodes_fts"); err != nil {
		// 忽略 FTS 表不存在
	}

	_, err := sqlDB.Exec(`
		INSERT INTO episodes_fts(episodes_fts)
		VALUES('rebuild')
	`)
	return err
}

// ─── FTS 增量操作 ────────────────────────────────────────────────

// InsertFactFTS 插入单条事实到 FTS 索引
func InsertFactFTS(dataRoot, factID, subject, summary, triggersText string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	_, err := sqlDB.Exec(
		"INSERT INTO memory_facts_fts(fact_id, subject, summary, triggers_text) VALUES (?, ?, ?, ?)",
		factID, subject, summary, triggersText,
	)
	return err
}

// DeleteFactFTS 从 FTS 索引删除单条事实
func DeleteFactFTS(dataRoot, factID string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	_, err := sqlDB.Exec(
		"DELETE FROM memory_facts_fts WHERE fact_id = ?",
		factID,
	)
	return err
}

// InsertEpisodeFTS 插入单条情节到 FTS 索引
func InsertEpisodeFTS(dataRoot, episodeID, summary, keywordsText, dominantEmotion string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	_, err := sqlDB.Exec(
		"INSERT INTO episodes_fts(episode_id, summary, keywords_text, dominant_emotion) VALUES (?, ?, ?, ?)",
		episodeID, summary, keywordsText, dominantEmotion,
	)
	return err
}

// DeleteEpisodeFTS 从 FTS 索引删除单条情节
func DeleteEpisodeFTS(dataRoot, episodeID string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
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
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
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

// SearchEpisodeIDsFTS 全文搜索情节（返回 ID 列表）
func SearchEpisodeIDsFTS(dataRoot, query string, limit int) ([]string, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
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

// searchFactsByLike LIKE 降级搜索
func searchFactsByLike(sqldb *sql.DB, query string, limit int) ([]string, error) {
	likePattern := "%" + strings.ReplaceAll(query, " ", "%") + "%"
	rows, err := sqldb.Query(
		fmt.Sprintf("SELECT id FROM memory_facts WHERE subject LIKE ? OR summary LIKE ? OR triggers_text LIKE ? LIMIT %d", limit),
		likePattern, likePattern, likePattern,
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

// searchEpisodesByLike LIKE 降级搜索
func searchEpisodesByLike(sqldb *sql.DB, query string, limit int) ([]string, error) {
	likePattern := "%" + strings.ReplaceAll(query, " ", "%") + "%"
	rows, err := sqldb.Query(
		fmt.Sprintf("SELECT id FROM episodes WHERE summary LIKE ? OR dominant_emotion LIKE ? LIMIT %d", limit),
		likePattern, likePattern,
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
