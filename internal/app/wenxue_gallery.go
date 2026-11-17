package app

// wenxue_gallery.go — 文学板块「插图」标签页数据面（2026-11-18）。
//
// 图片来源（与 agent 能力同源）：
//   - image_gen 产出：书目录 .gaea/uploads（媒体根分档缝锚小说目录）——
//     封面/场景插图/角色立绘；
//   - 粘贴图：WenxueSavePastedImage 同落 .gaea/uploads；
//   - 用户自放图片：工作空间其他目录的图片文件（如实收集）。
//
// 前端取图经既有 WenxuePreview（image 类返回 DataURL），本绑定只做清单。

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// wenxueGalleryMaxImages 单次清单上限：图册防巨目录一次扫爆（mtime 倒序
// 截断——最新的图永远在场，老图按需清理工作空间）。
const wenxueGalleryMaxImages = 400

// wenxueImageExts 图册收集的图片扩展名（与 office isImagePath 同域）。
var wenxueImageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".bmp": true,
}

// WenxueImageItem 图册单项。
type WenxueImageItem struct {
	Rel     string `json:"rel"`    // 工作空间相对路径（WenxuePreview 取图键）
	Name    string `json:"name"`   // 文件名
	Source  string `json:"source"` // uploads（image_gen/粘贴产出）| workspace（其他目录）
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"` // unix 秒（前端排序展示）
}

// WenxueGalleryList 图文工作空间图片清单：递归收集图片扩展名文件，mtime
// 倒序、封顶 wenxueGalleryMaxImages。未选书（=图文根本身）时同样收集（各书
// .gaea/uploads 均在内）；路径越界防护（Rel 经 wenxueWorkspaceRelToRoot 归一，
// 越界条目跳过）。.git/node_modules 跳过；.gaea 只下钻 uploads（其余缓存目录
// 与图册无关）。
func (a *App) WenxueGalleryList() ([]WenxueImageItem, error) {
	ws := wenxueWorkspace()
	out := make([]WenxueImageItem, 0, 32)
	err := filepath.WalkDir(ws, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil // 单条目失败跳过（清单是尽力面，不因坏符号链炸整表）
		}
		name := d.Name()
		relToWs, rerr := filepath.Rel(ws, p)
		if rerr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(relToWs)
		// .gaea 段判定按路径段而非根前缀——工作空间可能是图文根（各书嵌套其
		// 下），rel 形如「书名/.gaea/uploads/…」，根前缀判定会漏（实机测试逮住）。
		inGaeaNonUploads := false
		segs := strings.Split(relSlash, "/")
		for i, seg := range segs {
			if seg == ".gaea" && i+1 < len(segs) {
				// 裸 .gaea 目录（访问时子目录未知）不标——必须放行下钻到
				// uploads；非 uploads 子目录（cache 等）在此标死。
				if segs[i+1] != "uploads" {
					inGaeaNonUploads = true
				}
			}
		}
		if d.IsDir() {
			switch name {
			case ".git", "node_modules":
				return filepath.SkipDir
			}
			// .gaea 非 uploads 子树（cache/工作日志等）整枝跳过——与图册无关。
			if inGaeaNonUploads {
				return filepath.SkipDir
			}
			return nil
		}
		if inGaeaNonUploads || strings.HasPrefix(name, ".") {
			return nil
		}
		if !wenxueImageExts[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		rel := wenxueWorkspaceRelToRoot(p)
		if rel == "" {
			return nil // 越界（软链逃逸等）：跳过，不外泄
		}
		out = append(out, WenxueImageItem{
			Rel:     filepath.ToSlash(rel),
			Name:    name,
			Source:  gallerySourceOf(relSlash),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModTime != out[j].ModTime {
			return out[i].ModTime > out[j].ModTime
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > wenxueGalleryMaxImages {
		out = out[:wenxueGalleryMaxImages]
	}
	return out, nil
}

// gallerySourceOf 来源分类：任一层的 .gaea/uploads = image_gen/粘贴产出
// （「生成」），其余目录 = 工作空间其他图片（「其他」）。按段匹配——工作
// 空间可能是图文根（各书嵌套其下），前缀匹配会漏。
func gallerySourceOf(relSlash string) string {
	if strings.Contains("/"+relSlash+"/", "/.gaea/uploads/") {
		return "uploads"
	}
	return "workspace"
}
