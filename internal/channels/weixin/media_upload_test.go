package weixin

// v4.8.3 媒体协议测试：AES-128-ECB/PKCS7 原语、aes_key 双形态编解码、
// image_item 真机形态解析与 resolveDownload、DownloadImageEncrypted 解密
// 下载（SSRF/魔数防线在解密之后）、入站加密图片全链路「发图即识别」。
// 协议形态对照：本机抓包 inbound_media 样例 + hermes-agent 生产实现。

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestAES128ECBRoundTrip 加解密回环（含非对齐明文、空明文、块对齐明文）。
func TestAES128ECBRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x2b}, 16)
	for _, plain := range [][]byte{
		[]byte("hello"),
		[]byte(""),
		bytes.Repeat([]byte{7}, 16),
		bytes.Repeat([]byte("abc"), 100),
	} {
		ct, err := aes128ECBEncrypt(plain, key)
		if err != nil {
			t.Fatalf("encrypt(%d): %v", len(plain), err)
		}
		if len(ct)%16 != 0 || len(ct) == 0 {
			t.Fatalf("密文应按块对齐: %d", len(ct))
		}
		got, err := aes128ECBDecrypt(ct, key)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if !bytes.Equal(got, plain) {
			t.Fatalf("回环不符: %q != %q", got, plain)
		}
	}
}

// TestAES128ECBErrors 非法密钥/长度报错；错误填充宽容返回原文（hermes 语义）。
func TestAES128ECBErrors(t *testing.T) {
	if _, err := aes128ECBEncrypt([]byte("x"), make([]byte, 15)); err == nil {
		t.Fatal("15 字节密钥应报错")
	}
	key := bytes.Repeat([]byte{1}, 16)
	if _, err := aes128ECBDecrypt([]byte("short"), key); err == nil {
		t.Fatal("非块对齐密文应报错")
	}
	// 合法密文但解密结果填充值不合法：不报错、原样返回 16 字节（宁交上层
	// 魔数终审，不在这里误杀）
	block := bytes.Repeat([]byte{9}, 16)
	out, err := aes128ECBDecrypt(block, key)
	if err != nil || len(out) != 16 {
		t.Fatalf("非法填充应宽容返回: %v %q", err, out)
	}
}

// TestAESKeyForAPI aes_key 编码必须为 base64(hex 字符串)——hermes 实测：
// base64(原始字节) 会让接收端解出灰框。
func TestAESKeyForAPI(t *testing.T) {
	key := []byte{0x00, 0x01, 0xfe, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x10}
	want := base64.StdEncoding.EncodeToString([]byte("0001feff000000000000000000000010"))
	if got := aesKeyForAPI(key); got != want {
		t.Fatalf("aes_key = %q, want %q", got, want)
	}
	// 双向还原
	back, err := aesKeyFromBase64Hex(aesKeyForAPI(key))
	if err != nil || !bytes.Equal(back, key) {
		t.Fatalf("双向还原失败: %v %q", err, back)
	}
}

// TestParseAESKeyHex 非法 hex / 错误长度报错。
func TestParseAESKeyHex(t *testing.T) {
	if _, err := parseAESKeyHex("zz"); err == nil {
		t.Fatal("非法 hex 应报错")
	}
	if _, err := parseAESKeyHex("aabb"); err == nil {
		t.Fatal("2 字节解码应报错（须 16 字节）")
	}
	k, err := parseAESKeyHex(strings.Repeat("ab", 16))
	if err != nil || len(k) != 16 {
		t.Fatalf("32 位 hex 应解出 16 字节: %v", err)
	}
}

