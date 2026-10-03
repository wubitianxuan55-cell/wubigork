package project

// project_migrate.go — 版本迁移与通用 JSON 辅助（批 28 IN1-01 文件拆分，自
// project.go 原位搬移，零逻辑改动）：writeJSON/loadJSON 泛型原子读写（全包
// 共用，落点在本文件纯属组织性）、DefaultSections、IsV4（与 legacyBrandDir
// 同源判定）、MigrateV3ToV4 及 finalizeV4Migration。

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/scene"
	"github.com/gaea/gaea/internal/types"
)

// ── 内部辅助 ─────────────────────────────────────────────────

// writeJSON 序列化后经 fileutil.AtomicWrite 原子落盘（同目录临时文件 +
// RenameWithRetry 覆盖，失败保留旧文件）。perm 传 0o600 与旧私有实现的
// os.CreateTemp 默认权限位一致，收敛后行为不变。
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败 (%s): %w", path, err)
	}
	if err := fileutil.AtomicWrite(path, data, 0o600); err != nil {
		return fmt.Errorf("写入文件失败 (%s): %w", path, err)
	}
	return nil
}

func loadJSON[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", path, err)
	}
	return &val, nil
}

func DefaultSections() []types.WorldviewSection {
	return []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: "", Order: 1},
		{ID: "geography", Title: "地理风貌", Content: "", Order: 2},
		{ID: "factions", Title: "势力格局", Content: "", Order: 3},
		{ID: "rules", Title: "规则体系", Content: "", Order: 4},
		{ID: "culture", Title: "文化习俗", Content: "", Order: 5},
		{ID: "history", Title: "历史事件", Content: "", Order: 6},
	}
}

// ── v4 支持 ──────────────────────────────────────────────────

// IsV4 检测项目是否为 v4 目录结构
func (m *Manager) IsV4() bool {
	// 兼容旧品牌：新标记 .gaea/v4，旧项目 .wubigork/v4 同样识别
	// （旧品牌目录名单源 legacyBrandDir，与本包档案级回退同源）
	if _, err := os.Stat(filepath.Join(m.Dir, ".gaea", "v4")); err == nil {
		return true
	}
	_, err := os.Stat(filepath.Join(legacyBrandDir(m.Dir), "v4"))
	return err == nil
}

// MigrateV3ToV4 将 v3 项目迁移到 v4 结构（非破坏性）
// 原始 v3 文件备份到 _v3_backup/ 子目录
func (m *Manager) MigrateV3ToV4() error {
	if m.IsV4() {
		return nil // 已经是 v4
	}

	// 备份目录
	backupDir := filepath.Join(m.Dir, "_v3_backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	// 读取所有已有章节，迁移到 v4 结构
	//
	// N12 修复：无法迁移的章（文件名解析不出章号 / 读失败）**收集后如实失败**，
	// 绝不静默跳过再照落 v4 标记——标记一落，IsV4() 为真，未迁移的 v3 章从此
	// 从读写视图里消失（ReadChapterAsStitch 优先 v4 视图），用户数据「没了」
	// 却无任何提示。
	chaptersDir := filepath.Join(m.Dir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return m.finalizeV4Migration()
		}
		return err
	}

	var skipped []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// 解析章节号
		name := e.Name()
		var chapterNum int
		if _, err := fmt.Sscanf(name, "%03d.md", &chapterNum); err != nil {
			skipped = append(skipped, name+"（文件名不是 NNN.md 章节格式）")
			continue
		}

		// 读旧内容
		oldPath := filepath.Join(chaptersDir, name)
		content, err := os.ReadFile(oldPath)
		if err != nil {
			skipped = append(skipped, name+"（读取失败: "+err.Error()+"）")
			continue
		}

		// 备份旧文件
		backupPath := filepath.Join(backupDir, name)
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			slog.Warn("v4迁移: 备份文件失败", "file", name, "error", err)
		}

		// 创建 v4 场景
		chDir := filepath.Join(chaptersDir, fmt.Sprintf("%03d", chapterNum))
		sm := scene.NewManager(chDir)
		sceneObj, err := sm.Create("chapter", fmt.Sprintf("第%d章", chapterNum))
		if err != nil {
			return fmt.Errorf("迁移第%d章失败: %w", chapterNum, err)
		}
		sceneObj.Content = string(content)
		if err := sm.Write(sceneObj); err != nil {
			return fmt.Errorf("写入迁移场景失败: %w", err)
		}

		// 迁移摘要文件（如果有）
		oldSummaryPath := filepath.Join(chaptersDir, fmt.Sprintf("%03d-summary.json", chapterNum))
		if summaryData, err := os.ReadFile(oldSummaryPath); err == nil {
			backupSummaryPath := filepath.Join(backupDir, fmt.Sprintf("%03d-summary.json", chapterNum))
			_ = os.WriteFile(backupSummaryPath, summaryData, 0644)
		}
	}

	// 有未迁移章则失败且不落 v4 标记。v3 原文件非破坏保留（未删除），用户
	// 处理掉 listed 文件后重试即可。
	if len(skipped) > 0 {
		return fmt.Errorf("迁移中止：以下 %d 个章节文件无法迁移（未落 v4 标记，处理后可重试）：\n%s",
			len(skipped), strings.Join(skipped, "\n"))
	}

	return m.finalizeV4Migration()
}

func (m *Manager) finalizeV4Migration() error {
	// 写入 v4 标记
	markerDir := filepath.Join(m.Dir, ".gaea")
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(markerDir, "v4"), []byte("4"), 0644)
}
