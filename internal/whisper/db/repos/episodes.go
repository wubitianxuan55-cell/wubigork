// Package repos — episodes 情节仓库
// 100% 对齐 ackem src/main/db/repos/episodes.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

type episodeRow struct {
	ID                 string
	Summary            string
	EmotionalIntensity float64
	DominantEmotion    string
	Keywords           string
	PrevEpisodeID      sql.NullString
	SourceSessionID    string
	StartTurn          int
	EndTurn            int
	CreatedAt          string
}

func (r *episodeRow) toEpisode() whisper.Episode {
	ep := whisper.Episode{
		ID:                 r.ID,
		Summary:            r.Summary,
		EmotionalIntensity: r.EmotionalIntensity,
		DominantEmotion:    r.DominantEmotion,
		SourceSessionID:    r.SourceSessionID,
		StartTurn:          r.StartTurn,
		EndTurn:            r.EndTurn,
	}

	if r.PrevEpisodeID.Valid {
		s := r.PrevEpisodeID.String
		ep.PrevEpisodeID = &s
	}

	json.Unmarshal([]byte(r.Keywords), &ep.Keywords)

	t, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err == nil {
		ep.CreatedAt = t
	}

	return ep
}

// CountEpisodesInDB 返回情节数
func CountEpisodesInDB(dataRoot string) int {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return 0
	}
	var c int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM episodes").Scan(&c); err != nil {
		slog.Warn("episodes: COUNT 失败", "error", err)
	}
	return c
}

// LoadEpisodesFromDB 加载全部情节（按创建时间升序）
func LoadEpisodesFromDB(dataRoot string) ([]whisper.Episode, error) {
	return queryEpisodes(dataRoot, "")
}

// LoadEpisodesFromDBForSession 按会话加载情节（角色记忆隔离：每个角色只恢复自己的）
func LoadEpisodesFromDBForSession(dataRoot, sessionID string) ([]whisper.Episode, error) {
	return queryEpisodes(dataRoot, "WHERE source_session_id = ?", sessionID)
}

func queryEpisodes(dataRoot, where string, args ...interface{}) ([]whisper.Episode, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	rows, err := sqlDB.Query(`SELECT id, summary, emotional_intensity, dominant_emotion, keywords,
		prev_episode_id, source_session_id, start_turn, end_turn, created_at
		FROM episodes `+where+` ORDER BY created_at ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []whisper.Episode
	for rows.Next() {
		var r episodeRow
		if err := rows.Scan(&r.ID, &r.Summary, &r.EmotionalIntensity, &r.DominantEmotion,
			&r.Keywords, &r.PrevEpisodeID, &r.SourceSessionID, &r.StartTurn, &r.EndTurn, &r.CreatedAt); err != nil {
			continue
		}
		episodes = append(episodes, r.toEpisode())
	}
	// 迭代/扫描中断不再静默返部分数据（2026-09-19 审计：情节恢复主路径）
	return episodes, rows.Err()
}

// ReplaceEpisodesInDB 全量替换情节
func ReplaceEpisodesInDB(dataRoot string, episodes []whisper.Episode) error {
	return db.WithTransaction(dataRoot, func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM episodes"); err != nil {
			return fmt.Errorf("清空 episodes 失败: %w", err)
		}

		for _, ep := range episodes {
			if err := insertEpisodeTx(tx, ep); err != nil {
				return err
			}
		}
		return nil
	})
}

// InsertEpisode 单条插入情节（FTS 增量，失败降级全量重建——2026-09-19 审计：
// 原实现每写一条全量重建索引，批量写 O(N²)）。
func InsertEpisode(dataRoot string, ep whisper.Episode) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	if err := insertEpisodeStmt(sqlDB, ep); err != nil {
		return err
	}
	keywordsJSON, _ := json.Marshal(ep.Keywords)
	if err := InsertEpisodeFTS(dataRoot, ep.ID, ep.Summary, string(keywordsJSON), ep.DominantEmotion); err != nil {
		return RebuildEpisodesFTS(dataRoot)
	}
	return nil
}

// DeleteAllEpisodesFromDB 清空所有情节（FTS 索引直接整表删——全空场景无需
// 读零行再重建一遍）。
func DeleteAllEpisodesFromDB(dataRoot string) error {
	if err := db.WithTransaction(dataRoot, func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM episodes"); err != nil {
			return err
		}
		_, err := tx.Exec("DELETE FROM episodes_fts")
		return err
	}); err != nil {
		return err
	}
	return nil
}

// ─── 内部 ────────────────────────────────────────────────────────

const insertEpisodeSQL = `INSERT INTO episodes(
	id, summary, emotional_intensity, dominant_emotion, keywords,
	prev_episode_id, source_session_id, start_turn, end_turn, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func insertEpisodeTx(tx *sql.Tx, ep whisper.Episode) error {
	_, err := tx.Exec(insertEpisodeSQL, episodeArgs(ep)...)
	return err
}

func insertEpisodeStmt(sqldb *sql.DB, ep whisper.Episode) error {
	_, err := sqldb.Exec(insertEpisodeSQL, episodeArgs(ep)...)
	return err
}

func episodeArgs(ep whisper.Episode) []interface{} {
	keywordsJSON, _ := json.Marshal(ep.Keywords)
	createdAt := ep.CreatedAt.Format(time.RFC3339)

	var prevID sql.NullString
	if ep.PrevEpisodeID != nil && *ep.PrevEpisodeID != "" {
		prevID = nullStr(*ep.PrevEpisodeID)
	}

	return []interface{}{
		ep.ID, ep.Summary, ep.EmotionalIntensity, ep.DominantEmotion,
		string(keywordsJSON), prevID, ep.SourceSessionID,
		ep.StartTurn, ep.EndTurn, createdAt,
	}
}
