package proposal

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "golang.org/x/image/webp"

	"github.com/gaea/gaea/internal/gaea/wspath"
)

const (
	// AssetURIPrefix 是方案正文里的稳定图片引用前缀。后面跟 64 位 sha256，
	// 不暴露资产实际路径；阅读端经绑定读取时再换成 data URL。
	AssetURIPrefix = "gaea-asset://"

	maxProposalImageBytes = 20 << 20
)

type proposalAsset struct {
	SHA256       string
	MIME         string
	Bytes        int64
	Width        int
	Height       int
	OriginalName string
	Caption      string
}

type preparedBody struct {
	Body    string
	Assets  []proposalAsset
	Missing []string
}

type imageRef struct {
	start       int
	end         int
	targetStart int
	targetEnd   int
	target      string
	caption     string
}

type assetReplacement struct {
	start int
	end   int
	text  string
}

type assetDiskRef struct {
	SHA256 string
	MIME   string
}

// AssetRoot returns the managed image root next to the user-global gaea data.
func AssetRoot(userDir string) string {
	userDir = strings.TrimSpace(userDir)
	if userDir == "" {
		return ""
	}
	return filepath.Join(userDir, "proposal-assets")
}

// OpenWithAssets opens the proposal store and enables managed image assets.
func OpenWithAssets(gdb *sql.DB, userDir string) (*Store, error) {
	st, err := Open(gdb)
	if err != nil {
		return nil, err
	}
	root := AssetRoot(userDir)
	if root == "" {
		return nil, fmt.Errorf("方案图片目录不可用（用户数据目录为空）")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("创建方案图片目录: %w", err)
	}
	st.assetRoot = root
	return st, nil
}

