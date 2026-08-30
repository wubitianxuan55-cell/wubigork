package weixin

// v4.8.2 真机抓包基建测试：capture JSONL 写入/no-op、qr_status 抓取与媒体域
// 缓存、getUpdates inbound_media 整批抓包、SendFileCard 探针序列（全败降级 /
// 成功发图卡片 / multipart 直发送达 / 无域跳过 / 白名单外拒绝）。

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// tempCapturePath 注册临时抓包文件，测试结束清空注册（包级状态隔离）。
func tempCapturePath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wx_capture.jsonl")
	setCapturePath(p)
	t.Cleanup(func() { setCapturePath("") })
	return p
}

// resetMediaHosts 清空登录媒体域缓存（测试隔离），结束时再清一次。
func resetMediaHosts(t *testing.T) {
	t.Helper()
	setLatestMediaHosts("", "")
	t.Cleanup(func() { setLatestMediaHosts("", "") })
}

// readCaptureLines 读取抓包文件全部行（每行解析为 map，兼验 JSONL 合法性）。
func readCaptureLines(t *testing.T, p string) []map[string]interface{} {
	t.Helper()
	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("打开抓包文件失败: %v", err)
	}
	defer f.Close()
	var lines []map[string]interface{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 4*1024*1024) // inbound_media 原始批次可能较大
	for sc.Scan() {
		var m map[string]interface{}
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("抓包行非合法 JSON: %v (%q)", err, sc.Text())
		}
		lines = append(lines, m)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("读抓包文件失败: %v", err)
	}
	return lines
}

// TestCapture_JSONLWriteAndNoop capture 逐行追加合法 JSONL（ts/kind/data）；
// 路径未注册时 no-op 不追加不 panic（抓包绝不影响主流程的底线）。
func TestCapture_JSONLWriteAndNoop(t *testing.T) {
	p := tempCapturePath(t)
	capture("upload_probe", map[string]interface{}{"endpoint": "/x", "ok": false})
	capture("inbound_media", json.RawMessage(`{"msgs":[]}`))

	lines := readCaptureLines(t, p)
	if len(lines) != 2 {
		t.Fatalf("应有 2 行抓包, 实际 %d", len(lines))
	}
	if lines[0]["kind"] != "upload_probe" || lines[1]["kind"] != "inbound_media" {
		t.Fatalf("kind 顺序异常: %v / %v", lines[0]["kind"], lines[1]["kind"])
	}
	if _, ok := lines[0]["ts"].(float64); !ok {
		t.Fatalf("首行应有数值 ts: %v", lines[0]["ts"])
	}
	if data, ok := lines[1]["data"].(map[string]interface{}); !ok || data["msgs"] == nil {
		t.Fatalf("RawMessage data 应原样嵌入: %v", lines[1]["data"])
	}

	// 未注册路径：no-op
	setCapturePath("")
	capture("upload_probe", map[string]interface{}{"endpoint": "/y"})
	if lines = readCaptureLines(t, p); len(lines) != 2 {
		t.Fatalf("未注册路径时不应追加行, 实际 %d", len(lines))
	}
}

// TestQRStatus_CapturedAndHostsStored 扫码登录响应带 baseurl/redirect_host 时：
// 整响应原文抓包（kind=qr_status）+ 媒体域入包级缓存；PollQRStatusWithCode
// 同钩子；无媒体域的中间轮询态响应不抓（避免登录轮询刷屏）。
func TestQRStatus_CapturedAndHostsStored(t *testing.T) {
	p := tempCapturePath(t)
	resetMediaHosts(t)
	qrTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("qrcode") == "qr-2" {
			_, _ = w.Write([]byte(`{}`)) // 中间态：无媒体域
			return
		}
		_, _ = w.Write([]byte(`{"status":"confirmed","bot_token":"tok-1","baseurl":"https://media.weixin.qq.com","redirect_host":"rd.weixin.qq.com"}`))
	})

	resp, err := PollQRStatus("qr-1")
	if err != nil {
		t.Fatalf("PollQRStatus: %v", err)
	}
	if resp.BaseURL != "https://media.weixin.qq.com" || resp.RedirectHost != "rd.weixin.qq.com" {
		t.Fatalf("QRStatusResp 媒体域解析异常: %+v", resp)
	}
	if base, redirect := latestMediaHosts(); base != resp.BaseURL || redirect != resp.RedirectHost {
		t.Fatalf("媒体域缓存未更新: (%q,%q)", base, redirect)
	}

	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "qr_status" {
		t.Fatalf("应恰好 1 行 qr_status: %v", lines)
	}
	data, _ := lines[0]["data"].(map[string]interface{})
	if data["baseurl"] != "https://media.weixin.qq.com" || data["redirect_host"] != "rd.weixin.qq.com" {
		t.Fatalf("qr_status 应为整响应原文: %v", data)
	}

	// PollQRStatusWithCode 同钩子
	if _, err = PollQRStatusWithCode("qr-1", "vc"); err != nil {
		t.Fatalf("PollQRStatusWithCode: %v", err)
	}
	if lines = readCaptureLines(t, p); len(lines) != 2 {
		t.Fatalf("WithCode 也应抓一行, 实际 %d 行", len(lines))
	}

	// 无媒体域响应：不抓包、缓存不动
	if _, err = PollQRStatus("qr-2"); err != nil {
		t.Fatalf("PollQRStatus(qr-2): %v", err)
	}
	if lines = readCaptureLines(t, p); len(lines) != 2 {
		t.Fatalf("无媒体域响应不应抓包, 实际 %d 行", len(lines))
	}
	if base, _ := latestMediaHosts(); base != "https://media.weixin.qq.com" {
		t.Fatalf("无媒体域响应不应改写缓存: %q", base)
	}
}