// TestImageItem_ResolveRealShape 真机抓包形态：type=2，image_item 携带
// aeskey(hex) + media{full_url, encrypt_query_param, aes_key(base64-of-hex)}，
// 无 url/file_id。resolveDownload 应给出 CDN 地址与 hex 密钥；media.aes_key
// 反解失败时宁漏勿误（key 留空）。
func TestImageItem_ResolveRealShape(t *testing.T) {
	rawKey := "0123456789abcdef0123456789abcdef"
	b64 := base64.StdEncoding.EncodeToString([]byte(rawKey))
	full := "https://novac2c.cdn.weixin.qq.com/c2c/download?encrypted_query_param=UU53"

	// 形态一（真机实测）：顶层 aeskey + media（经 inboundMsg 解析链验真实路径）
	var it imageItem
	msgBody := `{"from_user_id":"u1","item_list":[{"type":2,"image_item":{"aeskey":"` + rawKey +
		`","media":{"encrypt_query_param":"Q1","aes_key":"` + b64 + `","full_url":"` + full + `"},"mid_size":1024,"hd_size":1008}}]}`
	var msg inboundMsg
	if err := json.Unmarshal([]byte(msgBody), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(msg.ItemList) != 1 || msg.ItemList[0].ImageItem == nil {
		t.Fatalf("真机形态解析为空: %+v", msg)
	}
	it = *msg.ItemList[0].ImageItem
	url, key := it.resolveDownload()
	if url != full || key != rawKey {
		t.Fatalf("resolveDownload = (%q,%q), want (%q,%q)", url, key, full, rawKey)
	}

	// 形态二：仅 media.aes_key（base64-of-hex 反解）
	var it2 imageItem
	if err := json.Unmarshal([]byte(`{"media":{"full_url":"`+full+`","aes_key":"`+b64+`"}}`), &it2); err != nil {
		t.Fatalf("unmarshal2: %v", err)
	}
	if u, k := it2.resolveDownload(); u != full || k != rawKey {
		t.Fatalf("media.aes_key 反解 = (%q,%q)", u, k)
	}

	// 形态三：aes_key 非法 base64 → key 留空（宁漏勿误），url 仍可用
	var it3 imageItem
	if err := json.Unmarshal([]byte(`{"media":{"full_url":"u","aes_key":"!!!"}}`), &it3); err != nil {
		t.Fatalf("unmarshal3: %v", err)
	}
	if u, k := it3.resolveDownload(); u != "u" || k != "" {
		t.Fatalf("非法 aes_key 应降级: (%q,%q)", u, k)
	}

	// 形态四：明文 URL（留位字段 url 有值、无 media）→ key 空，保持原路
	var it4 imageItem
	if err := json.Unmarshal([]byte(`{"url":"https://cdn/a.png"}`), &it4); err != nil {
		t.Fatalf("unmarshal4: %v", err)
	}
	if u, k := it4.resolveDownload(); u != "https://cdn/a.png" || k != "" {
		t.Fatalf("明文形态应保持原路: (%q,%q)", u, k)
	}
}

// TestDownloadImageEncrypted_RoundTrip 加密 CDN 全链路：密文 → 解密 → 魔数
// 终审 → 临时文件内容等于明文；cleanup 幂等删除。
func TestDownloadImageEncrypted_RoundTrip(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes.Repeat([]byte{0x42}, 16)
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}, bytes.Repeat([]byte{0}, 64)...)
	jpeg = append(jpeg, 0xFF, 0xD9, 0, 0, 0, 0) // 真机形态：EOI 后带少量尾随字节
	ct, err := aes128ECBEncrypt(jpeg, key)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(ct)
	}))
	defer srv.Close()

	path, cleanup, err := DownloadImageEncrypted(srv.URL+"/download?encrypted_query_param=x", hex.EncodeToString(key))
	if err != nil {
		t.Fatalf("DownloadImageEncrypted: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, jpeg) {
		t.Fatalf("解密产物不符（尾随字节应保留）: %q", data)
	}
	cleanup()
	cleanup() // 幂等
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("cleanup 后临时文件应删除")
	}
}

// TestDownloadImageEncrypted_Rejections 错误密钥（魔数不过）、密文长度非块
// 对齐、URL 非 http(s)：显式报错，不落盘。
func TestDownloadImageEncrypted_Rejections(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes.Repeat([]byte{0x42}, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("short") == "1" {
			_, _ = w.Write([]byte("not-block-aligned")) // 17 字节
			return
		}
		_, _ = w.Write(bytes.Repeat([]byte{0xAB}, 64)) // 解密后非图片魔数
	}))
	defer srv.Close()

	if _, _, err := DownloadImageEncrypted(srv.URL+"?short=1", hex.EncodeToString(key)); err == nil {
		t.Fatal("非块对齐密文应报错")
	}
	if _, _, err := DownloadImageEncrypted(srv.URL, hex.EncodeToString(key)); err == nil {
		t.Fatal("解密后魔数不过应报错")
	}
	if _, _, err := DownloadImageEncrypted("ftp://x/y", hex.EncodeToString(key)); err == nil {
		t.Fatal("非 http(s) 应报错")
	}
	if _, _, err := DownloadImageEncrypted(srv.URL, "zz"); err == nil {
		t.Fatal("非法 aeskey 应报错")
	}
}

