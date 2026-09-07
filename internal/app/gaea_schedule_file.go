package app

// gaea_schedule_file.go — 进度计划板块文件持久化绑定（v4.113.0 刀4；
// v4.139 差距 #15 多工程刀1+刀2：索引指针+列表+新建/打开/归档/删除）。
//
// 计划文件（进度计划/*.gsched.json）是板块与 agent 的共享资产：
// 板块经本绑定读写（轻量 IO，不落证据卡——UI 自动保存不刷 Journal）；
// agent 经 schedule_get/apply/analyze 三工具读写（快照+证据卡在工具内）。
// 两侧校验/CPM 口径同源 internal/schedule，落盘一律 fail-closed。
//
// 多工程语义（docs/gaea-schedule-multi-project-design-2026-09.md §3.1/§3.3）：
// 索引 .gaea/schedule/index.json 只是缓存与指针，工程事实真相=目录扫描；
// Load/Save/agent 缺省 path 都解析同一 current 指针——「对话改的 = 板块打开的」
// 跨多工程保持；显式 rel/path 永远优先。

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/schedule"
)

// ScheduleLoadResult 板块装载结果：Project 为计划 JSON 串（前端再解析，
// 避免绑定面引入 schedule 结构体的 wails model 生成）。
type ScheduleLoadResult struct {
	Path    string `json:"path"`    // 工作区相对路径（=索引 current 指针）
	Exists  bool   `json:"exists"`  // false=尚无计划文件（前端走迁移/空态）
	Project string `json:"project"` // 计划 JSON（存在时）
}

// ScheduleSaveResult 保存回执。
type ScheduleSaveResult struct {
	Path     string `json:"path"`
	SavedAt  string `json:"savedAt"`  // ISO 时间（前端「已保存 HH:MM」显示）
	Duration int    `json:"duration"` // 保存后总工期（工作日），前端校对用
	Critical int    `json:"critical"` // 关键工作数
}

// ScheduleProjectSummary 工程摘要（列表视图行：GaeaScheduleProjects /
// ProjectArchive / ProjectDelete 共用）。
type ScheduleProjectSummary struct {
	Rel       string `json:"rel"`
	Name      string `json:"name"`
	Archived  bool   `json:"archived"`  // 归档只标索引，文件在原位
	Duration  int    `json:"duration"`  // 总工期（工作日；CPM 不过/文件坏=0）
	TaskCount int    `json:"taskCount"` // 子任务数（分组行不计）
	Ok        bool   `json:"ok"`        // false=装载/CPM 失败（坏文件诚实呈现）
	UpdatedAt string `json:"updatedAt"` // RFC3339
}

// ScheduleProjectsResult 工程列表+当前指针视图。
type ScheduleProjectsResult struct {
	Current  string                   `json:"current"`
	Projects []ScheduleProjectSummary `json:"projects"`
}

// ScheduleOpenResult 打开（切指针）回执。
type ScheduleOpenResult struct {
	Current string `json:"current"`
}

// ScheduleCreateResult 新建工程回执：Rel 即落盘文件（此后文件名不再改，
// agent 显式 path 引用稳定），Current=自动切为当前。
type ScheduleCreateResult struct {
	Rel     string `json:"rel"`
	Current string `json:"current"`
}

// GaeaScheduleLoad 装载当前工程计划文件（索引 current 指针；索引不可用回落
// DefaultRelPath）。文件不存在返回 Exists=false（不视为错误）。
func (a *App) GaeaScheduleLoad() (ScheduleLoadResult, error) {
	rel := scheduleCurrentRel(gaeaCwd())
	path := filepath.Join(gaeaCwd(), filepath.FromSlash(rel))
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ScheduleLoadResult{Path: rel, Exists: false}, nil
		}
		return ScheduleLoadResult{}, fmt.Errorf("读取计划文件失败：%w", err)
	}
	// 落盘前过引擎校验（坏文件不在板块里静默炸，给出可读错误）
	if _, err := schedule.Load(path); err != nil {
		return ScheduleLoadResult{}, err
	}
	return ScheduleLoadResult{Path: rel, Exists: true, Project: string(raw)}, nil
}

// GaeaScheduleSave 保存板块计划（整量覆盖）：校验+CPM fail-closed+原子写。
// rel 为保存目标（板块装载时绑定，规避切换竞态——切换只改指针不动保存目标）；
// rel 为空回落索引 current（wails 缺参零值语义，老调用形态兜底）。
func (a *App) GaeaScheduleSave(projectJSON, rel string) (ScheduleSaveResult, error) {
	if strings.TrimSpace(projectJSON) == "" {
		return ScheduleSaveResult{}, fmt.Errorf("计划内容为空")
	}
	var p schedule.Project
	if err := json.Unmarshal([]byte(projectJSON), &p); err != nil {
		return ScheduleSaveResult{}, fmt.Errorf("计划 JSON 解析失败：%w", err)
	}
	if err := checkScheduleRel(rel); err != nil {
		return ScheduleSaveResult{}, err
	}
	cwd := gaeaCwd()
	rel = strings.TrimSpace(rel)
	if rel == "" {
		rel = scheduleCurrentRel(cwd)
	}
	rel = filepath.ToSlash(rel)
	if err := schedule.Save(filepath.Join(cwd, filepath.FromSlash(rel)), p); err != nil {
		return ScheduleSaveResult{}, err
	}
	// 保存成功后更新索引该条目 updatedAt（索引不存在则顺带建）；索引是缓存
	// 非权威，写失败不拦保存回执（下次 Load 轻扫会再收编）。
	touchScheduleIndexEntry(cwd, rel, p.Name)
	cpm, a2 := p.Analyze()
	dur, crit := 0, 0
	if cpm.OK {
		dur, crit = cpm.Duration, len(a2.Critical)
	}
	return ScheduleSaveResult{
		Path:     rel,
		SavedAt:  time.Now().Format("15:04"),
		Duration: dur,
		Critical: crit,
	}, nil
}

