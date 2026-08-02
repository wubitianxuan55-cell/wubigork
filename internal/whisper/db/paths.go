// Package db - whisper embedded SQLite persistence
// Path: {dataRoot}/hermes.db (whisper agent = Hermes; legacy whisper.db auto-migrated on first open)
package db

import (
	"os"
	"path/filepath"
)

// HermesDBFilename 数据库文件名（轻语 agent = Hermes）
const HermesDBFilename = "hermes.db"

// LegacyDBFilename 旧版数据库文件名（改名迁移源）
const LegacyDBFilename = "whisper.db"

// DatabasePath 返回数据库文件完整路径
func DatabasePath(dataRoot string) string {
	return filepath.Join(dataRoot, HermesDBFilename)
}

// EnsureDataRoot 确保 dataRoot 目录存在
func EnsureDataRoot(dataRoot string) error {
	return os.MkdirAll(dataRoot, 0755)
}
