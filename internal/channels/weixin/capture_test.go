package weixin

// v4.8.3 真机抓包基建测试：capture JSONL 写入/no-op、qr_status 抓取、
// getUpdates inbound_media 整批抓包、SendFileCard 真协议全链路
// （getuploadurl → CDN 密文上传 → image_item 卡片 → caption 补发 / 各失败
// 节点降级文本）。上传协议断言对照 hermes-agent 生产实现逐字段锁定。

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// TestQRStatus_Captured 扫码登录响应带 baseurl/redirect_host 时整响应原文
// 抓包（kind=qr_status）；PollQRStatusWithCode 同钩子；无媒体域的中间轮询态
// 响应不抓（避免登录轮询刷屏）。v4.8.3 起上传不再依赖这两个字段，抓包仅作
// 协议证据留存。
func TestQRStatus_Captured(t *testing.T) {
	p := tempCapturePath(t)
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

	// 无媒体域响应：不抓包
	if _, err = PollQRStatus("qr-2"); err != nil {
		t.Fatalf("PollQRStatus(qr-2): %v", err)
	}
	if lines = readCaptureLines(t, p); len(lines) != 2 {
		t.Fatalf("无媒体域响应不应抓包, 实际 %d 行", len(lines))
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

// newCardServer SendFileCard 测试通用 httptest：JSON API 端点按路径分派，
// 行为由各用例注入（返回状态码与响应体）。
func newCardServer(t *testing.T, onJSON func(path string, r *http.Request, body []byte) (int, string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		status, respBody := onJSON(r.URL.Path, r, data)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
}

// useTestCDN 重定向包级 CDN 基地址到测试服务器，测试结束恢复。
func useTestCDN(t *testing.T, url string) {
	t.Helper()
	old := wxCDNBaseURL
	wxCDNBaseURL = url
	t.Cleanup(func() { wxCDNBaseURL = old })
}

// TestSendFileCard_UploadFlowEndToEnd 真协议全链路：getuploadurl 请求字段
// 逐项锁定（filekey/media_type/to_user_id/rawsize/rawfilemd5/filesize=
// PKCS7 对齐/no_need_thumb/aeskey）→ CDN 密文上传（octet-stream，密文可用
// 同一 aeskey 解回明文）→ image_item 卡片（type=2、media.encrypt_query_param
// =x-encrypted-param、aes_key=base64(hex)、mid_size）→ caption 独立补发；
// 成功路径不文本降级，抓一行 delivered。
func TestSendFileCard_UploadFlowEndToEnd(t *testing.T) {
	p := tempCapturePath(t)
	plaintext := []byte("PNGDATA-真实图片字节")

	var mu sync.Mutex
	var uploadedCipher []byte

	// fake CDN：/upload 校验密文可解回明文，回 x-encrypted-param
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/c2c/upload" {
			t.Errorf("CDN 应只收 /c2c/upload: %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/octet-stream" {
			t.Errorf("CDN 上传应为 octet-stream: %q", ct)
		}
		if r.URL.Query().Get("filekey") == "" {
			t.Error("upload_param 形态应附 filekey")
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 || len(body)%16 != 0 {
			t.Errorf("密文长度应按块对齐: %d", len(body))
		}
		// 密文可解性在 getuploadurl 记录 aeskey 后于用例末尾统一断言
		mu.Lock()
		uploadedCipher = body
		mu.Unlock()
		w.Header().Set("x-encrypted-param", "TICKET-1")
		w.WriteHeader(200)
	}))
	defer cdn.Close()
	useTestCDN(t, cdn.URL+"/c2c")

	var guReq map[string]interface{}
	var aesKeyHex string
	var lastMsg map[string]interface{}
	var captionTexts []string
	srv := newCardServer(t, func(path string, r *http.Request, body []byte) (int, string) {
		if ah := r.Header.Get("Authorization"); ah != "Bearer tok" {
			t.Errorf("鉴权头应同 apiPost: %q", ah)
		}
		switch path {
		case "/ilink/bot/getuploadurl":
			_ = json.Unmarshal(body, &guReq)
			mu.Lock()
			aesKeyHex, _ = guReq["aeskey"].(string)
			mu.Unlock()
			return 200, `{"upload_param":"UP-PARAM-1","upload_id":"up-1"}`
		case "/ilink/bot/sendmessage":
			var payload struct {
				Msg map[string]interface{} `json:"msg"`
			}
			_ = json.Unmarshal(body, &payload)
			mu.Lock()
			isImage := false
			if items, ok := payload.Msg["item_list"].([]interface{}); ok && len(items) > 0 {
				if it, ok := items[0].(map[string]interface{}); ok && it["type"] == float64(2) {
					isImage = true
					lastMsg = payload.Msg // 仅记录图片消息（caption 消息另行收集）
				}
			}
			if !isImage { // 文本消息（caption 补发）
				if items, ok := payload.Msg["item_list"].([]interface{}); ok {
					for _, x := range items {
						if it, ok := x.(map[string]interface{}); ok {
							if ti, ok := it["text_item"].(map[string]interface{}); ok {
								captionTexts = append(captionTexts, ti["text"].(string))
							}
						}
					}
				}
			}
			mu.Unlock()
			return 200, `{"errcode":0}`
		default:
			t.Errorf("不应请求其他端点: %s", path)
			return 404, ""
		}
	})
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "书房绘梦.png")
	if err := os.WriteFile(img, plaintext, 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.sendFn = func(string, string, string) error { return nil } // 吞 handle 自动回复
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画猫"}}}})

	if err := s.SendFileCard(img, "一只橘猫"); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}

	// ① getuploadurl 请求字段（hermes 同款）
	mu.Lock()
	defer mu.Unlock()
	if guReq == nil {
		t.Fatal("应请求 getuploadurl")
	}
	if guReq["media_type"] != float64(1) || guReq["to_user_id"] != "u1" {
		t.Fatalf("media_type/to_user_id 不符: %v", guReq)
	}
	if guReq["no_need_thumb"] != true {
		t.Fatalf("no_need_thumb 应为 true: %v", guReq)
	}
	if guReq["rawsize"] != float64(len(plaintext)) {
		t.Fatalf("rawsize 不符: %v", guReq)
	}
	if guReq["rawfilemd5"] != md5Hex(plaintext) {
		t.Fatalf("rawfilemd5 应为明文 MD5: %v", guReq)
	}
	if _, ok := guReq["base_info"].(map[string]interface{}); !ok {
		t.Fatalf("应带 base_info: %v", guReq)
	}
	fk, _ := guReq["filekey"].(string)
	if len(fk) != 32 {
		t.Fatalf("filekey 应为 32 位 hex: %q", fk)
	}
	key, err := parseAESKeyHex(aesKeyHex)
	if err != nil {
		t.Fatalf("aeskey 应为合法 32 位 hex: %q", aesKeyHex)
	}

	// ② CDN 密文可用同一 aeskey 解回明文（AES-128-ECB + PKCS7）
	plain, err := aes128ECBDecrypt(uploadedCipher, key)
	if err != nil || string(plain) != string(plaintext) {
		t.Fatalf("CDN 密文应可解回明文: err=%v got=%q", err, plain)
	}
	if guReq["filesize"] != float64(len(uploadedCipher)) {
		t.Fatalf("filesize 应为密文（PKCS7 对齐后）大小: %v vs %d", guReq["filesize"], len(uploadedCipher))
	}

	// ③ image_item 卡片（真机 type=2 形态）
	if lastMsg == nil {
		t.Fatal("应发送 sendmessage")
	}
	if lastMsg["to_user_id"] != "u1" || lastMsg["context_token"] != "ctx" {
		t.Fatalf("图片卡片路由不符: %v", lastMsg)
	}
	items, _ := lastMsg["item_list"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("图片消息应只含 1 个 item（caption 走独立消息）: %v", lastMsg)
	}
	it0, _ := items[0].(map[string]interface{})
	ii, _ := it0["image_item"].(map[string]interface{})
	if it0["type"] != float64(2) {
		t.Fatalf("type 应为 2（真机形态）: %v", it0)
	}
	media, _ := ii["media"].(map[string]interface{})
	if media["encrypt_query_param"] != "TICKET-1" {
		t.Fatalf("encrypt_query_param 应为 CDN 回传票据: %v", media)
	}
	if media["aes_key"] != aesKeyForAPI(key) {
		t.Fatalf("aes_key 应为 base64(hex字符串): %v", media)
	}
	if media["encrypt_type"] != float64(1) {
		t.Fatalf("encrypt_type 应为 1: %v", media)
	}
	if ii["mid_size"] != float64(len(uploadedCipher)) {
		t.Fatalf("mid_size 应为密文大小: %v", ii["mid_size"])
	}

	// ④ caption 独立补发（图先文后）
	if len(captionTexts) != 1 || captionTexts[0] != "一只橘猫" {
		t.Fatalf("caption 应作为独立文本消息补发: %v", captionTexts)
	}

	// ⑤ 抓包：一行 delivered
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰 1 行 delivered 记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["stage"] != "delivered" {
		t.Fatalf("应记 stage=delivered: %v", data)
	}
}

// TestSendFileCard_UploadFullURLPreferred getuploadurl 返回 upload_full_url
// 时直连该地址上传（hermes 实测：老 PUT 会 404，统一 POST）。
func TestSendFileCard_UploadFullURLPreferred(t *testing.T) {
	p := tempCapturePath(t)

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/direct/upload" {
			t.Errorf("应直连 upload_full_url: %s", r.URL.Path)
			w.WriteHeader(404)
			return
		}
		w.Header().Set("x-encrypted-param", "TICKET-2")
		w.WriteHeader(200)
	}))
	defer cdn.Close()

	srv := newCardServer(t, func(path string, r *http.Request, body []byte) (int, string) {
		switch path {
		case "/ilink/bot/getuploadurl":
			return 200, `{"upload_full_url":"` + cdn.URL + `/direct/upload"}`
		case "/ilink/bot/sendmessage":
			return 200, `{"errcode":0}`
		default:
			t.Errorf("不应请求其他端点: %s", path)
			return 404, ""
		}
	})
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.sendFn = func(string, string, string) error { return nil }
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画"}}}})
	if err := s.SendFileCard(img, ""); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
}

