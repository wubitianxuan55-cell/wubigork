package docmd

// ocr_test.go — T6-9.3「OCR 兜底名实相符」单测：超时杀进程树、单图降级链路。
// 全部为纯单测，不依赖真实 OvisOCR2 服务 / tesseract / 扫描 PDF：
//   - 启动用当前测试二进制的 helper 模式充当「快速启动的假 llama-server」；
//   - 健康探针 / 命令构造 / 等待时长 / tesseract 查找与执行均为包级注入点。

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOvisServerHelperProcess 是 startOvisServer 超时/成功测试用的假 llama-server：
// 由测试以 os.Args[0] -test.run=^TestOvisServerHelperProcess$ 拉起；设置
// GAEA_OCR_TEST_HELPER=1 后长时间阻塞，等待被测代码用 proc.KillTracked 终止
// （或由测试清理兜底杀掉）。未设置该环境变量时本测试直接通过。
// 注意：必须用 time.Sleep 阻塞而非 `select {}`——空 select 会触发 Go
// 运行时 deadlock 检测（fatal error: all goroutines are asleep），导致假进程
// 自己崩溃退出，测试就测不到「杀进程」路径了。
func TestOvisServerHelperProcess(t *testing.T) {
	if os.Getenv("GAEA_OCR_TEST_HELPER") != "1" {
		return
	}
	time.Sleep(time.Hour) // 模拟长时间加载模型、迟迟不就绪的 llama-server
}

// fakeOvisServer 构造一个启动极快的假 llama-server（当前测试二进制的 helper 模式）。
func fakeOvisServer() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=^TestOvisServerHelperProcess$")
	cmd.Env = append(os.Environ(), "GAEA_OCR_TEST_HELPER=1")
	return cmd
}

// restoreOvisVars 记录并恢复 startOvisServer 的三个注入点（等待时长/健康探针/命令构造）。
func restoreOvisVars(t *testing.T) {
	t.Helper()
	origWait, origHealthy, origBuild := ovisStartWait, ovisHealthy, ovisBuildCmd
	t.Cleanup(func() {
		ovisStartWait, ovisHealthy, ovisBuildCmd = origWait, origHealthy, origBuild
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

// TestStartOvisServerTimeoutKillsProcessTree 注入「永不健康」探针 + 快速启动的假进程，
// 断言超时后 startOvisServer 返回 false 且进程已被 KillTracked 杀掉（无孤儿残留）。
func TestStartOvisServerTimeoutKillsProcessTree(t *testing.T) {
	restoreOvisVars(t)

	var started *exec.Cmd
	ovisBuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOvisServer()
		return started, nil, true
	}
	ovisHealthy = func(*http.Client, string) bool { return false } // 永不健康 → 必走超时
	ovisStartWait = 2 * time.Second

	// 兜底：断言失败时也杀掉假进程，避免留下孤儿测试二进制。
	t.Cleanup(func() {
		if started != nil && started.Process != nil && processAlive(started.Process.Pid) {
			_ = started.Process.Kill()
		}
	})

	if startOvisServer(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startOvisServer 超时后应返回 false")
	}
	if started == nil || started.Process == nil {
		t.Fatal("假进程未启动")
	}
	waitProcessDead(t, started.Process.Pid)
}

// TestStartOvisServerSuccessKeepsProcessAlive 注入「立即健康」探针，断言启动成功返回
// true 且假进程保留存活（成功路径不得杀进程）。
func TestStartOvisServerSuccessKeepsProcessAlive(t *testing.T) {
	restoreOvisVars(t)

	var started *exec.Cmd
	ovisBuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOvisServer()
		return started, nil, true
	}
	ovisHealthy = func(*http.Client, string) bool { return true } // 立即就绪
	ovisStartWait = 2 * time.Second

	t.Cleanup(func() {
		if started != nil && started.Process != nil && processAlive(started.Process.Pid) {
			_ = started.Process.Kill()
		}
	})

	if !startOvisServer(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startOvisServer 就绪后应返回 true")
	}
	if started == nil || started.Process == nil {
		t.Fatal("假进程未启动")
	}
	if !processAlive(started.Process.Pid) {
		t.Fatal("启动成功应保留进程存活")
	}
}

// restoreTesseractVars 记录并恢复 tesseract 查找/执行注入点。
func restoreTesseractVars(t *testing.T) {
	t.Helper()
	origLook, origRun := tesseractLookPath, tesseractImage
	t.Cleanup(func() {
		tesseractLookPath, tesseractImage = origLook, origRun
	})
}

// TestOCRImageTextFallsBackToTesseract GAEA_OCR_* 指向不存在路径（OvisOCR2 不可用，
// ovisServerBase()==""）时，OCRImageText 应降级 tesseract 分支并返回注入的识别结果。
func TestOCRImageTextFallsBackToTesseract(t *testing.T) {
	t.Setenv("GAEA_OCR_DIR", filepath.Join(t.TempDir(), "no-such-ocr"))
	t.Setenv("GAEA_OCR_PORT", "1") // 无人监听的端口，健康检查必失败

	img := filepath.Join(t.TempDir(), "img.png")
	if err := os.WriteFile(img, []byte("fake-image"), 0o644); err != nil {
		t.Fatal(err)
	}

	restoreTesseractVars(t)
	tesseractLookPath = func(string) (string, error) { return `C:\fake\tesseract.exe`, nil }
	tesseractImage = func(_, _ string) (string, error) { return "注入的识别文本", nil }

	text, err := OCRImageText(img)
	if err != nil {
		t.Fatalf("OCRImageText: %v", err)
	}
	if text != "注入的识别文本" {
		t.Fatalf("text = %q, want 注入的 tesseract 结果", text)
	}
}

// TestOCRImageTextBothUnavailable OvisOCR2 与 tesseract 都不可用时，OCRImageText 应
// 返回明确错误，且错误信息包含两者的安装提示。
func TestOCRImageTextBothUnavailable(t *testing.T) {
	t.Setenv("GAEA_OCR_DIR", filepath.Join(t.TempDir(), "no-such-ocr"))
	t.Setenv("GAEA_OCR_PORT", "1")

	restoreTesseractVars(t)
	tesseractLookPath = func(string) (string, error) { return "", errors.New("tesseract 未安装") }

	_, err := OCRImageText("whatever.png")
	if err == nil {
		t.Fatal("两个引擎都不可用时应返回错误")
	}
	for _, want := range []string{"OvisOCR2", "tesseract", "安装"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("错误信息 %q 缺少安装提示 %q", err.Error(), want)
		}
	}
}
