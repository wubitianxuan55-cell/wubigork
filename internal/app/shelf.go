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

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/types"
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

// SaveConfig 将单个配置项写回 ~/.gaea_config.json 并更新内存。
// T7-2 可见性收口：内存同步补齐到全部支持项（写盘后同步 a.cfg），
// 避免「盘上已改、内存未动」的设置不同步；无内存对应项的配置写盘后
// 明确记 Warn 日志标注需重启生效。
func (a *App) SaveConfig(key, value string) error {
	if err := config.Save(key, value); err != nil {
		return err
	}

	// 更新内存中的对应字段（config.Save 已校验值格式，这里解析失败仅记录
	// 并跳过——磁盘已是权威来源，不阻断）。
	switch key {
	case config.KeyNovelsDir:
		// 如果当前打开了旧目录下的项目，先关闭
		oldDir := a.cfg.NovelsDir
		if oldDir != value {
			if pm := a.getPM(); pm != nil {
				a.closePM()
			}
		}
		a.cfg.NovelsDir = value
	case config.KeyXaiClientID:
		a.cfg.XaiClientID = value
	case config.KeyModel:
		a.cfg.Model = value
	case config.KeyHTTPTimeoutSeconds:
		if n, err := strconv.Atoi(value); err == nil {
			a.cfg.HTTPTimeoutSeconds = n
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（整数解析失败）", "key", key, "value", value)
		}
	case config.KeyDefaultTemperature:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			a.cfg.DefaultTemperature = f
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（浮点解析失败）", "key", key, "value", value)
		}
	case config.KeyAnalysisTemperature:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			a.cfg.AnalysisTemperature = f
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（浮点解析失败）", "key", key, "value", value)
		}
	case config.KeyReasoningEffort:
		a.cfg.ReasoningEffort = value
	case config.KeyQualityThreshold:
		if n, err := strconv.Atoi(value); err == nil {
			a.cfg.QualityThreshold = n
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（整数解析失败）", "key", key, "value", value)
		}
	case config.KeyQualityMaxRetries:
		if n, err := strconv.Atoi(value); err == nil {
			a.cfg.QualityMaxRetries = n
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（整数解析失败）", "key", key, "value", value)
		}
	case config.KeyTTSBinaryPath:
		a.cfg.TTSBinaryPath = value
	case config.KeyTTSModelPath:
		a.cfg.TTSModelPath = value
	case config.KeyTTSPort:
		if n, err := strconv.Atoi(value); err == nil {
			a.cfg.TTSPort = n
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（整数解析失败）", "key", key, "value", value)
		}
	case config.KeyTTSBackend:
		a.cfg.TTSBackend = value
	case config.KeyTTSSpeed:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			a.cfg.TTSSpeed = f
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（浮点解析失败）", "key", key, "value", value)
		}
	case config.KeyImageBackend:
		a.cfg.ImageBackend = value
	case config.KeyComfyUIURL:
		a.cfg.ComfyUIURL = value
	case config.KeyImageSaveDir:
		a.cfg.ImageSaveDir = value
	case config.KeyImageModel:
		a.cfg.ImageModel = value
	case config.KeyPortraitBackend:
		a.cfg.PortraitBackend = value
	case config.KeyPortraitModel:
		a.cfg.PortraitModel = value
	case config.KeyComfyUIPath:
		a.cfg.ComfyUIPath = value
	case config.KeyComfyUIPythonPath:
		a.cfg.ComfyUIPythonPath = value
	case config.KeyActiveEngineID:
		a.cfg.ActiveEngineID = value
	case config.KeyDeepseekAPIKey:
		a.cfg.DeepseekAPIKey = value
	case config.KeyOpencodeGoAPIKey:
		a.cfg.OpenCodeGoAPIKey = value
	case config.KeyOpencodeZenAPIKey:
		a.cfg.OpenCodeZenAPIKey = value
	case config.KeyActiveASREngine:
		a.cfg.ActiveASREngine = value
	case config.KeyActiveASRModel:
		a.cfg.ActiveASRModel = value
	case config.KeyActiveTTSEngine:
		a.cfg.ActiveTTSEngine = value
	case config.KeyActiveTTSModel:
		a.cfg.ActiveTTSModel = value
	case config.KeyTTSVoice:
		a.cfg.TTSVoice = value
	case config.KeyActiveOCREngine:
		a.cfg.ActiveOCREngine = value
	case config.KeyActiveOCRModel:
		a.cfg.ActiveOCRModel = value
	case config.KeyVoicePersonality:
		a.cfg.VoicePersonality = value
	case config.KeyFuncChatVoiceEngine:
		a.cfg.FuncChatVoiceEngine = value
	case config.KeyFuncChatVoiceModel:
		a.cfg.FuncChatVoiceModel = value
	case config.KeyFuncChatEngine:
		a.cfg.FuncChatEngine = value
	case config.KeyFuncChatModel:
		a.cfg.FuncChatModel = value
	case config.KeyFuncNovelEngine:
		a.cfg.FuncNovelEngine = value
	case config.KeyFuncNovelModel:
		a.cfg.FuncNovelModel = value
	case config.KeyFuncOfficeEngine:
		a.cfg.FuncOfficeEngine = value
	case config.KeyFuncOfficeModel:
		a.cfg.FuncOfficeModel = value
	case config.KeyFuncGaeaEngine:
		a.cfg.FuncGaeaEngine = value
	case config.KeyFuncGaeaModel:
		a.cfg.FuncGaeaModel = value
	case config.KeyFuncCharLibEngine:
		a.cfg.FuncCharLibEngine = value
	case config.KeyFuncCharLibModel:
		a.cfg.FuncCharLibModel = value
	case config.KeyFuncRoutineEngine:
		a.cfg.FuncRoutineEngine = value
	case config.KeyFuncRoutineModel:
		a.cfg.FuncRoutineModel = value
	case config.KeyUsdCnyRate:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			a.cfg.UsdCnyRate = f
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（浮点解析失败）", "key", key, "value", value)
		}
	case config.KeyCosyVoiceDir:
		a.cfg.CosyVoiceDir = value
	case config.KeyCosyVoicePort:
		if n, err := strconv.Atoi(value); err == nil {
			a.cfg.CosyVoicePort = n
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（整数解析失败）", "key", key, "value", value)
		}
	// 布尔开关（*bool 在盘上，内存为 bool）
	case config.KeyFuncChatEnabled:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.FuncChatEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyFuncNovelEnabled:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.FuncNovelEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyFuncOfficeEnabled:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.FuncOfficeEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyFuncGaeaEnabled:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.FuncGaeaEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyFuncCharLibEnabled:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.FuncCharLibEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyFuncRoutineEnabled:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.FuncRoutineEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeySensitiveLocal:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.SensitiveLocal = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyKeepWarm:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.KeepWarmEnabled = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	case config.KeyAutoPreload:
		if b, err := strconv.ParseBool(value); err == nil {
			a.cfg.AutoPreload = b
		} else {
			slog.Warn("SaveConfig: 内存同步跳过（布尔解析失败）", "key", key, "value", value)
		}
	default:
		// 无内存对应项（或未来新增键）：已成功写盘，标注需重启生效。
		slog.Warn("SaveConfig: 配置项已持久化，无内存同步项（重启后生效）", "key", key, "value", value)
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
	re := regexp.MustCompile(`^\d+[a-z]?\.md$`)
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
