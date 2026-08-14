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
// 注意带点前缀：实际 home 文件是 ~/.gaea_config.json，恢复时 e.Name() 直接拼到 homeDir，
// 若这里写成 gaea_config.json 会恢复到错误的文件名（#2 根因）。
const HomeConfigRel = "home-config/.gaea_config.json"

// NewPlan 构造备份计划：数据根 + 显式条目 + 跳过规则。
func NewPlan(root string, entries []Source, skip []string) *Plan {
	return &Plan{Root: root, Sources: entries, Skip: skip}
}

// shouldSkip 判断相对路径是否命中跳过规则（匹配任意路径段）。
// 命中条件：#15 精确化——段完全相等，或段以 ".db-wal"/".db-shm" 等精确后缀结尾
// （避免 "-wal" 前缀/后缀匹配误伤 my-wal.md、gaea.log.backup 等正常文件）。
func (p *Plan) shouldSkip(rel string) bool {
	for _, s := range p.Skip {
		if s == "" {
			continue
		}
		for _, seg := range strings.Split(rel, "/") {
			if strings.EqualFold(seg, s) {
				return true
			}
			// 精确后缀（如 .db-wal / .db-shm）：仅当该段以特定后缀结尾
			if strings.HasSuffix(seg, ".db-wal") || strings.HasSuffix(seg, ".db-shm") {
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
	Warnings   []string  `json:"warnings,omitempty"` // 备份不完整告警（如快照失败回退）
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
					// 快照失败：先尝试 checkpoint 后复制（保证 WAL 已合并），仍失败才原样复制并告警
					if checkpointThenCopy(path, snap) {
						files = append(files, fileEntry{rel: rel, abs: snap})
						m.Warnings = append(m.Warnings, "快照失败已用 checkpoint 复制回退: "+rel)
						return nil
					}
					m.Warnings = append(m.Warnings, "快照与 checkpoint 均失败，可能含未合并 WAL 数据: "+rel)
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
				} else if checkpointThenCopy(s.Abs, snap) {
					files = append(files, fileEntry{rel: rel, abs: snap})
					m.Warnings = append(m.Warnings, "快照失败已用 checkpoint 复制回退: "+rel)
				} else {
					m.Warnings = append(m.Warnings, "快照与 checkpoint 均失败，可能含未合并 WAL 数据: "+rel)
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
// 常驻连接（busy_timeout=5000 的 WAL 模式）可能在写入提交期间持锁，只读连接必须
// 带 busy_timeout 等待（否则立刻 SQLITE_BUSY），并重试一次。
func snapshotSQLite(src, dst string) error {
	dsn := "file:" + filepath.ToSlash(src) + "?mode=ro&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt := "VACUUM INTO '" + strings.ReplaceAll(filepath.ToSlash(dst), "'", "''") + "'"
	if _, err := db.Exec(stmt); err != nil {
		// 重试一次（等锁释放）
		time.Sleep(200 * time.Millisecond)
		if _, err2 := db.Exec(stmt); err2 != nil {
			return fmt.Errorf("vacuum into: %w", err2)
		}
	}
	return nil
}

// checkpointThenCopy 回退快照：先对源库执行 wal_checkpoint(TRUNCATE) 让 WAL 合并进主文件，
// 再复制主文件。用于 VACUUM INTO 失败（如磁盘紧张）时保证已提交数据不丢失。
// 返回是否执行了 checkpoint（false 表示源不是可打开的 SQLite 或 checkpoint 失败，走原样复制）。
func checkpointThenCopy(src, dst string) bool {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(src)+"?_busy_timeout=5000")
	if err != nil {
		return false
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return false
	}
	return copyFile(src, dst) == nil
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
	if vol := filepath.VolumeName(clean); vol != "" {
		return "", fmt.Errorf("非法盘符路径: %s", name)
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
	// #13：两阶段解压——先建全部目录（含父目录），再写文件；同名冲突文件覆盖目录条目，
	// 不依赖 zip 内条目顺序。
	var dirs []string
	type fileOut struct {
		target string
		f      *zip.File
	}
	var files []fileOut
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
			dirs = append(dirs, target)
			continue
		}
		dirs = append(dirs, filepath.Dir(target))
		files = append(files, fileOut{target: target, f: f})
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return m, err
		}
	}
	for _, fo := range files {
		rc, err := fo.f.Open()
		if err != nil {
			return m, err
		}
		out, err := os.OpenFile(fo.target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
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

// WritePending 写 pending 标记（DataRoot/.restore-pending.json），原子写（#12）。
func WritePending(state PendingState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(state.DataRoot, PendingFile)
	if err := os.MkdirAll(state.DataRoot, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

// ClearPending 删除 pending 标记与 staging 目录，并清理无主孤儿 staging（#5）。
func ClearPending(dataRoot string) error {
	st, err := ReadPending(dataRoot)
	if err != nil {
		return err
	}
	if st != nil && st.StageDir != "" {
		_ = os.RemoveAll(st.StageDir)
	}
	_ = os.Remove(filepath.Join(dataRoot, PendingFile))
	// 清理任何残留的 .restore-stage-* 孤儿目录（无 pending 引用）
	entries, _ := os.ReadDir(dataRoot)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".restore-stage-") {
			if st == nil || filepath.Base(st.StageDir) != name {
				_ = os.RemoveAll(filepath.Join(dataRoot, name))
			}
		}
	}
	return nil
}

// RollbackBefore 把恢复前的数据备份目录（.restore-before）整体移回数据根（#7）。
// 用于恢复失败/取消后用户选择回滚到恢复前状态。返回是否执行了回滚。
func RollbackBefore(dataRoot string) (bool, error) {
	beforeDir := filepath.Join(dataRoot, ".restore-before")
	info, err := os.Stat(beforeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.IsDir() {
		return false, errors.New(".restore-before 不是目录")
	}
	entries, err := os.ReadDir(beforeDir)
	if err != nil {
		return false, err
	}
	moved := 0
	for _, e := range entries {
		src := filepath.Join(beforeDir, e.Name())
		dst := filepath.Join(dataRoot, e.Name())
		// 目标已存在（恢复后的新数据）则先移开，避免覆盖
		if _, err := os.Stat(dst); err == nil {
			keep := filepath.Join(dataRoot, ".restore-new-keep-"+e.Name())
			_ = os.RemoveAll(keep)
			if err := os.Rename(dst, keep); err != nil {
				continue
			}
		}
		if err := os.Rename(src, dst); err != nil {
			if err2 := copyTree(src, dst); err2 != nil {
				continue
			}
			_ = os.RemoveAll(src)
		}
		moved++
	}
	if moved > 0 {
		_ = os.RemoveAll(beforeDir)
	}
	return moved > 0, nil
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
func ApplyPending(dataRoot, homeDir string) (res ApplyResult, err error) {
	res = ApplyResult{AppDataRoot: dataRoot}
	// 无论成功失败都写 .restore-result.json（#4：失败路径对前端可见）
	defer func() {
		if res.Error != "" || res.Applied {
			if data, e := json.MarshalIndent(res, "", "  "); e == nil {
				_ = os.WriteFile(filepath.Join(dataRoot, ".restore-result.json"), data, 0o644)
			}
		}
	}()

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

	// 幂等 before 目录：部分失败重试时复用同一目录，避免把已应用数据反复搬入新目录（#1）。
	// #11：应用成功后保留最近 2 份（见 res.Applied 分支的 pruneBeforeDirs）。
	beforeDir := filepath.Join(dataRoot, ".restore-before")
	if err := os.MkdirAll(beforeDir, 0o755); err != nil {
		res.Error = err.Error()
		return res, err
	}

	entries, err := os.ReadDir(st.StageDir)
	if err != nil {
		res.Error = err.Error()
		return res, err
	}

	// 阶段 1：把当前数据全部移入 before（不依赖 src，重试时已移过的 dst 不存在则跳过，幂等可重入）。
	// 排除 manifest.json 与 home-config（后者单独处理，避免被主循环 rename 走——#2）。
	for _, e := range entries {
		name := e.Name()
		if name == "manifest.json" || name == "home-config" {
			continue
		}
		dst := filepath.Join(dataRoot, name)
		before := filepath.Join(beforeDir, name)
		if _, err := os.Stat(dst); err == nil {
			if _, err := os.Stat(before); err == nil {
				_ = os.RemoveAll(before) // 上次重试可能留下半成品，先清
			}
			if err := os.Rename(dst, before); err != nil {
				res.Error = fmt.Sprintf("移动当前数据 %s 失败: %v", name, err)
				return res, err
			}
		}
	}

	// 阶段 2：把 staging 内容移入数据根。src 缺失 = 该条目上次已应用成功，跳过而非报错（#1 重试幂等）。
	for _, e := range entries {
		name := e.Name()
		if name == "manifest.json" || name == "home-config" {
			continue
		}
		src := filepath.Join(st.StageDir, name)
		if _, err := os.Stat(src); err != nil {
			continue // 已应用，跳过
		}
		dst := filepath.Join(dataRoot, name)
		if err := os.Rename(src, dst); err != nil {
			if err2 := copyTree(src, dst); err2 != nil {
				res.Error = fmt.Sprintf("应用 %s 失败: %v", name, err)
				return res, err
			}
			_ = os.RemoveAll(src)
		}
	}

	// home 配置单独恢复（#2：staging/home-config 仍在，未被主循环消费）
	// #14：homeDir 参数为空时回退到 pending 记录的值（调用方未传时仍可恢复）
	if homeDir == "" && st.HomeDir != "" {
		homeDir = st.HomeDir
	}
	if homeDir != "" {
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
	}

	res.Applied = true
	res.BeforeDir = beforeDir
	res.AppliedAt = time.Now()
	_ = os.RemoveAll(st.StageDir)
	_ = os.Remove(filepath.Join(dataRoot, PendingFile))
	pruneBeforeDirs(dataRoot, 2) // #11：保留最近 2 份恢复前备份，清理更早的
	return res, nil
}

// pruneBeforeDirs 保留最近 keep 份 .restore-before-* 目录（按 mtime），清理更早的。
func pruneBeforeDirs(dataRoot string, keep int) {
	pattern := filepath.Join(dataRoot, ".restore-before*")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) <= keep {
		return
	}
	sort.Slice(matches, func(i, j int) bool {
		mi, _ := os.Stat(matches[i])
		mj, _ := os.Stat(matches[j])
		if mi == nil || mj == nil {
			return false
		}
		return mi.ModTime().After(mj.ModTime())
	})
	for _, m := range matches[keep:] {
		_ = os.RemoveAll(m)
	}
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
