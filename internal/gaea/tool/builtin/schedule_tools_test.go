package builtin

// schedule_tools_test.go — 进度计划三工具 Execute 级测试（v4.113.0 刀4）：
// ops/project 双通道、环 fail-closed、写根越界拒绝、get/analyze 回读。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func schedApply(t *testing.T, dir string, args string) map[string]any {
	t.Helper()
	tool := scheduleApply{workDir: dir, roots: []string{dir}}
	out, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("apply 输出非 JSON: %v", err)
	}
	return m
}

func TestScheduleToolsProjectChannelAndReadBack(t *testing.T) {
	dir := t.TempDir()
	proj := `{
		"name":"办公楼","startDate":"2026-09-01",
		"calendar":{"workweek":[1,2,3,4,5],"holidays":[]},
		"tasks":[{"id":"g1","name":"前期","level":0,"duration":0,"progress":0},
		         {"id":"a","name":"场地","level":1,"duration":10,"progress":0},
		         {"id":"m","name":"竣工","level":1,"duration":0,"progress":0,"isMilestone":true}],
		"links":[{"from":"a","to":"m","type":"FS","lag":0}]
	}`
	out := schedApply(t, dir, `{"project":`+proj+`,"summary":"生成整计划"}`)
	if out["duration"].(float64) != 10 {
		t.Fatalf("duration = %v", out["duration"])
	}
	if out["taskCount"].(float64) != 2 {
		t.Fatalf("taskCount = %v", out["taskCount"])
	}

	get, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(get, `"duration":10`) || !strings.Contains(get, `"critical":true`) {
		t.Fatalf("get 缺关键字段: %s", get)
	}

	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !strings.Contains(ana, `"name":"竣工"`) || !strings.Contains(ana, "checks") {
		t.Fatalf("analyze 缺里程碑/质检: %s", ana)
	}
}

func TestScheduleToolsOpsChannelAndCycleReject(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01","tasks":[{"id":"a","name":"A","level":1,"duration":3,"progress":0}]}}`)
	out := schedApply(t, dir, `{"ops":[{"type":"upsert_task","task":{"id":"b","name":"B","level":1,"duration":2}},{"type":"set_links","toId":"b","links":[{"from":"a"}]}]}`)
	if out["duration"].(float64) != 5 {
		t.Fatalf("ops 后 duration = %v", out)
	}

	tool := scheduleApply{workDir: dir, roots: []string{dir}}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"ops":[{"type":"set_links","toId":"a","links":[{"from":"b"}]}]}`))
	if err == nil || !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("环应拒绝：%v", err)
	}

	// project 与 ops 同时给 → 二选一报错
	_, err = tool.Execute(context.Background(), json.RawMessage(`{"project":{"name":"x","tasks":[],"links":[]},"ops":[{"type":"set_meta","name":"y"}]}`))
	if err == nil || !strings.Contains(err.Error(), "二选一") {
		t.Fatalf("双通道应拒绝：%v", err)
	}
}

func TestScheduleAnalyzeNoPredFinding(t *testing.T) {
	dir := t.TempDir()
	// a(首叶,天然无前置不计) b(无前置→发现) c(有前置) manual(手动不计)
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01","tasks":[
		{"id":"a","name":"A","level":1,"duration":2},
		{"id":"b","name":"B","level":1,"duration":2},
		{"id":"c","name":"C","level":1,"duration":2},
		{"id":"m","name":"M","level":1,"duration":1,"mode":"manual","manualStart":3}],
		"links":[{"from":"a","to":"c","type":"FS","lag":0}]}}`)
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !strings.Contains(ana, "1 个任务无前置搭接") {
		t.Fatalf("缺无前置发现：%s", ana)
	}
	// auto_chain 修复后复查：发现清零
	schedApply(t, dir, `{"ops":[{"type":"auto_chain"}]}`)
	ana2, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze2: %v", err)
	}
	if strings.Contains(ana2, "无前置搭接") {
		t.Fatalf("auto_chain 后应清零：%s", ana2)
	}
}