// TestSendFileCard_UploadFailFallsBackToText getuploadurl 失败：抓一行
// stage=upload 错误记录后走文本降级（逐字节不变）。
func TestSendFileCard_UploadFailFallsBackToText(t *testing.T) {
	p := tempCapturePath(t)

	var mu sync.Mutex
	var lastText string
	srv := newCardServer(t, func(path string, r *http.Request, body []byte) (int, string) {
		switch path {
		case "/ilink/bot/getuploadurl":
			return 500, "boom"
		case "/ilink/bot/sendmessage":
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
			_ = json.Unmarshal(body, &payload)
			for _, it := range payload.Msg.ItemList {
				if it.Type == 1 {
					mu.Lock()
					lastText = it.TextItem.Text
					mu.Unlock()
				}
			}
			return 200, `{"errcode":0}`
		default:
			t.Errorf("getuploadurl 失败后不应请求其他端点: %s", path)
			return 404, ""
		}
	})
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.sendFn = func(string, string, string) error { return nil }
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画"}}}})
	if err := s.SendFileCard(img, "cap"); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	sent := lastText
	mu.Unlock()
	if !strings.Contains(sent, "🖼 产物已生成：a.png") || !strings.Contains(sent, "cap") {
		t.Fatalf("上传失败应文本降级: %q", sent)
	}
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰 1 行 upload 错误记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["stage"] != "upload" || data["err"] == "" {
		t.Fatalf("应记 stage=upload + err: %v", data)
	}
}

