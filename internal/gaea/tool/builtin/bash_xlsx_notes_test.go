package builtin

// bash_xlsx_notes_test.go — 2026-11-27 AI 反馈 #7 回归钉：命令写出的 xlsx 没有
// 公式缓存值时，回执尾部要提一句（库保存不写公式结果，Excel/WPS 打开是空值，
// 而命令本身 rc=0）。备注只提示、不隔离、不置错；隔离件按隔离后的落点体检。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/jobs"
	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/xuri/excelize/v2"
)

// xlsxNotesWorkspace 工作区里预置三份种子件（非 durable 扩展名，不进发布边界
// 快照）：nocache.bin = 有公式没缓存值；recalced.bin = 重算过的真实样表；
// junk.bin = 根本不是 zip。命令只做相对路径拷贝，bash 与 PowerShell 写法一致。
func xlsxNotesWorkspace(t *testing.T) string {
	t.Helper()
	ws := publishBashWorkdir(t)
	seed := filepath.Join(ws, "seed")
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, ".gaea", "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seed, "nocache.bin"), xlsxNoCacheBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	sample, err := os.ReadFile(filepath.Join("..", "..", "skill", "assets", "cost-estimate-sample.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seed, "recalced.bin"), sample, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seed, "junk.bin"), []byte("not a zip at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	return ws
}

func TestBashXlsxCacheNotes(t *testing.T) {
	ws := xlsxNotesWorkspace(t)
	b := bash{shell: sandbox.ResolveShell(), workDir: ws}
	ctx := context.Background()

	t.Run("缺缓存值的过程件要点名", func(t *testing.T) {
		out, err := b.Execute(ctx, []byte(`{"command":"cp seed/nocache.bin .gaea/work/plain.xlsx"}`))
		if err != nil {
			t.Fatalf("过程目录是豁免面：备注不隔离、不置错，got err=%v out=%q", err, out)
		}
		for _, want := range []string{"命令写出的 .gaea/work/plain.xlsx", "公式 1 格，其中 1 格没有缓存值", "xlsx_recalc"} {
			if !strings.Contains(out, want) {
				t.Errorf("回执缺 %q：\n%s", want, out)
			}
		}
	})

	t.Run("json 模式进 xlsx_notes 字段", func(t *testing.T) {
		out, err := b.Execute(ctx, []byte(`{"command":"cp seed/nocache.bin .gaea/work/json.xlsx","output_format":"json"}`))
		if err != nil {
			t.Fatalf("备注不置错，got err=%v out=%q", err, out)
		}
		var res map[string]any
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatalf("json 模式回执应是合法 JSON：%v\n%s", err, out)
		}
		note, _ := res["xlsx_notes"].(string)
		if !strings.Contains(note, ".gaea/work/json.xlsx") || !strings.Contains(note, "没有缓存值") {
			t.Errorf("xlsx_notes 应点名文件与缺缓存值，实得 %q", note)
		}
	})

	t.Run("重算过的文件不提", func(t *testing.T) {
		out, err := b.Execute(ctx, []byte(`{"command":"cp seed/recalced.bin .gaea/work/fine.xlsx"}`))
		if err != nil {
			t.Fatalf("got err=%v out=%q", err, out)
		}
		if strings.Contains(out, "命令写出的") {
			t.Errorf("重算过的文件不该出体检备注：\n%s", out)
		}
	})

	t.Run("坏文件如实说打不开", func(t *testing.T) {
		out, err := b.Execute(ctx, []byte(`{"command":"cp seed/junk.bin .gaea/work/broken.xlsx"}`))
		if err != nil {
			t.Fatalf("打不开也是备注不是错误，got err=%v out=%q", err, out)
		}
		if !strings.Contains(out, "命令写出的 .gaea/work/broken.xlsx 打不开") {
			t.Errorf("坏文件应报打不开：\n%s", out)
		}
	})

	t.Run("非 xlsx 不体检", func(t *testing.T) {
		out, err := b.Execute(ctx, []byte(`{"command":"cp seed/junk.bin notes.txt"}`))
		if err != nil {
			t.Fatalf("got err=%v out=%q", err, out)
		}
		if strings.Contains(out, "命令写出的") || strings.Contains(out, "打不开") {
			t.Errorf("非 xlsx 不该有体检备注：\n%s", out)
		}
	})
}

