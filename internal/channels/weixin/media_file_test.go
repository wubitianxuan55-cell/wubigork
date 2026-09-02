package weixin

// v4.9 入站文件消息测试：file_item 真机形态防御解析（len 数字/字符串双容忍、
// 旧字段兼容、media 同构）、resolveInboundFile 下载解密落盘（MD5 校验策略/
// 50MiB 上限/SSRF 私网拒）、handle FileHandler 跨线契约（成功注入/panic/空串/
// nil 零流量回退占位）。真机形态对照 2026-09-02 16:58 抓包
// （%LocalAppData%\gaea\wx_capture.jsonl 最新 inbound_media，type=4 与图片同构）。

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bytes16Key 重复 b 得 16 字节测试密钥。
func bytes16Key(b byte) []byte { return bytes.Repeat([]byte{b}, 16) }

// realFileItem 按真机形态构造 file_item（aes_key=base64(hex 字符串)、len 为
// 字符串形态的明文字节数、md5 为明文摘要）。
func realFileItem(fullURL string, key, plaintext []byte) fileItem {
	return fileItem{
		FileName: "专家评审打分报告_中科一兵.docx",
		MD5:      md5Hex(plaintext),
		Len:      int64(len(plaintext)),
		Media: &cdnMedia{
			EncryptQueryParam: "WmlpdjJSdFN",
			AESKey:            base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(key))),
			FullURL:           fullURL,
		},
	}
}

// TestFileItem_ParseRealShape 真机抓包形态解析：type=4 + media 同构 +
// file_name/md5/len；len 真机为字符串 → 数值；旧留位字段兼容；media 多态
// 不炸整批解析。
func TestFileItem_ParseRealShape(t *testing.T) {
	rawKey := "0123456789abcdef0123456789abcdef"
	b64 := base64.StdEncoding.EncodeToString([]byte(rawKey))
	full := "https://novac2c.cdn.weixin.qq.com/c2c/download?encrypted_query_param=WmlpdjJSdFN&taskid=t1"

	// 形态一（真机实测）：经 pollResp 真实解析链
	raw := `{"msgs":[{"from_user_id":"u1","item_list":[{"type":4,"is_completed":true,"file_item":{` +
		`"media":{"encrypt_query_param":"EQ","aes_key":"` + b64 + `","full_url":"` + full + `"},` +
		`"file_name":"专家评审打分报告_中科一兵.docx","md5":"e3874218611c34c92b4474663761fb9d","len":"433849"}}]}]}`
	var resp pollResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("真机形态解析不应失败: %v", err)
	}
	fi := resp.Msgs[0].ItemList[0].FileItem
	if fi == nil || fi.FileName != "专家评审打分报告_中科一兵.docx" || fi.MD5 != "e3874218611c34c92b4474663761fb9d" {
		t.Fatalf("真机 file_item 解析 = %+v", fi)
	}
	if fi.Len != 433849 { // ⚠ len 真机为字符串 "433849"
		t.Fatalf("len 字符串应解析为 433849, 实际 %d", fi.Len)
	}
	if fi.Media == nil || fi.Media.FullURL != full || fi.Media.EncryptQueryParam != "EQ" {
		t.Fatalf("media 同构解析 = %+v", fi.Media)
	}
	key, err := aesKeyFromBase64Hex(fi.Media.AESKey)
	if err != nil || len(key) != 16 {
		t.Fatalf("media.aes_key 应反解出 16 字节密钥: %v", err)
	}

	// 形态二：len 数字形态 + 旧留位字段兼容
	var fi2 fileItem
	if err := json.Unmarshal([]byte(`{"file_id":123,"file_name":"a.docx","len":123,"md5":"x","size":"9","url":"https://c/a"}`), &fi2); err != nil {
		t.Fatalf("len 数字形态解析: %v", err)
	}
	if fi2.Len != 123 || fi2.Size != 9 || fi2.FileID != "123" || fi2.URL != "https://c/a" || fi2.MD5 != "x" {
		t.Fatalf("数字/旧字段降级结果 = %+v", fi2)
	}

	// 形态三：media 整体非对象、url 数组等怪异负载——不炸、降级零值
	var fi3 fileItem
	if err := json.Unmarshal([]byte(`{"name":"d.pdf","size":"2048","url":["bad"],"media":"garbage"}`), &fi3); err != nil {
		t.Fatalf("media 怪异形态不应报错: %v", err)
	}
	if fi3.Name != "d.pdf" || fi3.Size != 2048 || fi3.URL != "" || fi3.Media != nil {
		t.Fatalf("怪异负载降级结果 = %+v", fi3)
	}

	// 形态四：file_item 整体非对象（数组/null）——零值/nil，宁漏勿误
	var resp4 pollResp
	_ = json.Unmarshal([]byte(`{"msgs":[{"item_list":[{"type":4,"file_item":[1,2]},{"type":4,"file_item":null}]}]}`), &resp4)
	if len(resp4.Msgs) != 1 || len(resp4.Msgs[0].ItemList) != 2 {
		t.Fatalf("非对象负载应不影响整批解析: %+v", resp4)
	}
	if it := resp4.Msgs[0].ItemList[0].FileItem; it == nil || it.Name != "" {
		t.Fatalf("数组形态应降级零值: %+v", it)
	}
	if it := resp4.Msgs[0].ItemList[1].FileItem; it != nil {
		t.Fatalf("null 形态应保持 nil: %+v", it)
	}
}

