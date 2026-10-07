package workcost

// entrycache.go —— cost_entries.price 的核算缓存化（工料法目标③）。
//
// 用户定调：综合单价是**核算输出**，不是输入。但成本库的 price 列是 20+ 个
// 既有消费方（PriceBand 分位 / contentband 含量对标 / matchindex / 检索 /
// 图谱 / 归因对标）的读取面，改列名等于全面返工。
//
// 折中（对齐 SchemaV27）：**保留 price 列与对外契约不变，把语义降级为
// 「最近一次核算结果缓存」**，并加 price_derived 标记（1=由工料机核算而来）。
// 于是：
//   - 有工料机组成的条目 → price 重算为 Σ(含量×单价×(1+损耗))，price_derived=1；
//   - 无组成的条目（纯资源价）→ 一个字不动，price_derived=0（旧语义）。
//
// 包依赖纪律：本包**不 import cost**（cost 是纯资源包，反向依赖会成环——
// memo：retrieval→cost 已存在，cost→workcost 会让 cost 与 workcost 互相依赖）。
// 故对成本库的读写一律经 RecomposeHooks 注入，编排留在本包、实现留在调用方。

import (
	"fmt"
	"log/slog"
	"strings"
)

// RecomposeEntry 核算缓存化所需的条目视图（cost.Entry 的字段子集）。
type RecomposeEntry struct {
	Name         string
	Title        string
	Unit         string
	CategoryPath string
	// Components 工料机组成行（空 = 纯资源价条目，不参与重算）。
	Components []RecomposeComponent
}

// RecomposeComponent 组成行视图。
type RecomposeComponent struct {
	Kind     string
	Title    string
	Unit     string
	Quantity float64
	Price    float64
}

// RecomposeWrite 一条重算结果的写回载荷。
type RecomposeWrite struct {
	Name        string
	Price       float64
	LaborFee    float64
	MaterialFee float64
	MachineFee  float64
}

// RecomposeHooks 成本库读写注入缝（实现由 app 层提供，接 cost.Store）。
type RecomposeHooks struct {
	// Names 返回全部条目名（顺序不限）。
	Names func() ([]string, error)
	// Load 按名读取条目（含组成行）。
	Load func(name string) (*RecomposeEntry, error)
	// Write 写回重算结果（实现须同时置 price_derived=1）。
	Write func(w RecomposeWrite) error
}

// RecomposeResult 核算缓存化结果。
type RecomposeResult struct {
	Scanned   int      `json:"scanned"`   // 扫描到的条目总数
	WithComp  int      `json:"withComp"`  // 带工料机组成的条目数
	Updated   int      `json:"updated"`   // 单价被重算且发生变化的条目数
	Unchanged int      `json:"unchanged"` // 重算结果与原价一致的条目数
	Errors    []string `json:"errors"`
}

// ResolveComponentKind 判定一条组成行归属工料机哪一类。
//
// 优先级：
//  1. 行自身 kind 非空且归一后是四类之一 → 直接采信；
//  2. 标题关键词命中外委/机械/人工 → 按命中；
//  3. 该条目的 Unit 暗示是纯资源（工日/台班）→ 按暗示；
//  4. 兜底：材料。
//
// 第 1 条的关键细节：**空 kind 不能走归一**——normalizeKind("") 会返回「材料」
// （成本库空类别默认材料），若据此判定「已标注」，后面所有关键词分支都会被
// 短路成材料（实测踩到：挖掘机台班被归成材料）。
func ResolveComponentKind(compKind, compTitle, entryUnit, entryCategoryPath string) string {
	if strings.TrimSpace(compKind) != "" {
		if k := normalizeKind(compKind); isKnownKind(k) {
			return k
		}
	}
	// 标题只看**强信号**（外委/人工/机械关键词）。刻意不用 ClassifyKind 的整体
	// 判定——它有无条件「兜底材料」，会把「工日」这类单位暗示永远短路掉。
	if k := kindByStrongKeywords(compTitle); k != "" {
		return k
	}
	if strings.TrimSpace(entryUnit) != "" {
		if k := normalizeKind(entryUnit); isKnownKind(k) {
			return k
		}
	}
	_ = entryCategoryPath
	return KindMaterial
}