// v4.117.0 刀8：倒排校核全链——set_meta deadline 落盘 → analyze 可行性裁决 →
// 调整超期后发现报超期量 → 清除后消失。
func TestScheduleDeadlineChain(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01","tasks":[
		{"id":"a","name":"A","level":1,"duration":3},
		{"id":"b","name":"B","level":1,"duration":2},
		{"id":"c","name":"C","level":1,"duration":4}],
		"links":[{"from":"a","to":"b","type":"FS","lag":0},{"from":"b","to":"c","type":"FS","lag":0}]}}`)

	// 目标竣工 2026-09-10：周一开工起第 8 个工作日，总工期 9 → 超 1 天
	out := schedApply(t, dir, `{"ops":[{"type":"set_meta","deadline":"2026-09-10"}]}`)
	if out["duration"].(float64) != 9 {
		t.Fatalf("总工期 = %v", out["duration"])
	}
	dc, ok := out["deadlineCheck"].(map[string]any)
	if !ok || dc["feasible"].(bool) || dc["overrun"].(float64) != 1 {
		t.Fatalf("apply 回执缺倒排校核或口径错：%v", out["deadlineCheck"])
	}
	get, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !strings.Contains(get, `"deadline":"2026-09-10"`) {
		t.Fatalf("get 缺 deadline：err=%v %s", err, get)
	}
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !strings.Contains(ana, "倒排校核：目标竣工 2026-09-10（第 8 工作日）不可达——总工期 9 天，超 1 天") {
		t.Fatalf("analyze 缺超期发现：err=%v %s", err, ana)
	}

	// 压缩 C 4→3 达标后：不可达发现消失（压线 feasible）
	schedApply(t, dir, `{"ops":[{"type":"patch_task","id":"c","patch":{"duration":3}}]}`)
	ana, err = scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || strings.Contains(ana, "不可达") {
		t.Fatalf("达标后不应报不可达：err=%v %s", err, ana)
	}

	// 富余充足（目标 09-30）不产生倒排发现，仅 deadlineCheck 数据
	schedApply(t, dir, `{"ops":[{"type":"set_meta","deadline":"2026-09-30"}]}`)
	ana, err = scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || strings.Contains(ana, "倒排校核") {
		t.Fatalf("富余充足不应有倒排发现：err=%v %s", err, ana)
	}

	// 空串清除 → get/analyze 无 deadline
	schedApply(t, dir, `{"ops":[{"type":"set_meta","deadline":""}]}`)
	get, err = scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || strings.Contains(get, `"deadline"`) {
		t.Fatalf("清除后 get 不应带 deadline：err=%v %s", err, get)
	}
}

func TestScheduleApplyConfine(t *testing.T) {
	outside := t.TempDir()
	roots := t.TempDir()
	tool := scheduleApply{workDir: outside, roots: []string{roots}}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"project":{"name":"P","tasks":[],"links":[]}}`))
	if err == nil {
		t.Fatal("越界写入应拒绝")
	}
}

// v4.116.0 刀7：基线对比全链——set_baseline 落盘 → 调整产生漂移 →
// analyze/apply 回执带对比 → clear_baseline 后消失。
func TestScheduleBaselineDriftChain(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01","tasks":[
		{"id":"a","name":"A","level":1,"duration":3},
		{"id":"b","name":"B","level":1,"duration":2},
		{"id":"c","name":"C","level":1,"duration":4}],
		"links":[{"from":"a","to":"b","type":"FS","lag":0},{"from":"b","to":"c","type":"FS","lag":0}]}}`)

	// 保存基线（savedAt 缺省由工具层标注）
	out := schedApply(t, dir, `{"ops":[{"type":"set_baseline","baselineName":"开工版"}]}`)
	if out["duration"].(float64) != 9 {
		t.Fatalf("基线时总工期 = %v", out["duration"])
	}
	// get 带基线元信息
	get, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !strings.Contains(get, `"baseline"`) || !strings.Contains(get, "开工版") {
		t.Fatalf("get 缺基线信息：err=%v %s", err, get)
	}

	// 无变化：analyze 不出漂移发现
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || strings.Contains(ana, "较基线") {
		t.Fatalf("无变化不应有漂移发现：err=%v %s", err, ana)
	}

	// 调整 A 工期 3→5：回执与 analyze 都带漂移（9→11）
	out = schedApply(t, dir, `{"ops":[{"type":"patch_task","id":"a","patch":{"duration":5}}]}`)
	drift, ok := out["baselineDrift"].(map[string]any)
	if !ok || drift["durationDrift"].(float64) != 2 {
		t.Fatalf("apply 回执缺漂移或口径错：%v", out["baselineDrift"])
	}
	ana, err = scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || !strings.Contains(ana, "较基线「开工版」：总工期 9→11 天（+2）") {
		t.Fatalf("analyze 缺漂移发现：err=%v %s", err, ana)
	}
	if !strings.Contains(ana, `"baselineDrift"`) || !strings.Contains(ana, `"shiftedCount":3`) {
		t.Fatalf("analyze 缺漂移数据：%s", ana)
	}

	// 更新基线到新状态后漂移清零
	schedApply(t, dir, `{"ops":[{"type":"set_baseline"}]}`)
	ana, err = scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || strings.Contains(ana, "较基线") {
		t.Fatalf("更新基线后不应再有漂移发现：err=%v %s", err, ana)
	}

	// clear_baseline 后 get/analyze 无基线
	schedApply(t, dir, `{"ops":[{"type":"clear_baseline"}]}`)
	get, err = scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil || strings.Contains(get, `"baseline"`) {
		t.Fatalf("清除后 get 不应带基线：err=%v %s", err, get)
	}
	tool := scheduleApply{workDir: dir, roots: []string{dir}}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"ops":[{"type":"clear_baseline"}]}`)); err == nil {
		t.Fatal("无基线再清除应报错")
	}
}

func TestScheduleGetMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("缺文件应报错")
	}
}

// ── 资源成本刀2（v4.122）：三工具成本回执 ─────────────────────────────────

// get 回执带 resources/assignments/costs{ok,total,byTask,byResource}。
func TestScheduleGetCarriesResourcesAndCosts(t *testing.T) {
	dir := t.TempDir()
	proj := `{
		"name":"P","startDate":"2026-01-05",
		"tasks":[{"id":"a","name":"A","level":1,"duration":3,"progress":0,"fixedCost":120.5},
		         {"id":"b","name":"B","level":1,"duration":2,"progress":0}],
		"resources":[{"id":"r1","name":"人力","type":"work","standardRate":100,"costPerUse":200},
		             {"id":"m1","name":"钢材","type":"material","standardRate":55,"unit":"t"}],
		"assignments":[{"taskId":"a","resourceId":"r1"},{"taskId":"b","resourceId":"m1","quantity":10}]
	}`
	schedApply(t, dir, `{"project":`+proj+`}`)
	out, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("get 输出非 JSON: %v", err)
	}
	if rs, ok := m["resources"].([]any); !ok || len(rs) != 2 {
		t.Fatalf("get 缺 resources：%s", out)
	}
	if as, ok := m["assignments"].([]any); !ok || len(as) != 2 {
		t.Fatalf("get 缺 assignments：%s", out)
	}
	costs, ok := m["costs"].(map[string]any)
	if !ok || costs["ok"] != true {
		t.Fatalf("get 缺 costs：%s", out)
	}
	// a = 120.5 + (3×1×100+200) = 620.5；b = 10×55 = 550；总 1170.5
	if costs["total"].(float64) != 1170.5 {
		t.Fatalf("costs.total = %v, want 1170.5", costs["total"])
	}
	byTask := costs["byTask"].(map[string]any)
	if byTask["a"].(map[string]any)["total"].(float64) != 620.5 {
		t.Fatalf("byTask[a] = %v", byTask["a"])
	}
	byRes := costs["byResource"].(map[string]any)
	if byRes["r1"].(float64) != 500 || byRes["m1"].(float64) != 550 {
		t.Fatalf("byResource = %v", byRes)
	}
}

// apply 回执带写入后总成本；project 通道只带 totalCost，ops 通道另带 taskCosts。
func TestScheduleApplyReceiptCarriesTotalCost(t *testing.T) {
	dir := t.TempDir()
	proj := `{"name":"P","startDate":"2026-01-05",
		"tasks":[{"id":"a","name":"A","level":1,"duration":3,"progress":0},
		         {"id":"b","name":"B","level":1,"duration":2,"progress":0,"fixedCost":50}],
		"resources":[{"id":"r1","name":"人力","type":"work","standardRate":100,"costPerUse":200}],
		"assignments":[{"taskId":"a","resourceId":"r1"}]}`
	out := schedApply(t, dir, `{"project":`+proj+`}`)
	// a = 3×100+200 = 500；b 固定 50 → 总 550
	if out["totalCost"].(float64) != 550 {
		t.Fatalf("totalCost = %v, want 550", out["totalCost"])
	}
	if _, has := out["taskCosts"]; has {
		t.Fatalf("project 通道不应带 taskCosts：%v", out["taskCosts"])
	}
	// ops 通道：压工期 3→5 → a 行成本 500→700，总 550→750
	out = schedApply(t, dir, `{"ops":[{"type":"patch_task","id":"a","patch":{"duration":5}}]}`)
	if out["totalCost"].(float64) != 750 {
		t.Fatalf("ops 后 totalCost = %v, want 750", out["totalCost"])
	}
	taskCosts, ok := out["taskCosts"].(map[string]any)
	if !ok {
		t.Fatalf("ops 回执缺 taskCosts：%v", out)
	}
	row, ok := taskCosts["a"].(map[string]any)
	if !ok || row["total"].(float64) != 700 {
		t.Fatalf("taskCosts[a] = %v, want total 700", taskCosts["a"])
	}
}