// TestFileItemLabel_ShowsNameAndSize 占位提示展示 file_name 与人类可读大小；
// 旧字段与全缺形态兼容。
func TestFileItemLabel_ShowsNameAndSize(t *testing.T) {
	if got := fileItemLabel(fileItem{FileName: "报告.docx", Len: 433849}); !strings.Contains(got, "报告.docx") || !strings.Contains(got, "KB") {
		t.Fatalf("应展示 file_name 与大小: %q", got)
	}
	if got := fileItemLabel(fileItem{Name: "旧.pdf", Size: 3}); !strings.Contains(got, "旧.pdf") || !strings.Contains(got, "3B") {
		t.Fatalf("旧留位字段兼容: %q", got)
	}
	if got := fileItemLabel(fileItem{FileName: "无大小.docx"}); got != "文件消息（无大小.docx）" {
		t.Fatalf("无大小不应显示尺寸: %q", got)
	}
	if got := fileItemLabel(fileItem{}); got != "文件消息" {
		t.Fatalf("全缺应退裸标签: %q", got)
	}
}

// newFileCDN 加密文件 CDN httptest：返回 key 加密后的 plaintext 密文。
func newFileCDN(t *testing.T, key, plaintext []byte) *httptest.Server {
	t.Helper()
	ct, err := aes128ECBEncrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(ct)
	}))
}

// TestResolveInboundFile_DownloadDecryptMD5 真机形态全链路：下载密文 →
// AES-128-ECB 解密 → MD5 与 file_item.md5 比对一致 → 落 gaea-wxfile-*.tmp，
// 明文/文件名/大小/摘要逐项锁定。
func TestResolveInboundFile_DownloadDecryptMD5(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes16Key(0x5a)
	plaintext := []byte("DOCX-评审打分报告真实字节内容-中科一兵")
	cdn := newFileCDN(t, key, plaintext)
	defer cdn.Close()

	fi := realFileItem(cdn.URL+"/download?encrypted_query_param=W&taskid=t1", key, plaintext)
	path, name, size, md5sum, err := resolveInboundFile(fi)
	if err != nil {
		t.Fatalf("resolveInboundFile: %v", err)
	}
	defer os.Remove(path)
	if name != "专家评审打分报告_中科一兵.docx" {
		t.Fatalf("文件名应取 file_name: %q", name)
	}
	if size != int64(len(plaintext)) || md5sum != md5Hex(plaintext) {
		t.Fatalf("大小/摘要应为明文实测值: size=%d md5=%s", size, md5sum)
	}
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "gaea-wxfile-") || !strings.HasSuffix(base, ".tmp") {
		t.Fatalf("临时文件命名应为 gaea-wxfile-*.tmp: %q", base)
	}
	got, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(got), "评审打分报告真实字节内容") {
		t.Fatalf("落盘应为解密明文: err=%v got=%q", err, got)
	}
}

