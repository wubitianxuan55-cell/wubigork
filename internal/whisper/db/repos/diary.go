// Package repos — diary 日记仓库
// 100% 对齐 ackem src/main/db/repos/diary.ts
package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/whisper/db"
)

// SaveDiaryToDB 保存日记（按日期 UPSERT）
func SaveDiaryToDB(dataRoot, date, content, metaJSON string) error {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return fmt.Errorf("数据库不可用: %w", openErr)
	}

	updatedAt := time.Now().Format(time.RFC3339)

	var meta sql.NullString
	if metaJSON != "" {
		meta = nullStr(metaJSON)
	}

	_, err := sqlDB.Exec(
		`INSERT INTO diary(date, content, meta_json, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(date) DO UPDATE SET
		   content = excluded.content,
		   meta_json = excluded.meta_json,
		   updated_at = excluded.updated_at`,
		date, content, meta, updatedAt,
	)
	return err
}

// LoadDiaryFromDB 按日期加载日记
func LoadDiaryFromDB(dataRoot, date string) (string, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return "", fmt.Errorf("数据库不可用: %w", openErr)
	}

	var content string
	err := sqlDB.QueryRow(
		"SELECT content FROM diary WHERE date = ?", date,
	).Scan(&content)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return content, err
}

// ListDiaryDatesFromDB 列出所有日记日期
func ListDiaryDatesFromDB(dataRoot string) ([]string, error) {
	sqlDB, openErr := db.GetDatabase(dataRoot)
	if openErr != nil {
		return nil, fmt.Errorf("数据库不可用: %w", openErr)
	}

	rows, err := sqlDB.Query("SELECT date FROM diary ORDER BY date DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			continue
		}
		dates = append(dates, d)
	}
	return dates, nil
}
