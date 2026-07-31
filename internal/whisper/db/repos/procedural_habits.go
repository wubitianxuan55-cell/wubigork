// Package repos — procedural_habits 程序化习惯仓库
// 100% 对齐 ackem src/main/db/repos/proceduralHabits.ts
package repos

import (
	"fmt"

	"github.com/gaea/gaea/internal/whisper/db"
)

// HabitLine 习惯行
type HabitLine struct {
	TS   string `json:"ts"`
	Text string `json:"text"`
}

// AppendHabitToDB 追加一条习惯
func AppendHabitToDB(dataRoot, text, ts string) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	_, err := sqlDB.Exec(
		"INSERT INTO procedural_habits(ts, text) VALUES (?, ?)",
		ts, text,
	)
	return err
}

// LoadHabitsFromDB 加载所有习惯
func LoadHabitsFromDB(dataRoot string) ([]HabitLine, error) {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return nil, fmt.Errorf("数据库不可用")
	}

	rows, err := sqlDB.Query("SELECT ts, text FROM procedural_habits ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []HabitLine
	for rows.Next() {
		var h HabitLine
		if err := rows.Scan(&h.TS, &h.Text); err != nil {
			continue
		}
		lines = append(lines, h)
	}
	return lines, nil
}

// ReplaceHabitsInDB 全量替换习惯
func ReplaceHabitsInDB(dataRoot string, lines []HabitLine) error {
	sqlDB := db.GetDatabase(dataRoot)
	if sqlDB == nil {
		return fmt.Errorf("数据库不可用")
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM procedural_habits"); err != nil {
		return err
	}

	for _, h := range lines {
		if _, err := tx.Exec(
			"INSERT INTO procedural_habits(ts, text) VALUES (?, ?)",
			h.TS, h.Text,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
