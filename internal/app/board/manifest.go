package board

import (
	"fmt"
	"sort"
)

// Manifest 板块声明（§5.2「板块 Manifest」）——字段对齐 doc §5.2 的 TS schema
// BoardManifest（frontend/src/boards/types.ts 的目标形态）。Go 侧与前端共用
// 同一份 JSON 视图：GetBoardManifests() []Manifest 直接序列化给前端消费。
//
// 字段语义（缺省值由前端消费方应用）：
//   - keepAlive 默认 true（visitedPages display:none 保活）；显式声明用指针。
//   - inMenu 默认 true；settings 声明 false（右上按钮入口，不进菜单）。
//   - intents 声明板块的模块注册表意图：ID 是 Dispatch 的 intent，Handler 是
//     App 上提供该意图的绑定方法名（gen_bindings 的 manifest 覆盖层与
//     initModules 的处理器解析共用）。
type Manifest struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Page  string `json:"page"`
	Lazy  bool   `json:"lazy"`
	// Space 板块空间归属（S2.1 双空间壳，docs/gaea-space-shell-design.md §3）：
	// ""（缺省，按 work 兼容）| "work" 工位 | "play" 乐园 | "shared" 共用 |
	// "independent" 独立窗口（编程 DSH——不进工位/乐园导航，单独入口）。
	// 前端导航/启动器/快捷键按 space 过滤（shared 两空间均可达；independent 两空间均不出现）。
	Space        string          `json:"space,omitempty"`
	KeepAlive    *bool           `json:"keepAlive,omitempty"`
	Layout       string          `json:"layout,omitempty"`
	Shortcut     string          `json:"shortcut,omitempty"`
	MenuOrder    int             `json:"menuOrder,omitempty"`
	InMenu       *bool           `json:"inMenu,omitempty"`
	Breadcrumb   *BreadcrumbSpec `json:"breadcrumb,omitempty"`
	IsHome       bool            `json:"isHome,omitempty"`
	Nav          *NavSpec        `json:"nav,omitempty"`
	FeatureModel string          `json:"featureModel,omitempty"`
	Bindings     []string        `json:"bindings,omitempty"`
	Intents      []IntentDecl    `json:"intents,omitempty"`
	Tools        []string        `json:"tools,omitempty"`
}

// 板块空间归属常量（S2.1，与前端 boards/space.ts 的 BoardSpace 同构）。
const (
	SpaceWork        = "work"
	SpacePlay        = "play"
	SpaceShared      = "shared"
	SpaceIndependent = "independent"
)

// BreadcrumbSpec 面包屑锚点语义（附 B #8：novel 声明自己是项目锚点）。
type BreadcrumbSpec struct {
	AnchorTo string `json:"anchorTo,omitempty"`
}

// NavSpec 板块内子导航（NovelPage 6 tab / SettingsPage 9 分类等）。
type NavSpec struct {
	Children []NavChild `json:"children,omitempty"`
}

// NavChild 子导航项。
type NavChild struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Page  string `json:"page,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

// IntentDecl 意图声明：ID 是模块注册表意图（intent），Handler 是 App 上提供
// 该意图的绑定方法名。Handler 供 gen_bindings（manifest 覆盖层）与启动自检
// （intent→handler 存在性断言）使用。
type IntentDecl struct {
	ID      string `json:"id"`
	Handler string `json:"handler"`
}

// Bool 返回 bool 指针（manifest 显式声明布尔字段用）。
func Bool(v bool) *bool { return &v }

// Validate 校验 manifest 完整性（启动自检 + 测试用）：
//   - 必填字段（ID/Label/Icon/Page）非空；
//   - 每个意图必须声明 handler（缺陷 2 的机器保证：intent 无 handler 直接报错）；
//   - layout 仅允许 full/padded（空视为默认 padded）。
func (m *Manifest) Validate() error {
	switch {
	case m.ID == "":
		return fmt.Errorf("manifest: id 为空")
	case m.Label == "":
		return fmt.Errorf("manifest %q: label 为空", m.ID)
	case m.Icon == "":
		return fmt.Errorf("manifest %q: icon 为空", m.ID)
	case m.Page == "" && m.ID != "weixin": // weixin 服务板块无前端页面（beta）
		return fmt.Errorf("manifest %q: page 为空", m.ID)
	case m.Layout != "" && m.Layout != "full" && m.Layout != "padded":
		return fmt.Errorf("manifest %q: layout %q 非法（仅 full|padded）", m.ID, m.Layout)
	case m.Space != "" && m.Space != SpaceWork && m.Space != SpacePlay && m.Space != SpaceShared && m.Space != SpaceIndependent:
		return fmt.Errorf("manifest %q: space %q 非法（仅 work|play|shared|independent 或空）", m.ID, m.Space)
	}
	seen := map[string]bool{}
	for _, it := range m.Intents {
		if it.ID == "" {
			return fmt.Errorf("manifest %q: intent id 为空", m.ID)
		}
		if seen[it.ID] {
			return fmt.Errorf("manifest %q: intent %q 重复声明", m.ID, it.ID)
		}
		seen[it.ID] = true
		if it.Handler == "" {
			return fmt.Errorf("manifest %q: intent %q 无 handler（缺陷 2 防复发）", m.ID, it.ID)
		}
	}
	return nil
}

// ValidateAll 校验整批 manifest：必填字段 + 重复 id 拒绝（canonical 集内部
// 不允许重复；层间覆盖的「后写者胜」由覆盖层负责，不在此合并）。
func ValidateAll(manifests []Manifest) error {
	ids := make(map[string]string, len(manifests))
	for _, m := range manifests {
		if err := m.Validate(); err != nil {
			return err
		}
		if prev, dup := ids[m.ID]; dup {
			return fmt.Errorf("manifest 重复 id %q（先前声明于板块 %q）", m.ID, prev)
		}
		ids[m.ID] = m.Label
	}
	return nil
}

// IntentIDs 返回 manifest 声明的意图 id 列表（注册表填充用，保持声明顺序）。
func (m *Manifest) IntentIDs() []string {
	if len(m.Intents) == 0 {
		return nil
	}
	out := make([]string, 0, len(m.Intents))
	for _, it := range m.Intents {
		out = append(out, it.ID)
	}
	return out
}

// SortedIDs 返回排序后的板块 id 列表（测试/诊断断言用）。
func SortedIDs(manifests []Manifest) []string {
	out := make([]string, 0, len(manifests))
	for _, m := range manifests {
		out = append(out, m.ID)
	}
	sort.Strings(out)
	return out
}