// TestGetUpdates_CapturesInboundMediaBatch 批次含 image_item（或未知 type）
// 时抓整批原始 JSON（kind=inbound_media，保留服务端第一手多态形态）；
// 纯文本批次不抓。
func TestGetUpdates_CapturesInboundMediaBatch(t *testing.T) {
	p := tempCapturePath(t)

	mediaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0,"errcode":0,"msgs":[{"from_user_id":"u1","item_list":[{"type":3,"image_item":{"name":"真机.png","url":"https://cdn/a.png","file_id":12345}}]}],"get_updates_buf":"buf1"}`))
	}))
	defer mediaSrv.Close()
	s := New(Config{ILinkURL: mediaSrv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, nil)

	resp, err := s.getUpdates(&pollReq{}, time.Second)
	if err != nil {
		t.Fatalf("getUpdates: %v", err)
	}
	if len(resp.Msgs) != 1 || resp.Msgs[0].ItemList[0].ImageItem == nil {
		t.Fatalf("解析结果异常: %+v", resp)
	}

	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "inbound_media" {
		t.Fatalf("应恰好 1 行 inbound_media: %v", lines)
	}
	raw, _ := json.Marshal(lines[0]["data"])
	if !strings.Contains(string(raw), `"file_id":12345`) {
		t.Fatalf("应保留原始多态形态（file_id 数字）: %s", raw)
	}

	// 纯文本批次：不抓
	textSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ret":0,"errcode":0,"msgs":[{"from_user_id":"u1","item_list":[{"type":1,"text_item":{"text":"hi"}}]}]}`))
	}))
	defer textSrv.Close()
	s2 := New(Config{ILinkURL: textSrv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, nil)
	if _, err = s2.getUpdates(&pollReq{}, time.Second); err != nil {
		t.Fatalf("getUpdates(文本): %v", err)
	}
	if lines = readCaptureLines(t, p); len(lines) != 1 {
		t.Fatalf("纯文本批次不应抓包, 实际 %d 行", len(lines))
	}
}

