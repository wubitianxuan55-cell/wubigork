package app

// gaea_schedule_file_test.go — 进度计划多工程绑定用例（v4.139 差距 #15 刀1+刀2）：
// 列表摘要 / 删当前切剩余 / Open 未登记收编 / Load 落新指针 / Save rel 显式与
// 空串回落 / Archive 标记 / Create slug 冲突 / Delete 级联。
// 工作区隔离沿用 gaea_spaces_test.go 先例：workspaceTestIsolate + ga.cfg.Workspace。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/schedule"
)

// isolateScheduleWorkspace 隔离出临时工作区并接管 ga.cfg（gaeaCwd 依赖）。
func isolateScheduleWorkspace(t *testing.T) string {
	t.Helper()
	restore := workspaceTestIsolate(t)
	t.Cleanup(restore)
	oldCfg := ga.cfg
	ws := t.TempDir()
	ga.cfg = &gaeaConfig.Config{Workspace: ws}
	t.Cleanup(func() { ga.cfg = oldCfg })
	return ws
}

// chainScheduleProject 构造 FS 串联的合法计划（A→B→…，总工期=durs 之和）。
func chainScheduleProject(name string, durs ...int) schedule.Project {
	p := schedule.Project{Name: name, StartDate: "2026-09-01", Tasks: []schedule.Task{}, Links: []schedule.Link{}}
	const letters = "ABCDEFGHIJ"
	for i, d := range durs {
		id := string(letters[i])
		p.Tasks = append(p.Tasks, schedule.Task{ID: id, Name: id, Duration: d, Level: 1})
		if i > 0 {
			p.Links = append(p.Links, schedule.Link{From: string(letters[i-1]), To: id, Type: schedule.FS})
		}
	}
	return p
}

// saveSchedulePlan 落一个合法计划文件。
func saveSchedulePlan(t *testing.T, ws, rel, name string, durs ...int) {
	t.Helper()
	abs := filepath.Join(ws, filepath.FromSlash(rel))
	if err := schedule.Save(abs, chainScheduleProject(name, durs...)); err != nil {
		t.Fatalf("save %s: %v", rel, err)
	}
}

// seedScheduleIndex 预置索引（UpdatedAt 由调用方断言用可另改）。
func seedScheduleIndex(t *testing.T, ws string, idx schedule.ScheduleIndex) {
	t.Helper()
	if err := schedule.SaveScheduleIndex(ws, idx); err != nil {
		t.Fatal(err)
	}
}

// findScheduleSummary 按 rel 找摘要行。
func findScheduleSummary(res ScheduleProjectsResult, rel string) (ScheduleProjectSummary, bool) {
	for _, s := range res.Projects {
		if s.Rel == rel {
			return s, true
		}
	}
	return ScheduleProjectSummary{}, false
}

func TestGaeaScheduleLoadFallsBackWithoutIndex(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	a := &App{}
	got, err := a.GaeaScheduleLoad()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Path != schedule.DefaultRelPath || got.Exists {
		t.Fatalf("空工作区应回落默认且 Exists=false：%+v", got)
	}
	_ = ws
}

func TestGaeaScheduleLoadAndOpenFollowIndexPointer(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 3, 2)
	saveSchedulePlan(t, ws, "进度计划/办公楼二期.gsched.json", "办公楼二期", 5)
	a := &App{}

	got, err := a.GaeaScheduleLoad()
	if err != nil || got.Path != schedule.DefaultRelPath || !got.Exists {
		t.Fatalf("初始应落默认指针：%+v err=%v", got, err)
	}

	opened, err := a.GaeaScheduleProjectOpen("进度计划/办公楼二期.gsched.json")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if opened.Current != "进度计划/办公楼二期.gsched.json" {
		t.Fatalf("open 回执 = %q", opened.Current)
	}
	// Open 后 Load 落新指针（「对话改的 = 板块打开的」）
	got, err = a.GaeaScheduleLoad()
	if err != nil || got.Path != "进度计划/办公楼二期.gsched.json" || !strings.Contains(got.Project, "办公楼二期") {
		t.Fatalf("Open 后 Load 应落新指针：%+v err=%v", got, err)
	}
	// 显式 path 优先于指针已由工具层覆盖；此处确认指针持久化（重启保持）
	idx, err := schedule.LoadScheduleIndex(ws)
	if err != nil || idx.Current != "进度计划/办公楼二期.gsched.json" {
		t.Fatalf("索引指针未持久化：%+v err=%v", idx, err)
	}
}

