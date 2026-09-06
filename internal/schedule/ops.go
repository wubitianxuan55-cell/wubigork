// schedule/ops.go — 计划增量操作集（schedule_apply 的 ops 通道，v4.113.0 刀4）。
//
// 与 xlsxedit.Op 同范式：单结构体 + Type 判别 + 可选字段。全量替换走 project
// 通道；ops 用于对话式调整（「主体结构压 10 天」→ patch_task 等）。
// 应用后由调用方统一校验 + CPM fail-closed（环依赖拒绝落盘）。
package schedule

import (
	"fmt"
	"strings"
)

// patchTask 部分更新载荷（指针三态：nil=不动；Mode 指针区分「不改」与「改 auto」）。
type patchTask struct {
	Name        *string   `json:"name,omitempty"`
	Level       *int      `json:"level,omitempty"`
	Duration    *int      `json:"duration,omitempty"`
	Progress    *int      `json:"progress,omitempty"`
	IsMilestone *bool     `json:"isMilestone,omitempty"`
	Mode        *TaskMode `json:"mode,omitempty"`
	ManualStart *int      `json:"manualStart,omitempty"`
}

// opLink set_links 的单条前置关系。
type opLink struct {
	From string   `json:"from"`
	Type LinkType `json:"type,omitempty"`
	Lag  int      `json:"lag,omitempty"`
}

// Op 计划增量操作。
type Op struct {
	Type string `json:"type"` // upsert_task | patch_task | remove_task | set_links | set_meta | auto_chain | set_baseline | clear_baseline

	// upsert_task：整任务（含 id）；afterId 缺省追加表尾。
	AfterID string `json:"afterId,omitempty"`
	Task    *Task  `json:"task,omitempty"`

	// patch_task：部分更新载荷（指针三态：nil=不动）。
	Patch *patchTask `json:"patch,omitempty"`

	// patch_task / remove_task：目标任务 id。
	ID string `json:"id,omitempty"`

	// set_links：整体替换 toId 的入边（前端 setPreds 同语义）。
	ToID  string   `json:"toId,omitempty"`
	Links []opLink `json:"links,omitempty"`

	// set_meta：工程名/开工日期/日历。
	Name      string    `json:"name,omitempty"`
	StartDate string    `json:"startDate,omitempty"`
	Calendar  *Calendar `json:"calendar,omitempty"`

	// set_baseline：基线名（缺省「基线」）+保存时间（调用方标注，缺省拒绝）。
	BaselineName string `json:"baselineName,omitempty"`
	SavedAt      string `json:"savedAt,omitempty"`
}

// ApplyOps 依次应用操作集，返回人类可读摘要（供 Journal 与工具回执）。
// 单条失败即中止返回错误（fail-closed，不做部分应用）。
func ApplyOps(p *Project, ops []Op) ([]string, error) {
	summary := make([]string, 0, len(ops))
	for i, op := range ops {
		s, err := applyOne(p, op)
		if err != nil {
			return summary, fmt.Errorf("第 %d 条操作失败：%w", i+1, err)
		}
		summary = append(summary, s)
	}
	return summary, nil
}