// GaeaScheduleProjects 工程列表+当前指针：逐文件 Load+Analyze 出摘要
// （工程数量级=个位数，开销可接受；单文件坏 → Ok=false 不计 Duration，
// 诚实呈现不静默吞）。含已归档条目（Archived 标记，切换器/管理列表各取所需）。
func (a *App) GaeaScheduleProjects() (ScheduleProjectsResult, error) {
	cwd := gaeaCwd()
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// GaeaScheduleProjectOpen 打开工程：SetCurrentSchedule 语义（校验 rel 在注册
// 表或文件在盘且后缀合法 → 原子写索引 current）。不搬数据、不改工程文件。
func (a *App) GaeaScheduleProjectOpen(rel string) (ScheduleOpenResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if err := schedule.SetCurrentSchedule(gaeaCwd(), rel); err != nil {
		return ScheduleOpenResult{}, err
	}
	return ScheduleOpenResult{Current: rel}, nil
}

// GaeaScheduleProjectCreate 新建工程：name trim 非空 → SafeSlugName（冲突加
// 序号）→ 进度计划/<slug>.gsched.json 落空 Project（走 schedule.Save 全套
// 校验，name=用户工程名）→ 索引登记 → 自动切为当前。此后文件名不再改。
func (a *App) GaeaScheduleProjectCreate(name string) (ScheduleCreateResult, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ScheduleCreateResult{}, fmt.Errorf("工程名称为空")
	}
	cwd := gaeaCwd()
	// 文件名占用判定：登记条目与目录游离文件都算（此后文件名不再改，撞名代价高）
	existing := map[string]bool{}
	if idx, err := schedule.LoadScheduleIndex(cwd); err == nil {
		for _, e := range idx.Projects {
			existing[scheduleRelBase(e.Rel)] = true
		}
	}
	if scanned, err := schedule.ScanScheduleProjects(cwd); err == nil {
		for _, e := range scanned {
			existing[scheduleRelBase(e.Rel)] = true
		}
	}
	slug := schedule.SafeSlugName(name, existing)
	rel := path.Join(schedule.ScheduleProjectsDir, slug+schedule.ScheduleFileSuffix)
	p := schedule.Project{Name: name, Tasks: []schedule.Task{}, Links: []schedule.Link{}}
	if err := schedule.Save(filepath.Join(cwd, filepath.FromSlash(rel)), p); err != nil {
		return ScheduleCreateResult{}, err
	}
	if err := schedule.SetCurrentSchedule(cwd, rel); err != nil {
		return ScheduleCreateResult{}, err
	}
	return ScheduleCreateResult{Rel: rel, Current: rel}, nil
}

