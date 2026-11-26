package docmd

// ocr_test.go — 本地 OCR（唯一引擎 Unlimited-OCR-2）的单测：服务拉起的进程纪律、
// 引擎不可用时的报错、以及路由类用例共用的替身引擎。全部为纯单测，不依赖真实
// 服务 / 扫描 PDF：
//   - 启动用当前测试二进制的 helper 模式充当「快速启动的假 llama-server」；
//   - 健康探针 / 命令构造 / 等待时长均为包级注入点；
//   - 引擎链路由类用例（ocr_seam_test.go / ocrcache_test.go）用注册的替身引擎
//     ocrTestKind，不碰真引擎。OvisOCR2 / tesseract / uocr2 python 后端已退役
//     （2026-11-26），相关用例随之删除。

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ocrTestKind 引擎链路由测试用的替身引擎 kind（ocr_seam_test.go / ocrcache_test.go 共用）。
const ocrTestKind = "fake-ocr"

// fakeOCRProvider 替身引擎：结果与错误由测试经 useFakeOCR 注入，并统计被调用次数。
type fakeOCRProvider struct{}

var (
	fakeOCRRuns int
	fakeOCRText = "替身引擎的识别结果"
	fakeOCRErr  error
)

func (fakeOCRProvider) Name() string { return ocrTestKind }

func (fakeOCRProvider) ExtractImage(string) (string, error) {
	fakeOCRRuns++
	return fakeOCRText, fakeOCRErr
}

func init() {
	RegisterOCRProvider(ocrTestKind, func() OCRProvider { return fakeOCRProvider{} })
}

// useFakeOCR 设定替身引擎的结果 / 错误，返回调用计数；测试结束后恢复默认值。
func useFakeOCR(t *testing.T, text string, err error) *int {
	t.Helper()
	origText, origErr := fakeOCRText, fakeOCRErr
	fakeOCRText, fakeOCRErr, fakeOCRRuns = text, err, 0
	t.Cleanup(func() { fakeOCRText, fakeOCRErr, fakeOCRRuns = origText, origErr, 0 })
	return &fakeOCRRuns
}

// writeFakeImage 写一张内容随意的「图片」，返回路径。
func writeFakeImage(t *testing.T) string {
	t.Helper()
	img := filepath.Join(t.TempDir(), "img.png")
	if err := os.WriteFile(img, []byte("fake-image"), 0o644); err != nil {
		t.Fatal(err)
	}
	return img
}

// TestOCRServerHelperProcess 是 startTrackedServer 超时/成功测试用的假 llama-server：
// 由测试以 os.Args[0] -test.run=^TestOCRServerHelperProcess$ 拉起；设置
// GAEA_OCR_TEST_HELPER=1 后长时间阻塞，等待被测代码用 proc.KillTracked 终止
// （或由测试清理兜底杀掉）。未设置该环境变量时本测试直接通过。
// 注意：必须用 time.Sleep 阻塞而非 `select {}`——空 select 会触发 Go
// 运行时 deadlock 检测（fatal error: all goroutines are asleep），导致假进程
// 自己崩溃退出，测试就测不到「杀进程」路径了。
func TestOCRServerHelperProcess(t *testing.T) {
	if os.Getenv("GAEA_OCR_TEST_HELPER") != "1" {
		return
	}
	time.Sleep(time.Hour) // 模拟长时间加载模型、迟迟不就绪的 llama-server
}

// fakeOCRServer 构造一个启动极快的假 llama-server（当前测试二进制的 helper 模式）。
func fakeOCRServer() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestOCRServerHelperProcess$")
	cmd.Env = append(os.Environ(), "GAEA_OCR_TEST_HELPER=1")
	return cmd
}

