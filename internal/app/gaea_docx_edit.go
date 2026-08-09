package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gaea/gaea/internal/office/docxedit"
)

// GaeaOfficeEditText 框选即改：按自然语言指令生成选中文本的替换内容
// （办公向提示词：信息不变、措辞严谨、纯文本输出）。
func (a *App) GaeaOfficeEditText(selectedText, instruction string) (map[string]interface{}, error) {
	if a.client == nil {
		return nil, fmt.Errorf("AI 客户端未初始化")
	}
	if selectedText == "" {
		return nil, fmt.Errorf("选中文本为空")
	}
	if instruction == "" {
		return nil, fmt.Errorf("编辑指令为空")
	}

	featEng, featModel, _ := a.routeModel("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	edited, err := a.client.OfficeEditText(ctx, featEng, featModel, selectedText, instruction)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"edited": edited}, nil
}

// GaeaDocxApplyEdit 把替换以修订模式（w:del + w:ins）写入 docx，
// 返回更新后的预览（前端直接重渲染，修订样式可见）。
func (a *App) GaeaDocxApplyEdit(rel, selectedText, replacement string) (PreviewResult, error) {
	if rel == "" {
		return PreviewResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if err := docxedit.ApplyTrackedReplace(path, selectedText, replacement, "gaea AI"); err != nil {
		return PreviewResult{}, err
	}
	return a.GaeaPreview(rel), nil
}

// GaeaDocxAcceptChanges 接受/拒绝 gaea 的待处理修订（accept=true 接受全部，
// false 拒绝全部），返回更新后的预览。
func (a *App) GaeaDocxAcceptChanges(rel string, accept bool) (PreviewResult, error) {
	if rel == "" {
		return PreviewResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		path = filepath.Join(gaeaCwd(), rel)
	}
	if accept {
		if err := docxedit.AcceptChanges(path, "gaea AI"); err != nil {
			return PreviewResult{}, err
		}
	} else {
		if err := docxedit.RejectChanges(path, "gaea AI"); err != nil {
			return PreviewResult{}, err
		}
	}
	return a.GaeaPreview(rel), nil
}