// TestResolveInboundFile_MD5MismatchKept MD5 对不上：不拒收（Warn 保留），
// 文件照常落盘返回——微信 CDN 偶发差异以实测为准。
func TestResolveInboundFile_MD5MismatchKept(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes16Key(0x5b)
	plaintext := []byte("文件字节-MD5-与声明不符")
	cdn := newFileCDN(t, key, plaintext)
	defer cdn.Close()

	fi := realFileItem(cdn.URL+"/d", key, plaintext)
	fi.MD5 = "00000000000000000000000000000000" // 声明摘要故意错
	path, _, size, md5sum, err := resolveInboundFile(fi)
	if err != nil {
		t.Fatalf("MD5 不符不应拒收: %v", err)
	}
	defer os.Remove(path)
	if size != int64(len(plaintext)) || md5sum != md5Hex(plaintext) {
		t.Fatalf("应返回明文实测值: size=%d md5=%s", size, md5sum)
	}
}

// TestResolveInboundFile_Rejections 防线矩阵：声明大小超 50MiB（len 与旧留位
// size 双形态，超限不发起下载）、SSRF 私网/回环拒（生产 mediaAllowLoopback=
// false，dial 前拦截）、缺下载地址、media.aes_key 非法（绝不把密文当明文消费）。
func TestResolveInboundFile_Rejections(t *testing.T) {
	key := bytes16Key(0x5c)
	plaintext := []byte("x")

	// 声明大小预检：超 50MiB 直接拒绝，CDN 不应被请求
	cdnHit := false
	oversize := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnHit = true
		_, _ = w.Write([]byte("should not be fetched"))
	}))
	defer oversize.Close()

	fi := realFileItem(oversize.URL+"/d", key, plaintext)
	fi.Len = wxFileMaxBytes + 1
	if _, _, _, _, err := resolveInboundFile(fi); err == nil {
		t.Fatal("声明大小超 50MiB 应拒绝")
	}
	// 旧留位 size 字段同样参与预检
	fi2 := realFileItem(oversize.URL+"/d", key, plaintext)
	fi2.Len = 0
	fi2.Size = wxFileMaxBytes + 1
	if _, _, _, _, err := resolveInboundFile(fi2); err == nil {
		t.Fatal("留位 size 超限应拒绝")
	}
	if cdnHit {
		t.Fatal("声明大小超限不应发起下载")
	}

	// 私网/回环 SSRF（生产防线：回环也拒；无需真实监听——dial 前拦截）
	for _, u := range []string{
		"http://127.0.0.1:1/x",
		"http://10.1.2.3/x",
		"http://192.168.1.1/x",
		"http://169.254.169.254/latest",
		"http://100.100.100.200/x",
	} {
		fi3 := realFileItem(u, key, plaintext)
		path, _, _, _, err := resolveInboundFile(fi3)
		if err == nil {
			os.Remove(path)
			t.Fatalf("%s 应被 SSRF 防线拒绝", u)
		}
	}

	// 缺下载地址
	if _, _, _, _, err := resolveInboundFile(fileItem{FileName: "a.docx"}); err == nil {
		t.Fatal("无 media.full_url/url 应报错")
	}

	// media.aes_key 非法：显式报错，绝不落盘密文
	cdn := newFileCDN(t, key, plaintext)
	defer cdn.Close()
	fi4 := realFileItem(cdn.URL+"/d", key, plaintext)
	fi4.Media.AESKey = "!!!not-base64-hex!!!"
	if _, _, _, _, err := resolveInboundFile(fi4); err == nil {
		t.Fatal("aes_key 解析失败应报错")
	}
}

