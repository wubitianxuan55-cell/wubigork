// Package backup — gaea 数据可迁移（P4-3，2026-08-14）。
//
// 个人使用收口：一键备份/恢复用户数据（Hephaestus.db 记忆/知识/成本/语义向量、
// whisper_data 轻语/办公/角色库/聊天、配置、sessions），zip + manifest。
// 设计要点：
//   - SQLite 用 VACUUM INTO 生成一致性快照（运行中备份安全，WAL 自动合并，不复制 -wal/-shm）；
//   - 恢复分两步：Restore 解压到 staging + 写 pending 标记；Startup 时 applyPendingRestore()
//     在打开任何数据库前应用（先备份当前数据到 .restore-before-<ts>，再移动 staging 内容）；
//   - 恢复前自动备份当前数据，失败可回滚；zip 条目防 zip-slip。
package backup

import (
	"archive/zip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动
)

// ─── 数据源 ────────────────────────────────────────────────────────

// Source 描述一个备份条目：ZipRel 是 zip 内相对路径，Abs 是源绝对路径。
type Source struct {
	ZipRel string
	Abs    string
}

// Plan 一次备份的完整清单（含应跳过的文件）。
type Plan struct {
	// Root 是数据根目录（备份 zip 内的相对路径基于它）。
	Root string
	// Sources 是待打包条目（目录会递归）。
	Sources []Source
	// Skip 是相对 Root 的跳过前缀（如 *.log、*-wal、*-shm）。
	Skip []string
}

// HomeConfigRel 是 home 下应用配置在 zip 内的相对名（与数据根分离）。
const HomeConfigRel = "home-config/gaea_config.json"

// NewPlan 构造备份计划：数据根 + 显式条目 + 跳过规则。
func NewPlan(root string, entries []Source, skip []string) *Plan {
	return &Plan{Root: root, Sources: entries, Skip: skip}
}

// shouldSkip 判断相对路径是否命中跳过规则（匹配任意路径段）。
// 命中条件：段完全相等 / 段以规则开头 / 段以规则结尾（如 "-wal" 匹配 "chat.db-wal"）。
func (p *Plan) shouldSkip(rel string) bool {
	for _, s := range p.Skip {
		if s == "" {
			continue
		}
		for _, seg := range strings.Split(rel, "/") {
			if strings.EqualFold(seg, s) || strings.HasPrefix(seg, s) || strings.HasSuffix(seg, s) {
				return true
			}
		}
	}
	return false
}

// isSQLite 按扩展名判断是否 SQLite 数据库（走快照）。
func isSQLite(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	for _, name := range []string{"hephaestus.db", "hermes.db", "whisper.db", "office.db", "chat.db", "characterlib.db"} {
		if base == name {
			return true
		}
	}
	return strings.HasSuffix(base, ".db")
}

// ─── manifest ──────────────────────────────────────────────────────

// Manifest 备份清单元数据（zip 内 manifest.json）。
type Manifest struct {
	App        string    `json:"app"`
	Version    string    `json:"version"`     // gaea 版本
	CreatedAt  time.Time `json:"created_at"`  // 备份时间
	DataRoot   string    `json:"data_root"`   // 备份时的数据根（信息用）
	EntryCount int       `json:"entry_count"` // 文件条目数
	TotalBytes int64     `json:"total_bytes"` // 文件总大小
}

// ValidateManifest 校验 manifest 是否为 gaea 备份。
func ValidateManifest(m Manifest) error {
	if m.App != "gaea" {
		return fmt.Errorf("不是 gaea 备份（app=%q）", m.App)
	}
	if m.Version == "" {
		return errors.New("备份缺少版本信息")
	}
	return nil
}

// ─── 打包 ──────────────────────────────────────────────────────────

