package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// comfyHTTPServer 一个可编程的 ComfyUI 假服务：按路径记录调用并返回脚本化响应。
type comfyHTTPServer struct {
	srv *httptest.Server

	mu        sync.Mutex
	promptID  string
	historyOK string // /history 响应体；空串表示按状态机（prompt 提交过即完成）
	viewBody  []byte

	interruptHits int
	queueHits     int
	uploadHits    int
	viewHits      int
	historyHits   int
}

func newComfyHTTPServer(t *testing.T) *comfyHTTPServer {
	t.Helper()
	c := &comfyHTTPServer{promptID: "pid-1", viewBody: []byte{0x89, 'P', 'N', 'G', 1, 2, 3}}
	c.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		defer c.mu.Unlock()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/prompt":
			c.queueHits++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "{\"prompt_id\":%q}", c.promptID)
		case r.Method == http.MethodPost && r.URL.Path == "/interrupt":
			c.interruptHits++
			w.WriteHeader(200)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/history/"):
			c.historyHits++
			if c.historyOK != "" {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, c.historyOK)
				return
			}
			// 状态机：prompt 提交过即视为完成
			body := "{}"
			if c.queueHits > 0 {
				body = fmt.Sprintf("{\"%s\":{\"outputs\":{\"9\":{\"images\":[{\"filename\":\"gaea_00001_.png\",\"subfolder\":\"\",\"type\":\"output\",\"format\":\"png\"}]}},\"status\":{\"status_str\":\"success\"}}}", c.promptID)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, body)
		case r.Method == http.MethodGet && r.URL.Path == "/view":
			c.viewHits++
			w.Header().Set("Content-Type", "image/png")
			w.Write(c.viewBody)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(c.srv.Close)
	return c
}

func (c *comfyHTTPServer) backend() *ComfyUIBackend {
	b := NewComfyUIBackend(c.srv.URL)
	b.httpClient = c.srv.Client()
	b.pollInterval = 10 * time.Millisecond
	return b
}

func pngDataURL(s string) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(s))
}

// ── 提交 /queuePrompt ────────────────────────────────────────────

func TestQueuePrompt_Success(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	id, err := b.queuePrompt(context.Background(), map[string]interface{}{"4": map[string]interface{}{}})
	if err != nil {
		t.Fatalf("queuePrompt: %v", err)
	}
	if id != "pid-1" {
		t.Fatalf("prompt_id = %q, want pid-1", id)
	}
}

func TestQueuePrompt_HTTPError_WithHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, "{\"error\":\"value_not_in_list: model not found\"}")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	_, err := b.queuePrompt(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("期望错误，实际成功")
	}
	if !strings.Contains(err.Error(), "ComfyUI HTTP 400") {
		t.Errorf("错误应包含 HTTP 状态: %v", err)
	}
	if !strings.Contains(err.Error(), "value_not_in_list") {
		t.Errorf("错误应包含 value_not_in_list 提示: %v", err)
	}
}

func TestQueuePrompt_ContextCanceled(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 已取消：提交应立即失败
	if _, err := b.queuePrompt(ctx, map[string]interface{}{}); err == nil {
		t.Fatal("取消后提交应失败")
	}
}

// ── 轮询 /checkHistory ───────────────────────────────────────────

func TestCheckHistory_NotDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	files, done, err := b.checkHistory(context.Background(), "pid-x")
	if err != nil {
		t.Fatalf("checkHistory: %v", err)
	}
	if done {
		t.Fatal("空 history 不应判定完成")
	}
	if len(files) != 0 {
		t.Fatalf("空 history 不应有输出文件: %v", files)
	}
}

func TestCheckHistory_Done_WithFiles(t *testing.T) {
	srv := newComfyHTTPServer(t)
	srv.mu.Lock()
	srv.historyOK = "{\"pid-1\":{\"outputs\":{\"9\":{\"images\":[{\"filename\":\"gaea_00001_.png\",\"subfolder\":\"\",\"type\":\"output\",\"format\":\"png\"}]}},\"status\":{\"status_str\":\"success\"}}}"
	srv.mu.Unlock()
	b := srv.backend()
	files, done, err := b.checkHistory(context.Background(), "pid-1")
	if err != nil {
		t.Fatalf("checkHistory: %v", err)
	}
	if !done {
		t.Fatal("应判定完成")
	}
	if len(files) != 1 {
		t.Fatalf("输出文件数 = %d, want 1", len(files))
	}
	if files[0].filename != "gaea_00001_.png" || files[0].kind != "image" {
		t.Fatalf("文件 = %+v", files[0])
	}
}

