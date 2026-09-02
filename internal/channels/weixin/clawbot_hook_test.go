package weixin

// clawbot_hook_test.go — 入站图片旁路钩子 OnInboundImage（v4.9 对话式改图）：
// 解密落盘成功即以 (发送者, 本地路径) 旁路触发；明文 URL 不触发；钩子 panic
// recover 不打断 OCR 识别主流程；识别链路行为与注入钩子前完全一致。

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// encryptedImageServer 起一个返回 AES-128-ECB(明文 PNG) 的测试源（真实走
// DownloadImageEncrypted 下载→解密→落盘链）。
func encryptedImageServer(t *testing.T, key []byte) *httptest.Server {
	t.Helper()
	ct, err := aes128ECBEncrypt(pngMagic, key)
	if err != nil {
		t.Fatalf("加密测试图: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(ct)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestRecognizeImage_InboundImageHookEncrypted 加密 CDN：解密落盘后钩子收到
// (fromUser, 本地路径)，且识别主流程照旧收到 file:// 路径。
func TestRecognizeImage_InboundImageHookEncrypted(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes.Repeat([]byte{0x2a}, 16)
	srv := encryptedImageServer(t, key)

	var hookFrom, hookPath string
	hookCalled := false
	hookFileErr := error(nil)
	// 钩子在 recognizeImage 内、cleanup defer 之前同步触发——文件存在性必须
	// 在钩子闭包内断言（返回后临时文件已清理，缓存 Set 的自持复制正是靠这一点）。
	s, _, _ := newHandleCaptureServer()
	s.OnInboundImage = func(fromUser, localPath string) {
		hookFrom, hookPath, hookCalled = fromUser, localPath, true
		st, err := os.Stat(localPath)
		if err != nil || st.Size() == 0 {
			hookFileErr = err
			return
		}
		data, rerr := os.ReadFile(localPath)
		if rerr != nil || !bytes.Equal(data, pngMagic) {
			hookFileErr = rerr
		}
	}
	var recogURL string
	s.MediaRecognizer = func(u string) (string, error) { recogURL = u; return "识别文本", nil }

	it := imageItem{Name: "截图.png", URL: srv.URL, AESKey: hex.EncodeToString(key)}
	desc, ok := s.recognizeImage(it, "u1")
	if !ok || desc != "识别文本" {
		t.Fatalf("识别主流程应照常工作: desc=%q ok=%v", desc, ok)
	}
	if !hookCalled {
		t.Fatal("加密图片解密落盘后应触发旁路钩子")
	}
	if hookFrom != "u1" {
		t.Errorf("钩子 fromUser = %q, want u1", hookFrom)
	}
	if hookFileErr != nil {
		t.Errorf("钩子收到的本地文件应存在且内容=解密 PNG: err=%v", hookFileErr)
	}
	want := "file://" + hookPath
	if recogURL != want {
		t.Errorf("识别器应收 file:// 路径 %q, 实际 %q", want, recogURL)
	}
}

// TestRecognizeImage_InboundImageHookPlaintextSkipped 明文 URL：无本地落盘
// 文件，钩子不触发（宁漏勿误——旁路只消费自己能自持的解密产物）。
func TestRecognizeImage_InboundImageHookPlaintextSkipped(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(pngMagic)
	}))
	t.Cleanup(plain.Close)

	called := false
	s, _, _ := newHandleCaptureServer()
	s.OnInboundImage = func(fromUser, localPath string) { called = true }
	s.MediaRecognizer = func(u string) (string, error) { return "识别文本", nil }

	if _, ok := s.recognizeImage(imageItem{URL: plain.URL}, "u1"); !ok {
		t.Fatal("明文图片识别应照常成功")
	}
	if called {
		t.Fatal("明文 URL 不应触发旁路钩子（无本地落盘文件）")
	}
}

// TestInvokeInboundImageHook_PanicRecovered 钩子 panic 必须 recover——旁路
// 消费者绝不能打断识别主流程（handle 全链不 panic）。
func TestInvokeInboundImageHook_PanicRecovered(t *testing.T) {
	s, _, _ := newHandleCaptureServer()
	s.OnInboundImage = func(fromUser, localPath string) { panic("缓存炸了") }
	s.MediaRecognizer = func(u string) (string, error) { return "识别文本", nil }

	// 直呼 invoker：不应向外传播 panic。
	s.invokeInboundImageHook("u1", "/tmp/x.png")

	// 走 recognizeImage 全链：识别照常成功。
	restore := allowLoopback(t)
	defer restore()
	key := bytes.Repeat([]byte{0x2a}, 16)
	srv := encryptedImageServer(t, key)
	desc, ok := s.recognizeImage(imageItem{URL: srv.URL, AESKey: hex.EncodeToString(key)}, "u1")
	if !ok || desc != "识别文本" {
		t.Fatalf("钩子 panic 后识别主流程应照常: desc=%q ok=%v", desc, ok)
	}
}

// TestInvokeInboundImageHook_NilNoOp 未注入钩子：零开销 no-op（与现状一致）。
func TestInvokeInboundImageHook_NilNoOp(t *testing.T) {
	s, _, _ := newHandleCaptureServer()
	s.invokeInboundImageHook("u1", "/tmp/x.png") // 不应 panic
}

// TestHandle_InboundImageHookWired 经 handle 全链验证钩子接线：图片批进来
// （type=3 image_item），钩子与识别同批触发，聊天管道收到的提示行不变。
func TestHandle_InboundImageHookWired(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes.Repeat([]byte{0x2a}, 16)
	srv := encryptedImageServer(t, key)

	hookCalled := false
	s, got, _ := newHandleCaptureServer()
	s.OnInboundImage = func(fromUser, localPath string) { hookCalled = true }
	s.MediaRecognizer = func(u string) (string, error) { return "截图上的文字", nil }

	s.handle(&inboundMsg{FromUserID: "u2", ItemList: []itemElem{
		{Type: 3, ImageItem: &imageItem{Name: "截图.png", URL: srv.URL, AESKey: hex.EncodeToString(key)}},
	}})
	if !hookCalled {
		t.Fatal("handle 全链应触发旁路钩子")
	}
	// 识别链路零改动：chatFn 收到的提示行保持既有格式。
	want := "（用户发来图片「截图.png」，识别内容：截图上的文字）"
	if !bytes.Contains([]byte(*got), []byte(want)) {
		t.Errorf("识别提示行应保持不变: %q", *got)
	}
}