// newProbeCardServer 构造探针测试通用 httptest：multipart 探针端点与 JSON
// sendmessage（文本降级/图片卡片）按 Content-Type 区分，行为由各用例注入。
func newProbeCardServer(t *testing.T, onProbes func(path string) (status int, body string), onText func(text string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		switch {
		case r.URL.Path == "/ilink/bot/sendmessage" && strings.HasPrefix(ct, "multipart/"):
			status, body := onProbes(r.URL.Path)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		case r.URL.Path == "/ilink/bot/sendmessage":
			var payload struct {
				Msg struct {
					ItemList []struct {
						Type     int `json:"type"`
						TextItem struct {
							Text string `json:"text"`
						} `json:"text_item"`
					} `json:"item_list"`
				} `json:"msg"`
			}
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &payload)
			for _, it := range payload.Msg.ItemList {
				if it.Type == 1 && onText != nil {
					onText(it.TextItem.Text)
				}
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"errcode":0}`))
		default:
			status, body := onProbes(r.URL.Path)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}
	}))
}

// TestSendFileCard_ProbeAllFailFallsBackToText 有媒体域但推断端点全部失败：
// 逐端点抓 upload_probe（含 404 的 err 记录），最终文本降级内容不变。
func TestSendFileCard_ProbeAllFailFallsBackToText(t *testing.T) {
	p := tempCapturePath(t)
	resetMediaHosts(t)

	var mu sync.Mutex
	var lastText string
	srv := newProbeCardServer(t,
		func(path string) (int, string) {
			if path == "/ilink/bot/upload" {
				return http.StatusNotFound, "not found"
			}
			return 200, `{"errcode":-1,"errmsg":"unsupported"}`
		},
		func(text string) { mu.Lock(); lastText = text; mu.Unlock() },
	)
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "书房绘梦.png")
	if err := os.WriteFile(img, []byte("png-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.SetMediaHosts(srv.URL, "")
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画一张猫"}}}})

	if err := s.SendFileCard(img, "一只橘猫"); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	sent := lastText
	mu.Unlock()
	if !strings.Contains(sent, "🖼 产物已生成：书房绘梦.png") || !strings.Contains(sent, "一只橘猫") {
		t.Fatalf("探针全败应文本降级: %q", sent)
	}

	var probes []string
	var sawNotFound bool
	for _, l := range readCaptureLines(t, p) {
		if l["kind"] != "upload_probe" {
			continue
		}
		data, _ := l["data"].(map[string]interface{})
		ep, _ := data["endpoint"].(string)
		probes = append(probes, ep)
		if data["ok"] != false {
			t.Fatalf("探针应记 ok=false: %v", data)
		}
		if data["status"] == float64(404) && data["err"] == "HTTP 404" {
			sawNotFound = true
		}
	}
	if want := []string{"/ilink/bot/uploadfile", "/ilink/bot/sendmessage", "/ilink/bot/upload"}; !reflect.DeepEqual(probes, want) {
		t.Fatalf("探针序列 = %v, want %v", probes, want)
	}
	if !sawNotFound {
		t.Fatal("404 端点应记 status+err")
	}
}

// TestSendFileCard_ProbeSuccessSendsImageCard uploadfile 探针返回 errcode=0 +
// url/file_id：组 image_item sendmessage（字段名参照入站形态 + caption），
// 不发文本降级；multipart 携带 file 内容与 base_info，鉴权头同 apiPost。
func TestSendFileCard_ProbeSuccessSendsImageCard(t *testing.T) {
	p := tempCapturePath(t)
	resetMediaHosts(t)

	var mu sync.Mutex
	var lastMsg map[string]interface{}
	var fileBytes []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ilink/bot/uploadfile":
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
				t.Errorf("uploadfile 应为 multipart: %q", r.Header.Get("Content-Type"))
			}
			if ah := r.Header.Get("Authorization"); ah != "Bearer tok" {
				t.Errorf("鉴权头应同 apiPost 形态: %q", ah)
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("解析 multipart 失败: %v", err)
			} else {
				if r.FormValue("base_info") == "" {
					t.Error("uploadfile 应带 base_info 字段")
				}
				f, _, err := r.FormFile("file")
				if err != nil {
					t.Errorf("multipart 缺 file 字段: %v", err)
				} else {
					got, _ := io.ReadAll(f)
					_ = f.Close()
					mu.Lock()
					fileBytes = got
					mu.Unlock()
				}
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"errcode":0,"url":"https://cdn/x.png","file_id":"f-1"}`))
		case "/ilink/bot/sendmessage":
			var payload struct {
				Msg map[string]interface{} `json:"msg"`
			}
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &payload)
			mu.Lock()
			lastMsg = payload.Msg
			mu.Unlock()
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"errcode":0}`))
		default:
			t.Errorf("不应请求其他端点: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "cat.png")
	if err := os.WriteFile(img, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.SetMediaHosts(srv.URL, "")
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画猫"}}}})

	if err := s.SendFileCard(img, "一只橘猫"); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	msg, got := lastMsg, fileBytes
	mu.Unlock()
	if string(got) != "PNGDATA" {
		t.Fatalf("上传文件内容不符: %q", got)
	}
	if msg["to_user_id"] != "u1" || msg["context_token"] != "ctx" {
		t.Fatalf("图片卡片路由不符: %v", msg)
	}
	items, _ := msg["item_list"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("图片卡片应含 image+caption 两 item: %v", msg)
	}
	it0, _ := items[0].(map[string]interface{})
	ii, _ := it0["image_item"].(map[string]interface{})
	if it0["type"] != float64(3) || ii["url"] != "https://cdn/x.png" || ii["file_id"] != "f-1" || ii["name"] != "cat.png" {
		t.Fatalf("image_item 字段应参照入站形态: %v / %v", it0, ii)
	}
	it1, _ := items[1].(map[string]interface{})
	if it1["type"] != float64(1) {
		t.Fatalf("次 item 应为 caption 文本: %v", it1)
	}

	// 抓包：仅 uploadfile 一行探针记录（成功，不记 send_image_card）
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰好 1 行探针记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["ok"] != true || data["endpoint"] != "/ilink/bot/uploadfile" {
		t.Fatalf("探针记录异常: %v", data)
	}
}

// TestSendFileCard_SendMessageMultipartDelivered 推断端点 b（sendmessage
// multipart 直发）返回 errcode=0：视为图片（含 caption）已送达，不再补发
// 图片卡片、不再文本降级。
func TestSendFileCard_SendMessageMultipartDelivered(t *testing.T) {
	p := tempCapturePath(t)
	resetMediaHosts(t)

	var mu sync.Mutex
	var mpText, mpTo string
	var jsonHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ilink/bot/sendmessage" && strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("ParseMultipartForm: %v", err)
			}
			mu.Lock()
			mpText, mpTo = r.FormValue("text"), r.FormValue("to_user_id")
			mu.Unlock()
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"errcode":0}`))
			return
		}
		if r.URL.Path == "/ilink/bot/sendmessage" {
			mu.Lock()
			jsonHits++
			mu.Unlock()
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"errcode":0}`))
			return
		}
		// 其余探针端点：不支持
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"errcode":-1}`))
	}))
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "图.webp")
	if err := os.WriteFile(img, []byte("WEBP"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.sendFn = func(string, string, string) error { return nil } // 吞掉 handle 自动回复，只统计探针流量
	s.SetMediaHosts(srv.URL, "")
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画"}}}})

	if err := s.SendFileCard(img, "caption-x"); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if mpText != "caption-x" || mpTo != "u1" {
		t.Fatalf("multipart 直发应带 caption 与路由: text=%q to=%q", mpText, mpTo)
	}
	if jsonHits != 0 {
		t.Fatalf("直发成功不应再发图片卡片/文本降级, JSON sendmessage %d 次", jsonHits)
	}
	lines := readCaptureLines(t, p)
	// uploadfile 先试（不支持，errcode=-1），sendmessage multipart 直发成功送达
	if len(lines) != 2 {
		t.Fatalf("应恰 2 行探针记录: %v", lines)
	}
	d1, _ := lines[0]["data"].(map[string]interface{})
	d2, _ := lines[1]["data"].(map[string]interface{})
	if d1["endpoint"] != "/ilink/bot/uploadfile" || d1["ok"] != false {
		t.Fatalf("首行应为 uploadfile 失败记录: %v", lines[0])
	}
	if d2["endpoint"] != "/ilink/bot/sendmessage" || d2["ok"] != true {
		t.Fatalf("次行应为 sendmessage 直发成功记录: %v", lines[1])
	}
}

// TestSendFileCard_NoHostSkipsProbe 无媒体域线索（未登录）：不发起任何探针
// 请求，抓一行 skipped=no_media_host 后走现有文本降级（逐字节不变）。
func TestSendFileCard_NoHostSkipsProbe(t *testing.T) {
	p := tempCapturePath(t)
	resetMediaHosts(t)

	var mu sync.Mutex
	var lastText string
	srv := newProbeCardServer(t,
		func(string) (int, string) {
			t.Error("无媒体域不应发起探针请求")
			return 500, ""
		},
		func(text string) { mu.Lock(); lastText = text; mu.Unlock() },
	)
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	// 不注入媒体域，包级缓存已清空
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画"}}}})
	if err := s.SendFileCard(img, ""); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	if !strings.Contains(lastText, "🖼 产物已生成：a.png") {
		t.Fatalf("无媒体域应只发文本降级: %q", lastText)
	}
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰 1 行 skipped 记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["skipped"] != "no_media_host" {
		t.Fatalf("skipped 原因应为 no_media_host: %v", lines[0])
	}
}

// TestSendFileCard_FileRejectedFallsBack 扩展名白名单外（图片产物场景限定
// png/jpeg/webp/gif）：不发起探针，抓 skipped=file_rejected 后文本降级。
func TestSendFileCard_FileRejectedFallsBack(t *testing.T) {
	p := tempCapturePath(t)
	resetMediaHosts(t)

	srv := newProbeCardServer(t,
		func(string) (int, string) {
			t.Error("白名单外文件不应发起探针")
			return 500, ""
		},
		func(string) {},
	)
	defer srv.Close()

	txt := filepath.Join(t.TempDir(), "笔记.txt")
	if err := os.WriteFile(txt, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.SetMediaHosts(srv.URL, "")
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "写"}}}})
	if err := s.SendFileCard(txt, ""); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰 1 行记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["skipped"] != "file_rejected" {
		t.Fatalf("应记 skipped=file_rejected: %v", lines[0])
	}
}
