// Package repos — episodes 情节仓库
// 100% 对齐 ackem src/main/db/repos/episodes.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wubigork/wubigork/internal/whisper"
	"github.com/wubigork/wubigork/internal/whisper/db"
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
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return 0
	}
	var c int
	sqlDB.QueryRow("SELECT COUNT(*) FROM episodes").Scan(&c)
	return c
}

// LoadEpisodesFromDB 加载全部情节（按创建时间升序）
func LoadEpisodesFromDB(dataRoot string) ([]whisper.Episode, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
	}

	rows, err := sqlDB.Query(`SELECT id, summary, emotional_intensity, dominant_emotion, keywords,
		prev_episode_id, source_session_id, start_turn, end_turn, created_at
		FROM episodes ORDER BY created_at ASC`)
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
	return episodes, nil
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

// InsertEpisode 单条插入情节
func InsertEpisode(dataRoot string, ep whisper.Episode) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	if err := insertEpisodeStmt(sqlDB, ep); err != nil {
		return err
	}
	return RebuildEpisodesFTS(dataRoot)
}

// DeleteAllEpisodesFromDB 清空所有情节
func DeleteAllEpisodesFromDB(dataRoot string) error {
	if err := db.WithTransaction(dataRoot, func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM episodes")
		return err
	}); err != nil {
		return err
	}
	return RebuildEpisodesFTS(dataRoot)
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
