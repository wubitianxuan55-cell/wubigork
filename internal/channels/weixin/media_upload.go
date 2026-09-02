// Package weixin — iLink 媒体上传真实现（v4.8.3 图片真机定稿 + v4.9 文件卡
// 探针制泛化）：协议经三方印证定稿——本机抓包（入站密文 AES-128-ECB 实测解密
// 成功）、hermes-agent weixin.py 生产实现、openilink-sdk-go 导出符号。上传不猜端点：
//
//	① POST {api}/ilink/bot/getuploadurl（filekey/media_type/to_user_id/
//	   rawsize/rawfilemd5/filesize/no_need_thumb/aeskey，鉴权头同 apiPost）
//	   → 响应 upload_full_url 或 upload_param
//	② 明文 AES-128-ECB(PKCS7) 加密（随机密钥随信下发）
//	③ POST 上传地址（优先 upload_full_url；否则 CDN 基地址拼
//	   /upload?encrypted_query_param=…&filekey=…），body=密文，
//	   Content-Type: application/octet-stream → 响应头 x-encrypted-param
//	④ sendmessage 媒体卡片：
//	   图片 image_item{type:2, media{encrypt_query_param, aes_key: base64(hex),
//	   encrypt_type:1}, mid_size}（真机定稿）
//	   文件 file_item{type:4, file_name, len(数字), md5, media{...同款}}
//	   （探针制：media_type=3/file_item 形态待真机验证，upload_probe 已埋点）
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
	mediaTypeFile      = 3                // ⚠探针制：file=3 取自同一枚举注释，真机尚未验证，upload_probe 已埋点
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

// uploadImageToCDN 图片上传链路（真机定稿，逐字节不变）：读文件（扩展名
// 白名单 + 20MiB）→ 共享上传内核（media_type=1）→ 组 sendmessage 可用的
// image_item（map 形态）。任何失败原样返回错误，由 SendFileCard 抓包后降级文本。
func (s *Server) uploadImageToCDN(localPath, toUserID string) (map[string]interface{}, error) {
	data, err := readUploadableFile(localPath)
	if err != nil {
		return nil, err
	}
	param, aesKey, cipherLen, err := s.uploadMediaBytes(data, toUserID, "", mediaTypeImage)
	if err != nil {
		return nil, err
	}

	// ④ 组 image_item（字段参照入站真实形态 + hermes 出站构造）
	return map[string]interface{}{
		"type": 2, // ITEM_IMAGE（真机实测 type=2，非文档猜的 3）
		"image_item": map[string]interface{}{
			"media": map[string]interface{}{
				"encrypt_query_param": param,
				"aes_key":             aesKeyForAPI(aesKey), // base64(hex 字符串)，勿改
				"encrypt_type":        1,
			},
			"mid_size": cipherLen,
		},
	}, nil
}

// uploadFileToCDN 文件卡上传链路（v4.9 探针制）：与图片五步完全同构，差异仅
// 三处——① getuploadurl 的 media_type=3（枚举注释 image/video/file/voice=
// 1/2/3/4，file=3 的真实取值待真机验证，upload_probe 已埋点）；② 文件不设
// 扩展名白名单（docx/xlsx/pptx/pdf/zip/txt/md 等均可），尺寸上限 50MiB（与
// 入站文件防线同口径）；③ sendmessage 发 type=4 file_item{file_name,
// len(数字), md5, media{encrypt_query_param, aes_key}}（len/md5 均为明文口径，
// 对齐真机入站形态）。每次尝试经 upload_probe 记录完整请求/响应/errcode。
// 任何失败原样返回错误，由 SendFileCard 降级现有「图片失败→文本路径卡」链。
func (s *Server) uploadFileToCDN(localPath, toUserID string) (map[string]interface{}, error) {
	data, err := readUploadableFileAny(localPath, wxFileMaxBytes)
	if err != nil {
		return nil, err
	}
	param, aesKey, _, err := s.uploadMediaBytes(data, toUserID, filepath.Base(localPath), mediaTypeFile)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"type": 4, // ITEM_FILE（hermes 枚举 1=文本 2=图片 4=文件；探针制待真机验证）
		"file_item": map[string]interface{}{
			"file_name": filepath.Base(localPath),
			"len":       len(data), // 数字形态（真机入站 len 同为明文字节数）
			"md5":       md5Hex(data),
			"media": map[string]interface{}{
				"encrypt_query_param": param,
				"aes_key":             aesKeyForAPI(aesKey), // base64(hex 字符串)，与图片口径一致
				"encrypt_type":        1,
			},
		},
	}, nil
}