func TestCheckHistory_ExecutionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{\"pid-e\":{\"status\":{\"status_str\":\"error\",\"messages\":[[\"execution_error\",{\"exception_message\":\"OOM\"}]],\"completed\":true}}}")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	_, done, err := b.checkHistory(context.Background(), "pid-e")
	if err == nil {
		t.Fatal("执行错误应返回 error")
	}
	if !done {
		t.Fatal("执行错误应标记完成（终止轮询）")
	}
	if !strings.Contains(err.Error(), "OOM") {
		t.Errorf("错误应包含异常消息: %v", err)
	}
}

func TestCheckHistory_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	if _, _, err := b.checkHistory(context.Background(), "pid-x"); err == nil {
		t.Fatal("HTTP 500 应返回 error")
	}
}

// ── 取消 /interrupt ──────────────────────────────────────────────

func TestInterrupt_Success_AndIdempotent(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	for i := 0; i < 2; i++ {
		if err := b.Interrupt(context.Background()); err != nil {
			t.Fatalf("第 %d 次 Interrupt: %v", i+1, err)
		}
	}
	srv.mu.Lock()
	hits := srv.interruptHits
	srv.mu.Unlock()
	if hits != 2 {
		t.Fatalf("/interrupt 调用次数 = %d, want 2（幂等重复调用均触发）", hits)
	}
	if !b.isCancelled() {
		t.Fatal("Interrupt 后本地取消标记应置位")
	}
}

func TestInterrupt_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	err := b.Interrupt(context.Background())
	if err == nil {
		t.Fatal("HTTP 500 应返回 error")
	}
	// 标记仍应置位（取消意图已记录，不因网络错误丢失）
	if !b.isCancelled() {
		t.Fatal("Interrupt 失败也应置位本地取消标记")
	}
}

// ── 上传 /uploadImage ────────────────────────────────────────────

func TestUploadImage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/image" {
			w.WriteHeader(404)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{\"name\":\"ref.png\"}")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	name, err := b.uploadImage(context.Background(), pngDataURL("fake-png-bytes"))
	if err != nil {
		t.Fatalf("uploadImage: %v", err)
	}
	if name != "ref.png" {
		t.Fatalf("文件名 = %q, want ref.png", name)
	}
}

func TestUploadImage_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, "boom")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	if _, err := b.uploadImage(context.Background(), pngDataURL("x")); err == nil {
		t.Fatal("HTTP 500 应返回 error")
	}
}

func TestUploadImage_InvalidDataURL(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	if _, err := b.uploadImage(context.Background(), "not-a-data-url"); err == nil {
		t.Fatal("非法 data URL 应返回 error")
	}
}

// ── 下载 /downloadFile ───────────────────────────────────────────

func TestDownloadFile_Success(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	dataURL, err := b.downloadFile(context.Background(), comfyOutputFile{filename: "a.png", subfolder: "", outputType: "output", format: "png", kind: "image"})
	if err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("data URL 前缀异常: %.40s", dataURL)
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, "not found")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()

	_, err := b.downloadFile(context.Background(), comfyOutputFile{filename: "x.png", outputType: "output"})
	if err == nil {
		t.Fatal("HTTP 404 应返回 error（禁止把错误体当图片返回）")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("错误应包含状态码: %v", err)
	}
}

// ── 全链路 GenerateImage ─────────────────────────────────────────

func TestGenerateImage_FullChain(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{
		Model: "krea2", Prompt: "测试", Size: "512x512", Seed: 42,
	})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("输出数 = %d, want 1", len(resp.Data))
	}
	if !strings.HasPrefix(resp.Data[0].B64JSON, "data:image/png;base64,") {
		t.Fatalf("输出 data URL 异常: %.40s", resp.Data[0].B64JSON)
	}
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.queueHits != 1 || srv.historyHits == 0 || srv.viewHits != 1 {
		t.Fatalf("链路调用: queue=%d history=%d view=%d", srv.queueHits, srv.historyHits, srv.viewHits)
	}
}

