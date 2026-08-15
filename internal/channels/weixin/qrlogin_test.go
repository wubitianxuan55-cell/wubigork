package weixin

// T7-2 可见性收口：PollQRStatusWithCode 对齐 PollQRStatus——非 200 状态码与
// 坏 JSON 必须返回 error，绝不静默吞错（伪造成功的空结果）。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// qrTestServer 启动指向 qrBaseURL 的 httptest server，返回可注入的基地址。
func qrTestServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := qrBaseURL
	qrBaseURL = srv.URL
	t.Cleanup(func() { qrBaseURL = old })
	return srv.URL
}

// TestPollQRStatusWithCode_Non200ReturnsError 非 200 状态码必须返回 error
// （当前端伪造 HTTP 500 时，调用方应看到失败而不是假装成功）。
func TestPollQRStatusWithCode_Non200ReturnsError(t *testing.T) {
	qrTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/get_qrcode_status" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if q := r.URL.Query().Get("qrcode"); q != "abc123" {
			t.Errorf("qrcode = %q", q)
		}
		if v := r.URL.Query().Get("verify_code"); v != "vc9" {
			t.Errorf("verify_code = %q", v)
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	resp, err := PollQRStatusWithCode("abc123", "vc9")
	if err == nil {
		t.Fatal("非 200 状态码应返回 error")
	}
	if resp != nil {
		t.Errorf("失败时不应返回结果, got %+v", resp)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("错误应包含状态码, got %q", err.Error())
	}
}

// TestPollQRStatusWithCode_BadJSONReturnsError 坏 JSON 必须返回 error，
// 而不是静默返回零值结果。
func TestPollQRStatusWithCode_BadJSONReturnsError(t *testing.T) {
	qrTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not-json"))
	})
	resp, err := PollQRStatusWithCode("abc123", "vc9")
	if err == nil {
		t.Fatal("坏 JSON 应返回 error")
	}
	if resp != nil {
		t.Errorf("坏 JSON 不应返回结果, got %+v", resp)
	}
}

// TestPollQRStatusWithCode_Success 200 + 合法 JSON：正常解析出结果。
func TestPollQRStatusWithCode_Success(t *testing.T) {
	qrTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"scanned","bot_token":"tok-1","ilink_bot_id":"b1"}`))
	})
	resp, err := PollQRStatusWithCode("abc123", "vc9")
	if err != nil {
		t.Fatalf("成功路径不应报错: %v", err)
	}
	if resp == nil || resp.Status != "scanned" || resp.BotToken != "tok-1" || resp.ILinkBotID != "b1" {
		t.Fatalf("解析结果异常: %+v", resp)
	}
}

// TestPollQRStatusWithCode_EmptyJSON 200 + 空 JSON（合法但无字段）：解析成功，
// 返回零值结构（对齐 PollQRStatus 语义，不误报为错误）。
func TestPollQRStatusWithCode_EmptyJSON(t *testing.T) {
	qrTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	resp, err := PollQRStatusWithCode("abc123", "vc9")
	if err != nil {
		t.Fatalf("空 JSON 不应报错: %v", err)
	}
	if resp == nil || resp.Status != "" {
		t.Fatalf("空 JSON 应返回零值结果: %+v", resp)
	}
}