// TestSendFileCard_CDNFailFallsBackToText CDN 上传 200 但缺
// x-encrypted-param 头：抓错误记录后文本降级。
func TestSendFileCard_CDNFailFallsBackToText(t *testing.T) {
	p := tempCapturePath(t)

	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200) // 200 但缺 x-encrypted-param 头 → 失败
	}))
	defer cdn.Close()
	useTestCDN(t, cdn.URL+"/c2c")

	var mu sync.Mutex
	var lastText string
	srv := newCardServer(t, func(path string, r *http.Request, body []byte) (int, string) {
		switch path {
		case "/ilink/bot/getuploadurl":
			return 200, `{"upload_param":"UP"}`
		case "/ilink/bot/sendmessage":
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
			_ = json.Unmarshal(body, &payload)
			for _, it := range payload.Msg.ItemList {
				if it.Type == 1 {
					mu.Lock()
					lastText = it.TextItem.Text
					mu.Unlock()
				}
			}
			return 200, `{"errcode":0}`
		default:
			t.Errorf("不应请求其他端点: %s", path)
			return 404, ""
		}
	})
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.sendFn = func(string, string, string) error { return nil }
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "画"}}}})
	if err := s.SendFileCard(img, ""); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	sent := lastText
	mu.Unlock()
	if !strings.Contains(sent, "🖼 产物已生成：a.png") {
		t.Fatalf("CDN 失败应文本降级: %q", sent)
	}
	lines := readCaptureLines(t, p)
	if len(lines) != 1 {
		t.Fatalf("应恰 1 行记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["stage"] != "upload" || !strings.Contains(data["err"].(string), "x-encrypted-param") {
		t.Fatalf("应记 CDN 缺头错误: %v", data)
	}
}

