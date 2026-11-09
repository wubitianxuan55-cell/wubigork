package app

// P4 持久任务中心·办公批处理测试（2026-11-10 众神殿 pan_1794234593182 裁决
// action-7）：批量转 PDF 进任务调度器——提交即返回、进度逐文件、ctx 可取消、
// 全败才 failed、重试 re-enqueue 同轨。转换 seam 注入（无 soffice 依赖）；
// 断窗续跑（Startup 重排 queued/running）由 tasks 包 Start 持久机械钉住，
// 此处验办公任务走同轨（Retry 后新轮 succeeded）。

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/tasks"
)

// batchSeams 转换 seam 注入（ctx 供取消观测）。
func batchSeams(convert func(ctx context.Context, src, out string) error) func() {
	old := officeBatchConvert
	officeBatchConvert = convert
	return func() { officeBatchConvert = old }
}

func setupBatchApp(t *testing.T) *App {
	t.Helper()
	a := newTestTaskApp(t)
	a.officeState.tasks.Register(tasks.KindOfficeBatch, a.officeBatchTaskHandler)
	return a
}

func seedBatchFiles(t *testing.T, ws string, names []string) []string {
	t.Helper()
	var rels []string
	for _, n := range names {
		full := filepath.Join(ws, n)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("fake office bytes"), 0o644); err != nil {
			t.Fatal(err)
		}
		rels = append(rels, n)
	}
	return rels
}

