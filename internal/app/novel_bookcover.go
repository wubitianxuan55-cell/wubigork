package app

// ── v4.3g 书封生成（设计 docs/gaea-v43-play-deepen-design.md §3/§5.4）──
//
// 复用章节插图（GenerateSceneIllustration）的生成管线：读项目名/世界观/角色
// 构建中文封面 prompt → ai 图片生成（3:4 竖版）→ 下载远端图片 → 落盘
// <项目根>/.gaea/play/exports/cover-<projectID>.png（play 空间，红线：不落
// work 目录）→ 返回封面文件绝对路径。
//
// 接收者说明：App 直接嵌入 *writingState（getPM/chapterAgent 提升）与 *core
// （client/ctx 提升）；writingState.initAgents() 把 w.client 注入 chapter.New，
// 因此 a.client 与 a.chapterAgent 内部 client 是同一 *ai.Client 实例，这里
// 直接经 a.client.GenerateImage 构建书封专用 prompt（不新增 chapter 包方法）。

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/types"
	"github.com/gaea/gaea/internal/util"
)

// coverImageModel 与章节插图同一图片模型（Aurora）。
const coverImageModel = "grok-imagine-image-quality"

// GaeaGenerateBookCover 生成项目书封（3:4），落盘 .gaea/play/exports/cover-<projectID>.png，
// 返回封面文件绝对路径。promptHint 可选补充提示词（如风格/元素）。
func (a *App) GaeaGenerateBookCover(projectID, promptHint string) (string, error) {
	pm := a.getPM()
	if pm == nil {
		return "", fmt.Errorf("请先打开项目")
	}
	// projectID 应对应当前打开的项目（支持传项目目录或目录名；空则用当前项目）。
	if projectID != "" {
		want := filepath.ToSlash(filepath.Clean(projectID))
		got := filepath.ToSlash(filepath.Clean(pm.Dir))
		if want != got && want != filepath.Base(got) {
			return "", fmt.Errorf("项目不匹配:当前打开的项目是 %q", filepath.Base(pm.Dir))
		}
	}

	// 读项目信息：读不到就降级为空串，不阻断生成。
	wv, err := pm.ReadWorldview()
	if err != nil {
		slog.Warn("生成书封:读取世界观失败（降级为空）", "error", err)
		wv = ""
	}
	cf, err := pm.ReadCharacters()
	if err != nil {
		slog.Warn("生成书封:读取角色失败（降级为空）", "error", err)
		cf = nil
	}

	// 构建 3:4 竖版书封 prompt（仿 GenerateSceneIllustration：世界观截断 200）。
	var b strings.Builder
	b.WriteString("小说封面插画。")
	if pm.Meta != nil {
		if pm.Meta.Title != "" {
			b.WriteString("作品名: " + pm.Meta.Title + "。")
		}
		if pm.Meta.Genre != "" {
			b.WriteString("题材: " + pm.Meta.Genre + "。")
		}
	}
	if wv != "" {
		b.WriteString("世界观: " + util.Truncate(wv, 200) + "。")
	}
	if cf != nil && len(cf.Characters) > 0 {
		// 主角优先，最多取 3 名角色的外貌/性格。
		var ordered []types.Character
		for _, c := range cf.Characters {
			if c.RoleType == "protagonist" {
				ordered = append(ordered, c)
			}
		}
		for _, c := range cf.Characters {
			if c.RoleType != "protagonist" {
				ordered = append(ordered, c)
			}
		}
		b.WriteString(" 主角: ")
		for i, c := range ordered {
			if i >= 3 {
				break
			}
			b.WriteString(fmt.Sprintf("%s(%s, %s)", c.Name, c.Appearance, c.Personality))
			if i < 2 && i < len(ordered)-1 {
				b.WriteString("; ")
			}
		}
		b.WriteString("。")
	}
	b.WriteString(" 风格: 数字油画，电影级光影，高细节。")
	b.WriteString(" 竖版书籍封面构图（3:4），主体突出、画面平衡，顶部与底部留白以排版书名与作者名，画面中不直接绘制文字。")
	if promptHint != "" {
		b.WriteString(" 补充要求: " + promptHint + "。")
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// 严格 3:4（768x1024）；后端若不支持该尺寸会报错，透传错误，不做尺寸兜底。
	req := &ai.ImageGenerationRequest{
		Model:  coverImageModel,
		Prompt: b.String(),
		N:      1,
		Size:   "768x1024",
	}
	resp, err := a.client.GenerateImage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("生成书封失败: %w", err)
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("未生成图片")
	}

	// 云端/远端 URL → 下载字节；个别后端直接给 b64 时解码兜底。
	var data []byte
	switch {
	case resp.Data[0].URL != "":
		data, err = downloadCoverImage(ctx, resp.Data[0].URL)
		if err != nil {
			return "", fmt.Errorf("下载封面图片失败: %w", err)
		}
	case resp.Data[0].B64JSON != "":
		data, err = base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
		if err != nil {
			return "", fmt.Errorf("解码封面图片失败: %w", err)
		}
	default:
		return "", fmt.Errorf("生成结果不含图片数据")
	}

	// play 空间 exports（<项目根>/.gaea/play/exports），红线：路径含 play，不落 work。
	playExports := filepath.Join(pm.Dir, ".gaea", "play", "exports")
	if err := os.MkdirAll(playExports, 0o755); err != nil {
		return "", fmt.Errorf("创建导出目录失败: %w", err)
	}
	id := projectID
	if id == "" {
		id = filepath.Base(pm.Dir)
	}
	outPath := filepath.Join(playExports, "cover-"+sanitizeCoverID(id)+".png")
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return "", fmt.Errorf("写入封面文件失败: %w", err)
	}
	abs, err := filepath.Abs(outPath)
	if err != nil {
		abs = outPath
	}
	slog.Info("书封已生成", "project", pm.Meta.Title, "path", abs)
	// T0 图像域试点：书封产物登记（play/novel/media.generate，失败只 warn）。
	asset := imageHubAsset{Kind: ImageHubAssetKindImage, Path: abs, MIME: "image/png"}
	if err := recordImageHubGeneratedAsset(gaeaCwd(), "play", "novel", "", coverImageModel,
		b.String(),
		map[string]interface{}{"project_id": id, "size": "768x1024", "n": 1},
		asset, []string{playExports}); err != nil {
		slog.Warn("书封产物登记失败（不影响生成）", "path", abs, "error", err)
	}
	return abs, nil
}

// downloadCoverImage 下载远端图片字节（30s 超时，上限 32MB）。
func downloadCoverImage(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造下载请求失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// sanitizeCoverID 清洗 projectID 为安全的文件名段：去掉路径只取末段、
// 替换 Windows 非法文件名字符、去尾部点/空格、防空与保留设备名。
// 注意取末段用纯字符串切分而非 filepath.Base——Windows 上 filepath.Base 会把
// "a:b*c?d" 中的 "a:" 当作盘符卷丢弃（实测丢失前缀），本项目 ID 是普通字符串。
func sanitizeCoverID(projectID string) string {
	seg := strings.ReplaceAll(projectID, "\\", "/")
	if i := strings.LastIndexByte(seg, '/'); i >= 0 {
		seg = seg[i+1:]
	}
	id := strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, seg)
	id = strings.TrimRight(id, ". ")
	lower := strings.ToLower(id)
	if id == "" {
		return "project"
	}
	switch lower {
	case "con", "prn", "aux", "nul":
		return "project"
	}
	if len(lower) == 4 && (lower[:3] == "com" || lower[:3] == "lpt") && lower[3] >= '1' && lower[3] <= '9' {
		return "project"
	}
	return id
}