// uploadMediaBytes 上传五步的媒体无关内核：getuploadurl(mediaType) → 随机
// AES-128-ECB 密钥加密 → CDN PUT → 下载票据。probeFile 非空时逐节点经
// upload_probe 记录请求字段/响应原文/errcode（文件卡探针制；图片链传空串
// 保持既有 capture 语义零新增）。返回 (票据, aesKey 原始字节, 密文长度, 错误)。
func (s *Server) uploadMediaBytes(data []byte, toUserID, probeFile string, mediaType int) (encryptedParam string, aesKey []byte, cipherLen int, err error) {
	fileKey := randomHex(16) // 32 位 hex（hermes: secrets.token_hex(16)）
	aesKey = make([]byte, 16)
	if _, err = rand.Read(aesKey); err != nil {
		return "", nil, 0, err
	}
	ciphertext, err := aes128ECBEncrypt(data, aesKey)
	if err != nil {
		return "", nil, 0, err
	}

	// ① getuploadurl（鉴权头同 apiPost，请求体带 base_info）
	reqBody, _ := json.Marshal(getUploadURLReq{
		FileKey:     fileKey,
		MediaType:   mediaType,
		ToUserID:    toUserID,
		RawSize:     int64(len(data)),
		RawFileMD5:  md5Hex(data),
		FileSize:    int64(len(ciphertext)),
		NoNeedThumb: true,
		AESKeyHex:   hex.EncodeToString(aesKey),
		BaseInfo:    s.baseInfo(),
	})
	if probeFile != "" {
		capture("upload_probe", map[string]interface{}{
			"stage": "file_getuploadurl", "file": probeFile, "media_type": mediaType,
			"rawsize": len(data), "rawfilemd5": md5Hex(data), "filesize": len(ciphertext),
			"filekey": fileKey, "aeskey": hex.EncodeToString(aesKey),
		})
	}
	respBody, err := s.apiPost("/ilink/bot/getuploadurl", reqBody, wxUploadAPITimeout)
	if probeFile != "" {
		if err != nil {
			capture("upload_probe", map[string]interface{}{"stage": "file_getuploadurl", "file": probeFile, "err": err.Error()})
		} else {
			capture("upload_probe", map[string]interface{}{"stage": "file_getuploadurl", "file": probeFile, "resp": truncateRunes(string(respBody), 512)})
		}
	}
	if err != nil {
		return "", nil, 0, fmt.Errorf("getuploadurl: %w", err)
	}
	var resp getUploadURLResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", nil, 0, fmt.Errorf("getuploadurl 响应解析: %w", err)
	}

	// ② 上传地址：优先 upload_full_url，其次 upload_param 拼 CDN
	// （hermes 实测注释：upload_full_url 的旧 PUT 会 404，统一用 POST）
	uploadURL := strings.TrimSpace(resp.UploadFullURL)
	if uploadURL == "" {
		if resp.UploadParam == "" {
			return "", nil, 0, fmt.Errorf("getuploadurl 未返回 upload_param/upload_full_url: %s", truncateRunes(string(respBody), 200))
		}
		uploadURL = wxCDNBaseURL + "/upload?encrypted_query_param=" +
			url.QueryEscape(resp.UploadParam) + "&filekey=" + url.QueryEscape(fileKey)
	}

	// ③ CDN 上传：密文直出 body，响应头 x-encrypted-param 即下载票据
	encryptedParam, err = s.uploadCiphertext(uploadURL, ciphertext)
	if probeFile != "" {
		if err != nil {
			capture("upload_probe", map[string]interface{}{"stage": "file_cdn_upload", "file": probeFile, "err": err.Error()})
		} else {
			capture("upload_probe", map[string]interface{}{"stage": "file_cdn_upload", "file": probeFile, "ticket_len": len(encryptedParam)})
		}
	}
	if err != nil {
		return "", nil, 0, fmt.Errorf("cdn upload: %w", err)
	}
	return encryptedParam, aesKey, len(ciphertext), nil
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
	respBody, err := s.sendMessageCard(item)
	if err != nil {
		return err
	}
	return sendAckErr(respBody)
}

// sendFileCardViaUpload sendmessage 发文件卡片（探针制：每次尝试记录完整
// 响应原文——errcode 含在响应体内；失败同样先记录再上抛）。与图片卡片共用
// 信封 sendMessageCard。
func (s *Server) sendFileCardViaUpload(item map[string]interface{}, file string) error {
	respBody, err := s.sendMessageCard(item)
	if err != nil {
		capture("upload_probe", map[string]interface{}{"stage": "file_send_file_card", "file": file, "err": err.Error()})
		return err
	}
	capture("upload_probe", map[string]interface{}{"stage": "file_send_file_card", "file": file, "resp": truncateRunes(string(respBody), 512)})
	return sendAckErr(respBody)
}

// sendMessageCard 组 sendmessage 发送媒体卡片（image_item/file_item 通用信封，
// 与 Send 同款），返回响应原文供探针记录。
func (s *Server) sendMessageCard(item map[string]interface{}) ([]byte, error) {
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
	return s.apiPost("/ilink/bot/sendmessage", body, wxUploadAPITimeout)
}

// sendAckErr 解析 sendmessage 响应的显式错误码（缺失视为成功，与 Send 语义一致）。
func sendAckErr(respBody []byte) error {
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
