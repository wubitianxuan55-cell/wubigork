package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gaea/gaea/internal/types"
)

// ── 章节计划闭环落盘（规格 docs/gaea-longform-novel-system-2026-09.md §7.1-2 / §7.4）──
//
// 两个文件、同一套三态语义：
//   - 缺失 = 空正常态（返回空结构 + nil error / bool=false），新项目不是错误；
//   - 损坏 = 返回 error 且**绝不覆盖**原文件（损坏档可能是作者手改或半写，
//     顺手覆盖即永久丢数据；修复入口留给作者）；
//   - 写入 = 原子替换（复用本包既有 writeJSON → writeFileAtomic，与
//     characters.json/outline.json/章节同款语义：同目录临时文件 + fsync +
//     fileutil.RenameWithRetry，失败保留旧文件）。
//
// 路径口径沿用 Manager 既有目录字段，不自行拼绝对路径：
//   - plans.json 与 chapters/NNN.md 同目录（ChapterPath 的 chapters/）；
//   - 偏差与 analysis/annotations/MMM.json 同根同补零口径（%03d.json）。

// ChapterPlanVersion plans.json 当前契约版本（types.ChapterPlanFile.Version）。
const ChapterPlanVersion = 1

// ChapterPlansPath 章节计划表路径 chapters/plans.json
// （key 用十进制章节号字符串，与 chapters/NNN.md 命名口径一致）。
func (m *Manager) ChapterPlansPath() string {
	return filepath.Join(m.Dir, "chapters", "plans.json")
}

// chapterPlanKey 章节号 → 计划表键（十进制字符串；不做补零——
// 契约 §plan_v1.go:13 明确 key 是十进制章节号字符串，补零会让 `12` 与 `012`
// 分裂成两条计划）。
func chapterPlanKey(num int) string {
	return strconv.Itoa(num)
}

// emptyChapterPlans 空计划表（非 nil map，调用方可直接 range/写）。
func emptyChapterPlans() *types.ChapterPlanFile {
	return &types.ChapterPlanFile{
		Version: ChapterPlanVersion,
		Plans:   map[string]types.ChapterPlan{},
	}
}

// ReadChapterPlans 读章节计划表 chapters/plans.json。
//
// 文件缺失 → 空结构 + nil error（正常态）；JSON 损坏 → error 且原文件原封不动。
// 读侧同样补默认（Plans 为 nil 视作空表、Version<=0 补 1），避免旧档/手写档
// 让调用方踩 nil map；Version>0 时原样返回，不吞未来版本的版本号。
func (m *Manager) ReadChapterPlans() (*types.ChapterPlanFile, error) {
	path := m.ChapterPlansPath()
	f, err := loadJSON[types.ChapterPlanFile](path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyChapterPlans(), nil
		}
		return nil, err
	}
	if f == nil {
		return emptyChapterPlans(), nil
	}
	if f.Plans == nil {
		f.Plans = map[string]types.ChapterPlan{}
	}
	if f.Version <= 0 {
		f.Version = ChapterPlanVersion
	}
	return f, nil
}

// WriteChapterPlans 原子写章节计划表 chapters/plans.json。
//
// 自动补 Version=1（Version<=0 时）；f 为 nil 或 Plans 为 nil 均视为空表写入
// （避免 json null 让读取方拿到 nil map）。入参不被修改（内部拷贝后落盘）。
// chapters 目录不存在时自动创建，便于在 Manager 未经 Create 的场景使用。
func (m *Manager) WriteChapterPlans(f *types.ChapterPlanFile) error {
	out := types.ChapterPlanFile{}
	if f != nil {
		out = *f
	}
	if out.Version <= 0 {
		out.Version = ChapterPlanVersion
	}
	if out.Plans == nil {
		out.Plans = map[string]types.ChapterPlan{}
	}
	path := m.ChapterPlansPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建 chapters 目录失败: %w", err)
	}
	return writeJSON(path, &out)
}

// ReadChapterPlan 读单章计划。存在与否用 bool 表达（不靠 error）：
//   - 文件缺失 / 该章未登记 → (nil, false, nil)；
//   - 该章已登记 → (计划, true, nil)；
//   - 文件损坏 / IO 失败 → (nil, false, err)。
func (m *Manager) ReadChapterPlan(num int) (*types.ChapterPlan, bool, error) {
	f, err := m.ReadChapterPlans()
	if err != nil {
		return nil, false, err
	}
	plan, ok := f.Plans[chapterPlanKey(num)]
	if !ok {
		return nil, false, nil
	}
	return &plan, true, nil
}

// PlanDeviationPath 计划偏差文件路径 analysis/plan-deviation/<MMM>.json
// （三位补零章号，与 analysis/annotations/<MMM>.json 同口径）。
func (m *Manager) PlanDeviationPath(num int) string {
	return filepath.Join(m.Dir, "analysis", "plan-deviation", fmt.Sprintf("%03d.json", num))
}

// ReadPlanDeviation 读单章计划偏差。缺失 = (nil, false, nil) 正常态（未分析）；
// 损坏 = error 且原文件原封不动。
func (m *Manager) ReadPlanDeviation(num int) (*types.PlanDeviation, bool, error) {
	path := m.PlanDeviationPath(num)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("读取计划偏差失败 (%s): %w", path, err)
	}
	var pd types.PlanDeviation
	if err := json.Unmarshal(raw, &pd); err != nil {
		return nil, false, fmt.Errorf("解析计划偏差失败 (%s): %w", path, err)
	}
	return &pd, true, nil
}

// WritePlanDeviation 原子写单章计划偏差（目录不存在自动创建）。
// 章号取自 pd.ChapterNum（文件路径由它决定）；nil 或章号 <1 直接报错，
// 不写出 000.json 之类无主档。
func (m *Manager) WritePlanDeviation(pd *types.PlanDeviation) error {
	if pd == nil {
		return fmt.Errorf("计划偏差为空")
	}
	if pd.ChapterNum < 1 {
		return fmt.Errorf("计划偏差章号非法: %d", pd.ChapterNum)
	}
	path := m.PlanDeviationPath(pd.ChapterNum)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("创建计划偏差目录失败: %w", err)
	}
	return writeJSON(path, pd)
}
