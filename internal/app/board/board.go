package board

// AppHost 板块装配宿主——由 internal/app 的 *App 实现。
//
// 说明：doc §5.2 原文是 Init(a *App) error；这里以 board 包内的窄接口 AppHost
// 承接同一语义，避免 board ↔ app 包级循环依赖（internal/app 需要导入 board
// 做 manifest 装配，board 便不能再反向导入 internal/app）。当前 Step 2 的
// canonical 板块全部为声明式装配（Init 空实现），未来需要宿主能力的板块
// 通过扩展本接口暴露能力面。
type AppHost interface{}

// Board 板块接口（§5.2）：板块 = manifest 声明 + 装配单元。
// ID/Name/Icon/PageKey/Bindings/Intents/Tools 提供 manifest 数据源；
// Init 只做装配、不写业务（§5.4：每板块 Init 只做装配不写业务）。
type Board interface {
	ID() string
	Name() string
	Icon() string
	PageKey() string
	Bindings() []string
	Intents() []string
	Tools() []string
	Init(host AppHost) error
}

// staticBoard 以 Manifest 为单一数据源的通用板块实现：canonical 板块全部是
// 声明式（manifest 即实现），不需要各自的装配逻辑；有特殊装配的板块可在
// builtins 中覆写 Init（未来扩展点）。
type staticBoard struct {
	manifest Manifest
}

func (b *staticBoard) ID() string       { return b.manifest.ID }
func (b *staticBoard) Name() string     { return b.manifest.Label }
func (b *staticBoard) Icon() string     { return b.manifest.Icon }
func (b *staticBoard) PageKey() string  { return b.manifest.Page }
func (b *staticBoard) Bindings() []string { return b.manifest.Bindings }
func (b *staticBoard) Intents() []string {
	return b.manifest.IntentIDs()
}
func (b *staticBoard) Tools() []string { return b.manifest.Tools }

// Init 声明式板块的装配为空操作（§5.4：只做装配不写业务；canonical 板块
// 的装配在 App.Startup 既有顺序中完成，Init 保留为板块级扩展点）。
func (b *staticBoard) Init(AppHost) error { return nil }

// boardFromManifest 把 manifest 包装为 Board 实例。
func boardFromManifest(m Manifest) Board {
	return &staticBoard{manifest: m}
}
