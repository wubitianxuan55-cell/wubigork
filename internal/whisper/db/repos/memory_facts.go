// Package repos — memory_facts 记忆事实仓库
// 100% 对齐 ackem src/main/db/repos/memoryFacts.ts
package repos

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/whisper/db"
)

// ─── 列映射 ──────────────────────────────────────────────────────

type factRow struct {
	ID               string
	Domain           string
	Subcategory      string
	Subject          string
	Summary          string
	Weight           float64
	Confidence       float64
	Status           string
	EmotionalContext sql.NullString
	SelfRelevance    float64
	Triggers         sql.NullString
	TriggersText     sql.NullString
	UpdateTrail      sql.NullString
	SourceSessionID  sql.NullString
	SourceTurnIndex  int
	CreatedAt        string
	UpdatedAt        string
	DerivedFrom      sql.NullString
	FactLayer        sql.NullString
	Tier             sql.NullString
	Sensitivity      sql.NullString
	PrivacyLevel     sql.NullString
	AgeValue         sql.NullInt64
	AgeBirthYear     sql.NullInt64
	AgeBirthdayMMDD  sql.NullString
	AgeRecordedAt    sql.NullString
	AgeIsEstimate    sql.NullInt64
}

func (r *factRow) toFact() whisper.MemoryFact {
	f := whisper.MemoryFact{
		ID:              r.ID,
		Domain:          r.Domain,
		Subcategory:     r.Subcategory,
		Subject:         r.Subject,
		Summary:         r.Summary,
		Weight:          r.Weight,
		Confidence:      r.Confidence,
		Status:          r.Status,
		SelfRelevance:   r.SelfRelevance,
		SourceTurnIndex: r.SourceTurnIndex,
		Tier:            valOr(r.Tier, "archival"),
		Sensitivity:     valOr(r.Sensitivity, "normal"),
		PrivacyLevel:    valOr(r.PrivacyLevel, "normal"),
		FactLayer:       valOr(r.FactLayer, "raw"),
	}

	if r.SourceSessionID.Valid {
		f.SourceSessionID = r.SourceSessionID.String
	}

	t, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err == nil {
		f.CreatedAt = t
	}
	t, err = time.Parse(time.RFC3339, r.UpdatedAt)
	if err == nil {
		f.UpdatedAt = t
	}

	if r.EmotionalContext.Valid {
		var ec whisper.EmotionalContext
		if json.Unmarshal([]byte(r.EmotionalContext.String), &ec) == nil {
			f.EmotionalContext = &ec
		}
	}
	if r.Triggers.Valid {
		json.Unmarshal([]byte(r.Triggers.String), &f.Triggers)
	}
	if r.UpdateTrail.Valid {
		json.Unmarshal([]byte(r.UpdateTrail.String), &f.UpdateTrail)
	}
	if r.DerivedFrom.Valid {
		json.Unmarshal([]byte(r.DerivedFrom.String), &f.DerivedFrom)
	}

	// AgeMeta
	if r.AgeValue.Valid && r.AgeValue.Int64 > 0 {
		am := &whisper.AgeMeta{Age: int(r.AgeValue.Int64)}
		if r.AgeBirthYear.Valid {
			am.BirthYear = int(r.AgeBirthYear.Int64)
		}
		if r.AgeBirthdayMMDD.Valid {
			am.BirthdayMMDD = r.AgeBirthdayMMDD.String
		}
		if r.AgeRecordedAt.Valid {
			am.RecordedAt = r.AgeRecordedAt.String
		} else {
			am.RecordedAt = r.CreatedAt
		}
		am.IsEstimate = r.AgeIsEstimate.Valid && r.AgeIsEstimate.Int64 == 1
		f.AgeMeta = am
	}

	return f
}

func marshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func valOr(ns sql.NullString, def string) string {
	if ns.Valid {
		return ns.String
	}
	return def
}

// ─── CRUD ────────────────────────────────────────────────────────

