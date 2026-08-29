package characterlib

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// portraitFileDir 返回角色库剧照文件目录。
func portraitFileDir(dataDir string) string {
	return filepath.Join(dataDir, "portraits")
}

// savePortraitFile 把 data URL 剧照落盘为文件并返回文件路径；
// 非 data URL（已是文件路径）原样返回。落盘失败时回退原值。
func savePortraitFile(dataDir, id, dataURL string) string {
	return saveImageFile(dataDir, portraitFileBase(id), dataURL)
}

// saveRemotePortrait 把远程剧照 URL（如 xAI 临时生成图，会过期）下载到本地
// portraits 目录并返回本地路径；下载失败返回原 URL（保存不阻塞，启动迁移会重试）。
func saveRemotePortrait(dataDir, id, remoteURL string) string {
	return saveRemoteImage(dataDir, portraitFileBase(id), remoteURL)
}

// saveImageFile 把 data URL 图片落盘为 portraits 目录下的文件（base 为清洗后
// 的文件名基名，如角色 ID 或 "ID_ref_0"）；非 data URL 原样返回，落盘失败回退原值。
func saveImageFile(dataDir, base, dataURL string) string {
	if !strings.HasPrefix(dataURL, "data:") {
		return dataURL
	}
	comma := strings.Index(dataURL, ",")
	if comma < 0 {
		return dataURL
	}
	data, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return dataURL
	}
	return writePortraitBytes(dataDir, base, extFromDataURL(dataURL[:comma]), data)
}

// saveRemoteImage 把远程图片 URL 下载到本地 portraits 目录（base 为清洗后的
// 文件名基名）；下载失败返回原 URL。
func saveRemoteImage(dataDir, base, remoteURL string) string {
	if !strings.HasPrefix(remoteURL, "http://") && !strings.HasPrefix(remoteURL, "https://") {
		return remoteURL
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(remoteURL)
	if err != nil {
		return remoteURL
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return remoteURL
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20)) // 20MB 上限
	if err != nil || len(body) == 0 {
		return remoteURL
	}
	ext := extFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		ext = extFromPath(remoteURL)
	}
	if ext == "" {
		ext = ".png"
	}
	if path := writePortraitBytes(dataDir, base, ext, body); path != "" {
		return path
	}
	return remoteURL
}

// localizeImageList 把图片列表（data URL / 远程 URL）逐项本地化为
// portraits 目录下的文件路径（basePrefix 如 "ID_ref" / "ID_gallery"，逐项加下标），
// 已本地化（文件路径）或下载失败的原样保留；空串项剔除。
// 与剧照同策略：库里不存巨型 base64，防撑爆 Wails IPC。
func localizeImageList(dataDir, basePrefix string, items []string) []string {
	out := make([]string, 0, len(items))
	for i, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		base := basePrefix + "_" + strconv.Itoa(i)
		it = saveRemoteImage(dataDir, base, it)
		it = saveImageFile(dataDir, base, it)
		out = append(out, it)
	}
	return out
}

// writePortraitBytes 把图片字节写入 portraits 目录（base 为清洗后的文件名基名，
// 防穿越），成功返回完整路径，失败返回空串。
func writePortraitBytes(dataDir, base, ext string, data []byte) string {
	dir := portraitFileDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	path := filepath.Join(dir, base+ext)
	// 纵深防御：清洗后最终路径必须仍落在 portraits 目录内（防路径穿越）。
	if !strings.HasPrefix(path, dir+string(filepath.Separator)) {
		return ""
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ""
	}
	return path
}

// extFromDataURL 从 data URL 的 meta 段推断扩展名。
func extFromDataURL(meta string) string {
	switch {
	case strings.Contains(meta, "jpeg") || strings.Contains(meta, "jpg"):
		return ".jpg"
	case strings.Contains(meta, "webp"):
		return ".webp"
	case strings.Contains(meta, "gif"):
		return ".gif"
	default:
		return ".png"
	}
}

// extFromContentType 从 HTTP Content-Type 推断图片扩展名。
func extFromContentType(ct string) string {
	ct = strings.ToLower(strings.TrimSpace(strings.SplitN(ct, ";", 2)[0]))
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

// extFromPath 从 URL 路径推断扩展名（Content-Type 缺失时兜底）。
func extFromPath(rawURL string) string {
	lower := strings.ToLower(rawURL)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		if strings.Contains(lower, ext) {
			if ext == ".jpeg" {
				return ".jpg"
			}
			return ext
		}
	}
	return ""
}

// MigrateRemotePortraits 把库内远程剧照 URL（http/https，多为会过期的
// xAI 临时图）下载到本地 portraits 目录并回写路径。幂等：非 http 行跳过。
func (s *Store) MigrateRemotePortraits() int {
	if s == nil || s.db == nil {
		return 0
	}
	rows, err := s.db.Query(
		`SELECT id, portrait_url FROM characters WHERE portrait_url LIKE 'http://%' OR portrait_url LIKE 'https://%'`)
	if err != nil {
		return 0
	}
	type pending struct {
		id, url string
	}
	var todo []pending
	for rows.Next() {
		var id, url string
		if err := rows.Scan(&id, &url); err != nil {
			continue
		}
		todo = append(todo, pending{id: id, url: url})
	}
	rows.Close()

	migrated := 0
	for _, p := range todo {
		path := saveRemotePortrait(s.dataDir, p.id, p.url)
		if path == p.url {
			continue // 下载失败，保持原值（下次启动重试）
		}
		if _, err := s.db.Exec(`UPDATE characters SET portrait_url=? WHERE id=?`, path, p.id); err != nil {
			continue
		}
		migrated++
	}
	return migrated
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
