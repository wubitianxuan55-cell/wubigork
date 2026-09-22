package app

// ── 图像能力域 T0 契约（设计 docs/gaea-image-domain-t0-contract-design-2026-09.md）──
//
// 本文件只立契约与登记主链，不搬任何引擎实现：
//   - 五原语能力注册表（识图-读/识图-懂/生图/改图/图示）；
//   - 产物溯源登记的调用入口（Asset Ledger v1，落 imagehub_ledger.go）；
//   - 运行态闸（真实 App 初始化后才落盘，单测/未初始化不写盘）。

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/ai"
)

// ImageCapability 图像域能力原语。
type ImageCapability string

const (
	CapabilityVisionRead       ImageCapability = "vision.read"       // 识图-读：OCR/取字
	CapabilityVisionUnderstand ImageCapability = "vision.understand" // 识图-懂：理解/描述/问答
	CapabilityMediaGenerate    ImageCapability = "media.generate"    // 生图（含 txt2img/img2img/t2v）
	CapabilityMediaEdit        ImageCapability = "media.edit"        // 改图（T3 起有实现）
	CapabilityMediaDiagram     ImageCapability = "media.diagram"     // 图示：流程图/导图/架构图
)

// imageCapabilityEntry 能力注册条目：只回答「是否可用/是否产物」，模型档位问模型中心目录。
type imageCapabilityEntry struct {
	Capability    ImageCapability
	ProducesAsset bool // 是否产生可登记资产（识图返回文本，不产物）
	Available     bool // T0: edit=false（契约先立，实现 T3）
}

// imageDomainRegistry 能力注册表（单写点；未知能力 fail-closed）。
var imageDomainRegistry = map[ImageCapability]imageCapabilityEntry{
	CapabilityVisionRead:       {Capability: CapabilityVisionRead},
	CapabilityVisionUnderstand: {Capability: CapabilityVisionUnderstand},
	CapabilityMediaGenerate:    {Capability: CapabilityMediaGenerate, ProducesAsset: true, Available: true},
	CapabilityMediaEdit:        {Capability: CapabilityMediaEdit, ProducesAsset: true},
	CapabilityMediaDiagram:     {Capability: CapabilityMediaDiagram, ProducesAsset: true, Available: true},
}

// imageDomainEntry 取能力条目；未知能力报错（fail-closed，不静默放行）。
func imageDomainEntry(cap ImageCapability) (imageCapabilityEntry, error) {
	e, ok := imageDomainRegistry[cap]
	if !ok {
		return imageCapabilityEntry{}, fmt.Errorf("图像域未知能力: %s", cap)
	}
	return e, nil
}

// 资产类型常量。
const (
	ImageHubAssetKindImage   = "image"
	ImageHubAssetKindVideo   = "video"
	ImageHubAssetKindDiagram = "diagram"
)

// imageHubAsset 产物登记最小集（路径优先，不塞 base64）。
type imageHubAsset struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Path string `json:"path"`
	MIME string `json:"mime,omitempty"`
}

// imageHubAssetMeta 产物溯源元数据（v1 最小集 + 阶段二刀 D 变体簇）。
type imageHubAssetMeta struct {
	Space       string                 `json:"space"`
	SourceBoard string                 `json:"source_board"`
	Capability  string                 `json:"capability"`
	Backend     string                 `json:"backend,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Cost        string                 `json:"cost,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	LicenseHint string                 `json:"license_hint,omitempty"`
	AIFlag      bool                   `json:"ai_flag"`
	// ParentID 变体簇（阶段二刀 D）：编辑/扩图产物指向源图条目——A→A'→A'' 同源
	// 链；空=非派生产物或源图不在台账（导入图，链条诚实断开）。旧 JSONL 无此
	// 字段反序列化零值空，向后兼容。
	ParentID string `json:"parent_id,omitempty"`
}

// imageHubAssetSeq 资产 ID 自增源（进程内原子，防同一纳秒并发重号）。
var imageHubAssetSeq int64

// newImageHubAssetID 生成资产 ID（纳秒时间戳 + 进程内序号）。
func newImageHubAssetID() string {
	n := atomic.AddInt64(&imageHubAssetSeq, 1)
	return fmt.Sprintf("ih-%d-%06d", time.Now().UnixNano(), n)
}