// CountFactsInDB 返回记忆事实总数
func CountFactsInDB(dataRoot string) int {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return 0
	}
	var c int
	sqlDB.QueryRow("SELECT COUNT(*) FROM memory_facts").Scan(&c)
	return c
}

// LoadFactsFromDB 加载全部记忆事实
func LoadFactsFromDB(dataRoot string) []whisper.MemoryFact {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil
	}

	rows, err := sqlDB.Query(`SELECT id, domain, subcategory, subject, summary, weight, confidence, status,
		emotional_context, self_relevance, triggers, triggers_text, update_trail,
		source_session_id, source_turn_index, created_at, updated_at,
		derived_from, fact_layer, tier, sensitivity, privacy_level,
		age_value, age_birth_year, age_birthday_mmdd, age_recorded_at, age_is_estimate
		FROM memory_facts`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var facts []whisper.MemoryFact
	for rows.Next() {
		var r factRow
		if err := rows.Scan(
			&r.ID, &r.Domain, &r.Subcategory, &r.Subject, &r.Summary,
			&r.Weight, &r.Confidence, &r.Status,
			&r.EmotionalContext, &r.SelfRelevance, &r.Triggers, &r.TriggersText, &r.UpdateTrail,
			&r.SourceSessionID, &r.SourceTurnIndex, &r.CreatedAt, &r.UpdatedAt,
			&r.DerivedFrom, &r.FactLayer, &r.Tier, &r.Sensitivity, &r.PrivacyLevel,
			&r.AgeValue, &r.AgeBirthYear, &r.AgeBirthdayMMDD, &r.AgeRecordedAt, &r.AgeIsEstimate,
		); err != nil {
			continue
		}
		facts = append(facts, r.toFact())
	}
	return facts
}

// ReplaceFactsInDB 全量替换记忆事实（事务 + FTS 重建）
func ReplaceFactsInDB(dataRoot string, facts []whisper.MemoryFact) error {
	return db.WithTransaction(dataRoot, func(tx *sql.Tx) error {
		if _, err := tx.Exec("DELETE FROM memory_facts"); err != nil {
			return fmt.Errorf("清空 memory_facts 失败: %w", err)
		}

		for _, f := range facts {
			if err := insertFactTx(tx, f); err != nil {
				return err
			}
		}
		return nil
	})
}

// InsertFact 单条插入记忆事实
func InsertFact(dataRoot string, f whisper.MemoryFact) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	if err := insertFactStmt(sqlDB, f); err != nil {
		return err
	}
	return RebuildFactsFTS(dataRoot)
}

// UpdateFactInDB 单条更新记忆事实
func UpdateFactInDB(dataRoot string, f whisper.MemoryFact) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	if err := updateFactStmt(sqlDB, f); err != nil {
		return err
	}
	return RebuildFactsFTS(dataRoot)
}

// DeleteFactFromDB 单条删除记忆事实
func DeleteFactFromDB(dataRoot string, id string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}
	if _, err := sqlDB.Exec("DELETE FROM memory_facts WHERE id = ?", id); err != nil {
		return err
	}
	return RebuildFactsFTS(dataRoot)
}

// ─── 内部 SQL ────────────────────────────────────────────────────

const insertFactSQL = `INSERT INTO memory_facts(
	id, domain, subcategory, subject, summary, weight, confidence, status,
	emotional_context, self_relevance, triggers, triggers_text, update_trail,
	source_session_id, source_turn_index, created_at, updated_at,
	derived_from, fact_layer, tier, sensitivity, privacy_level,
	age_value, age_birth_year, age_birthday_mmdd, age_recorded_at, age_is_estimate
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const updateFactSQL = `UPDATE memory_facts SET
	domain=?, subcategory=?, subject=?, summary=?, weight=?, confidence=?, status=?,
	emotional_context=?, self_relevance=?, triggers=?, triggers_text=?, update_trail=?,
	source_session_id=?, source_turn_index=?, created_at=?, updated_at=?,
	derived_from=?, fact_layer=?, tier=?, sensitivity=?, privacy_level=?,
	age_value=?, age_birth_year=?, age_birthday_mmdd=?, age_recorded_at=?, age_is_estimate=?
