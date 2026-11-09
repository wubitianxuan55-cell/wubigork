package app

// P4 持久任务中心·办公批处理（2026-11-10 众神殿 pan_1794234593182 裁决
// action-7，缓冲项按 D3 通过启动）：批量文档转 PDF 进任务调度器——断窗续跑
// （Startup 把 running/queued 重排）、可取消（两段式 stopping→cancelled）、
// 可重试（Retry=re-enqueue）、进度逐文件上报，全部继承 internal/gaea/tasks
// 持久机械（Hephaestus.db），不依赖单回合存活。提交入口=agent 工具
// office_batch（提交即返回任务 id；声明 out_dir=实际产物目录，与 P1a 结构化
// 声明纪律同口径）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/tasks"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// officeBatchConvert 转换 seam（测试注入；生产=docmd.ConvertToPdfFile，ctx
// 仅供测试观测取消——生产转换内部无 ctx 语义，取消由 handler 逐文件间隙判定）。
var officeBatchConvert = func(ctx context.Context, src, out string) error {
	return docmd.ConvertToPdfFile(src, out)
}

// officeBatchPayload office_batch 任务载荷（Submit 方定义）。
type officeBatchPayload struct {
	Files  []string `json:"files"`  // 工作区相对路径
	OutDir string   `json:"outDir"` // 产物目录（工作区相对）
}

// officeBatchTaskHandler 逐文件转 PDF：进度逐文件上报、ctx 可取消、单文件
// 失败不中断批次（Result 逐文件 ok/fail 报告；全败才整任务 failed——诚实口径）。
func (a *App) officeBatchTaskHandler(ctx context.Context, t *tasks.Task, p *tasks.Progress) error {
	var in officeBatchPayload
	if err := json.Unmarshal([]byte(t.Payload), &in); err != nil {
		return fmt.Errorf("载荷无效: %w", err)
	}
	if len(in.Files) == 0 {
		return fmt.Errorf("files 为空")
	}
	outDirAbs := ""
	if strings.TrimSpace(in.OutDir) != "" {
		p2, err := wspath.ResolveRelWithin(gaeaCwd(), in.OutDir)
		if err != nil {
			return fmt.Errorf("非法 outDir（须为工作区相对路径）: %s", in.OutDir)
		}
		outDirAbs = p2
		if err := os.MkdirAll(outDirAbs, 0o755); err != nil {
			return err
		}
	}
	type fileResult struct {
		File   string `json:"file"`
		Out    string `json:"out,omitempty"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	results := make([]fileResult, 0, len(in.Files))
	okCount := 0
	for i, rel := range in.Files {
		// 用户取消：ctx.Err 返回后调度器按两段式收口（running→stopping→cancelled）。
		if err := ctx.Err(); err != nil {
			return err
		}
		src, err := wspath.ResolveRelWithin(gaeaCwd(), rel)
		if err != nil {
			results = append(results, fileResult{File: rel, Status: "fail", Error: "非法工作区路径"})
		} else {
			outPath := strings.TrimSuffix(src, filepath.Ext(src)) + ".pdf"
			if outDirAbs != "" {
				outPath = filepath.Join(outDirAbs, strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))+".pdf")
			}
			if ferr := officeBatchConvert(ctx, src, outPath); ferr != nil {
				results = append(results, fileResult{File: rel, Status: "fail", Error: ferr.Error()})
			} else {
				okCount++
				relOut, _ := filepath.Rel(gaeaCwd(), outPath)
				results = append(results, fileResult{File: rel, Out: filepath.ToSlash(relOut), Status: "ok"})
			}
		}
		if p != nil {
			p.Report((i+1)*100/len(in.Files), fmt.Sprintf("%d/%d 已处理", i+1, len(in.Files)))
		}
	}
	res := map[string]any{"ok": okCount, "fail": len(in.Files) - okCount, "files": results}
	if b, err := json.Marshal(res); err == nil && p != nil {
		p.Result(string(b)) // 结果经 Progress 落库（t.Result 直接赋值不持久化）
	}
	if okCount == 0 {
		return fmt.Errorf("全部 %d 个文件转换失败（详见任务结果逐文件报告）", len(in.Files))
	}
	return nil
}

// officeBatchMaxFiles 单批上限（防误提交全工作区；批处理队列本就为长跑设计，
// 超大批应拆批提交）。
const officeBatchMaxFiles = 200

// submitOfficeBatch office_batch 提交入口：进持久调度器，立即返回任务（不占
// 回合）。space 归一走 gaeaEffectiveSpace（任务中心按空间分账口径）。
func (a *App) submitOfficeBatch(files []string, outDir string) (*tasks.Task, error) {
	m := a.taskMgr()
	if m == nil {
		return nil, fmt.Errorf("任务调度器不可用")
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("files 为空：至少一个待转换文件")
	}
	// 声明纪律（P1a 同口径）：批量产物必须声明落点目录，不接受隐式散落。
	if strings.TrimSpace(outDir) == "" {
		return nil, fmt.Errorf("缺少 out_dir（声明写路径）：批量产物必须声明落点目录")
	}
	if len(files) > officeBatchMaxFiles {
		return nil, fmt.Errorf("单批最多 %d 个文件（当前 %d），请拆批提交", officeBatchMaxFiles, len(files))
	}
	payload := map[string]any{"files": files, "outDir": outDir}
	label := fmt.Sprintf("批量转 PDF（%d 个文件）", len(files))
	return m.SubmitSpace(tasks.KindOfficeBatch, label, payload, gaeaEffectiveSpace())
}
