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

// TestScheduleDualDurationChain 双工期刀2（v4.151）：project 通道写 cd 任务 →
// get 带 durationUnit → analyze 出 cdTasks 清单 → ops patch 口径词回执。
func TestScheduleDualDurationChain(t *testing.T) {
	dir := t.TempDir()
	proj := `{
		"name":"养护样板","startDate":"2026-09-07",
		"calendar":{"workweek":[1,2,3,4,5],"holidays":[]},
		"tasks":[{"id":"a","name":"挖土","level":1,"duration":5,"progress":0},
		         {"id":"h","name":"养护","level":1,"duration":28,"progress":0,"durationUnit":"cd"},
		         {"id":"b","name":"回填","level":1,"duration":5,"progress":0}],
		"links":[{"from":"a","to":"h","type":"FS","lag":0},{"from":"h","to":"b","type":"FS","lag":0}]
	}`
	out := schedApply(t, dir, `{"project":`+proj+`,"summary":"生成混排计划"}`)
	if out["duration"].(float64) != 30 {
		t.Fatalf("duration = %v, want 30（5+20+5 等效跨度）", out["duration"])
	}

	get, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(get, `"durationUnit":"cd"`) || !strings.Contains(get, `"ef":25`) {
		t.Fatalf("get 缺 cd 口径: %s", get)
	}

	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(ana), &m); err != nil {
		t.Fatalf("analyze 输出非 JSON: %v", err)
	}
	cds, ok := m["cdTasks"].([]any)
	if !ok || len(cds) != 1 {
		t.Fatalf("cdTasks 缺失或数量错: %s", ana)
	}
	cd := cds[0].(map[string]any)
	if cd["name"].(string) != "养护" || cd["duration"].(float64) != 28 || cd["span"].(float64) != 20 || cd["anchor"].(string) != "2026-09-14" {
		t.Fatalf("cdTasks 行 = %v", cd)
	}

	// ops 通道：patch 口径切换与 cd 工期回执词
	out = schedApply(t, dir, `{"ops":[{"type":"patch_task","id":"b","patch":{"durationUnit":"cd","duration":10}}],"summary":"改口径"}`)
	changes := out["changes"].([]any)
	if len(changes) == 0 || !strings.Contains(changes[0].(string), "工期口径→日历天（自然日定时）") || !strings.Contains(changes[0].(string), "10 日历天") {
		t.Fatalf("patch 口径回执 = %v", changes)
	}
}