WHERE id=?`

func insertFactTx(tx *sql.Tx, f whisper.MemoryFact) error {
	_, err := tx.Exec(insertFactSQL, factArgs(f)...)
	return err
}

func insertFactStmt(sqldb *sql.DB, f whisper.MemoryFact) error {
	_, err := sqldb.Exec(insertFactSQL, factArgs(f)...)
	return err
}

func updateFactStmt(sqldb *sql.DB, f whisper.MemoryFact) error {
	args := factArgs(f)
	args = append(args, f.ID) // WHERE id=?
	_, err := sqldb.Exec(updateFactSQL, args...)
	return err
}

func factArgs(f whisper.MemoryFact) []interface{} {
	createdAt := f.CreatedAt.Format(time.RFC3339)
	updatedAt := f.UpdatedAt.Format(time.RFC3339)
	if f.UpdatedAt.IsZero() {
		updatedAt = createdAt
	}

	var ecJSON, triggersJSON, triggersText, updateTrailJSON, derivedFromJSON sql.NullString
	if f.EmotionalContext != nil {
		ecJSON = nullStr(marshalJSON(f.EmotionalContext))
	}
	if len(f.Triggers) > 0 {
		triggersJSON = nullStr(marshalJSON(f.Triggers))
		triggersText = nullStr(strings.Join(f.Triggers, " "))
	}
	if len(f.UpdateTrail) > 0 {
		updateTrailJSON = nullStr(marshalJSON(f.UpdateTrail))
	}
	if len(f.DerivedFrom) > 0 {
		derivedFromJSON = nullStr(marshalJSON(f.DerivedFrom))
	}

	factLayer := "raw"
	if f.FactLayer != "" {
		factLayer = f.FactLayer
	}
	tier := "archival"
	if f.Tier != "" {
		tier = f.Tier
	}
	sensitivity := "normal"
	if f.Sensitivity != "" {
		sensitivity = f.Sensitivity
	}
	privacyLevel := "normal"
	if f.PrivacyLevel != "" {
		privacyLevel = f.PrivacyLevel
	}

	args := []interface{}{
		f.ID, f.Domain, f.Subcategory, f.Subject, f.Summary,
		f.Weight, f.Confidence, f.Status,
		ecJSON, f.SelfRelevance, triggersJSON, triggersText, updateTrailJSON,
		nullStr(f.SourceSessionID), f.SourceTurnIndex, createdAt, updatedAt,
		derivedFromJSON, nullStr(factLayer), nullStr(tier), nullStr(sensitivity), nullStr(privacyLevel),
	}

	// AgeMeta
	var ageVal, birthYear, ageIsEstimate sql.NullInt64
	var birthdayMMDD, ageRecordedAt sql.NullString
	if f.AgeMeta != nil && f.AgeMeta.Age > 0 {
		ageVal = sql.NullInt64{Int64: int64(f.AgeMeta.Age), Valid: true}
		if f.AgeMeta.BirthYear > 0 {
			birthYear = sql.NullInt64{Int64: int64(f.AgeMeta.BirthYear), Valid: true}
		}
		if f.AgeMeta.BirthdayMMDD != "" {
			birthdayMMDD = nullStr(f.AgeMeta.BirthdayMMDD)
		}
		if f.AgeMeta.RecordedAt != "" {
			ageRecordedAt = nullStr(f.AgeMeta.RecordedAt)
		} else {
			ageRecordedAt = nullStr(createdAt)
		}
		if f.AgeMeta.IsEstimate {
			ageIsEstimate = sql.NullInt64{Int64: 1, Valid: true}
		}
	}
	args = append(args, ageVal, birthYear, birthdayMMDD, ageRecordedAt, ageIsEstimate)

	return args
}
