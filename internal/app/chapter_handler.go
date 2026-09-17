package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// ── 章节 I/O ─────────────────────────────────────────────────
func (a *writingState) GetChapter(num int) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	content, err := pm.ReadChapter(num)
	if err != nil {
		return nil, err
	}
	summary, err := pm.ReadChapterSummary(num)
	if err != nil {
		slog.Warn("读取章节摘要失败", "chapter", num, "error", err)
	}
	return map[string]interface{}{
		"content": content,
		"summary": summary,
	}, nil
}

// GetChapterBranch 读取分支章节内容（branch 为空时读主线）
func (a *writingState) GetChapterBranch(num int, branch string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	var content string
	var err error
	if branch != "" {
		content, err = pm.ReadChapterBranch(num, branch)
	} else {
		content, err = pm.ReadChapter(num)
	}
	if err != nil {
		return nil, err
	}
	var summary *types.ChapterSummary
	if branch != "" {
		summary, err = pm.ReadChapterBranchSummary(num, branch)
	} else {
		summary, err = pm.ReadChapterSummary(num)
	}
	if err != nil {
		slog.Warn("读取章节摘要失败", "chapter", num, "error", err)
	}
	return map[string]interface{}{
		"content": content,
		"summary": summary,
	}, nil
}

// SaveChapterContent 手动保存章节内容（主线）
func (a *writingState) SaveChapterContent(num int, content string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if err := pm.WriteChapter(num, content); err != nil {
		return err
	}
	a.markChapterWritten(num, "")
	return nil
}

// SaveChapterBranchContent 手动保存分支章节内容
func (a *writingState) SaveChapterBranchContent(num int, branch string, content string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if err := pm.WriteChapterBranch(num, branch, content); err != nil {
		return err
	}
	a.markChapterWritten(num, branch)
	return nil
}

// markChapterWritten 手动保存章节后同步大纲状态，避免阅读页保存了正文但大纲仍显示未写/生成中。
func (a *writingState) markChapterWritten(num int, branch string) {
	pm := a.getPM()
	if pm == nil {
		return
	}
	of, err := pm.ReadOutlines()
	if err != nil {
		slog.Warn("手动保存章节后读取大纲失败", "chapter", num, "error", err)
		return
	}
	changed := false
	for i := range of.Nodes {
		if markNodeWritten(&of.Nodes[i], num, branch) {
			changed = true
			break
		}
	}
	if !changed {
		return
	}
	if err := pm.WriteOutlines(of); err != nil {
		slog.Warn("手动保存章节后更新大纲状态失败", "chapter", num, "error", err)
	}
}

func markNodeWritten(node *types.OutlineNode, num int, branch string) bool {
	if node.OrderIndex == num && node.Branch == branch {
		node.Status = types.OutlineDone
		return true
	}
	for i := range node.Children {
		if markNodeWritten(&node.Children[i], num, branch) {
			return true
		}
	}
	return false
}

// GenerateSceneIllustration 为指定章节生成场景插图（Aurora）
// GenerateSceneIllustration 为指定章节生成场景插图（Aurora）
// sceneIllustrationOpts 配图 v2 选项（阶段一刀 D 核心片，规格
// 进度计划/gaea-scene-illustration-v2-20260917.md）。空 JSON/空串=旧行为零变化。
type sceneIllustrationOpts struct {
	CharacterIDs []string `json:"characterIds"` // 参考图角色 ID（无 PortraitURL 的诚实跳过）
	Style        string   `json:"style"`        // 风格槽（空=默认风格句）
}

