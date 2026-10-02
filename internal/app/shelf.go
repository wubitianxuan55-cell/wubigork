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
	"github.com/gaea/gaea/internal/maturecraft"
	"github.com/gaea/gaea/internal/types"
)

// chapterNameRe 是章节文件名正则（刀E v4.249 普查#20：包级）。
var chapterNameRe = regexp.MustCompile(`^\d+[a-z]?\.md$`)

// ProjectCard 书架上的项目卡片数据（返回给前端）
type ProjectCard struct {
	Title        string `json:"title"`
	Genre        string `json:"genre"`
	Style        string `json:"style"`
	Path         string `json:"path"` // 项目完整路径
	WordCount    int    `json:"word_count"`
	ChapterCount int    `json:"chapter_count"`
	CreatedAt    string `json:"created_at"`       // ISO8601
	LastOpenedAt string `json:"last_opened_at"`   // ISO8601
	Mature       string `json:"mature,omitempty"` // v4.443 成人向档位（sensual/explicit，白名单归一后；空=非成人向）
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
			Mature:       maturecraft.NormalizeLevel(meta.Mature), // v4.443 徽标链；坏值不上墙
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
		_ = a.closePM()
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
//
// AP2-05（god-func 大拆试水第一刀）：原 63 case 巨型 switch（int/float/bool
// 三种「strconv 解析+slog.Warn」样板各重复 5/4/10 遍）表驱动化——键→内存
// setter 见 cfgMemSetters；书架键（novels_dir）按审计要求留在本文件特判。
func (a *App) SaveConfig(key, value string) error {
	if err := config.Save(key, value); err != nil {
		return err
	}

	// 书架特判键（本文件保留的唯一配置键）：如果当前打开了旧目录下的项目，
	// 先关闭再换目录。
	if key == config.KeyNovelsDir {
		if a.cfg.NovelsDir != value {
			if pm := a.getPM(); pm != nil {
				_ = a.closePM()
			}
		}
		a.cfg.NovelsDir = value
		return nil
	}

	set, ok := cfgMemSetters[key]
	if !ok {
		// 无内存对应项（或未来新增键）：已成功写盘，标注需重启生效。
		slog.Warn("SaveConfig: 配置项已持久化，无内存同步项（重启后生效）", "key", key, "value", value)
		return nil
	}

	// 更新内存中的对应字段（config.Save 已校验值格式，这里解析失败仅记录
	// 并跳过——磁盘已是权威来源，不阻断）。
	if err := set(a.cfg, value); err != nil {
		skip, ok := err.(*memParseSkipError)
		if !ok {
			return err
		}
		slog.Warn(skip.msg, "key", key, "value", value)
	}
	return nil
}

// ── SaveConfig 内存同步表驱动域（AP2-05）────────────────────────────
//
// 「key → 内存 setter」注册表，与 internal/config 的 saveSetters（盘上
// configFile 写入域）成对：那边管落盘与盘上校验，这边管内存态 config.Config
// 同步；两表键集刻意不必一致——盘有内存无的键（glm_api_key、func_sin_*、
// realtime_* 等）走 SaveConfig 的「无内存同步项」Warn 分支，现状如此
// （saveconfig_keys_test.go TestSaveConfig_DiskOnlyKeysWarnNoSync 钉死）。
//
// 迁移落点备忘：本域为自包含块（1 表 + 4 构造器 + 3 哨兵），后续如整体迁出
// shelf.go（独立 settings 内存同步文件或 config 包）可整块剪贴零改动搬家；
// 书架键 novels_dir 不入表，永久留在 SaveConfig 特判（关旧项目副作用）。

// memParseSkipError 内存同步解析失败信号：只用于触发「跳过」Warn，不外抛给
// 调用方（msg 与原 switch 文案逐字一致）。
type memParseSkipError struct{ msg string }

func (e *memParseSkipError) Error() string { return e.msg }

// 三类解析失败哨兵（文案逐字保留原 switch）。
var (
	errMemIntSkip   = &memParseSkipError{msg: "SaveConfig: 内存同步跳过（整数解析失败）"}
	errMemFloatSkip = &memParseSkipError{msg: "SaveConfig: 内存同步跳过（浮点解析失败）"}
	errMemBoolSkip  = &memParseSkipError{msg: "SaveConfig: 内存同步跳过（布尔解析失败）"}
)

// setStr/setInt/setFloat/setBool setter 构造器：键→字段的绑定收在表注册处，
// int/float/bool 的「解析失败→哨兵」与告警文案由 SaveConfig 单点处理
// （原先三种样板各手写 5/4/10 遍）。
func setStr(assign func(*config.Config, string)) func(*config.Config, string) error {
	return func(cfg *config.Config, v string) error {
		assign(cfg, v)
		return nil
	}
}

func setInt(assign func(*config.Config, int)) func(*config.Config, string) error {
	return func(cfg *config.Config, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil {
			return errMemIntSkip
		}
		assign(cfg, n)
		return nil
	}
}

func setFloat(assign func(*config.Config, float64)) func(*config.Config, string) error {
	return func(cfg *config.Config, v string) error {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return errMemFloatSkip
		}
		assign(cfg, f)
		return nil
	}
}