// TestSendFileCard_NoPeerSkipsUpload 无活跃会话（LastPeer 为空）：不发起
// 上传请求，抓一行 skipped=no_peer 后文本降级（Push 无回推目标返回错误）。
func TestSendFileCard_NoPeerSkipsUpload(t *testing.T) {
	p := tempCapturePath(t)

	srv := newCardServer(t, func(path string, r *http.Request, body []byte) (int, string) {
		t.Errorf("无会话不应发起上传请求: %s", path)
		return 500, ""
	})
	defer srv.Close()

	img := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(img, []byte("PNG"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, nil)
	if err := s.SendFileCard(img, ""); err == nil {
		t.Fatal("无活跃会话时 Push 应返回错误")
	}
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰 1 行 skipped 记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["skipped"] != "no_peer" {
		t.Fatalf("skipped 原因应为 no_peer: %v", lines[0])
	}
}

// TestSendFileCard_FileRejectedFallsBack 扩展名白名单外（图片产物场景限定
// png/jpeg/webp/gif）：不发起上传，抓 stage=upload 错误后文本降级。
func TestSendFileCard_FileRejectedFallsBack(t *testing.T) {
	p := tempCapturePath(t)

	var mu sync.Mutex
	var gotTextPath bool
	srv := newCardServer(t, func(path string, r *http.Request, body []byte) (int, string) {
		switch path {
		case "/ilink/bot/getuploadurl":
			t.Error("白名单外文件不应发起上传")
			return 500, ""
		case "/ilink/bot/sendmessage":
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
			_ = json.Unmarshal(body, &payload)
			for _, it := range payload.Msg.ItemList {
				if it.Type == 1 {
					mu.Lock()
					gotTextPath = true
					mu.Unlock()
				}
			}
			return 200, `{"errcode":0}`
		default:
			t.Errorf("不应请求其他端点: %s", path)
			return 404, ""
		}
	})
	defer srv.Close()

	txt := filepath.Join(t.TempDir(), "笔记.txt")
	if err := os.WriteFile(txt, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t", CapturePath: p}, func(string, string) (string, error) { return "ok", nil })
	s.sendFn = func(string, string, string) error { return nil }
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{Type: 1, TextItem: &textItem{Text: "写"}}}})
	if err := s.SendFileCard(txt, ""); err != nil {
		t.Fatalf("SendFileCard: %v", err)
	}
	mu.Lock()
	textOK := gotTextPath
	mu.Unlock()
	if !textOK {
		t.Fatal("白名单外应走文本降级")
	}
	lines := readCaptureLines(t, p)
	if len(lines) != 1 || lines[0]["kind"] != "upload_probe" {
		t.Fatalf("应恰 1 行记录: %v", lines)
	}
	if data, _ := lines[0]["data"].(map[string]interface{}); data["stage"] != "upload" {
		t.Fatalf("白名单外应记 stage=upload 错误: %v", lines[0])
	}
}