// TestBashXlsxCacheNotesOnQuarantinedFile 正式名下的越界件：隔离文案照旧（result
// 与 err 都带），体检备注跟在后面——按隔离后的落点体检，不把「被移走」说成「打不开」。
func TestBashXlsxCacheNotesOnQuarantinedFile(t *testing.T) {
	ws := xlsxNotesWorkspace(t)
	b := bash{shell: sandbox.ResolveShell(), workDir: ws}
	out, err := b.Execute(context.Background(), []byte(`{"command":"cp seed/nocache.bin out.xlsx"}`))
	if err == nil {
		t.Skipf("发布边界未拦截，跳过")
	}
	if !strings.HasPrefix(err.Error(), "发布边界拦截") {
		t.Errorf("err 首行（信封 error 字段）仍应是拦截标题：%v", err)
	}
	for _, want := range []string{"发布边界拦截", "取回", "命令写出的 out.xlsx", "没有缓存值", "xlsx_recalc"} {
		if !strings.Contains(out, want) || !strings.Contains(err.Error(), want) {
			t.Errorf("result 与 err 都应含 %q：\nout=%s\nerr=%v", want, out, err)
		}
	}
	if strings.Contains(out, "打不开") {
		t.Errorf("隔离件原位不在不等于打不开：\n%s", out)
	}
	if _, serr := os.Stat(filepath.Join(ws, "out.xlsx")); !os.IsNotExist(serr) {
		t.Error("越界件未隔离出正式工作区")
	}
}

// TestBashXlsxCacheNotesBackgroundJob 后台 job：备注进 job 缓冲（bash_output /
// wait 读的是它），job 以 Done 收场。
func TestBashXlsxCacheNotesBackgroundJob(t *testing.T) {
	ws := xlsxNotesWorkspace(t)
	b := bash{workDir: ws}
	m := jobs.NewManager(event.FuncSink(func(event.Event) {}))
	ctx := jobs.WithManager(context.Background(), m)
	defer m.Close()
	out, err := b.Execute(ctx, []byte(`{"command":"cp seed/nocache.bin .gaea/work/bg.xlsx","run_in_background":true}`))
	if err != nil {
		t.Fatalf("后台启动不应失败：%v", err)
	}
	jobID := extractJobID(out)
	deadline := time.Now().Add(10 * time.Second)
	var gotStatus jobs.Status
	var gotText string
	for time.Now().Before(deadline) {
		if _, status, ok := m.Output(jobID); ok && status != jobs.Running {
			if res := m.Wait(context.Background(), []string{jobID}, 2); len(res) > 0 {
				gotStatus, gotText = res[0].Status, res[0].Output
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if gotStatus != jobs.Done {
		t.Fatalf("后台 job 应以 Done 收场（备注不置错）：status=%v text=%q", gotStatus, gotText)
	}
	for _, want := range []string{"命令写出的 .gaea/work/bg.xlsx", "没有缓存值"} {
		if !strings.Contains(gotText, want) {
			t.Errorf("后台输出缺 %q：%s", want, gotText)
		}
	}
}

// TestXlsxCacheNotesUnsettledSkipsUnopenable 进程还在后台跑（提前返回）时不报
// 「打不开」：半写的文件不算坏；收场后再如实报。
func TestXlsxCacheNotesUnsettledSkipsUnopenable(t *testing.T) {
	if xlsxCacheNotes(nil, true) != "" {
		t.Fatal("无守卫（非工作区装配）不应出备注")
	}
	ws := publishBashWorkdir(t)
	work := filepath.Join(ws, ".gaea", "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	g := tool.NewPublishGuard([]string{ws}, filepath.Join(ws, ".gaea", "publish-quarantine"), nil)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	// 过程目录里落下两件：一份半写（不是 zip）、一份有公式没缓存值。
	if err := os.WriteFile(filepath.Join(work, "junk.xlsx"), []byte("half"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "nocache.xlsx"), xlsxNoCacheBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	vs, err := g.Scan()
	if err != nil || len(vs) != 0 {
		t.Fatalf("过程目录应豁免：%v %v", vs, err)
	}
	running := xlsxCacheNotes(g, false)
	if !strings.Contains(running, "没有缓存值") || strings.Contains(running, "打不开") {
		t.Errorf("进程未收场：缺缓存值照报，但不该报打不开（半写不算坏）：%q", running)
	}
	settled := xlsxCacheNotes(g, true)
	if strings.Count(settled, "打不开") != 1 || strings.Count(settled, "没有缓存值") != 1 {
		t.Errorf("收场后两件各报一条：%q", settled)
	}
}

// xlsxNoCacheBytes 一份「有公式、没缓存值」的工作簿字节（excelize 写出的形态）。
func xlsxNoCacheBytes(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetCellValue(sheet, "A1", 2)
	_ = f.SetCellFormula(sheet, "A2", "A1*3")
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return buf.Bytes()
}
