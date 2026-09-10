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
	"strings"

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

// SpaceProfileView 是单个空间的装配 profile 视图（GaeaSpaceProfiles 返回，模型
// 中心「总闸/空间策略」分区消费，长期规划阶段二）：汇报 gaea.toml 里配了什么、
// 经既有链生效成什么。写路径=GaeaSpaceProfileSet（生效时机同为下次引擎重建）。
type SpaceProfileView struct {
	Space string `json:"space"` // "work" | "play"
	// Gaea 是办公 agent 功能域模型覆写原值（space_profiles.<space>.gaea，引用
	// GetFeatureModel 键体系；""=未配置=现状模型）。
	Gaea string `json:"gaea"`
	// GaeaResolved / GaeaOk 是覆写解析结果：Ok=true 时 Resolved 形如
	// "provider · model"；Gaea 非空且 Ok=false = 引用无法解析（boot 告警后
	// 现状模型继续生效——失败说人话，不让用户猜）。
	GaeaResolved string `json:"gaeaResolved"`
	GaeaOk       bool   `json:"gaeaOk"`
	// Models 是其余功能域覆写（仅非空项；键=chat/whisper/novel/office/
	// characterlib/routine）。
	Models map[string]string `json:"models"`
	// PermMode 是生效权限模式（PermissionsForSpace 既有链：顶层策略+空间段，
	// play 未配置段=产品默认 allow）。
	PermMode string `json:"permMode"`
	// PermHardAskCount / PermHardAskBySpace：生效强制审批工具数；BySpace=false
	// = 未按空间配置（control 用包级默认集）。
	PermHardAskCount   int  `json:"permHardAskCount"`
	PermHardAskBySpace bool `json:"permHardAskBySpace"`
	// GuardrailsOn 是 play 内容护栏生效态（PlayGuardrails 既有链，false=零钳制）。
	GuardrailsOn bool `json:"guardrailsOn"`
	// ModeOn 报告 space.mode 分区开关（false=空间维度整体关闭，策略不生效，
	// 全域回退 work 现状）。
	ModeOn bool `json:"modeOn"`
}

// buildSpaceProfileViews 组装双空间 profile 视图（纯函数，便于测试；顺序恒
// work→play）。cfg 为 nil 时返回零配置视图（modeOn=true 与 gaeaSpaceActiveView
// 未初始化口径一致）。
func buildSpaceProfileViews(cfg *gaeaConfig.Config) []SpaceProfileView {
	out := make([]SpaceProfileView, 0, 2)
	modeOn := cfg == nil || cfg.SpaceModeIsOn()
	for _, space := range []string{spaces.SpaceWork, spaces.SpacePlay} {
		v := SpaceProfileView{Space: space, ModeOn: modeOn}
		if cfg != nil {
			perm := cfg.PermissionsForSpace(space)
			v.PermMode = perm.Mode
			v.PermHardAskCount = len(perm.HardAsk)
			v.PermHardAskBySpace = perm.HardAsk != nil
			if prof, err := cfg.SpaceProfile(space); err == nil {
				v.Gaea = strings.TrimSpace(prof.Gaea)
				if v.Gaea != "" {
					if e, ok := cfg.ResolveModel(v.Gaea); ok && e != nil {
						v.GaeaOk = true
						v.GaeaResolved = strings.TrimSpace(e.Name) + " · " + strings.TrimSpace(e.Model)
					}
				}
				rest := map[string]string{}
				for k, ref := range map[string]string{
					"chat":         prof.Chat,
					"whisper":      prof.Whisper,
					"novel":        prof.Novel,
					"office":       prof.Office,
					"characterlib": prof.CharacterLib,
					"routine":      prof.Routine,
				} {
					if s := strings.TrimSpace(ref); s != "" {
						rest[k] = s
					}
				}
				if len(rest) > 0 {
					v.Models = rest
				}
			}
			v.GuardrailsOn = cfg.PlayGuardrails(space).Enabled
		}
		out = append(out, v)
	}
	return out
}

