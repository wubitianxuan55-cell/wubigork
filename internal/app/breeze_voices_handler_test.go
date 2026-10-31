package app

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

// breezeVoicesTestState 构造指向假 breeze-server 的 mediaState（探活/端口全走 httptest）。
func breezeVoicesTestState(t *testing.T, handler http.Handler) (*mediaState, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	port := portFromURL(ts.URL)
	a := &mediaState{core: &core{cfg: &config.Config{BreezePort: port}}}
	return a, ts
}

func portFromURL(u string) int {
	idx := strings.LastIndex(u, ":")
	n := 0
	for _, c := range u[idx+1:] {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func TestBreezeListVoices(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/v1/voices", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"harbour","seconds":5.36,"saved":true,"ref_text":"t"}]`))
	})
	a, _ := breezeVoicesTestState(t, mux)

	res := a.BreezeListVoices()
	if res["started"] != true {
		t.Fatalf("started = %v, want true", res["started"])
	}
	voices, _ := res["voices"].([]map[string]interface{})
	if len(voices) != 1 || voices[0]["id"] != "harbour" {
		t.Fatalf("voices 解析不符: %v", res["voices"])
	}

	// 服务不在：started=false 且不报错（面板据此提示先启动）
	a2 := &mediaState{core: &core{cfg: &config.Config{BreezePort: portFromURL("http://127.0.0.1:1")}}}
	res2 := a2.BreezeListVoices()
	if res2["started"] != false {
		t.Fatalf("服务不在时 started = %v, want false", res2["started"])
	}
}

func TestBreezeSaveVoiceValidation(t *testing.T) {
	a := &mediaState{core: &core{cfg: &config.Config{BreezePort: 1}}} // 不会触网：校验先失败
	cases := []struct {
		nm            string
		name          string
		refText       string
		refAudioB64   string
		wantErrSubstr string
	}{
		{"非法名称", "bad name!", "t", "aGk=", "只允许字母/数字"},
		{"空转写", "ok_name", "", "aGk=", "转写文本不能为空"},
		{"坏 base64", "ok_name", "t", "!!!", "解码失败"},
	}
	for _, tc := range cases {
		res := a.BreezeSaveVoice(tc.name, "", tc.refText, tc.refAudioB64)
		if res["ok"] != false {
			t.Errorf("[%s] 应校验失败, got %v", tc.nm, res)
		}
		if err, _ := res["error"].(string); !strings.Contains(err, tc.wantErrSubstr) {
			t.Errorf("[%s] 错误信息应含 %q, got %q", tc.nm, tc.wantErrSubstr, err)
		}
	}
}

func TestBreezeSaveVoiceHappyPath(t *testing.T) {
	var gotName, gotRefText, gotDesc, gotAudio string
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/v1/voices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			w.WriteHeader(400)
			return
		}
		gotName = r.FormValue("name")
		gotRefText = r.FormValue("ref_text")
		gotDesc = r.FormValue("description")
		f, _, err := r.FormFile("ref_audio")
		if err != nil {
			w.WriteHeader(400)
			return
		}
		b, _ := io.ReadAll(f)
		gotAudio = string(b)
		_, _ = w.Write([]byte(`{"id":"harbour","seconds":5.36}`))
	})
	a, _ := breezeVoicesTestState(t, mux)

	wav := "RIFF....fake"
	res := a.BreezeSaveVoice("harbour", "沉稳男声", "The harbour lights came on.", base64.StdEncoding.EncodeToString([]byte(wav)))
	if res["ok"] != true {
		t.Fatalf("保存应成功, got %v", res)
	}
	if gotName != "harbour" || gotRefText != "The harbour lights came on." || gotDesc != "沉稳男声" {
		t.Errorf("multipart 字段不符: name=%q refText=%q desc=%q", gotName, gotRefText, gotDesc)
	}
	if gotAudio != wav {
		t.Errorf("音频字节不符: %q", gotAudio)
	}
}

func TestBreezeDeleteVoice(t *testing.T) {
	var gotMethod string
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/v1/voices/", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{"deleted":"harbour","file_kept":true}`))
	})
	a, _ := breezeVoicesTestState(t, mux)

	res := a.BreezeDeleteVoice("harbour")
	if res["ok"] != true || gotMethod != http.MethodDelete {
		t.Fatalf("删除应成功且走 DELETE, got %v method=%s", res, gotMethod)
	}
	// 非法 id 直接拒（不触网）
	if res := a.BreezeDeleteVoice("bad id"); res["ok"] != false {
		t.Errorf("非法 id 应拒绝, got %v", res)
	}
}

func TestBreezeVoiceNameRe(t *testing.T) {
	// 服务端规则镜像：字母数字-_，1-64
	for _, ok := range []string{"harbour", "my_voice-2", "A"} {
		if !breezeVoiceNameRe.MatchString(ok) {
			t.Errorf("%q 应通过", ok)
		}
	}
	for _, bad := range []string{"", "bad name", "中文", strings.Repeat("a", 65)} {
		if breezeVoiceNameRe.MatchString(bad) {
			t.Errorf("%q 应拒绝", bad)
		}
	}
	if _, err := json.Marshal(map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
}