// Create 把计划打包为 zip，返回 manifest 与 zip 路径。
func (p *Plan) Create(zipPath, appVersion string) (Manifest, error) {
	m := Manifest{App: "gaea", Version: appVersion, CreatedAt: time.Now(), DataRoot: p.Root}

	tmpDir, err := os.MkdirTemp("", "gaea-backup-*")
	if err != nil {
		return m, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	type fileEntry struct {
		rel string
		abs string
	}
	var files []fileEntry
	for _, s := range p.Sources {
		info, err := os.Stat(s.Abs)
		if err != nil {
			continue // 源不存在则跳过（首次运行无库等）
		}
		if info.IsDir() {
			err := filepath.WalkDir(s.Abs, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				rel, rerr := filepath.Rel(p.Root, path)
				if rerr != nil {
					return rerr
				}
				rel = filepath.ToSlash(rel)
				if p.shouldSkip(rel) {
					return nil
				}
				if isSQLite(rel) {
					snap := filepath.Join(tmpDir, fmt.Sprintf("snap-%d.db", len(files)))
					if serr := snapshotSQLite(path, snap); serr == nil {
						files = append(files, fileEntry{rel: rel, abs: snap})
						return nil
					}
					// 快照失败回退原样复制
				}
				files = append(files, fileEntry{rel: rel, abs: path})
				return nil
			})
			if err != nil {
				return m, fmt.Errorf("扫描 %s 失败: %w", s.Abs, err)
			}
		} else {
			rel := s.ZipRel
			if rel == "" {
				rel, _ = filepath.Rel(p.Root, s.Abs)
			}
			rel = filepath.ToSlash(rel)
			if isSQLite(rel) {
				snap := filepath.Join(tmpDir, fmt.Sprintf("snap-single-%d.db", len(files)))
				if serr := snapshotSQLite(s.Abs, snap); serr == nil {
					files = append(files, fileEntry{rel: rel, abs: snap})
				} else {
					files = append(files, fileEntry{rel: rel, abs: s.Abs})
				}
			} else {
				files = append(files, fileEntry{rel: rel, abs: s.Abs})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })

	out, err := os.Create(zipPath)
	if err != nil {
		return m, fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)

	m.EntryCount = len(files)
	var total int64
	for _, f := range files {
		if fi, err := os.Stat(f.abs); err == nil {
			total += fi.Size()
		}
	}
	m.TotalBytes = total
	mf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return m, err
	}
	mh, err := zw.Create("manifest.json")
	if err != nil {
		return m, err
	}
	if _, err := mh.Write(mf); err != nil {
		return m, err
	}

	for _, f := range files {
		if err := addFileToZip(zw, f.rel, f.abs); err != nil {
			return m, fmt.Errorf("打包 %s 失败: %w", f.rel, err)
		}
	}

	if err := zw.Close(); err != nil {
		return m, fmt.Errorf("关闭 zip 失败: %w", err)
	}
	if err := out.Close(); err != nil {
		return m, err
	}
	return m, nil
}

func addFileToZip(zw *zip.Writer, rel, abs string) error {
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	hdr.Name = rel
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	f, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// snapshotSQLite 用 VACUUM INTO 生成一致快照（源保持原样，WAL 自动合并）。
func snapshotSQLite(src, dst string) error {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(src)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	stmt := "VACUUM INTO '" + strings.ReplaceAll(filepath.ToSlash(dst), "'", "''") + "'"
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}
	return nil
}

// ─── 校验与读取 ────────────────────────────────────────────────────

// ReadManifest 从 zip 读取并校验 manifest。
func ReadManifest(zipPath string) (Manifest, error) {
	var m Manifest
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return m, fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return m, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return m, err
		}
		if err := json.Unmarshal(data, &m); err != nil {
			return m, fmt.Errorf("解析 manifest 失败: %w", err)
		}
		if err := ValidateManifest(m); err != nil {
			return m, err
		}
		return m, nil
	}
	return m, errors.New("备份缺少 manifest.json")
}

// safeZipRel 校验 zip 内路径安全（防 zip-slip）：必须相对、不含 ..
func safeZipRel(name string) (string, error) {
	name = filepath.ToSlash(name)
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return "", fmt.Errorf("非法绝对路径: %s", name)
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "..\\") {
		return "", fmt.Errorf("非法路径穿越: %s", name)
	}
	return clean, nil
}

// Extract 把 zip 解压到 destDir（校验 manifest + 防穿越），返回 manifest。
func Extract(zipPath, destDir string) (Manifest, error) {
	m, err := ReadManifest(zipPath)
	if err != nil {
		return m, err
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return m, err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			continue
		}
		rel, err := safeZipRel(f.Name)
		if err != nil {
			return m, err
		}
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return m, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return m, err
		}
		rc, err := f.Open()
		if err != nil {
			return m, err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return m, err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return m, err
		}
		out.Close()
		rc.Close()
	}
	return m, nil
}

// ─── 恢复（pending 应用）──────────────────────────────────────────

// PendingFile 是 DataRoot 下的 pending 标记文件名（相对 DataRoot）。
const PendingFile = ".restore-pending.json"

// PendingState 记录一次待应用恢复。
type PendingState struct {
	StageDir  string    `json:"stage_dir"`
	ZipName   string    `json:"zip_name"`
	CreatedAt time.Time `json:"created_at"`
	DataRoot  string    `json:"data_root"`
	HomeDir   string    `json:"home_dir"`
}

