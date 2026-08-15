package characterlib

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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
	base := portraitFileBase(id)
	path := filepath.Join(dir, base+ext)
	// 纵深防御：清洗后最终路径必须仍落在 portraits 目录内（防路径穿越）。
	if !strings.HasPrefix(path, dir+string(filepath.Separator)) {
		return dataURL
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return dataURL
	}
	return path
}

// portraitFileBase 把角色 ID 清洗为安全文件名基名（T7-2 防路径穿越）：
//   - 含路径分隔符（/ \）、".." 片段或为空 → 用 ID 的 SHA-256 短哈希代替
//     （哈希无分隔符，天然安全且保持唯一性）；
//   - 其余非安全字符（空白/标点/控制符）替换为 _，首尾 . 与 _ 去除；
//   - 全部清洗后为空 → 同样回退哈希。
func portraitFileBase(id string) string {
	clean := strings.TrimSpace(id)
	if clean == "" || strings.ContainsAny(clean, `/\`) || strings.Contains(clean, "..") {
		return portraitIDHash(id)
	}
	var b strings.Builder
	for _, r := range clean {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._")
	if out == "" {
		return portraitIDHash(id)
	}
	return out
}

// portraitIDHash 由角色 ID 生成固定 16 位十六进制哈希（防穿越/空 ID 回退）。
func portraitIDHash(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:8])
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
	// 注意：连接池 SetMaxOpenConns(1)。不能在 rows 未关闭时于循环内执行
	// UPDATE，否则唯一连接被 rows 占用，Exec 会永久等待，启动卡死
	// （chatStore 等后续初始化永远不执行）。先收集再写回。
	type pending struct {
		id       string
		portrait string
	}
	var todo []pending
	for rows.Next() {
		var id, portrait string
		if err := rows.Scan(&id, &portrait); err != nil {
			continue
		}
		todo = append(todo, pending{id: id, portrait: portrait})
	}
	rows.Close()

	migrated := 0
	for _, p := range todo {
		if len(p.portrait) <= maxPortraitDataURL {
			continue
		}
		path := savePortraitFile(s.dataDir, p.id, p.portrait)
		if path == p.portrait {
			continue // 落盘失败
		}
		if _, err := s.db.Exec(`UPDATE characters SET portrait_url=? WHERE id=?`, path, p.id); err != nil {
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
