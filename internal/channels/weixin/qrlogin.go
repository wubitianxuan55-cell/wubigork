// Package weixin — QR 码登录
package weixin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gaea/gaea/internal/netclient"
	"io"
	"net/http"
	"time"
)

// ─── QR 登录 API ─────────────────────────────────────────────

const (
	qrCodeEndpoint  = "/ilink/bot/get_bot_qrcode?bot_type=3"
	qrStatusFmt     = "/ilink/bot/get_qrcode_status?qrcode=%s"
	qrVerifyCodeFmt = "/ilink/bot/get_qrcode_status?qrcode=%s&verify_code=%s"
	qrPollTimeout   = 35 * time.Second
)

// qrBaseURL 微信扫码登录 API 基地址（var 而非 const：测试可注入 httptest 地址）。
var qrBaseURL = "https://ilinkai.weixin.qq.com"

type QRCodeResp struct {
	Qrcode           string `json:"qrcode"`
	QrcodeImgContent string `json:"qrcode_img_content"`
}

type QRStatusResp struct {
	Status       string `json:"status"`
	BotToken     string `json:"bot_token,omitempty"`
	ILinkBotID   string `json:"ilink_bot_id,omitempty"`
	BaseURL      string `json:"baseurl,omitempty"`
	ILinkUserID  string `json:"ilink_user_id,omitempty"`
	RedirectHost string `json:"redirect_host,omitempty"`
	VerifyCode   string `json:"verify_code,omitempty"`
}

// GetQRCode 获取微信扫码登录的二维码
func GetQRCode() (*QRCodeResp, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"local_token_list": []string{},
	})

	req, _ := http.NewRequest("POST", qrBaseURL+qrCodeEndpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WECHAT-UIN", randomUIN())
	req.Header.Set("iLink-App-Id", "bot")
	req.Header.Set("iLink-App-ClientVersion", "132099") // buildClientVersion("2.4.3") = 0x020403

	client := netclient.NewSimpleClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取二维码失败: %w", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b[:min(200, len(b))]))
	}

	var result QRCodeResp
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("解析二维码响应失败: %w", err)
	}
	return &result, nil
}

// PollQRStatus 轮询二维码扫码状态，返回最终结果
func PollQRStatus(qrcode string) (*QRStatusResp, error) {
	url := fmt.Sprintf(qrBaseURL+qrStatusFmt, qrcode)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-WECHAT-UIN", randomUIN())
	req.Header.Set("iLink-App-Id", "bot")
	req.Header.Set("iLink-App-ClientVersion", "132099") // buildClientVersion("2.4.3") = 0x020403

	client := netclient.NewSimpleClient(qrPollTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("轮询状态失败: %w", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result QRStatusResp
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %w", err)
	}
	// v4.8.2 真机抓包窗口：响应带 baseurl/redirect_host（登录成功态）时抓整
	// 响应原文并存媒体域缓存（capture.go recordQRStatus，无媒体域不抓不刷屏）。
	recordQRStatus(b, &result)
	return &result, nil
}

// PollQRStatusWithCode 带验证码轮询
func PollQRStatusWithCode(qrcode, verifyCode string) (*QRStatusResp, error) {
	url := fmt.Sprintf(qrBaseURL+qrVerifyCodeFmt, qrcode, verifyCode)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-WECHAT-UIN", randomUIN())
	req.Header.Set("iLink-App-Id", "bot")
	req.Header.Set("iLink-App-ClientVersion", "132099") // buildClientVersion("2.4.3") = 0x020403

	client := netclient.NewSimpleClient(qrPollTimeout)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("轮询状态失败: %w", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var result QRStatusResp
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("解析状态响应失败: %w", err)
	}
	// v4.8.2 真机抓包窗口：响应带 baseurl/redirect_host（登录成功态）时抓整
	// 响应原文并存媒体域缓存（capture.go recordQRStatus，无媒体域不抓不刷屏）。
	recordQRStatus(b, &result)
	return &result, nil
}
