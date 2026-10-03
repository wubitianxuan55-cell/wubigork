package project

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// Manager 项目管理器 — 一部小说一个文件夹
type Manager struct {
	Dir  string // 项目根目录
	Meta *types.ProjectMeta

	// fileMu 串行化 Manager 内部的「整表读-改-写」复合操作（审计 P1 IN1-07）：
	// UpsertAnalysisV2 / syncRewriteIndex 都是 读 JSON→改→writeJSON 整表覆盖，
	// 并发时后写者用旧快照覆盖先写者丢更新（plans.json 同构问题在 app 层有
	// chapterPlanMu 专锁先例）。读路径不持锁：写走原子替换，读到的要么旧要么
	// 新，无半截文件。
	fileMu sync.Mutex
}

// Create 新建小说项目目录
// 生成: project.json, worldview.md, characters.json, outline.json, chapters/, foreshadows.json
func Create(dir, title, genre, style, description string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建项目目录失败: %w", err)
	}

	now := time.Now()
	meta := &types.ProjectMeta{
		SchemaVersion: 1,
		Title:         title,
		Genre:         genre,
		Style:         style,
		Description:   description,
		CreatedAt:     now,
		LastOpenedAt:  now,
		Version:       1,
	}

	// project.json
	if err := writeJSON(filepath.Join(dir, "project.json"), meta); err != nil {
		return nil, err
	}

	// worldview.json（结构化，空）
	wf := types.WorldviewFile{Sections: DefaultSections()}
	if err := writeJSON(filepath.Join(dir, "worldview.json"), wf); err != nil {
		return nil, err
	}

	// characters.json（空）
	cf := types.CharacterFile{
		Characters:    []types.Character{},
		Organizations: []types.Organization{},
		Relationships: []types.Relationship{},
	}
	if err := writeJSON(filepath.Join(dir, "characters.json"), cf); err != nil {
		return nil, err
	}

	// outline.json（空）
	of := types.OutlineFile{
		Nodes: []types.OutlineNode{},
	}
	if err := writeJSON(filepath.Join(dir, "outline.json"), of); err != nil {
		return nil, err
	}

	// foreshadows.json（空）
	ff := types.ForeshadowFile{
		Items: []types.Foreshadow{},
	}
	if err := writeJSON(filepath.Join(dir, "foreshadows.json"), ff); err != nil {
		return nil, err
	}

	// chapters/ 目录（v4: 每章一个子目录含 scenes/）
	if err := os.MkdirAll(filepath.Join(dir, "chapters"), 0755); err != nil {
		return nil, err
	}
	// v4 标记文件
	versionMarker := filepath.Join(dir, ".gaea", "v4")
	if err := os.MkdirAll(filepath.Join(dir, ".gaea"), 0755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(versionMarker, []byte("4"), 0644); err != nil {
		return nil, err
	}

	return &Manager{Dir: dir, Meta: meta}, nil
}

// Open 打开已有小说项目目录
func Open(dir string) (*Manager, error) {
	meta, err := loadJSON[types.ProjectMeta](filepath.Join(dir, "project.json"))
	if err != nil {
		return nil, fmt.Errorf("无效的项目目录（缺少 project.json）: %w", err)
	}

	meta.LastOpenedAt = time.Now()
	meta.Version++
	if err := writeJSON(filepath.Join(dir, "project.json"), meta); err != nil {
		slog.Warn("更新 project.json 最后打开时间失败", "error", err)
	}

	return &Manager{Dir: dir, Meta: meta}, nil
}

// Close 关闭项目（保存元信息）
func (m *Manager) Close() error {
	if m.Meta == nil {
		return nil
	}
	m.Meta.LastOpenedAt = time.Now()
	return writeJSON(filepath.Join(m.Dir, "project.json"), m.Meta)
}

// WriteMeta 将当前元信息写回 project.json（UpdateProjectMeta 的落盘出口）。
// Version 不动——递增语义归 Open 的「打开次数」口径。
func (m *Manager) WriteMeta() error {
	if m.Meta == nil {
		return nil
	}
	return writeJSON(filepath.Join(m.Dir, "project.json"), m.Meta)
}

// ── 上下文构建 ──────────────────────────────────────────────

// LoadContext 加载完整 ProjectContext（Phase 1 简化版，不含 Memory）
func (m *Manager) LoadContext(currentOutlineID string) (*types.ProjectContext, error) {
	worldview, err := m.ReadWorldview()
	if err != nil {
		slog.Warn("LoadContext: 读取世界观失败", "error", err)
	}
	chars, err := m.ReadCharacters()
	if err != nil {
		slog.Warn("LoadContext: 读取角色失败", "error", err)
	}
	outlines, err := m.ReadOutlines()
	if err != nil {
		slog.Warn("LoadContext: 读取大纲失败", "error", err)
	}
	foreshadows, err := m.ReadForeshadows()
	if err != nil {
		slog.Warn("LoadContext: 读取伏笔失败", "error", err)
	}

	ctx := &types.ProjectContext{
		Project:   *m.Meta,
		Worldview: worldview,
	}

	if chars != nil {
		ctx.Characters = chars.Characters
		ctx.Organizations = chars.Organizations
		ctx.Relationships = chars.Relationships
	}
	if outlines != nil {
		ctx.Outlines = outlines.Nodes
		ctx.StoryThread = outlines.StoryThread
		// 找到当前大纲节点
		for i := range outlines.Nodes {
			findNode(outlines.Nodes[i], currentOutlineID, &ctx.CurrentOutline)
		}
		// 找到当前节点的父卷上下文
		for i := range outlines.Nodes {
			if parent := findParentVolume(outlines.Nodes[i], currentOutlineID); parent != nil {
				ctx.VolumeContext = fmt.Sprintf("卷: %s\n卷摘要: %s\n卷关键点: %s",
					parent.Title,
					parent.Summary,
					strings.Join(parent.KeyPoints, " / "),
				)
				break
			}
		}
	}
	if foreshadows != nil {
		ctx.Foreshadows = foreshadows.Items
	}

	return ctx, nil
}

func findNode(node types.OutlineNode, targetID string, result **types.OutlineNode) {
	if *result != nil {
		return
	}
	if node.ID == targetID {
		*result = &node
		return
	}
	for _, child := range node.Children {
		findNode(child, targetID, result)
	}
}

// findParentVolume 查找包含 targetID 子节点的卷节点
func findParentVolume(node types.OutlineNode, targetID string) *types.OutlineNode {
	for _, child := range node.Children {
		if child.ID == targetID {
			return &node
		}
		if len(child.Children) > 0 {
			if result := findParentVolume(child, targetID); result != nil {
				return result
			}
		}
	}
	return nil
}
