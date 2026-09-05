package app

// ── T1 章节配图落盘 + 图像域登记（设计 docs/gaea-image-domain-t0-contract-design-2026-09.md
//    §6：GenerateSceneIllustration 由「仅返回 URL」升级为「落盘 play exports + 登记」）──
//
// 落点与书封同口径：<项目根>/.gaea/play/exports/scene-<章>-<时间戳>.png（play 红线）。
// 返回 URL 不变；落盘/登记失败只 warn，不阻断既有预览路径。

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/project"
)

// saveSceneIllustrationToPlayExports 保存章节配图到项目 play exports，返回绝对路径。
// 复用书封的下载/解码口径（URL 下载 32MB 上限；b64 解码兜底）。
func saveSceneIllustrationToPlayExports(pm *project.Manager, chapterNum int, resp *ai.ImageGenerationResponse) (string, error) {
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("未生成图片")
	}
	var data []byte
	var err error
	switch {
	case resp.Data[0].URL != "":
		data, err = downloadCoverImage(context.Background(), resp.Data[0].URL)
		if err != nil {
			return "", fmt.Errorf("下载配图失败: %w", err)
		}
	case resp.Data[0].B64JSON != "":
		data, err = base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
		if err != nil {
			return "", fmt.Errorf("解码配图失败: %w", err)
		}
	default:
		return "", fmt.Errorf("生成结果不含图片数据")
	}
	playExports := filepath.Join(pm.Dir, ".gaea", "play", "exports")
	if err := os.MkdirAll(playExports, 0o755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	name := fmt.Sprintf("scene-%d-%s.png", chapterNum, time.Now().Format("20060102-150405"))
	outPath := filepath.Join(playExports, name)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return "", fmt.Errorf("写入配图文件失败: %w", err)
	}
	abs, err := filepath.Abs(outPath)
	if err != nil {
		abs = outPath
	}
	return abs, nil
}