// imageHubRuntimeArmed 运行态武装标记：仅真实 App 生命周期（Startup）置位。
var imageHubRuntimeArmed atomic.Bool

// imageHubLedgerRuntimeCheck 运行态闸：真实 App 启动后登记才落盘。
// 单测/未初始化环境（如既有 handler 测试）跳过，避免污染目录且不影响主流程。
// v4.396 修订（真机走查抓出）：判据只留 armed 位——原叠加的 gaeaCfgSnapshot()!=nil
// 会把「纯绘梦会话」误判非运行态（ga.cfg 仅办公引擎 GaeaInit 赋值，不碰办公的
// 会话恒 nil），台账静默跳过、素材库/变体簇全断。测试进程不调 Startup，
// armed 位本身已是测试/运行的可靠判据。
var imageHubLedgerRuntimeCheck = func() bool {
	return imageHubRuntimeArmed.Load()
}

// recordImageHubGeneratedAsset 生图原语产物登记（media.generate 统一入口）。
//   - 登记失败只向上返回 error，由调用方 warn；绝不拖垮生成主路径；
//   - allowRoots 非空时校验路径必须落在允许根内（防穿越）；
//   - 空 allowRoots = 信任调用方已自持的保存路径（绘梦 ImageSaveDir 等外部根）。
func recordImageHubGeneratedAsset(cwd, space, sourceBoard, backend, model, prompt string,
	params map[string]interface{}, asset imageHubAsset, allowRoots []string, parentID string) error {

	if !imageHubLedgerRuntimeCheck() {
		return nil // 非运行态：不落盘（行为保持，登记是辅助视图）
	}
	entry, err := imageDomainEntry(CapabilityMediaGenerate)
	if err != nil {
		return err
	}
	if !entry.Available || !entry.ProducesAsset {
		return fmt.Errorf("图像域能力不可用: %s", CapabilityMediaGenerate)
	}
	if strings.TrimSpace(asset.Path) == "" {
		return fmt.Errorf("缺少产物路径")
	}
	if len(allowRoots) > 0 && !imagePathWithinAny(asset.Path, allowRoots) {
		return fmt.Errorf("产物路径不在允许根内: %s", asset.Path)
	}
	if asset.ID == "" {
		asset.ID = newImageHubAssetID()
	}
	if asset.Kind == "" {
		asset.Kind = imageHubKindByPath(asset.Path)
	}
	if asset.MIME == "" {
		asset.MIME = imageHubMIMEByExt(asset.Path)
	}
	cost, license := imageHubCostAndLicense(model)
	meta := imageHubAssetMeta{
		Space:       normalizeImageHubSpace(space),
		SourceBoard: sourceBoard,
		Capability:  string(CapabilityMediaGenerate),
		Backend:     backend,
		Model:       model,
		Cost:        cost,
		Prompt:      prompt,
		Params:      params,
		CreatedAt:   time.Now().Format(time.RFC3339),
		ParentID:    strings.TrimSpace(parentID),
		LicenseHint: license,
		AIFlag:      true,
	}
	led := newImageHubLedger(cwd)
	return led.record(meta.Space, imageHubLedgerRecord{Meta: meta, Asset: asset})
}

// imagePathWithinAny 判断 path 是否落在任一允许根内（filepath.Rel 防穿越口径）。
func imagePathWithinAny(path string, roots []string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, r := range roots {
		if strings.TrimSpace(r) == "" {
			continue
		}
		ra, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(ra, abs)
		if err != nil {
			continue
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return true
	}
	return false
}

// imageHubKindByPath 按扩展名推断资产类型（视频优先；缺省 image）。
func imageHubKindByPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".webm", ".mov":
		return ImageHubAssetKindVideo
	case ".mmd":
		return ImageHubAssetKindDiagram
	default:
		return ImageHubAssetKindImage
	}
}

// imageHubMIMEByExt 按扩展名推断 MIME（对齐 GaeaAttachmentDataURL 白名单口径）。
func imageHubMIMEByExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return ""
	}
}

// recordImageHubGenerated 绘梦工作台/媒体落盘后的登记（mediaState 便捷方法）。
func (m *mediaState) recordImageHubGenerated(item imageItem, mode, characterID, parentID string) {
	m.recordImageHubGeneratedFor(item, mode, characterID, "imagegen", parentID)
}

