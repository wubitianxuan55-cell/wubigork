package app

// sin_engine_prompt.go — 原罪引擎系统提示词与记忆装配（独立 fork）。
//
// 隔离纪律：fork boot/sysprompt.go 的装配思路进 sin 命名空间，不调用也不修改
// boot 包任何私有件。与办公装配的差异：
//   - 基座 = sinSystemPrompt()（成人边界/玩法/工具纪律，锁同步测试在案），
//     不做 outputstyle / promptPrefix / genui 指针 / 语言策略 / 技能索引；
//   - 记忆 = 书工作区（cwd=书路径）+ play 空间收窄读端——同一套 facts 存储，
//     project slug=书路径，跨会话记忆由引擎记忆面免费获得；
//   - 晨报/项目本体（work 专属）与 pins 固化资料不注入。

import (
	"path/filepath"

	"github.com/gaea/gaea/internal/gaea/cache"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
	"github.com/gaea/gaea/internal/gaea/vision"
)

// sinPromptOut 提示词装配产物（对齐 boot.syspromptOut 的 sin 子集）。
type sinPromptOut struct {
	prompt     string
	mem        *memory.Set
	compiler   *cache.Compiler
	runtimeCtx *cache.RuntimeLayer
}

// sinBuildEnginePrompt 装配原罪引擎的系统提示词与书级记忆。
// space 恒 "play"（记忆读端收窄到 play facts；project slug=书路径）。
func sinBuildEnginePrompt(cfg *gaeaConfig.Config, bookPath string) (*sinPromptOut, error) {
	prompt := sinSystemPrompt()

	userDir := gaeaConfig.MemoryUserDir()
	gdb := db.GetDatabase(userDir)
	// 记忆读端按书路径 slug 收窄：书 = 工作区 = 记忆项目键（跨会话记忆锚）。
	mem := memory.Load(memory.Options{CWD: bookPath, UserDir: userDir, DB: gdb, Space: "play"})
	if cfg.Memory.Enabled {
		prompt = memory.Compose(prompt, mem)
		if mem.Empty() {
			memory.InitDefaults(mem)
		}
		// 固化事实正文随会话快照装配（与办公同语义，play 视图）。
		if block := memory.BuildPinnedFactsBlock(mem.Store.List(), 0); block != "" {
			prompt += "\n\n" + block
		}
	}

	// builtin 全局运行时缝：与办公 boot 同一批 setter（同配置幂等，读配置
	// 不改配置）。sin 的 web_search/识图等工具委托同链，必须先就位。
	builtin.SetMemorySearchIndex(mem.Search)
	builtin.SetSearchConfig(cfg.Search)
	builtin.SetSearchProxy(cfg.NetworkProxySpec())
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

	// L1 编译器（缓存稳定前缀）+ L2 运行时层（书目录画像）。sin 不需要技能层
	// 业务，但 ContextManager 需要 SkillLayer 占位实例（空技能=零注入）。
	projectProfile := &cache.Profile{}
	projectProfile.Scan(bookPath)
	compiler := cache.New(prompt, nil)
	runtimeCtx := cache.NewRuntimeLayer()
	runtimeCtx.SetProject(cache.ProjectState{
		Language:     projectProfile.Language,
		Module:       projectProfile.Module,
		EntryPoints:  projectProfile.EntryPoints,
		TopDirs:      projectProfile.TopDirs,
		TotalFiles:   projectProfile.TotalFiles,
		Dependencies: projectProfile.Dependencies,
		RootPath:     filepath.Base(bookPath),
	})
	runtimeCtx.SetCompactL2(true)

	return &sinPromptOut{
		prompt:     prompt,
		mem:        mem,
		compiler:   compiler,
		runtimeCtx: runtimeCtx,
	}, nil
}

// sinEngineAssembleInput 引擎回合输入富化：在用户指令前拼角色块+底稿块。
// 与 legacy buildSinUserPrompt 的差异：历史窗口段不需要——引擎会话自带完整
// 历史与压缩；底稿/选角键直接传书 slug（绑定故事）或会话 id（未绑存量）。
// 各块为空时零注入，逐字节对齐 legacy 空态行为。
func (a *App) sinEngineAssembleInput(storeKey, input string) string {
	out := ""
	if cast := a.sinCastCharacters(storeKey); len(cast) > 0 {
		out += sinCastBlock(cast) + "\n\n"
	}
	doc := a.sinDraftForPrompt(storeKey)
	if doc.Outline != "" || len(doc.Notes) > 0 {
		out += sinDraftBlock(doc) + "\n\n"
	}
	return out + input
}