// GaeaScheduleProjectArchive 归档/反归档：索引标 archived，文件不动（用户
// 资产，agent 显式 path 引用不失效）；未知 rel 报错。
func (a *App) GaeaScheduleProjectArchive(rel string, archived bool) (ScheduleProjectsResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	cwd := gaeaCwd()
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	found := false
	for i := range idx.Projects {
		if idx.Projects[i].Rel == rel {
			idx.Projects[i].Archived = archived
			found = true
			break
		}
	}
	if !found {
		return ScheduleProjectsResult{}, fmt.Errorf("未知工程：%s", rel)
	}
	if err := schedule.SaveScheduleIndex(cwd, idx); err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// GaeaScheduleProjectDelete 物理删文件+索引摘除（不可恢复，前端二次确认）。
// 删的是当前 → current 切到剩余第一个未归档工程；没有则回落 DefaultRelPath
// 并确保该文件存在（不存在则落空工程）。DefaultRelPath 文件允许删除，删后
// 同上规则。未知 rel（不在注册表且文件也不在盘）报错。
func (a *App) GaeaScheduleProjectDelete(rel string) (ScheduleProjectsResult, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if err := checkScheduleRel(rel); err != nil {
		return ScheduleProjectsResult{}, err
	}
	cwd := gaeaCwd()
	idx, err := schedule.LoadScheduleIndex(cwd)
	if err != nil {
		return ScheduleProjectsResult{}, err
	}
	kept := make([]schedule.ScheduleIndexEntry, 0, len(idx.Projects))
	found := false
	for _, e := range idx.Projects {
		if e.Rel == rel {
			found = true
			continue
		}
		kept = append(kept, e)
	}
	abs := filepath.Join(cwd, filepath.FromSlash(rel))
	_, statErr := os.Stat(abs)
	if !found && statErr != nil {
		return ScheduleProjectsResult{}, fmt.Errorf("未知工程：%s", rel)
	}
	if statErr == nil {
		if err := os.Remove(abs); err != nil {
			return ScheduleProjectsResult{}, fmt.Errorf("删除计划文件失败：%w", err)
		}
	}
	idx.Projects = kept
	if idx.Current == rel {
		idx.Current = schedule.DefaultRelPath
		for _, e := range kept {
			if !e.Archived {
				idx.Current = e.Rel
				break
			}
		}
		if idx.Current == schedule.DefaultRelPath {
			if err := ensureSchedulePlanFile(cwd, schedule.DefaultRelPath); err != nil {
				return ScheduleProjectsResult{}, err
			}
		}
	}
	if err := schedule.SaveScheduleIndex(cwd, idx); err != nil {
		return ScheduleProjectsResult{}, err
	}
	return buildScheduleProjectsResult(cwd, idx), nil
}

// ── 内部助手 ─────────────────────────────────────────────────────────

// scheduleCurrentRel 当前工程 rel：索引 current 优先，索引不可用回落
// DefaultRelPath（板块 Load/Save 与 agent 缺省 path 同一解析口径）。
func scheduleCurrentRel(cwd string) string {
	if idx, err := schedule.LoadScheduleIndex(cwd); err == nil && strings.TrimSpace(idx.Current) != "" {
		return idx.Current
	}
	return schedule.DefaultRelPath
}

// checkScheduleRel 显式 rel 的最低防线：空放行（走缺省回落分支），非空则
// 后缀必须 .gsched.json 且禁止 .. 逃逸工作区。
func checkScheduleRel(rel string) error {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil
	}
	if !strings.HasSuffix(rel, schedule.ScheduleFileSuffix) {
		return fmt.Errorf("非法计划文件后缀（须 .gsched.json）：%s", rel)
	}
	for _, seg := range strings.Split(filepath.ToSlash(rel), "/") {
		if seg == ".." {
			return fmt.Errorf("非法计划文件路径（禁止 .. 逃逸）：%s", rel)
		}
	}
	return nil
}

// touchScheduleIndexEntry 保存后更新条目 updatedAt（未登记则顺带收编；
// 索引不存在则顺带建）。尽力而为：失败不拦保存回执。
func touchScheduleIndexEntry(cwd, rel, name string) {
	if idx, err := schedule.LoadScheduleIndex(cwd); err == nil {
		now := time.Now()
		found := false
		for i := range idx.Projects {
			if idx.Projects[i].Rel == rel {
				idx.Projects[i].UpdatedAt = now
				found = true
				break
			}
		}
		if !found {
			idx.Projects = append(idx.Projects, schedule.ScheduleIndexEntry{Rel: rel, Name: name, UpdatedAt: now})
		}
		_ = schedule.SaveScheduleIndex(cwd, idx)
	}
}

// buildScheduleProjectsResult 索引 → 列表视图（逐文件 Load+Analyze 出摘要）。
func buildScheduleProjectsResult(cwd string, idx schedule.ScheduleIndex) ScheduleProjectsResult {
	current := strings.TrimSpace(idx.Current)
	if current == "" {
		current = schedule.DefaultRelPath
	}
	res := ScheduleProjectsResult{Current: current, Projects: make([]ScheduleProjectSummary, 0, len(idx.Projects))}
	for _, e := range idx.Projects {
		s := ScheduleProjectSummary{
			Rel:       e.Rel,
			Name:      e.Name,
			Archived:  e.Archived,
			UpdatedAt: e.UpdatedAt.Format(time.RFC3339),
		}
		if strings.TrimSpace(s.Name) == "" {
			s.Name = scheduleRelBase(e.Rel)
		}
		if p, err := schedule.Load(filepath.Join(cwd, filepath.FromSlash(e.Rel))); err == nil {
			if cpm, a := p.Analyze(); cpm.OK {
				s.Ok = true
				s.Duration = cpm.Duration
				s.TaskCount = a.LeafCount
			}
		}
		res.Projects = append(res.Projects, s)
	}
	return res
}

// ensureSchedulePlanFile 目标计划文件不存在时落空工程（name=文件基础名）。
func ensureSchedulePlanFile(cwd, rel string) error {
	abs := filepath.Join(cwd, filepath.FromSlash(rel))
	if _, err := os.Stat(abs); err == nil {
		return nil
	}
	p := schedule.Project{Name: scheduleRelBase(rel), Tasks: []schedule.Task{}, Links: []schedule.Link{}}
	return schedule.Save(abs, p)
}

// scheduleRelBase rel → 去目录去后缀的基础名。
func scheduleRelBase(rel string) string {
	return strings.TrimSuffix(path.Base(filepath.ToSlash(rel)), schedule.ScheduleFileSuffix)
}