// imageHubAssetIDByPath 按产物路径在台账同空间查最近条目 ID（变体簇溯源键，
// 阶段二刀 D）。倒序扫描（最新在前）；未命中返回空串=链条诚实断开。
func imageHubAssetIDByPath(cwd, space, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	recs := newImageHubLedger(cwd).list(space, 0)
	for i := len(recs) - 1; i >= 0; i-- {
		if recs[i].Asset.Path == path {
			return recs[i].Asset.ID
		}
	}
	return ""
}

// recordImageHubGeneratedFor 与 recordImageHubGenerated 同语义，但把来源板块
// 参数化：绘梦/媒体走 "imagegen"（原行为逐字节），原罪板块走 "sin"。
// 动机（v4.257 硬隔离刀）：一个产物只登记一条——生成链内部登记，调用方
// 不得再补第二条（否则画室同一张图会以两个来源重复出现）。
func (m *mediaState) recordImageHubGeneratedFor(item imageItem, mode, characterID, sourceBoard, parentID string) {
	if item.FilePath == "" {
		return
	}
	kind := item.Kind
	if kind == "" {
		if mode == "t2v" {
			kind = ImageHubAssetKindVideo
		} else {
			kind = ImageHubAssetKindImage
		}
	}
	params := map[string]interface{}{"seed": item.Seed, "size": item.Size, "mode": mode, "n": 1}
	if characterID != "" {
		params["character_id"] = characterID
	}
	space := gaeaEffectiveSpace()
	if err := recordImageHubGeneratedAsset(gaeaCwd(), space, sourceBoard, m.cfg.ImageBackend,
		item.Model, item.Prompt,
		params,
		imageHubAsset{Kind: kind, Path: item.FilePath}, nil, parentID); err != nil {
		slog.Warn("绘梦产物登记失败（不影响生成）", "path", item.FilePath, "error", err)
	}
}

// recordImageHubDiagramAsset 图示原语产物登记（media.diagram 统一入口，T1 收口）。
// 与 recordImageHubGeneratedAsset 同一试点模式：登记失败只向上返回 error，由
// 调用方 warn，绝不拖垮生成主路径；allowRoots 非空时校验路径必须落在允许根内。
func recordImageHubDiagramAsset(cwd, space, sourceBoard, backend, model, prompt string,
	params map[string]interface{}, asset imageHubAsset, allowRoots []string) error {

	if !imageHubLedgerRuntimeCheck() {
		return nil // 非运行态：不落盘（行为保持，登记是辅助视图）
	}
	entry, err := imageDomainEntry(CapabilityMediaDiagram)
	if err != nil {
		return err
	}
	if !entry.Available || !entry.ProducesAsset {
		return fmt.Errorf("图像域能力不可用: %s", CapabilityMediaDiagram)
	}
	if strings.TrimSpace(asset.Path) == "" {
		return fmt.Errorf("缺少产物路径")
	}
	if len(allowRoots) > 0 && !imagePathWithinAny(asset.Path, allowRoots) {
		return fmt.Errorf("产物路径不在允许根内: %s", asset.Path)
	}
	if asset.ID == "" {
		asset.ID = newImageHubAssetID()
	}
	if asset.Kind == "" {
		asset.Kind = imageHubKindByPath(asset.Path)
	}
	if asset.MIME == "" {
		asset.MIME = imageHubMIMEByExt(asset.Path)
	}
	cost, license := imageHubCostAndLicense(model)
	meta := imageHubAssetMeta{
		Space:       normalizeImageHubSpace(space),
		SourceBoard: sourceBoard,
		Capability:  string(CapabilityMediaDiagram),
		Backend:     backend,
		Model:       model,
		Cost:        cost,
		Prompt:      prompt,
		Params:      params,
		CreatedAt:   time.Now().Format(time.RFC3339),
		LicenseHint: license,
		AIFlag:      true,
	}
	led := newImageHubLedger(cwd)
	return led.record(meta.Space, imageHubLedgerRecord{Meta: meta, Asset: asset})
}

// ImageHubAssets 画室素材库读取（T1 绑定入口：空间/来源筛选，最新在前）。
func (m *mediaState) ImageHubAssets(space, sourceBoard string, limit int) []imageHubAssetView {
	return imageHubAssetSummaries(gaeaCwd(), space, sourceBoard, "", limit)
}