// AssetDataURL resolves a gaea-asset URI (or bare sha256) to a browser data URL.
func (s *Store) AssetDataURL(ref string) (string, error) {
	if err := s.available(); err != nil {
		return "", err
	}
	hash, err := assetHash(ref)
	if err != nil {
		return "", err
	}
	if s.assetRoot == "" {
		return "", fmt.Errorf("方案图片目录不可用")
	}
	var mime string
	var size int64
	err = s.db.QueryRow(`SELECT mime, bytes FROM proposal_assets WHERE sha256=? ORDER BY id LIMIT 1`, hash).
		Scan(&mime, &size)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("方案图片不存在或已删除")
	}
	if err != nil {
		return "", unreadable(err)
	}
	if imageExt(mime) == "" {
		return "", fmt.Errorf("方案图片类型异常: %s", mime)
	}
	if size <= 0 || size > maxProposalImageBytes {
		return "", fmt.Errorf("方案图片大小异常: %d", size)
	}
	path := s.assetPath(hash, mime)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("读取方案图片: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() != size || size > maxProposalImageBytes {
		return "", fmt.Errorf("方案图片已损坏（数据库 %d 字节，磁盘 %d 字节）", size, info.Size())
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("读取方案图片: %w", err)
	}
	raw, readErr := io.ReadAll(io.LimitReader(f, maxProposalImageBytes+1))
	closeErr := f.Close()
	if readErr != nil {
		return "", fmt.Errorf("读取方案图片: %w", readErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("关闭方案图片: %w", closeErr)
	}
	if int64(len(raw)) != size {
		return "", fmt.Errorf("方案图片已损坏（期望 %d 字节，读到 %d 字节）", size, len(raw))
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func (s *Store) prepareBodyAssets(body, baseDir string) (preparedBody, error) {
	out := preparedBody{Body: body}
	refs := findMarkdownImageRefs(body)
	if len(refs) == 0 {
		return out, nil
	}
	baseAbs := ""
	if strings.TrimSpace(baseDir) != "" {
		var err error
		baseAbs, err = filepath.Abs(baseDir)
		if err != nil {
			return out, fmt.Errorf("解析图片来源目录: %w", err)
		}
	}
	var replacements []assetReplacement
	seen := make(map[string]bool)
	missing := make(map[string]bool)
	for _, ref := range refs {
		target := strings.TrimSpace(ref.target)
		if target == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(target), AssetURIPrefix) {
			asset, ok, reason := s.loadStoredAsset(target)
			if !ok {
				missing[shortRef(ref.caption, target, reason)] = true
				continue
			}
			asset.Caption = ref.caption
			asset.OriginalName = "managed-image"
			if !seen[asset.SHA256] {
				seen[asset.SHA256] = true
				out.Assets = append(out.Assets, asset)
			}
			continue
		}
		asset, ok, reason := s.storeImageTarget(target, baseAbs, ref.caption)
		if !ok {
			if strings.EqualFold(reason, "external") {
				continue
			}
			missing[shortRef(ref.caption, target, reason)] = true
			continue
		}
		replacements = append(replacements, assetReplacement{
			start: ref.targetStart, end: ref.targetEnd, text: AssetURIPrefix + asset.SHA256,
		})
		if !seen[asset.SHA256] {
			seen[asset.SHA256] = true
			out.Assets = append(out.Assets, asset)
		}
	}
	if len(replacements) > 0 {
		sort.Slice(replacements, func(i, j int) bool { return replacements[i].start > replacements[j].start })
		for _, rep := range replacements {
			out.Body = out.Body[:rep.start] + rep.text + out.Body[rep.end:]
		}
	}
	for item := range missing {
		out.Missing = append(out.Missing, item)
	}
	sort.Strings(out.Missing)
	return out, nil
}

func (s *Store) loadStoredAsset(ref string) (proposalAsset, bool, string) {
	hash, err := assetHash(ref)
	if err != nil {
		return proposalAsset{}, false, err.Error()
	}
	if s.assetRoot == "" {
		return proposalAsset{}, false, "方案图片目录不可用"
	}
	matches, err := filepath.Glob(filepath.Join(s.assetRoot, "objects", hash[:2], hash+".*"))
	if err != nil || len(matches) == 0 {
		return proposalAsset{}, false, "托管图片文件不存在"
	}
	path := matches[0]
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxProposalImageBytes {
		return proposalAsset{}, false, "托管图片文件异常"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return proposalAsset{}, false, err.Error()
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != hash {
		return proposalAsset{}, false, "托管图片哈希不匹配"
	}
	mime, width, height, err := inspectImage(raw)
	if err != nil {
		return proposalAsset{}, false, err.Error()
	}
	return proposalAsset{
		SHA256: hash, MIME: mime, Bytes: int64(len(raw)), Width: width, Height: height,
		OriginalName: filepath.Base(path),
	}, true, ""
}

func (s *Store) storeImageTarget(target, baseDir, caption string) (proposalAsset, bool, string) {
	if s.assetRoot == "" {
		return proposalAsset{}, false, "方案图片目录不可用"
	}
	lower := strings.ToLower(target)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "blob:") {
		return proposalAsset{}, false, "external"
	}
	var (
		raw      []byte
		origName string
	)
	if strings.HasPrefix(lower, "data:") {
		decoded, err := decodeImageDataURL(target)
		if err != nil {
			return proposalAsset{}, false, err.Error()
		}
		raw = decoded
		origName = "inline-image"
	} else {
		path, err := resolveImageSource(target, baseDir)
		if err != nil {
			return proposalAsset{}, false, err.Error()
		}
		info, err := os.Stat(path)
		if err != nil {
			return proposalAsset{}, false, "文件读不到"
		}
		if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxProposalImageBytes {
			return proposalAsset{}, false, "不是可用的图片文件或超过上限"
		}
		f, err := os.Open(path)
		if err != nil {
			return proposalAsset{}, false, err.Error()
		}
		raw, err = io.ReadAll(io.LimitReader(f, maxProposalImageBytes+1))
		closeErr := f.Close()
		if err != nil {
			return proposalAsset{}, false, err.Error()
		}
		if closeErr != nil {
			return proposalAsset{}, false, closeErr.Error()
		}
		if len(raw) > maxProposalImageBytes {
			return proposalAsset{}, false, "图片超过 20 MB"
		}
		origName = filepath.Base(path)
	}
	mime, width, height, err := inspectImage(raw)
	if err != nil {
		return proposalAsset{}, false, err.Error()
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	if err := s.writeAssetFile(hash, mime, raw); err != nil {
		return proposalAsset{}, false, err.Error()
	}
	return proposalAsset{
		SHA256: hash, MIME: mime, Bytes: int64(len(raw)), Width: width, Height: height,
		OriginalName: origName, Caption: strings.TrimSpace(caption),
	}, true, ""
}

func (s *Store) writeAssetFile(hash, mime string, raw []byte) error {
	path := s.assetPath(hash, mime)
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() == int64(len(raw)) {
		existing, readErr := os.ReadFile(path)
		if readErr == nil {
			sum := sha256.Sum256(existing)
			if hex.EncodeToString(sum[:]) == hash {
				return nil
			}
		}
		_ = os.Remove(path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+hash[:12]+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			_ = os.Remove(tmpName)
			ok = true
			return nil
		}
		return err
	}
	ok = true
	return nil
}

func (s *Store) assetPath(hash, mime string) string {
	ext := imageExt(mime)
	if ext == "" {
		ext = ".img"
	}
	return filepath.Join(s.assetRoot, "objects", hash[:2], hash+ext)
}

func (s *Store) insertAssetsTx(tx *sql.Tx, proposalID, sectionID int64, assets []proposalAsset, now string) error {
	for _, a := range assets {
		_, err := tx.Exec(`INSERT OR IGNORE INTO proposal_assets(
  proposal_id, section_id, sha256, mime, bytes, width, height, original_name, caption, created_at
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			proposalID, sectionID, a.SHA256, a.MIME, a.Bytes, a.Width, a.Height, a.OriginalName, a.Caption, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func assetRefsTx(tx *sql.Tx, proposalID int64) ([]assetDiskRef, error) {
	rows, err := tx.Query(`SELECT DISTINCT sha256, mime FROM proposal_assets WHERE proposal_id=?`, proposalID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []assetDiskRef
	for rows.Next() {
		var ref assetDiskRef
		if err := rows.Scan(&ref.SHA256, &ref.MIME); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

func (s *Store) gcAssetRefs(refs []assetDiskRef) {
	if s.assetRoot == "" || len(refs) == 0 {
		return
	}
	for _, ref := range refs {
		var n int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM proposal_assets WHERE sha256=?`, ref.SHA256).Scan(&n); err != nil || n > 0 {
			continue
		}
		_ = os.Remove(s.assetPath(ref.SHA256, ref.MIME))
	}
}

func decodeImageDataURL(value string) ([]byte, error) {
	comma := strings.IndexByte(value, ',')
	if comma < 0 {
		return nil, fmt.Errorf("data URL 缺少内容")
	}
	meta := strings.ToLower(value[len("data:"):comma])
	if !strings.Contains(meta, ";base64") {
		return nil, fmt.Errorf("只支持 base64 图片 data URL")
	}
	encoded := value[comma+1:]
	encoded = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, encoded)
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > maxProposalImageBytes {
		return nil, fmt.Errorf("图片超过 20 MB")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("图片 data URL 解码失败")
	}
	if len(raw) == 0 || len(raw) > maxProposalImageBytes {
		return nil, fmt.Errorf("图片大小异常")
	}
	return raw, nil
}

func resolveImageSource(target, baseDir string) (string, error) {
	value := strings.TrimSpace(target)
	if strings.HasPrefix(strings.ToLower(value), "file:") {
		u, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("图片 file URL 无效")
		}
		value = u.Path
		if u.Host != "" && u.Host != "localhost" {
			value = "//" + u.Host + u.Path
		}
		if len(value) >= 3 && value[0] == '/' && value[2] == ':' {
			value = value[1:]
		}
	} else {
		if i := strings.IndexAny(value, "?#"); i >= 0 {
			value = value[:i]
		}
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
	}
	path := filepath.FromSlash(value)
	if !filepath.IsAbs(path) {
		if baseDir == "" {
			return "", fmt.Errorf("相对图片缺少来源目录")
		}
		path = filepath.Join(baseDir, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("解析图片路径失败")
	}
	if baseDir == "" || !wspath.Within(baseDir, abs) {
		return "", fmt.Errorf("图片不在来源目录内")
	}
	return abs, nil
}

func inspectImage(raw []byte) (mime string, width, height int, err error) {
	head := raw
	if len(head) > 512 {
		head = head[:512]
	}
	mime = strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(head), ";")[0]))
	if imageExt(mime) == "" {
		return "", 0, 0, fmt.Errorf("不支持的图片格式（仅 PNG/JPEG/GIF/WebP）")
	}
	cfg, _, decErr := image.DecodeConfig(bytes.NewReader(raw))
	if decErr != nil {
		return "", 0, 0, fmt.Errorf("图片内容无效")
	}
	return mime, cfg.Width, cfg.Height, nil
}

