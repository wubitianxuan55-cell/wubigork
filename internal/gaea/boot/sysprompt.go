package boot

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/cache"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/outputstyle"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
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
func buildSystemPrompt(cfg *config.Config, cwd string, stderrPath io.Writer) (*syspromptOut, error) {
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
	mem := memory.Load(memory.Options{CWD: cwd, UserDir: userDir, DB: gdb})
	sysPrompt = memory.Compose(sysPrompt, mem)
	builtin.SetMemorySearchIndex(mem.Search)
	builtin.SetSearchConfig(cfg.Search)
	builtin.SetSearchProxy(cfg.NetworkProxySpec())
	if mem.Empty() {
		memory.InitDefaults(mem)
	}

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