func TestGaeaScheduleProjectsSummaries(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 3, 2)
	saveSchedulePlan(t, ws, "进度计划/办公楼二期.gsched.json", "办公楼二期", 5)
	// 单文件坏：Ok=false 不计 Duration
	badPath := filepath.Join(ws, "进度计划", "坏损.gsched.json")
	if err := os.WriteFile(badPath, []byte("{坏JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedScheduleIndex(t, ws, schedule.ScheduleIndex{Version: 1, Current: schedule.DefaultRelPath, Projects: []schedule.ScheduleIndexEntry{
		{Rel: schedule.DefaultRelPath, Name: "当前计划", UpdatedAt: time.Now()},
		{Rel: "进度计划/办公楼二期.gsched.json", Name: "办公楼二期", UpdatedAt: time.Now()},
		{Rel: "进度计划/坏损.gsched.json", Name: "坏损计划", UpdatedAt: time.Now()},
	}})
	a := &App{}

	res, err := a.GaeaScheduleProjects()
	if err != nil {
		t.Fatalf("projects: %v", err)
	}
	if res.Current != schedule.DefaultRelPath || len(res.Projects) != 3 {
		t.Fatalf("current=%q projects=%d", res.Current, len(res.Projects))
	}
	def, ok := findScheduleSummary(res, schedule.DefaultRelPath)
	if !ok || !def.Ok || def.Duration != 5 || def.TaskCount != 2 || def.Name != "当前计划" || def.UpdatedAt == "" {
		t.Fatalf("默认工程摘要不符：%+v", def)
	}
	office, ok := findScheduleSummary(res, "进度计划/办公楼二期.gsched.json")
	if !ok || !office.Ok || office.Duration != 5 || office.TaskCount != 1 {
		t.Fatalf("办公楼摘要不符：%+v", office)
	}
	bad, ok := findScheduleSummary(res, "进度计划/坏损.gsched.json")
	if !ok || bad.Ok || bad.Duration != 0 || bad.Name != "坏损计划" {
		t.Fatalf("坏文件应 Ok=false 不计 Duration：%+v", bad)
	}
}

func TestGaeaScheduleProjectOpenUnregistered(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 1)
	// 未登记但在盘且后缀合法
	saveSchedulePlan(t, ws, "进度计划/临时计划.gsched.json", "临时计划", 4)
	a := &App{}

	if _, err := a.GaeaScheduleProjectOpen("进度计划/文档.txt"); err == nil {
		t.Fatal("非法后缀应拒绝")
	}
	if _, err := a.GaeaScheduleProjectOpen("进度计划/不存在.gsched.json"); err == nil {
		t.Fatal("不存在文件应拒绝")
	}
	if _, err := a.GaeaScheduleProjectOpen("进度计划/临时计划.gsched.json"); err != nil {
		t.Fatalf("open 未登记但在盘: %v", err)
	}
	res, err := a.GaeaScheduleProjects()
	if err != nil {
		t.Fatal(err)
	}
	if res.Current != "进度计划/临时计划.gsched.json" || len(res.Projects) != 2 {
		t.Fatalf("Open 应收编登记并切指针：current=%q n=%d", res.Current, len(res.Projects))
	}
}

