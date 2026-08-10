package app

import (
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/pins"
)

// pinnedView 把固定清单转成资料视图（FileSearchHit），缺失文件跳过。
func pinnedView(cwd string) []FileSearchHit {
	paths, err := pins.Load(cwd)
	if err != nil || len(paths) == 0 {
		return []FileSearchHit{}
	}
	out := make([]FileSearchHit, 0, len(paths))
	for _, rel := range paths {
		abs := filepath.Join(cwd, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		out = append(out, FileSearchHit{
			Path:    rel,
			Name:    filepath.Base(abs),
			IsDir:   false,
			Size:    info.Size(),
			ModTime: info.ModTime().UnixMilli(),
		})
	}
	return out
}

// GaeaPinnedMaterials 返回当前工作区的固定常用资料（P1-②）。
// 固定后的文件在新会话启动时自动带入系统提示词（见 boot 系统提示词组装）。
func (a *App) GaeaPinnedMaterials() []FileSearchHit {
	return pinnedView(gaeaCwd())
}

// GaeaPinMaterial 固定一个工作区相对路径文件，返回更新后的固定清单。
func (a *App) GaeaPinMaterial(rel string) []FileSearchHit {
	if _, err := pins.Add(gaeaCwd(), rel); err != nil {
		return []FileSearchHit{}
	}
	return pinnedView(gaeaCwd())
}

// GaeaUnpinMaterial 取消固定一个文件，返回更新后的固定清单。
func (a *App) GaeaUnpinMaterial(rel string) []FileSearchHit {
	if _, err := pins.Remove(gaeaCwd(), rel); err != nil {
		return []FileSearchHit{}
	}
	return pinnedView(gaeaCwd())
}
