package app

// 记忆中枢「数据库」绑定（v4.479）：管理本设备保存的聊天记录与上传的文件。
//
//   - 聊天记录 = 各工作区 .gaea/sessions/ 下的 .jsonl 会话档（含 work/play
//     空间分区与 archive/ 归档子目录）；
//   - 上传的文件 = 各工作区 .gaea/uploads/（attach-* 聊天上传附件、paste-*
//     粘贴图片，GaeaSaveAttachmentFile/粘贴链路写入）。
//
// 作用域 = 当前工作区 + 最近工作区（与 GaeaListProjectSessions 同 roots 口径）
// ——记忆中枢是设备级管理面，只看当前工作区会重复「办公记忆按项目过滤」的
// 旧困惑。删除面（会话/附件）带白名单护栏：路径必须落在已知 roots 内。

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// DatabaseProjectStats 一个工作区的数据库存档统计。
type DatabaseProjectStats struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
	// 会话（.jsonl 档，活跃+归档分开计）
	SessionCount  int   `json:"sessionCount"`
	ArchivedCount int   `json:"archivedCount"`
	SessionBytes  int64 `json:"sessionBytes"`
	// 上传文件（.gaea/uploads）
	UploadCount int   `json:"uploadCount"`
	UploadBytes int64 `json:"uploadBytes"`
	// 最近活动（会话/附件 mtime 最大值，UnixMilli；0=无记录）
	LatestMod int64 `json:"latestMod"`
}

// DatabaseOverview 本设备数据库存档总览。
type DatabaseOverview struct {
	Projects      []DatabaseProjectStats `json:"projects"`
	TotalSessions int                    `json:"totalSessions"`
	TotalArchived int                    `json:"totalArchived"`
	TotalUploads  int                    `json:"totalUploads"`
	TotalBytes    int64                  `json:"totalBytes"`
}

// UploadFileRow 一条上传文件（.gaea/uploads 内）。
type UploadFileRow struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	// Kind: "attach"=聊天上传附件 | "paste"=粘贴图片 | "other"
	Kind string `json:"kind"`
}

// databaseRoots 返回数据库存档的作用域 roots：当前工作区 + 最近工作区
// （去重、仍存在且为目录）。与 GaeaListProjectSessions 的 roots 口径一致。
// 去重键=Clean+小写归一：recent-workspaces 历史条目可能同路径两种斜杠风格
// （"C:/AI/x" 与 "C:\AI\x"）甚至盘符大小写不同，原样去重会重复统计。
func databaseRoots() []string {
	roots := []string{gaeaCwd()}
	roots = append(roots, config.LoadRecentWorkspaces()...)
	return dedupeRoots(roots)
}