func TestGaeaScheduleSaveRelExplicitAndEmptyFallback(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 3, 2)
	saveSchedulePlan(t, ws, "进度计划/办公楼二期.gsched.json", "办公楼二期", 5)
	officeRel := "进度计划/办公楼二期.gsched.json"
	seed := time.Now().Add(-time.Hour)
	seedScheduleIndex(t, ws, schedule.ScheduleIndex{Version: 1, Current: schedule.DefaultRelPath, Projects: []schedule.ScheduleIndexEntry{
		{Rel: schedule.DefaultRelPath, Name: "当前计划", UpdatedAt: seed},
		{Rel: officeRel, Name: "办公楼二期", UpdatedAt: seed},
	}})
	a := &App{}

	// 显式 rel：写到指定文件（切换竞态防护：保存目标不随指针漂移）
	raw, err := json.Marshal(chainScheduleProject("改名后", 6))
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.GaeaScheduleSave(string(raw), officeRel)
	if err != nil || got.Path != officeRel || got.Duration != 6 {
		t.Fatalf("显式 rel 保存：%+v err=%v", got, err)
	}
	if p, err := schedule.Load(filepath.Join(ws, "进度计划", "办公楼二期.gsched.json")); err != nil || p.Name != "改名后" {
		t.Fatalf("显式 rel 未落盘：name=%q err=%v", p.Name, err)
	}

	// 空串 rel：回落索引 current（wails 缺参零值语义兜底）
	raw, err = json.Marshal(chainScheduleProject("默认更新", 2, 3))
	if err != nil {
		t.Fatal(err)
	}
	got, err = a.GaeaScheduleSave(string(raw), "")
	if err != nil || got.Path != schedule.DefaultRelPath {
		t.Fatalf("空串 rel 应回落索引 current：%+v err=%v", got, err)
	}
	if p, err := schedule.Load(filepath.Join(ws, "进度计划", "当前计划.gsched.json")); err != nil || p.Name != "默认更新" {
		t.Fatalf("回落保存未落盘：name=%q err=%v", p.Name, err)
	}

	// 两处条目 updatedAt 均被刷新（顺带建索引/收编同路径覆盖）
	idx, err := schedule.LoadScheduleIndex(ws)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range idx.Projects {
		if !e.UpdatedAt.After(seed) {
			t.Fatalf("条目 updatedAt 未刷新：%+v", e)
		}
	}
}

func TestGaeaScheduleProjectCreateSlug(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 1)
	a := &App{}

	got, err := a.GaeaScheduleProjectCreate("办公楼二期")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.Rel != "进度计划/办公楼二期.gsched.json" || got.Current != got.Rel {
		t.Fatalf("create 回执：%+v", got)
	}
	if p, err := schedule.Load(filepath.Join(ws, "进度计划", "办公楼二期.gsched.json")); err != nil || p.Name != "办公楼二期" || len(p.Tasks) != 0 {
		t.Fatalf("空工程落盘不符：name=%q err=%v", p.Name, err)
	}
	// 自动切为当前（Load 落新指针）
	if l, err := a.GaeaScheduleLoad(); err != nil || l.Path != got.Rel {
		t.Fatalf("新建后 Load 应落新工程：%+v err=%v", l, err)
	}

	// 同名冲突加序号；非法字符清洗
	again, err := a.GaeaScheduleProjectCreate("办公楼二期")
	if err != nil || again.Rel != "进度计划/办公楼二期-2.gsched.json" {
		t.Fatalf("同名应加序号：%+v err=%v", again, err)
	}
	clean, err := a.GaeaScheduleProjectCreate("a/b*c")
	if err != nil || clean.Rel != "进度计划/a-b-c.gsched.json" {
		t.Fatalf("非法字符应清洗：%+v err=%v", clean, err)
	}
	if _, err := a.GaeaScheduleProjectCreate("   "); err == nil {
		t.Fatal("空名应拒绝")
	}
	res, err := a.GaeaScheduleProjects()
	if err != nil || len(res.Projects) != 4 || res.Current != clean.Rel {
		t.Fatalf("create 后列表/指针不符：current=%q n=%d err=%v", res.Current, len(res.Projects), err)
	}
}