// 执行期治理叙事（斑马复刻 S2/S5）：schedule_analyze 回执 execution 段 + findings 文案。
func TestScheduleAnalyzeExecutionNarrative(t *testing.T) {
	dir := t.TempDir()
	// A(3)→B(2) FS；A 要求完成=0（红灯：完成日 2，超期 2）；B 未设要求完成。
	// 检视一期 at=3：A 100%、B 50% → 加权 (300+100)/5=80%。
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01",
		"tasks":[{"id":"a","name":"挖土","level":1,"duration":3,"progress":0,"askFinish":0},
		         {"id":"b","name":"吊装","level":1,"duration":2,"progress":0}],
		"links":[{"from":"a","to":"b","type":"FS","lag":0}],
		"reviews":[{"at":3,"date":"2026-09-04","actuals":{"a":{"percent":100},"b":{"percent":50}}}]}}`)
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(ana), &m); err != nil {
		t.Fatalf("analyze 输出非 JSON: %v", err)
	}
	exec, ok := m["execution"].(map[string]any)
	if !ok {
		t.Fatalf("缺 execution 段: %s", ana)
	}
	if exec["reviewCount"].(float64) != 1 {
		t.Fatalf("reviewCount = %v", exec["reviewCount"])
	}
	warn := exec["warn"].(map[string]any)
	if warn["red"].(float64) != 1 || warn["green"].(float64) != 0 || warn["unassigned"].(float64) != 1 {
		t.Fatalf("warn 计数不齐: %v", warn)
	}
	rows := exec["redRows"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["overdue"].(float64) != 2 {
		t.Fatalf("红灯行口径（完成日=EF−1=2，要求 0 → 2 天）: %v", rows)
	}
	latest := exec["latest"].(map[string]any)
	// 加权 80%（A3 天全跨+B 半跨）；计划应完成 A3+B1=4/5=80% → 偏差 0
	if latest["percent"].(float64) != 80 || latest["plannedPercent"].(float64) != 80 || latest["gapPercent"].(float64) != 0 {
		t.Fatalf("检视快照口径: %v", latest)
	}
	if !strings.Contains(ana, "预警红灯 1 项突破要求完成时间") || !strings.Contains(ana, "超期 2 天（0-15天）") {
		t.Fatalf("缺红灯 finding 文案: %s", ana)
	}
	// 无信号计划（零检视、无人设要求完成）→ 回执无 execution 键（不造假零值）
	schedApply(t, dir, `{"project":{"name":"Q","startDate":"2026-09-01",
		"tasks":[{"id":"x","name":"挖土","level":1,"duration":2,"progress":0}]}}`)
	ana2, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze2: %v", err)
	}
	var m2 map[string]any
	if err := json.Unmarshal([]byte(ana2), &m2); err != nil {
		t.Fatalf("analyze2 输出非 JSON: %v", err)
	}
	if _, exists := m2["execution"]; exists {
		t.Fatalf("无信号不应出 execution 段: %s", ana2)
	}
}

// schedule_get 执行期治理段 + 任务行 askFinish（斑马复刻 S2/S5 读面收口）。
func TestScheduleGetExecutionAndAskFinish(t *testing.T) {
	dir := t.TempDir()
	// A(3)→B(2) FS；A askFinish=0（红灯：完成日 2，超期 2）；B 未设置。
	// 检视一期 at=3：A 100%、B 50% → 加权 80%。
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01",
		"tasks":[{"id":"a","name":"挖土","level":1,"duration":3,"progress":0,"askFinish":0},
		         {"id":"b","name":"吊装","level":1,"duration":2,"progress":0}],
		"links":[{"from":"a","to":"b","type":"FS","lag":0}],
		"reviews":[{"at":3,"date":"2026-09-04","actuals":{"a":{"percent":100},"b":{"percent":50}}}]}}`)
	out, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("get 输出非 JSON: %v", err)
	}
	exec, ok := m["execution"].(map[string]any)
	if !ok {
		t.Fatalf("缺 execution 段: %s", out)
	}
	if exec["reviewCount"].(float64) != 1 {
		t.Fatalf("reviewCount = %v", exec["reviewCount"])
	}
	tasks := m["tasks"].([]any)
	ta := tasks[0].(map[string]any)
	if af, exists := ta["askFinish"]; !exists || af.(float64) != 0 {
		t.Fatalf("任务行 askFinish=0 应在（*int 契约，不静默丢）: %v", ta)
	}
	tb := tasks[1].(map[string]any)
	if _, exists := tb["askFinish"]; exists {
		t.Fatalf("未设置任务行不应出 askFinish 键（omitempty）: %v", tb)
	}
	// 无信号计划 → 无 execution 键（不造假零值）
	schedApply(t, dir, `{"project":{"name":"Q","startDate":"2026-09-01",
		"tasks":[{"id":"x","name":"平整","level":1,"duration":2,"progress":0}]}}`)
	out2, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get2: %v", err)
	}
	var m2 map[string]any
	if err := json.Unmarshal([]byte(out2), &m2); err != nil {
		t.Fatalf("get2 输出非 JSON: %v", err)
	}
	if _, exists := m2["execution"]; exists {
		t.Fatalf("无信号不应出 execution 段: %s", out2)
	}
}

