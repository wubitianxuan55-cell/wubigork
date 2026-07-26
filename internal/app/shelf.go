package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/types"
)

// ProjectCard 书架上的项目卡片数据（返回给前端）
type ProjectCard struct {
	Title        string `json:"title"`
	Genre        string `json:"genre"`
	Style        string `json:"style"`
	Path         string `json:"path"` // 项目完整路径
	WordCount    int    `json:"word_count"`
	ChapterCount int    `json:"chapter_count"`
	CreatedAt    string `json:"created_at"`     // ISO8601
	LastOpenedAt string `json:"last_opened_at"` // ISO8601
}

// GetNovelsDir 返回小说书架根目录
func (a *App) GetNovelsDir() string {
	return a.cfg.NovelsDir
}

// ListProjects 扫描书架目录，返回所有小说项目摘要
func (a *App) ListProjects() ([]ProjectCard, error) {
	novelsDir := a.cfg.NovelsDir

	// 确保书架目录存在
	if err := os.MkdirAll(novelsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建书架目录失败: %w", err)
	}

	entries, err := os.ReadDir(novelsDir)
	if err != nil {
		return nil, fmt.Errorf("读取书架目录失败: %w", err)
	}

	var cards []ProjectCard
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(novelsDir, entry.Name())
		metaPath := filepath.Join(dirPath, "project.json")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			continue // 不是有效项目目录，跳过
		}

		meta, err := loadProjectMeta(metaPath)
		if err != nil {
			continue // 项目文件损坏，跳过
		}

		chapterCount, wordCount := scanChapterStats(dirPath)

		cards = append(cards, ProjectCard{
			Title:        meta.Title,
			Genre:        meta.Genre,
			Style:        meta.Style,
			Path:         dirPath,
			WordCount:    wordCount,
			ChapterCount: chapterCount,
			CreatedAt:    meta.CreatedAt.Format(time.RFC3339),
			LastOpenedAt: meta.LastOpenedAt.Format(time.RFC3339),
		})
	}

	return cards, nil
}

// DeleteProject 删除整个项目目录
func (a *App) DeleteProject(dir string) error {
	// 安全检查：必须在书架目录下
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("路径解析失败: %w", err)
	}
	absNovels, err := filepath.Abs(a.cfg.NovelsDir)
	if err != nil {
		slog.Warn("shelf: 解析小说目录路径失败", "error", err)
	}
	if !strings.HasPrefix(absDir, absNovels+string(filepath.Separator)) {
		return fmt.Errorf("出于安全考虑，只能删除书架目录下的项目")
	}

	// 如果该项目当前已打开，先关闭
	if pm := a.getPM(); pm != nil && pm.Dir == dir {
		a.closePM()
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("删除项目失败: %w", err)
	}
	return nil
}

// ── 内部辅助 ─────────────────────────────────────────────────

// SaveConfig 将单个配置项写回 ~/.wubigork_config.json 并更新内存。
func (a *App) SaveConfig(key, value string) error {
	if err := config.Save(key, value); err != nil {
		return err
	}

	// 更新内存中的对应字段
	switch key {
	case "novels_dir":
		// 如果当前打开了旧目录下的项目，先关闭
		oldDir := a.cfg.NovelsDir
		if oldDir != value {
			if pm := a.getPM(); pm != nil {
				a.closePM()
			}
		}
		a.cfg.NovelsDir = value
	case "xai_client_id":
		a.cfg.XaiClientID = value
	case "http_timeout_seconds":
		if n, err := strconv.Atoi(value); err == nil {
			a.cfg.HTTPTimeoutSeconds = n
		}
	case "default_temperature":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			a.cfg.DefaultTemperature = f
		}
	case "model":
		a.cfg.Model = value
	}
	return nil
}

// ── 内部辅助 ─────────────────────────────────────────────────

// loadProjectMeta 轻量读取 project.json（只取需要的字段）
func loadProjectMeta(path string) (*types.ProjectMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var partial struct {
		Title        string    `json:"title"`
		Genre        string    `json:"genre"`
		Style        string    `json:"style"`
		CreatedAt    time.Time `json:"created_at"`
		LastOpenedAt time.Time `json:"last_opened_at"`
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return nil, err
	}
	return &types.ProjectMeta{
		Title:        partial.Title,
		Genre:        partial.Genre,
		Style:        partial.Style,
		CreatedAt:    partial.CreatedAt,
		LastOpenedAt: partial.LastOpenedAt,
	}, nil
}

// scanChapterStats 快速扫描 chapters/ 目录获取章节数和总字数
func scanChapterStats(projectDir string) (chapterCount int, totalWords int) {
	chaptersDir := filepath.Join(projectDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return 0, 0
	}
	re := regexp.MustCompile(`^\d+\.md$`)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 只统计 NNN.md 文件（纯数字前缀），跳过 NNN-summary.json
		if !re.MatchString(name) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(chaptersDir, name))
		if err != nil {
			continue
		}
		chapterCount++
		totalWords += utf8.RuneCountInString(string(content))
	}
	return
}