// GenerateSceneIllustration 为章节生成场景插图（v2 签名扩展，v4.139 Save 先例：
// 增 optsJSON 尾参，空串零行为变化；绑定面不增名）。
func (a *writingState) GenerateSceneIllustration(chapterNum int, optsJSON string) (map[string]interface{}, error) {
	if a.chapterAgent == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	var opts sceneIllustrationOpts
	if strings.TrimSpace(optsJSON) != "" {
		if err := json.Unmarshal([]byte(optsJSON), &opts); err != nil {
			return nil, fmt.Errorf("配图选项格式不正确: %w", err)
		}
	}

	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取章节失败: %w", err)
	}
	summary, err := pm.ReadChapterSummary(chapterNum)
	if err != nil {
		slog.Warn("读取章节摘要失败", "chapter", chapterNum, "error", err)
	}

	// 获取角色和世界观
	chars, err := pm.ReadCharacters()
	if err != nil {
		slog.Warn("读取角色失败", "error", err)
	}
	var characterList []types.Character
	if chars != nil {
		characterList = chars.Characters
	}

	wv, err := pm.ReadWorldview()
	if err != nil {
		slog.Warn("读取世界观失败", "error", err)
	}

	// 参考槽路由（Q3/Q4）：角色 PortraitURL → data URL 列表；仅参考能力后端
	// 附着（comfyui/herdsman——刀 B img2img 口径），其它后端诚实降级提示。
	refs, refNote := a.sceneIllustrationRefs(pm, characterList, opts.CharacterIDs)

	resp, err := a.chapterAgent.GenerateSceneIllustrationV2(a.ctx, content, summary, characterList, wv, refs, opts.Style)
	if err != nil {
		return nil, fmt.Errorf("生成插图失败: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未生成图片")
	}

	// T1：配图落盘 play exports + 图像域登记（失败只 warn，URL 返回不变）。
	outPath, saveErr := saveSceneIllustrationToPlayExports(pm, chapterNum, resp)
	if saveErr != nil {
		slog.Warn("章节配图落盘失败（继续返回 URL）", "chapter", chapterNum, "error", saveErr)
	} else {
		revised := ""
		if resp.Data[0].RevisedPrompt != "" {
			revised = resp.Data[0].RevisedPrompt
		}
		asset := imageHubAsset{
			ID:   newImageHubAssetID(),
			Kind: ImageHubAssetKindImage,
			Path: outPath,
			MIME: "image/png",
		}
		regErr := recordImageHubGeneratedAsset(gaeaCwd(), "play", "novel", "",
			"grok-imagine-image-quality", revised,
			map[string]interface{}{"chapter": chapterNum, "size": "1024x576", "n": 1},
			asset, nil)
		if regErr != nil {
			slog.Warn("章节配图登记失败（不影响生成）", "path", outPath, "error", regErr)
		}
		if artErr := appendChapterArt(pm, chapterNum, asset.ID, outPath); artErr != nil {
			slog.Warn("章节插图清单追加失败（不影响生成）", "path", outPath, "error", artErr)
		}
	}

	result := map[string]interface{}{
		"url":            resp.Data[0].URL,
		"revised_prompt": resp.Data[0].RevisedPrompt,
		"refNote":        refNote, // 参考使用情况（Q6：恒回，前端浅色说明行）
	}
	if outPath != "" {
		result["file_path"] = outPath
	}
	return result, nil
}

// sceneIllustrationRefs 参考槽收集与路由（Q3/Q4）：
//   - characterIDs → 项目角色 PortraitURL（data URL 直用；本地路径读盘转
//     data URL；无 PortraitURL 的角色诚实跳过并计入说明）；
//   - 仅参考能力后端（comfyui/herdsman，刀 B img2img 口径）返回非空 refs；
//     其它后端 refs=nil + refNote「不支持参考图」——诚实降级提示不静默。
func (a *writingState) sceneIllustrationRefs(pm *project.Manager, characters []types.Character, characterIDs []string) (refs []string, refNote string) {
	if len(characterIDs) == 0 {
		return nil, ""
	}
	byID := make(map[string]types.Character, len(characters))
	for _, c := range characters {
		byID[c.ID] = c
	}
	var skipped []string
	for _, id := range characterIDs {
		c, ok := byID[id]
		if !ok {
			skipped = append(skipped, id+"（不存在）")
			continue
		}
		p := strings.TrimSpace(c.PortraitURL)
		if p == "" {
			skipped = append(skipped, c.Name+"（无立绘）")
			continue
		}
		if strings.HasPrefix(p, "data:") {
			refs = append(refs, p)
			continue
		}
		if dataURL, err := readImageFileAsDataURL(p); err == nil {
			refs = append(refs, dataURL)
		} else {
			slog.Warn("角色参考图读取失败（跳过）", "character", c.Name, "path", p, "error", err)
			skipped = append(skipped, c.Name+"（参考图读取失败）")
		}
	}
	// 后端能力路由：非参考能力后端不附着（后端会拒绝 img2img 或文生图带参考）。
	backend := ""
	if a.cfg != nil {
		backend = a.cfg.ImageBackend
	}
	if len(refs) > 0 && backend != "comfyui" && backend != "herdsman" {
		note := "当前引擎（" + backend + "）不支持参考图，本次纯文生图"
		if len(skipped) > 0 {
			note += "；另有角色未取到参考：" + strings.Join(skipped, "、")
		}
		return nil, note
	}
	if len(refs) > 0 {
		note := fmt.Sprintf("已附 %d 张角色参考图（img2img）", len(refs))
		if len(skipped) > 0 {
			note += "；未取到参考：" + strings.Join(skipped, "、")
		}
		return refs, note
	}
	return nil, "未取到任何角色参考图：" + strings.Join(skipped, "、")
}