func applyOne(p *Project, op Op) (string, error) {
	switch op.Type {
	case "upsert_task":
		t := op.Task
		if t == nil {
			return "", fmt.Errorf("upsert_task 缺少 task")
		}
		if strings.TrimSpace(t.ID) == "" {
			return "", fmt.Errorf("upsert_task 缺少任务 id")
		}
		if t.Level != 0 && t.Level != 1 {
			return "", fmt.Errorf("任务 %s 层级非法（仅 0/1）：%d", t.ID, t.Level)
		}
		if t.Progress < 0 || t.Progress > 100 {
			return "", fmt.Errorf("任务 %s 进度超出 0-100", t.ID)
		}
		at := len(p.Tasks)
		if op.AfterID != "" {
			idx := indexOfTask(p.Tasks, op.AfterID)
			if idx < 0 {
				return "", fmt.Errorf("afterId 不存在：%s", op.AfterID)
			}
			at = idx + 1
		}
		if idx := indexOfTask(p.Tasks, t.ID); idx >= 0 {
			p.Tasks[idx] = *t
			return fmt.Sprintf("更新任务「%s」", t.Name), nil
		}
		p.Tasks = append(p.Tasks[:at], append([]Task{*t}, p.Tasks[at:]...)...)
		return fmt.Sprintf("新增任务「%s」（工期 %d 工作日）", t.Name, t.Duration), nil

	case "patch_task":
		idx := indexOfTask(p.Tasks, op.ID)
		if idx < 0 {
			return "", fmt.Errorf("任务不存在：%s", op.ID)
		}
		t := &p.Tasks[idx]
		changes := make([]string, 0, 4)
		if op.Patch != nil {
			pt := op.Patch
			if pt.Name != nil {
				t.Name = *pt.Name
				changes = append(changes, fmt.Sprintf("改名「%s」", *pt.Name))
			}
			if pt.Duration != nil {
				if *pt.Duration < 0 {
					return "", fmt.Errorf("工期为负")
				}
				changes = append(changes, fmt.Sprintf("工期 %d→%d 工作日", t.Duration, *pt.Duration))
				t.Duration = *pt.Duration
			}
			if pt.Progress != nil {
				if *pt.Progress < 0 || *pt.Progress > 100 {
					return "", fmt.Errorf("进度超出 0-100")
				}
				changes = append(changes, fmt.Sprintf("进度 %d→%d%%", t.Progress, *pt.Progress))
				t.Progress = *pt.Progress
			}
			if pt.Level != nil {
				if *pt.Level != 0 && *pt.Level != 1 {
					return "", fmt.Errorf("层级非法（仅 0/1）")
				}
				t.Level = *pt.Level
				changes = append(changes, fmt.Sprintf("层级→%d", *pt.Level))
			}
			if pt.IsMilestone != nil {
				t.IsMilestone = *pt.IsMilestone
				if *pt.IsMilestone {
					changes = append(changes, "设为里程碑")
				}
			}
			if pt.Mode != nil && *pt.Mode != "" {
				if *pt.Mode != ModeAuto && *pt.Mode != ModeManual {
					return "", fmt.Errorf("模式非法：%s", *pt.Mode)
				}
				t.Mode = *pt.Mode
				changes = append(changes, fmt.Sprintf("模式→%s", *pt.Mode))
			}
			if pt.ManualStart != nil {
				t.ManualStart = *pt.ManualStart
				changes = append(changes, fmt.Sprintf("锁定开始→第 %d 工作日", *pt.ManualStart))
			}
		}
		if len(changes) == 0 {
			return "", fmt.Errorf("patch_task 未提供任何字段")
		}
		return fmt.Sprintf("调整「%s」：%s", t.Name, strings.Join(changes, "、")), nil

	case "remove_task":
		idx := indexOfTask(p.Tasks, op.ID)
		if idx < 0 {
			return "", fmt.Errorf("任务不存在：%s", op.ID)
		}
		kill := descendantIDs(p.Tasks, idx)
		remaining := make([]Task, 0, len(p.Tasks))
		for _, t := range p.Tasks {
			if !kill[t.ID] {
				remaining = append(remaining, t)
			}
		}
		name := p.Tasks[idx].Name
		links := make([]Link, 0, len(p.Links))
		for _, l := range p.Links {
			if !kill[l.From] && !kill[l.To] {
				links = append(links, l)
			}
		}
		removed := len(p.Tasks) - len(remaining)
		p.Tasks = remaining
		p.Links = links
		return fmt.Sprintf("移除「%s」及子孙共 %d 行", name, removed), nil

	case "set_links":
		idx := indexOfTask(p.Tasks, op.ToID)
		if idx < 0 {
			return "", fmt.Errorf("任务不存在：%s", op.ToID)
		}
		toName := p.Tasks[idx].Name
		kept := make([]Link, 0, len(p.Links))
		for _, l := range p.Links {
			if l.To != op.ToID {
				kept = append(kept, l)
			}
		}
		byID := make(map[string]bool, len(p.Tasks))
		for _, t := range p.Tasks {
			byID[t.ID] = true
		}
		for _, nl := range op.Links {
			if nl.From == op.ToID {
				return "", fmt.Errorf("任务不能以自身为前置")
			}
			if !byID[nl.From] {
				return "", fmt.Errorf("前置任务不存在：%s", nl.From)
			}
			typ := nl.Type
			if typ == "" {
				typ = FS
			}
			kept = append(kept, Link{From: nl.From, To: op.ToID, Type: typ, Lag: nl.Lag})
		}
		p.Links = kept
		return fmt.Sprintf("更新「%s」前置关系（共 %d 条）", toName, len(op.Links)), nil

	case "set_meta":
		changes := make([]string, 0, 3)
		if op.Name != "" {
			p.Name = op.Name
			changes = append(changes, fmt.Sprintf("工程名→「%s」", op.Name))
		}
		if op.StartDate != "" {
			if len(op.StartDate) != 10 {
				return "", fmt.Errorf("开工日期口径应为 YYYY-MM-DD：%s", op.StartDate)
			}
			p.StartDate = op.StartDate
			changes = append(changes, fmt.Sprintf("开工日期→%s", op.StartDate))
		}
		if op.Calendar != nil {
			p.Calendar = ptr(NormalizeCalendar(op.Calendar))
			changes = append(changes, "更新工作日历")
		}
		if len(changes) == 0 {
			return "", fmt.Errorf("set_meta 未提供任何字段")
		}
		return strings.Join(changes, "、"), nil

	case "auto_chain":
		// 推荐逻辑关系的确定性缺省：仅对【无前置】的叶任务按 WBS 顺序补
		// FS 串联（上一个叶任务→当前；全程首个叶不补）。已有逻辑/手动
		// 任务不动（手动=有意定位；并行例外由 AI 用 set_links 覆写）。
		hasIn := make(map[string]bool, len(p.Links))
		for _, l := range p.Links {
			hasIn[l.To] = true
		}
		added := 0
		lastLeaf := ""
		for i := range p.Tasks {
			t := p.Tasks[i]
			if t.Level == 0 {
				continue
			}
			if t.Mode == ModeManual {
				continue // 手动=有意定位：不补、也不作链源（manual 忽略入边，链它无意义）
			}
			if lastLeaf != "" && !hasIn[t.ID] {
				p.Links = append(p.Links, Link{From: lastLeaf, To: t.ID, Type: FS, Lag: 0})
				added++
			}
			lastLeaf = t.ID
		}
		if added == 0 {
			return "", fmt.Errorf("auto_chain 没有可补的任务（无前置的叶任务不足两个，或均已手动定位）")
		}
		return fmt.Sprintf("自动补全 %d 条缺省串联逻辑（仅无前置任务，FS lag=0）", added), nil

	case "set_baseline":
		// 基线快照：固化此刻排程（ops 序列中的当前位置），供后续漂移对比。
		// CPM 不过/无叶任务 fail-closed 拒绝；savedAt 必须由调用方标注
		// （引擎是纯函数不取时钟）。
		if strings.TrimSpace(op.SavedAt) == "" {
			return "", fmt.Errorf("set_baseline 缺少 savedAt（YYYY-MM-DD HH:mm，由工具层标注）")
		}
		b, err := SnapshotBaseline(p, op.SavedAt, op.BaselineName)
		if err != nil {
			return "", err
		}
		p.Baseline = b
		return fmt.Sprintf("保存基线「%s」（%d 项工作，总工期 %d 天）", b.Name, len(b.Rows), b.Duration), nil

	case "clear_baseline":
		if p.Baseline == nil {
			return "", fmt.Errorf("当前没有基线可清除")
		}
		p.Baseline = nil
		return "清除基线", nil

	default:
		return "", fmt.Errorf("不支持的操作类型：%s", op.Type)
	}
}

func indexOfTask(tasks []Task, id string) int {
	for i := range tasks {
		if tasks[i].ID == id {
			return i
		}
	}
	return -1
}

// descendantIDs 以扁平数组 level 语义取第 idx 行及其子孙 id 集合（前端
// descendantIds 同口径）。
func descendantIDs(tasks []Task, idx int) map[string]bool {
	out := map[string]bool{}
	if idx < 0 || idx >= len(tasks) {
		return out
	}
	base := tasks[idx].Level
	out[tasks[idx].ID] = true
	for i := idx + 1; i < len(tasks) && tasks[i].Level > base; i++ {
		out[tasks[i].ID] = true
	}
	return out
}

func ptr(c Calendar) *Calendar { return &c }
