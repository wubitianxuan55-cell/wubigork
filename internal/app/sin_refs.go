package app

// ── 原罪插图 × 角色库参考图（v4.258，「继续」刀：人物一致性锚定）──
//
// 两段式落地，互为补偿：
//   1. 文本锚点（所有后端都吃）：提示词没写到角色外观时，用角色库设定补一句；
//   2. 图像参考槽（仅 ComfyUI / Herdsman 的图生图路径）：把角色立绘/参考图
//      作为一致性参考——与绘梦 T2 角色参考槽同一条链路（mode=img2img +
//      refImages/refMethod），不新增第二条生成实现。
//
// 诚实纪律：
//   - 后端不支持参考图 → 不硬塞（会整单报错），跳过并在返回值里给原因；
//   - 参考图生成失败 → 自动退回纯文本重试一次（插图不该因参考图而失败），
//     并在返回值里标记 fallback；
//   - 只在提示词点名该角色（或故事里只选了一个角色）时使用其参考图，
//     避免多人参考图互相污染。

import (
	"encoding/base64"
	"log/slog"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/characterlib"
)

const (
	// sinRefAppearanceRunes 文本锚点里角色外观描述的最大长度（提示词预算）。
	sinRefAppearanceRunes = 120
	// sinRefDenoise 参考图 img2img 的重绘幅度：太低会锁死构图（场景出不来），
	// 太高会丢掉人物特征；0.65 与绘梦 T2 参考槽默认一致。
	sinRefDenoise = 0.65
)

// sinRefPlan 图像后端 × 模型 → 参考槽可用性（纯函数，便于测试矩阵）。
// 返回 mode（img2img=走图生图）/ refMethod / 是否可用 / 不可用原因。
func sinRefPlan(backend, model string) (mode string, refMethod string, ok bool, reason string) {
	switch backend {
	case "comfyui":
		// 与 ai/image_comfyui.go 的图生图工作流支持面一致（其余模型禁止静默降级）
		if model == "z-image-turbo" || model == "krea2" || strings.HasPrefix(model, "krea2") {
			return "img2img", "img2img", true, ""
		}
		return "", "", false, "ComfyUI 当前模型不支持图生图参考（支持 krea2 / z-image-turbo）"
	case "herdsman":
		// herdsman 参考槽只走 /images/img2img（文生图端点明确拒绝参考图）
		return "img2img", "img2img", true, ""
	default:
		return "", "", false, "当前图像后端不支持参考图（参考槽仅 ComfyUI / Herdsman 图生图可用）"
	}
}

// sinPickRefCharacters 选出本次插图要锚定的角色：
//   - 提示词点名（名字出现在提示词里）的角色优先；
//   - 提示词没点名时，若本故事只选了一个角色则用它（单角色故事不必强行点名）；
//   - 多角色且都没点名 → 不锚定（避免参考图互串）。
func sinPickRefCharacters(cast []*characterlib.Character, prompt string) []*characterlib.Character {
	var named []*characterlib.Character
	for _, c := range cast {
		if c == nil || strings.TrimSpace(c.Name) == "" {
			continue
		}
		if strings.Contains(prompt, c.Name) {
			named = append(named, c)
		}
	}
	if len(named) > 0 {
		return named
	}
	if len(cast) == 1 && cast[0] != nil && strings.TrimSpace(cast[0].Name) != "" {
		return cast[:1]
	}
	return nil
}

// sinCastAppearanceAnchor 角色外观锚点文本（外观优先，其次身材；都空则空串）。
func sinCastAppearanceAnchor(c *characterlib.Character) string {
	if c == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if v := strings.TrimSpace(c.Appearance); v != "" {
		parts = append(parts, v)
	}
	if v := strings.TrimSpace(c.Figure); v != "" {
		parts = append(parts, v)
	}
	if len(parts) == 0 {
		return ""
	}
	return truncateRunes(strings.Join(parts, "；"), sinRefAppearanceRunes)
}

// sinAugmentPromptWithCast 文本锚点补强（对所有后端生效）：把选中角色的外观
// 追加进画面提示词——已有的就不重复，返回补强后的提示词与补进去的角色名。
func sinAugmentPromptWithCast(prompt string, picked []*characterlib.Character) (string, []string) {
	out := prompt
	var added []string
	for _, c := range picked {
		anchor := sinCastAppearanceAnchor(c)
		if anchor == "" {
			continue
		}
		// 提示词已经写到同一段外观描述（前 20 字命中）就不再重复
		head := truncateRunes(anchor, 20)
		if strings.Contains(out, anchor) || (head != "" && strings.Contains(out, head)) {
			continue
		}
		out = strings.TrimRight(out, "。 　") + "。人物锚点（" + c.Name + "）：" + anchor
		added = append(added, c.Name)
	}
	return out, added
}

// sinCharacterRefPath 取角色的一致性参考图路径：参考图优先，其次剧照。
func sinCharacterRefPath(c *characterlib.Character) string {
	if c == nil {
		return ""
	}
	for _, p := range c.ReferenceImages {
		if strings.TrimSpace(p) != "" {
			return p
		}
	}
	return strings.TrimSpace(c.PortraitURL)
}

// sinRefDataURL 把参考图转成后端可用的 data URL：
//   - data: 开头原样返回；
//   - 本地路径读文件转 data URL（与前端 GaeaAttachmentDataURL 同口径）；
//   - 远端 URL 跳过（不额外扩展网络面），文件缺失跳过。
func sinRefDataURL(path string) (string, bool) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", false
	}
	if strings.HasPrefix(p, "data:image/") {
		return p, true
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return "", false
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		slog.Warn("原罪参考图读取失败（跳过该参考）", "path", p, "error", err)
		return "", false
	}
	ext := strings.ToLower(pathExt(p))
	mime := "image/png"
	switch ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw), true
}

// sinRefPick 参考槽解析结果：可用的参考图（data URL）与同序角色名，
// charID 为「台账归属」——唯一一个真正带上参考图的角色 id。
type sinRefPick struct {
	images []string
	names  []string
	charID string
}

// sinResolveRefImages 把候选角色解析成可用的参考图：参考图优先、其次剧照；
// 远端 URL 与缺失文件按 sinRefDataURL 口径跳过。台账归属取第一个真正带图的
// 角色——多角色里只有一人有图时不能记成 picked[0]（那会把登记记到别人身上）。
func sinResolveRefImages(picked []*characterlib.Character) sinRefPick {
	var out sinRefPick
	for _, c := range picked {
		path := sinCharacterRefPath(c)
		if path == "" {
			continue
		}
		dataURL, ok := sinRefDataURL(path)
		if !ok {
			continue
		}
		out.images = append(out.images, dataURL)
		out.names = append(out.names, c.Name)
		if out.charID == "" {
			out.charID = c.ID
		}
	}
	return out
}

// pathExt 取小写扩展名（.png 等）。
func pathExt(p string) string {
	idx := strings.LastIndexAny(p, `/\`)
	name := p
	if idx >= 0 {
		name = p[idx+1:]
	}
	dot := strings.LastIndex(name, ".")
	if dot < 0 {
		return ""
	}
	return name[dot:]
}
