package proc

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// T13（codex 蒸馏）验收：Drain 不重叠 / 省略标记唯一且计数准确 / 双闸门
// 时间预算 / 退出收尾上限 / 写已退会话明确报错 / stdin 审批阈值。
//
// 测试用测试二进制自重执行（TestMain 检测 env）驱动真实子进程，
// 不依赖平台 shell。

const sessionHelperEnv = "GAEA_PROC_SESSION_HELPER"

func TestMain(m *testing.M) {
	if v := os.Getenv(sessionHelperEnv); v == "1" || v == "quick" {
		runSessionHelper()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runSessionHelper 分相位输出 + 读一行 stdin 回显，给会话层提供真实子进程。
// GAEA_PROC_SESSION_HELPER=quick 时只输出一行即退出（自然退出路径的测试用）。
func runSessionHelper() {
	out := bufio.NewWriter(os.Stdout)
	if os.Getenv(sessionHelperEnv) == "quick" {
		fmt.Fprintln(out, "quick-exit-line")
		out.Flush()
		return
	}
	fmt.Fprintln(out, "phase-1-alpha")
	out.Flush()
	time.Sleep(400 * time.Millisecond)
	fmt.Fprintln(out, "phase-2-beta")
	out.Flush()
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Fprintf(out, "got:%s", line)
	out.Flush()
	time.Sleep(100 * time.Millisecond)
}

// helperArgv 构造自重执行 argv（跨平台，无 shell）。
func helperArgv(t *testing.T) []string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("executable: %v", err)
	}
	// 测试二进制需要知道跑哪个测试的 helper 形态：env 开关即可（main 分支）。
	cmd := []string{exe, "-test.run=TestSessionHelperEntry"}
	if runtime.GOOS == "windows" {
		// Windows 下测试二进制直接重入即可。
		_ = cmd
	}
	return []string{exe, "-test.run=TestSessionHelperEntry"}
}

// TestSessionHelperEntry 是 helper 进程的入口桩（真实逻辑在 TestMain 分支）。
func TestSessionHelperEntry(t *testing.T) {
	// 被 -test.run 命中但 env 未置位时（正常测试运行）直接跳过。
	if os.Getenv(sessionHelperEnv) != "1" {
		t.Skip("helper entry")
	}
}

// ① 双闸门：yieldMS=100 的调用在 ≈100ms 返回（±30% 宽松取 ±150ms），进程仍在跑。
func TestRunSessionYieldDeadline(t *testing.T) {
	os.Setenv(sessionHelperEnv, "1")
	defer os.Unsetenv(sessionHelperEnv)
	ctx := context.Background()
	start := time.Now()
	chunk, running, sid, err := RunSession(ctx, "call-1", "env-1", "", helperArgv(t), 100, 0)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("RunSession: %v", err)
	}
	if !running {
		t.Fatalf("100ms 预算下进程应仍在跑")
	}
	if sid == "" {
		t.Fatalf("应返回会话 id")
	}
	// 时序口径：elapsed 含子进程 spawn（Windows 起测试二进制 ~300ms，非
	// 本层可控）。预算窗口本身是 100ms；断言「窗口没有失控」：不低于窗口、
	// 不超过 spawn+窗口+轮询粒度的宽余。
	if elapsed < 100*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Fatalf("返回耗时 %v, 期望 spawn+100ms 窗口内", elapsed)
	}
	if !bytes.Contains(chunk, []byte("phase-1-alpha")) {
		t.Fatalf("首个增量应含 phase-1: %q", chunk)
	}
}