// TestHandle_InboundFileHandlerPipeline FileHandler 跨线契约（handle 真实路径）：
// 成功→返回值注入喂模型+临时文件用后即删；空串/panic→占位提示；nil→零流量
// 占位（不发起下载）。
func TestHandle_InboundFileHandlerPipeline(t *testing.T) {
	restore := allowLoopback(t)
	defer restore()

	key := bytes16Key(0x5d)
	plaintext := []byte("季度评审数据：合格率 98.6%")
	cdn := newFileCDN(t, key, plaintext)
	defer cdn.Close()

	newFileServer := func() (*Server, *string) {
		got := ""
		s := New(Config{ILinkURL: "http://unused", BotToken: "tok", AssistantID: "t"}, func(q, from string) (string, error) {
			got = q
			return "ok", nil
		})
		s.sendFn = func(string, string, string) error { return nil }
		return s, &got
	}
	fileMsg := func(fullURL string) *inboundMsg {
		fi := realFileItem(fullURL, key, plaintext)
		return &inboundMsg{FromUserID: "u1", ContextToken: "ctx", ItemList: []itemElem{{
			Type:     4,
			FileItem: &fi,
		}}}
	}

	// 成功路：返回值作为注入文本行；契约参数逐项校验；临时文件用后即删
	s, got := newFileServer()
	var handlerPath string
	s.FileHandler = func(localPath, fileName string, sizeBytes int64, md5sum string) string {
		handlerPath = localPath
		if fileName != "专家评审打分报告_中科一兵.docx" || sizeBytes != int64(len(plaintext)) || md5sum != md5Hex(plaintext) {
			t.Errorf("契约参数不符: name=%q size=%d md5=%s", fileName, sizeBytes, md5sum)
		}
		return "（用户发来文件「" + fileName + "」，提取内容：" + string(plaintext) + "）"
	}
	s.handle(fileMsg(cdn.URL + "/download?encrypted_query_param=W"))
	if !strings.Contains(*got, "（用户发来文件「专家评审打分报告_中科一兵.docx」") || !strings.Contains(*got, "98.6%") {
		t.Fatalf("处理器返回值应注入喂模型: %q", *got)
	}
	if handlerPath == "" {
		t.Fatal("处理器应收到本地路径")
	}
	if _, err := os.Stat(handlerPath); !os.IsNotExist(err) {
		t.Fatal("处理器返回后临时文件应删除（自持需复制）")
	}

	// 空串：回退占位提示行
	s2, got2 := newFileServer()
	s2.FileHandler = func(string, string, int64, string) string { return "  " }
	s2.handle(fileMsg(cdn.URL + "/d"))
	if !strings.Contains(*got2, "文件消息（专家评审打分报告_中科一兵.docx") || !strings.Contains(*got2, "内容暂无法读取") {
		t.Fatalf("空串应回退占位: %q", *got2)
	}

	// panic：recover 后回退占位（同 OnInboundImage 先例）
	s3, got3 := newFileServer()
	s3.FileHandler = func(string, string, int64, string) string { panic("提取器爆炸") }
	s3.handle(fileMsg(cdn.URL + "/d"))
	if !strings.Contains(*got3, "内容暂无法读取") {
		t.Fatalf("panic 应回退占位: %q", *got3)
	}

	// nil：零流量（不请求 CDN），行为与现状一致
	s4, got4 := newFileServer()
	var hits int
	counting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("x"))
	}))
	defer counting.Close()
	s4.handle(fileMsg(counting.URL + "/d"))
	if hits != 0 {
		t.Fatalf("FileHandler nil 不应发起下载, 实际 %d 次", hits)
	}
	if !strings.Contains(*got4, "文件消息") {
		t.Fatalf("nil 应占位提示: %q", *got4)
	}
}

// TestReadUploadableFileAny 文件卡链的可上传校验：类型不限、仅尺寸上限。
func TestReadUploadableFileAny(t *testing.T) {
	small := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(small, []byte("# md 内容"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := readUploadableFileAny(small, wxFileMaxBytes)
	if err != nil || string(data) != "# md 内容" {
		t.Fatalf("类型不限应放行: %v %q", err, data)
	}
	if _, err := readUploadableFileAny(small, 2); err == nil {
		t.Fatal("超上限应拒绝")
	}
	if _, err := readUploadableFileAny(filepath.Join(t.TempDir(), "missing.docx"), wxFileMaxBytes); err == nil {
		t.Fatal("文件不存在应报错")
	}
}
