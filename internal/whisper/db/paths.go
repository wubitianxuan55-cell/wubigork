// Package db — whisper 嵌入式 SQLite 持久层
// 路径：{dataRoot}/whisper.db
package db

import (
	"os"
	"path/filepath"
)

// WhisperDBFilename 数据库文件名
const WhisperDBFilename = "whisper.db"

// DatabasePath 返回数据库文件完整路径
func DatabasePath(dataRoot string) string {
	return filepath.Join(dataRoot, WhisperDBFilename)
}

// EnsureDataRoot 确保 dataRoot 目录存在
func EnsureDataRoot(dataRoot string) error {
	return os.MkdirAll(dataRoot, 0755)
}