func imageExt(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

func assetHash(ref string) (string, error) {
	value := strings.TrimSpace(ref)
	if strings.HasPrefix(strings.ToLower(value), AssetURIPrefix) {
		value = value[len(AssetURIPrefix):]
	}
	if len(value) != sha256.Size*2 {
		return "", fmt.Errorf("方案图片引用无效")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("方案图片引用无效")
	}
	return strings.ToLower(value), nil
}

func findMarkdownImageRefs(markdown string) []imageRef {
	var refs []imageRef
	for offset := 0; offset < len(markdown); {
		start := strings.Index(markdown[offset:], "![")
		if start < 0 {
			break
		}
		start += offset
		altEnd := strings.Index(markdown[start+2:], "](")
		if altEnd < 0 {
			break
		}
		altEnd += start + 2
		targetStart := altEnd + 2
		for targetStart < len(markdown) && (markdown[targetStart] == ' ' || markdown[targetStart] == '\t') {
			targetStart++
		}
		if targetStart >= len(markdown) {
			break
		}
		var (
			target string
			close  int
		)
		if markdown[targetStart] == '<' {
			angleEnd := strings.IndexByte(markdown[targetStart+1:], '>')
			if angleEnd < 0 {
				break
			}
			angleEnd += targetStart + 1
			target = markdown[targetStart+1 : angleEnd]
			close = strings.IndexByte(markdown[angleEnd+1:], ')')
			if close < 0 {
				break
			}
			close += angleEnd + 1
			replaceStart := targetStart + 1
			targetEnd := angleEnd
			refs = append(refs, imageRef{
				start: start, end: close + 1, targetStart: replaceStart, targetEnd: targetEnd,
				target: strings.TrimSpace(target), caption: strings.TrimSpace(markdown[start+2 : altEnd]),
			})
		} else {
			closeRel := strings.IndexByte(markdown[targetStart:], ')')
			if closeRel < 0 {
				break
			}
			close = targetStart + closeRel
			target = markdown[targetStart:close]
			if i := strings.IndexAny(target, " \t\r\n"); i >= 0 {
				target = target[:i]
			}
			refs = append(refs, imageRef{
				start: start, end: close + 1, targetStart: targetStart, targetEnd: targetStart + len(target),
				target: strings.TrimSpace(target), caption: strings.TrimSpace(markdown[start+2 : altEnd]),
			})
		}
		offset = close + 1
	}
	return refs
}

func shortRef(caption, target, reason string) string {
	label := strings.TrimSpace(caption)
	if label == "" {
		label = filepath.Base(strings.Split(target, "?")[0])
	}
	if len([]rune(label)) > 80 {
		label = string([]rune(label)[:80]) + "…"
	}
	if reason == "" {
		return label
	}
	return label + "：" + reason
}
