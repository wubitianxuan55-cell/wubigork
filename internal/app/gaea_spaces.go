package app

// S4 双空间绑定面（设计 docs/gaea-space-dimension-design.md §5/§6）：
//   - gaeaEffectiveSpace：产物路径分区取「当前生效空间」的单点
//     （space.mode=off → "" = 整体回退 work 现状路径）；
//   - GaeaSpaceList / GaeaSpaceActive / GaeaSpaceActivate：空间静态枚举、
//     当前生效空间、切换持久化。挂 CoreB（gen_bindings explicitOverrides
//     显式映射 core，设计 §6）。

import (
	"fmt"
	"path/filepath"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// SpaceOption 是空间静态枚举项（GaeaSpaceList 返回，work/play 固定两值）。
type SpaceOption struct {
	ID    string `json:"id"`    // "work" | "play"
	Title string `json:"title"` // 展示名
	Desc  string `json:"desc"`  // 一句话说明
}

// SpaceActiveView 是当前生效空间视图（GaeaSpaceActive / GaeaSpaceActivate 返回）。
type SpaceActiveView struct {
	// Space 是当前生效空间："work"/"play"。space.mode=off 时分区整体关闭，
	// 所有路径回退 work 现状，此处恒报 work（ModeOn=false 标记关闭态）。
	Space string `json:"space"`
	// ModeOn 报告 space.mode 分区开关（缺省 on；显式 "off" 关闭）。
	ModeOn bool `json:"modeOn"`
	// ExportsDir / WorkDir 是当前生效的产物/过程目录（工作区相对，slash 形态），
	// 供前端展示落点；work 恒为现状路径 .gaea/exports、.gaea/work。
	ExportsDir string `json:"exportsDir"`
	WorkDir    string `json:"workDir"`
}

// gaeaCfgSnapshot 返回当前办公引擎配置快照（gaeaCtrl 同款锁模式：短临界区
// 取指针，调用方在锁外读字段）。未初始化返回 nil。
func gaeaCfgSnapshot() *gaeaConfig.Config {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	return ga.cfg
}

// gaeaEffectiveSpace 返回产物路径分区使用的当前生效空间（S4 写死点统一取法）：
//   - space.mode=on → session.space 归一值（"work"/"play"）；
//   - space.mode=off → ""（spaces.ExportsDir/WorkDir 对 "" 恒回 work 现状路径，
//     即整体回退语义）；
//   - 引擎未初始化（ga.cfg==nil）→ ""（= work 缺省，行为与改造前一致）。
func gaeaEffectiveSpace() string {
	if cfg := gaeaCfgSnapshot(); cfg != nil {
		return cfg.EffectiveSessionSpace()
	}
	return ""
}

// gaeaSetSessionSpace 把 session.space 写入内存配置并持久化到用户配置文件
// （引擎未初始化时先加载）。引擎重建由调用方决定——绑定面当前只做配置写
// （设计 §6「先实现配置写」）：gaeaBuildController 每次构建都会重新读取该
// 配置，下次引擎重建/重启后新会话目录即指向新空间。
func gaeaSetSessionSpace(space string) error {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.cfg == nil {
		cfg, err := gaeaLoadConfig()
		if err != nil {
			return err
		}
		ga.cfg = cfg
	}
	ga.cfg.Session.Space = space
	return gaeaConfig.Save(ga.cfg)
}

// gaeaSpaceActiveView 组装当前生效空间视图（磁盘配置兜底：引擎未初始化时
// 读取持久化配置，避免「激活了 play 但重启前查询仍报 work」的假象）。
func gaeaSpaceActiveView() SpaceActiveView {
	cfg := gaeaCfgSnapshot()
	if cfg == nil {
		if loaded, err := gaeaLoadConfig(); err == nil {
			cfg = loaded
		}
	}
	modeOn := cfg == nil || cfg.SpaceModeIsOn()
	space := spaces.SpaceWork
	if cfg != nil && modeOn {
		space = cfg.SessionSpace()
	}
	cwd := gaeaCwd()
	exportsRel, _ := filepath.Rel(cwd, spaces.ExportsDir(cwd, space))
	workRel, _ := filepath.Rel(cwd, spaces.WorkDir(cwd, space))
	return SpaceActiveView{
		Space:      space,
		ModeOn:     modeOn,
		ExportsDir: filepath.ToSlash(exportsRel),
		WorkDir:    filepath.ToSlash(workRel),
	}
}

// GaeaSpaceList 返回双空间静态枚举（work/play，顺序稳定）。
func (a *App) GaeaSpaceList() []SpaceOption {
	return []SpaceOption{
		{ID: spaces.SpaceWork, Title: "办公空间", Desc: "交付物落 .gaea/exports（现状路径，兼容既有产物链接）"},
		{ID: spaces.SpacePlay, Title: "娱乐空间", Desc: "交付物落 .gaea/play/exports（轻语/聊天等娱乐域分区）"},
	}
}

// GaeaSpaceActive 返回当前生效空间（含分区开关与产物/过程目录落点）。
func (a *App) GaeaSpaceActive() SpaceActiveView {
	return gaeaSpaceActiveView()
}

// GaeaSpaceActivate 持久化当前空间（session.space 配置键）并返回新生效空间。
// 非法空间（非 work/play，区分大小写）直接拒绝。生效时机：gaeaBuildController
// 每次构建读取 EffectiveSessionSpace——下次引擎重建/重启后，新会话目录与产物
// 路径即指向新空间；运行中的会话与当前引擎不受影响（设计 §6：本步不做前端
// 切换 UI 大改，静默重建会让当前会话视图与后端失步，故不在此处重建）。
func (a *App) GaeaSpaceActivate(space string) (SpaceActiveView, error) {
	if !spaces.Valid(space) {
		return SpaceActiveView{}, fmt.Errorf("非法空间 %q（仅 work|play）", space)
	}
	if err := gaeaSetSessionSpace(space); err != nil {
		return SpaceActiveView{}, fmt.Errorf("持久化空间配置失败: %w", err)
	}
	return gaeaSpaceActiveView(), nil
}
