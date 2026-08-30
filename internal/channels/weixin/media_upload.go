// Package weixin — iLink 图片上传真实现（v4.8.3）：协议经三方印证定稿——
// 本机抓包（入站密文 AES-128-ECB 实测解密成功）、hermes-agent weixin.py
// 生产实现、openilink-sdk-go 导出符号。上传不再猜端点：
//
//	① POST {api}/ilink/bot/getuploadurl（filekey/media_type/to_user_id/
//	   rawsize/rawfilemd5/filesize/no_need_thumb/aeskey，鉴权头同 apiPost）
//	   → 响应 upload_full_url 或 upload_param
//	② 明文 AES-128-ECB(PKCS7) 加密
//	③ POST 上传地址（优先 upload_full_url；否则 CDN 基地址拼
//	   /upload?encrypted_query_param=…&filekey=…），body=密文，
//	   Content-Type: application/octet-stream → 响应头 x-encrypted-param
//	④ sendmessage image_item{type:2, media{encrypt_query_param,
//	   aes_key: base64(hex), encrypt_type:1}, mid_size}
//	   ⚠ aes_key 必须 base64(hex字符串)，base64(原始字节) 接收端解出灰框
//
// 上传域即本 API 域 + 固定 CDN 域（novac2c.cdn.weixin.qq.com），与扫码登录
// 的 baseurl/redirect_host 无关。任何失败一律降级现有文本卡片。
package weixin

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// wxCDNBaseURL 微信媒体 CDN 基地址（入站抓包与 hermes/SDK 一致；var 供测试注入）。
var wxCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"

const (
	wxUploadAPITimeout = 15 * time.Second // getuploadurl/sendmessage
	wxCDNTimeout       = 60 * time.Second // CDN 上传（大文件给足余量）
	mediaTypeImage     = 1                // getuploadurl media_type（image/video/file/voice = 1/2/3/4）
)

// getUploadURLReq / getUploadURLResp getuploadurl 请求/响应（hermes 同款字段）。
type getUploadURLReq struct {
	FileKey     string            `json:"filekey"`
	MediaType   int               `json:"media_type"`
	ToUserID    string            `json:"to_user_id"`
	RawSize     int64             `json:"rawsize"`
	RawFileMD5  string            `json:"rawfilemd5"`
	FileSize    int64             `json:"filesize"` // 加密后（PKCS7 对齐）大小
	NoNeedThumb bool              `json:"no_need_thumb"`
	AESKeyHex   string            `json:"aeskey"`
	BaseInfo    map[string]string `json:"base_info,omitempty"`
}

type getUploadURLResp struct {
	UploadFullURL string `json:"upload_full_url,omitempty"`
	UploadParam   string `json:"upload_param,omitempty"`
}

