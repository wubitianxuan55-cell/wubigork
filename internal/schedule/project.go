// schedule/project.go — 计划文件（.gsched.json）装载/保存/校验（v4.113.0 刀4）。
//
// 计划文件是板块与 agent 的共享资产：前端板块（GaeaScheduleLoad/Save）与
// agent 工具（schedule_get/apply/analyze）读写同一文件。保存一律先校验
// （fail-closed：环依赖/悬空引用/非法字段拒绝落盘），再临时文件+改名原子写。
package schedule

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultRelPath 默认计划文件（工作区相对路径）。多工程管理留后续刀次，
// v1 板块与 agent 缺省都指向它，保证「对话改的 = 板块打开的」。
const DefaultRelPath = "进度计划/当前计划.gsched.json"

// Load 装载计划文件并校验。文件不存在返回 os.ErrNotExist 包装错误（调用方
// 据此走「空计划/迁移」分支）。
func Load(path string) (Project, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Project{}, err
	}
	var p Project
	if err := json.Unmarshal(raw, &p); err != nil {
		return Project{}, fmt.Errorf("计划文件解析失败（%s）：%w", filepath.Base(path), err)
	}
	norm := NormalizeCalendar(p.Calendar)
	p.Calendar = &norm
	if err := Validate(&p); err != nil {
		return Project{}, fmt.Errorf("计划文件校验失败：%w", err)
	}
	return p, nil
}

// Save 校验后原子写（临时文件+改名；Windows 上 os.Rename 覆盖已存在目标安全）。
func Save(path string, p Project) error {
	if err := Validate(&p); err != nil {
		return err
	}
	if c := ComputeCpm(p.Tasks, p.Links); !c.OK {
		return fmt.Errorf("%s", c.Error)
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建目录失败：%w", err)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("写入临时文件失败：%w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("替换计划文件失败：%w", err)
	}
	return nil
}

// Validate 结构校验（fail-closed）：id 唯一非空、层级 0/1、搭接引用存在且
// 类型合法、工期非负、进度 0-100、日期口径合法。空任务表合法（空计划）。
func Validate(p *Project) error {
	seen := make(map[string]bool, len(p.Tasks))
	for _, t := range p.Tasks {
		if strings.TrimSpace(t.ID) == "" {
			return fmt.Errorf("存在空 id 任务")
		}
		if seen[t.ID] {
			return fmt.Errorf("任务 id 重复：%s", t.ID)
		}
		seen[t.ID] = true
		if t.Level != 0 && t.Level != 1 {
			return fmt.Errorf("任务 %s 层级非法（仅 0=分组/1=子任务）：%d", t.ID, t.Level)
		}
		if t.Duration < 0 {
			return fmt.Errorf("任务 %s 工期为负", t.ID)
		}
		if t.Progress < 0 || t.Progress > 100 {
			return fmt.Errorf("任务 %s 进度超出 0-100：%d", t.ID, t.Progress)
		}
		if t.Mode != "" && t.Mode != ModeAuto && t.Mode != ModeManual {
			return fmt.Errorf("任务 %s 模式非法：%s", t.ID, t.Mode)
		}
	}
	for _, l := range p.Links {
		if !seen[l.From] {
			return fmt.Errorf("搭接引用了不存在的任务：%s", l.From)
		}
		if !seen[l.To] {
			return fmt.Errorf("搭接引用了不存在的任务：%s", l.To)
		}
		switch l.Type {
		case FS, SS, FF, SF:
		default:
			return fmt.Errorf("搭接类型非法（%s→%s）：%s", l.From, l.To, l.Type)
		}
	}
	if p.StartDate != "" && len(p.StartDate) != 10 {
		return fmt.Errorf("开工日期口径应为 YYYY-MM-DD：%s", p.StartDate)
	}
	if p.Deadline != "" && len(p.Deadline) != 10 {
		return fmt.Errorf("目标竣工日期口径应为 YYYY-MM-DD：%s", p.Deadline)
	}
	return nil
}

// Analyze CPM 分析（含关键链与近关键清单）——schedule_analyze 与 apply
// 回执共用的叙事数据。
func (p Project) Analyze() (CpmResult, Analysis) {
	cpm := ComputeCpm(p.Tasks, p.Links)
	a := Analysis{Cpm: cpm}
	if !cpm.OK {
		return cpm, a
	}
	for i := range p.Tasks {
		t := p.Tasks[i]
		row := cpm.Rows[t.ID]
		if t.Level != 0 {
			a.LeafCount++
		}
		if t.Level != 0 && t.Mode != ModeManual {
			if row.Critical {
				a.Critical = append(a.Critical, t.Name)
			} else if row.TF <= 2 {
				a.NearCritical = append(a.NearCritical, NearCriticalTask{Name: t.Name, TF: row.TF})
			}
		}
		if t.IsMilestone {
			a.Milestones = append(a.Milestones, MilestoneInfo{Name: t.Name, Workday: row.ES})
		}
	}
	return cpm, a
}