// analyze 带成本叙事（costs）与「已分配未定价」发现；补价后收敛。
func TestScheduleAnalyzeCostNarrativeAndUnpriced(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01",
		"tasks":[{"id":"a","name":"挖土","level":1,"duration":2,"progress":0},
		         {"id":"b","name":"吊装","level":1,"duration":3,"progress":0}],
		"resources":[{"id":"r1","name":"普工","type":"work"},
		             {"id":"c1","name":"差旅","type":"cost"}],
		"assignments":[{"taskId":"a","resourceId":"r1"},{"taskId":"b","resourceId":"c1"}]}}`)
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !strings.Contains(ana, "已分配未定价 2 处") ||
		!strings.Contains(ana, "「挖土」的资源「普工」未设费率") ||
		!strings.Contains(ana, "「吊装」的资源「差旅」未填金额") {
		t.Fatalf("analyze 缺未定价发现：%s", ana)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(ana), &m); err != nil {
		t.Fatalf("analyze 输出非 JSON: %v", err)
	}
	costs := m["costs"].(map[string]any)
	if costs["total"].(float64) != 0 {
		t.Fatalf("未定价全按 0 计，total = %v", costs["total"])
	}
	unpriced := costs["unpriced"].([]any)
	if len(unpriced) != 2 || unpriced[0].(map[string]any)["kind"] != "work_norate" {
		t.Fatalf("costs.unpriced = %v", unpriced)
	}
	// patch_resource 补费率后：普工计价（2×100=200），未定价收敛到差旅 1 处
	schedApply(t, dir, `{"ops":[{"type":"patch_resource","id":"r1","resourcePatch":{"standardRate":100}}]}`)
	ana, err = scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze2: %v", err)
	}
	if !strings.Contains(ana, "已分配未定价 1 处") {
		t.Fatalf("补价后未定价应收敛：%s", ana)
	}
	if err := json.Unmarshal([]byte(ana), &m); err != nil {
		t.Fatalf("analyze2 输出非 JSON: %v", err)
	}
	costs = m["costs"].(map[string]any)
	if costs["total"].(float64) != 200 {
		t.Fatalf("补价后 total = %v, want 200", costs["total"])
	}
	top := costs["topTasks"].([]any)
	if len(top) != 1 || top[0].(map[string]any)["id"] != "a" || top[0].(map[string]any)["total"].(float64) != 200 {
		t.Fatalf("costs.topTasks = %v", top)
	}
}

// 资源 ops 全链（工具层）：先建资源再挂分配 → 回执总成本 → 分组行拒绝 →
// fixedCost → remove_resource 级联（get 复核）。
func TestScheduleResourceOpsChain(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01","tasks":[
		{"id":"g","name":"分组","level":0,"duration":0,"progress":0},
		{"id":"a","name":"A","level":1,"duration":4,"progress":0}]}}`)

	// upsert_resource + set_assignments：a = 4×2×300+200 = 2600
	out := schedApply(t, dir, `{"ops":[
		{"type":"upsert_resource","resource":{"id":"r1","name":"塔吊","type":"work","standardRate":300,"costPerUse":200}},
		{"type":"set_assignments","taskId":"a","assignments":[{"resourceId":"r1","units":2}]}]}`)
	if out["totalCost"].(float64) != 2600 {
		t.Fatalf("totalCost = %v, want 2600", out["totalCost"])
	}
	if !strings.Contains(fmt.Sprint(out["changes"]), "更新「A」资源分配（共 1 条）") {
		t.Fatalf("changes 缺分配摘要：%v", out["changes"])
	}

	// 分组行挂分配：错误原文回传
	tool := scheduleApply{workDir: dir, roots: []string{dir}}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"ops":[{"type":"set_assignments","taskId":"g","assignments":[{"resourceId":"r1"}]}]}`))
	if err == nil || !strings.Contains(err.Error(), "禁止挂分配") {
		t.Fatalf("分组行应拒绝：%v", err)
	}

	// fixedCost：总成本 2600→3000
	out = schedApply(t, dir, `{"ops":[{"type":"patch_task","id":"a","patch":{"fixedCost":400}}]}`)
	if out["totalCost"].(float64) != 3000 {
		t.Fatalf("fixedCost 后 totalCost = %v, want 3000", out["totalCost"])
	}

	// remove_resource 级联删分配：只剩固定成本 400
	out = schedApply(t, dir, `{"ops":[{"type":"remove_resource","id":"r1"}]}`)
	if out["totalCost"].(float64) != 400 {
		t.Fatalf("remove 后 totalCost = %v, want 400", out["totalCost"])
	}
	get, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(get), &m); err != nil {
		t.Fatalf("get 输出非 JSON: %v", err)
	}
	if as, ok := m["assignments"].([]any); !ok || len(as) != 0 {
		t.Fatalf("remove_resource 应级联清空分配：%s", get)
	}
	if rs, ok := m["resources"].([]any); !ok || len(rs) != 0 {
		t.Fatalf("资源应已删：%s", get)
	}
}
