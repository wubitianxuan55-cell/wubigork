package characterlib

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
)

// portraitFileDir 返回角色库剧照文件目录。
func portraitFileDir(dataDir string) string {
	return filepath.Join(dataDir, "portraits")
}

// savePortraitFile 把 data URL 剧照落盘为文件并返回文件路径；
// 非 data URL（远程 URL / 已是文件路径）原样返回。落盘失败时回退原值。
func savePortraitFile(dataDir, id, dataURL string) string {
	if !strings.HasPrefix(dataURL, "data:") {
		return dataURL
	}
	comma := strings.Index(dataURL, ",")
	if comma < 0 {
		return dataURL
	}
	meta := dataURL[5:comma]
	ext := ".png"
	switch {
	case strings.Contains(meta, "jpeg") || strings.Contains(meta, "jpg"):
		ext = ".jpg"
	case strings.Contains(meta, "webp"):
		ext = ".webp"
	case strings.Contains(meta, "gif"):
		ext = ".gif"
	}
	data, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return dataURL
	}
	dir := portraitFileDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dataURL
	}
	path := filepath.Join(dir, id+ext)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return dataURL
	}
	return path
}

// MigratePortraitsToFiles 把库内超大 base64 剧照迁移为文件（启动时调用）。
// 返回迁移条数；幂等——已迁移（路径）或远程 URL 的行跳过。
func (s *Store) MigratePortraitsToFiles() int {
	if s == nil || s.db == nil {
		return 0
	}
	rows, err := s.db.Query(`SELECT id, portrait_url FROM characters WHERE portrait_url LIKE 'data:%'`)
	if err != nil {
		return 0
	}
	defer rows.Close()

	migrated := 0
	for rows.Next() {
		var id, portrait string
		if err := rows.Scan(&id, &portrait); err != nil {
			continue
		}
		if len(portrait) <= maxPortraitDataURL {
			continue
		}
		path := savePortraitFile(s.dataDir, id, portrait)
		if path == portrait {
			continue // 落盘失败
		}
		if _, err := s.db.Exec(`UPDATE characters SET portrait_url=? WHERE id=?`, path, id); err != nil {
			continue
		}
		migrated++
	}
	return migrated
}

// DataDir 返回角色库数据目录（供启动迁移等使用）。
func (s *Store) DataDir() string {
	if s == nil {
		return ""
	}
	return s.dataDir
}

// PortraitsDir 返回剧照文件目录（供前端路径展示/调试）。
func (s *Store) PortraitsDir() string {
	if s == nil {
		return ""
	}
	return portraitFileDir(s.dataDir)
}

// PortraitFilePath 返回某角色剧照文件路径（不存在返回空串）。
func (s *Store) PortraitFilePath(id string) string {
	if s == nil || s.db == nil {
		return ""
	}
	var p string
	if err := s.db.QueryRow(`SELECT portrait_url FROM characters WHERE id=?`, id).Scan(&p); err != nil {
		return ""
	}
	if strings.HasPrefix(p, "data:") || strings.HasPrefix(p, "http") {
		return ""
	}
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}
