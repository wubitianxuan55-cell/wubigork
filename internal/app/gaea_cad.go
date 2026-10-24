// gaea_cad.go — 办公「CAD 图纸」卡绑定（一期 P0：只读解析/分幅光栅化/矢量
// 文本提取；「问图纸」复用 GaeaRecognizeImage 零新面）。
//
// 红线（docs/gaea-cad-tool-master-plan-2026-10.md）：
//   - 只读不记账：不触碰引擎/模型路由（对比 visionLocalRuntime 的记账纪律）；
//   - 产物落 <工作区>/.gaea/cad/<内容键>/ 缓存（LRU 收缩），图纸与提取结果
//     不入 SQLite；
//   - 图纸路径护栏与 GaeaReadFileB64 同源：文件对话框所选（pickedFiles
//     白名单）或 .gaea/ 工作目录内（附件/自身产物），其余拒绝。
package app

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gaea/gaea/internal/gaea/cad"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// cadMaxTexts 文本清单截断上限（PoC 真实图 542 条；2000 为大图防护）。
const cadMaxTexts = 2000

// cadCacheKeep 每图纸缓存目录保留上限（超出删最旧——图纸渲染产物可达数 MB，
// 不做无界累积）。
const cadCacheKeep = 8

const (
	cadOpenTimeout  = 2 * time.Minute
	cadQueryTimeout = 90 * time.Second
)

// cadCacheRoot 缓存根目录 seam（测试注入 TempDir，防污染仓库工作目录）。
var cadCacheRoot = func() string { return filepath.Join(gaeaCwd(), ".gaea", "cad") }

// cadPathAllowed 图纸读取护栏：对话框所选或 .gaea/ 工作目录内。
func cadPathAllowed(path string) bool {
	if isPickedFile(path) {
		return true
	}
	root, err := wspath.RealPath(filepath.Join(gaeaCwd(), ".gaea"))
	if err != nil {
		return false
	}
	abs, err := wspath.RealPath(path)
	if err != nil {
		return false
	}
	return wspath.Within(root, abs)
}

type cadCacheMeta struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mtime"`
}

// cadCacheDir 图纸专属缓存目录（内容键=绝对路径|大小|mtime 毫秒；图纸变更
// 即换键重渲）+ LRU 收缩兄弟目录。
func cadCacheDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("读取图纸失败: %w", err)
	}
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", abs, st.Size(), st.ModTime().UnixMilli())))
	dir := filepath.Join(cadCacheRoot(), fmt.Sprintf("%x", sum)[:16])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建缓存目录失败: %w", err)
	}
	meta, err := json.Marshal(cadCacheMeta{
		Source: abs, Name: filepath.Base(path), Size: st.Size(), ModTime: st.ModTime().UnixMilli(),
	})
	if err != nil {
		return "", fmt.Errorf("序列化 CAD 缓存元数据失败: %w", err)
	}
	if err := fileutil.AtomicWrite(filepath.Join(dir, "meta.json"), meta, 0o644); err != nil {
		return "", fmt.Errorf("写入 CAD 缓存元数据失败: %w", err)
	}
	if root, err := os.ReadDir(cadCacheRoot()); err == nil && len(root) > cadCacheKeep {
		type ent struct {
			path string
			mod  int64
		}
		var ents []ent
		for _, e := range root {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(cadCacheRoot(), e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			ents = append(ents, ent{path: p, mod: info.ModTime().UnixMilli()})
		}
		sort.Slice(ents, func(i, j int) bool { return ents[i].mod > ents[j].mod })
		for _, e := range ents[cadCacheKeep:] {
			_ = os.RemoveAll(e.path)
		}
	}
	return dir, nil
}

// cadGuard 打开前三连护栏：类型/白名单/可读。
func cadGuard(path string) error {
	if !cadPathAllowed(path) {
		return fmt.Errorf("仅允许打开文件对话框选择的图纸（或 .gaea 工作目录内的文件）")
	}
	if _, _, err := cad.ValidateInput(path); err != nil {
		return err
	}
	return nil
}

func cadContext(a *App, timeout time.Duration) (context.Context, context.CancelFunc) {
	var parent context.Context
	if a != nil && a.core != nil {
		parent = a.ctx
	}
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}

type cadFileMeta struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type cadOpenOut struct {
	cad.OpenResult
	File cadFileMeta `json:"file"`
}

