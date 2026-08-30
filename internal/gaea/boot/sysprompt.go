package boot

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cache"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/outputstyle"
	"github.com/gaea/gaea/internal/gaea/pins"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/gaea/vision"
)

// syspromptOut contains the artifacts produced by building the system prompt.
type syspromptOut struct {
	prompt     string
	mem        *memory.Set
	skills     []skill.Skill
	compiler   *cache.Compiler
	runtimeCtx *cache.RuntimeLayer
	store      *skill.Store
}

// buildSystemPrompt assembles the L1 identity block: base system prompt +
// output style + language policy + persistent memory + skills index. It also
// scans the project profile and initialises the runtime context layer.
//
// space 是装配空间的配置生效值（v4.5.1a 红线补课：""=space.mode=off 平铺形态
// 走空值全量=旧行为；"work"/"play"=系统提示词记忆索引只在该空间读——会话画像
// 与记忆索引不再跨空间泄露，兑现 S1.2 读端硬隔离的注入侧承诺）。
func buildSystemPrompt(cfg *config.Config, cwd, space string, stderrPath io.Writer) (*syspromptOut, error) {
	sysPrompt, err := cfg.ResolveSystemPrompt()
	if err != nil {
		return nil, err
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if st, ok := outputstyle.Resolve(cfg.Agent.OutputStyle, outputstyle.Dirs()); ok {
		sysPrompt = outputstyle.Apply(sysPrompt, st)
	}
	sysPrompt += "\n\n" + config.LanguagePolicy

	userDir := config.MemoryUserDir()
	gdb := db.GetDatabase(userDir)
	if _, err := memory.MigrateLegacyFileMemory(userDir, gdb); err != nil {
		log.Printf("[hephaestus] 办公记忆迁移失败: %v", err)
	}
	// 记忆发现（项目级 AGENTS.md/文档索引）基于工作区根，桌面端传入的是
	// 用户选定的工作空间（opts.Cwd），而不是进程启动目录。
	// v4.5.1a：Store 经 InSpace 视图收窄读端——work 会话的画像/索引只含 work
	// 记忆，play 只含 play；space.mode=off 时 space="" 不过滤（旧行为零变化）。
	mem := memory.Load(memory.Options{CWD: cwd, UserDir: userDir, DB: gdb, Space: space})
	// 记忆开关：关闭时不把画像/文档索引折进系统提示词（磁盘记忆保留，
	// 面板仍可管理；重新开启后下一次引擎重建即恢复注入）。
	if cfg.Memory.Enabled {
		sysPrompt = memory.Compose(sysPrompt, mem)
		if mem.Empty() {
			memory.InitDefaults(mem)
		}
	}
	// 常用资料（已固定）：工作区 .gaea/pinned.json 清单自动带入新会话，
	// 文本类附正文摘要、办公文档列名按需读取（装配而非灌输）。
	if block := pins.Block(cwd); block != "" {
		sysPrompt = strings.TrimRight(sysPrompt, "\n") + "\n\n" + block
	}
	builtin.SetMemorySearchIndex(mem.Search)
	builtin.SetSearchConfig(cfg.Search)
	builtin.SetSearchProxy(cfg.NetworkProxySpec())
	// 3.0 Step 3d Provider Seam：搜索/检索/视觉/文档转换后端由 gaea.toml 配置驱动，
	// 零值回落各自默认（本地 herdsman 兼容端点），切换后端只改配置、代码零改动。
	builtin.SetSearchEngineOrder(cfg.Search.EngineOrder)
	builtin.SetRetrievalRuntime(builtin.RetrievalRuntime{
		EmbedKind:     cfg.Retrieval.EmbedKind,
		EmbedBaseURL:  cfg.Retrieval.EmbedBaseURL,
		EmbedModel:    cfg.Retrieval.EmbedModel,
		RerankKind:    cfg.Retrieval.RerankKind,
		RerankBaseURL: cfg.Retrieval.RerankBaseURL,
		RerankModel:   cfg.Retrieval.RerankModel,
	})
	vision.SetVisionRuntime(vision.VisionRuntime{
		Kind:    cfg.Vision.Kind,
		BaseURL: cfg.Vision.BaseURL,
		Model:   cfg.Vision.Model,
	})
	builtin.SetMarkdownConverterRuntime(builtin.MarkdownConverterRuntime{
		Kind: cfg.MarkdownConverter.Kind,
	})

	skillStore := skill.New(skill.Options{ProjectRoot: cwd, CustomPaths: cfg.SkillCustomPaths(), Stderr: stderrPath})
	skills := skillStore.List()
	sysPrompt = skill.ApplyIndex(sysPrompt, skills)

	builtin.WireReadSkillResolver(func(name string) (string, error) {
		sk, ok := skillStore.Read(name)
		if !ok {
			return "", fmt.Errorf("skill %q not found", name)
		}
		return sk.Body, nil
	})

	projectProfile := &cache.Profile{}
	projectProfile.Scan(cwd)
	compiler := cache.New(sysPrompt, nil)

	runtimeCtx := cache.NewRuntimeLayer()
	runtimeCtx.SetProject(cache.ProjectState{
		Language:     projectProfile.Language,
		Module:       projectProfile.Module,
		EntryPoints:  projectProfile.EntryPoints,
		TopDirs:      projectProfile.TopDirs,
		TotalFiles:   projectProfile.TotalFiles,
		Dependencies: projectProfile.Dependencies,
		RootPath:     filepath.Base(cwd),
	})
	runtimeCtx.SetCompactL2(true)

	return &syspromptOut{
		prompt:     sysPrompt,
		mem:        mem,
		skills:     skills,
		compiler:   compiler,
		runtimeCtx: runtimeCtx,
		store:      skillStore,
	}, nil
}
