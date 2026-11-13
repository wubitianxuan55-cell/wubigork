package comhelper

// client_test.go — 行协议测试：假 helper 进程（测试二进制重跑自身，进程模式
// 语义同 go test 标准配套）。覆盖：ready 握手、ping 往返、大载荷（bufio.Reader
// 无行长上限）、BUSY 结构化错误码、超时杀进程+自动重启、ctx 取消不杀进程。

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestFakeHelperProcess 只在 GAEA_OFFICE_FAKE_HELPER=1 的重跑进程里作为 helper
// 活着：首行 ready，随后逐行收 `{id,cmd,...}` 回标准回复；stdin EOF 即退出。
func TestFakeHelperProcess(t *testing.T) {
	if os.Getenv("GAEA_OFFICE_FAKE_HELPER") != "1" {
		t.Skip("仅作为 helper 进程模式运行")
	}
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	fmt.Fprintln(out, `{"event":"ready"}`)
	if err := out.Flush(); err != nil {
		return
	}
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			return
		}
		var req struct {
			ID      int64  `json:"id"`
			Cmd     string `json:"cmd"`
			Value   string `json:"value"`
			SleepMs int    `json:"sleep_ms"`
		}
		if jsonErr := json.Unmarshal([]byte(line), &req); jsonErr != nil {
			continue
		}
		switch req.Cmd {
		case "ping":
			fmt.Fprintf(out, `{"id":%d,"ok":true,"result":"pong"}`+"\n", req.ID)
		case "echo":
			payload, _ := json.Marshal(map[string]string{"value": req.Value})
			fmt.Fprintf(out, `{"id":%d,"ok":true,"result":%s}`+"\n", req.ID, payload)
		case "busy":
			fmt.Fprintf(out, `{"id":%d,"ok":false,"error":"app busy","code":"BUSY"}`+"\n", req.ID)
		case "hang":
			time.Sleep(time.Duration(max(req.SleepMs, 5000)) * time.Millisecond)
			fmt.Fprintf(out, `{"id":%d,"ok":true,"result":"late"}`+"\n", req.ID)
		default:
			fmt.Fprintf(out, `{"id":%d,"ok":false,"error":"unknown","code":"BAD_ARGS"}`+"\n", req.ID)
		}
		if flushErr := out.Flush(); flushErr != nil {
			return
		}
	}
}

// spawnFakeHelper 把测试二进制重跑成假 helper（绕过 EnsureHelperExe 真编译）。
func spawnFakeHelper(t *testing.T) func(exe string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	t.Helper()
	testExe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return func(string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
		cmd := exec.Command(testExe, "-test.run=TestFakeHelperProcess", "-test.v=false")
		cmd.Env = append(os.Environ(), "GAEA_OFFICE_FAKE_HELPER=1")
		stdin, err1 := cmd.StdinPipe()
		stdout, err2 := cmd.StdoutPipe()
		stderr, err3 := cmd.StderrPipe()
		if err := errors.Join(err1, err2, err3); err != nil {
			return nil, nil, nil, nil, err
		}
		// 不 Start：client.ensureLocked 统一启动（hook 只装配）。
		return cmd, stdin, stdout, stderr, nil
	}
}

func TestClientProtocolRoundTrip(t *testing.T) {
	c := NewClient()
	c.testHookSpawn = spawnFakeHelper(t)
	defer c.Dispose()
	ctx := context.Background()

	// ready 握手由首条 Call 隐式等待；ping 往返。
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	// 大载荷（1MB）：bufio.Reader 无行长上限。
	big := strings.Repeat("文档内容汉字ABC", 60_000)
	raw, err := c.Call(ctx, "echo", map[string]any{"value": big}, 30*time.Second)
	if err != nil {
		t.Fatalf("echo 大载荷: %v", err)
	}
	var got map[string]string
	if json.Unmarshal(raw, &got) != nil || got["value"] != big {
		t.Fatalf("大载荷回环不一致（len=%d）", len(got["value"]))
	}

	// BUSY 结构化错误码。
	if _, err = c.Call(ctx, "busy", nil, 30*time.Second); !IsBusy(err) {
		t.Fatalf("busy 应归类 BUSY: %v", err)
	}
	var ce *ComError
	if !errors.As(err, &ce) || ce.Code != "BUSY" {
		t.Fatalf("应为 *ComError{BUSY}: %T", err)
	}

	// 迟到的回复按未知 id 丢弃、不影响后续调用（busy 之后再 ping）。
	if err = c.Ping(ctx); err != nil {
		t.Fatalf("busy 后 ping: %v", err)
	}
}

func TestClientTimeoutKillsAndRespawns(t *testing.T) {
	c := NewClient()
	c.testHookSpawn = spawnFakeHelper(t)
	defer c.Dispose()
	ctx := context.Background()

	if _, err := c.Call(ctx, "hang", map[string]any{"sleep_ms": 30000}, 300*time.Millisecond); err == nil {
		t.Fatal("卡死命令应超时")
	}
	// 超时杀进程后下一次调用自动重启（fake helper 重新拉起、重新 ready）。
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("超时后应自动重启: %v", err)
	}
}

func TestClientCtxCancelKeepsProcess(t *testing.T) {
	c := NewClient()
	c.testHookSpawn = spawnFakeHelper(t)
	defer c.Dispose()
	ctx := context.Background()

	cctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := c.Call(cctx, "hang", map[string]any{"sleep_ms": 8000}, 30*time.Second)
		done <- err
	}()
	time.Sleep(200 * time.Millisecond) // 等 helper 接到 hang
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("取消应返回 context.Canceled: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("取消后调用未返回")
	}
	// 取消不杀进程：迟到的 "late" 回复被丢弃，进程仍在服务。
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("取消后进程应仍在: %v", err)
	}
}

func TestCompileCachePathAndHash(t *testing.T) {
	// SourceHash 稳定性：同源码同哈希（缓存键语义）。
	if len(SourceHash()) != 12 {
		t.Fatalf("SourceHash 长度应 12: %q", SourceHash())
	}
	a, b := SourceHash(), SourceHash()
	if a != b {
		t.Fatal("SourceHash 不稳定")
	}
	// 内嵌源码非空且含上游标记。
	if len(helperSource) < 100_000 || !strings.Contains(string(helperSource), "static void Main()") {
		t.Fatal("内嵌 helper 源码不完整")
	}
}
