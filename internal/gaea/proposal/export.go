package proposal

// export.go — 整册导出 Markdown。方案库板块的「导出」与 proposal_export 工具走
// 同一段逻辑（2026-11-28 前只有工具面，库页约定里的「导出」在界面上一直缺位）。

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExportResult 导出回执。
type ExportResult struct {
	Title     string
	Path      string // 写入的 Markdown 文件
	AssetsDir string // 托管图片落到的目录（没有图片时为空）
	Sections  int
	Chars     int
	Images    int // 落成文件的图片数
	Missing   int // 托管文件缺失、保留了原引用的图片数
}

// assetRefRe 正文里的托管图片引用。
var assetRefRe = regexp.MustCompile(regexp.QuoteMeta(AssetURIPrefix) + `([0-9a-fA-F]{64})`)

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// Export 把一份方案导回 Markdown 文件：标题按层级写成 # ~ ####，正文原样；正文里
// 的托管图片落成文件，存到输出文件同名的 -assets 目录并改为相对引用。
// outPath 须是 .md / .markdown / .txt。
func (s *Store) Export(nameOrTitle, outPath string) (*ExportResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	switch strings.ToLower(filepath.Ext(outPath)) {
	case ".md", ".markdown", ".txt":
	default:
		return nil, fmt.Errorf("导出文件只收 .md/.markdown/.txt，收到 %s", filepath.Ext(outPath))
	}
	p, secs, err := s.FullText(nameOrTitle)
	if err != nil {
		return nil, err
	}
	res := &ExportResult{Title: p.Title, Path: outPath, Sections: len(secs)}

	// 图片：托管目录 → 输出文件同名的 -assets 目录；文件缺失或写不出来的引用保留原样并计数。
	stem := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))
	assetDir := filepath.Join(filepath.Dir(outPath), stem+"-assets")
	relDir := stem + "-assets"
	var b strings.Builder
	fmt.Fprintf(&b, "> 方案库导出：%s（%s / %s）\n\n", p.Title, orDash(p.ProjectType), orDash(p.Year))
	for _, sec := range secs {
		fmt.Fprintf(&b, "%s %s\n\n", strings.Repeat("#", sec.Level), sec.Title)
		res.Chars += sec.Chars
		body := assetRefRe.ReplaceAllStringFunc(sec.Body, func(ref string) string {
			src, err := s.AssetPath(ref)
			if err != nil {
				res.Missing++
				return ref
			}
			raw, err := os.ReadFile(src)
			if err == nil {
				err = os.MkdirAll(assetDir, 0o755)
			}
			if err == nil {
				err = os.WriteFile(filepath.Join(assetDir, filepath.Base(src)), raw, 0o644)
			}
			if err != nil {
				res.Missing++
				return ref
			}
			res.Images++
			return relDir + "/" + filepath.Base(src)
		})
		if strings.TrimSpace(body) != "" {
			b.WriteString(strings.TrimRight(body, "\n") + "\n\n")
		}
	}
	if dir := filepath.Dir(outPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建输出目录失败: %w", err)
		}
	}
	if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
		return nil, fmt.Errorf("写入导出文件失败: %w", err)
	}
	if res.Images > 0 {
		res.AssetsDir = assetDir
	}
	return res, nil
}
