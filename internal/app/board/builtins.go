package board

// builtins 注册 canonical 板块（§3.1 清单 + 决策 D7）。
//
// canonical 集合 = §3.1 的 9 业务板块 + 编程板块（code，DeepSeek Harness Web
// 进程管理）+ knowledge（决策 D7：恢复挂载为独立板块 = 3.0.0 最小 manifest 试点）。
// home 是壳（启动器，不进 manifest 业务清单，见 §3.1 注；前端 manifests.ts 的
// 静态清单含 home 属壳层数据）。
//
// Page 字段 = 前端 PageRegistry 的页面组件 key（与 frontend/src/boards/
// pageRegistry.ts 的 registerPage key 一致，如 'ChatPage'）。weixin 无前端
// 页面（§3.1 beta 服务板块），Page 留空。
//
// 每个板块的 Bindings 对齐 §3.1 绑定门面列；Intents 声明主脑模块意图
// （Handler = App 方法名，gen_bindings manifest 覆盖层 + 启动自检共用）。

// builtinManifests canonical 板块 manifest（顺序即菜单顺序；GetBoardManifests
// 直接返回其拷贝，前端 MainLayout/ModuleLauncher 数据驱动）。
var builtinManifests = []Manifest{
	{
		ID: "chat", Label: "聊天", Icon: "MessageOutlined",
		Page: "ChatPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "full", Shortcut: "ctrl+1",
		MenuOrder: 1, InMenu: Bool(true),
		FeatureModel: "chat",
		Bindings:     []string{"VoiceB", "ChatB"}, // §3.1：ChatB + VoiceB(whisper 部分)
		Intents:      []IntentDecl{{ID: "chat", Handler: "WhisperChat"}},
	},
	{
		ID: "novel", Label: "小说", Icon: "ReadOutlined",
		Page: "NovelPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "padded", Shortcut: "ctrl+2",
		MenuOrder: 2, InMenu: Bool(true),
		Breadcrumb: &BreadcrumbSpec{AnchorTo: "project"}, // 附 B #8：项目名→novel 锚点
		Nav: &NavSpec{Children: []NavChild{
			{ID: "home", Label: "书架"}, {ID: "novelsetting", Label: "设定"},
			{ID: "character", Label: "角色"}, {ID: "create", Label: "创作"},
			{ID: "chapter", Label: "阅读"}, {ID: "export", Label: "导出"},
		}},
		FeatureModel: "novel",
		Bindings:     []string{"NovelB"},
		Intents:      []IntentDecl{{ID: "create_chapter", Handler: "CreateChapter"}},
	},
	{
		ID: "imagegen", Label: "绘梦", Icon: "PictureOutlined",
		Page: "ImageGenPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "padded", Shortcut: "ctrl+3",
		MenuOrder: 3, InMenu: Bool(true),
		FeatureModel: "imagegen",
		Bindings:     []string{"ImageB"},
		Intents:      []IntentDecl{{ID: "generate", Handler: "GenerateFreeImage"}},
	},
	{
		ID: "gaea", Label: "办公", Icon: "ToolOutlined",
		Page: "GaeaPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "full", Shortcut: "ctrl+4",
		MenuOrder: 4, InMenu: Bool(true),
		FeatureModel: "gaea",
		Bindings:     []string{"OfficeB", "MemoryB", "CostB"}, // §3.1：OfficeB + MemoryB + CostB
		Intents: []IntentDecl{
			{ID: "chat", Handler: "ChatGeneral"}, // 主脑 gaea 域
			{ID: "create", Handler: "GaeaSend"},  // D8：office.create→GaeaSend（office 模块保留）
		},
	},
	{
		ID: "code", Label: "编程", Icon: "CodeOutlined",
		Page: "ProgrammingPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "full", // 桌面内嵌 Harness Web 工作台（全出血）
		MenuOrder: 5, InMenu: Bool(true),
		Bindings: []string{"CoreB"}, // dsh web 进程管理（Get/Start/StopProgrammingWeb）
	},
	{
		ID: "memoryhub", Label: "记忆中枢", Icon: "DatabaseOutlined",
		Page: "MemoryHubPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "padded",
		MenuOrder: 6, InMenu: Bool(true),
		Nav: &NavSpec{Children: []NavChild{
			{ID: "knowledge", Label: "知识库"}, {ID: "cost", Label: "成本库"},
			{ID: "profile", Label: "用户画像"}, {ID: "office", Label: "办公记忆"},
			{ID: "materials", Label: "项目资料"}, {ID: "whisper", Label: "聊天记忆"},
			{ID: "graph", Label: "记忆图谱"}, {ID: "digitallife", Label: "数字生命"},
		}},
		Bindings: []string{"MemoryB", "CostB"}, // 8 库聚合（§3.1）
	},
	{
		ID: "modelcenter", Label: "模型中心", Icon: "ApiOutlined",
		Page: "ModelCenterPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "padded",
		MenuOrder: 7, InMenu: Bool(true),
		Bindings: []string{"ModelB"},
	},
	{
		ID: "characterlib", Label: "角色库", Icon: "TeamOutlined",
		Page: "CharacterLibraryPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "padded",
		MenuOrder: 8, InMenu: Bool(true),
		FeatureModel: "characterlib",
		Bindings:     []string{"CharlibB"},
	},
	{
		ID: "settings", Label: "设置", Icon: "SettingOutlined",
		Page: "SettingsPage", Lazy: true,
		KeepAlive: Bool(true), Layout: "padded",
		MenuOrder: 9, InMenu: Bool(false), // 附 B #12：不进菜单，右上角按钮入口
		Nav: &NavSpec{Children: []NavChild{
			{ID: "general", Label: "通用"}, {ID: "chat", Label: "聊天"},
			{ID: "novel", Label: "小说"}, {ID: "imagegen", Label: "绘梦"},
			{ID: "office", Label: "办公"}, {ID: "model", Label: "模型"},
			{ID: "security", Label: "安全"}, {ID: "data", Label: "数据"},
			{ID: "about", Label: "关于"},
		}},
		Bindings: []string{"CoreB"}, // §3.1：CoreB(部分)
	},
	{
		ID: "weixin", Label: "微信助手", Icon: "WechatOutlined",
		Page:      "", // §3.1：无前端页面（beta），Channels/weixin 服务板块
		Lazy:      false,
		KeepAlive: Bool(true), Layout: "padded",
		MenuOrder: 10, InMenu: Bool(false),
	},
	{
		ID: "knowledge", Label: "知识库", Icon: "BookOutlined",
		Page: "KnowledgePage", Lazy: true, // D7：恢复挂载为独立板块（3.0.0 最小 manifest 试点；§3.1 孤儿页）
		KeepAlive: Bool(true), Layout: "padded",
		MenuOrder: 8, InMenu: Bool(true),
		FeatureModel: "knowledge",
		Bindings:     []string{"MemoryB"}, // 知识库能力经记忆域门面（§3.1 建议）
	},
}

// BuiltinManifests 返回 canonical 板块 manifest 清单（拷贝，调用方可安全修改）。
// 供 GetBoardManifests（前端/审计）与启动自检使用。
func BuiltinManifests() []Manifest {
	out := make([]Manifest, len(builtinManifests))
	copy(out, builtinManifests)
	return out
}

// Builtins 返回 canonical 板块实例清单（模块注册表 manifest 驱动装配用）。
// 每个实例都以 builtinManifests 为单一数据源；Init 为声明式空装配。
func Builtins() []Board {
	manifests := BuiltinManifests()
	out := make([]Board, 0, len(manifests))
	for i := range manifests {
		m := manifests[i] // 拷贝，避免闭包捕获循环变量
		out = append(out, boardFromManifest(m))
	}
	return out
}

// CanonicalIDs 返回 canonical 业务板块的 id（验收断言用；不含 knowledge——
// knowledge 是 D7 独立板块，见 BuiltinManifests）。
func CanonicalIDs() []string {
	return []string{"chat", "novel", "imagegen", "gaea", "code", "memoryhub", "modelcenter", "characterlib", "settings", "weixin"}
}