// GaeaCadOpen 打开图纸并产出摘要+全图框分幅 PNG（DWG/DXF=go-cad 解析渲染；
// PDF=poppler 页图）。返回 JSON：{kind,version,entityCount,skipped,textCount,
// layers,warnings,file,sheets:[{name,box,passRate,texts,png}]}。passRate 是
// 图框质量闸（PoC：真图幅 0.77+，误判巨框 <0.1），前端据此排序供人工选框。
func (a *App) GaeaCadOpen(path string) (string, error) {
	if err := cadGuard(path); err != nil {
		return "", err
	}
	kind, st, err := cad.ValidateInput(path)
	if err != nil {
		return "", err
	}
	dir, err := cadCacheDir(path)
	if err != nil {
		return "", err
	}
	ctx, cancel := cadContext(a, cadOpenTimeout)
	defer cancel()
	var res *cad.OpenResult
	switch kind {
	case cad.KindPDF:
		res, err = cad.OpenPDFContext(ctx, path, dir, 0, cad.PDFMaxPages)
	default:
		res, err = cad.OpenAndRenderContext(ctx, path, dir, 0)
	}
	if err != nil {
		return "", err
	}
	out := cadOpenOut{OpenResult: *res, File: cadFileMeta{Name: filepath.Base(path), Size: st.Size()}}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type cadRasterOut struct {
	PNGPath string `json:"pngPath"`
	DataURL string `json:"dataUrl"`
}

// GaeaCadRaster 取第 sheet 个图框/页（1 基，与 CadOpen 的 sheets 序一致）的
// PNG：width<=0 用打开时的缓存图，>0 按指定宽度重渲（清晰度不足时前端可加
// 宽重取；PDF 恒用打开时的页图，width 忽略）。返回 JSON {pngPath,dataUrl}；
// pngPath 可直接喂 GaeaRecognizeImage 做「问图纸」。
func (a *App) GaeaCadRaster(path string, sheet int, width int) (string, error) {
	if err := cadGuard(path); err != nil {
		return "", err
	}
	if sheet < 1 {
		return "", fmt.Errorf("图框序号从 1 开始：%d", sheet)
	}
	dir, err := cadCacheDir(path)
	if err != nil {
		return "", err
	}
	var pngPath string
	switch cad.Sniff(path) {
	case cad.KindPDF:
		pngPath = filepath.Join(dir, fmt.Sprintf("sheet-%d.png", sheet))
		if _, err := os.Stat(pngPath); err != nil {
			ctx, cancel := cadContext(a, cadOpenTimeout)
			defer cancel()
			if _, err := cad.OpenPDFContext(ctx, path, dir, 0, cad.PDFMaxPages); err != nil {
				return "", err
			}
		}
	default:
		pngPath = filepath.Join(dir, fmt.Sprintf("sheet-%d.png", sheet))
		if width > 0 {
			ctx, cancel := cadContext(a, cadQueryTimeout)
			defer cancel()
			d, err := cad.OpenContext(ctx, path)
			if err != nil {
				return "", err
			}
			if pngPath, err = d.RenderSheet(sheet-1, dir, width); err != nil {
				return "", err
			}
		} else if _, err := os.Stat(pngPath); err != nil {
			// 未开过（或缓存被清）：按默认宽度补渲单框。
			ctx, cancel := cadContext(a, cadQueryTimeout)
			defer cancel()
			d, err := cad.OpenContext(ctx, path)
			if err != nil {
				return "", err
			}
			if pngPath, err = d.RenderSheet(sheet-1, dir, 0); err != nil {
				return "", err
			}
		}
	}
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return "", fmt.Errorf("读取分幅 PNG 失败: %w", err)
	}
	b, err := json.Marshal(cadRasterOut{
		PNGPath: pngPath,
		DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type cadTextsOut struct {
	Total     int        `json:"total"`
	Truncated bool       `json:"truncated"`
	Texts     []cad.Text `json:"texts"`
}

// GaeaCadTexts 矢量文本全量清单（含图层/世界坐标/图框归属；\U+ 转义已解码）。
// PDF 无矢量文本面——用「问图纸」识别页图（一期口径，报错明示）。
func (a *App) GaeaCadTexts(path string) (string, error) {
	if err := cadGuard(path); err != nil {
		return "", err
	}
	if cad.Sniff(path) == cad.KindPDF {
		return "", fmt.Errorf("PDF 图纸暂不支持矢量文本清单（一期仅 DWG/DXF），请用「问图纸」识别页图")
	}
	ctx, cancel := cadContext(a, cadQueryTimeout)
	defer cancel()
	d, err := cad.OpenContext(ctx, path)
	if err != nil {
		return "", err
	}
	all, _ := d.Texts(0)
	texts, truncated := d.Texts(cadMaxTexts)
	out := cadTextsOut{Total: len(all), Truncated: truncated, Texts: texts}
	if out.Texts == nil {
		out.Texts = []cad.Text{}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GaeaCadCount 通用符号统计（算量表数据源）：全图纸块引用计数（天然精确）+
// 邻近文字众数自动命名（置信度随行）+ 图框归属分布 + 图层线长。机制与专业
// 无关；无文字命中的块保留原名待确认表人工命名。PDF 无矢量面，报错引导。
func (a *App) GaeaCadCount(path string) (string, error) {
	if err := cadGuard(path); err != nil {
		return "", err
	}
	if cad.Sniff(path) == cad.KindPDF {
		return "", fmt.Errorf("PDF 图纸暂不支持符号统计（矢量面仅 DWG/DXF），请用「问图纸」识别页图")
	}
	ctx, cancel := cadContext(a, cadQueryTimeout)
	defer cancel()
	d, err := cad.OpenContext(ctx, path)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(d.Count())
	if err != nil {
		return "", err
	}
	return string(b), nil
}