// imageHubAssetView 素材库/画室用的登记读取视图（T1 预留；绑定层接入时直接映射）。
type imageHubAssetView struct {
	ID             string                 `json:"id"`
	Kind           string                 `json:"kind"`
	Path           string                 `json:"path"`
	MIME           string                 `json:"mime,omitempty"`
	Space          string                 `json:"space"`
	SourceBoard    string                 `json:"source_board"`
	Capability     string                 `json:"capability"`
	Backend        string                 `json:"backend,omitempty"`
	Model          string                 `json:"model,omitempty"`
	Cost           string                 `json:"cost,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	PromptTruncate string                 `json:"prompt_truncate,omitempty"`
	Params         map[string]interface{} `json:"params,omitempty"`
	ParentID       string                 `json:"parent_id,omitempty"`
}

// imageHubAssetSummaries 读取某空间最近的登记（可按来源板块/能力过滤；按时间倒序）。
// limit<=0 或超过登记数时返回全部命中（由调用方分页）。
func imageHubAssetSummaries(cwd, space, sourceBoard string, capability ImageCapability, limit int) []imageHubAssetView {
	recs := newImageHubLedger(cwd).list(space, 0)
	out := make([]imageHubAssetView, 0, len(recs))
	// JSONL 为追加序（旧→新），倒序取最新在前。
	for i := len(recs) - 1; i >= 0; i-- {
		r := recs[i]
		if sourceBoard != "" && r.Meta.SourceBoard != sourceBoard {
			continue
		}
		if capability != "" && r.Meta.Capability != string(capability) {
			continue
		}
		out = append(out, imageHubAssetView{
			ID:             r.Asset.ID,
			Kind:           r.Asset.Kind,
			Path:           r.Asset.Path,
			MIME:           r.Asset.MIME,
			Space:          r.Meta.Space,
			SourceBoard:    r.Meta.SourceBoard,
			Capability:     r.Meta.Capability,
			Backend:        r.Meta.Backend,
			Model:          r.Meta.Model,
			Cost:           r.Meta.Cost,
			CreatedAt:      r.Meta.CreatedAt,
			PromptTruncate: truncatePromptForView(r.Meta.Prompt, 80),
			Params:         r.Meta.Params,
			ParentID:       r.Meta.ParentID,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// truncatePromptForView 提示词只进视图摘要（防登记文件被当全文库用）。
func truncatePromptForView(p string, n int) string {
	if len(p) <= n {
		return p
	}
	runes := []rune(p)
	if len(runes) <= n {
		return p
	}
	return string(runes[:n]) + "…"
}

// ImageCutout 抠图/透明底导出（阶段二刀 E 绑定入口，T3 收官）：涂选保留主体
// → 蒙版即 alpha → 透明 PNG 落盘 ImageSaveDir + 台账登记（v4.396 修复后闸
// 正常）→ 返回 {path, asset_id}。零模型调用不设引擎门；校验/合成失败如实报错。
func (m *mediaState) ImageCutout(initImage, maskData string) map[string]interface{} {
	if strings.TrimSpace(initImage) == "" || strings.TrimSpace(maskData) == "" {
		return map[string]interface{}{"error": "抠图需要原图与涂选蒙版（涂抹要保留的主体）"}
	}
	png, err := ai.ComposeCutout(initImage, maskData)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	dir := m.cfg.ImageSaveDir
	if dir == "" {
		dir = defaultImageSaveDir()
	}
	path := m.saveMediaToDisk(png, "抠图-透明底", dir)
	if path == "" {
		return map[string]interface{}{"error": "抠图产物落盘失败（保存目录不可写）"}
	}
	// 台账登记（mode=cutout 入 params；asset_id 按路径回填——v4.395 链路复用）
	item := imageItem{Image: png, Prompt: "抠图-透明底", Model: "cutout", Kind: "image", FilePath: path}
	m.recordImageHubGeneratedFor(item, "cutout", "", "imagegen", "")
	return map[string]interface{}{
		"path":     path,
		"asset_id": imageHubAssetIDByPath(gaeaCwd(), gaeaEffectiveSpace(), path),
	}
}

// defaultImageSaveDir ImageSaveDir 未配置时的兜底（与 saveToNovelImages 落点
// 分区不同：抠图产物是绘梦工作台资产，走用户图片目录惯例）。
func defaultImageSaveDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return filepath.Join(home, "Pictures", "gaea")
}