// WritePending 写 pending 标记（DataRoot/.restore-pending.json）。
func WritePending(state PendingState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(state.DataRoot, PendingFile)
	if err := os.MkdirAll(state.DataRoot, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadPending 读取 pending 标记；不存在返回 (nil, nil)。
func ReadPending(dataRoot string) (*PendingState, error) {
	path := filepath.Join(dataRoot, PendingFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st PendingState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("解析 pending 标记失败: %w", err)
	}
	return &st, nil
}

// ClearPending 删除 pending 标记与 staging 目录。
func ClearPending(dataRoot string) error {
	st, err := ReadPending(dataRoot)
	if err != nil {
		return err
	}
	if st != nil && st.StageDir != "" {
		_ = os.RemoveAll(st.StageDir)
	}
	return os.Remove(filepath.Join(dataRoot, PendingFile))
}

// ApplyResult 一次恢复应用的结果（写回 .restore-result.json 供前端提示）。
type ApplyResult struct {
	Applied     bool      `json:"applied"`
	ZipName     string    `json:"zip_name,omitempty"`
	BeforeDir   string    `json:"before_dir,omitempty"` // 恢复前数据备份目录
	AppliedAt   time.Time `json:"applied_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	AppDataRoot string    `json:"app_data_root"`
}

// ApplyPending 应用待恢复数据（Startup 早期、打开任何数据库前调用）。
// 流程：读 pending → 校验 staging → 把当前数据整体移动为 .restore-before-<ts>
// → 把 staging 内容移动到位 → 清理 pending → 写结果。失败保留 pending 以便重试。
func ApplyPending(dataRoot, homeDir string) (ApplyResult, error) {
	res := ApplyResult{AppDataRoot: dataRoot}
	st, err := ReadPending(dataRoot)
	if err != nil {
		res.Error = err.Error()
		return res, err
	}
	if st == nil {
		return res, nil // 无待应用
	}
	res.ZipName = st.ZipName
	if st.StageDir == "" {
		res.Error = "pending 标记缺少 staging 目录"
		return res, errors.New(res.Error)
	}
	stageInfo, err := os.Stat(st.StageDir)
	if err != nil || !stageInfo.IsDir() {
		res.Error = fmt.Sprintf("staging 目录不可用: %v", err)
		return res, errors.New(res.Error)
	}

	ts := time.Now().Format("20060102-150405")
	beforeDir := filepath.Join(dataRoot, ".restore-before-"+ts)
	if err := os.MkdirAll(beforeDir, 0o755); err != nil {
		res.Error = err.Error()
		return res, err
	}

	entries, err := os.ReadDir(st.StageDir)
	if err != nil {
		res.Error = err.Error()
		return res, err
	}
	for _, e := range entries {
		name := e.Name()
		if name == "manifest.json" {
			continue
		}
		src := filepath.Join(st.StageDir, name)
		dst := filepath.Join(dataRoot, name)
		before := filepath.Join(beforeDir, name)
		if _, err := os.Stat(dst); err == nil {
			if err := os.Rename(dst, before); err != nil {
				res.Error = fmt.Sprintf("移动当前数据 %s 失败: %v", name, err)
				return res, err
			}
		}
		if err := os.Rename(src, dst); err != nil {
			if err2 := copyTree(src, dst); err2 != nil {
				res.Error = fmt.Sprintf("应用 %s 失败: %v", name, err)
				return res, err
			}
			_ = os.RemoveAll(src)
		}
	}

	homeCfgStage := filepath.Join(st.StageDir, "home-config")
	if fi, err := os.Stat(homeCfgStage); err == nil && fi.IsDir() {
		entries2, _ := os.ReadDir(homeCfgStage)
		for _, e := range entries2 {
			src := filepath.Join(homeCfgStage, e.Name())
			dst := filepath.Join(homeDir, e.Name())
			before := filepath.Join(beforeDir, "home-config-"+e.Name())
			if _, err := os.Stat(dst); err == nil {
				_ = os.Rename(dst, before)
			}
			if err := copyFile(src, dst); err != nil {
				res.Error = fmt.Sprintf("恢复 home 配置 %s 失败: %v", e.Name(), err)
				return res, err
			}
		}
	}

	res.Applied = true
	res.BeforeDir = beforeDir
	res.AppliedAt = time.Now()
	_ = os.RemoveAll(st.StageDir)
	_ = os.Remove(filepath.Join(dataRoot, PendingFile))
	resultData, _ := json.MarshalIndent(res, "", "  ")
	_ = os.WriteFile(filepath.Join(dataRoot, ".restore-result.json"), resultData, 0o644)
	return res, nil
}

// copyTree 目录递归复制。
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ─── 工具 ──────────────────────────────────────────────────────────

// SHA256 计算文件哈希（用于备份校验展示）。
func SHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
