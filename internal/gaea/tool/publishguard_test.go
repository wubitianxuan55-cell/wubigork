package tool

// PublishGuard 单元腿（全平台 hermetic）+ P1b spike 腿（Windows 真实五类写入）。
//
// spike 背景（2026-11-09 众神殿 pan_1794234593182 裁决 action-2）：P1b 发布边界
// 工期是全计划唯一量级变量，spike 五类写入（直接重定向/脚本生成/改名/后台任务/
// 脚本文件）的拦提能力实证必须最先补。本文件即 spike 交付物：
//   - 单元腿：机制语义（created/modified/rename/非 durable 豁免/隔离移出/锁件豁免）；
//   - spike 腿：Windows cmd.exe 真实子进程按裁决五类逐一打点，验证 Begin→执行→
//     Scan→Quarantine 全链；后台类额外实证「前台 Scan 恒 clean、须 job 完成回调
//     再 Scan」这一 P1b 设计前提。
// spike 结论（2026-11-09 本机实跑）：五类全部可拦可提交判定；快照遍历开销见
// TestSpikePublishGuardSnapshotCost；P1b 按 4–6 人日基线推进，P4 不后置。

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func publishTestWorkspace(t *testing.T) string {
	t.Helper()
	ws := filepath.Join(t.TempDir(), "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	return ws
}

func newTestGuard(ws string) *PublishGuard {
	return NewPublishGuard([]string{ws}, filepath.Join(ws, ".gaea", "publish-quarantine"), nil)
}

func writePublishFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPublishGuardDetectsCreatedAndModified(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "base.xlsx"), "baseline")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	writePublishFile(t, filepath.Join(ws, "new.docx"), "new")
	writePublishFile(t, filepath.Join(ws, "base.xlsx"), "baseline-modified")
	writePublishFile(t, filepath.Join(ws, "notes.md"), "非 durable 交付物，v1 范围外")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 2 {
		t.Fatalf("Scan = %d violations, want 2（new.docx created + base.xlsx modified）: %+v", len(vs), vs)
	}
	byKind := map[string]string{}
	for _, v := range vs {
		byKind[v.Kind] = filepath.Base(v.Path)
		if v.RelPath == "" || v.Root == "" {
			t.Fatalf("violation 缺 Root/RelPath: %+v", v)
		}
	}
	if byKind["created"] != "new.docx" || byKind["modified"] != "base.xlsx" {
		t.Fatalf("kind 归属错误: %+v", byKind)
	}
}

func TestPublishGuardRenameIntoDurableIsCreated(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "stage.tmp"), "payload")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(ws, "stage.tmp"), filepath.Join(ws, "out.docx")); err != nil {
		t.Fatal(err)
	}
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Kind != "created" || filepath.Base(vs[0].Path) != "out.docx" {
		t.Fatalf("改名进 durable 档应判 created: %+v", vs)
	}
}

func TestPublishGuardQuarantineMovesOutOfWorkspace(t *testing.T) {
	ws := publishTestWorkspace(t)
	writePublishFile(t, filepath.Join(ws, "sub", "deep", "leak.pptx"), "越界件")
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	writePublishFile(t, filepath.Join(ws, "sub", "deep", "leak.pptx"), "越界件-改")
	vs, err := g.Scan()
	if err != nil || len(vs) != 1 {
		t.Fatalf("Scan = %+v err=%v, want 1 modified", vs, err)
	}
	moved, qerr := g.Quarantine(vs)
	if qerr != nil {
		t.Fatal(qerr)
	}
	if len(moved) != 1 {
		t.Fatalf("moved = %v", moved)
	}
	if _, err := os.Stat(filepath.Join(ws, "sub", "deep", "leak.pptx")); !os.IsNotExist(err) {
		t.Fatalf("越界件仍在正式工作区: %v", err)
	}
	// 隔离目录保留相对结构
	qfiles, _ := filepath.Glob(filepath.Join(ws, ".gaea", "publish-quarantine", "*", "sub", "deep", "leak.pptx"))
	if len(qfiles) != 1 {
		t.Fatalf("隔离目录未保留相对结构: %v", qfiles)
	}
	// 隔离后复扫恒 clean（隔离目录被剪枝，removed 不报）。
	vs2, err := g.Scan()
	if err != nil || len(vs2) != 0 {
		t.Fatalf("隔离后复扫应 clean: %+v err=%v", vs2, err)
	}
}