func TestGaeaScheduleProjectCopy(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "调整1", 3, 2)
	seedScheduleIndex(t, ws, schedule.ScheduleIndex{Version: 1, Current: schedule.DefaultRelPath, Projects: []schedule.ScheduleIndexEntry{
		{Rel: schedule.DefaultRelPath, Name: "调整1", UpdatedAt: time.Now()},
	}})
	a := &App{}

	newRel := "进度计划/调整2.gsched.json"
	res, err := a.GaeaScheduleProjectCopy(schedule.DefaultRelPath, "调整2")
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(newRel))); err != nil {
		t.Fatalf("副本未落盘：%v", err)
	}
	if p, err := schedule.Load(filepath.Join(ws, filepath.FromSlash(newRel))); err != nil || p.Name != "调整2" || len(p.Tasks) != 2 {
		t.Fatalf("副本内容不符（任务应随行）：name=%q n=%d err=%v", p.Name, len(p.Tasks), err)
	}
	// 指针不动（文件级动作语义，同归档）
	if l, err := a.GaeaScheduleLoad(); err != nil || l.Path != schedule.DefaultRelPath {
		t.Fatalf("复制不应切指针：%+v err=%v", l, err)
	}
	if s, ok := findScheduleSummary(res, newRel); !ok || s.Name != "调整2" || s.Archived {
		t.Fatalf("回执列表应含未归档副本：%+v ok=%v", res, ok)
	}

	// 同名 slug 加序号；源不存在/空名拒绝
	if _, err := a.GaeaScheduleProjectCopy(schedule.DefaultRelPath, "调整2"); err != nil {
		t.Fatalf("同名复制: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, "进度计划", "调整2-2.gsched.json")); err != nil {
		t.Fatalf("同名复制应加序号落盘：%v", err)
	}
	if _, err := a.GaeaScheduleProjectCopy("进度计划/不存在.gsched.json", "x"); err == nil {
		t.Fatal("源不存在应拒绝")
	}
	if _, err := a.GaeaScheduleProjectCopy(schedule.DefaultRelPath, "  "); err == nil {
		t.Fatal("空名应拒绝")
	}
}

func TestGaeaScheduleProjectArchiveKeepsFile(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 3, 2)
	saveSchedulePlan(t, ws, "进度计划/办公楼二期.gsched.json", "办公楼二期", 5)
	seedScheduleIndex(t, ws, schedule.ScheduleIndex{Version: 1, Current: schedule.DefaultRelPath, Projects: []schedule.ScheduleIndexEntry{
		{Rel: schedule.DefaultRelPath, Name: "当前计划", UpdatedAt: time.Now()},
		{Rel: "进度计划/办公楼二期.gsched.json", Name: "办公楼二期", UpdatedAt: time.Now()},
	}})
	a := &App{}

	res, err := a.GaeaScheduleProjectArchive("进度计划/办公楼二期.gsched.json", true)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	e, ok := findScheduleSummary(res, "进度计划/办公楼二期.gsched.json")
	if !ok || !e.Archived {
		t.Fatalf("归档未标记：%+v", e)
	}
	if res.Current != schedule.DefaultRelPath {
		t.Fatalf("归档不应动指针：%q", res.Current)
	}
	// 文件在原位（agent 显式 path 引用不失效）
	if _, err := os.Stat(filepath.Join(ws, "进度计划", "办公楼二期.gsched.json")); err != nil {
		t.Fatalf("归档不应动文件：%v", err)
	}
	// 反归档
	res, err = a.GaeaScheduleProjectArchive("进度计划/办公楼二期.gsched.json", false)
	if err != nil {
		t.Fatal(err)
	}
	if e, _ := findScheduleSummary(res, "进度计划/办公楼二期.gsched.json"); e.Archived {
		t.Fatal("反归档未清除标记")
	}
	if _, err := a.GaeaScheduleProjectArchive("进度计划/不存在.gsched.json", true); err == nil {
		t.Fatal("未知 rel 应报错")
	}
}