func setBool(assign func(*config.Config, bool)) func(*config.Config, string) error {
	return func(cfg *config.Config, v string) error {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return errMemBoolSkip
		}
		assign(cfg, b)
		return nil
	}
}

// cfgMemSetters SaveConfig 配置键 → 内存 setter 注册表（62 键；novels_dir
// 见 SaveConfig 特判）。model 经 SetModelMem（并发安全写，modelMu）。
var cfgMemSetters = map[string]func(*config.Config, string) error{
	config.KeyXaiClientID:         setStr(func(cfg *config.Config, v string) { cfg.XaiClientID = v }),
	config.KeyModel:               setStr(func(cfg *config.Config, v string) { config.SetModelMem(cfg, v) }),
	config.KeyHTTPTimeoutSeconds:  setInt(func(cfg *config.Config, n int) { cfg.HTTPTimeoutSeconds = n }),
	config.KeyDefaultTemperature:  setFloat(func(cfg *config.Config, f float64) { cfg.DefaultTemperature = f }),
	config.KeyAnalysisTemperature: setFloat(func(cfg *config.Config, f float64) { cfg.AnalysisTemperature = f }),
	config.KeyReasoningEffort:     setStr(func(cfg *config.Config, v string) { cfg.ReasoningEffort = v }),
	config.KeyQualityThreshold:    setInt(func(cfg *config.Config, n int) { cfg.QualityThreshold = n }),
	config.KeyQualityMaxRetries:   setInt(func(cfg *config.Config, n int) { cfg.QualityMaxRetries = n }),
	config.KeyTTSBinaryPath:       setStr(func(cfg *config.Config, v string) { cfg.TTSBinaryPath = v }),
	config.KeyTTSModelPath:        setStr(func(cfg *config.Config, v string) { cfg.TTSModelPath = v }),
	config.KeyTTSPort:             setInt(func(cfg *config.Config, n int) { cfg.TTSPort = n }),
	config.KeyTTSBackend:          setStr(func(cfg *config.Config, v string) { cfg.TTSBackend = v }),
	config.KeyTTSSpeed:            setFloat(func(cfg *config.Config, f float64) { cfg.TTSSpeed = f }),
	config.KeyImageBackend:        setStr(func(cfg *config.Config, v string) { cfg.ImageBackend = v }),
	config.KeyComfyUIURL:          setStr(func(cfg *config.Config, v string) { cfg.ComfyUIURL = v }),
	config.KeyImageSaveDir:        setStr(func(cfg *config.Config, v string) { cfg.ImageSaveDir = v }),
	config.KeyImageModel:          setStr(func(cfg *config.Config, v string) { cfg.ImageModel = v }),
	config.KeyPortraitBackend:     setStr(func(cfg *config.Config, v string) { cfg.PortraitBackend = v }),
	config.KeyPortraitModel:       setStr(func(cfg *config.Config, v string) { cfg.PortraitModel = v }),
	config.KeySinImageBackend:     setStr(func(cfg *config.Config, v string) { cfg.SinImageBackend = v }),
	config.KeySinImageModel:       setStr(func(cfg *config.Config, v string) { cfg.SinImageModel = v }),
	config.KeyComfyUIPath:         setStr(func(cfg *config.Config, v string) { cfg.ComfyUIPath = v }),
	config.KeyComfyUIPythonPath:   setStr(func(cfg *config.Config, v string) { cfg.ComfyUIPythonPath = v }),
	config.KeyActiveEngineID:      setStr(func(cfg *config.Config, v string) { cfg.ActiveEngineID = v }),
	config.KeyDeepseekAPIKey:      setStr(func(cfg *config.Config, v string) { cfg.DeepseekAPIKey = v }),
	config.KeyOpencodeGoAPIKey:    setStr(func(cfg *config.Config, v string) { cfg.OpenCodeGoAPIKey = v }),
	config.KeyOpencodeZenAPIKey:   setStr(func(cfg *config.Config, v string) { cfg.OpenCodeZenAPIKey = v }),
	config.KeyActiveASREngine:     setStr(func(cfg *config.Config, v string) { cfg.ActiveASREngine = v }),
	config.KeyActiveASRModel:      setStr(func(cfg *config.Config, v string) { cfg.ActiveASRModel = v }),
	config.KeyActiveTTSEngine:     setStr(func(cfg *config.Config, v string) { cfg.ActiveTTSEngine = v }),
	config.KeyActiveTTSModel:      setStr(func(cfg *config.Config, v string) { cfg.ActiveTTSModel = v }),
	config.KeyTTSVoice:            setStr(func(cfg *config.Config, v string) { cfg.TTSVoice = v }),
	config.KeyActiveOCREngine:     setStr(func(cfg *config.Config, v string) { cfg.ActiveOCREngine = v }),
	config.KeyActiveOCRModel:      setStr(func(cfg *config.Config, v string) { cfg.ActiveOCRModel = v }),
	config.KeyVoicePersonality:    setStr(func(cfg *config.Config, v string) { cfg.VoicePersonality = v }),
	config.KeyFuncChatVoiceEngine: setStr(func(cfg *config.Config, v string) { cfg.FuncChatVoiceEngine = v }),
	config.KeyFuncChatVoiceModel:  setStr(func(cfg *config.Config, v string) { cfg.FuncChatVoiceModel = v }),
	config.KeyFuncChatEngine:      setStr(func(cfg *config.Config, v string) { cfg.FuncChatEngine = v }),
	config.KeyFuncChatModel:       setStr(func(cfg *config.Config, v string) { cfg.FuncChatModel = v }),
	config.KeyFuncNovelEngine:     setStr(func(cfg *config.Config, v string) { cfg.FuncNovelEngine = v }),
	config.KeyFuncNovelModel:      setStr(func(cfg *config.Config, v string) { cfg.FuncNovelModel = v }),
	config.KeyFuncOfficeEngine:    setStr(func(cfg *config.Config, v string) { cfg.FuncOfficeEngine = v }),
	config.KeyFuncOfficeModel:     setStr(func(cfg *config.Config, v string) { cfg.FuncOfficeModel = v }),
	config.KeyFuncGaeaEngine:      setStr(func(cfg *config.Config, v string) { cfg.FuncGaeaEngine = v }),
	config.KeyFuncGaeaModel:       setStr(func(cfg *config.Config, v string) { cfg.FuncGaeaModel = v }),
	config.KeyFuncCharLibEngine:   setStr(func(cfg *config.Config, v string) { cfg.FuncCharLibEngine = v }),
	config.KeyFuncCharLibModel:    setStr(func(cfg *config.Config, v string) { cfg.FuncCharLibModel = v }),
	config.KeyFuncRoutineEngine:   setStr(func(cfg *config.Config, v string) { cfg.FuncRoutineEngine = v }),
	config.KeyFuncRoutineModel:    setStr(func(cfg *config.Config, v string) { cfg.FuncRoutineModel = v }),
	config.KeyUsdCnyRate:          setFloat(func(cfg *config.Config, f float64) { cfg.UsdCnyRate = f }),
	config.KeyCosyVoiceDir:        setStr(func(cfg *config.Config, v string) { cfg.CosyVoiceDir = v }),
	config.KeyCosyVoicePort:       setInt(func(cfg *config.Config, n int) { cfg.CosyVoicePort = n }),
	// 布尔开关（*bool 在盘上，内存为 bool）
	config.KeyFuncChatEnabled:    setBool(func(cfg *config.Config, b bool) { cfg.FuncChatEnabled = b }),
	config.KeyFuncNovelEnabled:   setBool(func(cfg *config.Config, b bool) { cfg.FuncNovelEnabled = b }),
	config.KeyFuncOfficeEnabled:  setBool(func(cfg *config.Config, b bool) { cfg.FuncOfficeEnabled = b }),
	config.KeyFuncGaeaEnabled:    setBool(func(cfg *config.Config, b bool) { cfg.FuncGaeaEnabled = b }),
	config.KeyFuncCharLibEnabled: setBool(func(cfg *config.Config, b bool) { cfg.FuncCharLibEnabled = b }),
	config.KeyFuncRoutineEnabled: setBool(func(cfg *config.Config, b bool) { cfg.FuncRoutineEnabled = b }),
	config.KeySensitiveLocal:     setBool(func(cfg *config.Config, b bool) { cfg.SensitiveLocal = b }),
	config.KeyOfflineMode:        setBool(func(cfg *config.Config, b bool) { cfg.OfflineMode = b }),
	config.KeyKeepWarm:           setBool(func(cfg *config.Config, b bool) { cfg.KeepWarmEnabled = b }),
	config.KeyAutoPreload:        setBool(func(cfg *config.Config, b bool) { cfg.AutoPreload = b }),
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
		Mature       string    `json:"mature"`
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
		Mature:       partial.Mature,
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
	re := chapterNameRe
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