func TestPublishGuardTildeLockFilesIgnored(t *testing.T) {
	ws := publishTestWorkspace(t)
	g := newTestGuard(ws)
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	writePublishFile(t, filepath.Join(ws, "~$report.docx"), "Word 锁件")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("~$ 锁件应豁免: %+v", vs)
	}
}

func TestPublishGuardMissingRootAndQuarantinePruned(t *testing.T) {
	ws := publishTestWorkspace(t)
	g := newTestGuard(ws)
	// 根不存在=空快照合法；随后创建含 quarantine 自身的树不得自查。
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ws, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePublishFile(t, filepath.Join(ws, ".gaea", "publish-quarantine", "s1", "old.docx"), "隔离件")
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Fatalf("空根+纯隔离件应 clean: %+v", vs)
	}
}

func TestExpectedPathsNoteContent(t *testing.T) {
	note := ExpectedPathsNote([]PublishViolation{{RelPath: "a.docx", Kind: "created"}}, nil)
	if !strings.Contains(note, "a.docx") || !strings.Contains(note, "office_generate") {
		t.Fatalf("fail-loud 文案缺关键信息: %s", note)
	}
}

// TestExpectedPathsNoteGuidanceAlwaysPresent（2026-11-20 实测修正回归）：
// expected 非空（bash 带声明写路径）时出路指引句必须仍在——旧实现两句
// 互斥，带声明的拦截只见归因不见合规入口，用户绕行多轮。
func TestExpectedPathsNoteGuidanceAlwaysPresent(t *testing.T) {
	note := ExpectedPathsNote([]PublishViolation{{RelPath: "b.xlsx", Kind: "modified"}},
		[]string{`D:\w\b.xlsx`})
	if !strings.Contains(note, "声明写路径集") || !strings.Contains(note, "b.xlsx") {
		t.Fatalf("归因信息缺失: %s", note)
	}
	if !strings.Contains(note, "office_generate") || !strings.Contains(note, "docx_apply") {
		t.Fatalf("expected 非空时出路指引被吞: %s", note)
	}
}

// TestExpectedPathsNoteRecoveryAndAllowedSinks（2026-11-20 实测反馈⑤）：
// 拦截文案三件恒显——隔离到哪（dest 缺席时兜底可定位）、能不能取回、
// 允许的落点是什么。此前用户试错一轮才找到临时目录中转。
func TestExpectedPathsNoteRecoveryAndAllowedSinks(t *testing.T) {
	vs := []PublishViolation{{RelPath: "c.docx", Kind: "created"}}
	// dest 缺席：兜底句必须可定位（不许只剩「已隔离」）。
	note := ExpectedPathsNote(vs, nil)
	for _, want := range []string{"publish-quarantine", "隔离账本", "取回", "move_file", "允许落点", "非 durable"} {
		if !strings.Contains(note, want) {
			t.Fatalf("dest 缺席时文案缺 %q: %s", want, note)
		}
	}
	// dest 在场：具体路径优先，三件仍恒显。
	note = ExpectedPathsNote(vs, nil, `W:\ws\.gaea\publish-quarantine\20261120-120000`)
	if !strings.Contains(note, `20261120-120000`) || strings.Contains(note, "本次回执未取到") {
		t.Fatalf("dest 在场应显示具体路径: %s", note)
	}
	if !strings.Contains(note, "取回") || !strings.Contains(note, "允许落点") {
		t.Fatalf("取回/允许落点恒显被吞: %s", note)
	}
}