func TestGaeaScheduleProjectDeleteCascadesCurrent(t *testing.T) {
	ws := isolateScheduleWorkspace(t)
	saveSchedulePlan(t, ws, schedule.DefaultRelPath, "当前计划", 3, 2)
	saveSchedulePlan(t, ws, "进度计划/办公楼二期.gsched.json", "办公楼二期", 5)
	seedScheduleIndex(t, ws, schedule.ScheduleIndex{Version: 1, Current: schedule.DefaultRelPath, Projects: []schedule.ScheduleIndexEntry{
		{Rel: schedule.DefaultRelPath, Name: "当前计划", UpdatedAt: time.Now()},
		{Rel: "进度计划/办公楼二期.gsched.json", Name: "办公楼二期", UpdatedAt: time.Now()},
	}})
	a := &App{}

	// 删当前（DefaultRelPath 文件允许删）→ current 切剩余第一个未归档
	res, err := a.GaeaScheduleProjectDelete(schedule.DefaultRelPath)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, "进度计划", "当前计划.gsched.json")); !os.IsNotExist(err) {
		t.Fatalf("文件应物理删除：%v", err)
	}
	if res.Current != "进度计划/办公楼二期.gsched.json" || len(res.Projects) != 1 {
		t.Fatalf("删当前应切剩余：current=%q n=%d", res.Current, len(res.Projects))
	}

	// 删最后一个 → 回落 DefaultRelPath 并确保文件存在（落空工程）
	res, err = a.GaeaScheduleProjectDelete("进度计划/办公楼二期.gsched.json")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if res.Current != schedule.DefaultRelPath || len(res.Projects) != 0 {
		t.Fatalf("删空应回落默认：current=%q n=%d", res.Current, len(res.Projects))
	}
	p, err := schedule.Load(filepath.Join(ws, "进度计划", "当前计划.gsched.json"))
	if err != nil || p.Name != "当前计划" || len(p.Tasks) != 0 {
		t.Fatalf("回落应确保默认文件存在（空工程）：name=%q err=%v", p.Name, err)
	}

	// 未知 rel 报错
	if _, err := a.GaeaScheduleProjectDelete("进度计划/不存在.gsched.json"); err == nil {
		t.Fatal("未知 rel 应报错")
	}
}

// ── 前后端数据契约往返（FE7-01 / FE7-02 回归，2026-10-02 审计）──────────
//
// frontendContractProjectJSON 是**前端契约形态**的计划 JSON：字段名/可选性
// 严格照 frontend/src/schedule/types.ts 声明书写（多基线 baselines[]、活跃
// 基线 baseline、资源个人日历 calendar、assignment 数值三态）。
//
// 契约不变式（本文件三条用例共同守护）：
//  1. 前端声明的字段经 Go Load→Save 一次往返必须**逐字段存活**——Go 是落盘
//     权威（GaeaScheduleSave 先 json.Unmarshal 进 schedule.Project 再
//     json.MarshalIndent 整量覆盖），Go 结构体缺字段 = 静默擦除用户数据。
//  2. 数值字段的「缺失」与「显式 0」必须可区分：缺失不落键，显式 0 落 0。
//  3. 槽位数组顺序必须原样保留（baselines 是 FIFO 槽位序，顺序即语义）。
const frontendContractProjectJSON = `{
  "name": "契约往返工程",
  "startDate": "2026-09-07",
  "tasks": [
    { "id": "A", "name": "挖土", "duration": 3, "level": 1, "progress": 0 },
    { "id": "B", "name": "垫层", "duration": 2, "level": 1, "progress": 0 }
  ],
  "links": [{ "from": "A", "to": "B", "type": "FS", "lag": 0 }],
  "calendar": { "workweek": [1, 2, 3, 4, 5], "holidays": [] },
  "baseline": { "name": "签证B", "savedAt": "2026-09-20 09:30", "duration": 5, "rows": { "A": { "name": "挖土", "es": 0, "ef": 3, "dur": 3, "critical": true } } },
  "baselines": [
    { "name": "开工版", "savedAt": "2026-09-07 08:00", "duration": 5, "rows": { "A": { "name": "挖土", "es": 0, "ef": 3, "dur": 3, "critical": true } } },
    { "name": "签证A", "savedAt": "2026-09-15 14:05", "duration": 5, "rows": { "A": { "name": "挖土", "es": 0, "ef": 3, "dur": 3, "critical": false } } },
    { "name": "签证B", "savedAt": "2026-09-20 09:30", "duration": 5, "rows": { "A": { "name": "挖土", "es": 1, "ef": 4, "dur": 3, "critical": true } } }
  ],
  "resources": [
    { "id": "r1", "name": "挖机", "type": "work", "standardRate": 800, "costPerUse": 200, "maxUnits": 2, "calendar": { "workweek": [1, 2, 3, 4, 5, 6], "holidays": ["2026-10-01"] } },
    { "id": "r2", "name": "混凝土", "type": "material", "unit": "m³", "standardRate": 450 },
    { "id": "r3", "name": "措施费", "type": "cost" }
  ],
  "assignments": [
    { "taskId": "A", "resourceId": "r1" },
    { "taskId": "A", "resourceId": "r2", "quantity": 0 },
    { "taskId": "A", "resourceId": "r3" },
    { "taskId": "B", "resourceId": "r1", "units": 0 }
  ]
}`

