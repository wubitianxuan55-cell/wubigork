package builtin

// P1b 发布边界接线测试（2026-11-09 众神殿裁决 action-4）：真 bash 工具级——
//   - 前台：durable 写命中 → fail-loud + 文件被隔离出工作区；
//   - 后台：job 完成=拦截点（spike 前提：调用返回时恒 clean）→ job 终态失败；
//   - 非 durable（.md/.csv）：放行零干扰（D1 v1 口径）。
// 机制单元腿在 internal/gaea/tool/publishguard_test.go（含五类写入 spike）。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/jobs"
)

func publishBashWorkdir(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestBashPublishGuardBlocksForegroundDurableWrite(t *testing.T) {
	ws := publishBashWorkdir(t)
	b := bash{workDir: ws}
	out, err := b.Execute(context.Background(),
		// bash/pwsh 两形态都成立的写法：echo 重定向落 docx（durable 档）。
		// pwsh echo 产生 UTF16/换行差异不影响存在性判定。
		[]byte(`{"command":"echo payload > report.docx"}`))
	if err == nil {
		t.Fatalf("durable 越界写应 fail-loud, got out=%q", out)
	}
	if !strings.Contains(err.Error(), "发布边界拦截") || !strings.Contains(err.Error(), "report.docx") {
		t.Fatalf("fail-loud 文案缺关键信息: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(ws, "report.docx")); !os.IsNotExist(serr) {
		t.Fatal("越界件未隔离出正式工作区")
	}
	q, _ := filepath.Glob(filepath.Join(ws, ".gaea", "publish-quarantine", "*", "report.docx"))
	if len(q) != 1 {
		t.Fatalf("隔离目录未收到越界件: %v", q)
	}
}

func TestBashPublishGuardPassesNonDurableWrite(t *testing.T) {
	ws := publishBashWorkdir(t)
	b := bash{workDir: ws}
	out, err := b.Execute(context.Background(),
		[]byte(`{"command":"echo payload > notes.md"}`))
	if err != nil {
		t.Fatalf("非 durable（.md）写应放行（D1 v1 口径）: %v（out=%q）", err, out)
	}
	raw, rerr := os.ReadFile(filepath.Join(ws, "notes.md"))
	if rerr != nil {
		t.Fatalf("非 durable 文件应保留: %v", rerr)
	}
	if !strings.Contains(string(raw), "payload") {
		t.Fatalf("非 durable 文件内容异常: %q", string(raw))
	}
}

func TestBashPublishGuardBlocksBackgroundJobDurableWrite(t *testing.T) {
	ws := publishBashWorkdir(t)
	b := bash{workDir: ws}
	m := jobs.NewManager(event.FuncSink(func(event.Event) {}))
	ctx := jobs.WithManager(context.Background(), m)
	defer m.Close()
	out, err := b.Execute(ctx, []byte(`{"command":"sleep 1 && echo payload > bg.docx","run_in_background":true}`))
	if err != nil {
		t.Fatalf("后台启动不应失败: %v", err)
	}
	if !strings.Contains(out, "Started background job") {
		t.Fatalf("后台回执异常: %q", out)
	}
	// job 完成=拦截点：轮询 job 终态（≤10s）。
	deadline := time.Now().Add(10 * time.Second)
	jobID := extractJobID(out)
	var gotStatus jobs.Status
	var gotText string
	for time.Now().Before(deadline) {
		if _, status, ok := m.Output(jobID); ok && status != jobs.Running {
			res := m.Wait(context.Background(), []string{jobID}, 2)
			if len(res) > 0 {
				gotStatus = res[0].Status
				gotText = res[0].Output
			}
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if gotStatus != jobs.Failed {
		t.Fatalf("后台 job 未以失败终态收场: status=%v text=%q", gotStatus, gotText)
	}
	if !strings.Contains(gotText, "发布边界拦截") {
		t.Fatalf("后台 fail-loud 文案缺关键信息: %s", gotText)
	}
	if _, serr := os.Stat(filepath.Join(ws, "bg.docx")); !os.IsNotExist(serr) {
		t.Fatal("后台越界件未隔离出正式工作区")
	}
}

// extractJobID 从后台启动回执提取 job id（回执格式见 bash.go 后台分支）。
func extractJobID(out string) string {
	const marker = `job_id="`
	i := strings.Index(out, marker)
	if i < 0 {
		return ""
	}
	rest := out[i+len(marker):]
	if j := strings.Index(rest, `"`); j >= 0 {
		return rest[:j]
	}
	return ""
}