// --- spike 腿（Windows 真实子进程；非 Windows 跳过，CI linux 不受影响） ---

func runSpikeCmd(t *testing.T, dir, command string) time.Duration {
	t.Helper()
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cmd", "/c", command)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cmd %q 失败: %v\n%s", command, err, out)
	}
	return time.Since(start)
}

func spikeAssertCaught(t *testing.T, g *PublishGuard, wantRel string) PublishViolation {
	t.Helper()
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].RelPath != wantRel || vs[0].Kind != "created" {
		t.Fatalf("spike 拦截断言失败（want 1×created %s）: %+v", wantRel, vs)
	}
	return vs[0]
}

func spikeGuardFor(ws string) *PublishGuard {
	return NewPublishGuard([]string{ws}, filepath.Join(ws, ".gaea", "publish-quarantine"), nil)
}

func TestSpikePublishGuardFiveWriteCategories(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("spike 腿依赖 cmd.exe（Windows 实弹）；机制语义由上方单元腿全平台覆盖")
	}
	t.Run("redirect", func(t *testing.T) {
		ws := publishTestWorkspace(t)
		g := spikeGuardFor(ws)
		if err := g.Begin(); err != nil {
			t.Fatal(err)
		}
		runSpikeCmd(t, ws, `echo payload> out1.docx`)
		v := spikeAssertCaught(t, g, "out1.docx")
		moved, err := g.Quarantine([]PublishViolation{v})
		if err != nil || len(moved) != 1 {
			t.Fatalf("quarantine: %v %v", moved, err)
		}
	})
	t.Run("python_gen", func(t *testing.T) {
		py, err := exec.LookPath("python")
		if err != nil {
			t.Skip("python 不在 PATH；本类降级由 redirect/script 腿同机制覆盖")
		}
		ws := publishTestWorkspace(t)
		g := spikeGuardFor(ws)
		if err := g.Begin(); err != nil {
			t.Fatal(err)
		}
		// 实弹教训：cmd /c 对 python -c 的内联引号会被吃掉（exit 0 但没写文件
		// =python 进 REPL 静默退出，2026-11-09 probe 实锤）——脚本类负载直调
		// 解释器不经 cmd 引号转发；脚本文件形态由 script 腿覆盖。
		pyCmd := exec.Command(py, "-c", "open('out2.docx','wb').write(b'py')")
		pyCmd.Dir = ws
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		go func() { <-ctx.Done(); _ = pyCmd.Process.Kill() }()
		if out, err := pyCmd.CombinedOutput(); err != nil {
			t.Fatalf("python 写入失败: %v\n%s", err, out)
		} else if ctx.Err() != nil {
			t.Fatal("python 超时")
		}
		spikeAssertCaught(t, g, "out2.docx")
	})
	t.Run("rename", func(t *testing.T) {
		ws := publishTestWorkspace(t)
		g := spikeGuardFor(ws)
		if err := g.Begin(); err != nil {
			t.Fatal(err)
		}
		runSpikeCmd(t, ws, `echo x> stage.tmp && ren stage.tmp out3.docx`)
		spikeAssertCaught(t, g, "out3.docx")
	})
	t.Run("script", func(t *testing.T) {
		ws := publishTestWorkspace(t)
		g := spikeGuardFor(ws)
		if err := g.Begin(); err != nil {
			t.Fatal(err)
		}
		writePublishFile(t, filepath.Join(ws, "writer.cmd"), "@echo off\r\necho x> out5.docx\r\n")
		runSpikeCmd(t, ws, `writer.cmd`)
		spikeAssertCaught(t, g, "out5.docx")
	})
	t.Run("background", func(t *testing.T) {
		ws := publishTestWorkspace(t)
		g := spikeGuardFor(ws)
		if err := g.Begin(); err != nil {
			t.Fatal(err)
		}
		// 后台腿=「工具调用即返回、后代进程跨回合存活」形态（与 jobs.Manager
		// 后台 bash 同构：StartIn 返回 job 后进程继续跑）。实弹教训：cmd /c
		// start /b 嵌套引号会被吃错（曾起出交互式 cmd 永挂 CombinedOutput
		// 管道 30s）——延迟写负载落盘为批处理文件，Go 直接 Start 不 Wait，
		// 不挂管道。
		writePublishFile(t, filepath.Join(ws, "bg_writer.cmd"),
			"@echo off\r\nping -n 2 127.0.0.1 >nul\r\necho x> out4.docx\r\n")
		bg := exec.Command("cmd", "/c", "bg_writer.cmd")
		bg.Dir = ws
		if err := bg.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = bg.Process.Kill() }()
		// 关键前提实证：调用返回瞬间（后台写尚未落盘）前台 Scan 恒 clean。
		vs0, err := g.Scan()
		if err != nil {
			t.Fatal(err)
		}
		if len(vs0) != 0 {
			t.Fatalf("后台写尚未落盘时前台 Scan 应 clean（否则前台边界误报）: %+v", vs0)
		}
		// 等后台写落盘（≤8s），再 Scan=job 完成回调边界的拦截点。
		deadline := time.Now().Add(8 * time.Second)
		for {
			if _, err := os.Stat(filepath.Join(ws, "out4.docx")); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("后台写 8s 内未落盘，spike 环境异常")
			}
			time.Sleep(100 * time.Millisecond)
		}
		v := spikeAssertCaught(t, g, "out4.docx")
		moved, qerr := g.Quarantine([]PublishViolation{v})
		if qerr != nil || len(moved) != 1 {
			t.Fatalf("后台件 quarantine: %v %v", moved, qerr)
		}
	})
}

