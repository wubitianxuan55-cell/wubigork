// Package repos — knowledge_triples 知识三元组仓库
// 100% 对齐 ackem src/main/db/repos/knowledgeTriples.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

type tripleRow struct {
	ID                 string
	Subject            string
	Predicate          string
	Object             string
	Confidence         float64
	SourceFactIDs      sql.NullString
	CreatedAt          string
	EmotionLabel       string
	EmotionalIntensity float64
	Valence            float64
}

func (r *tripleRow) toTriple() whisper.Triple {
	t := whisper.Triple{
		ID:                 r.ID,
		Subject:            r.Subject,
		Predicate:          r.Predicate,
		Object:             r.Object,
		Confidence:         r.Confidence,
		EmotionLabel:       r.EmotionLabel,
		EmotionalIntensity: r.EmotionalIntensity,
		Valence:            r.Valence,
	}

	if r.SourceFactIDs.Valid {
		json.Unmarshal([]byte(r.SourceFactIDs.String), &t.SourceFactIDs)
	}

	parsed, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err == nil {
		t.CreatedAt = parsed
	}

	return t
}

// CountTriplesInDB 返回三元组总数
func CountTriplesInDB(dataRoot string) int {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return 0
	}
	var c int
	sqlDB.QueryRow("SELECT COUNT(*) FROM knowledge_triples").Scan(&c)
	return c
}

// LoadTriplesFromDB 加载全部三元组
func LoadTriplesFromDB(dataRoot string) ([]whisper.Triple, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	rows, err := sqlDB.Query(
		"SELECT id, subject, predicate, object, confidence, source_fact_ids, created_at, emotion_label, emotional_intensity, valence FROM knowledge_triples",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triples []whisper.Triple
	for rows.Next() {
		var r tripleRow
		if err := rows.Scan(&r.ID, &r.Subject, &r.Predicate, &r.Object, &r.Confidence,
			&r.SourceFactIDs, &r.CreatedAt, &r.EmotionLabel, &r.EmotionalIntensity, &r.Valence); err != nil {
			continue
		}
		triples = append(triples, r.toTriple())
	}
	return triples, nil
}

// ReplaceTriplesInDB 全量替换三元组
func ReplaceTriplesInDB(dataRoot string, triples []whisper.Triple) error {
	return db.WithTransaction(dataRoot, func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM knowledge_triples"); err != nil {
			return err
		}
		for _, t := range triples {
			if err := insertTripleTx(tx, t); err != nil {
				return err
			}
		}
		return nil
	})
}

// InsertTriple 单条插入三元组
func InsertTriple(dataRoot string, t whisper.Triple) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}
	return insertTripleStmt(sqlDB, t)
}

// DeleteAllTriplesFromDB 清空所有三元组
func DeleteAllTriplesFromDB(dataRoot string) error {
	return db.WithTransaction(dataRoot, func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM knowledge_triples")
		return err
	})
}

func insertTripleTx(tx *sql.Tx, t whisper.Triple) error {
	_, err := tx.Exec(
		"INSERT INTO knowledge_triples(id, subject, predicate, object, confidence, source_fact_ids, created_at, emotion_label, emotional_intensity, valence) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		tripleArgs(t)...,
	)
	return err
}

func insertTripleStmt(sqldb *sql.DB, t whisper.Triple) error {
	_, err := sqldb.Exec(
		"INSERT INTO knowledge_triples(id, subject, predicate, object, confidence, source_fact_ids, created_at, emotion_label, emotional_intensity, valence) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		tripleArgs(t)...,
	)
	return err
}

func tripleArgs(t whisper.Triple) []interface{} {
	srcJSON, _ := json.Marshal(t.SourceFactIDs)
	createdAt := t.CreatedAt.Format(time.RFC3339)

	return []interface{}{
		t.ID, t.Subject, t.Predicate, t.Object, t.Confidence,
		string(srcJSON), createdAt, t.EmotionLabel, t.EmotionalIntensity, t.Valence,
	}
}