func TestOfficeBatchTaskSucceedsPerFileReport(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	a := setupBatchApp(t)
	defer batchSeams(func(ctx context.Context, src, out string) error {
		return os.WriteFile(out, []byte("%PDF-fake"), 0o644)
	})()

	seedBatchFiles(t, ws, []string{"报告A.docx", "台账B.xlsx", "演示C.pptx"})
	out, err := officeBatchTool{a: a}.Execute(context.Background(), []byte(
		`{"files":["报告A.docx","台账B.xlsx","演示C.pptx"],"out_dir":"产物/batch"}`))
	if err != nil {
		t.Fatalf("提交: %v (%s)", err, out)
	}
	if !strings.Contains(out, "任务 tsk_") || !strings.Contains(out, "产物/batch") {
		t.Fatalf("提交回执缺任务 id/声明目录: %q", out)
	}
	startTasks(t, a)
	id := extractBatchTaskID(out)
	task, err := a.officeState.tasks.Wait(context.Background(), id, 10*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if task.Status != string(tasks.StatusSucceeded) {
		t.Fatalf("status=%s err=%s result=%s", task.Status, task.Error, task.Result)
	}
	var res struct {
		Ok    int `json:"ok"`
		Fail  int `json:"fail"`
		Files []struct {
			File, Out, Status string
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(task.Result), &res); err != nil {
		t.Fatalf("Result 非法: %v", err)
	}
	if res.Ok != 3 || res.Fail != 0 {
		t.Fatalf("逐文件报告 ok=%d fail=%d", res.Ok, res.Fail)
	}
	// 声明 out_dir=实际产物目录（P1a 声明纪律）：三件全部落在 产物/batch
	for _, f := range res.Files {
		if !strings.HasPrefix(f.Out, "产物/batch/") || !strings.HasSuffix(f.Out, ".pdf") {
			t.Fatalf("落点越出声明目录: %+v", f)
		}
		if _, err := os.Stat(filepath.Join(ws, filepath.FromSlash(f.Out))); err != nil {
			t.Fatalf("产物缺失 %s: %v", f.Out, err)
		}
	}
}

func TestOfficeBatchTaskAllFailFailsHonest(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	a := setupBatchApp(t)
	defer batchSeams(func(ctx context.Context, src, out string) error {
		return fmt.Errorf("soffice not found")
	})()

	seedBatchFiles(t, ws, []string{"坏件.docx"})
	out, err := officeBatchTool{a: a}.Execute(context.Background(), []byte(
		`{"files":["坏件.docx"],"out_dir":"产物/batch"}`))
	if err != nil {
		t.Fatalf("提交: %v", err)
	}
	startTasks(t, a)
	task, err := a.officeState.tasks.Wait(context.Background(), extractBatchTaskID(out), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != string(tasks.StatusFailed) || !strings.Contains(task.Error, "全部 1 个文件转换失败") {
		t.Fatalf("全败应 failed+诚实口径: status=%s err=%s", task.Status, task.Error)
	}
	// 重试同轨（机械继承）：换好 seam → Retry re-enqueue → succeeded
	defer batchSeams(func(ctx context.Context, src, out string) error {
		return os.WriteFile(out, []byte("%PDF-fake"), 0o644)
	})()
	if err := a.officeState.tasks.Retry(task.ID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	task2, err := a.officeState.tasks.Wait(context.Background(), task.ID, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if task2.Status != string(tasks.StatusSucceeded) {
		t.Fatalf("重试后应 succeeded: %s %s", task2.Status, task2.Error)
	}
}

func TestOfficeBatchTaskCancelTwoPhase(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	a := setupBatchApp(t)
	// seam 阻塞到 ctx 取消（模拟长转换）
	defer batchSeams(func(ctx context.Context, src, out string) error {
		<-ctx.Done() // 模拟长转换：任务取消传导进来
		return ctx.Err()
	})()

	seedBatchFiles(t, ws, []string{"大件.docx"})
	out, err := officeBatchTool{a: a}.Execute(context.Background(), []byte(
		`{"files":["大件.docx"],"out_dir":"产物/batch"}`))
	if err != nil {
		t.Fatalf("提交: %v", err)
	}
	startTasks(t, a)
	id := extractBatchTaskID(out)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if tk, err := a.officeState.tasks.Get(id); err == nil && tk.Status == string(tasks.StatusRunning) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := a.officeState.tasks.Cancel(id); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	task, err := a.officeState.tasks.Wait(context.Background(), id, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != string(tasks.StatusCancelled) {
		t.Fatalf("取消应 cancelled（两段式收口）: %s", task.Status)
	}
}

func TestOfficeBatchToolRejects(t *testing.T) {
	ws := t.TempDir()
	isolateWorkspaceTo(t, ws)
	a := setupBatchApp(t)
	tl := officeBatchTool{a: a}
	// 空 files
	if _, err := tl.Execute(context.Background(), []byte(`{"files":[],"out_dir":"o"}`)); err == nil {
		t.Fatal("空 files 应拒绝")
	}
	// 缺 out_dir（声明写路径纪律）
	if _, err := tl.Execute(context.Background(), []byte(`{"files":["a.docx"]}`)); err == nil || !strings.Contains(err.Error(), "out_dir") {
		t.Fatalf("缺 out_dir 应拒绝: %v", err)
	}
	// 超批上限
	files := make([]string, officeBatchMaxFiles+1)
	for i := range files {
		files[i] = "x.docx"
	}
	b, _ := json.Marshal(map[string]any{"files": files, "out_dir": "o"})
	if _, err := tl.Execute(context.Background(), b); err == nil || !strings.Contains(err.Error(), "拆批") {
		t.Fatalf("超批应拒绝: %v", err)
	}
	// WritePaths 声明=产物目录（越界 out_dir 拒绝）
	if got, err := (officeBatchTool{}).WritePaths([]byte(`{"out_dir":"产物/batch"}`)); err != nil || len(got) != 1 ||
		got[0] != filepath.Join(ws, "产物", "batch") {
		t.Fatalf("WritePaths=%v err=%v", got, err)
	}
	if _, err := (officeBatchTool{}).WritePaths([]byte(`{"out_dir":"../逃逸"}`)); err == nil {
		t.Fatal("越界 out_dir 应拒绝")
	}
}

// extractBatchTaskID 从提交回执提取 tsk_ 任务 id。
func extractBatchTaskID(out string) string {
	i := strings.Index(out, "tsk_")
	if i < 0 {
		return ""
	}
	rest := out[i:]
	for j := 0; j < len(rest); j++ {
		c := rest[j]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_') {
			return rest[:j]
		}
	}
	return rest
}
