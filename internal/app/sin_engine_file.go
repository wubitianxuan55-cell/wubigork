package app

// ── 原罪工作区文件读口（升格刀3）──────────────────────────────────────
//
// 根 = 当前书目录（引擎装配时固定）；前端 SinFilePreview/交付物预览消费。
// 复用办公原语（workbench.ReadText/WriteText/ReadPage、GaeaPreview 的绝对
// 路径直通分支）——零 fork、零办公改动；containment 由 wspath.ResolveRelWithin
// fail-closed 保证（.. 穿越/绝对注入拒绝）。

import (
	"fmt"

	"github.com/gaea/gaea/internal/gaea/wspath"
	"github.com/gaea/gaea/internal/gaea/workbench"
)

// sinEngineBookRoot 当前书目录（未装配 → 错误，绝不回落办公工作区）。
func sinEngineBookRoot() (string, error) {
	sinEng.mu.Lock()
	defer sinEng.mu.Unlock()
	if sinEng.bookPath == "" {
		return "", fmt.Errorf("未绑定书工作空间：先在书架选择一本书")
	}
	return sinEng.bookPath, nil
}

// sinResolveBookRel 相对路径解析（拒穿越；返回书目录下绝对路径）。
func sinResolveBookRel(rel string) (string, error) {
	root, err := sinEngineBookRoot()
	if err != nil {
		return "", err
	}
	p, err := wspath.ResolveRelWithin(root, rel)
	if err != nil {
		return "", fmt.Errorf("路径越出书工作区: %s", rel)
	}
	return p, nil
}

// SinWorkspacePreview 书内文件预览（走 GaeaPreview 绝对路径直通分支）。
func (a *App) SinWorkspacePreview(rel string) PreviewResult {
	p, err := sinResolveBookRel(rel)
	if err != nil {
		return PreviewResult{Path: rel, Kind: "error", Error: err.Error()}
	}
	return a.GaeaPreview(p)
}

// SinWorkspaceReadText 书内文件全文读取（编辑器底稿；带 mtime 基线）。
func (a *App) SinWorkspaceReadText(rel string) (workbench.TextFile, error) {
	if _, err := sinResolveBookRel(rel); err != nil {
		return workbench.TextFile{}, err
	}
	root, err := sinEngineBookRoot()
	if err != nil {
		return workbench.TextFile{}, err
	}
	return workbench.ReadText(root, rel)
}

// SinWorkspaceWriteText 书内文件保存（编码/EOL 保持 + mtime 冲突拒绝）。
func (a *App) SinWorkspaceWriteText(rel string, content string, expectedMtimeMs int64) (workbench.WriteResult, error) {
	if _, err := sinResolveBookRel(rel); err != nil {
		return workbench.WriteResult{}, err
	}
	root, err := sinEngineBookRoot()
	if err != nil {
		return workbench.WriteResult{}, err
	}
	return workbench.WriteText(root, rel, content, expectedMtimeMs)
}

// SinWorkspaceReadPage 大文件按行对齐分页读取。
func (a *App) SinWorkspaceReadPage(rel string, offset int64) (workbench.TextPage, error) {
	if _, err := sinResolveBookRel(rel); err != nil {
		return workbench.TextPage{}, err
	}
	root, err := sinEngineBookRoot()
	if err != nil {
		return workbench.TextPage{}, err
	}
	return workbench.ReadPage(root, rel, offset)
}