// GaeaSpaceProfiles 返回双空间装配 profile 视图（引擎未初始化时读盘兜底，
// gaeaSpaceActiveView 同款，避免「配置了但重启前查不到」的假空）。
func (a *App) GaeaSpaceProfiles() []SpaceProfileView {
	cfg := gaeaCfgSnapshot()
	if cfg == nil {
		if loaded, err := gaeaLoadConfig(); err == nil {
			cfg = loaded
		}
	}
	return buildSpaceProfileViews(cfg)
}

// spaceProfileKeys 是 GaeaSpaceProfileSet 可写的 profile 键 → 写入函数
// （gaea=办公 agent 功能域为总闸主控；其余功能域覆写同键体系一并放开）。
var spaceProfileKeys = map[string]func(*gaeaConfig.SpaceProfile, string){
	"chat":         func(p *gaeaConfig.SpaceProfile, v string) { p.Chat = v },
	"whisper":      func(p *gaeaConfig.SpaceProfile, v string) { p.Whisper = v },
	"novel":        func(p *gaeaConfig.SpaceProfile, v string) { p.Novel = v },
	"office":       func(p *gaeaConfig.SpaceProfile, v string) { p.Office = v },
	"gaea":         func(p *gaeaConfig.SpaceProfile, v string) { p.Gaea = v },
	"characterlib": func(p *gaeaConfig.SpaceProfile, v string) { p.CharacterLib = v },
	"routine":      func(p *gaeaConfig.SpaceProfile, v string) { p.Routine = v },
}

// applySpaceProfileEdit 校验并落一处空间 profile 编辑（纯函数，便于测试）：
// ref 空=清除该键（未配置语义）；非空必须经 ResolveModel 解析（与 boot 装配
// 同链）——解析不了的坏引用宁拒不写，错误消息带已配置 provider 清单（说人话）。
// 生效时机与 GaeaSpaceActivate 同口径：gaeaBuildController 下次构建/重启生效。
func applySpaceProfileEdit(cfg *gaeaConfig.Config, space, key, ref string) error {
	if !spaces.Valid(space) {
		return fmt.Errorf("非法空间 %q（仅 work|play）", space)
	}
	apply, ok := spaceProfileKeys[key]
	if !ok {
		return fmt.Errorf("未知 profile 键 %q（可写：gaea/chat/whisper/novel/office/characterlib/routine）", key)
	}
	ref = strings.TrimSpace(ref)
	if ref != "" {
		if _, ok := cfg.ResolveModel(ref); !ok {
			names := make([]string, 0, len(cfg.Providers))
			for _, p := range cfg.Providers {
				if n := strings.TrimSpace(p.Name); n != "" {
					names = append(names, n)
				}
			}
			return fmt.Errorf("无法解析 %q——坏引用不写入（写入只会让现状模型继续生效还骗人）。provider/model 形态，已配置 provider: %s", ref, strings.Join(names, ", "))
		}
	}
	prof := cfg.SpaceProfiles[space] // map 缺键取零值副本
	apply(&prof, ref)
	if cfg.SpaceProfiles == nil {
		cfg.SpaceProfiles = map[string]gaeaConfig.SpaceProfile{}
	}
	if ref == "" && prof.Permissions == nil && prof.Guardrails == nil && prof == (gaeaConfig.SpaceProfile{}) {
		delete(cfg.SpaceProfiles, space) // 全空段不落盘噪音
	} else {
		cfg.SpaceProfiles[space] = prof
	}
	return nil
}

// GaeaSpaceProfileSet 写一处空间 profile（space+key+ref；ref 空=清除），
// 返回刷新后的双空间视图。持久化到用户配置文件；生效时机=下次引擎重建/
// 重启（boot 装配世界，运行中引擎不动——与 GaeaSpaceActivate 同口径）。
func (a *App) GaeaSpaceProfileSet(space, key, ref string) ([]SpaceProfileView, error) {
	ga.mu.Lock()
	cfg := ga.cfg
	if cfg == nil {
		loaded, err := gaeaLoadConfig()
		if err != nil {
			ga.mu.Unlock()
			return nil, err
		}
		cfg = loaded
	}
	if err := applySpaceProfileEdit(cfg, space, key, ref); err != nil {
		ga.mu.Unlock()
		return nil, err
	}
	ga.cfg = cfg
	err := gaeaConfig.Save(cfg)
	ga.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("持久化空间配置失败: %w", err)
	}
	return buildSpaceProfileViews(cfg), nil
}