// schedule_get 视图库清单（斑马复刻 S6.2：视图=列可见性预设，viewbackend-re §6）。
func TestScheduleGetCarriesViews(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01",
		"tasks":[{"id":"a","name":"挖土","level":1,"duration":2,"progress":0}],
		"views":[{"id":"cv-1","name":"管控视图","kind":"gantt","hide":["ls","lf"],"groupBy":["status"],"sort":{"field":"start","dir":"asc"},"filter":"all"}]}}`)
	out, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("get 输出非 JSON: %v", err)
	}
	views, ok := m["views"].([]any)
	if !ok || len(views) != 1 {
		t.Fatalf("缺 views 清单: %s", out)
	}
	v := views[0].(map[string]any)
	if v["id"] != "cv-1" || v["name"] != "管控视图" || v["kind"] != "gantt" {
		t.Fatalf("views 行不齐: %v", v)
	}
	// 无视图 → 不出键
	schedApply(t, dir, `{"project":{"name":"Q","startDate":"2026-09-01",
		"tasks":[{"id":"x","name":"平整","level":1,"duration":2,"progress":0}]}}`)
	out2, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get2: %v", err)
	}
	var m2 map[string]any
	if err := json.Unmarshal([]byte(out2), &m2); err != nil {
		t.Fatalf("get2 输出非 JSON: %v", err)
	}
	if _, exists := m2["views"]; exists {
		t.Fatalf("无视图不应出 views 键: %s", out2)
	}
}

// 2026-11-21 方案模式复盘 A4/A5/B1/B4：结构化反序列化会静默吞掉错位字段，
// 预检在吞字段之前点名错法并给正解。
func TestSchedulePreScanRawRejectsMisplacedFields(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
	}{
		{"task 内 links", `{"project":{"name":"P","tasks":[{"id":"A1","name":"x","level":1,"duration":2,"links":[{"from":"A0","to":"A1"}]}],"links":[]}}`, "project 顶层"},
		{"workweek 传 7", `{"project":{"name":"P","tasks":[],"links":[],"calendar":{"workweek":[1,2,3,4,5,6,7]}}}`, "0-6"},
		{"op 字段名", `{"ops":[{"op":"patch_task","id":"A1","patch":{"duration":3}}]}`, "type 不是 op"},
		{"缺 type", `{"ops":[{"id":"A1"}]}`, "缺少 type"},
		{"set_links 写 taskId", `{"ops":[{"type":"set_links","taskId":"B6","links":[]}]}`, "toId 不是 taskId"},
		{"set_meta workweek 越界", `{"ops":[{"type":"set_meta","calendar":{"workweek":[0,1,7]}}]}`, "0-6"},
	}
	for _, c := range cases {
		err := schedulePreScanRaw(json.RawMessage(c.args))
		if err == nil {
			t.Errorf("%s：应拒绝（复盘静默丢弃面）", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s：错误应含 %q，got %v", c.name, c.want, err)
		}
	}
	// 合法载荷放行（七天连续=[0..6]）。
	if err := schedulePreScanRaw(json.RawMessage(`{"project":{"name":"P","tasks":[],"links":[],"calendar":{"workweek":[0,1,2,3,4,5,6]}}}`)); err != nil {
		t.Fatalf("合法 workweek [0..6] 不应拒绝: %v", err)
	}
	if err := schedulePreScanRaw(json.RawMessage(`{"ops":[{"type":"patch_task","id":"A10","patch":{"duration":15}}]}`)); err != nil {
		t.Fatalf("合法 ops 不应拒绝: %v", err)
	}
}

// B7：无前置发现点名到任务；cdTasks 无 cd 任务出 [] 不出 null。
func TestScheduleAnalyzeNamesNoPredAndCdTasksEmptyArray(t *testing.T) {
	dir := t.TempDir()
	schedApply(t, dir, `{"project":{"name":"P","startDate":"2026-09-01","tasks":[
		{"id":"a","name":"A","level":1,"duration":2},
		{"id":"b","name":"场地平整","level":1,"duration":2},
		{"id":"c","name":"C","level":1,"duration":2}],
		"links":[{"from":"a","to":"c","type":"FS","lag":0}]}}`)
	ana, err := scheduleAnalyze{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if !strings.Contains(ana, "1 个任务无前置搭接：场地平整(b)") {
		t.Fatalf("无前置发现应点名任务（B7）：%s", ana)
	}
	if !strings.Contains(ana, `"cdTasks":[]`) {
		t.Fatalf("cdTasks 空集应出 [] 不出 null（B7）：%s", ana)
	}
}