// readImageFileAsDataURL 本地图片路径 → data URL（参考槽只认 data URL——
// comfyui uploadImage 与 herdsman img2img 同口径）。
func readImageFileAsDataURL(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(b) > 8<<20 {
		return "", fmt.Errorf("图片过大（上限 8MB）")
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// ── v4 场景 API ──────────────────────────────────────────────

// GetChapterScenes 获取章节的场景列表
func (a *writingState) GetChapterScenes(chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	if !pm.IsV4() {
		// v3 项目：返回单场景视图
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, err
		}
		return []map[string]interface{}{{
			"id":      fmt.Sprintf("%03d-chapter", chapterNum),
			"title":   fmt.Sprintf("第%d章", chapterNum),
			"content": content,
			"status":  "done",
			"order":   1,
		}}, nil
	}

	sm := pm.SceneManager(chapterNum)
	// 1A 接通：纯 blob 章首次读场景时物化为首场景，逐场景生成由此拿到真实 id。
	ensureBlobChapterScene(pm, chapterNum)
	scenes, err := sm.List()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, meta := range scenes {
		scene, err := sm.Read(meta.ID)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"id":        scene.Meta.ID,
			"slug":      scene.Meta.Slug,
			"title":     scene.Meta.Title,
			"summary":   scene.Meta.Summary,
			"povCharId": scene.Meta.POVCharID,
			"location":  scene.Meta.Location,
			"timeOfDay": scene.Meta.TimeOfDay,
			"emotion":   scene.Meta.Emotion,
			"tags":      scene.Meta.Tags,
			"status":    string(scene.Meta.Status),
			"wordCount": scene.Meta.WordCount,
			"order":     scene.Meta.Order,
			"content":   scene.Content,
		})
	}
	return result, nil
}

// SaveScene 保存单个场景
func (a *writingState) SaveScene(chapterNum int, sceneID string, content string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	if !pm.IsV4() {
		return pm.WriteChapter(chapterNum, content)
	}

	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return err
	}
	scene.Content = content
	if err := sm.Write(scene); err != nil {
		return err
	}
	syncBlobFromScenes(pm, chapterNum)
	return nil
}

// ReorderScenes 重排场景顺序
func (a *writingState) ReorderScenes(chapterNum int, sceneIDs []string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if !pm.IsV4() {
		return nil // v3 项目无场景可排
	}
	if err := pm.SceneManager(chapterNum).Reorder(sceneIDs); err != nil {
		return err
	}
	syncBlobFromScenes(pm, chapterNum)
	return nil
}

