package app

// AP2-02 反向证据钉子：runBookJob 模板的 panic 防线与注销防线的既有用例。
// 审计第 19 批线 1——收敛前三处 goroutine 骨架无此覆盖，收敛后防线收进单一
// 模板，本文件把「拍掉 recover / 拍掉注销 defer 必红」钉死。

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/booksource"
)

// emitRecorder 收集 runBookJob 发出的事件（直接注入 emit 方法值，不走 SSE）。
type emitRecorder struct {
	mu     sync.Mutex
	names  []string
	stacks []map[string]interface{}
}

func (r *emitRecorder) emit(name string, data map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = append(r.names, name)
	r.stacks = append(r.stacks, data)
}

func (r *emitRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.names)
}

func (r *emitRecorder) snapshot() ([]string, []map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.names...), append([]map[string]interface{}{}, r.stacks...)
}

// waitEvents 等后台协程发出至少 n 个事件（goroutine 时序解耦）。
func (r *emitRecorder) waitEvents(t *testing.T, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for r.count() < n {
		if time.Now().After(deadline) {
			names, _ := r.snapshot()
			t.Fatalf("等待 %d 个事件超时，实得 %d 个：%v", n, r.count(), names)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// waitUnregistered 等待 jobID 从取消簿消失（注销 defer 已执行）。
func waitUnregistered(t *testing.T, jobID string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		bookImportMu.Lock()
		_, ok := bookImportRuns[jobID]
		bookImportMu.Unlock()
		if !ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("job %s 完成后仍留在取消簿（注销 defer 未执行）", jobID)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestRunBookJob_PanicTurnsIntoErrorEvent AP2-02 钉子：work panic 必须被模板
// recover 转 error 事件（事件名=通道前缀+jobID，文案=panicPrefix+": 原文"），
// 且注销 defer 照常执行（先 recover 后注销的 defer 顺序）。拍掉模板里的
// recover → 本用例红（goroutine panic 带崩测试进程）；拍掉注销 defer → 本用例
// 尾段红。
func TestRunBookJob_PanicTurnsIntoErrorEvent(t *testing.T) {
	rec := &emitRecorder{}
	jobID := runBookJob(rec.emit, bookJobSpec{
		jobPrefix:   "tst_",
		eventPrefix: "test-bookjob:",
		panicPrefix: "导入异常",
		doneType:    "done",
		work: func(ctx context.Context, onProgress func(done, total int)) (any, []booksource.FailedChapter, error) {
			panic("畸形书源内容") // 模拟解析面对畸形内容 panic
		},
	})

	rec.waitEvents(t, 1)
	names, stacks := rec.snapshot()
	if len(names) != 1 {
		t.Fatalf("panic 路径应只发 1 个事件，得到 %d 个：%v", len(names), names)
	}
	if names[0] != "test-bookjob:"+jobID {
		t.Fatalf("事件通道应为 test-bookjob:%s，得到 %s", jobID, names[0])
	}
	if got := stacks[0]["type"]; got != "error" {
		t.Fatalf("panic 应转 error 事件，得到 type=%v", got)
	}
	if got, _ := stacks[0]["error"].(string); !strings.HasPrefix(got, "导入异常: ") || !strings.Contains(got, "畸形书源内容") {
		t.Fatalf("error 文案应为「导入异常: …畸形书源内容…」，得到 %q", got)
	}
	// panic 防线转事件后，注销 defer 仍要执行（jobID 出簿，可被重复起跑顶替）。
	waitUnregistered(t, jobID)
}

// TestRunBookJob_LifecycleThrottleAndDonePayload AP2-02 钉子：正常路径骨架——
// onProgress 节流（步长整数倍/完成必发、其余不发）、done 载荷（doneType/result/
// failed 原样清单）、完成后 jobID 出簿。拍掉注销 defer → 尾段红。
func TestRunBookJob_LifecycleThrottleAndDonePayload(t *testing.T) {
	rec := &emitRecorder{}
	jobID := runBookJob(rec.emit, bookJobSpec{
		jobPrefix:      "tsd_",
		eventPrefix:    "test-done:",
		panicPrefix:    "导入异常",
		doneType:       "done",
		errWithFailed:  true,
		doneWithFailed: true,
		work: func(ctx context.Context, onProgress func(done, total int)) (any, []booksource.FailedChapter, error) {
			onProgress(19, 40) // 非步长非完成 → 不发
			onProgress(20, 40) // 步长 → 发
			onProgress(21, 40) // 不发
			onProgress(40, 40) // 完成 → 必发
			return map[string]any{"path": "x"}, []booksource.FailedChapter{{URL: "u1"}}, nil
		},
	})

	rec.waitEvents(t, 3) // progress(20) + progress(40) + done
	names, stacks := rec.snapshot()
	var progresses []map[string]interface{}
	var done map[string]interface{}
	for i, name := range names {
		switch stacks[i]["type"] {
		case "progress":
			if name != "test-done:"+jobID {
				t.Fatalf("progress 通道应为 test-done:%s，得到 %s", jobID, name)
			}
			progresses = append(progresses, stacks[i])
		case "done":
			if done != nil {
				t.Fatalf("done 事件应只发一次，得到多次：%v", names)
			}
			done = stacks[i]
		default:
			t.Fatalf("正常路径不应有 %v 事件：%v", stacks[i]["type"], names)
		}
	}
	if len(progresses) != 2 {
		t.Fatalf("节流后应只发 2 次 progress（20 与 40），得到 %d 次：%v", len(progresses), progresses)
	}
	if done == nil {
		t.Fatalf("缺 done 终态事件：%v", names)
	}
	if done["result"] == nil {
		t.Fatalf("done 载荷应带 result，得到 %v", done)
	}
	failed, ok := done["failed"].([]booksource.FailedChapter)
	if !ok || len(failed) != 1 || failed[0].URL != "u1" {
		t.Fatalf("done 载荷应原样带 failed 清单，得到 %#v", done["failed"])
	}
	waitUnregistered(t, jobID)
}

// TestRunBookJob_ErrorPayloadShapes AP2-02 钉子：error 终态载荷按 errWithFailed
// 开关带/不带 failed 计数；doneWithFailed=false 的 done 不出现 failed 键
// （补下/原罪成书与整本导入的载荷差异面，逐字段冻结）。
func TestRunBookJob_ErrorPayloadShapes(t *testing.T) {
	rec := &emitRecorder{}
	jobID := runBookJob(rec.emit, bookJobSpec{
		jobPrefix:      "tse_",
		eventPrefix:    "test-err:",
		panicPrefix:    "导入异常",
		doneType:       "append-done",
		errWithFailed:  true,
		doneWithFailed: false,
		work: func(ctx context.Context, onProgress func(done, total int)) (any, []booksource.FailedChapter, error) {
			return nil, []booksource.FailedChapter{{URL: "a"}, {URL: "b"}}, context.DeadlineExceeded
		},
	})
	rec.waitEvents(t, 1)
	_, stacks := rec.snapshot()
	if stacks[0]["type"] != "error" {
		t.Fatalf("work 返回 err 应走 error 终态，得到 %v", stacks[0]["type"])
	}
	if n := stacks[0]["failed"]; n != 2 {
		t.Fatalf("error 载荷应带 failed 计数 2，得到 %#v", stacks[0]["failed"])
	}

	rec2 := &emitRecorder{}
	jobID2 := runBookJob(rec2.emit, bookJobSpec{
		jobPrefix:      "tsf_",
		eventPrefix:    "test-ok:",
		panicPrefix:    "下载异常",
		doneType:       "done",
		errWithFailed:  false,
		doneWithFailed: false,
		work: func(ctx context.Context, onProgress func(done, total int)) (any, []booksource.FailedChapter, error) {
			return "res", nil, nil
		},
	})
	rec2.waitEvents(t, 1)
	_, stacks2 := rec2.snapshot()
	if stacks2[0]["type"] != "done" {
		t.Fatalf("成功应走 done 终态，得到 %v", stacks2[0]["type"])
	}
	if _, has := stacks2[0]["failed"]; has {
		t.Fatalf("doneWithFailed=false 时 done 载荷不得出现 failed 键：%v", stacks2[0])
	}
	if _, has := stacks2[0]["result"]; !has {
		t.Fatalf("done 载荷应带 result：%v", stacks2[0])
	}
	waitUnregistered(t, jobID)
	waitUnregistered(t, jobID2)
}