// TestSpikePublishGuardSnapshotCost 快照遍历开销定量（P1b 工期证据）：200 文件
// 两层树 Begin/Scan 应远小于一次 bash 调用时长（亚秒级）。
func TestSpikePublishGuardSnapshotCost(t *testing.T) {
	ws := publishTestWorkspace(t)
	for i := 0; i < 20; i++ {
		dir := filepath.Join(ws, "dir"+string(rune('a'+i%26)))
		for j := 0; j < 10; j++ {
			writePublishFile(t, filepath.Join(dir, "f"+string(rune('a'+j%26))+".txt"), "x")
		}
	}
	g := spikeGuardFor(ws)
	b0 := time.Now()
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	beginCost := time.Since(b0)
	s0 := time.Now()
	if _, err := g.Scan(); err != nil {
		t.Fatal(err)
	}
	scanCost := time.Since(s0)
	t.Logf("snapshot cost: begin=%v scan=%v (200 files)", beginCost, scanCost)
	if beginCost > 2*time.Second || scanCost > 2*time.Second {
		t.Fatalf("快照遍历开销异常（>2s）：begin=%v scan=%v", beginCost, scanCost)
	}
}

// TestPublishGuardProcessFileExemption 过程件豁免（P0-1 建议三，2026-11-21）：
// _draft/ 段路径与文件名含探针字样（probe/test/tmp/draft/草稿/临时）不判
// durable——实测往 _draft/probe/ 写探针 docx 也被拦、每次交付多绕路。
func TestPublishGuardProcessFileExemption(t *testing.T) {
	g, _ := d6Guard(t)
	ws := g.roots[0]
	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"_draft/build/过程稿.docx", // _draft 段豁免
		"_probe/探针.docx",        // _probe 段豁免
		"工作区根/probe探测.docx",     // 文件名 probe 豁免
		"工作区根/临时台账.docx",        // 文件名 临时 豁免
		"工作区根/正式交付物.docx",       // 不豁免（对照）
	} {
		abs := filepath.Join(ws, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(abs), 0o755)
		if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].RelPath != "工作区根/正式交付物.docx" {
		var got []string
		for _, v := range vs {
			got = append(got, v.RelPath)
		}
		t.Fatalf("应只拦正式交付物（过程件全豁免），实得 %v", got)
	}
}

