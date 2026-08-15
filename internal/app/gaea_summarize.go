package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/gaea/largefile"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// GaeaSummaryResult 是资料「摘要」操作的返回：摘要文本 + 元信息。
type GaeaSummaryResult struct {
	Path       string `json:"path"`
	TotalPages int    `json:"totalPages"`
	Chars      int    `json:"chars"`
	Chunks     int    `json:"chunks"`
	Summary    string `json:"summary"`
}

// GaeaSummarizeFile 对工作区资料做分块摘要（map-reduce，走办公功能模型），
// 供资料面板「摘要后引用」一键把摘要插入输入框（对标千问/aily 上传即摘要）。
func (a *App) GaeaSummarizeFile(rel string, focus string) (GaeaSummaryResult, error) {
	path, _ := resolvePreviewPath(rel)
	if path == "" {
		return GaeaSummaryResult{}, fmt.Errorf("文件不存在: %s", rel)
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return GaeaSummaryResult{}, fmt.Errorf("文件不存在: %s", rel)
	}
	if a.client == nil {
		return GaeaSummaryResult{}, fmt.Errorf("模型服务不可用")
	}
	featEng, featModel, _ := a.routeModel("office")
	prov, err := provider.NewLLM("", provider.Config{Name: "office-summary", Model: featModel, Engine: featEng})
	if err != nil {
		return GaeaSummaryResult{}, fmt.Errorf("摘要模型初始化失败: %w", err)
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	res, err := largefile.SummarizeFile(ctx, prov, path, largefile.Options{Focus: focus})
	if err != nil {
		return GaeaSummaryResult{}, err
	}
	return GaeaSummaryResult{
		Path:       filepath.ToSlash(res.Path),
		TotalPages: res.TotalPages,
		Chars:      res.Chars,
		Chunks:     res.Chunks,
		Summary:    res.Summary,
	}, nil
}
