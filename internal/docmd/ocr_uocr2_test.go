package docmd

// ocr_uocr2_test.go — uocr2 引擎（Unlimited-OCR-2）单测：结构标记清洗、
// /ocr 客户端解码与截断 fail-closed、服务拉起超时杀进程树/成功保留存活。
// 全部为纯单测，不依赖真实 python 服务（拉起用 ocr_test.go 的 helper 假进程）。

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestUocr2CleanText(t *testing.T) {
	raw := "<PAGE><BLOCK>header[42, 16, 143, 27]<FORK>2026/9/20 21:35\n" +
		"<BLOCK>header[490, 16, 654, 29]<FORK>福建省环境监测行业指导价\n" +
		"<BLOCK>table[133, 36, 868, 874]<FORK><table><tr><td>序号</td></tr></table>\n" +
		"<BLOCK>footer[42, 971, 958, 982]<FORK>https://mp.weixin.qq.com/s?src=11 2/15"
	want := "2026/9/20 21:35\n福建省环境监测行业指导价\n<table><tr><td>序号</td></tr></table>\n" +
		"https://mp.weixin.qq.com/s?src=11 2/15"
	if got := uocr2CleanText(raw); got != want {
		t.Errorf("uocr2CleanText = %q, want %q", got, want)
	}
	if got := uocr2CleanText("  纯文本无标记  "); got != "纯文本无标记" {
		t.Errorf("无标记输入应原样去空白: %q", got)
	}
}

// TestUocr2CleanTextDropsMalformedBlock 坏标记行（<FORK> 缺失、坐标与正文粘连，
// GPU bf16 数字噪声实测）整行剥掉，好行保留——宁缺勿把坐标碎片漏进正文。
func TestUocr2CleanTextDropsMalformedBlock(t *testing.T) {
	raw := "<PAGE><BLOCK>header[42, 16, 143, 27]<FORK>好标题\n" +
		"<BLOCK>header[490, 16、坏行坐标粘连无FORK\n" +
		"<BLOCK>table[1, 2, 3, 4]<FORK><table><tr><td>正文</td></tr></table>"
	want := "好标题\n<table><tr><td>正文</td></tr></table>"
	if got := uocr2CleanText(raw); got != want {
		t.Errorf("uocr2CleanText = %q, want %q", got, want)
	}
}