// restoreUocr2ServerVars 记录并恢复 startTrackedServer 的三个注入点
// （等待时长 / 健康探针 / 命令构造）。
func restoreUocr2ServerVars(t *testing.T) {
	t.Helper()
	origWait, origHealthy, origBuild := uocr2StartWait, uocr2Healthy, uocr2LlamaBuildCmd
	t.Cleanup(func() {
		uocr2StartWait, uocr2Healthy, uocr2LlamaBuildCmd = origWait, origHealthy, origBuild
	})
}

// waitProcessDead 轮询等待 pid 进程终止（最多 5s）；超时判定为孤儿残留。
func waitProcessDead(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("进程 %d 在超时后仍然存活（孤儿残留）", pid)
}

// killIfAlive 测试收尾时兜底杀掉假进程，避免断言失败后留下孤儿测试二进制。
// 传指针：假进程是在被测代码回调命令构造时才赋值的。
func killIfAlive(t *testing.T, cmdp **exec.Cmd) {
	t.Helper()
	t.Cleanup(func() {
		cmd := *cmdp
		if cmd != nil && cmd.Process != nil && processAlive(cmd.Process.Pid) {
			_ = cmd.Process.Kill()
		}
	})
}

// TestStartUocr2ServerTimeoutKillsProcessTree 注入「永不健康」探针 + 快速启动的假进程，
// 断言超时后 startUocr2LlamaServer 返回 false 且进程已被 KillTracked 杀掉（无孤儿残留）。
func TestStartUocr2ServerTimeoutKillsProcessTree(t *testing.T) {
	restoreUocr2ServerVars(t)

	var started *exec.Cmd
	uocr2LlamaBuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOCRServer()
		return started, nil, true
	}
	uocr2Healthy = func(*http.Client, string) bool { return false } // 永不健康 → 必走超时
	uocr2StartWait = 2 * time.Second
	killIfAlive(t, &started)

	if startUocr2LlamaServer(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startUocr2LlamaServer 超时后应返回 false")
	}
	if started == nil || started.Process == nil {
		t.Fatal("假进程未启动")
	}
	waitProcessDead(t, started.Process.Pid)
}

// TestStartUocr2ServerSuccessKeepsProcessAlive 注入「立即健康」探针，断言启动成功返回
// true 且假进程保留存活（成功路径不得杀进程）。
func TestStartUocr2ServerSuccessKeepsProcessAlive(t *testing.T) {
	restoreUocr2ServerVars(t)

	var started *exec.Cmd
	uocr2LlamaBuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOCRServer()
		return started, nil, true
	}
	uocr2Healthy = func(*http.Client, string) bool { return true } // 立即就绪
	uocr2StartWait = 2 * time.Second
	killIfAlive(t, &started)

	if !startUocr2LlamaServer(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startUocr2LlamaServer 就绪后应返回 true")
	}
	if started == nil || started.Process == nil {
		t.Fatal("假进程未启动")
	}
	if !processAlive(started.Process.Pid) {
		t.Fatal("启动成功应保留进程存活")
	}
}

// TestOCRImageText_EngineUnavailable 唯一的本地引擎不可用（工件缺失 + 端口无人监听）
// 时，OCRImageText 返回明确错误，错误信息带安装提示、指路 Unlimited-OCR-2，且不再
// 提已退役的引擎。
func TestOCRImageText_EngineUnavailable(t *testing.T) {
	t.Setenv("GAEA_UOCR2_LLAMA_DIR", filepath.Join(t.TempDir(), "no-such-llama"))
	t.Setenv("GAEA_UOCR2_LLAMA_URL", "http://127.0.0.1:1") // 无人监听的端口，健康检查必失败

	_, err := OCRImageText("whatever.png")
	if err == nil {
		t.Fatal("引擎不可用时应返回错误")
	}
	for _, want := range []string{"Unlimited-OCR-2", "安装"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("错误信息 %q 缺少 %q", err.Error(), want)
		}
	}
	for _, retired := range []string{"OvisOCR2", "tesseract"} {
		if strings.Contains(err.Error(), retired) {
			t.Fatalf("错误信息不应再提已退役的引擎 %q：%q", retired, err.Error())
		}
	}
}
