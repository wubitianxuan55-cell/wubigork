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
	"github.com/gaea/gaea/internal/gaea/genui"
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
	// readSkill 是实例化 read_skill 的解析闭包（刀2）：filter 收口 + 统计。
	readSkill func(name string) (string, error)
}

// buildSystemPrompt assembles the L1 identity block: base system prompt +
// output style + language policy + persistent memory + skills index. It also
// scans the project profile and initialises the runtime context layer.
//
// space 是装配空间的配置生效值（v4.5.1a 红线补课：""=space.mode=off 平铺形态
// 走空值全量=旧行为；"work"/"play"=系统提示词记忆索引只在该空间读——会话画像
// 与记忆索引不再跨空间泄露，兑现 S1.2 读端硬隔离的注入侧承诺）。
// memMode 是会话运行模式（2026-11-25 办公工具梳理批·记忆按模式隔离）：非空时
// 记忆索引/检索/逐轮召回只含「通用 + 本模式」的记忆，""=模式维度未启用（现状）。
// 晨报预载与项目本体注入两块同批停注——它们与记忆索引逐条重复，同一批记忆一轮
// 曾列四遍；现在系统提示词里只留索引一处，正文另由逐轮召回按相关度注入。
// skillFilter/skillExtras（2026-11-15 刀2，图文板块技能子集）：filter 非空时
// 索引/slash/run_skill/ctrl.Skills 面只留放行名单；extras 追加宿主内嵌技能
// （appended before filter——extras 必须自证通过 filter）。read_skill 解析器
// 同步收口（同 filter），经 syspromptOut.readSkill 交 boot 注册实例化工具
// （替换全局 builtin 同名件——双引擎下全局 WireReadSkillResolver 后建者覆盖
// 会跨引擎串仓，实例化是刀2 的结构性修复）。
// modeLine（2026-11-21 办公运行模式，众神殿 pan_1795202763988 终案）：非空时
// 在技能索引之后追加固定文本「当前模式 + 本档未载入能力组 + 切档指引」
// （DSH-r5 裁决：ToolHide 与模式提示必须同批上线，防能力无声消失）。空 =
// 零注入、现状逐字节。
func buildSystemPrompt(cfg *config.Config, cwd, space, memMode string, stderrPath io.Writer, onSkillUse func(name string, ok bool), skillFilter func(name string) bool, skillExtras []skill.Skill, memUserDir string, systemPromptOverride string, modeLine string) (*syspromptOut, error) {
	// 板块身份覆写（2026-11-16 图文身份分板）：非空顶替全局配置解析——图文
	// 引擎传自己的身份，不再吃办公 system_prompt。空 = 现状逐字节。
	var sysPrompt string
	if s := strings.TrimSpace(systemPromptOverride); s != "" {
		sysPrompt = s
	} else {
		var err error
		sysPrompt, err = cfg.ResolveSystemPrompt()
		if err != nil {
			return nil, err
		}
	}
	// 部署级提示词首拼接（刀14，dsh-context promptPrefix 蒸馏）：非空时置于
	// 所有会话系统提示词最前（outputstyle/语言策略/记忆/技能索引之前）。
	if prefix := strings.TrimSpace(cfg.Agent.PromptPrefix); prefix != "" {
		sysPrompt = prefix + "\n\n" + sysPrompt
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if st, ok := outputstyle.Resolve(cfg.Agent.OutputStyle, outputstyle.Dirs()); ok {
		sysPrompt = outputstyle.Apply(sysPrompt, st)
	}
	sysPrompt += "\n\n" + config.LanguagePolicy
	sysPrompt += "\n\n" + genui.OfficePointer

	// memUserDir 由 Build 一处算好传入（2026-11-16 记忆左右脑批）：非空覆盖值
	// =板块脑目录（docs 发现/文件后端/ScopeUser 写/SQLite 后端全随目录；DB 随
	// 目录推导分池），空值=config.MemoryUserDir() 现状逐字节。
	userDir := memUserDir
	gdb := db.GetDatabase(userDir)
	if _, err := memory.MigrateLegacyFileMemory(userDir, gdb); err != nil {
		log.Printf("[hephaestus] 办公记忆迁移失败: %v", err)
	}
	// 记忆发现（项目级 AGENTS.md/文档索引）基于工作区根，桌面端传入的是
	// 用户选定的工作空间（opts.Cwd），而不是进程启动目录。
	// v4.5.1a：Store 经 InSpace 视图收窄读端——work 会话的画像/索引只含 work
	// 记忆，play 只含 play；space.mode=off 时 space="" 不过滤（旧行为零变化）。
	// 2026-11-25：再叠 InMode 视图按运行模式收窄（memMode 空=不过滤）。
	mem := memory.Load(memory.Options{CWD: cwd, UserDir: userDir, DB: gdb, Space: space, Mode: memMode})
	// 记忆开关：关闭时不把画像/文档索引折进系统提示词（磁盘记忆保留，
	// 面板仍可管理；重新开启后下一次引擎重建即恢复注入）。
	if cfg.Memory.Enabled {
		sysPrompt = memory.Compose(sysPrompt, mem)
		if mem.Empty() {
			memory.InitDefaults(mem)
		}
	}
	// 固化正文注入（v4.378，activation 二维：pinned=正文随会话快照装配，
	// relevant=仅检索可达——reasonix memory/activation.go 蒸馏）：固化事实
	// 正文预算化进入缓存稳定前缀，「固化」动作有功能后果。仅门控记忆总开关
	// （两空间各自的空间收窄视图都注入——activation 与 scope 正交）；排序
	// Name 升序保证 touch 不翻动前缀；空集合零注入。
	if cfg.Memory.Enabled {
		res := memory.BuildPinnedFacts(mem.Store.List(), 0)
		if res.Block != "" {
			sysPrompt = strings.TrimRight(sysPrompt, "\n") + "\n\n" + res.Block
		}
		// 固化块发射面留给召回侧去重（众神殿 2026-11-28 论证 action-2/4）：
		// 去重依据跟「实际注入的前缀」走——本 Set 即 controller 持有的实例，
		// 会话中途重载换 Set 时由 finishMemoryReload 携带本状态。
		mem.SetPinnedFactsState(memory.PinnedFactsState{
			Emitted:   res.Emitted,
			Truncated: res.Truncated,
			Omitted:   res.Omitted,
		})
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
	// 可见集合唯一装配点（2026-11-15 众神殿首刀）：合并+同名去重（store-first）
	// + filter 收口一次完成——技能索引/read_skill/run_skill 三面共享同一可见
	// 集合（run_skill 经 Options 分流取 sp.skills，见 Build 内 run_skill 构造位）。
	skills := skill.VisibleSkills(skillStore.List(), skillExtras, skillFilter)
	sysPrompt = skill.ApplyIndex(sysPrompt, skills)
	// 运行模式行（2026-11-21 终案）：紧随技能索引——模式告知在能力清单旁边
	// 语义最顺（本档未载入的能力组与切档指引同段）；PromptSuffix 之前，不碰
	// 部署级「最后」语义。空串零注入（standard 缺省配置不传）。
	if line := strings.TrimSpace(modeLine); line != "" {
		sysPrompt = strings.TrimRight(sysPrompt, "\n") + "\n\n" + line
	}

	// read_skill 解析器（实例化，经 syspromptOut.readSkill 交 boot 注册；
	// 2026-11-15 刀2 起不再走全局 WireReadSkillResolver——双引擎串仓缝）。
	readSkillResolve := func(name string) (string, error) {
		if !toolAllowed(skillFilter, name) {
			if onSkillUse != nil {
				onSkillUse(name, false)
			}
			return "", fmt.Errorf("skill %q not found", name)
		}
		for _, sk := range skills {
			if sk.Name == name {
				if onSkillUse != nil {
					onSkillUse(name, true)
				}
				return sk.Body, nil
			}
		}
		if onSkillUse != nil {
			onSkillUse(name, false)
		}
		return "", fmt.Errorf("skill %q not found", name)
	}

	// 部署级提示词尾拼接（刀14）：非空时置于全部装配块（记忆/晨报/项目/
	// 固化/固定资料/技能索引）之后，保证「最后」语义；空串零注入。
	if suffix := strings.TrimSpace(cfg.Agent.PromptSuffix); suffix != "" {
		sysPrompt = strings.TrimRight(sysPrompt, "\n") + "\n\n" + suffix
	}

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
		readSkill:  readSkillResolve,
	}, nil
}
