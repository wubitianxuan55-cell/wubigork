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
// 因此 a.clientRef() 与 a.chapterAgent 内部 client 是同一 *ai.Client 实例，这里
// 直接经 a.clientRef().GenerateImage 构建书封专用 prompt（不新增 chapter 包方法）。

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

// 书封生效模型统一由 effectiveSceneIllustrationModel 解析（审计 AP7-05：
// 登记与请求必须同源，不得各自写死常量）——此处不再留本地别名常量。

// bookCoverStyles 题材→风格层关键词表（oh-story story-cover 题材风格库蒸馏，
// 按 gaea 题材档位 GENRE_OPTIONS 对齐）：项目题材子串命中取首条，全不命中用
// 通用档。只给画面/色彩/光效关键词，不写死构图——构图由三层模板统一承担。
var bookCoverStyles = []struct{ key, style string }{
	{"玄幻", "东方玄幻：云海仙山远景，中景持剑主角衣袂翻飞，金与玄青主色，逆光金边勾勒，气势恢宏"},
	{"仙侠", "仙侠古风：亭台烟霞远景，中景白衣人物御风，青白主色一抹朱红，柔光云雾，飘逸出尘"},
	{"武侠", "水墨武侠：远山孤城剪影，中景持刀人物侧立，墨色留白配一点冷金，硬光刀锋反光，江湖肃杀"},
	{"奇幻", "西方奇幻：魔幻城堡与异月远景，中景持法杖人物，紫蓝撞金主色，魔力光粒，史诗魔幻"},
	{"都市", "现代都市：霓虹街景与天际线中景，前景人物立姿回眸，冷暖对比色，电影感侧光，写实细腻"},
	{"现实", "现实题材：生活街巷暖光中景，前景朴素人物，低饱和暖色，自然柔光，生活流质感"},
	{"历史", "历史古风：宫阙城墙远景，中景古装人物，绛红与鎏金主色，烛光宫灯暖调，厚重庄严"},
	{"军事", "军事战争：战场硝烟远景，中景军人剪影，钢灰与橄榄绿主色，冷调硬光，紧张压迫"},
	{"游戏", "虚拟游戏风：数据流光远景，中景装备人物，荧光蓝紫主色，HUD 光效，赛博电竞感"},
	{"体育", "热血体育：赛场远景，中景动态人物瞬间，高饱和撞色，顶光汗珠反光，动感张力"},
	{"科幻", "科幻未来：星舰废土或星云远景，中景装甲/机甲人物，冷蓝配橙主色，体积光，宏大冷峻"},
	{"悬疑", "悬疑推理：雨夜街灯或密闭空间中景，前景人物半侧影，深灰藏蓝主色一束冷光，阴影切割，克制紧张"},
	{"灵异", "灵异惊悚：老宅浓雾远景，中景持灯人物背影，墨绿与惨白主色，底光鬼影，阴冷压抑"},
	{"二次元", "日系轻小说风：天空教室或花海背景，前景大半身人物特写构图，明亮糖果色，均匀柔光，赛璐璐上色"},
	{"短篇", "极简人像：单色渐变背景，前景人物半身剪影，低饱和双主色，一束侧光，大量留白"},
}

// bookCoverStyleFor 返回题材风格层文案：按题材串子串命中表内首条（表序即
// 优先序），无命中回落通用档。
func bookCoverStyleFor(genre string) string {
	for _, s := range bookCoverStyles {
		if strings.Contains(genre, s.key) {
			return s.style
		}
	}
	return "与题材气质相符的插画风格"
}

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
	// 三层提示词（oh-story story-cover 蒸馏）：画面层（主角/世界观）→ 风格层
	// （题材风格表）→ 文字排版层（安全区留白；本地生图后端中文渲染不可靠，
	// 不要求画面内绘制文字，书名作者名由前端封面叠加排版）。
	var b strings.Builder
	genre := ""
	b.WriteString("小说封面插画。")
	if pm.Meta != nil {
		if pm.Meta.Title != "" {
			b.WriteString("作品名: " + pm.Meta.Title + "。")
		}
		if pm.Meta.Genre != "" {
			genre = pm.Meta.Genre
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
	b.WriteString(" 风格: " + bookCoverStyleFor(genre) + "；数字油画质感，电影级光影，高细节。")
	b.WriteString(" 竖版书籍封面构图（3:4），主体突出、画面平衡：前景人物-中景场景-远景氛围三层排布，画面顶部约六分之一与底部窄条保持干净留白（书名与作者名的排版安全区，主体元素避开四边），画面中不直接绘制文字。")
	if promptHint != "" {
		b.WriteString(" 补充要求: " + promptHint + "。")
	}

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// 严格 3:4（768x1024）；后端若不支持该尺寸会报错，透传错误，不做尺寸兜底。
	// 生效模型（审计 AP7-05）：ImageModel 配置优先，未配置回落 Aurora 档——
	// 登记侧必须与这里下发的模型同源。
	coverModel := effectiveSceneIllustrationModel(a.cfg)
	// S1.5-B play 内容护栏（审计 2026-10-02 AP2-04）：书封此前直接
	// clientRef().GenerateImage，绕过 applyImageSafeMode 的三个生产注入点
	// （gaea_tools.go / image_handler.go 统一入口两处），play 空间「小说」书封
	// 提示词不受 image_safe_mode 约束（同板块章节插图、原罪插图都受）。
	//
	// 为什么是「同处补 applyImageSafeMode」而不是改走 mediaState.generateImageInternal
	// （审计给的首选方案，理由必须写明，勿日后误改）：
	//  1) 统一入口顺带接管 size 策略（非 comfyui/herdsman/glm 后端清空 size）、
	//     落盘目录（saveDir）与并发槽（beginImageGen）——书封的严格 3:4 契约
	//     （TestGaeaGenerateBookCoverSuccessB64 断言 Size=="768x1024"）、
	//     <项目根>/.gaea/play/exports 落点、单条台账登记都会随之改变，属真实
	//     行为回归而非「把不安全行为当期望的用例」，不能拿它换护栏；
	//  2) 书封是单图原语，play 护栏里与生图相关的只有 image_safe_mode 一项
	//     （persona_lock / temperature_max / max_output_tokens 是文本生成参数）。
	// 口径与统一入口逐字对齐：安全段只进请求，台账仍记原始提示词
	// （image_handler.go 的 item.Prompt=fullPrompt 同款）；关闭安全模式时提示词
	// 逐字节不变（applyImageSafeMode 对 safe=false 原样返回）。
	safePrompt := applyImageSafeMode(b.String(), playGuardrails().ImageSafeMode)
	req := &ai.ImageGenerationRequest{
		Model:  coverModel,
		Prompt: safePrompt,
		N:      1,
		Size:   "768x1024",
	}
	resp, err := a.clientRef().GenerateImage(ctx, req)
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
	// 后端/模型记**生效值**（审计 AP7-05）：请求由 a.clientRef() 按全局
	// ImageBackend 发出，模型是上面解析出的 coverModel。
	asset := imageHubAsset{Kind: ImageHubAssetKindImage, Path: abs, MIME: "image/png"}
	if err := recordImageHubGeneratedAsset(gaeaCwd(), "play", "novel", a.cfg.ImageBackend, coverModel,
		b.String(),
		map[string]interface{}{"project_id": id, "size": "768x1024", "n": 1},
		asset, []string{playExports}, ""); err != nil {
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