// ② 连续 Drain 互不重叠，拼接等于 Transcript 的出现序；③ 收尾上限内拿到全部。
func TestSessionDrainSequence(t *testing.T) {
	os.Setenv(sessionHelperEnv, "1")
	defer os.Unsetenv(sessionHelperEnv)
	ctx := context.Background()
	s, err := StartSession(ctx, "call-2", "env-2", "", helperArgv(t))
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	d1 := s.Collect(200)
	if !bytes.Contains(d1, []byte("phase-1-alpha")) {
		t.Fatalf("第一段应含 phase-1: %q", d1)
	}
	d2 := s.Collect(600)
	if bytes.Contains(d2, []byte("phase-1-alpha")) {
		t.Fatalf("第二段不得重复第一段: %q", d2)
	}
	if !bytes.Contains(d2, []byte("phase-2-beta")) {
		t.Fatalf("第二段应含 phase-2: %q", d2)
	}
	// 续取 stdin：写入一行，回显出现在第三段。
	d3, werr := s.WriteStdin([]byte("hello-session\n"), 800)
	if werr != nil {
		t.Fatalf("WriteStdin: %v", werr)
	}
	if !bytes.Contains(d3, []byte("got:hello-session")) {
		t.Fatalf("第三段应含回显: %q", d3)
	}
	// 进程退出后 100ms 收尾窗口内 Transcript 完整。
	_ = s.Wait()
	full := string(s.Transcript())
	for _, want := range []string{"phase-1-alpha", "phase-2-beta", "got:hello-session"} {
		if !strings.Contains(full, want) {
			t.Fatalf("Transcript 缺 %q: %q", want, full)
		}
	}
	// 三段 Drain 拼接 = Transcript 出现序（无省略时逐字节相等）。
	if s.transcript.OmittedBytes() == 0 {
		joined := string(d1) + string(d2) + string(d3)
		if joined != full {
			t.Fatalf("Drain 拼接应等于 Transcript:\n got %q\nwant %q", joined, full)
		}
	}
}

// ④ 写已退出的会话 → 明确错误；⑤ 超 8KB stdin → 审批哨兵。
func TestSessionWriteGuards(t *testing.T) {
	// quick helper 打一行即自然退出——退出路径不依赖杀树时序。
	t.Setenv(sessionHelperEnv, "quick")
	s, err := StartSession(context.Background(), "call-3", "env-3", "", helperArgv(t))
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	if _, err := s.WriteStdin(make([]byte, MaxStdinApprovalBytes+1), 50); !errors.Is(err, ErrStdinApproval) {
		t.Fatalf("超阈值应返回审批哨兵, got %v", err)
	}
	// 等自然退出（收尾窗口 PostExitCloseWait 内 read 侧标退出）。
	deadline := time.Now().Add(2 * time.Second)
	for !s.Exited() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !s.Exited() {
		t.Fatalf("quick helper 应自然退出")
	}
	if _, err := s.WriteStdin([]byte("x\n"), 50); err == nil {
		t.Fatalf("写已退会话应报错")
	}
}

// ⑥ 头尾缓冲：超限只有一个省略标记，字节数 = 实际丢弃量。
func TestHeadTailOverflow(t *testing.T) {
	h := NewHeadTail(100)
	// 200 字节：head 50 + tail 50，中间 100 被省略。
	payload := bytes.Repeat([]byte("x"), 200)
	h.Write(payload)
	if got := h.OmittedBytes(); got != 100 {
		t.Fatalf("省略计数 = %d, want 100", got)
	}
	out := h.Transcript()
	if n := bytes.Count(out, []byte("bytes omitted")); n != 1 {
		t.Fatalf("省略标记应恰好一个, got %d", n)
	}
	if !bytes.Contains(out, []byte("... 100 bytes omitted ...")) {
		t.Fatalf("标记字节数应等于实际丢弃量: %q", out)
	}
	wantLen := 50 + 1 + len(omittedMarker(100)) + 1 + 50
	if !bytes.Contains(out, []byte("xxxxx")) || len(out) != wantLen {
		t.Fatalf("头尾各保留 50: got %d, want %d", len(out), wantLen)
	}
	// 无省略时不带标记。
	h2 := NewHeadTail(100)
	h2.Write([]byte("small"))
	if bytes.Contains(h2.Transcript(), []byte("omitted")) {
		t.Fatalf("无省略不应有标记")
	}
}