// dedupeRoots 归一化去重并过滤不存在的目录（保序：首个原始写法胜出，
// 供前端展示/回传）。大小写折叠仅 Windows（Linux 路径大小写敏感，折叠会
// 误并不同目录）。
func dedupeRoots(roots []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, r := range roots {
		if strings.TrimSpace(r) == "" {
			continue
		}
		key := filepath.Clean(r)
		if runtime.GOOS == "windows" {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		if info, err := os.Stat(r); err == nil && info.IsDir() {
			out = append(out, r)
		}
	}
	return out
}

// databaseRootSet 把 roots 转成上传删除护栏用的 .gaea/uploads 白名单目录集。
func databaseUploadDirs() []string {
	var dirs []string
	for _, r := range databaseRoots() {
		dirs = append(dirs, filepath.Join(r, ".gaea", "uploads"))
	}
	return dirs
}

// GaeaDatabaseOverview 返回本设备数据库存档总览：按工作区的会话/归档/上传
// 文件计数与磁盘占用。只 stat 目录项不读内容，总览在记忆中枢每次切换 tab
// 时刷新，成本与 GaeaListProjectSessions 同级。
func (a *App) GaeaDatabaseOverview() DatabaseOverview {
	ov := DatabaseOverview{Projects: []DatabaseProjectStats{}}
	for _, root := range databaseRoots() {
		ps := DatabaseProjectStats{
			Path:    root,
			Name:    filepath.Base(root),
			Current: root == gaeaCwd(),
		}
		sessDir := config.WorkspaceSessionDir(root, "")
		active, archived, bytes, sessLatest := walkSessionFiles(sessDir)
		ps.SessionCount = active
		ps.ArchivedCount = archived
		ps.SessionBytes = bytes
		ups, ubytes, upLatest := statUploadsDir(filepath.Join(root, ".gaea", "uploads"))
		ps.UploadCount = ups
		ps.UploadBytes = ubytes
		if upLatest > sessLatest {
			ps.LatestMod = upLatest
		} else {
			ps.LatestMod = sessLatest
		}
		ov.Projects = append(ov.Projects, ps)
		ov.TotalSessions += active
		ov.TotalArchived += archived
		ov.TotalUploads += ups
		ov.TotalBytes += bytes + ubytes
	}
	sort.SliceStable(ov.Projects, func(i, j int) bool {
		if ov.Projects[i].Current != ov.Projects[j].Current {
			return ov.Projects[i].Current
		}
		return ov.Projects[i].LatestMod > ov.Projects[j].LatestMod
	})
	return ov
}

// GaeaUploadsList 返回指定工作区的上传文件列表（新→旧）。
// root 必须在数据库存档作用域内（当前+最近工作区白名单），防任意目录枚举。
func (a *App) GaeaUploadsList(root string) []UploadFileRow {
	out := []UploadFileRow{}
	root = filepath.Clean(root)
	known := false
	for _, r := range databaseRoots() {
		// 双侧 Clean：recent-workspaces 可能存正斜杠路径（历史 JSON），
		// 前端回传 overview 的 path 原样，比较须归一（否则白名单误拒）。
		if filepath.Clean(r) == root {
			known = true
			break
		}
	}
	if !known {
		return out
	}
	dir := filepath.Join(root, ".gaea", "uploads")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		name := e.Name()
		kind := "other"
		switch {
		case strings.HasPrefix(name, "attach-"):
			kind = "attach"
		case strings.HasPrefix(name, "paste-"):
			kind = "paste"
		}
		out = append(out, UploadFileRow{
			Name:    name,
			Path:    filepath.Join(dir, name),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixMilli(),
			Kind:    kind,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ModTime > out[j].ModTime })
	return out
}

// GaeaDeleteUpload 删除一条上传文件。护栏：路径必须落在数据库存档作用域内
// 某工作区的 .gaea/uploads 目录下（白名单根，防任意路径删除）。
func (a *App) GaeaDeleteUpload(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("路径为空")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("路径解析失败: %w", err)
	}
	for _, dir := range databaseUploadDirs() {
		if wspath.Within(dir, abs) {
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("文件不存在: %s", path)
			}
			return os.Remove(abs)
		}
	}
	return fmt.Errorf("路径不在上传目录白名单内（.gaea/uploads）: %s", path)
}

// walkSessionFiles 递归统计会话目录下的 .jsonl 档：活跃/归档分开计数
// （归档 = 位于 archive/ 目录子树内），返回 (活跃数, 归档数, 总字节, 最近 mtime)。
func walkSessionFiles(sessDir string) (active, archived int, bytes int64, latest int64) {
	_ = filepath.WalkDir(sessDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // 不存在的目录/读错误按 0 计
		}
		if !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		bytes += info.Size()
		if m := info.ModTime().UnixMilli(); m > latest {
			latest = m
		}
		// 归档判定：相对路径上任一段目录名为 archive
		rel, rerr := filepath.Rel(sessDir, p)
		if rerr == nil && strings.Contains(strings.ToLower(filepath.ToSlash(rel)), "/archive/") {
			archived++
		} else {
			active++
		}
		return nil
	})
	return active, archived, bytes, latest
}

// statUploadsDir 统计上传目录：文件数、总字节、最近 mtime（UnixMilli）。
func statUploadsDir(dir string) (count int, bytes int64, latest int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, 0
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		count++
		bytes += info.Size()
		if m := info.ModTime().UnixMilli(); m > latest {
			latest = m
		}
	}
	return count, bytes, latest
}