// contractRoundTripProject 保存前端契约 JSON 再读回，返回读回后的原始 JSON。
// 走的是板块真实链路（GaeaScheduleSave → GaeaScheduleLoad，绑定层生产函数），
// 不经过任何测试专用序列化入口。
func contractRoundTripProject(t *testing.T) []byte {
	t.Helper()
	isolateScheduleWorkspace(t)
	a := &App{}
	if _, err := a.GaeaScheduleSave(frontendContractProjectJSON, schedule.DefaultRelPath); err != nil {
		t.Fatalf("保存前端契约计划失败：%v", err)
	}
	l, err := a.GaeaScheduleLoad()
	if err != nil {
		t.Fatalf("读回前端契约计划失败：%v", err)
	}
	if !l.Exists || strings.TrimSpace(l.Project) == "" {
		t.Fatalf("读回结果为空：%+v", l)
	}
	return []byte(l.Project)
}

// contractBaselineSeen 读回结果的基线/资源/分配视图（RawMessage 保留「键是否
// 存在」这一信息——三态断言的唯一手段，解进 struct 会把缺失摊成零值）。
type contractBaselineSeen struct {
	Baselines []struct {
		Name     string                     `json:"name"`
		SavedAt  string                     `json:"savedAt"`
		Duration int                        `json:"duration"`
		Rows     map[string]json.RawMessage `json:"rows"`
	} `json:"baselines"`
	Baseline *struct {
		Name string `json:"name"`
	} `json:"baseline"`
	Resources   []map[string]json.RawMessage `json:"resources"`
	Assignments []map[string]json.RawMessage `json:"assignments"`
}

func decodeContractSeen(t *testing.T, raw []byte) contractBaselineSeen {
	t.Helper()
	var seen contractBaselineSeen
	if err := json.Unmarshal(raw, &seen); err != nil {
		t.Fatalf("读回 JSON 解析失败：%v", err)
	}
	return seen
}

// contractAssignment 按 (taskId,resourceId) 找分配（唯一键，Validate 保证）。
func contractAssignment(t *testing.T, seen contractBaselineSeen, taskID, resID string) map[string]json.RawMessage {
	t.Helper()
	for _, a := range seen.Assignments {
		if string(a["taskId"]) == `"`+taskID+`"` && string(a["resourceId"]) == `"`+resID+`"` {
			return a
		}
	}
	t.Fatalf("读回结果缺少分配 %s↔%s：%v", taskID, resID, seen.Assignments)
	return nil
}