// uploadImageToCDN 完整上传链路：读文件→生成密钥→getuploadurl→加密→CDN
// 上传→返回 sendmessage 可用的 image_item（map 形态）。任何失败原样返回
// 错误，由 SendFileCard 抓包后降级文本。
func (s *Server) uploadImageToCDN(localPath, toUserID string) (map[string]interface{}, error) {
	data, err := readUploadableFile(localPath)
	if err != nil {
		return nil, err
	}

	fileKey := randomHex(16) // 32 位 hex（hermes: secrets.token_hex(16)）
	aesKey := make([]byte, 16)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}
	ciphertext, err := aes128ECBEncrypt(data, aesKey)
	if err != nil {
		return nil, err
	}

	// ① getuploadurl（鉴权头同 apiPost，请求体带 base_info）
	reqBody, _ := json.Marshal(getUploadURLReq{
		FileKey:     fileKey,
		MediaType:   mediaTypeImage,
		ToUserID:    toUserID,
		RawSize:     int64(len(data)),
		RawFileMD5:  md5Hex(data),
		FileSize:    int64(len(ciphertext)),
		NoNeedThumb: true,
		AESKeyHex:   hex.EncodeToString(aesKey),
		BaseInfo:    s.baseInfo(),
	})
	respBody, err := s.apiPost("/ilink/bot/getuploadurl", reqBody, wxUploadAPITimeout)
	if err != nil {
		return nil, fmt.Errorf("getuploadurl: %w", err)
	}
	var resp getUploadURLResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("getuploadurl 响应解析: %w", err)
	}

	// ② 上传地址：优先 upload_full_url，其次 upload_param 拼 CDN
	// （hermes 实测注释：upload_full_url 的旧 PUT 会 404，统一用 POST）
	uploadURL := strings.TrimSpace(resp.UploadFullURL)
	if uploadURL == "" {
		if resp.UploadParam == "" {
			return nil, fmt.Errorf("getuploadurl 未返回 upload_param/upload_full_url: %s", truncateRunes(string(respBody), 200))
		}
		uploadURL = wxCDNBaseURL + "/upload?encrypted_query_param=" +
			url.QueryEscape(resp.UploadParam) + "&filekey=" + url.QueryEscape(fileKey)
	}

	// ③ CDN 上传：密文直出 body，响应头 x-encrypted-param 即下载票据
	encryptedParam, err := s.uploadCiphertext(uploadURL, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("cdn upload: %w", err)
	}

	// ④ 组 image_item（字段参照入站真实形态 + hermes 出站构造）
	return map[string]interface{}{
		"type": 2, // ITEM_IMAGE（真机实测 type=2，非文档猜的 3）
		"image_item": map[string]interface{}{
			"media": map[string]interface{}{
				"encrypt_query_param": encryptedParam,
				"aes_key":             aesKeyForAPI(aesKey), // base64(hex 字符串)，勿改
				"encrypt_type":        1,
			},
			"mid_size": len(ciphertext),
		},
	}, nil
}

// uploadCiphertext POST 密文到 CDN，成功响应必带 x-encrypted-param 头。
func (s *Server) uploadCiphertext(uploadURL string, ciphertext []byte) (string, error) {
	req, err := http.NewRequest("POST", uploadURL, bytes.NewReader(ciphertext))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	c := s.client
	if wxCDNTimeout < c.Timeout {
		c = netclient.NewSimpleClient(wxCDNTimeout)
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateRunes(string(body), 120))
	}
	param := resp.Header.Get("x-encrypted-param")
	if param == "" {
		return "", fmt.Errorf("CDN 响应缺 x-encrypted-param 头: %s", truncateRunes(string(body), 120))
	}
	return param, nil
}

// sendImageCardViaUpload 组 sendmessage 发送图片卡片（image_item 形态；
// 鉴权与信封同 Send）。caption 不并入本条——失败降级路径避免图文重复，
// 发送成功后由调用方补发独立文本（对齐 hermes：caption 为独立消息）。
func (s *Server) sendImageCardViaUpload(item map[string]interface{}) error {
	from, ctxTokn := s.LastPeer()
	msg := map[string]interface{}{
		"from_user_id":  "",
		"to_user_id":    from,
		"client_id":     genClientID(),
		"message_type":  2,
		"message_state": 2,
		"item_list":     []interface{}{item},
	}
	if ctxTokn != "" {
		msg["context_token"] = ctxTokn
	}
	body, _ := json.Marshal(map[string]interface{}{"msg": msg, "base_info": s.baseInfo()})
	respBody, err := s.apiPost("/ilink/bot/sendmessage", body, wxUploadAPITimeout)
	if err != nil {
		return err
	}
	// 服务端显式错误码（缺失视为成功，与 Send 语义一致）
	var ack struct {
		ErrCode *int `json:"errcode"`
		Ret     *int `json:"ret"`
	}
	_ = json.Unmarshal(respBody, &ack)
	if (ack.ErrCode != nil && *ack.ErrCode != 0) || (ack.Ret != nil && *ack.Ret != 0) {
		return fmt.Errorf("sendmessage errcode=%d ret=%d", derefInt(ack.ErrCode), derefInt(ack.Ret))
	}
	return nil
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// baseName 测试友好的 filepath.Base 别名（保持上传流程文件名语义集中）。
func baseName(path string) string { return filepath.Base(path) }