// SaveSceneMeta 保存场景元数据（标题/概要/POV/地点/时间/情感基调/标签/状态）。
// 正文不经此路（SaveScene 专职）；元数据不影响 blob 投影内容，故不触发
// syncBlobFromScenes。身份字段 ID/Slug/Order/WordCount 以存储为准不可改；
// 状态仅接受 draft/revising/done（非法值整单拒绝，不静默吞）；标题 trim 后
// 为空视为「保持原标题」。
func (a *writingState) SaveSceneMeta(chapterNum int, sceneID string, metaJSON string) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	if !pm.IsV4() {
		return fmt.Errorf("v3 项目无场景元数据")
	}
	var patch struct {
		Title     string   `json:"title"`
		Summary   string   `json:"summary"`
		POVCharID string   `json:"povCharId"`
		Location  string   `json:"location"`
		TimeOfDay string   `json:"timeOfDay"`
		Emotion   string   `json:"emotion"`
		Tags      []string `json:"tags"`
		Status    string   `json:"status"`
	}
	if err := json.Unmarshal([]byte(metaJSON), &patch); err != nil {
		return fmt.Errorf("元数据解析失败: %w", err)
	}
	if patch.Status != "" && types.SceneStatus(patch.Status) != types.SceneDraft &&
		types.SceneStatus(patch.Status) != types.SceneRevising && types.SceneStatus(patch.Status) != types.SceneDone {
		return fmt.Errorf("非法场景状态 %q（仅 draft/revising/done）", patch.Status)
	}
	sm := pm.SceneManager(chapterNum)
	scene, err := sm.Read(sceneID)
	if err != nil {
		return err
	}
	if t := strings.TrimSpace(patch.Title); t != "" {
		scene.Meta.Title = t
	}
	scene.Meta.Summary = patch.Summary
	scene.Meta.POVCharID = strings.TrimSpace(patch.POVCharID)
	scene.Meta.Location = strings.TrimSpace(patch.Location)
	scene.Meta.TimeOfDay = strings.TrimSpace(patch.TimeOfDay)
	scene.Meta.Emotion = strings.TrimSpace(patch.Emotion)
	if patch.Tags != nil {
		scene.Meta.Tags = patch.Tags
	}
	if patch.Status != "" {
		scene.Meta.Status = types.SceneStatus(patch.Status)
	}
	return sm.Write(scene)
}

// CreateSnapshot 手动创建场景快照
func (a *writingState) CreateSnapshot(sceneID string, chapterNum int, label string) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	var content string
	if pm.IsV4() {
		scene, err := pm.SceneManager(chapterNum).Read(sceneID)
		if err != nil {
			return nil, err
		}
		content = scene.Content
	} else {
		var err error
		content, err = pm.ReadChapter(chapterNum)
		if err != nil {
			return nil, err
		}
	}

	store := pm.SnapshotStore(chapterNum)
	snap, err := store.Capture(sceneID, content, label, "manual")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        snap.ID,
		"timestamp": snap.Timestamp,
		"label":     snap.Label,
		"wordCount": snap.WordCount,
	}, nil
}

// ListSnapshots 列出场景的所有快照
func (a *writingState) ListSnapshots(sceneID string, chapterNum int) ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	store := pm.SnapshotStore(chapterNum)
	snaps, err := store.List(sceneID)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, snap := range snaps {
		result = append(result, map[string]interface{}{
			"id":        snap.ID,
			"timestamp": snap.Timestamp,
			"label":     snap.Label,
			"trigger":   snap.Trigger,
			"wordCount": snap.WordCount,
		})
	}
	return result, nil
}

// RestoreSnapshot 恢复到指定快照
func (a *writingState) RestoreSnapshot(snapshotID string, sceneID string, chapterNum int) error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}

	store := pm.SnapshotStore(chapterNum)
	content, err := store.Restore(snapshotID, sceneID)
	if err != nil {
		return err
	}

	if pm.IsV4() {
		scene, err := pm.SceneManager(chapterNum).Read(sceneID)
		if err != nil {
			return err
		}
		scene.Content = content
		if werr := pm.SceneManager(chapterNum).Write(scene); werr != nil {
			return werr
		}
		syncBlobFromScenes(pm, chapterNum)
		return nil
	}

	return pm.WriteChapter(chapterNum, content)
}

// MigrateProjectToV4 手动触发项目迁移到 v4
func (a *writingState) MigrateProjectToV4() error {
	pm := a.getPM()
	if pm == nil {
		return fmt.Errorf("请先打开项目")
	}
	return pm.MigrateV3ToV4()
}

// IsProjectV4 检查当前项目是否为 v4 结构
func (a *writingState) IsProjectV4() bool {
	pm := a.getPM()
	if pm == nil {
		return false
	}
	return pm.IsV4()
}
