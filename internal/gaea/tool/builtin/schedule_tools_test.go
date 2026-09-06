package builtin

// schedule_tools_test.go — 进度计划三工具 Execute 级测试（v4.113.0 刀4）：
// ops/project 双通道、环 fail-closed、写根越界拒绝、get/analyze 回读。

import (
	"context"
	"encoding/json"
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
