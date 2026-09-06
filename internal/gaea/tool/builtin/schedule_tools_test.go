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

func TestScheduleApplyConfine(t *testing.T) {
	outside := t.TempDir()
	roots := t.TempDir()
	tool := scheduleApply{workDir: outside, roots: []string{roots}}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"project":{"name":"P","tasks":[],"links":[]}}`))
	if err == nil {
		t.Fatal("越界写入应拒绝")
	}
}

func TestScheduleGetMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := scheduleGet{workDir: dir}.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("缺文件应报错")
	}
}