// TestPublishGuardAtomicDirIsolation 目录级原子隔离（P1-12：mv 目录→骨架搬走
// +durable 被逐件隔离=半完成中间态）。真实 mv 形态=整目录在 Begin 前已存在
// （durable 与非 durable 伴行件 mtime 均旧），Begin 后整体搬入 → 整树隔离不
// 留空壳；往既有目录塞文件/目录内现写进行中内容=维持逐件。
func TestPublishGuardAtomicDirIsolation(t *testing.T) {
	g, _ := d6Guard(t)
	ws := g.roots[0]
	// Begin 前在工作区外备好整目录（mtime=旧，模拟随迁旧文件）。
	outer := filepath.Join(t.TempDir(), "交付包")
	os.MkdirAll(filepath.Join(outer, "子目录"), 0o755)
	os.WriteFile(filepath.Join(outer, "标书.docx"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(outer, "子目录", "说明.md"), []byte("y"), 0o644)
	os.WriteFile(filepath.Join(outer, "附表.xlsx"), []byte("z"), 0o644)
	// mtime 需严格早于 Begin——显式回拨 1 分钟防同刻精度边界。
	past := time.Now().Add(-time.Minute)
	os.Chtimes(filepath.Join(outer, "子目录", "说明.md"), past, past)
	os.Chtimes(filepath.Join(outer, "标书.docx"), past, past)
	os.Chtimes(filepath.Join(outer, "附表.xlsx"), past, past)

	if err := g.Begin(); err != nil {
		t.Fatal(err)
	}
	// 形态一：整目录 mv 进工作区（os.Rename 模拟 mv，mtime 保持旧值）。
	if err := os.Rename(outer, filepath.Join(ws, "交付包")); err != nil {
		t.Fatal(err)
	}
	// 形态二：往既有目录（工作区根，Begin 时已存在）现写 durable。
	os.WriteFile(filepath.Join(ws, "散件.docx"), []byte("w"), 0o644)
	// 形态三：新建目录+现写进行中内容（docx+md 都是新文件）→ 不得整搬。
	wipDir := filepath.Join(ws, "编稿中")
	os.MkdirAll(wipDir, 0o755)
	os.WriteFile(filepath.Join(wipDir, "草稿.md"), []byte("正在写"), 0o644)
	os.WriteFile(filepath.Join(wipDir, "稿件.docx"), []byte("d"), 0o644)

	vs, err := g.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 4 {
		t.Fatalf("应检出 4 件 durable 越界（散件/交付包2/编稿中1（稿件）），实得 %d", len(vs))
	}
	moved, qerr := g.Quarantine(vs)
	if qerr != nil {
		t.Fatal(qerr)
	}
	if len(moved) != 5 {
		t.Fatalf("moved 应 5 件（整目录3+散件1+编稿中 docx 逐件1，md 留守），实得 %v", moved)
	}
	// 整目录连骨架消失；编稿中目录与 md 留在工作区（进行中内容不动）。
	if _, err := os.Stat(filepath.Join(ws, "交付包")); !os.IsNotExist(err) {
		t.Fatal("整目录应连骨架一起隔离，不留空壳")
	}
	if _, err := os.Stat(filepath.Join(wipDir, "草稿.md")); err != nil {
		t.Fatalf("进行中的 md 不应被整目录带走: %v", err)
	}
	dest := g.LastQuarantineDest()
	for _, rel := range []string{"标书.docx", "附表.xlsx", filepath.Join("子目录", "说明.md")} {
		if _, err := os.Stat(filepath.Join(dest, "交付包", rel)); err != nil {
			t.Fatalf("隔离区缺整目录成员 %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "散件.docx")); err != nil {
		t.Fatalf("隔离区缺散件: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "编稿中", "稿件.docx")); err != nil {
		t.Fatalf("编稿中 docx 应逐件隔离: %v", err)
	}
}