// TestGaeaScheduleRoundTripKeepsBaselines 守护 FE7-01：多基线槽位不得被 Go
// 保存链路静默擦除（CHANGELOG v4.137.0 的「baselines[] 上限 3 FIFO+活跃指针」
// 必须在文件里真的存在，否则每次 800ms 防抖自动保存都在删用户数据）。
func TestGaeaScheduleRoundTripKeepsBaselines(t *testing.T) {
	seen := decodeContractSeen(t, contractRoundTripProject(t))

	if len(seen.Baselines) != 3 {
		t.Fatalf("多基线槽位丢失：期望 3 条，读回 %d 条（%+v）", len(seen.Baselines), seen.Baselines)
	}
	// 顺序即槽位序（FIFO），不是字典序——必须逐位相等
	want := []string{"开工版", "签证A", "签证B"}
	for i, name := range want {
		if seen.Baselines[i].Name != name {
			t.Fatalf("基线槽位顺序漂移：槽 %d 期望 %q，读回 %q（全量 %+v）", i, name, seen.Baselines[i].Name, seen.Baselines)
		}
	}
	// 内容不只计数：savedAt/rows 必须随行（只留名字也算丢数据）
	if seen.Baselines[1].SavedAt != "2026-09-15 14:05" {
		t.Fatalf("基线槽位 savedAt 丢失：%+v", seen.Baselines[1])
	}
	if rows := seen.Baselines[1].Rows; len(rows) != 1 || len(rows["A"]) == 0 {
		t.Fatalf("基线槽位 rows 丢失：%+v", seen.Baselines[1])
	}
	// 活跃基线指针（单数 baseline）与槽位并存
	if seen.Baseline == nil || seen.Baseline.Name != "签证B" {
		t.Fatalf("活跃基线指针丢失或错位：%+v", seen.Baseline)
	}
}

// TestGaeaScheduleRoundTripKeepsAssignmentTriState 守护 FE7-02：assignment
// 数值字段的「缺失」与「显式 0」往返后必须仍可区分——缺失不落键（前端
// undefined 语义），显式 0 落 0（前端 numOrUndefined(0)=0 语义）。
// 0 与缺失在成本/可用性口径上分叉（units/maxUnits 的 ?? 1 缺省），
// applyDiff.sameScalar 也不把两者当同值。
func TestGaeaScheduleRoundTripKeepsAssignmentTriState(t *testing.T) {
	seen := decodeContractSeen(t, contractRoundTripProject(t))

	missing := contractAssignment(t, seen, "A", "r1")
	if raw, ok := missing["units"]; ok {
		t.Fatalf("缺失的 units 被写成 %s（应为不落键）", raw)
	}

	zeroQuantity := contractAssignment(t, seen, "A", "r2")
	raw, ok := zeroQuantity["quantity"]
	if !ok {
		t.Fatal("显式 quantity=0 丢失（键被 omitempty 抹掉），0 与缺失不可区分")
	}
	if string(raw) != "0" {
		t.Fatalf("显式 quantity=0 被改写为 %s", raw)
	}

	missingAmount := contractAssignment(t, seen, "A", "r3")
	if raw, ok := missingAmount["amount"]; ok {
		t.Fatalf("缺失的 amount 被写成 %s（应为不落键）", raw)
	}

	zeroUnits := contractAssignment(t, seen, "B", "r1")
	raw, ok = zeroUnits["units"]
	if !ok {
		t.Fatal("显式 units=0 丢失（键被 omitempty 抹掉），0 与缺失不可区分")
	}
	if string(raw) != "0" {
		t.Fatalf("显式 units=0 被改写为 %s", raw)
	}
}

// TestGaeaScheduleRoundTripKeepsResourceCalendar 守护同类静默丢失（本次核实
// 新发现，审计 FE7-01 同因）：前端 SchedResource.calendar（v4.137 #13 资源级
// 日历，ResourcePanel/usage.ts 消费）在 Go Resource 结构体里同样不存在。
func TestGaeaScheduleRoundTripKeepsResourceCalendar(t *testing.T) {
	seen := decodeContractSeen(t, contractRoundTripProject(t))

	var r1 map[string]json.RawMessage
	for _, r := range seen.Resources {
		if string(r["id"]) == `"r1"` {
			r1 = r
			break
		}
	}
	if r1 == nil {
		t.Fatalf("读回结果缺少资源 r1：%v", seen.Resources)
	}
	raw, ok := r1["calendar"]
	if !ok {
		t.Fatal("资源个人日历 calendar 被 Go 保存链路静默擦除（ResourcePanel「已自定义」将回落「跟随项目日历」）")
	}
	if !strings.Contains(string(raw), "2026-10-01") || !strings.Contains(string(raw), "6") {
		t.Fatalf("资源个人日历内容丢失：%s", raw)
	}
}