// TestDownloadImage_FileScheme file:// 分支：TempDir 内解密产物可读（no-op
// cleanup）、TempDir 外拒绝、非图片内容拒绝——消息内 URL 永远走不进这条分支。
func TestDownloadImage_FileScheme(t *testing.T) {
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 'J', 'F', 'I', 'F'}, bytes.Repeat([]byte{0}, 32)...)
	in := filepath.Join(t.TempDir(), "decrypted.img")
	if err := os.WriteFile(in, jpeg, 0o600); err != nil {
		t.Fatal(err)
	}

	path, cleanup, err := DownloadImage("file://" + in)
	if err != nil {
		t.Fatalf("file:// 读取: %v", err)
	}
	if path != in {
		t.Fatalf("应原样返回路径: %q", path)
	}
	cleanup() // no-op：文件所有权归构造方
	if _, err := os.Stat(in); err != nil {
		t.Fatalf("no-op cleanup 不应删除文件: %v", err)
	}

	// TempDir 外拒绝（t.TempDir 是 os.TempDir 的子目录，取其父目录之外才越界；
	// 前缀检查在任何文件访问之前，无需真实文件）
	outside := filepath.Clean(filepath.Join(os.TempDir(), "..", "outside.img"))
	if strings.HasPrefix(strings.ToLower(outside), strings.ToLower(os.TempDir())) {
		t.Skipf("测试环境 TempDir 父目录仍在临时区内: %s", outside)
	}
	if _, _, err := DownloadImage("file://" + outside); err == nil {
		t.Fatal("TempDir 外应拒绝")
	}

	// 非图片内容拒绝
	txt := filepath.Join(t.TempDir(), "evil.img")
	if err := os.WriteFile(txt, []byte("<script>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := DownloadImage("file://" + txt); err == nil {
		t.Fatal("非图片魔数应拒绝")
	}
}

// TestHandle_EncryptedImageOCR 入站加密图片全链路（v4.8.3 主目标）：
// 真机形态 type=2 消息（media.full_url 指向加密 CDN）→ 下载解密落盘 →
// MediaRecognizer 经 file:// 读到明文 → 识别文案进入回复。注入的识别器模拟
// OCRMediaRecognizer（DownloadImage(file://) → 读文件），证明 app 层现有
// 注入零改动即生效。
func TestHandle_EncryptedImageOCR(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes.Repeat([]byte{0x11}, 16)
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'},
		bytes.Repeat([]byte("画中一只橘猫"), 2)...)
	ct, err := aes128ECBEncrypt(jpeg, key)
	if err != nil {
		t.Fatal(err)
	}
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(ct)
	}))
	defer cdn.Close()

	var mu sync.Mutex
	var chatInput string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/ilink/bot/getupdates":
			item := `{"type":2,"image_item":{"aeskey":"` + hex.EncodeToString(key) +
				`","media":{"encrypt_query_param":"Q","aes_key":"` +
				base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(key))) +
				`","full_url":"` + cdn.URL + `/download?encrypted_query_param=U"},"mid_size":` + strconv.Itoa(len(ct)) + `}}`
			_, _ = w.Write([]byte(`{"ret":0,"errcode":0,"msgs":[{"from_user_id":"u1","item_list":[` + item + `]}],"get_updates_buf":"b"}`))
		default:
			t.Errorf("不应请求其他端点: %s", r.URL.Path)
			_, _ = w.Write([]byte(`{"errcode":0}`))
		}
	}))
	defer srv.Close()

	s := New(Config{ILinkURL: srv.URL, BotToken: "tok", AssistantID: "t"}, func(q, from string) (string, error) {
		mu.Lock()
		chatInput = q // 识别内容应拼进喂给模型的提示行
		mu.Unlock()
		return "已识别", nil
	})
	s.MediaRecognizer = func(rawURL string) (string, error) {
		// 模拟 OCRMediaRecognizer：DownloadImage 支持 file:// → 读文件交 OCR
		path, cleanup, err := DownloadImage(rawURL)
		if err != nil {
			return "", err
		}
		defer cleanup()
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return "OCR识别内容:" + string(data), nil
	}
	s.sendFn = func(string, string, string) error { return nil } // 吞自动回复
	s.handle(&inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{
		Type:      2,
		ImageItem: &imageItem{AESKey: hex.EncodeToString(key), Media: &cdnMedia{FullURL: cdn.URL + "/download?encrypted_query_param=U"}},
	}}})

	mu.Lock()
	defer mu.Unlock()
	if !strings.Contains(chatInput, "识别内容：OCR识别内容") || !strings.Contains(chatInput, "画中一只橘猫") {
		t.Fatalf("识别管线未生效，模型输入: %q", chatInput)
	}
}
