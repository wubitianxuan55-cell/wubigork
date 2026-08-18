package app

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ZipDeliverableResult 是会话产物打包结果（工作区相对路径）。
type ZipDeliverableResult struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Entries int    `json:"entries"`
	Bytes   int64  `json:"bytes"`
}

// GaeaZipDeliverables 会话产物一键打包：把本次会话交付的文件（工作区相对路径列表）
// 打成一个 zip 放进 .gaea/exports/，对标 Kimi 工作空间 / WorkBuddy 会话产物打包。
// 安全：只接受工作区内的相对路径（拒绝绝对路径与 .. 穿越），缺失/目录条目静默跳过。
func (a *App) GaeaZipDeliverables(paths []string) (ZipDeliverableResult, error) {
	if len(paths) == 0 {
		return ZipDeliverableResult{}, fmt.Errorf("没有可打包的会话产物")
	}

	// 收集存在的工作区文件（相对路径清洗 + 去重），zip 内保留相对路径结构，
	// 不同目录下的同名文件不会互相覆盖。
	type entry struct{ abs, rel string }
	var files []entry
	seen := map[string]bool{}
	for _, p := range paths {
		clean := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(p, "\\", "/")))
		clean = strings.TrimPrefix(clean, "./")
		if clean == "" || clean == "." || strings.HasPrefix(clean, "../") {
			continue // 拒绝 .. 穿越
		}
		if filepath.IsAbs(clean) || filepath.IsAbs(p) {
			continue // 拒绝绝对路径（产物面板路径均为工作区相对路径）
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		abs := filepath.Join(gaeaCwd(), clean)
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		files = append(files, entry{abs: abs, rel: clean})
	}
	if len(files) == 0 {
		return ZipDeliverableResult{}, fmt.Errorf("会话产物文件都不存在或不可访问")
	}

	exportsDir := filepath.Join(gaeaCwd(), ".gaea", "exports")
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return ZipDeliverableResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	zipName := fmt.Sprintf("gaea-会话产物-%s.zip", stamp)
	zipPath := filepath.Join(exportsDir, zipName)

	fz, err := os.Create(zipPath)
	if err != nil {
		return ZipDeliverableResult{}, err
	}
	zw := zip.NewWriter(fz)
	var total int64
	for _, f := range files {
		info, err := os.Stat(f.abs)
		if err != nil {
			continue
		}
		fh, err := zip.FileInfoHeader(info)
		if err != nil {
			continue
		}
		fh.Name = filepath.ToSlash(f.rel)
		fh.Method = zip.Deflate
		w, err := zw.CreateHeader(fh)
		if err != nil {
			continue
		}
		raw, err := os.ReadFile(f.abs)
		if err != nil {
			continue
		}
		if _, err := w.Write(raw); err != nil {
			continue
		}
		total += int64(len(raw))
	}
	if err := zw.Close(); err != nil {
		fz.Close()
		os.Remove(zipPath)
		return ZipDeliverableResult{}, err
	}
	if err := fz.Close(); err != nil {
		os.Remove(zipPath)
		return ZipDeliverableResult{}, err
	}

	rel, _ := filepath.Rel(gaeaCwd(), zipPath)
	return ZipDeliverableResult{
		Path:    filepath.ToSlash(rel),
		Name:    filepath.Base(zipPath),
		Entries: len(files),
		Bytes:   total,
	}, nil
}