func TestGenerateImage_UnknownModel(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1") // 不会真正发请求
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "grok-imagine", Prompt: "x"})
	if err == nil {
		t.Fatal("未知模型应返回错误（T6-4.2 禁止静默降级到 Krea2）")
	}
	if !strings.Contains(err.Error(), "不支持的模型") {
		t.Errorf("错误应为中文白名单提示: %v", err)
	}
}

func TestGenerateImage_InvalidSize(t *testing.T) {
	b := NewComfyUIBackend("http://127.0.0.1:1")
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "krea2", Prompt: "x", Size: "abc"})
	if err == nil {
		t.Fatal("非法尺寸应返回错误（T6-4.4）")
	}
	if !strings.Contains(err.Error(), "尺寸格式无效") {
		t.Errorf("错误应为中文尺寸提示: %v", err)
	}
}

func TestGenerateImage_FluxModel(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "flux", Prompt: "x", Seed: 1})
	if err != nil {
		t.Fatalf("flux 生成: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("输出数 = %d, want 1", len(resp.Data))
	}
}

// ── 取消标记（T6-4.1 排队项删除）───────────────────────────────

func TestGenerateImage_CancelledMarker_BlocksNewSubmission(t *testing.T) {
	srv := newComfyHTTPServer(t)
	b := srv.backend()

	// 取消（Interrupt 置位本地标记）后，新提交应被拒绝（等价删除排队项）
	if err := b.Interrupt(context.Background()); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	_, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "krea2", Prompt: "x"})
	if err == nil {
		t.Fatal("取消标记置位后 GenerateImage 应拒绝新提交")
	}
	if !strings.Contains(err.Error(), "生成已取消") {
		t.Errorf("错误应为取消提示: %v", err)
	}

	// 新一轮生成前 ResetCancel → 恢复正常
	b.ResetCancel()
	resp, err := b.GenerateImage(context.Background(), &ImageGenerationRequest{Model: "krea2", Prompt: "x", Seed: 1})
	if err != nil {
		t.Fatalf("ResetCancel 后应可正常生成: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("输出数 = %d, want 1", len(resp.Data))
	}
}

// ── 取消后轮询即刻退出（T6-4.1）────────────────────────────────

func TestWaitForResult_CancelExitsPromptly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 永远不完成：返回空 history
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
	}))
	defer srv.Close()
	b := NewComfyUIBackend(srv.URL)
	b.httpClient = srv.Client()
	b.pollInterval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, _, err := b.waitForResult(ctx, "pid-cancel", &ImageGenerationRequest{}, nil)
	if err != context.Canceled {
		t.Fatalf("取消后应返回 context.Canceled, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("取消后轮询未即刻退出: %v", elapsed)
	}
}

// TestPollComfyProgress 验证 WebSocket 实时进度：percent 计算 + 节点 id→class_type 映射。
func TestPollComfyProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/ws") {
			w.WriteHeader(404)
			return
		}
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]interface{}{
			"type": "progress_state",
			"data": map[string]interface{}{
				"prompt_id": "p1",
				"nodes": map[string]interface{}{
					"3": map[string]interface{}{"value": 6, "max": 8, "state": "running", "node_id": "3"},
				},
			},
		})
		// 保持连接直到客户端断开（cancel 后关闭）
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	b := NewComfyUIBackend(srv.URL)
	got := make(chan string, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.pollComfyProgress(ctx, "p1", map[string]string{"3": "KSampler"}, func(status string, elapsed int, percent int, node string) {
		select {
		case got <- fmt.Sprintf("%d|%s", percent, node):
		default:
		}
	})

	select {
	case v := <-got:
		if v != "75|KSampler" {
			t.Fatalf("progress = %q, want 75|KSampler", v)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("等待 ws 进度回调超时")
	}
}

func TestClampPercent(t *testing.T) {
	if got := clampPercent(0); got != 0 {
		t.Fatalf("clampPercent(0) = %d", got)
	}
	if got := clampPercent(0.5); got != 50 {
		t.Fatalf("clampPercent(0.5) = %d", got)
	}
	if got := clampPercent(1.2); got != 100 {
		t.Fatalf("clampPercent(1.2) = %d", got)
	}
	if got := clampPercent(-0.2); got != 0 {
		t.Fatalf("clampPercent(-0.2) = %d", got)
	}
}