// fakeUocr2Server 起 httptest 服务模拟 server.py：/health 探活 + /ocr 返回注入文本。
func fakeUocr2Server(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ocr", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// mustTempPNG 写一个内容任意的临时「PNG」（客户端只读字节转发，服务端才解码）。
func mustTempPNG(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "page.png")
	if err := os.WriteFile(p, []byte("fake-png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUocr2PageOCRDecodesAndCleans(t *testing.T) {
	raw := "<PAGE><BLOCK>table[1, 2, 3, 4]<FORK>水温\n<BLOCK>header[5, 6, 7, 8]<FORK>pH 值"
	srv := fakeUocr2Server(t, http.StatusOK, mustJSON(t, map[string]string{"text": raw}))
	got, err := uocr2PageOCR(srv.URL, mustTempPNG(t))
	if err != nil {
		t.Fatalf("uocr2PageOCR: %v", err)
	}
	if got != "水温\npH 值" {
		t.Errorf("text = %q, want 水温\\npH 值", got)
	}
}

func TestUocr2PageOCRTruncationFailsClosed(t *testing.T) {
	srv := fakeUocr2Server(t, http.StatusUnprocessableEntity,
		mustJSON(t, map[string]string{"error": "输出被 max_tokens 截断（finish_reason=length）"}))
	_, err := uocr2PageOCR(srv.URL, mustTempPNG(t))
	if err == nil {
		t.Fatal("422 截断应报错（fail-closed，不把残缺文本当完整结果）")
	}
	if !strings.Contains(err.Error(), "截断") {
		t.Errorf("错误应说明截断: %v", err)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestStartUocr2ServerTimeoutKillsProcessTree 镜像 ovis 同名测试：永不健康探针 +
// 快速假进程 → 超时返回 false 且进程被杀（无孤儿残留）。
func TestStartUocr2ServerTimeoutKillsProcessTree(t *testing.T) {
	restoreUocr2Vars(t)

	var started *exec.Cmd
	uocr2BuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOvisServer() // 同一 helper 假进程（长时间阻塞）
		return started, nil, true
	}
	uocr2Healthy = func(*http.Client, string) bool { return false }
	uocr2StartWait = 2 * time.Second

	t.Cleanup(func() {
		if started != nil && started.Process != nil && processAlive(started.Process.Pid) {
			_ = started.Process.Kill()
		}
	})

	if startUocr2Server(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startUocr2Server 超时后应返回 false")
	}
	if started == nil || started.Process == nil {
		t.Fatal("假进程未启动")
	}
	waitProcessDead(t, started.Process.Pid)
}

// TestStartUocr2ServerSuccessKeepsProcessAlive 立即健康探针 → 返回 true 且进程保留存活。
func TestStartUocr2ServerSuccessKeepsProcessAlive(t *testing.T) {
	restoreUocr2Vars(t)

	var started *exec.Cmd
	uocr2BuildCmd = func() (*exec.Cmd, *os.File, bool) {
		started = fakeOvisServer()
		return started, nil, true
	}
	uocr2Healthy = func(*http.Client, string) bool { return true }
	uocr2StartWait = 2 * time.Second

	t.Cleanup(func() {
		if started != nil && started.Process != nil && processAlive(started.Process.Pid) {
			_ = started.Process.Kill()
		}
	})

	if !startUocr2Server(&http.Client{Timeout: 2 * time.Second}, "http://127.0.0.1:1") {
		t.Fatal("startUocr2Server 就绪后应返回 true")
	}
	if !processAlive(started.Process.Pid) {
		t.Fatal("启动成功应保留进程存活")
	}
}

// TestUocr2ProviderDropsCacheOnTransportErr 传输层错误（服务中途消亡）必须清缓存
// base，让下一页重新探测/拉起；HTTP 状态错误（服务活着）不清。
func TestUocr2ProviderDropsCacheOnTransportErr(t *testing.T) {
	restoreUocr2Vars(t)

	// 1) 服务活着但返回 422：错误透传，缓存保留。
	srv := fakeUocr2Server(t, http.StatusUnprocessableEntity,
		mustJSON(t, map[string]string{"error": "截断"}))
	t.Setenv("GAEA_UOCR2_URL", srv.URL) // resolveBase 探测假服务 /health 即成功并缓存
	p := &uocr2OCRProvider{}
	if _, err := p.ExtractImage(mustTempPNG(t)); err == nil {
		t.Fatal("422 应报错")
	}
	if p.base != srv.URL {
		t.Fatalf("HTTP 状态错误不应清缓存（服务活着），base = %q", p.base)
	}

	// 2) 服务消亡（关掉监听）：传输层错误 → 缓存必须被清。
	srv.Close()
	if _, err := p.ExtractImage(mustTempPNG(t)); err == nil {
		t.Fatal("传输层失败应报错")
	}
	if p.base != "" {
		t.Fatalf("传输层错误后缓存应被清空重新探测, got %q", p.base)
	}
}

// restoreUocr2Vars 记录并恢复 startUocr2Server 的三个注入点。
func restoreUocr2Vars(t *testing.T) {
	t.Helper()
	origWait, origHealthy, origBuild := uocr2StartWait, uocr2Healthy, uocr2BuildCmd
	t.Cleanup(func() {
		uocr2StartWait, uocr2Healthy, uocr2BuildCmd = origWait, origHealthy, origBuild
	})
}

// fakeUocr2BatchServer 起假服务模拟 /ocr_batch：按注入的应答器返回。
func fakeUocr2BatchServer(t *testing.T, handle func() (int, string)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ocr_batch", func(w http.ResponseWriter, _ *http.Request) {
		status, body := handle()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestUocr2PagesOCRDetectsOrderAndCleans(t *testing.T) {
	raws := []string{
		"<PAGE><BLOCK>table[1, 2, 3, 4]<FORK>第一页表格",
		"<PAGE><BLOCK>header[5, 6, 7, 8]<FORK>第二页标题",
	}
	body, _ := json.Marshal(map[string]any{"texts": raws})
	srv := fakeUocr2BatchServer(t, func() (int, string) { return http.StatusOK, string(body) })
	got, err := uocr2PagesOCR(srv.URL, []string{mustTempPNG(t), mustTempPNG(t)})
	if err != nil {
		t.Fatalf("uocr2PagesOCR: %v", err)
	}
	if len(got) != 2 || got[0] != "第一页表格" || got[1] != "第二页标题" {
		t.Errorf("应与输入等长同序且已清洗: %+v", got)
	}
}

func TestUocr2PagesOCRCountMismatchFailsClosed(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"texts": []string{"只有一页"}})
	srv := fakeUocr2BatchServer(t, func() (int, string) { return http.StatusOK, string(body) })
	_, err := uocr2PagesOCR(srv.URL, []string{mustTempPNG(t), mustTempPNG(t)})
	if err == nil {
		t.Fatal("页数不齐应报错（fail-closed，编排层整窗回退逐页）")
	}
	if !strings.Contains(err.Error(), "页数不齐") {
		t.Errorf("错误应说明页数不齐: %v", err)
	}
}

func TestUocr2ProviderExtractPagesDropsCacheOnTransportErr(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"texts": []string{"<PAGE><BLOCK>x[1]<FORK>内容"}})
	srv := fakeUocr2BatchServer(t, func() (int, string) { return http.StatusOK, string(body) })
	t.Setenv("GAEA_UOCR2_URL", srv.URL)
	p := &uocr2OCRProvider{}
	got, err := p.ExtractPages([]string{mustTempPNG(t)})
	if err != nil || len(got) != 1 || got[0] != "内容" {
		t.Fatalf("ExtractPages 应成功: %v %+v", err, got)
	}
	srv.Close()
	if _, err := p.ExtractPages([]string{mustTempPNG(t)}); err == nil {
		t.Fatal("传输层失败应报错")
	}
	if p.base != "" {
		t.Fatalf("传输层错误后缓存应被清空, got %q", p.base)
	}
}