// kindByStrongKeywords 只看强关键词判类别（无命中返回空串，不兜底）。
// 外委优先——「水泥窑协同处置」既像材料又像外委，实测模版把它标为外委。
func kindByStrongKeywords(title string) string {
	if containsAny(title, outsourcedKeywords) {
		return KindOutsourced
	}
	if containsAny(title, laborKeywords) {
		return KindLabor
	}
	if containsAny(title, machineKeywords) {
		return KindMachine
	}
	return ""
}

// isKnownKind 判断是否为四类之一（normalizeKind 对未知写法原样返回）。
func isKnownKind(k string) bool {
	switch k {
	case KindLabor, KindMaterial, KindMachine, KindOutsourced:
		return true
	}
	return false
}

// ComponentsToLines 把成本库的人材机组成行转成核算组成行。
func ComponentsToLines(entryUnit, entryCategoryPath string, comps []RecomposeComponent) []ComposeLine {
	lines := make([]ComposeLine, 0, len(comps))
	for _, c := range comps {
		lines = append(lines, ComposeLine{
			Kind:     ResolveComponentKind(c.Kind, c.Title, entryUnit, entryCategoryPath),
			Title:    c.Title,
			Unit:     c.Unit,
			Quantity: c.Quantity,
			Price:    c.Price,
		})
	}
	return lines
}

// RecomposeEntries 对成本库中带工料机组成的条目重算综合单价并写回缓存。
//
// 只动有组成的条目：无组成条目的 price 是人工/导入的当期价，重算会把它抹成
// 0（把「查到的价」降级成「算不出的价」），属净损失，故严格跳过。
//
// 实测用户库（1590 条）组成行 **0 条**，故对存量是 no-op；它是后续从项目
// 导入回填组成后「一键重算生成新综合单价」的执行端。
func RecomposeEntries(h RecomposeHooks) (RecomposeResult, error) {
	var res RecomposeResult
	if h.Names == nil || h.Load == nil || h.Write == nil {
		return res, fmt.Errorf("核算缓存化需要 Names/Load/Write 三个注入实现")
	}
	names, err := h.Names()
	if err != nil {
		return res, fmt.Errorf("读取成本条目名失败: %w", err)
	}
	res.Scanned = len(names)

	for _, name := range names {
		e, err := h.Load(name)
		if err != nil || e == nil {
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("读取条目 %q 失败: %v", name, err))
			}
			continue
		}
		if len(e.Components) == 0 {
			continue
		}
		res.WithComp++

		lines := ComponentsToLines(e.Unit, e.CategoryPath, e.Components)
		c := ComposeUnitPrice(lines)
		if c.Subtotal <= 0 {
			res.Errors = append(res.Errors, fmt.Sprintf(
				"条目「%s」的工料机组成核算为 0（%d 行），跳过不改价", titleOrName(e), len(e.Components)))
			continue
		}
		if err := h.Write(RecomposeWrite{
			Name:        e.Name,
			Price:       c.CompositePrice,
			LaborFee:    c.LaborFee,
			MaterialFee: c.MaterialFee,
			MachineFee:  c.MachineFee,
		}); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("条目「%s」写回失败: %v", titleOrName(e), err))
			continue
		}
		res.Updated++
	}
	slog.Info("工料法核算缓存化完成", "扫描", res.Scanned, "有组成", res.WithComp,
		"已重算", res.Updated, "错误", len(res.Errors))
	return res, nil
}

// titleOrName 取条目标题（空则回退名称），供日志/错误文案。
func titleOrName(e *RecomposeEntry) string {
	if t := strings.TrimSpace(e.Title); t != "" {
		return t
	}
	return e.Name
}
