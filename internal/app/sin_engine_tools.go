package app

// sin_engine_tools.go — 原罪引擎工具注册表（独立装配，办公注册面零触碰）。
//
// 隔离纪律：不进 spacetags/AllowsSpace/RegistryMap——那是办公的装配过滤面；
// sin 自建 registry，工具集合按「只服务图文创作」拍板裁剪：
//   - builtin 精选（confined 到书目录、只读面）：read_file / ls；
//   - shared 元工具：ask（结构化提问）、remember/forget/memory_get/
//     promote_session_facts（书级跨会话记忆——引擎记忆面免费获得）；
//   - sin 9 工具适配器（sinTool 接口与 tool.Tool 逐签名相同，直接内嵌），
//     存储键=contextKey（书 slug）；sin_export 引擎路径暂缺（聊天库导出
//     依赖 chat.db 话题，刀3 按会话日志重建后再入表）。
// 办公的 bash/edit/task/子代理/技能/大文件摘要工具全部不装。

import (
	"github.com/gaea/gaea/internal/gaea/agent"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// sinEngineBuiltinTools sin 引擎启用的 builtin 白名单（只读面；写文件走
// sin_chapter 与书项目结构化落盘，不给模型裸写盘能力）。
var sinEngineBuiltinTools = []string{"read_file", "ls"}

// sinEngineSinTools sin 9 工具中进入引擎注册表的清单（sinToolOrder 子集；
// sin_export 引擎路径未接，见文件头注释）。
var sinEngineSinTools = []string{
	sinToolWebSearch, sinToolWebFetch, sinToolBookSearch, sinToolBookDownload,
	sinToolCast, sinToolNotes, sinToolOutline, sinToolIllustrate,
}

// sinToolAdapter sinTool → tool.Tool 适配：两接口方法集逐签名相同，内嵌即满足。
type sinToolAdapter struct {
	sinTool
}

// sinEngineRegistry 装配 sin 引擎工具注册表（执行闸/钩子/路径策略由执行器
// 与派发器侧持有，注册表只管工具集合）。
func sinEngineRegistry(a *App, cfg *gaeaConfig.Config, bookPath, bookSlug string, mem *memory.Set) *tool.Registry {
	reg := tool.NewRegistry()

	// builtin 精选（写具 confined 到书目录的白名单只取只读两件）。
	ws := builtin.Workspace{
		Dir:       bookPath,
		WriteRoots: cfg.WriteRootsAt(bookPath),
		ProxySpec: cfg.NetworkProxySpec(),
	}
	for _, t := range ws.Tools(sinEngineBuiltinTools...) {
		reg.Add(t)
	}

	// shared 元工具：结构化提问 + 书级记忆四件（facts 存储已在 sinPromptOut
	// 按 书 slug/play 装配，remember 落库即书级跨会话记忆）。
	reg.Add(agent.NewAskTool())
	userDir := gaeaConfig.MemoryUserDir()
	reg.Add(memory.NewRememberTool(mem.Store, memory.NewProfileStore(db.GetDatabase(userDir))))
	reg.Add(memory.NewForgetTool(mem.Store))
	reg.Add(memory.NewPromoteSessionFactsTool())
	reg.Add(memory.NewMemoryGetTool(mem.Store))

	// sin 9 工具适配器（存储键=书 slug；contextKey 双轨见 sin_tools.go）。
	c := sinToolContext{app: a, contextKey: bookSlug}
	for _, kind := range sinEngineSinTools {
		if f, ok := sinToolRegistry[kind]; ok {
			reg.Add(sinToolAdapter{sinTool: f(c)})
		}
	}
	return reg
}