// TestOCRPageLoopBatchWinsAndFallsBack 批路径：成功窗整批命中；失败窗整窗回退
// 逐页；顺序/失败页语义与纯逐页一致，progress 按窗跳进。6 页 + 窗口 5 → 两窗。
func TestOCRPageLoopBatchWinsAndFallsBack(t *testing.T) {
	files := []string{"p1.png", "p2.png", "p3.png", "p4.png", "p5.png", "p6.png"}
	var gotProgress []int
	batchCalls := 0
	batchOCR := func(jobs []ocrPageJob) ([]string, error) {
		batchCalls++
		if jobs[0].num == 1 { // 第一窗（5 页）成功（文本须 ≥48 rune 过近空页守卫）
			long := func(tag string) string {
				return tag + "：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误"
			}
			return []string{long("批-1"), long("批-2"), long("批-3"), long("批-4"), long("批-5")}, nil
		}
		return nil, errors.New("批失败") // 第二窗（1 页）失败 → 回退逐页
	}
	texts, total, failed, err := ocrPageLoopBatch(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		return "逐页-" + strconv.Itoa(pageNum), nil
	}, batchOCR, func(done, totalN int) {
		gotProgress = append(gotProgress, done)
	})
	if err != nil {
		t.Fatalf("批失败窗回退逐页不应整体报错: %v", err)
	}
	if batchCalls != 2 {
		t.Fatalf("6 页应分 2 窗（窗口=%d）: %d", ocrBatchWindow, batchCalls)
	}
	want := []string{
		"批-1：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-2：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-3：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-4：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-5：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"逐页-6",
	}
	if len(texts) != 6 {
		t.Fatalf("应 6 页全出: %+v", texts)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Fatalf("texts[%d] = %q, want %q", i, texts[i], want[i])
		}
	}
	if total != 6 || len(failed) != 0 {
		t.Fatalf("total=6 无失败页: total=%d failed=%v", total, failed)
	}
	if len(gotProgress) != 3 || gotProgress[0] != 0 || gotProgress[1] != 5 || gotProgress[2] != 6 {
		t.Fatalf("progress 应含起步事件(0)+按窗跳进 5/6: %+v", gotProgress)
	}
}

// TestOCRPageLoopBatchNearEmptyPageFallsBack 批结果含近空页（生成退化：仅页眉
// 无正文）→ 整窗回退逐页，近空页不进结果。
func TestOCRPageLoopBatchNearEmptyPageFallsBack(t *testing.T) {
	files := []string{"p1.png", "p2.png"}
	batchOCR := func([]ocrPageJob) ([]string, error) {
		// 页数对得上但第二页近空（3 字符 < 20 阈值）→ 不接受
		return []string{"第一页完整正文内容足够长", "页眉"}, nil
	}
	texts, _, failed, err := ocrPageLoopBatch(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		return "逐页完整结果-" + strconv.Itoa(pageNum), nil
	}, batchOCR, nil)
	if err != nil {
		t.Fatalf("近空页回退逐页不应报错: %v", err)
	}
	if len(texts) != 2 || texts[0] != "逐页完整结果-1" || texts[1] != "逐页完整结果-2" {
		t.Fatalf("整窗应回退逐页: %+v", texts)
	}
	if len(failed) != 0 {
		t.Fatalf("不应有失败页: %v", failed)
	}
}

// TestOCRPageLoopBatchSkippedForSinglePage 单页不走批（gundam 逐页切片更优）。
func TestOCRPageLoopBatchSkippedForSinglePage(t *testing.T) {
	batchOCR := func([]ocrPageJob) ([]string, error) {
		t.Error("单页不应调用批识别")
		return nil, errors.New("不应到达")
	}
	texts, _, _, err := ocrPageLoopBatch([]string{"p1.png"}, 1, "",
		func(int, string) (string, error) { return "单页", nil }, batchOCR, nil)
	if err != nil || len(texts) != 1 || texts[0] != "单页" {
		t.Fatalf("单页应走逐页路径: %v %+v", err, texts)
	}
}
